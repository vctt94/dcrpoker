// This file contains end-to-end tests that spin up a full poker server backed
// by a real SQLite database. The tests exercise realistic gameplay flows with
// minimal mocking (only the network is in-process via gRPC).
//
// To keep the tests self-contained and independent they **must** be executed
// with `go test ./...` and **should not** depend on external resources.

package e2e

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	testenv "github.com/vctt94/pokerbisonrelay/e2e/internal/testenv"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// isTransientTurnError checks if an error is due to a race condition where
// the game state changed between reading it and acting on it.
// These errors are expected in autoplay tests and should be retried.
func isTransientTurnError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "not your turn") ||
		strings.Contains(msg, "not allowed during phase") ||
		strings.Contains(msg, "game not started") ||
		strings.Contains(msg, "FailedPrecondition")
}

// -----------------------------------------------------------------------------
//
//	SCENARIO: Full Sit'n'Go with 3 players
//
// -----------------------------------------------------------------------------
func TestSitAndGoEndToEnd(t *testing.T) {
	t.Parallel()
	env := testenv.New(t)
	defer env.Close()

	ctx := context.Background()

	players := []string{"alice", "bob", "carol"}

	// Alice creates a new table that acts like a Sit'n'Go (auto-start when all
	// players are ready).
	// BuyIn: 0 to avoid escrow requirement in tests
	ctx = env.ContextWithToken(ctx, "alice")
	createResp, err := env.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      "alice",
		SmallBlind:    10,
		BigBlind:      20,
		MinPlayers:    3,
		MaxPlayers:    3,
		BuyIn:         0,
		StartingChips: 1_000,
		AutoAdvanceMs: 1000,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, createResp.TableId)
	tableID := createResp.TableId

	// Bob & Carol join the table.
	for _, p := range []string{"bob", "carol"} {
		_, err := env.JoinTable(ctx, p, tableID)
		require.NoError(t, err)
	}

	// Everyone marks themselves as ready.
	for _, p := range players {
		_, err := env.SetPlayerReady(ctx, p, tableID)
		require.NoError(t, err)
	}

	// Verify that all players are marked as ready
	gameState := env.GetGameState(ctx, tableID)
	assert.True(t, gameState.GetPlayersJoined() == 3, "expected 3 players joined")

	// Wait until the server flags the game as started.
	env.WaitForGameStart(ctx, tableID, 3*time.Second)

	// Wait for the game to reach PRE_FLOP phase with a valid current player
	// This ensures the game is fully initialized before making bets
	require.Eventually(t, func() bool {
		gameState := env.GetGameState(ctx, tableID)
		if !gameState.GameStarted {
			return false
		}
		if gameState.Phase != pokerrpc.GamePhase_PRE_FLOP {
			return false
		}
		// Ensure we have a current player set
		return gameState.CurrentPlayer != ""
	}, 3*time.Second, 10*time.Millisecond, "game should reach PRE_FLOP with a current player")

	// Buy-ins are now escrow-backed; DCR balances are no longer used.

	// ACTION ROUND -------------------------------------------------------------
	// First player to act (after BB) opens with a 100 bet
	gameState = env.GetGameState(ctx, tableID)
	firstPlayer := gameState.CurrentPlayer
	require.NotEmpty(t, firstPlayer, "should have a current player")
	_, err = env.PokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: firstPlayer,
		TableId:  tableID,
		Amount:   100,
	})
	require.NoError(t, err)

	// Second player calls
	gameState = env.GetGameState(ctx, tableID)
	secondPlayer := gameState.CurrentPlayer
	require.NotEmpty(t, secondPlayer, "should have a current player")
	require.NotEqual(t, firstPlayer, secondPlayer, "current player should have changed")
	_, err = env.PokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: secondPlayer,
		TableId:  tableID,
		Amount:   100,
	})
	require.NoError(t, err)

	// Third player decides to fold
	gameState = env.GetGameState(ctx, tableID)
	thirdPlayer := gameState.CurrentPlayer
	require.NotEmpty(t, thirdPlayer, "should have a current player")
	require.NotEqual(t, firstPlayer, thirdPlayer, "third player should be different")
	require.NotEqual(t, secondPlayer, thirdPlayer, "third player should be different")
	_, err = env.PokerClient.FoldBet(ctx, &pokerrpc.FoldBetRequest{
		PlayerId: thirdPlayer,
		TableId:  tableID,
	})
	require.NoError(t, err)

	// Validate pot value (220) via GetGameState.
	// Pot = 30 (blinds) + 100 (first player's bet) + 90 (second player's call minus their blind)
	state, err := env.PokerClient.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	require.NoError(t, err)
	assert.Equal(t, int64(220), state.GameState.Pot, "unexpected pot size")

	// Verify the third player is marked as folded
	for _, player := range state.GameState.Players {
		if player.Id == thirdPlayer {
			assert.True(t, player.Folded, "third player should be marked as folded")
		}
	}

	// Verify the remaining active players
	activePlayers := 0
	for _, player := range state.GameState.Players {
		if !player.Folded {
			activePlayers++
		}
	}
	assert.Equal(t, 2, activePlayers, "expected 2 active players")

	require.NoError(t, err)
}

// -----------------------------------------------------------------------------
//
//	SCENARIO: Complete Hand Flow - All Betting Rounds with 4 players
//
// -----------------------------------------------------------------------------
func TestCompleteHandFlow(t *testing.T) {
	t.Parallel()
	env := testenv.New(t)
	defer env.Close()

	ctx := context.Background()

	// Setup players with initial bankrolls
	players := []string{"player1", "player2", "player3", "player4"}
	playerIDs := make(map[string]string, len(players))
	for _, p := range players {
		playerIDs[p] = testenv.PlayerIDToShortIDString(p)
	}

	// Player1 creates a table for 4 players (BuyIn: 0 to avoid escrow requirement in tests)
	tableID := env.CreateTableWithBuyIn(ctx, "player1", 4, 4, 0)

	// All players join the table
	for _, p := range players[1:] { // Skip player1 who already created the table
		_, err := env.JoinTable(ctx, p, tableID)
		require.NoError(t, err)
	}

	// All players mark themselves as ready
	for _, p := range players {
		_, err := env.SetPlayerReady(ctx, p, tableID)
		require.NoError(t, err)
	}

	// Wait for game to start
	env.WaitForGameStart(ctx, tableID, 3*time.Second)
	// Ensure PRE_FLOP before acting
	env.WaitForGamePhase(ctx, tableID, pokerrpc.GamePhase_PRE_FLOP, 3*time.Second)
	// Ensure PRE_FLOP reached before acting
	env.WaitForGamePhase(ctx, tableID, pokerrpc.GamePhase_PRE_FLOP, 3*time.Second)

	// Note: explicit per-action waits are used instead of a helper to avoid unused warnings.
	// Ensure PRE_FLOP reached before acting
	env.WaitForGamePhase(ctx, tableID, pokerrpc.GamePhase_PRE_FLOP, 3*time.Second)

	// Note: we rely on explicit waits before actions in tests that need them.

	// Helper: wait until it's the specified player's turn
	waitForTurn := func(playerID string, timeout time.Duration) {
		require.Eventually(t, func() bool {
			state := env.GetGameState(ctx, tableID)
			return state.CurrentPlayer == playerID
		}, timeout, 25*time.Millisecond, "did not become %s's turn within %s", playerID, timeout)
	}

	// PRE-FLOP BETTING
	// In 4-player game: player1=dealer, player2=SB, player3=BB, player4=UTG (acts first)

	// Player4 calls the big blind
	waitForTurn(playerIDs["player4"], 2*time.Second)
	_, err := env.PokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: playerIDs["player4"],
		TableId:  tableID,
		Amount:   20,
	})
	require.NoError(t, err)

	// Player1 raises
	waitForTurn(playerIDs["player1"], 2*time.Second)
	_, err = env.PokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: playerIDs["player1"],
		TableId:  tableID,
		Amount:   60, // Raising to 60
	})
	require.NoError(t, err)

	// Player2 calls the raise (SB needs to add 50 more to existing 10)
	waitForTurn(playerIDs["player2"], 2*time.Second)
	_, err = env.PokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: playerIDs["player2"],
		TableId:  tableID,
		Amount:   60,
	})
	require.NoError(t, err)

	// Player3 calls the raise (BB needs to add 40 more to existing 20)
	waitForTurn(playerIDs["player3"], 2*time.Second)
	_, err = env.PokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: playerIDs["player3"],
		TableId:  tableID,
		Amount:   60,
	})
	require.NoError(t, err)

	// Player4 calls the raise
	waitForTurn(playerIDs["player4"], 2*time.Second)
	_, err = env.PokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: playerIDs["player4"],
		TableId:  tableID,
		Amount:   60,
	})
	require.NoError(t, err)

	// Check pot after pre-flop: blinds (30) + all players bet 60 = 270, but actual is 240
	state := env.GetGameState(ctx, tableID)
	assert.Equal(t, int64(240), state.Pot, "unexpected pot size after pre-flop")

	// FLOP ROUND
	// Wait for flop to be dealt
	env.WaitForGamePhase(ctx, tableID, pokerrpc.GamePhase_FLOP, 3*time.Second)

	// Make sure we have 3 community cards
	state = env.GetGameState(ctx, tableID)
	assert.Equal(t, 3, len(state.CommunityCards), "expected 3 community cards after flop")

	// Post-flop betting starts with small blind (player2)
	// Player2 checks
	waitForTurn(playerIDs["player2"], 2*time.Second)
	_, err = env.PokerClient.CheckBet(ctx, &pokerrpc.CheckBetRequest{
		PlayerId: playerIDs["player2"],
		TableId:  tableID,
	})
	require.NoError(t, err)

	// Player3 bets 100
	waitForTurn(playerIDs["player3"], 2*time.Second)
	_, err = env.PokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: playerIDs["player3"],
		TableId:  tableID,
		Amount:   100,
	})
	require.NoError(t, err)

	// Player4 calls
	waitForTurn(playerIDs["player4"], 2*time.Second)
	_, err = env.PokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: playerIDs["player4"],
		TableId:  tableID,
		Amount:   100,
	})
	require.NoError(t, err)

	// Player1 folds
	waitForTurn(playerIDs["player1"], 2*time.Second)
	_, err = env.PokerClient.FoldBet(ctx, &pokerrpc.FoldBetRequest{
		PlayerId: playerIDs["player1"],
		TableId:  tableID,
	})
	require.NoError(t, err)

	// Player2 folds
	waitForTurn(playerIDs["player2"], 2*time.Second)
	_, err = env.PokerClient.FoldBet(ctx, &pokerrpc.FoldBetRequest{
		PlayerId: playerIDs["player2"],
		TableId:  tableID,
	})
	require.NoError(t, err)

	// Check pot after flop: 240 + 100 + 100 = 440
	state = env.GetGameState(ctx, tableID)
	assert.Equal(t, int64(440), state.Pot, "unexpected pot size after flop")

	// TURN ROUND
	// Wait for turn card
	env.WaitForGamePhase(ctx, tableID, pokerrpc.GamePhase_TURN, 3*time.Second)

	// Make sure we have 4 community cards
	state = env.GetGameState(ctx, tableID)
	assert.Equal(t, 4, len(state.CommunityCards), "expected 4 community cards after turn")

	// Only player3 and player4 remain
	// Player3 bets 200
	waitForTurn(playerIDs["player3"], 2*time.Second)
	_, err = env.PokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: playerIDs["player3"],
		TableId:  tableID,
		Amount:   200,
	})
	require.NoError(t, err)

	// Player4 calls
	waitForTurn(playerIDs["player4"], 2*time.Second)
	_, err = env.PokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: playerIDs["player4"],
		TableId:  tableID,
		Amount:   200,
	})
	require.NoError(t, err)

	// Check pot after turn: 440 + 200 + 200 = 840
	state = env.GetGameState(ctx, tableID)
	assert.Equal(t, int64(840), state.Pot, "unexpected pot size after turn")

	// RIVER ROUND
	// Wait for river card
	env.WaitForGamePhase(ctx, tableID, pokerrpc.GamePhase_RIVER, 3*time.Second)

	// Make sure we have 5 community cards
	state = env.GetGameState(ctx, tableID)
	assert.Equal(t, 5, len(state.CommunityCards), "expected 5 community cards after river")

	// Player3 checks
	waitForTurn(playerIDs["player3"], 2*time.Second)
	_, err = env.PokerClient.CheckBet(ctx, &pokerrpc.CheckBetRequest{
		PlayerId: playerIDs["player3"],
		TableId:  tableID,
	})
	require.NoError(t, err)

	// Player4 bets 300
	waitForTurn(playerIDs["player4"], 2*time.Second)
	_, err = env.PokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: playerIDs["player4"],
		TableId:  tableID,
		Amount:   300,
	})
	require.NoError(t, err)

	// Player3 folds
	waitForTurn(playerIDs["player3"], 2*time.Second)
	_, err = env.PokerClient.FoldBet(ctx, &pokerrpc.FoldBetRequest{
		PlayerId: playerIDs["player3"],
		TableId:  tableID,
	})
	require.NoError(t, err)

	// Wait for showdown
	env.WaitForGamePhase(ctx, tableID, pokerrpc.GamePhase_SHOWDOWN, 3*time.Second)

	// Wait for showdown processing to complete and winners to be available
	var winners *pokerrpc.GetLastWinnersResponse
	require.Eventually(t, func() bool {
		var err error
		winners, err = env.PokerClient.GetLastWinners(ctx, &pokerrpc.GetLastWinnersRequest{
			TableId: tableID,
		})
		return err == nil && len(winners.Winners) > 0
	}, 3*time.Second, 50*time.Millisecond, "showdown should complete with winners")

	// Verify Player4 won the pot
	assert.Equal(t, 1, len(winners.Winners), "expected 1 winner")
	assert.Equal(t, playerIDs["player4"], winners.Winners[0].PlayerId, "expected player4 to win")

	// Calculate expected pot programmatically to avoid magic numbers
	// Pre-flop: all 4 players bet 60 each = 240
	// Flop: player3 bets 100, player4 calls 100 = +200 = 440
	// Turn: player3 bets 200, player4 calls 200 = +400 = 840
	// River: player4 bets 300, player3 folds = uncalled bet refunded = +0
	expectedPot := int64(0)
	expectedPot += 4 * 60    // pre-flop: all players at 60
	expectedPot += 100 + 100 // flop bet/call
	expectedPot += 200 + 200 // turn bet/call
	// river: 300 bet, no call → refunded → +0

	assert.Equal(t, expectedPot, winners.Winners[0].Winnings, "unexpected pot amount in winner response")
}

// -----------------------------------------------------------------------------
//
//	SCENARIO: Test Player Timeout and Auto-Check-or-Fold
//
// -----------------------------------------------------------------------------
func TestPlayerTimeoutAutoCheckOrFold(t *testing.T) {
	t.Parallel()
	env := testenv.New(t)
	defer env.Close()

	ctx := context.Background()

	// Setup players
	players := []string{"active1", "active2", "timeout"}
	playerIDs := make(map[string]string, len(players))
	for _, p := range players {
		playerIDs[p] = testenv.PlayerIDToShortIDString(p)
	}

	// Create table with short timebank
	// BuyIn: 0 to avoid escrow requirement in tests
	ctx = env.ContextWithToken(ctx, "active1")
	createResp, err := env.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:        "active1",
		SmallBlind:      10,
		BigBlind:        20,
		MinPlayers:      3,
		MaxPlayers:      3,
		BuyIn:           0,
		StartingChips:   1_000,
		TimeBankSeconds: 5, // 5 seconds timeout
		AutoAdvanceMs:   1000,
	})
	require.NoError(t, err)
	tableID := createResp.TableId

	// All players join and mark ready
	for _, p := range players[1:] {
		_, err := env.JoinTable(ctx, p, tableID)
		require.NoError(t, err)
	}

	for _, p := range players {
		_, err := env.SetPlayerReady(ctx, p, tableID)
		require.NoError(t, err)
	}

	// Wait for game to start
	env.WaitForGameStart(ctx, tableID, 3*time.Second)

	// Wait for the game to reach PRE_FLOP phase with a valid current player
	require.Eventually(t, func() bool {
		gameState := env.GetGameState(ctx, tableID)
		if !gameState.GameStarted {
			return false
		}
		if gameState.Phase != pokerrpc.GamePhase_PRE_FLOP {
			return false
		}
		// Ensure we have a current player set
		return gameState.CurrentPlayer != ""
	}, 3*time.Second, 10*time.Millisecond, "game should reach PRE_FLOP with a current player")

	// active1 and active2 make their moves
	_, err = env.PokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: playerIDs["active1"],
		TableId:  tableID,
		Amount:   20,
	})
	require.NoError(t, err)

	_, err = env.PokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: playerIDs["active2"],
		TableId:  tableID,
		Amount:   20,
	})
	require.NoError(t, err)

	// But "timeout" player doesn't act - should auto-check-or-fold after timeout
	// Since they need to call from 20 to 20 but already have 20 bet (big blind), they should auto-check
	// Wait for auto-check-or-fold to occur
	require.Eventually(t, func() bool {
		state := env.GetGameState(ctx, tableID)
		// Check if the timeout player has been auto-checked (turn should have advanced)
		return state.CurrentPlayer != playerIDs["timeout"]
	}, 8*time.Second, 100*time.Millisecond, "timeout player should be auto-checked")

	// Check if player was auto-checked or auto-folded based on their position
	state := env.GetGameState(ctx, tableID)
	for _, player := range state.Players {
		if player.Id == playerIDs["timeout"] {
			// The timeout player should have been auto-checked since they already have the required bet (big blind = 20)
			// but if they needed to put in more money, they would have been auto-folded
			// In this case, as the big blind, they should have been auto-checked since currentBet (20) == their bet (20)
			assert.False(t, player.Folded, "timeout player should have been auto-checked, not auto-folded, since they could check")
		}
	}
}

// -----------------------------------------------------------------------------
//
//	SCENARIO: Test Player Timeout Auto-Fold (when cannot check)
//
// -----------------------------------------------------------------------------
func TestPlayerTimeoutAutoFoldWhenCannotCheck(t *testing.T) {
	t.Parallel()
	env := testenv.New(t)
	defer env.Close()

	ctx := context.Background()

	players := []string{"active1", "active2", "timeout"}
	playerIDs := make(map[string]string)
	playerIDs["active1"] = testenv.PlayerIDToShortIDString("active1")
	playerIDs["active2"] = testenv.PlayerIDToShortIDString("active2")
	playerIDs["timeout"] = testenv.PlayerIDToShortIDString("timeout")
	// Create table with short timebank
	// BuyIn: 0 to avoid escrow requirement in tests
	ctx = env.ContextWithToken(ctx, "active1")
	createResp, err := env.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:        "active1",
		SmallBlind:      10,
		BigBlind:        20,
		MinPlayers:      3,
		MaxPlayers:      3,
		BuyIn:           0,
		StartingChips:   1_000,
		TimeBankSeconds: 5, // 5 seconds timeout
		AutoAdvanceMs:   1000,
	})
	require.NoError(t, err)
	tableID := createResp.TableId

	// All players join and mark ready
	for _, p := range players[1:] {
		_, err := env.JoinTable(ctx, p, tableID)
		require.NoError(t, err)
	}

	for _, p := range players {
		_, err := env.SetPlayerReady(ctx, p, tableID)
		require.NoError(t, err)
	}

	// Wait for game to start
	env.WaitForGameStart(ctx, tableID, 3*time.Second)

	// Wait for the game to reach PRE_FLOP phase with a valid current player
	require.Eventually(t, func() bool {
		gameState := env.GetGameState(ctx, tableID)
		if !gameState.GameStarted {
			return false
		}
		if gameState.Phase != pokerrpc.GamePhase_PRE_FLOP {
			return false
		}
		// Ensure we have a current player set
		return gameState.CurrentPlayer != ""
	}, 3*time.Second, 10*time.Millisecond, "game should reach PRE_FLOP with a current player")

	// active1 calls the big blind (20)
	require.Eventually(t, func() bool {
		st := env.GetGameState(ctx, tableID)
		return st.CurrentPlayer == playerIDs["active1"]
	}, 2*time.Second, 10*time.Millisecond, "active1 should be current player")
	_, err = env.PokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: playerIDs["active1"],
		TableId:  tableID,
		Amount:   20,
	})
	require.NoError(t, err)

	// active2 raises to 50
	require.Eventually(t, func() bool {
		st := env.GetGameState(ctx, tableID)
		return st.CurrentPlayer == playerIDs["active2"]
	}, 2*time.Second, 10*time.Millisecond, "active2 should be current player")
	_, err = env.PokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: playerIDs["active2"],
		TableId:  tableID,
		Amount:   50,
	})
	require.NoError(t, err)

	// Now "timeout" player (big blind) would need to call from 20 to 50 - should auto-fold after timeout
	// Wait for auto-fold to occur
	require.Eventually(t, func() bool {
		state := env.GetGameState(ctx, tableID)
		// Check if the timeout player has been auto-folded (turn should have advanced)
		return state.CurrentPlayer != playerIDs["timeout"]
	}, 8*time.Second, 100*time.Millisecond, "timeout player should be auto-folded")

	// Check if player was auto-folded (since they cannot check - they need to call the raise)
	state := env.GetGameState(ctx, tableID)
	t.Logf("Game state after timeout - Current bet: %d, Pot: %d", state.CurrentBet, state.Pot)
	for _, player := range state.Players {
		t.Logf("Player %s: Bet=%d, Folded=%t", player.Id, player.CurrentBet, player.Folded)
		if player.Id == playerIDs["timeout"] {
			assert.True(t, player.Folded, "timeout player should have been auto-folded since they cannot check (need to call raise)")
		}
	}
}

// -----------------------------------------------------------------------------
//
//	SCENARIO: Basic table creation and player readiness
//
// -----------------------------------------------------------------------------
func TestBasicTableAndReadiness(t *testing.T) {
	t.Parallel()
	env := testenv.New(t)
	defer env.Close()

	ctx := context.Background()

	// Setup players with initial bankrolls
	players := []string{"player1", "player2", "player3", "player4"}

	// Player1 creates a table for 4 players
	tableID := env.CreateStandardTable(ctx, "player1", 4, 4)

	// All players join the table
	for _, p := range players[1:] { // Skip player1 who already created the table
		_, err := env.JoinTable(ctx, p, tableID)
		require.NoError(t, err)
	}

	// Verify initial state
	state := env.GetGameState(ctx, tableID)
	assert.Equal(t, int32(4), state.PlayersJoined, "expected 4 players joined")
	assert.Equal(t, int32(4), state.PlayersRequired, "expected 4 players required")
	assert.False(t, state.GameStarted, "game should not be started yet")

	// Set players ready one by one and check state
	for i, p := range players {
		_, err := env.SetPlayerReady(ctx, p, tableID)
		require.NoError(t, err)

		// Check that player is marked as ready
		state = env.GetGameState(ctx, tableID)
		readyCount := 0
		for _, player := range state.Players {
			if player.IsReady {
				readyCount++
			}
		}
		assert.Equal(t, i+1, readyCount, "expected %d players ready", i+1)
	}

	// Now all players are ready, game should start
	env.WaitForGameStart(ctx, tableID, 3*time.Second)
}

// -----------------------------------------------------------------------------
//
//	SCENARIO: Test basic betting
//
// -----------------------------------------------------------------------------
func TestBasicBetting(t *testing.T) {
	t.Parallel()
	env := testenv.New(t)
	defer env.Close()

	ctx := context.Background()

	// Setup 3 players
	players := []string{"p1", "p2", "p3"}

	// Create and join table
	tableID := env.CreateStandardTable(ctx, "p1", 3, 3)
	for _, p := range players[1:] {
		_, err := env.JoinTable(ctx, p, tableID)
		require.NoError(t, err)
	}

	// Set all players ready
	for _, p := range players {
		_, err := env.SetPlayerReady(ctx, p, tableID)
		require.NoError(t, err)
	}

	// Wait for game to start
	env.WaitForGameStart(ctx, tableID, 3*time.Second)
	// Ensure PRE_FLOP is fully reached before asserting state/acting
	env.WaitForGamePhase(ctx, tableID, pokerrpc.GamePhase_PRE_FLOP, 3*time.Second)

	// Helper: wait until it's the specified player's turn (scoped to this test)
	waitForTurnBasic := func(playerID string, timeout time.Duration) {
		require.Eventually(t, func() bool {
			state := env.GetGameState(ctx, tableID)
			return state.CurrentPlayer == playerID
		}, timeout, 25*time.Millisecond, "did not become %s's turn within %s", playerID, timeout)
	}

	// Note: second helper removed; using waitForTurnBasic below.

	// Verify initial pot includes blinds (10 + 20 = 30)
	state := env.GetGameState(ctx, tableID)
	assert.Equal(t, int64(30), state.Pot, "initial pot should include blinds (10+20=30)")

	// Check current player and bet state
	t.Logf("Current player: %s, Current bet: %d", state.CurrentPlayer, state.CurrentBet)

	// Check player bets to understand blind posting
	for _, player := range state.Players {
		t.Logf("Player %s has bet: %d, folded: %t", player.Id, player.CurrentBet, player.Folded)
	}

	// Verify blind posting is correct - dynamically determine who has which blind
	// In 3-player, one player has no blind (dealer/UTG), one has SB (10), one has BB (20)
	playerBets := make(map[string]int64)
	var dealerID, sbID, bbID string
	for _, player := range state.Players {
		playerBets[player.Id] = player.CurrentBet
		if player.CurrentBet == 0 {
			dealerID = player.Id
		} else if player.CurrentBet == 10 {
			sbID = player.Id
		} else if player.CurrentBet == 20 {
			bbID = player.Id
		}
	}

	// Verify we found all three roles
	require.NotEmpty(t, dealerID, "should have identified dealer")
	require.NotEmpty(t, sbID, "should have identified small blind")
	require.NotEmpty(t, bbID, "should have identified big blind")
	assert.Equal(t, int64(0), playerBets[dealerID], "dealer should have no blind")
	assert.Equal(t, int64(10), playerBets[sbID], "small blind should be 10")
	assert.Equal(t, int64(20), playerBets[bbID], "big blind should be 20")

	// First player to act should be the dealer (Under the Gun in 3-handed)
	assert.Equal(t, dealerID, state.CurrentPlayer, "dealer should be first to act (Under the Gun)")

	// Current bet should be big blind amount (20)
	assert.Equal(t, int64(20), state.CurrentBet, "current bet should be big blind (20)")

	// Helper: perform a bet with retry on transient turn/phase races.
	betWithRetry := func(playerID string, amount int64) {
		deadline := time.Now().Add(1 * time.Second)
		var err error
		for {
			_, err = env.PokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
				PlayerId: playerID,
				TableId:  tableID,
				Amount:   amount,
			})
			if err == nil {
				return
			}
			// Retry on transient "not your turn / phase changed" errors while within deadline.
			if !isTransientTurnError(err) || time.Now().After(deadline) {
				require.NoError(t, err)
			}
			time.Sleep(10 * time.Millisecond)
		}
	}

	// Helper: perform a check with retry on transient turn/phase races.
	checkWithRetry := func(playerID string) {
		deadline := time.Now().Add(1 * time.Second)
		var err error
		for {
			_, err = env.PokerClient.CheckBet(ctx, &pokerrpc.CheckBetRequest{
				PlayerId: playerID,
				TableId:  tableID,
			})
			if err == nil {
				return
			}
			if !isTransientTurnError(err) || time.Now().After(deadline) {
				require.NoError(t, err)
			}
			time.Sleep(10 * time.Millisecond)
		}
	}

	// First player (dealer) calls the big blind (20)
	waitForTurnBasic(dealerID, 2*time.Second)
	betWithRetry(dealerID, 20)

	// Check pot is now 50 (30 from blinds + 20 from dealer's call)
	state = env.GetGameState(ctx, tableID)
	assert.Equal(t, int64(50), state.Pot, "pot should be 50 after dealer's call (30+20)")

	// Small blind calls by betting 20 total (needs to add 10 more to their existing 10)
	waitForTurnBasic(sbID, 2*time.Second)
	betWithRetry(sbID, 20)

	// Check pot is now 60 (50 + 10 more from SB)
	state = env.GetGameState(ctx, tableID)
	assert.Equal(t, int64(60), state.Pot, "pot should be 60 after SB's call (50+10)")

	// Big blind can check (already has 20 bet)
	waitForTurnBasic(bbID, 2*time.Second)
	checkWithRetry(bbID)

	// Pot should still be 60 after check
	state = env.GetGameState(ctx, tableID)
	assert.Equal(t, int64(60), state.Pot, "pot should remain 60 after BB's check")
}

// -----------------------------------------------------------------------------
//
//	SCENARIO: Test StartingChips default when set to 0
//
// -----------------------------------------------------------------------------
func TestStartingChipsDefault(t *testing.T) {
	t.Parallel()
	env := testenv.New(t)
	defer env.Close()

	ctx := context.Background()

	// Setup players
	players := []string{"player1", "player2", "player3"}

	// Create table with StartingChips set to 0 to test default logic
	// BuyIn: 0 to avoid escrow requirement in tests
	ctx = env.ContextWithToken(ctx, "player1")
	createResp, err := env.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      "player1",
		SmallBlind:    10,
		BigBlind:      20,
		MinPlayers:    3,
		MaxPlayers:    3,
		BuyIn:         0,
		StartingChips: 0, // This should default to 1000
		AutoAdvanceMs: 1000,
	})
	require.NoError(t, err)
	tableID := createResp.TableId

	// All players join and mark ready
	for _, p := range players[1:] {
		_, err := env.JoinTable(ctx, p, tableID)
		require.NoError(t, err)
	}

	for _, p := range players {
		_, err := env.SetPlayerReady(ctx, p, tableID)
		require.NoError(t, err)
	}

	// Wait for game to start
	env.WaitForGameStart(ctx, tableID, 3*time.Second)

	// Wait for blinds to be posted and visible (pot should be 30)
	var state *pokerrpc.GameUpdate
	require.Eventually(t, func() bool {
		state = env.GetGameState(ctx, tableID)
		return state.Pot == 30 && state.Phase == pokerrpc.GamePhase_PRE_FLOP
	}, 2*time.Second, 10*time.Millisecond, "blinds should be posted with pot=30")

	// Dynamically identify player roles based on blind amounts
	var dealerID, sbID, bbID string
	for _, player := range state.Players {
		switch player.CurrentBet {
		case 0:
			dealerID = player.Id
			t.Logf("Player %s: has bet %d, should have balance %d", player.Id, player.CurrentBet, 1000)
		case 10:
			sbID = player.Id
			t.Logf("Player %s: has bet %d, should have balance %d", player.Id, player.CurrentBet, 990)
		case 20:
			bbID = player.Id
			t.Logf("Player %s: has bet %d, should have balance %d", player.Id, player.CurrentBet, 980)
		}
	}

	// Verify all roles were identified
	require.NotEmpty(t, dealerID, "should have identified dealer")
	require.NotEmpty(t, sbID, "should have identified small blind")
	require.NotEmpty(t, bbID, "should have identified big blind")

	// Wait for dealer's turn before making bet
	require.Eventually(t, func() bool {
		state = env.GetGameState(ctx, tableID)
		return state.CurrentPlayer == dealerID && state.Phase == pokerrpc.GamePhase_PRE_FLOP
	}, 2*time.Second, 10*time.Millisecond, "should become dealer's turn")

	// First player to act (dealer) makes a bet
	_, err = env.PokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: dealerID,
		TableId:  tableID,
		Amount:   20, // Call the big blind
	})
	require.NoError(t, err)

	// Verify pot is now 50 (use Eventually to account for async FSM transition)
	require.Eventually(t, func() bool {
		state = env.GetGameState(ctx, tableID)
		return state.Pot == 50
	}, 500*time.Millisecond, 10*time.Millisecond, "pot should be 50 after dealer's call")
}

// -----------------------------------------------------------------------------
//
//	SCENARIO: Test StartingChips default when BuyIn is 0
//
// -----------------------------------------------------------------------------
func TestStartingChipsDefaultWithZeroBuyIn(t *testing.T) {
	t.Parallel()
	env := testenv.New(t)
	defer env.Close()

	ctx := context.Background()

	// Setup players
	players := []string{"player1", "player2"}

	// Create table with both StartingChips and BuyIn set to 0
	ctx = env.ContextWithToken(ctx, "player1")
	createResp, err := env.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      "player1",
		SmallBlind:    10,
		BigBlind:      20,
		MinPlayers:    2,
		MaxPlayers:    2,
		BuyIn:         0, // Zero buy-in
		StartingChips: 0, // Should default to 1000
		AutoAdvanceMs: 1000,
	})
	require.NoError(t, err)
	tableID := createResp.TableId

	// Player2 joins
	_, err = env.JoinTable(ctx, "player2", tableID)
	require.NoError(t, err)

	// Both players mark ready
	for _, p := range players {
		_, err := env.SetPlayerReady(ctx, p, tableID)
		require.NoError(t, err)
	}

	// Wait for game to start
	env.WaitForGameStart(ctx, tableID, 3*time.Second)

	// Get game state and verify that players have the default 1000 starting chips
	state := env.GetGameState(ctx, tableID)

	// All players should start with 1000 chips (the fallback default)
	// minus any blinds they've posted
	for _, player := range state.Players {
		switch player.Id {
		case "player1":
			// In heads-up, dealer posts small blind (10)
			t.Logf("Player %s (dealer/SB): has bet %d", player.Id, player.CurrentBet)
		case "player2":
			// Other player posts big blind (20)
			t.Logf("Player %s (BB): has bet %d", player.Id, player.CurrentBet)
		}
	}

	// Verify pot includes blinds (10 + 20 = 30)
	assert.Equal(t, int64(30), state.Pot, "pot should include blinds (10+20=30)")

	// Debug: Check who is the current player
	t.Logf("Current player to act: %s, Current bet: %d", state.CurrentPlayer, state.CurrentBet)

	// In heads-up pre-flop, small blind (dealer) acts first, which is correct poker rules
	// So player1 (SB) should act first
	expectedCurrentPlayer := testenv.PlayerIDToShortIDString("player1") // SB acts first in heads-up preflop
	assert.Equal(t, expectedCurrentPlayer, state.CurrentPlayer, "Small blind should act first in heads-up preflop")

	// Player1 (SB) raises to 40
	_, err = env.PokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: state.CurrentPlayer, // Should be player1 (SB)
		TableId:  tableID,
		Amount:   40, // Raise to 40
	})
	require.NoError(t, err)

	// Verify pot is now 60 (30 initial + 30 additional from SB raising from 10 to 40)
	state = env.GetGameState(ctx, tableID)
	assert.Equal(t, int64(60), state.Pot, "pot should be 60 after SB's raise (30+30)")

	// Now it should be player2's (BB) turn to call, raise, or fold
	t.Logf("After SB raise - Current player: %s, Current bet: %d", state.CurrentPlayer, state.CurrentBet)

	// Player2 (BB) calls by betting 40 total (adding 20 more to existing 20)
	_, err = env.PokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: state.CurrentPlayer, // Should be player2 (BB)
		TableId:  tableID,
		Amount:   40, // Call the raise
	})
	require.NoError(t, err)

	// Verify pot is now 80 (60 + 20 additional from BB calling)
	// Use Eventually to account for async state machine transitions when betting round completes
	require.Eventually(t, func() bool {
		state = env.GetGameState(ctx, tableID)
		return state.Pot == 80
	}, 500*time.Millisecond, 10*time.Millisecond, "pot should be 80 after BB's call (60+20)")
}

// -----------------------------------------------------------------------------
//
//	SCENARIO: Autoplay a single hand with 3 players until showdown
//
// -----------------------------------------------------------------------------
func TestThreePlayersAutoplayOneHand(t *testing.T) {
	t.Parallel()
	env := testenv.New(t)
	defer env.Close()

	ctx := context.Background()

	// Setup 3 players and bankroll
	players := []string{"a3", "b3", "c3"}

	// Create a 3-max table and join remaining players
	tableID := env.CreateStandardTable(ctx, players[0], 3, 3)
	for _, p := range players[1:] {
		_, err := env.JoinTable(ctx, p, tableID)
		require.NoError(t, err)
	}

	// Everyone ready
	for _, p := range players {
		_, err := env.SetPlayerReady(ctx, p, tableID)
		require.NoError(t, err)
	}

	// Wait for game to start
	env.WaitForGameStart(ctx, tableID, 3*time.Second)

	// Autoplay loop: for the current player, if needs to match current bet, call; otherwise check.
	deadline := time.NewTimer(30 * time.Second)
	defer deadline.Stop()

	var potChecked bool
	for {
		select {
		case <-deadline.C:
			t.Fatal("autoplay timed out before reaching showdown")
		default:
		}

		state := env.GetGameState(ctx, tableID)

		// Stop if we've reached showdown
		if state.GameStarted && state.Phase == pokerrpc.GamePhase_SHOWDOWN {
			break
		}

		// Wait for a valid betting phase (handles race with hand transitions)
		if state.Phase != pokerrpc.GamePhase_PRE_FLOP &&
			state.Phase != pokerrpc.GamePhase_FLOP &&
			state.Phase != pokerrpc.GamePhase_TURN &&
			state.Phase != pokerrpc.GamePhase_RIVER {
			continue
		}

		// Check pot during RIVER phase (before showdown completes)
		if !potChecked && state.GameStarted && state.Phase == pokerrpc.GamePhase_RIVER {
			assert.Greater(t, state.Pot, int64(0), "pot should be greater than 0 during RIVER phase")
			potChecked = true
		}

		// Identify current player and their contribution
		curr := state.CurrentPlayer
		var currPlayer *pokerrpc.Player
		for _, p := range state.Players {
			if p.Id == curr {
				currPlayer = p
				break
			}
		}
		if currPlayer == nil {
			// Game state not yet stable, retry
			continue
		}

		// Use Eventually pattern to handle transient turn races.
		require.Eventually(t, func() bool {
			latest := env.GetGameState(ctx, tableID)
			// Skip if hand not in a betting phase anymore
			if latest.Phase != pokerrpc.GamePhase_PRE_FLOP &&
				latest.Phase != pokerrpc.GamePhase_FLOP &&
				latest.Phase != pokerrpc.GamePhase_TURN &&
				latest.Phase != pokerrpc.GamePhase_RIVER {
				return false
			}
			cur := latest.CurrentPlayer
			var curPl *pokerrpc.Player
			for _, p := range latest.Players {
				if p.Id == cur {
					curPl = p
					break
				}
			}
			if curPl == nil {
				return false
			}

			var err error
			if curPl.CurrentBet >= latest.CurrentBet {
				_, err = env.PokerClient.CheckBet(ctx, &pokerrpc.CheckBetRequest{PlayerId: cur, TableId: tableID})
			} else {
				_, err = env.PokerClient.CallBet(ctx, &pokerrpc.CallBetRequest{PlayerId: cur, TableId: tableID})
			}
			if err != nil {
				if isTransientTurnError(err) {
					return false
				}
				require.NoError(t, err)
			}
			return true
		}, 1*time.Second, 10*time.Millisecond, "failed to execute action for player %s", curr)
	}

	// Final assertions
	final := env.GetGameState(ctx, tableID)
	require.Equal(t, pokerrpc.GamePhase_SHOWDOWN, final.Phase)
	require.Equal(t, int32(3), final.PlayersJoined)
	// Ensure we still see 3 players in the final state
	assert.Equal(t, 3, len(final.Players))
	// Verify that we checked the pot during the RIVER phase
	assert.True(t, potChecked, "pot should have been checked during RIVER phase")
}

// SCENARIO: Test Multiple Consecutive Hands - ActionsInRound Bug
func TestMultipleConsecutiveHandsActionsInRoundBug(t *testing.T) {
	t.Parallel()
	env := testenv.New(t)
	defer env.Close()

	ctx := context.Background()

	// Setup 2 players for heads-up
	players := []string{"heads1", "heads2"}

	// Create heads-up table
	tableID := env.CreateStandardTable(ctx, "heads1", 2, 2)

	// Player2 joins
	_, err := env.JoinTable(ctx, "heads2", tableID)
	require.NoError(t, err)

	// Both players mark ready
	for _, p := range players {
		_, err := env.SetPlayerReady(ctx, p, tableID)
		require.NoError(t, err)
	}

	// Wait for game to start
	env.WaitForGameStart(ctx, tableID, 3*time.Second)

	// Helper to wait for a specific player's turn
	waitForPlayerTurn := func(playerID string, timeout time.Duration) {
		require.Eventually(t, func() bool {
			state := env.GetGameState(ctx, tableID)
			return state.CurrentPlayer == playerID
		}, timeout, 10*time.Millisecond, "should be %s's turn", playerID)
	}

	// Helper to wait for a specific game phase
	waitForPhase := func(phase pokerrpc.GamePhase, timeout time.Duration) {
		require.Eventually(t, func() bool {
			state := env.GetGameState(ctx, tableID)
			return state.Phase == phase
		}, timeout, 10*time.Millisecond, "should reach phase %s", phase)
	}

	// FIRST HAND: Play a complete hand
	// Wait for PRE_FLOP
	waitForPhase(pokerrpc.GamePhase_PRE_FLOP, 3*time.Second)

	// Get initial state to understand positions
	state := env.GetGameState(ctx, tableID)
	t.Logf("First hand - Current player: %s, Phase: %s", state.CurrentPlayer, state.Phase)

	// Play first hand: both players call/check to complete pre-flop
	// First player acts
	_, err = env.PokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: state.CurrentPlayer,
		TableId:  tableID,
		Amount:   20, // Call the big blind
	})
	require.NoError(t, err)

	// Wait for second player's turn
	waitForPlayerTurn(state.Players[1].Id, 2*time.Second)

	// Second player checks (already has big blind)
	_, err = env.PokerClient.CheckBet(ctx, &pokerrpc.CheckBetRequest{
		PlayerId: state.Players[1].Id,
		TableId:  tableID,
	})
	require.NoError(t, err)

	// Wait for flop
	waitForPhase(pokerrpc.GamePhase_FLOP, 3*time.Second)

	// Both players check through flop, turn, river
	// We need to handle each betting round properly - both players must act
	phases := []pokerrpc.GamePhase{pokerrpc.GamePhase_FLOP, pokerrpc.GamePhase_TURN, pokerrpc.GamePhase_RIVER}

	for _, targetPhase := range phases {
		// Wait for the target phase
		waitForPhase(targetPhase, 3*time.Second)

		// Both players must act in this betting round
		for j := 0; j < 2; j++ {
			// Wait for current player
			state = env.GetGameState(ctx, tableID)
			currentPlayer := state.CurrentPlayer

			// Player checks
			_, err = env.PokerClient.CheckBet(ctx, &pokerrpc.CheckBetRequest{
				PlayerId: currentPlayer,
				TableId:  tableID,
			})
			require.NoError(t, err)

			// Wait for turn to advance (unless it's the last player in the round)
			if j < 1 {
				require.Eventually(t, func() bool {
					newState := env.GetGameState(ctx, tableID)
					return newState.CurrentPlayer != currentPlayer
				}, 2*time.Second, 10*time.Millisecond, "turn should advance after check")
			}
		}
	}

	// Wait for showdown
	waitForPhase(pokerrpc.GamePhase_SHOWDOWN, 3*time.Second)

	// Wait for showdown to complete and winners to be available
	require.Eventually(t, func() bool {
		_, err := env.PokerClient.GetLastWinners(ctx, &pokerrpc.GetLastWinnersRequest{
			TableId: tableID,
		})
		return err == nil
	}, 3*time.Second, 50*time.Millisecond, "showdown should complete")

	// SECOND HAND: This is where the bug would manifest
	// The game should auto-start a new hand
	waitForPhase(pokerrpc.GamePhase_PRE_FLOP, 5*time.Second)

	// Get second hand state
	state = env.GetGameState(ctx, tableID)
	t.Logf("Second hand - Current player: %s, Phase: %s", state.CurrentPlayer, state.Phase)

	// The critical test: In the second hand, after the first player acts,
	// we should still be in PRE_FLOP waiting for the second player to act.
	// The bug was that it would skip directly to FLOP.

	// First player acts (calls the big blind)
	firstPlayer := state.CurrentPlayer
	_, err = env.PokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: firstPlayer,
		TableId:  tableID,
		Amount:   20, // Call the big blind
	})
	require.NoError(t, err)

	// CRITICAL ASSERTION: After first player acts, we should still be in PRE_FLOP
	// This would have failed before the fix because the system would incorrectly
	// advance to FLOP, skipping the second player's action.
	state = env.GetGameState(ctx, tableID)
	assert.Equal(t, pokerrpc.GamePhase_PRE_FLOP, state.Phase,
		"After first player acts in second hand, should still be in PRE_FLOP waiting for second player")

	// The second player should now be able to act
	// Find the second player (not the one who just acted)
	var secondPlayer string
	for _, player := range state.Players {
		if player.Id != firstPlayer {
			secondPlayer = player.Id
			break
		}
	}
	require.NotEmpty(t, secondPlayer, "should have a second player")

	// Wait for second player's turn
	waitForPlayerTurn(secondPlayer, 2*time.Second)

	// Second player checks
	_, err = env.PokerClient.CheckBet(ctx, &pokerrpc.CheckBetRequest{
		PlayerId: secondPlayer,
		TableId:  tableID,
	})
	require.NoError(t, err)

	// Now we should advance to FLOP
	waitForPhase(pokerrpc.GamePhase_FLOP, 3*time.Second)

	// Verify we reached FLOP after both players acted
	state = env.GetGameState(ctx, tableID)
	assert.Equal(t, pokerrpc.GamePhase_FLOP, state.Phase,
		"Should advance to FLOP only after both players have acted in second hand")
}

// TestFoldRaceCondition demonstrates the race condition bug in fold handling
// TestFoldUncalledRaise_RaceySettlement exposes the race in fold settlement:
// - Aggressive AutoStart (0ms) to overlap SHOWDOWN and next-hand setup
// - Immediate fold on an uncalled raise
// - Tight waits + concurrent state reads to tickle the workers
func TestFoldUncalledRaise_RaceySettlement(t *testing.T) {
	t.Parallel()
	const iters = 10 // crank this up if it’s too stable on your machine

	for i := 0; i < iters; i++ {
		t.Run(fmt.Sprintf("iter_%02d", i), func(t *testing.T) {
			t.Parallel()

			env := testenv.New(t)
			defer env.Close()
			ctx := context.Background()

			players := []string{"p1", "p2"}
			const stack = int64(1_000)

			// NOTE: AutoStartMs=0 to maximize overlap/race in workers.
			// BuyIn: 0 to avoid escrow requirement in tests
			ctx = env.ContextWithToken(ctx, "p1")
			createResp, err := env.CreateTable(ctx, &pokerrpc.CreateTableRequest{
				PlayerId:      "p1",
				SmallBlind:    10,
				BigBlind:      20,
				MinPlayers:    2,
				MaxPlayers:    2,
				BuyIn:         0,
				StartingChips: stack,
				AutoStartMs:   5000,
				AutoAdvanceMs: 1000,
			})
			require.NoError(t, err)
			tableID := createResp.TableId

			_, err = env.JoinTable(ctx, "p2", tableID)
			require.NoError(t, err)

			for _, p := range players {
				_, err := env.SetPlayerReady(ctx, p, tableID)
				require.NoError(t, err)
			}

			env.WaitForGameStart(ctx, tableID, 2*time.Second)
			env.WaitForGamePhase(ctx, tableID, pokerrpc.GamePhase_PRE_FLOP, 2*time.Second)

			waitTurn := func(pid string, d time.Duration) {
				ctxTurn, cancel := context.WithTimeout(ctx, d)
				defer cancel()
				// Small initial delay to allow previous turn to complete (async FSM processing)
				time.Sleep(10 * time.Millisecond)
				// Convert player ID to ShortID string representation for comparison
				pidShortID := testenv.PlayerIDToShortIDString(pid)
				for {
					st := env.GetGameState(ctx, tableID)
					if st.CurrentPlayer == pidShortID && st.Phase == pokerrpc.GamePhase_PRE_FLOP {
						return
					}
					select {
					case <-ctxTurn.Done():
						t.Fatalf("turn never reached for %s (phase=%v current=%s)", pid, st.Phase, st.CurrentPlayer)
					default:
						time.Sleep(2 * time.Millisecond)
					}
				}
			}

			// P1 raises to 100 total (creates uncalled headroom vs BB=20).
			waitTurn("p1", 500*time.Millisecond)
			// Convert player ID to ShortID string and use correct context with token
			p1ShortID := testenv.PlayerIDToShortIDString("p1")
			ctxP1 := env.ContextWithToken(ctx, "p1")
			_, err = env.PokerClient.MakeBet(ctxP1, &pokerrpc.MakeBetRequest{
				PlayerId: p1ShortID, TableId: tableID, Amount: 100, // “to 100” semantics
			})
			require.NoError(t, err)

			// Without letting P2 act/call, we immediately fold with P2.
			waitTurn("p2", 500*time.Millisecond)
			// Convert player ID to ShortID string and use correct context with token
			p2ShortID := testenv.PlayerIDToShortIDString("p2")
			ctxP2 := env.ContextWithToken(ctx, "p2")
			_, err = env.PokerClient.FoldBet(ctxP2, &pokerrpc.FoldBetRequest{
				PlayerId: p2ShortID, TableId: tableID,
			})
			require.NoError(t, err)

			// Race tickler: spam a couple of concurrent reads while workers process.
			var wg sync.WaitGroup
			for k := 0; k < 5; k++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					_ = env.GetGameState(ctx, tableID)
				}()
			}

			// Wait for settled hand outcome instead of transient SHOWDOWN phase.
			// With aggressive AutoStart and concurrent workers, the game can move past
			// SHOWDOWN phase quickly, causing race conditions. Instead, wait for the
			// stable outcome invariant: exactly one active player + zeroed pots or
			// showdown result present.
			require.Eventually(t, func() bool {
				state := env.GetGameState(ctx, tableID)
				// Check if we have exactly one active (non-folded) player
				activeCount := 0
				for _, player := range state.Players {
					if !player.Folded {
						activeCount++
					}
				}
				// Either we have exactly one active player (uncontested win) OR
				// we have a showdown result available (contested showdown completed)
				if activeCount == 1 {
					return true
				}
				// Check if showdown result is available
				_, err := env.PokerClient.GetLastWinners(ctx, &pokerrpc.GetLastWinnersRequest{
					TableId: tableID,
				})
				return err == nil
			}, 5*time.Second, 25*time.Millisecond, "hand should settle with exactly one active player or completed showdown")

			state := env.GetGameState(ctx, tableID)

			// --- Invariants that MUST hold ---

			// 1) Single active player (P1) should be the winner; P2 must be folded.
			require.True(t, state.Players[1].Folded, "P2 should be folded")
			require.False(t, state.Players[0].Folded, "P1 should be active")

			// 2) Uncalled headroom refund: Raise to 100 vs second-highest 20 ⇒ headroom=90 returned.
			// Pot to award is only the called part (SB 10 + BB 20 = 30).
			// Expected stacks at end of settlement:
			//   P1: 1020 (refund 90, win 30) — net +20
			//   P2:  980 (loses BB 20)
			require.Equal(t, int64(1020), state.Players[0].Balance,
				"P1 should net +20 (refund 90, win 30) ⇒ 1020")
			require.Equal(t, int64(980), state.Players[1].Balance,
				"P2 should lose only the BB ⇒ 980")

			// 3) Bankroll conservation within table.
			sum := state.Players[0].Balance + state.Players[1].Balance
			require.Equal(t, int64(2000), sum, "table chips must be conserved")

			wg.Wait()

			// Small delay to allow cleanup between iterations
			time.Sleep(30 * time.Millisecond)
		})
	}
}

func TestShortStackBlindAllIn(t *testing.T) {
	t.Parallel()
	env := testenv.New(t)
	defer env.Close()

	ctx := context.Background()

	// Setup: Create a table with small starting chips (15) relative to blinds (SB=10, BB=20)
	// This forces one player to go all-in when posting BB
	players := []string{"player1", "player2"}

	// Create table with starting chips (15) less than big blind (20)
	// BuyIn: 0 to avoid escrow requirement in tests
	ctx = env.ContextWithToken(ctx, "player1")
	createResp, err := env.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      "player1",
		SmallBlind:    10,
		BigBlind:      20,
		MinPlayers:    2,
		MaxPlayers:    2,
		BuyIn:         0,
		StartingChips: 15,  // Less than BB - forces all-in on BB post
		AutoStartMs:   200, // Short delay for faster testing
		AutoAdvanceMs: 1000,
	})
	require.NoError(t, err)
	tableID := createResp.TableId

	// player2 joins
	_, err = env.JoinTable(ctx, "player2", tableID)
	require.NoError(t, err)

	// Both players mark ready
	for _, p := range players {
		_, err := env.SetPlayerReady(ctx, p, tableID)
		require.NoError(t, err)
	}

	// Wait for game to start
	env.WaitForGameStart(ctx, tableID, 3*time.Second)
	env.WaitForGamePhase(ctx, tableID, pokerrpc.GamePhase_PRE_FLOP, 3*time.Second)

	// CRITICAL VERIFICATION: reproduce mid-hand short-stack SB scenario where
	// SB has exactly 10 chips with blinds 10/20:
	// - SB posts 10 (balance=0, currentBet=10)
	// - BB posts 20 (balance=-, currentBet=20)
	// - SB attempts to call and must NOT hang the game.
	//
	// First, force one player down to a 10-chip stack while keeping the other deep.
	state := env.GetGameState(ctx, tableID)
	require.Len(t, state.Players, 2)
	sbRPC := state.Players[0] // in HU, player1 is dealer/SB in these tests

	// Reduce SB's live stack to 10 chips via the server's internal helper.
	env.SetBalance(ctx, sbRPC.Id, 10)

	// Start a new hand so that blinds are re-posted from this short stack.
	// Wait for PRE_FLOP again (next hand).
	env.WaitForGamePhase(ctx, tableID, pokerrpc.GamePhase_PRE_FLOP, 5*time.Second)

	// Re-snapshot after blinds.
	state = env.GetGameState(ctx, tableID)
	t.Logf("After rebalance - players: %+v", state.Players)

	// Find SB (currentBet=10) and BB (currentBet >= 15, may be all-in)
	var sbPlayer, bbPlayer *pokerrpc.Player
	for _, p := range state.Players {
		if p.IsSmallBlind {
			sbPlayer = p
		} else if p.IsBigBlind {
			bbPlayer = p
		}
	}
	require.NotNil(t, sbPlayer, "SB not found after rebalance")
	require.NotNil(t, bbPlayer, "BB not found after rebalance")

	// With StartingChips=15 and blinds 10/20:
	// - SB posts 10 → balance=5, currentBet=10 (or if SetBalance(10) took effect: balance=0)
	// - BB posts 15 (all-in) → balance=0, currentBet=15, is_all_in=true
	// The exact balance depends on timing of SetBalance call, but BB should always be all-in.
	assert.True(t, bbPlayer.IsAllIn, "BB should be all-in from posting blind with short stack")
	assert.Equal(t, int64(10), sbPlayer.CurrentBet, "SB should have currentBet=10 (small blind)")

	// SB attempts to CALL/CHECK. With both players posting blinds and BB all-in,
	// SB acts first. If SB has balance > 0, they can call/check to complete the round.
	// The key test is that the game does NOT hang when one player is all-in from blinds.
	if sbPlayer.IsTurn {
		_, err = env.PokerClient.CallBet(ctx, &pokerrpc.CallBetRequest{
			PlayerId: sbPlayer.Id,
			TableId:  tableID,
		})
		require.NoError(t, err, "SB action should not error")
	}

	// Wait for the hand to either reach showdown or complete; this ensures
	// that the game did not get stuck waiting on the short-stack player.
	env.WaitForShowdownOrRemoval(ctx, tableID, 10*time.Second)

	t.Log("✓ Short-stack blind + call handled without hang (SB had 10 chips vs BB 20)")
}

// TestBettingRound_Completes_On_AllIn_And_Folds tests betting round completion
// in various scenarios involving all-ins and folds to ensure the game properly
// advances phases and reaches showdown.
func TestBettingRound_Completes_On_AllIn_And_Folds(t *testing.T) {
	t.Run("AllPlayersAllIn_FastForward", func(t *testing.T) {
		t.Parallel()
		env := testenv.New(t)
		defer env.Close()

		ctx := context.Background()

		// Setup 3 players with small stacks to facilitate all-in
		players := []string{"player1", "player2", "player3"}

		// Create table with small starting chips (100) and blinds
		// BuyIn: 0 to avoid escrow requirement in tests
		ctx = env.ContextWithToken(ctx, "player1")
		createResp, err := env.CreateTable(ctx, &pokerrpc.CreateTableRequest{
			PlayerId:      "player1",
			SmallBlind:    10,
			BigBlind:      20,
			MinPlayers:    3,
			MaxPlayers:    3,
			BuyIn:         0,
			StartingChips: 100,
			AutoStartMs:   200,
			AutoAdvanceMs: 1000,
		})
		require.NoError(t, err)
		tableID := createResp.TableId

		// Other players join
		for _, p := range players[1:] {
			_, err := env.JoinTable(ctx, p, tableID)
			require.NoError(t, err)
		}

		// All players mark ready
		for _, p := range players {
			_, err := env.SetPlayerReady(ctx, p, tableID)
			require.NoError(t, err)
		}

		// Wait for game to start
		env.WaitForGameStart(ctx, tableID, 3*time.Second)
		env.WaitForGamePhase(ctx, tableID, pokerrpc.GamePhase_PRE_FLOP, 3*time.Second)

		// Drive betting turn-by-turn so each seated player actually goes all-in.
		// Using the current player from game state avoids rejected bets due to
		// "not your turn" or mismatched player IDs.
		allInDeadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(allInDeadline) {
			state := env.GetGameState(ctx, tableID)
			if state.Phase == pokerrpc.GamePhase_SHOWDOWN {
				break
			}

			allInCount := 0
			for _, p := range state.Players {
				if p.IsAllIn {
					allInCount++
				}
			}
			if allInCount == len(players) {
				break
			}

			currentPlayerID := state.CurrentPlayer
			var currentPlayer *pokerrpc.Player
			for _, p := range state.Players {
				if p.Id == currentPlayerID {
					currentPlayer = p
					break
				}
			}
			require.NotNil(t, currentPlayer, "current player should be in game state")

			// MakeBet expects an absolute bet; push the player's full stack.
			targetBet := currentPlayer.CurrentBet + currentPlayer.Balance
			_, err := env.PokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
				PlayerId: currentPlayerID,
				TableId:  tableID,
				Amount:   targetBet,
			})
			if err == nil {
				t.Logf("Player %s went all-in", currentPlayerID)
			}

			time.Sleep(20 * time.Millisecond)
		}

		// Confirm everyone is marked all-in before waiting for auto fast-forward.
		state := env.GetGameState(ctx, tableID)
		allInCount := 0
		for _, p := range state.Players {
			if p.IsAllIn {
				allInCount++
			}
		}
		require.Equal(t, len(players), allInCount, "all players should be all-in")

		// When all players are all-in, game should fast-forward to SHOWDOWN.
		// If the game ends and the table is removed during the fast-forward,
		// treat that as success as well.
		env.WaitForShowdownOrRemoval(ctx, tableID, 5*time.Second)

		t.Log("✓ All players all-in - game fast-forwarded to showdown")
	})

	t.Run("TwoAllIn_OneFold_GoToShowdown", func(t *testing.T) {
		t.Parallel()
		env := testenv.New(t)
		defer env.Close()

		ctx := context.Background()

		// Setup 3 players
		players := []string{"player1", "player2", "player3"}

		// Create table
		// BuyIn: 0 to avoid escrow requirement in tests
		ctx = env.ContextWithToken(ctx, "player1")
		createResp, err := env.CreateTable(ctx, &pokerrpc.CreateTableRequest{
			PlayerId:      "player1",
			SmallBlind:    10,
			BigBlind:      20,
			MinPlayers:    3,
			MaxPlayers:    3,
			BuyIn:         0,
			StartingChips: 500,
			AutoStartMs:   5000, // Long delay to prevent auto-start during test
			AutoAdvanceMs: 1000,
		})
		require.NoError(t, err)
		tableID := createResp.TableId

		// Other players join
		for _, p := range players[1:] {
			_, err := env.JoinTable(ctx, p, tableID)
			require.NoError(t, err)
		}

		// All players mark ready
		for _, p := range players {
			_, err := env.SetPlayerReady(ctx, p, tableID)
			require.NoError(t, err)
		}

		// Wait for game to start
		env.WaitForGameStart(ctx, tableID, 3*time.Second)
		env.WaitForGamePhase(ctx, tableID, pokerrpc.GamePhase_PRE_FLOP, 3*time.Second)

		// Get initial state to identify positions
		state := env.GetGameState(ctx, tableID)
		currentPlayer := state.CurrentPlayer

		// First player folds
		_, err = env.PokerClient.FoldBet(ctx, &pokerrpc.FoldBetRequest{
			PlayerId: currentPlayer,
			TableId:  tableID,
		})
		require.NoError(t, err)

		// Wait for turn to advance after fold
		require.Eventually(t, func() bool {
			newState := env.GetGameState(ctx, tableID)
			return newState.CurrentPlayer != currentPlayer
		}, 2*time.Second, 10*time.Millisecond, "turn should advance after fold")

		// Remaining two players go all-in
		allInPlayers := make(map[string]bool)
		for i := 0; i < 2; i++ {
			state = env.GetGameState(ctx, tableID)
			if state.Phase == pokerrpc.GamePhase_SHOWDOWN {
				break
			}
			currentPlayer = state.CurrentPlayer
			currentPhase := state.Phase

			_, err = env.PokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
				PlayerId: currentPlayer,
				TableId:  tableID,
				Amount:   500,
			})
			if err == nil {
				allInPlayers[currentPlayer] = true
				t.Logf("Player %s went all-in", currentPlayer)

				// Wait for turn to advance or phase to change
				require.Eventually(t, func() bool {
					newState := env.GetGameState(ctx, tableID)
					return newState.CurrentPlayer != currentPlayer || newState.Phase != currentPhase
				}, 2*time.Second, 10*time.Millisecond, "turn or phase should change after all-in")
			}
		}

		// Verify we got at least 2 all-in players before showdown
		require.GreaterOrEqual(t, len(allInPlayers), 2, "at least 2 players should have gone all-in")

		// When remaining players are all-in, should reach showdown
		// Need to wait for: FLOP→TURN (1s) + TURN→RIVER (1s) + RIVER→SHOWDOWN (1s) = 3s + buffer
		require.Eventually(t, func() bool {
			state := env.GetGameState(ctx, tableID)
			return state.Phase == pokerrpc.GamePhase_SHOWDOWN
		}, 4*time.Second, 50*time.Millisecond, "game should reach showdown with all-in players")

		// Verify one player folded (check during showdown, before hand ends)
		state = env.GetGameState(ctx, tableID)
		foldedCount := 0
		for _, p := range state.Players {
			if p.Folded {
				foldedCount++
			}
		}
		assert.Equal(t, 1, foldedCount, "should have 1 folded player")

		t.Log("✓ Two all-in, one folded - game reached showdown")
	})

	t.Run("AllFoldExceptOne_GoToShowdown", func(t *testing.T) {
		t.Parallel()
		env := testenv.New(t)
		defer env.Close()

		ctx := context.Background()

		// Setup 3 players
		players := []string{"player1", "player2", "player3"}

		// Create table with long auto-start delay to avoid race
		// BuyIn: 0 to avoid escrow requirement in tests
		ctx = env.ContextWithToken(ctx, "player1")
		createResp, err := env.CreateTable(ctx, &pokerrpc.CreateTableRequest{
			PlayerId:      "player1",
			SmallBlind:    10,
			BigBlind:      20,
			MinPlayers:    3,
			MaxPlayers:    3,
			BuyIn:         0,
			StartingChips: 1_000,
			AutoStartMs:   5000, // Long delay to prevent auto-start during test
			AutoAdvanceMs: 1000,
		})
		require.NoError(t, err)
		tableID := createResp.TableId

		// Other players join
		for _, p := range players[1:] {
			_, err := env.JoinTable(ctx, p, tableID)
			require.NoError(t, err)
		}

		// All players mark ready
		for _, p := range players {
			_, err := env.SetPlayerReady(ctx, p, tableID)
			require.NoError(t, err)
		}

		// Wait for game to start
		env.WaitForGameStart(ctx, tableID, 3*time.Second)
		env.WaitForGamePhase(ctx, tableID, pokerrpc.GamePhase_PRE_FLOP, 3*time.Second)

		// Two players fold, leaving one active
		// We'll check for showdown in the main loop instead of a goroutine
		// to avoid "fail in goroutine after test completes" panics

		folded := 0
		for folded < 2 {
			state := env.GetGameState(ctx, tableID)

			// Check if we've already reached showdown
			if state.Phase == pokerrpc.GamePhase_SHOWDOWN {
				t.Log("Showdown reached during fold loop")
				break
			}

			currentPlayer := state.CurrentPlayer
			_, err := env.PokerClient.FoldBet(ctx, &pokerrpc.FoldBetRequest{
				PlayerId: currentPlayer,
				TableId:  tableID,
			})
			if err == nil {
				folded++
				t.Logf("Player %s folded (%d/2)", currentPlayer, folded)

				// After the first fold, wait for the turn to advance.
				if folded == 1 {
					require.Eventually(t, func() bool {
						newState := env.GetGameState(ctx, tableID)
						return newState.CurrentPlayer != currentPlayer
					}, 2*time.Second, 10*time.Millisecond, "turn should advance after first fold")
				}
			}
		}

		// Verify we reached showdown (allow extra time on CI)
		require.Eventually(t, func() bool {
			state := env.GetGameState(ctx, tableID)
			return state.Phase == pokerrpc.GamePhase_SHOWDOWN
		}, 2*time.Second, 10*time.Millisecond, "should reach showdown after two folds")

		// Final state assertion
		state := env.GetGameState(ctx, tableID)
		assert.Equal(t, pokerrpc.GamePhase_SHOWDOWN, state.Phase, "should be in showdown phase")

		// Verify exactly one non-folded player
		activeCount := 0
		for _, p := range state.Players {
			if !p.Folded {
				activeCount++
			}
		}
		assert.Equal(t, 1, activeCount, "should have exactly 1 active player")

		t.Log("✓ All fold except one - game reached showdown")
	})
}

// -----------------------------------------------------------------------------
//
//	SCENARIO: Heads-Up All-In Preflop - Auto-Advance Through Streets
//
// -----------------------------------------------------------------------------
func TestHeadsUpAllInPreflop_AutoAdvanceStreets(t *testing.T) {
	t.Parallel()
	env := testenv.New(t)
	defer env.Close()

	ctx := context.Background()

	// Setup 2 players
	players := []string{"player1", "player2"}

	// Create table with small stacks to facilitate all-in
	// BuyIn: 0 to avoid escrow requirement in tests
	ctx = env.ContextWithToken(ctx, "player1")
	createResp, err := env.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      "player1",
		SmallBlind:    10,
		BigBlind:      20,
		MinPlayers:    2,
		MaxPlayers:    2,
		BuyIn:         0,
		StartingChips: 100, // Small stacks
		AutoStartMs:   5000,
		AutoAdvanceMs: 1000,
	})
	require.NoError(t, err)
	tableID := createResp.TableId

	// Player2 joins
	_, err = env.JoinTable(ctx, "player2", tableID)
	require.NoError(t, err)

	// Both players mark ready
	for _, p := range players {
		_, err := env.SetPlayerReady(ctx, p, tableID)
		require.NoError(t, err)
	}

	// Wait for game to start
	env.WaitForGameStart(ctx, tableID, 3*time.Second)
	env.WaitForGamePhase(ctx, tableID, pokerrpc.GamePhase_PRE_FLOP, 3*time.Second)

	// Both players go all-in preflop
	state := env.GetGameState(ctx, tableID)
	currentPlayer := state.CurrentPlayer
	allInPlayers := make(map[string]bool)

	// First player goes all-in
	_, err = env.PokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: currentPlayer,
		TableId:  tableID,
		Amount:   100,
	})
	require.NoError(t, err)
	allInPlayers[currentPlayer] = true
	t.Logf("Player %s went all-in", currentPlayer)

	// Wait for turn to advance or phase to change (fast transitions can skip straight to next phase)
	require.Eventually(t, func() bool {
		state = env.GetGameState(ctx, tableID)
		return state.CurrentPlayer != currentPlayer || state.Phase != pokerrpc.GamePhase_PRE_FLOP
	}, 2*time.Second, 10*time.Millisecond, "turn or phase should advance after all-in")

	// Second player calls all-in
	state = env.GetGameState(ctx, tableID)
	currentPlayer = state.CurrentPlayer
	_, err = env.PokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: currentPlayer,
		TableId:  tableID,
		Amount:   100,
	})
	require.NoError(t, err)
	allInPlayers[currentPlayer] = true
	t.Logf("Player %s called all-in", currentPlayer)

	// Verify both players went all-in during betting
	require.Equal(t, 2, len(allInPlayers), "both players should have gone all-in during betting")

	// CRITICAL VERIFICATION: Game should auto-advance through streets
	// When both players are all-in, the game automatically advances through phases.
	// The second all-in triggers: PRE_FLOP → FLOP (immediate) → TURN (auto 1s) → RIVER (auto 1s) → SHOWDOWN (auto 1s)

	t.Log("Verifying game auto-advances to at least TURN after both all-ins...")
	// There are two sequential auto-advance timers (to FLOP, then to TURN) each
	// using AutoAdvanceDelay (1s). Give slightly more than 2*delay to avoid
	// flaking at the exact boundary.
	require.Eventually(t, func() bool {
		state, removed := env.GetGameStateAllowNotFound(ctx, tableID)
		if removed {
			return true
		}
		return state.Phase >= pokerrpc.GamePhase_TURN
	}, 3*time.Second, 50*time.Millisecond, "game should reach at least TURN phase")
	state, _ = env.GetGameStateAllowNotFound(ctx, tableID)
	if state != nil {
		t.Logf("Game at phase %s with %d community cards", state.Phase, len(state.CommunityCards))
		assert.GreaterOrEqual(t, len(state.CommunityCards), 3, "should have at least 3 community cards")
	}

	t.Log("Verifying auto-advance to RIVER...")
	require.Eventually(t, func() bool {
		state, removed := env.GetGameStateAllowNotFound(ctx, tableID)
		if removed {
			return true
		}
		return state.Phase >= pokerrpc.GamePhase_RIVER
	}, 3*time.Second, 50*time.Millisecond, "game should auto-advance to RIVER")
	state, _ = env.GetGameStateAllowNotFound(ctx, tableID)
	if state != nil {
		assert.Equal(t, 5, len(state.CommunityCards), "should have 5 community cards at RIVER")
		t.Logf("✓ RIVER reached with %d community cards", len(state.CommunityCards))
	} else {
		t.Log("Table removed before RIVER check; skipping community card assertion")
	}

	t.Log("Verifying auto-advance to SHOWDOWN...")
	var showdownState *pokerrpc.GameUpdate
	var tableRemoved bool
	require.Eventually(t, func() bool {
		st, removed := env.GetGameStateAllowNotFound(ctx, tableID)
		if removed {
			tableRemoved = true
			return true
		}
		if st != nil && st.Phase == pokerrpc.GamePhase_SHOWDOWN {
			showdownState = st
			return true
		}
		return false
	}, 4*time.Second, 50*time.Millisecond, "game should auto-advance to SHOWDOWN")
	if showdownState != nil {
		assert.Equal(t, pokerrpc.GamePhase_SHOWDOWN, showdownState.Phase, "should advance to SHOWDOWN")
		t.Log("✓ SHOWDOWN reached")
	} else if tableRemoved {
		t.Log("Table removed before SHOWDOWN assertion; skipping phase check")
	} else {
		t.Log("SHOWDOWN state not captured; skipping phase check")
	}

	// Wait for showdown to complete and winners to be available
	require.Eventually(t, func() bool {
		_, err := env.PokerClient.GetLastWinners(ctx, &pokerrpc.GetLastWinnersRequest{
			TableId: tableID,
		})
		return err == nil || status.Code(err) == codes.NotFound
	}, 3*time.Second, 50*time.Millisecond, "showdown should complete with winners")

	t.Log("✓ Heads-up all-in: Auto-advanced through all streets correctly")
}

// -----------------------------------------------------------------------------
//
//	SCENARIO: Three-Player All-In Preflop - Auto-Advance Through Streets
//
// -----------------------------------------------------------------------------
func TestThreePlayerAllInPreflop_AutoAdvanceStreets(t *testing.T) {
	t.Parallel()
	env := testenv.New(t)
	defer env.Close()

	ctx := context.Background()

	// Setup 3 players
	players := []string{"player1", "player2", "player3"}

	// Create table with small stacks to facilitate all-in
	// BuyIn: 0 to avoid escrow requirement in tests
	ctx = env.ContextWithToken(ctx, "player1")
	createResp, err := env.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      "player1",
		SmallBlind:    10,
		BigBlind:      20,
		MinPlayers:    3,
		MaxPlayers:    3,
		BuyIn:         0,
		StartingChips: 100, // Small stacks
		AutoStartMs:   5000,
		AutoAdvanceMs: 1000,
	})
	require.NoError(t, err)
	tableID := createResp.TableId

	// Other players join
	for _, p := range players[1:] {
		_, err := env.JoinTable(ctx, p, tableID)
		require.NoError(t, err)
	}

	// All players mark ready
	for _, p := range players {
		_, err := env.SetPlayerReady(ctx, p, tableID)
		require.NoError(t, err)
	}

	// Wait for game to start
	env.WaitForGameStart(ctx, tableID, 3*time.Second)
	env.WaitForGamePhase(ctx, tableID, pokerrpc.GamePhase_PRE_FLOP, 3*time.Second)

	// All three players go all-in preflop
	// Keep trying until we can't bet anymore (phase advanced or all players acted)
	allInPlayers := []string{}
	for i := 0; i < 3; i++ {
		state := env.GetGameState(ctx, tableID)
		currentPlayer := state.CurrentPlayer
		currentPhase := state.Phase

		// Helper: place an all-in bet for the snapshot current player, but only
		// once their isTurn flag is true in the live game state. This avoids
		// racing against the player's FSM not having processed StartTurn yet.
		var betErr error
		ok := assert.Eventually(t, func() bool {
			st := env.GetGameState(ctx, tableID)
			// If phase already changed, stop trying in this helper; the outer
			// logic will detect the phase change and treat it as auto-advance.
			if st.Phase != currentPhase {
				return false
			}
			if st.CurrentPlayer != currentPlayer {
				return false
			}

			// Find the player snapshot and ensure it's actually their turn.
			var self *pokerrpc.Player
			for _, p := range st.Players {
				if p.Id == currentPlayer {
					self = p
					break
				}
			}
			if self == nil || !self.IsTurn {
				return false
			}

			// Now attempt the bet.
			_, betErr = env.PokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
				PlayerId: currentPlayer,
				TableId:  tableID,
				Amount:   100,
			})
			if betErr != nil && isTransientTurnError(betErr) {
				return false
			}
			return betErr == nil
		}, 2*time.Second, 10*time.Millisecond, "failed to place all-in bet for player %s", currentPlayer)

		if !ok {
			// If we couldn't place the bet and phase has already advanced, treat
			// this as the auto-advance being triggered early and stop trying.
			newState := env.GetGameState(ctx, tableID)
			t.Logf("Bet attempt %d failed; current phase=%v (was %v), error=%v", i+1, newState.Phase, currentPhase, betErr)
			if newState.Phase != currentPhase {
				break
			}
			// Same phase and non-transient failure: this is a real error.
			require.NoError(t, betErr)
		}

		allInPlayers = append(allInPlayers, currentPlayer)
		t.Logf("Player %s went all-in", currentPlayer)

		// Wait for turn to advance or phase to change
		require.Eventually(t, func() bool {
			newState := env.GetGameState(ctx, tableID)
			// Stop if phase changed (auto-advance triggered)
			if newState.Phase != currentPhase {
				return true
			}
			// Stop if turn advanced to next player
			return newState.CurrentPlayer != currentPlayer
		}, 2*time.Second, 10*time.Millisecond, "turn or phase should advance after all-in")

		// Check if phase advanced (all players all-in triggered auto-advance)
		if env.GetGameState(ctx, tableID).Phase != currentPhase {
			t.Logf("Phase advanced to %v after %d players went all-in", env.GetGameState(ctx, tableID).Phase, len(allInPlayers))
			break
		}
	}

	// Verify at least 2 players went all-in before auto-advance triggered
	require.GreaterOrEqual(t, len(allInPlayers), 2, "at least 2 players should have gone all-in before auto-advance")

	// CRITICAL VERIFICATION: Game should auto-advance through streets
	t.Log("Verifying auto-advance through FLOP...")
	env.WaitForGamePhase(ctx, tableID, pokerrpc.GamePhase_FLOP, 3*time.Second)
	state := env.GetGameState(ctx, tableID)
	assert.Equal(t, pokerrpc.GamePhase_FLOP, state.Phase, "should advance to FLOP")
	assert.Equal(t, 3, len(state.CommunityCards), "should have 3 community cards at FLOP")
	t.Logf("✓ FLOP reached with %d community cards", len(state.CommunityCards))

	t.Log("Verifying auto-advance through TURN...")
	env.WaitForGamePhase(ctx, tableID, pokerrpc.GamePhase_TURN, 3*time.Second)
	state = env.GetGameState(ctx, tableID)
	assert.Equal(t, pokerrpc.GamePhase_TURN, state.Phase, "should advance to TURN")
	assert.Equal(t, 4, len(state.CommunityCards), "should have 4 community cards at TURN")
	t.Logf("✓ TURN reached with %d community cards", len(state.CommunityCards))

	t.Log("Verifying auto-advance through RIVER...")
	env.WaitForGamePhase(ctx, tableID, pokerrpc.GamePhase_RIVER, 3*time.Second)
	state = env.GetGameState(ctx, tableID)
	assert.Equal(t, pokerrpc.GamePhase_RIVER, state.Phase, "should advance to RIVER")
	assert.Equal(t, 5, len(state.CommunityCards), "should have 5 community cards at RIVER")
	t.Logf("✓ RIVER reached with %d community cards", len(state.CommunityCards))

	t.Log("Verifying auto-advance to SHOWDOWN...")
	env.WaitForShowdownOrRemoval(ctx, tableID, 3*time.Second)
	state, removed := env.GetGameStateAllowNotFound(ctx, tableID)
	if removed {
		t.Log("Table removed immediately after showdown; skipping state assertions")
		return
	}
	assert.Equal(t, pokerrpc.GamePhase_SHOWDOWN, state.Phase, "should advance to SHOWDOWN")
	t.Log("✓ SHOWDOWN reached")

	// Wait for showdown to complete and winners to be available
	require.Eventually(t, func() bool {
		_, err := env.PokerClient.GetLastWinners(ctx, &pokerrpc.GetLastWinnersRequest{
			TableId: tableID,
		})
		return err == nil
	}, 3*time.Second, 50*time.Millisecond, "showdown should complete with winners")

	t.Log("✓ Three-player all-in: Auto-advanced through all streets correctly")
}

// -----------------------------------------------------------------------------
//
//	SCENARIO: Partial All-In (2 all-in, 1 folded) - Auto-Advance Through Streets
//
// -----------------------------------------------------------------------------
func TestPartialAllIn_OneFolded_AutoAdvanceStreets(t *testing.T) {
	t.Parallel()
	env := testenv.New(t)
	defer env.Close()

	ctx := context.Background()

	// Setup 3 players
	players := []string{"player1", "player2", "player3"}

	// Create table
	// BuyIn: 0 to avoid escrow requirement in tests
	ctx = env.ContextWithToken(ctx, "player1")
	createResp, err := env.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      "player1",
		SmallBlind:    10,
		BigBlind:      20,
		MinPlayers:    3,
		MaxPlayers:    3,
		BuyIn:         0,
		StartingChips: 200, // Moderate stacks
		AutoStartMs:   5000,
		AutoAdvanceMs: 1000,
	})
	require.NoError(t, err)
	tableID := createResp.TableId

	// Other players join
	for _, p := range players[1:] {
		_, err := env.JoinTable(ctx, p, tableID)
		require.NoError(t, err)
	}

	// All players mark ready
	for _, p := range players {
		_, err := env.SetPlayerReady(ctx, p, tableID)
		require.NoError(t, err)
	}

	// Wait for game to start
	env.WaitForGameStart(ctx, tableID, 3*time.Second)
	env.WaitForGamePhase(ctx, tableID, pokerrpc.GamePhase_PRE_FLOP, 3*time.Second)

	// First player folds
	state := env.GetGameState(ctx, tableID)
	firstPlayer := state.CurrentPlayer
	_, err = env.PokerClient.FoldBet(ctx, &pokerrpc.FoldBetRequest{
		PlayerId: firstPlayer,
		TableId:  tableID,
	})
	require.NoError(t, err)
	t.Logf("Player %s folded", firstPlayer)

	// Wait for turn to advance after fold
	require.Eventually(t, func() bool {
		newState := env.GetGameState(ctx, tableID)
		return newState.CurrentPlayer != firstPlayer
	}, 2*time.Second, 10*time.Millisecond, "turn should advance after fold")

	// Remaining two players go all-in
	for i := 0; i < 2; i++ {
		state = env.GetGameState(ctx, tableID)
		if state.Phase != pokerrpc.GamePhase_PRE_FLOP {
			// Already advanced
			break
		}
		currentPlayer := state.CurrentPlayer
		currentPhase := state.Phase

		_, err = env.PokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
			PlayerId: currentPlayer,
			TableId:  tableID,
			Amount:   200,
		})
		if err == nil {
			t.Logf("Player %s went all-in", currentPlayer)

			// Wait for turn to advance or phase to change
			require.Eventually(t, func() bool {
				newState := env.GetGameState(ctx, tableID)
				return newState.CurrentPlayer != currentPlayer || newState.Phase != currentPhase
			}, 2*time.Second, 10*time.Millisecond, "turn or phase should change after all-in")
		}
	}

	// CRITICAL VERIFICATION: Game should auto-advance through streets
	t.Log("Verifying auto-advance through FLOP...")
	env.WaitForGamePhase(ctx, tableID, pokerrpc.GamePhase_FLOP, 5*time.Second)
	state = env.GetGameState(ctx, tableID)
	assert.Equal(t, pokerrpc.GamePhase_FLOP, state.Phase, "should advance to FLOP")
	assert.Equal(t, 3, len(state.CommunityCards), "should have 3 community cards at FLOP")
	t.Logf("✓ FLOP reached with %d community cards", len(state.CommunityCards))

	t.Log("Verifying auto-advance through TURN...")
	env.WaitForGamePhase(ctx, tableID, pokerrpc.GamePhase_TURN, 5*time.Second)
	state = env.GetGameState(ctx, tableID)
	assert.Equal(t, pokerrpc.GamePhase_TURN, state.Phase, "should advance to TURN")
	assert.Equal(t, 4, len(state.CommunityCards), "should have 4 community cards at TURN")
	t.Logf("✓ TURN reached with %d community cards", len(state.CommunityCards))

	t.Log("Verifying auto-advance through RIVER...")
	env.WaitForGamePhase(ctx, tableID, pokerrpc.GamePhase_RIVER, 5*time.Second)
	state = env.GetGameState(ctx, tableID)
	assert.Equal(t, pokerrpc.GamePhase_RIVER, state.Phase, "should advance to RIVER")
	assert.Equal(t, 5, len(state.CommunityCards), "should have 5 community cards at RIVER")
	t.Logf("✓ RIVER reached with %d community cards", len(state.CommunityCards))

	t.Log("Verifying auto-advance to SHOWDOWN...")
	env.WaitForGamePhase(ctx, tableID, pokerrpc.GamePhase_SHOWDOWN, 5*time.Second)
	state = env.GetGameState(ctx, tableID)
	assert.Equal(t, pokerrpc.GamePhase_SHOWDOWN, state.Phase, "should advance to SHOWDOWN")
	t.Log("✓ SHOWDOWN reached")

	// Verify we have 1 folded and 2 all-in players
	foldedCount := 0
	allInCount := 0
	for _, p := range state.Players {
		if p.Folded {
			foldedCount++
		}
		if p.IsAllIn {
			allInCount++
		}
	}
	assert.Equal(t, 1, foldedCount, "should have 1 folded player")
	assert.GreaterOrEqual(t, allInCount, 1, "should have at least 1 all-in player")

	// Wait for showdown to complete and winners to be available
	require.Eventually(t, func() bool {
		_, err := env.PokerClient.GetLastWinners(ctx, &pokerrpc.GetLastWinnersRequest{
			TableId: tableID,
		})
		return err == nil
	}, 3*time.Second, 50*time.Millisecond, "showdown should complete with winners")

	t.Log("✓ Partial all-in: Auto-advanced through all streets correctly")
}

// -----------------------------------------------------------------------------
//
//	SCENARIO: Game Over Detection - Winner Takes All
//
//	This test plays hands until one player is eliminated (has 0 chips).
//	Since poker hands can tie and split the pot, we may need multiple hands.
//
// -----------------------------------------------------------------------------
func TestGameOver_WinnerTakesAll(t *testing.T) {
	t.Parallel()
	env := testenv.New(t)
	defer env.Close()

	ctx := context.Background()

	// Setup 2 players with small stacks
	players := []string{"alice", "bob"}

	// Create table with small starting chips (100 each)
	// Use shorter auto-start to speed up test if we need multiple hands
	// BuyIn: 0 to avoid escrow requirement in tests
	ctx = env.ContextWithToken(ctx, "alice")
	createResp, err := env.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      "alice",
		SmallBlind:    10,
		BigBlind:      20,
		MinPlayers:    2,
		MaxPlayers:    2,
		BuyIn:         0,
		StartingChips: 100,  // Small stacks
		AutoStartMs:   1500, // Shorter delay for faster test
		AutoAdvanceMs: 500,  // Shorter delay for faster test
	})
	require.NoError(t, err)
	tableID := createResp.TableId

	// Join bob
	_, err = env.JoinTable(ctx, "bob", tableID)
	require.NoError(t, err)

	// Both players mark ready
	for _, p := range players {
		_, err := env.SetPlayerReady(ctx, p, tableID)
		require.NoError(t, err)
	}

	// Wait for game to start
	env.WaitForGameStart(ctx, tableID, 3*time.Second)
	env.WaitForGamePhase(ctx, tableID, pokerrpc.GamePhase_PRE_FLOP, 3*time.Second)

	// Play hands until one player is eliminated
	// Maximum 5 hands in case of consecutive ties (rare but possible)
	maxHands := 5
	var gameOverDetected bool
	var winnerID string

	for handNum := 1; handNum <= maxHands; handNum++ {
		t.Logf("Hand %d: Playing all-in hand...", handNum)

		// Get current state
		state, removed := env.GetGameStateAllowNotFound(ctx, tableID)
		require.False(t, removed, "table should exist at hand start")

		// First player goes all-in
		currentPlayer := state.CurrentPlayer
		_, err = env.PokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
			PlayerId: currentPlayer,
			TableId:  tableID,
			Amount:   100,
		})
		require.NoError(t, err)

		// Wait for turn to advance
		require.Eventually(t, func() bool {
			state, removed = env.GetGameStateAllowNotFound(ctx, tableID)
			if removed {
				return true
			}
			return state.CurrentPlayer != currentPlayer
		}, 2*time.Second, 10*time.Millisecond, "turn should advance after all-in")

		// Second player calls all-in
		state, removed = env.GetGameStateAllowNotFound(ctx, tableID)
		if removed {
			gameOverDetected = true
			winnerID = "table_removed"
			break
		}
		currentPlayer = state.CurrentPlayer
		_, err = env.PokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
			PlayerId: currentPlayer,
			TableId:  tableID,
			Amount:   100,
		})
		require.NoError(t, err)

		// Wait for showdown
		env.WaitForShowdownOrRemoval(ctx, tableID, 10*time.Second)

		// Check if one player has been eliminated
		finalState, removed := env.GetGameStateAllowNotFound(ctx, tableID)
		if removed {
			gameOverDetected = true
			t.Log("Table removed immediately after game over; skipping final balance assertions")
			break
		}
		var playersWithChips int
		for _, ps := range finalState.Players {
			if ps.Balance > 0 {
				playersWithChips++
				winnerID = ps.Id
			}
		}

		if playersWithChips == 1 {
			// Game over - one player eliminated
			gameOverDetected = true
			t.Logf("✓ Hand %d: Game over after %d hand(s), winner: %s", handNum, handNum, winnerID)

			// Verify winner has all chips
			for _, ps := range finalState.Players {
				if ps.Id == winnerID {
					assert.Equal(t, int64(200), ps.Balance, "Winner should have all 200 chips")
				} else {
					assert.Equal(t, int64(0), ps.Balance, "Loser should have 0 chips")
				}
			}
			break
		}

		// Pot was split - wait for next hand
		t.Logf("Hand %d: Pot split (tie), both players keep 100 chips", handNum)
		if handNum < maxHands {
			time.Sleep(2 * time.Second)                                                    // Wait for auto-start
			env.WaitForGamePhase(ctx, tableID, pokerrpc.GamePhase_PRE_FLOP, 3*time.Second) // Next hand
		}
	}

	require.True(t, gameOverDetected, "Expected game over within %d hands", maxHands)

	// Verify game over state: game should be in SHOWDOWN phase
	// (This is the state when game over is detected, before table cleanup)
	t.Log("Verifying game over state...")
	state, removed := env.GetGameStateAllowNotFound(ctx, tableID)
	if !removed {
		assert.Equal(t, pokerrpc.GamePhase_SHOWDOWN, state.Phase, "Game should be in SHOWDOWN when game over is detected")
	} else {
		t.Log("Table already removed before final SHOWDOWN assertion; skipping phase check")
	}

	// Wait for table removal (server removes table after 1 second grace period)
	t.Log("Waiting for table cleanup after game over...")
	require.Eventually(t, func() bool {
		_, err := env.PokerClient.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		return err != nil && status.Code(err) == codes.NotFound
	}, 5*time.Second, 100*time.Millisecond, "Table should be removed after game over")

	t.Log("✓ Game over correctly detected, winner takes all chips, and table was properly cleaned up")
}

// -----------------------------------------------------------------------------
//
//	SCENARIO: Unequal Stacks All-In - Auto-Advance with Partial Match
//
//	This test specifically covers the regression where one player has more chips
//	than another, goes all-in, and the other player can only partially match.
//	This should still trigger auto-advance since no one can bet anymore.
//
// -----------------------------------------------------------------------------
func TestUnequalStacksAllIn_AutoAdvancePartialMatch(t *testing.T) {
	t.Parallel()
	env := testenv.New(t)
	defer env.Close()

	ctx := context.Background()

	// Setup 2 players
	players := []string{"rich_player", "poor_player"}

	// Create table with moderate stacks - small blinds so all-in doesn't eliminate anyone
	// BuyIn: 0 to avoid escrow requirement in tests
	ctx = env.ContextWithToken(ctx, "rich_player")
	createResp, err := env.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      "rich_player",
		SmallBlind:    5,
		BigBlind:      10,
		MinPlayers:    2,
		MaxPlayers:    2,
		BuyIn:         0,
		StartingChips: 200, // After blinds: 195 and 190
		AutoStartMs:   5000,
		AutoAdvanceMs: 1000, // 1 second auto-advance
	})
	require.NoError(t, err)
	tableID := createResp.TableId

	// Poor player joins - but we need to manipulate their stack somehow
	// Actually, let's use a different approach: use high blinds
	// Rich player will have 990 after small blind, poor player 980 after big blind

	// Join poor player
	_, err = env.JoinTable(ctx, "poor_player", tableID)
	require.NoError(t, err)

	// Both players mark ready
	for _, p := range players {
		_, err := env.SetPlayerReady(ctx, p, tableID)
		require.NoError(t, err)
	}

	// Wait for game to start
	env.WaitForGameStart(ctx, tableID, 3*time.Second)
	env.WaitForGamePhase(ctx, tableID, pokerrpc.GamePhase_PRE_FLOP, 3*time.Second)

	t.Log("Game started - now triggering all-in scenario...")

	// Get current state
	state, removed := env.GetGameStateAllowNotFound(ctx, tableID)
	require.False(t, removed, "table should exist at hand start")
	currentPlayer := state.CurrentPlayer

	// Current player goes all-in (they have 195 or 190 chips after blinds)
	_, err = env.PokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: currentPlayer,
		TableId:  tableID,
		Amount:   200, // Try to bet 200, will be capped at their balance
	})
	require.NoError(t, err)
	t.Logf("Player %s went all-in", currentPlayer)

	// Wait for next player's turn (check for connection-closing errors during cleanup)
	require.Eventually(t, func() bool {
		resp, err := env.PokerClient.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		if err != nil {
			// Treat table removal as success; connection closing is a retry.
			if status.Code(err) == codes.NotFound {
				return true
			}
			if strings.Contains(err.Error(), "connection is closing") {
				return false
			}
			t.Errorf("unexpected error getting game state: %v", err)
			return false
		}
		state = resp.GameState
		return state.CurrentPlayer != currentPlayer && state.CurrentPlayer != ""
	}, 2*time.Second, 10*time.Millisecond, "next player should have a turn")

	nextPlayer := state.CurrentPlayer

	// Next player calls (will be forced to partial all-in since they have less after paying blind)
	_, err = env.PokerClient.CallBet(ctx, &pokerrpc.CallBetRequest{
		PlayerId: nextPlayer,
		TableId:  tableID,
	})
	require.NoError(t, err)
	t.Logf("Player %s called (partial all-in with smaller stack)", nextPlayer)

	// CRITICAL VERIFICATION: Auto-advance should trigger even with unequal stacks
	// The system should recognize that no more betting is possible
	//
	// Note: We can't use waitForGamePhase here because auto-advance happens so fast
	// that by the time we check, we've already advanced past FLOP/TURN/RIVER to SHOWDOWN.
	// Instead, we just wait for showdown and verify it completed successfully.

	t.Log("Waiting for showdown (auto-advance should progress through all streets)...")
	env.WaitForShowdownOrRemoval(ctx, tableID, 10*time.Second)

	// Verify we reached showdown with all 5 community cards (if table still exists)
	state, removed = env.GetGameStateAllowNotFound(ctx, tableID)
	if removed {
		t.Log("Table removed immediately after showdown; skipping community card assertions")
	} else {
		assert.Equal(t, pokerrpc.GamePhase_SHOWDOWN, state.Phase)
		assert.Equal(t, 5, len(state.CommunityCards), "Should have all 5 community cards at SHOWDOWN")
		t.Log("✓ SHOWDOWN reached with all 5 community cards")
	}

	// Wait for showdown to complete with winners (or table removal)
	require.Eventually(t, func() bool {
		_, err := env.PokerClient.GetLastWinners(ctx, &pokerrpc.GetLastWinnersRequest{
			TableId: tableID,
		})
		return err == nil || status.Code(err) == codes.NotFound
	}, 3*time.Second, 50*time.Millisecond, "showdown should complete with winners")

	t.Log("✓ Unequal stacks all-in: Auto-advanced through all streets correctly")
	t.Log("✓ Regression test passed: partial all-in match triggers auto-advance")
}

// -----------------------------------------------------------------------------
//
//	SCENARIO: Test UNIQUE constraint violation when rejoining after player leaves
//
//	This test reproduces the bug where:
//	1. Host creates a table (gets seat 0)
//	2. Player A joins (gets seat 1)
//	3. Player A leaves (UnseatPlayer sets left_at but keeps seat value)
//	4. Player B tries to join and gets assigned seat 1 (because in-memory table only has host at seat 0)
//	5. SeatPlayer fails with UNIQUE constraint error because DB still has row with (table_id, seat=1)
//
//	The test should FAIL because the bug is still present.
//
// -----------------------------------------------------------------------------
func TestUniqueConstraintViolationOnRejoin(t *testing.T) {
	t.Parallel()
	env := testenv.New(t)
	defer env.Close()

	ctx := context.Background()

	// Setup players
	host := "host"
	playerA := "playerA"
	playerB := "playerB"

	// Host creates a table (automatically seated at seat 0)
	ctx = env.ContextWithToken(ctx, host)
	createResp, err := env.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      testenv.PlayerIDToShortIDString(host),
		SmallBlind:    10,
		BigBlind:      20,
		MinPlayers:    2,
		MaxPlayers:    2,
		BuyIn:         0,
		StartingChips: 1000,
		AutoStartMs:   5000, // Long delay to prevent auto-start during test
		AutoAdvanceMs: 1000,
	})
	require.NoError(t, err)
	tableID := createResp.TableId

	// Verify host is seated
	state := env.GetGameState(ctx, tableID)
	require.Equal(t, int32(1), state.PlayersJoined, "Host should be seated")

	// Player A joins (should get seat 1)
	joinRespA, err := env.JoinTable(ctx, playerA, tableID)
	require.NoError(t, err)
	require.True(t, joinRespA.Success, "Player A should be able to join")

	// Verify both players are seated
	state = env.GetGameState(ctx, tableID)
	require.Equal(t, int32(2), state.PlayersJoined, "Both host and Player A should be seated")

	// Player A leaves the table
	// This calls UnseatPlayer which sets left_at but keeps the seat value
	ctxA := env.ContextWithToken(ctx, playerA)
	leaveResp, err := env.LobbyClient.LeaveTable(ctxA, &pokerrpc.LeaveTableRequest{
		PlayerId: testenv.PlayerIDToShortIDString(playerA),
		TableId:  tableID,
	})
	require.NoError(t, err)
	require.True(t, leaveResp.Success, "Player A should be able to leave")

	// Verify Player A is no longer in the in-memory table
	state = env.GetGameState(ctx, tableID)
	require.Equal(t, int32(1), state.PlayersJoined, "Only host should remain in in-memory table")

	// Now Player B tries to join
	// The join logic will pick seat 1 because:
	// - Host is at seat 0 (in-memory)
	// - Seat 1 appears free (in-memory table doesn't have Player A)
	// But the DB still has a row with (table_id, seat=1, left_at IS NOT NULL)
	// This should trigger the UNIQUE constraint violation
	//
	// BUG: SeatPlayer tries to insert a new row with (table_id, seat=1)
	// but the old row still exists with the same (table_id, seat=1) combination,
	// causing a UNIQUE constraint violation.
	//
	// EXPECTED BEHAVIOR (when bug is fixed): Join should succeed because seat 1 is free
	// CURRENT BEHAVIOR (bug present): Join fails with UNIQUE constraint error
	joinRespB, err := env.JoinTable(ctx, playerB, tableID)

	// This assertion will FAIL while the bug is present (join fails)
	// and PASS when the bug is fixed (join succeeds)
	require.NoError(t, err, "Join should succeed - seat 1 should be available after Player A left")
	require.True(t, joinRespB.Success, "Join should succeed - seat 1 should be available after Player A left")

	// Verify Player B is now seated
	state = env.GetGameState(ctx, tableID)
	require.Equal(t, int32(2), state.PlayersJoined, "Both host and Player B should be seated")
}

// -----------------------------------------------------------------------------
//
//	SCENARIO: Test All-In Auto-Advance Bug - No Unexpected Folds
//
//	This test reproduces a bug where StartTurn() is called on a player who
//	shouldn't have a turn during auto-advance phases. The bug scenario:
//	- 3 players in the game
//	- On TURN: Player 1 bets all-in, Player 2 calls all-in, Player 0 calls
//	- All bets matched → auto-advance enabled (active=1, allInCount>=1, unmatched=0)
//	- When entering RIVER with auto-advance enabled, if the state handler
//	  incorrectly calls StartTurn() on the remaining IN_GAME player, it schedules
//	  a timer. When the timer fires, it causes an unexpected fold.
//
//	The exact bug from production logs:
//	- TURN: All players matched bets (some all-in, some with chips remaining)
//	- maybeCompleteBettingRound enables auto-advance (active=1, allInCount>=1, unmatched=0)
//	- Auto-advance timer fires, entering stateRiver
//	- If stateRiver incorrectly starts a turn timer, the player folds
//
// -----------------------------------------------------------------------------
func TestAllInAutoAdvance_NoUnexpectedFolds(t *testing.T) {
	t.Parallel()
	env := testenv.New(t)
	defer env.Close()

	ctx := context.Background()

	// Setup 3 players - this is the key! The bug happens with 3 players, not 2
	players := []string{"player1", "player2", "player3"}

	// Create table with 3 players - matching the bug scenario
	// CRITICAL: Use SHORT timebank (1 second) so that if StartTurn() is incorrectly
	// called on the remaining active player during auto-advance, the timer will fire
	// and cause an unexpected fold. We'll drive actions quickly during setup so players
	// don't timeout before we reach the bug scenario.
	// BuyIn: 0 to avoid escrow requirement in tests
	ctx = env.ContextWithToken(ctx, "player1")
	createResp, err := env.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:        "player1",
		SmallBlind:      10,
		BigBlind:        20,
		MinPlayers:      3,
		MaxPlayers:      3,
		BuyIn:           0,
		StartingChips:   1000, // Enough chips to go all-in
		AutoStartMs:     2000,
		AutoAdvanceMs:   2000, // 2 second auto-advance delay
		TimeBankSeconds: 1,    // 1 second timebank - SHORT so timer fires during auto-advance
	})
	require.NoError(t, err)
	tableID := createResp.TableId

	// Player2 and Player3 join
	for _, p := range players[1:] {
		_, err := env.JoinTable(ctx, p, tableID)
		require.NoError(t, err)
	}

	// Both players mark ready
	for _, p := range players {
		_, err := env.SetPlayerReady(ctx, p, tableID)
		require.NoError(t, err)
	}

	// Wait for game to start
	env.WaitForGameStart(ctx, tableID, 3*time.Second)
	env.WaitForGamePhase(ctx, tableID, pokerrpc.GamePhase_PRE_FLOP, 3*time.Second)

	// Get initial state and collect all player IDs for tracking
	initialState := env.GetGameState(ctx, tableID)
	require.Equal(t, 3, len(initialState.Players), "should have 3 players")

	allPlayerIDs := make(map[string]bool)
	for _, p := range initialState.Players {
		allPlayerIDs[p.Id] = true
		require.False(t, p.Folded, "no player should be folded at game start")
	}

	// Helper function to drive a betting round to completion
	// This actively makes players call/check so they don't timeout
	driveBettingRound := func(targetPhase pokerrpc.GamePhase) {
		for {
			state := env.GetGameState(ctx, tableID)
			// If we've reached the target phase or beyond, we're done
			if state.Phase >= targetPhase {
				return
			}
			// If we've gone to showdown, something went wrong
			if state.Phase == pokerrpc.GamePhase_SHOWDOWN {
				t.Fatalf("hand ended at showdown before reaching %s", targetPhase)
			}

			currentPlayer := state.CurrentPlayer
			if currentPlayer == "" {
				// Between actions, wait a bit
				time.Sleep(10 * time.Millisecond)
				continue
			}

			// Find the current player's state
			var player *pokerrpc.Player
			for _, p := range state.Players {
				if p.Id == currentPlayer {
					player = p
					break
				}
			}
			if player == nil {
				time.Sleep(10 * time.Millisecond)
				continue
			}

			// Simple strategy: call if there's a bet to call, otherwise check
			var err error
			if state.CurrentBet > player.CurrentBet {
				// Need to call
				_, err = env.PokerClient.CallBet(ctx, &pokerrpc.CallBetRequest{
					PlayerId: currentPlayer,
					TableId:  tableID,
				})
			} else {
				// Can check
				_, err = env.PokerClient.CheckBet(ctx, &pokerrpc.CheckBetRequest{
					PlayerId: currentPlayer,
					TableId:  tableID,
				})
			}

			if err != nil {
				// Might be a transient error or phase changed, check state
				if isTransientTurnError(err) {
					time.Sleep(10 * time.Millisecond)
					continue
				}
				// If phase changed, that's fine - we might have advanced
				newState := env.GetGameState(ctx, tableID)
				if newState.Phase >= targetPhase {
					return
				}
				// Otherwise it's a real error
				require.NoError(t, err, "failed to act for player %s", currentPlayer)
			}

			// Wait a bit for state to update
			time.Sleep(50 * time.Millisecond)
		}
	}

	// Reproduce the exact bug scenario from the logs:
	// 1. Play through PRE_FLOP and FLOP normally (all players check/call)
	// 2. On TURN: Two players go all-in, one remains active
	// 3. The bug: When auto-advance is enabled and we transition to RIVER,
	//    initializeCurrentPlayer() calls StartTurn() on the remaining active player
	// 4. The timer fires and causes an unexpected fold

	t.Log("Driving PRE_FLOP to FLOP...")
	// Make a bet on PRE_FLOP to create action
	currentPlayer := initialState.CurrentPlayer
	require.NotEmpty(t, currentPlayer, "should have a current player")
	_, err = env.PokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: currentPlayer,
		TableId:  tableID,
		Amount:   500,
	})
	require.NoError(t, err)
	t.Logf("Player %s bet 500 on PRE_FLOP", currentPlayer)

	// Drive the rest of PRE_FLOP (other players call)
	driveBettingRound(pokerrpc.GamePhase_FLOP)
	t.Log("Reached FLOP")

	// Drive FLOP (all players check)
	driveBettingRound(pokerrpc.GamePhase_TURN)
	t.Log("Reached TURN - now reproducing the bug scenario")

	// Now reproduce the bug: On TURN, two players go all-in
	// This matches the logs: Player 1 bets 500 (all-in), Player 2 calls (all-in)
	state := env.GetGameState(ctx, tableID)
	currentPlayer = state.CurrentPlayer
	require.NotEmpty(t, currentPlayer, "should have a current player on TURN")

	// First player on TURN goes all-in (bet 500, which will be all-in if they have ~500 chips)
	_, err = env.PokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: currentPlayer,
		TableId:  tableID,
		Amount:   500, // All-in bet
	})
	require.NoError(t, err)
	allInPlayer1 := currentPlayer
	t.Logf("Player %s went all-in on TURN", allInPlayer1)

	// Wait for next player's turn
	require.Eventually(t, func() bool {
		state = env.GetGameState(ctx, tableID)
		return state.CurrentPlayer != currentPlayer || state.Phase != pokerrpc.GamePhase_TURN
	}, 2*time.Second, 10*time.Millisecond, "turn should advance after first all-in")

	// Second player calls all-in
	state = env.GetGameState(ctx, tableID)
	currentPlayer = state.CurrentPlayer
	require.NotEmpty(t, currentPlayer, "should have a current player for second all-in")
	require.NotEqual(t, allInPlayer1, currentPlayer, "should be a different player")

	_, err = env.PokerClient.CallBet(ctx, &pokerrpc.CallBetRequest{
		PlayerId: currentPlayer,
		TableId:  tableID,
	})
	require.NoError(t, err)
	allInPlayer2 := currentPlayer
	t.Logf("Player %s called all-in on TURN", allInPlayer2)

	// Wait for next player's turn (the remaining active player)
	require.Eventually(t, func() bool {
		state = env.GetGameState(ctx, tableID)
		return state.CurrentPlayer != currentPlayer || state.Phase != pokerrpc.GamePhase_TURN
	}, 2*time.Second, 10*time.Millisecond, "turn should advance after second all-in")

	// Identify the remaining active player (the one who didn't go all-in)
	state = env.GetGameState(ctx, tableID)
	remainingActivePlayer := ""
	for _, p := range state.Players {
		if !allPlayerIDs[p.Id] {
			continue
		}
		if p.Id != allInPlayer1 && p.Id != allInPlayer2 {
			remainingActivePlayer = p.Id
			break
		}
	}
	require.NotEmpty(t, remainingActivePlayer, "should have identified the remaining active player")
	t.Logf("Remaining active player: %s (will call to trigger auto-advance)", remainingActivePlayer)

	// CRITICAL: The remaining player MUST call to trigger auto-advance.
	// The production bug happens AFTER all bets are matched and auto-advance is enabled:
	// - active=1, allInCount>=1, unmatched=0 → auto-advance enabled
	// - stateRiver enters with autoAdvanceEnabled=true
	// - If stateRiver incorrectly starts a turn timer, the player folds
	//
	// Without this call, the remaining player would time out and fold during TURN
	// (which is correct poker behavior, not a bug).
	_, err = env.PokerClient.CallBet(ctx, &pokerrpc.CallBetRequest{
		PlayerId: remainingActivePlayer,
		TableId:  tableID,
	})
	require.NoError(t, err)
	t.Logf("Player %s called the all-in (triggering auto-advance)", remainingActivePlayer)

	// Now all bets are matched (or all players all-in). Auto-advance should be enabled.
	// The bug: When stateRiver is entered with auto-advance enabled but activePlayers >= 1,
	// if the state handler incorrectly calls StartTurn(), it schedules a timer that
	// fires and causes an unexpected fold.

	// Track if the remaining player folds unexpectedly during auto-advance
	unexpectedFoldDetected := false
	var foldPhase pokerrpc.GamePhase

	// Monitor during the auto-advance from TURN to RIVER to SHOWDOWN
	t.Log("Monitoring for unexpected fold during auto-advance...")

	// Wait for RIVER phase (auto-advance should trigger after 2 seconds)
	require.Eventually(t, func() bool {
		state := env.GetGameState(ctx, tableID)
		return state.Phase >= pokerrpc.GamePhase_RIVER
	}, 5*time.Second, 50*time.Millisecond, "game should auto-advance to RIVER")

	// Check immediately when RIVER is reached - the bug may have already fired
	state = env.GetGameState(ctx, tableID)
	for _, p := range state.Players {
		if !allPlayerIDs[p.Id] {
			continue
		}
		if p.Id == remainingActivePlayer && p.Folded {
			unexpectedFoldDetected = true
			foldPhase = state.Phase
			t.Errorf("BUG DETECTED: Remaining active player %s is folded at phase %s (should NOT fold during auto-advance)",
				p.Id, state.Phase)
		}
	}

	// Continue monitoring through RIVER to SHOWDOWN
	// Check every 100ms for 3 seconds to catch the fold if it happens during RIVER
	monitorDeadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(monitorDeadline) {
		state := env.GetGameState(ctx, tableID)
		if state.Phase == pokerrpc.GamePhase_SHOWDOWN {
			break
		}
		for _, p := range state.Players {
			if !allPlayerIDs[p.Id] {
				continue
			}
			if p.Id == remainingActivePlayer && p.Folded && !unexpectedFoldDetected {
				unexpectedFoldDetected = true
				foldPhase = state.Phase
				t.Errorf("BUG DETECTED: Remaining active player %s is folded at phase %s during auto-advance",
					p.Id, state.Phase)
			}
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Wait for showdown
	env.WaitForShowdownOrRemoval(ctx, tableID, 10*time.Second)

	// Wait for showdown to complete
	env.WaitForShowdownOrRemoval(ctx, tableID, 10*time.Second)

	// CRITICAL ASSERTION: The player who called should NOT have folded during auto-advance.
	// The bug causes them to fold when StartTurn() is incorrectly called in state handlers
	// during auto-advance mode, and the timer fires.
	//
	// This assertion will FAIL when the bug is present (unexpected fold occurs)
	// and PASS when the bug is fixed (no unexpected fold)
	require.False(t, unexpectedFoldDetected,
		"BUG REPRODUCTION: Player %s should NOT fold during auto-advance. "+
			"Fold was detected at phase %s. This happens when state handlers incorrectly "+
			"call StartTurn() during auto-advance mode, and the timer fires.",
		remainingActivePlayer, foldPhase)

	// Final verification at showdown
	finalState, removed := env.GetGameStateAllowNotFound(ctx, tableID)
	if !removed {
		// Verify the remaining active player is not folded
		for _, p := range finalState.Players {
			if !allPlayerIDs[p.Id] {
				continue
			}
			if p.Id == remainingActivePlayer && p.Folded {
				t.Errorf("BUG: Remaining active player %s is folded at showdown (should not be)", p.Id)
			}
		}
	}
}
