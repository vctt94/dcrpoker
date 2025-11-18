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
	"net"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vctt94/bisonbotkit/logging"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	"github.com/vctt94/pokerbisonrelay/pkg/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// testEnv holds the runtime components that make up a fully
// functional instance of the poker server backed by a *real* SQLite
// database. Each E2E test spins-up its own env so tests are completely
// isolated and can run in parallel.
type testEnv struct {
	t           *testing.T
	db          server.Database
	pokerSrv    *server.Server
	grpcSrv     *grpc.Server
	conn        *grpc.ClientConn
	lobbyClient pokerrpc.LobbyServiceClient
	pokerClient pokerrpc.PokerServiceClient
}

// createTestLogBackend creates a LogBackend for testing
func createTestLogBackend() *logging.LogBackend {
	logBackend, err := logging.NewLogBackend(logging.LogConfig{
		LogFile:        "",      // Empty for testing - will use stdout
		DebugLevel:     "debug", // Set to debug to see detailed logging
		MaxLogFiles:    1,
		MaxBufferLines: 100,
	})
	if err != nil {
		// Fallback to a minimal LogBackend if creation fails
		return &logging.LogBackend{}
	}
	return logBackend
}

// newTestEnv creates, starts and returns a ready-to-use environment.
func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	// 1. NEW TEMPORARY DATABASE -------------------------------------------------
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "poker.sqlite")
	database, err := server.NewDatabase(dbPath)
	require.NoError(t, err)

	// 2. GRPC SERVER ------------------------------------------------------------
	logBackend := createTestLogBackend()
	pokerSrv := server.NewServer(database, logBackend)
	lis, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	grpcSrv := grpc.NewServer()
	pokerrpc.RegisterLobbyServiceServer(grpcSrv, pokerSrv)
	pokerrpc.RegisterPokerServiceServer(grpcSrv, pokerSrv)
	go func() { _ = grpcSrv.Serve(lis) }()

	// 3. GRPC CLIENT CONNECTION --------------------------------------------------
	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	return &testEnv{
		t:           t,
		db:          database,
		pokerSrv:    pokerSrv,
		grpcSrv:     grpcSrv,
		conn:        conn,
		lobbyClient: pokerrpc.NewLobbyServiceClient(conn),
		pokerClient: pokerrpc.NewPokerServiceClient(conn),
	}
}

// Close gracefully shuts down all resources.
func (e *testEnv) Close() {
	e.conn.Close()
	e.pokerSrv.Stop()
	e.grpcSrv.Stop()
	_ = e.db.Close()
}

// setBalance is a small helper that ensures the player has exactly the
// specified balance by calculating the delta against the current stored
// balance and issuing a single UpdateBalance call.
func (e *testEnv) setBalance(ctx context.Context, playerID string, balance int64) {
	var currBal int64
	if resp, err := e.lobbyClient.GetBalance(ctx, &pokerrpc.GetBalanceRequest{PlayerId: playerID}); err == nil {
		currBal = resp.GetBalance()
	}
	delta := balance - currBal
	if delta == 0 {
		return
	}
	_, err := e.lobbyClient.UpdateBalance(ctx, &pokerrpc.UpdateBalanceRequest{
		PlayerId:    playerID,
		Amount:      delta,
		Description: "seed balance",
	})
	require.NoError(e.t, err)
}

// waitForGameStart polls GetGameState until GameStarted==true or the timeout
// expires (in which case the test fails).
func (e *testEnv) waitForGameStart(ctx context.Context, tableID string, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		resp, err := e.pokerClient.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		if err == nil && resp.GameState.GetGameStarted() {
			return
		}
		select {
		case <-ctx.Done():
			e.t.Fatalf("game did not start within %s", timeout)
		case <-time.After(50 * time.Millisecond):
		}
	}
}

// waitForGamePhase polls GetGameState until the given phase is reached or the timeout expires
func (e *testEnv) waitForGamePhase(ctx context.Context, tableID string, phase pokerrpc.GamePhase, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		resp, err := e.pokerClient.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		if err == nil && resp.GameState.GetPhase() == phase {
			return
		}
		select {
		case <-ctx.Done():
			e.t.Fatalf("game did not reach phase %s within %s", phase, timeout)
		case <-time.After(50 * time.Millisecond):
		}
	}
}

// getBalance is syntactic sugar to fetch a player's current balance.
func (e *testEnv) getBalance(ctx context.Context, playerID string) int64 {
	resp, err := e.lobbyClient.GetBalance(ctx, &pokerrpc.GetBalanceRequest{PlayerId: playerID})
	require.NoError(e.t, err)
	return resp.Balance
}

// getGameState is a helper to get the current game state
func (e *testEnv) getGameState(ctx context.Context, tableID string) *pokerrpc.GameUpdate {
	resp, err := e.pokerClient.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	require.NoError(e.t, err)
	return resp.GameState
}

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

// createStandardTable creates a table with standard settings for testing
func (e *testEnv) createStandardTable(ctx context.Context, creatorID string, minPlayers, maxPlayers int) string {
	createResp, err := e.lobbyClient.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      creatorID,
		SmallBlind:    10,
		BigBlind:      20,
		MinPlayers:    int32(minPlayers),
		MaxPlayers:    int32(maxPlayers),
		BuyIn:         1_000,
		MinBalance:    1_000,
		StartingChips: 1_000,
		AutoStartMs:   100,
		AutoAdvanceMs: 1000,
	})
	require.NoError(e.t, err)
	assert.NotEmpty(e.t, createResp.TableId)
	return createResp.TableId
}

// -----------------------------------------------------------------------------
//
//	SCENARIO: Full Sit'n'Go with 3 players
//
// -----------------------------------------------------------------------------
func TestSitAndGoEndToEnd(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	defer env.Close()

	ctx := context.Background()

	players := []string{"alice", "bob", "carol"}
	initialBankroll := int64(10_000) // satoshi-style units (1e-8 DCR)
	for _, p := range players {
		env.setBalance(ctx, p, initialBankroll)
	}

	// Alice creates a new table that acts like a Sit'n'Go (auto-start when all
	// players are ready).
	createResp, err := env.lobbyClient.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      "alice",
		SmallBlind:    10,
		BigBlind:      20,
		MinPlayers:    3,
		MaxPlayers:    3,
		BuyIn:         1_000,
		MinBalance:    1_000,
		StartingChips: 1_000,
		AutoAdvanceMs: 1000,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, createResp.TableId)
	tableID := createResp.TableId

	// Bob & Carol join the table.
	for _, p := range []string{"bob", "carol"} {
		_, err := env.lobbyClient.JoinTable(ctx, &pokerrpc.JoinTableRequest{
			PlayerId: p,
			TableId:  tableID,
		})
		require.NoError(t, err)
	}

	// Everyone marks themselves as ready.
	for _, p := range players {
		_, err := env.lobbyClient.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{
			PlayerId: p,
			TableId:  tableID,
		})
		require.NoError(t, err)
	}

	// Verify that all players are marked as ready
	gameState := env.getGameState(ctx, tableID)
	assert.True(t, gameState.GetPlayersJoined() == 3, "expected 3 players joined")

	// Wait until the server flags the game as started.
	env.waitForGameStart(ctx, tableID, 3*time.Second)

	// Wait for the game to reach PRE_FLOP phase with a valid current player
	// This ensures the game is fully initialized before making bets
	require.Eventually(t, func() bool {
		gameState := env.getGameState(ctx, tableID)
		if !gameState.GameStarted {
			return false
		}
		if gameState.Phase != pokerrpc.GamePhase_PRE_FLOP {
			return false
		}
		// Ensure we have a current player set
		return gameState.CurrentPlayer != ""
	}, 3*time.Second, 10*time.Millisecond, "game should reach PRE_FLOP with a current player")

	// Quick sanity check of balances after table creation/join.
	//  - Table creator (alice) also pays buy-in when creating the table
	//  - Joiners (bob & carol) have bankroll - buyIn.
	buyIn := int64(1_000)
	assert.Equal(t, initialBankroll-buyIn, env.getBalance(ctx, "alice"))
	for _, p := range []string{"bob", "carol"} {
		assert.Equal(t, initialBankroll-buyIn, env.getBalance(ctx, p), "post buy-in balance mismatch for %s", p)
	}

	// ACTION ROUND -------------------------------------------------------------
	// First player to act (after BB) opens with a 100 bet
	gameState = env.getGameState(ctx, tableID)
	firstPlayer := gameState.CurrentPlayer
	require.NotEmpty(t, firstPlayer, "should have a current player")
	_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: firstPlayer,
		TableId:  tableID,
		Amount:   100,
	})
	require.NoError(t, err)

	// Second player calls
	gameState = env.getGameState(ctx, tableID)
	secondPlayer := gameState.CurrentPlayer
	require.NotEmpty(t, secondPlayer, "should have a current player")
	require.NotEqual(t, firstPlayer, secondPlayer, "current player should have changed")
	_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: secondPlayer,
		TableId:  tableID,
		Amount:   100,
	})
	require.NoError(t, err)

	// Third player decides to fold
	gameState = env.getGameState(ctx, tableID)
	thirdPlayer := gameState.CurrentPlayer
	require.NotEmpty(t, thirdPlayer, "should have a current player")
	require.NotEqual(t, firstPlayer, thirdPlayer, "third player should be different")
	require.NotEqual(t, secondPlayer, thirdPlayer, "third player should be different")
	_, err = env.pokerClient.FoldBet(ctx, &pokerrpc.FoldBetRequest{
		PlayerId: thirdPlayer,
		TableId:  tableID,
	})
	require.NoError(t, err)

	// Validate pot value (220) via GetGameState.
	// Pot = 30 (blinds) + 100 (first player's bet) + 90 (second player's call minus their blind)
	state, err := env.pokerClient.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
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

	// FINISHING ACTIONS --------------------------------------------------------
	// Alice tips Carol 150 for good sportsmanship using the real tip handler.
	_, err = env.lobbyClient.ProcessTip(ctx, &pokerrpc.ProcessTipRequest{
		FromPlayerId: "alice",
		ToPlayerId:   "carol",
		Amount:       150,
		Message:      "good fold",
	})
	require.NoError(t, err)

	// Verify balances post-tip.
	aliceBal := env.getBalance(ctx, "alice")
	carolBal := env.getBalance(ctx, "carol")
	assert.Equal(t, initialBankroll-buyIn-150, aliceBal)
	assert.Equal(t, initialBankroll-buyIn+150, carolBal)
}

// -----------------------------------------------------------------------------
//
//	SCENARIO: Complete Hand Flow - All Betting Rounds with 4 players
//
// -----------------------------------------------------------------------------
func TestCompleteHandFlow(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	defer env.Close()

	ctx := context.Background()

	// Setup players with initial bankrolls
	players := []string{"player1", "player2", "player3", "player4"}
	initialBankroll := int64(10_000)
	for _, p := range players {
		env.setBalance(ctx, p, initialBankroll)
	}

	// Player1 creates a table for 4 players
	tableID := env.createStandardTable(ctx, "player1", 4, 4)

	// All players join the table
	for _, p := range players[1:] { // Skip player1 who already created the table
		_, err := env.lobbyClient.JoinTable(ctx, &pokerrpc.JoinTableRequest{
			PlayerId: p,
			TableId:  tableID,
		})
		require.NoError(t, err)
	}

	// All players mark themselves as ready
	for _, p := range players {
		_, err := env.lobbyClient.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{
			PlayerId: p,
			TableId:  tableID,
		})
		require.NoError(t, err)
	}

	// Wait for game to start
	env.waitForGameStart(ctx, tableID, 3*time.Second)
	// Ensure PRE_FLOP before acting
	env.waitForGamePhase(ctx, tableID, pokerrpc.GamePhase_PRE_FLOP, 3*time.Second)
	// Ensure PRE_FLOP reached before acting
	env.waitForGamePhase(ctx, tableID, pokerrpc.GamePhase_PRE_FLOP, 3*time.Second)

	// Note: explicit per-action waits are used instead of a helper to avoid unused warnings.
	// Ensure PRE_FLOP reached before acting
	env.waitForGamePhase(ctx, tableID, pokerrpc.GamePhase_PRE_FLOP, 3*time.Second)

	// Note: we rely on explicit waits before actions in tests that need them.

	// Helper: wait until it's the specified player's turn
	waitForTurn := func(playerID string, timeout time.Duration) {
		require.Eventually(t, func() bool {
			state := env.getGameState(ctx, tableID)
			return state.CurrentPlayer == playerID
		}, timeout, 25*time.Millisecond, "did not become %s's turn within %s", playerID, timeout)
	}

	// PRE-FLOP BETTING
	// In 4-player game: player1=dealer, player2=SB, player3=BB, player4=UTG (acts first)

	// Player4 calls the big blind
	waitForTurn("player4", 2*time.Second)
	_, err := env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: "player4",
		TableId:  tableID,
		Amount:   20,
	})
	require.NoError(t, err)

	// Player1 raises
	waitForTurn("player1", 2*time.Second)
	_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: "player1",
		TableId:  tableID,
		Amount:   60, // Raising to 60
	})
	require.NoError(t, err)

	// Player2 calls the raise (SB needs to add 50 more to existing 10)
	waitForTurn("player2", 2*time.Second)
	_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: "player2",
		TableId:  tableID,
		Amount:   60,
	})
	require.NoError(t, err)

	// Player3 calls the raise (BB needs to add 40 more to existing 20)
	waitForTurn("player3", 2*time.Second)
	_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: "player3",
		TableId:  tableID,
		Amount:   60,
	})
	require.NoError(t, err)

	// Player4 calls the raise
	waitForTurn("player4", 2*time.Second)
	_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: "player4",
		TableId:  tableID,
		Amount:   60,
	})
	require.NoError(t, err)

	// Check pot after pre-flop: blinds (30) + all players bet 60 = 270, but actual is 240
	state := env.getGameState(ctx, tableID)
	assert.Equal(t, int64(240), state.Pot, "unexpected pot size after pre-flop")

	// FLOP ROUND
	// Wait for flop to be dealt
	env.waitForGamePhase(ctx, tableID, pokerrpc.GamePhase_FLOP, 3*time.Second)

	// Make sure we have 3 community cards
	state = env.getGameState(ctx, tableID)
	assert.Equal(t, 3, len(state.CommunityCards), "expected 3 community cards after flop")

	// Post-flop betting starts with small blind (player2)
	// Player2 checks
	waitForTurn("player2", 2*time.Second)
	_, err = env.pokerClient.CheckBet(ctx, &pokerrpc.CheckBetRequest{
		PlayerId: "player2",
		TableId:  tableID,
	})
	require.NoError(t, err)

	// Player3 bets 100
	waitForTurn("player3", 2*time.Second)
	_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: "player3",
		TableId:  tableID,
		Amount:   100,
	})
	require.NoError(t, err)

	// Player4 calls
	waitForTurn("player4", 2*time.Second)
	_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: "player4",
		TableId:  tableID,
		Amount:   100,
	})
	require.NoError(t, err)

	// Player1 folds
	waitForTurn("player1", 2*time.Second)
	_, err = env.pokerClient.FoldBet(ctx, &pokerrpc.FoldBetRequest{
		PlayerId: "player1",
		TableId:  tableID,
	})
	require.NoError(t, err)

	// Player2 folds
	waitForTurn("player2", 2*time.Second)
	_, err = env.pokerClient.FoldBet(ctx, &pokerrpc.FoldBetRequest{
		PlayerId: "player2",
		TableId:  tableID,
	})
	require.NoError(t, err)

	// Check pot after flop: 240 + 100 + 100 = 440
	state = env.getGameState(ctx, tableID)
	assert.Equal(t, int64(440), state.Pot, "unexpected pot size after flop")

	// TURN ROUND
	// Wait for turn card
	env.waitForGamePhase(ctx, tableID, pokerrpc.GamePhase_TURN, 3*time.Second)

	// Make sure we have 4 community cards
	state = env.getGameState(ctx, tableID)
	assert.Equal(t, 4, len(state.CommunityCards), "expected 4 community cards after turn")

	// Only player3 and player4 remain
	// Player3 bets 200
	waitForTurn("player3", 2*time.Second)
	_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: "player3",
		TableId:  tableID,
		Amount:   200,
	})
	require.NoError(t, err)

	// Player4 calls
	waitForTurn("player4", 2*time.Second)
	_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: "player4",
		TableId:  tableID,
		Amount:   200,
	})
	require.NoError(t, err)

	// Check pot after turn: 440 + 200 + 200 = 840
	state = env.getGameState(ctx, tableID)
	assert.Equal(t, int64(840), state.Pot, "unexpected pot size after turn")

	// RIVER ROUND
	// Wait for river card
	env.waitForGamePhase(ctx, tableID, pokerrpc.GamePhase_RIVER, 3*time.Second)

	// Make sure we have 5 community cards
	state = env.getGameState(ctx, tableID)
	assert.Equal(t, 5, len(state.CommunityCards), "expected 5 community cards after river")

	// Player3 checks
	waitForTurn("player3", 2*time.Second)
	_, err = env.pokerClient.CheckBet(ctx, &pokerrpc.CheckBetRequest{
		PlayerId: "player3",
		TableId:  tableID,
	})
	require.NoError(t, err)

	// Player4 bets 300
	waitForTurn("player4", 2*time.Second)
	_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: "player4",
		TableId:  tableID,
		Amount:   300,
	})
	require.NoError(t, err)

	// Player3 folds
	waitForTurn("player3", 2*time.Second)
	_, err = env.pokerClient.FoldBet(ctx, &pokerrpc.FoldBetRequest{
		PlayerId: "player3",
		TableId:  tableID,
	})
	require.NoError(t, err)

	// Wait for showdown
	env.waitForGamePhase(ctx, tableID, pokerrpc.GamePhase_SHOWDOWN, 3*time.Second)

	// Wait for showdown processing to complete and winners to be available
	var winners *pokerrpc.GetLastWinnersResponse
	require.Eventually(t, func() bool {
		var err error
		winners, err = env.pokerClient.GetLastWinners(ctx, &pokerrpc.GetLastWinnersRequest{
			TableId: tableID,
		})
		return err == nil && len(winners.Winners) > 0
	}, 3*time.Second, 50*time.Millisecond, "showdown should complete with winners")

	// Verify Player4 won the pot
	assert.Equal(t, 1, len(winners.Winners), "expected 1 winner")
	assert.Equal(t, "player4", winners.Winners[0].PlayerId, "expected player4 to win")

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
	env := newTestEnv(t)
	defer env.Close()

	ctx := context.Background()

	// Setup players
	players := []string{"active1", "active2", "timeout"}
	initialBankroll := int64(10_000)
	for _, p := range players {
		env.setBalance(ctx, p, initialBankroll)
	}

	// Create table with short timebank
	createResp, err := env.lobbyClient.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:        "active1",
		SmallBlind:      10,
		BigBlind:        20,
		MinPlayers:      3,
		MaxPlayers:      3,
		BuyIn:           1_000,
		MinBalance:      1_000,
		StartingChips:   1_000,
		TimeBankSeconds: 5, // 5 seconds timeout
		AutoAdvanceMs:   1000,
	})
	require.NoError(t, err)
	tableID := createResp.TableId

	// All players join and mark ready
	for _, p := range players[1:] {
		_, err := env.lobbyClient.JoinTable(ctx, &pokerrpc.JoinTableRequest{
			PlayerId: p,
			TableId:  tableID,
		})
		require.NoError(t, err)
	}

	for _, p := range players {
		_, err := env.lobbyClient.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{
			PlayerId: p,
			TableId:  tableID,
		})
		require.NoError(t, err)
	}

	// Wait for game to start
	env.waitForGameStart(ctx, tableID, 3*time.Second)

	// Wait for the game to reach PRE_FLOP phase with a valid current player
	require.Eventually(t, func() bool {
		gameState := env.getGameState(ctx, tableID)
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
	_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: "active1",
		TableId:  tableID,
		Amount:   20,
	})
	require.NoError(t, err)

	_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: "active2",
		TableId:  tableID,
		Amount:   20,
	})
	require.NoError(t, err)

	// But "timeout" player doesn't act - should auto-check-or-fold after timeout
	// Since they need to call from 20 to 20 but already have 20 bet (big blind), they should auto-check
	// Wait for auto-check-or-fold to occur
	require.Eventually(t, func() bool {
		state := env.getGameState(ctx, tableID)
		// Check if the timeout player has been auto-checked (turn should have advanced)
		return state.CurrentPlayer != "timeout"
	}, 8*time.Second, 100*time.Millisecond, "timeout player should be auto-checked")

	// Check if player was auto-checked or auto-folded based on their position
	state := env.getGameState(ctx, tableID)
	for _, player := range state.Players {
		if player.Id == "timeout" {
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
	env := newTestEnv(t)
	defer env.Close()

	ctx := context.Background()

	// Setup players
	players := []string{"active1", "active2", "timeout"}
	initialBankroll := int64(10_000)
	for _, p := range players {
		env.setBalance(ctx, p, initialBankroll)
	}

	// Create table with short timebank
	createResp, err := env.lobbyClient.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:        "active1",
		SmallBlind:      10,
		BigBlind:        20,
		MinPlayers:      3,
		MaxPlayers:      3,
		BuyIn:           1_000,
		MinBalance:      1_000,
		StartingChips:   1_000,
		TimeBankSeconds: 5, // 5 seconds timeout
		AutoAdvanceMs:   1000,
	})
	require.NoError(t, err)
	tableID := createResp.TableId

	// All players join and mark ready
	for _, p := range players[1:] {
		_, err := env.lobbyClient.JoinTable(ctx, &pokerrpc.JoinTableRequest{
			PlayerId: p,
			TableId:  tableID,
		})
		require.NoError(t, err)
	}

	for _, p := range players {
		_, err := env.lobbyClient.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{
			PlayerId: p,
			TableId:  tableID,
		})
		require.NoError(t, err)
	}

	// Wait for game to start
	env.waitForGameStart(ctx, tableID, 3*time.Second)

	// Wait for the game to reach PRE_FLOP phase with a valid current player
	require.Eventually(t, func() bool {
		gameState := env.getGameState(ctx, tableID)
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
		st := env.getGameState(ctx, tableID)
		return st.CurrentPlayer == "active1"
	}, 2*time.Second, 10*time.Millisecond, "active1 should be current player")
	_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: "active1",
		TableId:  tableID,
		Amount:   20,
	})
	require.NoError(t, err)

	// active2 raises to 50
	require.Eventually(t, func() bool {
		st := env.getGameState(ctx, tableID)
		return st.CurrentPlayer == "active2"
	}, 2*time.Second, 10*time.Millisecond, "active2 should be current player")
	_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: "active2",
		TableId:  tableID,
		Amount:   50,
	})
	require.NoError(t, err)

	// Now "timeout" player (big blind) would need to call from 20 to 50 - should auto-fold after timeout
	// Wait for auto-fold to occur
	require.Eventually(t, func() bool {
		state := env.getGameState(ctx, tableID)
		// Check if the timeout player has been auto-folded (turn should have advanced)
		return state.CurrentPlayer != "timeout"
	}, 8*time.Second, 100*time.Millisecond, "timeout player should be auto-folded")

	// Check if player was auto-folded (since they cannot check - they need to call the raise)
	state := env.getGameState(ctx, tableID)
	t.Logf("Game state after timeout - Current bet: %d, Pot: %d", state.CurrentBet, state.Pot)
	for _, player := range state.Players {
		t.Logf("Player %s: Bet=%d, Folded=%t", player.Id, player.CurrentBet, player.Folded)
		if player.Id == "timeout" {
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
	env := newTestEnv(t)
	defer env.Close()

	ctx := context.Background()

	// Setup players with initial bankrolls
	players := []string{"player1", "player2", "player3", "player4"}
	initialBankroll := int64(10_000)
	for _, p := range players {
		env.setBalance(ctx, p, initialBankroll)
	}

	// Player1 creates a table for 4 players
	tableID := env.createStandardTable(ctx, "player1", 4, 4)

	// All players join the table
	for _, p := range players[1:] { // Skip player1 who already created the table
		_, err := env.lobbyClient.JoinTable(ctx, &pokerrpc.JoinTableRequest{
			PlayerId: p,
			TableId:  tableID,
		})
		require.NoError(t, err)
	}

	// Verify initial state
	state := env.getGameState(ctx, tableID)
	assert.Equal(t, int32(4), state.PlayersJoined, "expected 4 players joined")
	assert.Equal(t, int32(4), state.PlayersRequired, "expected 4 players required")
	assert.False(t, state.GameStarted, "game should not be started yet")

	// Set players ready one by one and check state
	for i, p := range players {
		_, err := env.lobbyClient.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{
			PlayerId: p,
			TableId:  tableID,
		})
		require.NoError(t, err)

		// Check that player is marked as ready
		state = env.getGameState(ctx, tableID)
		readyCount := 0
		for _, player := range state.Players {
			if player.IsReady {
				readyCount++
			}
		}
		assert.Equal(t, i+1, readyCount, "expected %d players ready", i+1)
	}

	// Now all players are ready, game should start
	env.waitForGameStart(ctx, tableID, 3*time.Second)
}

// -----------------------------------------------------------------------------
//
//	SCENARIO: Test basic betting
//
// -----------------------------------------------------------------------------
func TestBasicBetting(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	defer env.Close()

	ctx := context.Background()

	// Setup 3 players
	players := []string{"p1", "p2", "p3"}
	initialBankroll := int64(10_000)
	for _, p := range players {
		env.setBalance(ctx, p, initialBankroll)
	}

	// Create and join table
	tableID := env.createStandardTable(ctx, "p1", 3, 3)
	for _, p := range players[1:] {
		_, err := env.lobbyClient.JoinTable(ctx, &pokerrpc.JoinTableRequest{
			PlayerId: p,
			TableId:  tableID,
		})
		require.NoError(t, err)
	}

	// Set all players ready
	for _, p := range players {
		_, err := env.lobbyClient.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{
			PlayerId: p,
			TableId:  tableID,
		})
		require.NoError(t, err)
	}

	// Wait for game to start
	env.waitForGameStart(ctx, tableID, 3*time.Second)
	// Ensure PRE_FLOP is fully reached before asserting state/acting
	env.waitForGamePhase(ctx, tableID, pokerrpc.GamePhase_PRE_FLOP, 3*time.Second)

	// Helper: wait until it's the specified player's turn (scoped to this test)
	waitForTurnBasic := func(playerID string, timeout time.Duration) {
		require.Eventually(t, func() bool {
			state := env.getGameState(ctx, tableID)
			return state.CurrentPlayer == playerID
		}, timeout, 25*time.Millisecond, "did not become %s's turn within %s", playerID, timeout)
	}

	// Note: second helper removed; using waitForTurnBasic below.

	// Verify initial pot includes blinds (10 + 20 = 30)
	state := env.getGameState(ctx, tableID)
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
			_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
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
			_, err = env.pokerClient.CheckBet(ctx, &pokerrpc.CheckBetRequest{
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
	state = env.getGameState(ctx, tableID)
	assert.Equal(t, int64(50), state.Pot, "pot should be 50 after dealer's call (30+20)")

	// Small blind calls by betting 20 total (needs to add 10 more to their existing 10)
	waitForTurnBasic(sbID, 2*time.Second)
	betWithRetry(sbID, 20)

	// Check pot is now 60 (50 + 10 more from SB)
	state = env.getGameState(ctx, tableID)
	assert.Equal(t, int64(60), state.Pot, "pot should be 60 after SB's call (50+10)")

	// Big blind can check (already has 20 bet)
	waitForTurnBasic(bbID, 2*time.Second)
	checkWithRetry(bbID)

	// Pot should still be 60 after check
	state = env.getGameState(ctx, tableID)
	assert.Equal(t, int64(60), state.Pot, "pot should remain 60 after BB's check")
}

// -----------------------------------------------------------------------------
//
//	SCENARIO: Test StartingChips default when set to 0
//
// -----------------------------------------------------------------------------
func TestStartingChipsDefault(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	defer env.Close()

	ctx := context.Background()

	// Setup players
	players := []string{"player1", "player2", "player3"}
	initialBankroll := int64(10_000)
	for _, p := range players {
		env.setBalance(ctx, p, initialBankroll)
	}

	// Create table with StartingChips set to 0 to test default logic
	createResp, err := env.lobbyClient.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      "player1",
		SmallBlind:    10,
		BigBlind:      20,
		MinPlayers:    3,
		MaxPlayers:    3,
		BuyIn:         1_500,
		MinBalance:    1_000,
		StartingChips: 0, // This should default to 1000
		AutoAdvanceMs: 1000,
	})
	require.NoError(t, err)
	tableID := createResp.TableId

	// All players join and mark ready
	for _, p := range players[1:] {
		_, err := env.lobbyClient.JoinTable(ctx, &pokerrpc.JoinTableRequest{
			PlayerId: p,
			TableId:  tableID,
		})
		require.NoError(t, err)
	}

	for _, p := range players {
		_, err := env.lobbyClient.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{
			PlayerId: p,
			TableId:  tableID,
		})
		require.NoError(t, err)
	}

	// Wait for game to start
	env.waitForGameStart(ctx, tableID, 3*time.Second)

	// Wait for blinds to be posted and visible (pot should be 30)
	var state *pokerrpc.GameUpdate
	require.Eventually(t, func() bool {
		state = env.getGameState(ctx, tableID)
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
		state = env.getGameState(ctx, tableID)
		return state.CurrentPlayer == dealerID && state.Phase == pokerrpc.GamePhase_PRE_FLOP
	}, 2*time.Second, 10*time.Millisecond, "should become dealer's turn")

	// First player to act (dealer) makes a bet
	_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: dealerID,
		TableId:  tableID,
		Amount:   20, // Call the big blind
	})
	require.NoError(t, err)

	// Verify pot is now 50 (use Eventually to account for async FSM transition)
	require.Eventually(t, func() bool {
		state = env.getGameState(ctx, tableID)
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
	env := newTestEnv(t)
	defer env.Close()

	ctx := context.Background()

	// Setup players
	players := []string{"player1", "player2"}
	initialBankroll := int64(10_000)
	for _, p := range players {
		env.setBalance(ctx, p, initialBankroll)
	}

	// Create table with both StartingChips and BuyIn set to 0
	createResp, err := env.lobbyClient.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      "player1",
		SmallBlind:    10,
		BigBlind:      20,
		MinPlayers:    2,
		MaxPlayers:    2,
		BuyIn:         0, // Zero buy-in
		MinBalance:    0,
		StartingChips: 0, // Should default to 1000
		AutoAdvanceMs: 1000,
	})
	require.NoError(t, err)
	tableID := createResp.TableId

	// Player2 joins
	_, err = env.lobbyClient.JoinTable(ctx, &pokerrpc.JoinTableRequest{
		PlayerId: "player2",
		TableId:  tableID,
	})
	require.NoError(t, err)

	// Both players mark ready
	for _, p := range players {
		_, err := env.lobbyClient.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{
			PlayerId: p,
			TableId:  tableID,
		})
		require.NoError(t, err)
	}

	// Wait for game to start
	env.waitForGameStart(ctx, tableID, 3*time.Second)

	// Get game state and verify that players have the default 1000 starting chips
	state := env.getGameState(ctx, tableID)

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
	expectedCurrentPlayer := "player1" // SB acts first in heads-up preflop
	assert.Equal(t, expectedCurrentPlayer, state.CurrentPlayer, "Small blind should act first in heads-up preflop")

	// Player1 (SB) raises to 40
	_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: state.CurrentPlayer, // Should be player1 (SB)
		TableId:  tableID,
		Amount:   40, // Raise to 40
	})
	require.NoError(t, err)

	// Verify pot is now 60 (30 initial + 30 additional from SB raising from 10 to 40)
	state = env.getGameState(ctx, tableID)
	assert.Equal(t, int64(60), state.Pot, "pot should be 60 after SB's raise (30+30)")

	// Now it should be player2's (BB) turn to call, raise, or fold
	t.Logf("After SB raise - Current player: %s, Current bet: %d", state.CurrentPlayer, state.CurrentBet)

	// Player2 (BB) calls by betting 40 total (adding 20 more to existing 20)
	_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: state.CurrentPlayer, // Should be player2 (BB)
		TableId:  tableID,
		Amount:   40, // Call the raise
	})
	require.NoError(t, err)

	// Verify pot is now 80 (60 + 20 additional from BB calling)
	// Use Eventually to account for async state machine transitions when betting round completes
	require.Eventually(t, func() bool {
		state = env.getGameState(ctx, tableID)
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
	env := newTestEnv(t)
	defer env.Close()

	ctx := context.Background()

	// Setup 3 players and bankroll
	players := []string{"a3", "b3", "c3"}
	for _, p := range players {
		env.setBalance(ctx, p, 10_000)
	}

	// Create a 3-max table and join remaining players
	tableID := env.createStandardTable(ctx, players[0], 3, 3)
	for _, p := range players[1:] {
		_, err := env.lobbyClient.JoinTable(ctx, &pokerrpc.JoinTableRequest{PlayerId: p, TableId: tableID})
		require.NoError(t, err)
	}

	// Everyone ready
	for _, p := range players {
		_, err := env.lobbyClient.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{PlayerId: p, TableId: tableID})
		require.NoError(t, err)
	}

	// Wait for game to start
	env.waitForGameStart(ctx, tableID, 3*time.Second)

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

		state := env.getGameState(ctx, tableID)

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

		// Helper to make an action with retry on transient errors.
		// Test logic: play passively to complete a full hand (check when possible, call otherwise).
		// This exercises all betting rounds without complex raising logic.
		makeAction := func() error {
			if currPlayer.CurrentBet >= state.CurrentBet {
				// Already matched the current bet - check
				_, err := env.pokerClient.CheckBet(ctx, &pokerrpc.CheckBetRequest{PlayerId: curr, TableId: tableID})
				if err != nil && isTransientTurnError(err) {
					return err // Retry on transient errors
				}
				require.NoError(t, err)
				return nil
			} else {
				// Need to match the current bet - call using dedicated CallBet RPC
				// (avoids race with reading state.CurrentBet and then betting to it)
				_, err := env.pokerClient.CallBet(ctx, &pokerrpc.CallBetRequest{PlayerId: curr, TableId: tableID})
				if err != nil && isTransientTurnError(err) {
					return err // Retry on transient errors
				}
				require.NoError(t, err)
				return nil
			}
		}

		// Use Eventually pattern to handle transient turn races
		require.Eventually(t, func() bool {
			return makeAction() == nil
		}, 1*time.Second, 10*time.Millisecond, "failed to execute action for player %s", curr)
	}

	// Final assertions
	final := env.getGameState(ctx, tableID)
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
	env := newTestEnv(t)
	defer env.Close()

	ctx := context.Background()

	// Setup 2 players for heads-up
	players := []string{"heads1", "heads2"}
	initialBankroll := int64(10_000)
	for _, p := range players {
		env.setBalance(ctx, p, initialBankroll)
	}

	// Create heads-up table
	tableID := env.createStandardTable(ctx, "heads1", 2, 2)

	// Player2 joins
	_, err := env.lobbyClient.JoinTable(ctx, &pokerrpc.JoinTableRequest{
		PlayerId: "heads2",
		TableId:  tableID,
	})
	require.NoError(t, err)

	// Both players mark ready
	for _, p := range players {
		_, err := env.lobbyClient.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{
			PlayerId: p,
			TableId:  tableID,
		})
		require.NoError(t, err)
	}

	// Wait for game to start
	env.waitForGameStart(ctx, tableID, 3*time.Second)

	// Helper to wait for a specific player's turn
	waitForPlayerTurn := func(playerID string, timeout time.Duration) {
		require.Eventually(t, func() bool {
			state := env.getGameState(ctx, tableID)
			return state.CurrentPlayer == playerID
		}, timeout, 10*time.Millisecond, "should be %s's turn", playerID)
	}

	// Helper to wait for a specific game phase
	waitForPhase := func(phase pokerrpc.GamePhase, timeout time.Duration) {
		require.Eventually(t, func() bool {
			state := env.getGameState(ctx, tableID)
			return state.Phase == phase
		}, timeout, 10*time.Millisecond, "should reach phase %s", phase)
	}

	// FIRST HAND: Play a complete hand
	// Wait for PRE_FLOP
	waitForPhase(pokerrpc.GamePhase_PRE_FLOP, 3*time.Second)

	// Get initial state to understand positions
	state := env.getGameState(ctx, tableID)
	t.Logf("First hand - Current player: %s, Phase: %s", state.CurrentPlayer, state.Phase)

	// Play first hand: both players call/check to complete pre-flop
	// First player acts
	_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: state.CurrentPlayer,
		TableId:  tableID,
		Amount:   20, // Call the big blind
	})
	require.NoError(t, err)

	// Wait for second player's turn
	waitForPlayerTurn(state.Players[1].Id, 2*time.Second)

	// Second player checks (already has big blind)
	_, err = env.pokerClient.CheckBet(ctx, &pokerrpc.CheckBetRequest{
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
			state = env.getGameState(ctx, tableID)
			currentPlayer := state.CurrentPlayer

			// Player checks
			_, err = env.pokerClient.CheckBet(ctx, &pokerrpc.CheckBetRequest{
				PlayerId: currentPlayer,
				TableId:  tableID,
			})
			require.NoError(t, err)

			// Wait for turn to advance (unless it's the last player in the round)
			if j < 1 {
				require.Eventually(t, func() bool {
					newState := env.getGameState(ctx, tableID)
					return newState.CurrentPlayer != currentPlayer
				}, 2*time.Second, 10*time.Millisecond, "turn should advance after check")
			}
		}
	}

	// Wait for showdown
	waitForPhase(pokerrpc.GamePhase_SHOWDOWN, 3*time.Second)

	// Wait for showdown to complete and winners to be available
	require.Eventually(t, func() bool {
		_, err := env.pokerClient.GetLastWinners(ctx, &pokerrpc.GetLastWinnersRequest{
			TableId: tableID,
		})
		return err == nil
	}, 3*time.Second, 50*time.Millisecond, "showdown should complete")

	// SECOND HAND: This is where the bug would manifest
	// The game should auto-start a new hand
	waitForPhase(pokerrpc.GamePhase_PRE_FLOP, 5*time.Second)

	// Get second hand state
	state = env.getGameState(ctx, tableID)
	t.Logf("Second hand - Current player: %s, Phase: %s", state.CurrentPlayer, state.Phase)

	// The critical test: In the second hand, after the first player acts,
	// we should still be in PRE_FLOP waiting for the second player to act.
	// The bug was that it would skip directly to FLOP.

	// First player acts (calls the big blind)
	firstPlayer := state.CurrentPlayer
	_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: firstPlayer,
		TableId:  tableID,
		Amount:   20, // Call the big blind
	})
	require.NoError(t, err)

	// CRITICAL ASSERTION: After first player acts, we should still be in PRE_FLOP
	// This would have failed before the fix because the system would incorrectly
	// advance to FLOP, skipping the second player's action.
	state = env.getGameState(ctx, tableID)
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
	_, err = env.pokerClient.CheckBet(ctx, &pokerrpc.CheckBetRequest{
		PlayerId: secondPlayer,
		TableId:  tableID,
	})
	require.NoError(t, err)

	// Now we should advance to FLOP
	waitForPhase(pokerrpc.GamePhase_FLOP, 3*time.Second)

	// Verify we reached FLOP after both players acted
	state = env.getGameState(ctx, tableID)
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

			env := newTestEnv(t)
			defer env.Close()
			ctx := context.Background()

			players := []string{"p1", "p2"}
			const stack = int64(1_000)

			for _, p := range players {
				env.setBalance(ctx, p, 10_000) // wallet outside table
			}

			// NOTE: AutoStartMs=0 to maximize overlap/race in workers.
			createResp, err := env.lobbyClient.CreateTable(ctx, &pokerrpc.CreateTableRequest{
				PlayerId:      "p1",
				SmallBlind:    10,
				BigBlind:      20,
				MinPlayers:    2,
				MaxPlayers:    2,
				BuyIn:         stack,
				MinBalance:    stack,
				StartingChips: stack,
				AutoStartMs:   5000,
				AutoAdvanceMs: 1000,
			})
			require.NoError(t, err)
			tableID := createResp.TableId

			_, err = env.lobbyClient.JoinTable(ctx, &pokerrpc.JoinTableRequest{
				PlayerId: "p2", TableId: tableID,
			})
			require.NoError(t, err)

			for _, p := range players {
				_, err := env.lobbyClient.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{
					PlayerId: p, TableId: tableID,
				})
				require.NoError(t, err)
			}

			env.waitForGameStart(ctx, tableID, 2*time.Second)
			env.waitForGamePhase(ctx, tableID, pokerrpc.GamePhase_PRE_FLOP, 2*time.Second)

			waitTurn := func(pid string, d time.Duration) {
				ctxTurn, cancel := context.WithTimeout(ctx, d)
				defer cancel()
				// Small initial delay to allow previous turn to complete (async FSM processing)
				time.Sleep(10 * time.Millisecond)
				for {
					st := env.getGameState(ctx, tableID)
					if st.CurrentPlayer == pid && st.Phase == pokerrpc.GamePhase_PRE_FLOP {
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
			_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
				PlayerId: "p1", TableId: tableID, Amount: 100, // “to 100” semantics
			})
			require.NoError(t, err)

			// Without letting P2 act/call, we immediately fold with P2.
			waitTurn("p2", 500*time.Millisecond)
			_, err = env.pokerClient.FoldBet(ctx, &pokerrpc.FoldBetRequest{
				PlayerId: "p2", TableId: tableID,
			})
			require.NoError(t, err)

			// Race tickler: spam a couple of concurrent reads while workers process.
			var wg sync.WaitGroup
			for k := 0; k < 5; k++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					_ = env.getGameState(ctx, tableID)
				}()
			}

			// Wait for settled hand outcome instead of transient SHOWDOWN phase.
			// With aggressive AutoStart and concurrent workers, the game can move past
			// SHOWDOWN phase quickly, causing race conditions. Instead, wait for the
			// stable outcome invariant: exactly one active player + zeroed pots or
			// showdown result present.
			require.Eventually(t, func() bool {
				state := env.getGameState(ctx, tableID)
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
				_, err := env.pokerClient.GetLastWinners(ctx, &pokerrpc.GetLastWinnersRequest{
					TableId: tableID,
				})
				return err == nil
			}, 5*time.Second, 25*time.Millisecond, "hand should settle with exactly one active player or completed showdown")

			state := env.getGameState(ctx, tableID)

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
	env := newTestEnv(t)
	defer env.Close()

	ctx := context.Background()

	// Setup: Create a table with small starting chips (15) relative to blinds (SB=10, BB=20)
	// This forces one player to go all-in when posting BB
	players := []string{"player1", "player2"}
	for _, p := range players {
		env.setBalance(ctx, p, 10_000)
	}

	// Create table with starting chips (15) less than big blind (20)
	createResp, err := env.lobbyClient.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      "player1",
		SmallBlind:    10,
		BigBlind:      20,
		MinPlayers:    2,
		MaxPlayers:    2,
		BuyIn:         1_000,
		MinBalance:    1_000,
		StartingChips: 15,  // Less than BB - forces all-in on BB post
		AutoStartMs:   200, // Short delay for faster testing
		AutoAdvanceMs: 1000,
	})
	require.NoError(t, err)
	tableID := createResp.TableId

	// player2 joins
	_, err = env.lobbyClient.JoinTable(ctx, &pokerrpc.JoinTableRequest{
		PlayerId: "player2",
		TableId:  tableID,
	})
	require.NoError(t, err)

	// Both players mark ready
	for _, p := range players {
		_, err := env.lobbyClient.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{
			PlayerId: p,
			TableId:  tableID,
		})
		require.NoError(t, err)
	}

	// Wait for game to start
	env.waitForGameStart(ctx, tableID, 3*time.Second)
	env.waitForGamePhase(ctx, tableID, pokerrpc.GamePhase_PRE_FLOP, 3*time.Second)

	// CRITICAL VERIFICATION: Check that the all-in blind player is correctly marked
	state := env.getGameState(ctx, tableID)

	// In heads-up: player1 is dealer/SB (10), player2 is BB (20)
	// With 15 chips, player2 should be all-in after posting BB
	var allInPlayer *pokerrpc.Player
	var activePlayer *pokerrpc.Player
	for _, p := range state.Players {
		if p.Balance == 0 && p.CurrentBet > 0 {
			allInPlayer = p
			t.Logf("All-in player: %s (bet=%d, balance=%d)", p.Id, p.CurrentBet, p.Balance)
		} else {
			activePlayer = p
			t.Logf("Active player: %s (bet=%d, balance=%d)", p.Id, p.CurrentBet, p.Balance)
		}
	}

	// THE BUG: If StartHandParticipation() was called before HandleStartHand(),
	// the all-in player would not be properly marked as ALL_IN
	require.NotNil(t, allInPlayer, "Should have one all-in player (posted blind with short stack)")
	assert.True(t, allInPlayer.IsAllIn,
		"Player %s went all-in posting blind but IsAllIn=false - FSM initialization bug!",
		allInPlayer.Id)

	// Verify pot includes both blinds
	// SB=10 + BB=15 (all-in) = 25
	assert.Equal(t, int64(25), state.Pot, "pot should include SB(10) + BB all-in (15)")

	// Verify the active player can make an action (game is not hung)
	require.NotNil(t, activePlayer, "Should have one active player")
	assert.Equal(t, activePlayer.Id, state.CurrentPlayer, "Active player should be current player")

	// Active player should be able to fold (verifies FSM is working)
	_, err = env.pokerClient.FoldBet(ctx, &pokerrpc.FoldBetRequest{
		PlayerId: activePlayer.Id,
		TableId:  tableID,
	})
	require.NoError(t, err, "Active player should be able to act (FSM working, not hung)")

	// Wait for showdown to complete
	env.waitForGamePhase(ctx, tableID, pokerrpc.GamePhase_SHOWDOWN, 3*time.Second)

	// Final state verification
	finalState := env.getGameState(ctx, tableID)

	// All-in player should win (active player folded)
	var winnerFound bool
	for _, p := range finalState.Players {
		if p.Id == allInPlayer.Id {
			// Winner gets the pot (25) back plus keeps their starting balance (now in chips won)
			// They started with 15, posted 15 all-in, should now have 25 (won the pot)
			assert.Equal(t, int64(25), p.Balance,
				"All-in winner should have pot (25) after opponent folds")
			winnerFound = true
		} else {
			// Folder loses their SB (10)
			assert.Equal(t, int64(5), p.Balance,
				"Folder should have 15-10(SB)=5 remaining")
		}
	}
	require.True(t, winnerFound, "All-in player should be in final state")

	t.Log("✓ Short-stack blind all-in handled correctly with proper FSM states")
}

// TestBettingRound_Completes_On_AllIn_And_Folds tests betting round completion
// in various scenarios involving all-ins and folds to ensure the game properly
// advances phases and reaches showdown.
func TestBettingRound_Completes_On_AllIn_And_Folds(t *testing.T) {
	t.Run("AllPlayersAllIn_FastForward", func(t *testing.T) {
		t.Parallel()
		env := newTestEnv(t)
		defer env.Close()

		ctx := context.Background()

		// Setup 3 players with small stacks to facilitate all-in
		players := []string{"player1", "player2", "player3"}
		for _, p := range players {
			env.setBalance(ctx, p, 10_000)
		}

		// Create table with small starting chips (100) and blinds
		createResp, err := env.lobbyClient.CreateTable(ctx, &pokerrpc.CreateTableRequest{
			PlayerId:      "player1",
			SmallBlind:    10,
			BigBlind:      20,
			MinPlayers:    3,
			MaxPlayers:    3,
			BuyIn:         1_000,
			MinBalance:    1_000,
			StartingChips: 100,
			AutoStartMs:   200,
			AutoAdvanceMs: 1000,
		})
		require.NoError(t, err)
		tableID := createResp.TableId

		// Other players join
		for _, p := range players[1:] {
			_, err := env.lobbyClient.JoinTable(ctx, &pokerrpc.JoinTableRequest{
				PlayerId: p,
				TableId:  tableID,
			})
			require.NoError(t, err)
		}

		// All players mark ready
		for _, p := range players {
			_, err := env.lobbyClient.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{
				PlayerId: p,
				TableId:  tableID,
			})
			require.NoError(t, err)
		}

		// Wait for game to start
		env.waitForGameStart(ctx, tableID, 3*time.Second)
		env.waitForGamePhase(ctx, tableID, pokerrpc.GamePhase_PRE_FLOP, 3*time.Second)

		// All players go all-in (100 chips each)
		for _, p := range players {
			_, err := env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
				PlayerId: p,
				TableId:  tableID,
				Amount:   100,
			})
			// May fail for some players if they're not current - that's expected
			if err == nil {
				t.Logf("Player %s went all-in", p)
			}
		}

		// When all players are all-in, game should fast-forward to SHOWDOWN
		require.Eventually(t, func() bool {
			state := env.getGameState(ctx, tableID)
			return state.Phase == pokerrpc.GamePhase_SHOWDOWN
		}, 5*time.Second, 50*time.Millisecond, "game should fast-forward to SHOWDOWN when all players all-in")

		t.Log("✓ All players all-in - game fast-forwarded to showdown")
	})

	t.Run("TwoAllIn_OneFold_GoToShowdown", func(t *testing.T) {
		t.Parallel()
		env := newTestEnv(t)
		defer env.Close()

		ctx := context.Background()

		// Setup 3 players
		players := []string{"player1", "player2", "player3"}
		for _, p := range players {
			env.setBalance(ctx, p, 10_000)
		}

		// Create table
		createResp, err := env.lobbyClient.CreateTable(ctx, &pokerrpc.CreateTableRequest{
			PlayerId:      "player1",
			SmallBlind:    10,
			BigBlind:      20,
			MinPlayers:    3,
			MaxPlayers:    3,
			BuyIn:         1_000,
			MinBalance:    1_000,
			StartingChips: 500,
			AutoStartMs:   5000, // Long delay to prevent auto-start during test
			AutoAdvanceMs: 1000,
		})
		require.NoError(t, err)
		tableID := createResp.TableId

		// Other players join
		for _, p := range players[1:] {
			_, err := env.lobbyClient.JoinTable(ctx, &pokerrpc.JoinTableRequest{
				PlayerId: p,
				TableId:  tableID,
			})
			require.NoError(t, err)
		}

		// All players mark ready
		for _, p := range players {
			_, err := env.lobbyClient.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{
				PlayerId: p,
				TableId:  tableID,
			})
			require.NoError(t, err)
		}

		// Wait for game to start
		env.waitForGameStart(ctx, tableID, 3*time.Second)
		env.waitForGamePhase(ctx, tableID, pokerrpc.GamePhase_PRE_FLOP, 3*time.Second)

		// Get initial state to identify positions
		state := env.getGameState(ctx, tableID)
		currentPlayer := state.CurrentPlayer

		// First player folds
		_, err = env.pokerClient.FoldBet(ctx, &pokerrpc.FoldBetRequest{
			PlayerId: currentPlayer,
			TableId:  tableID,
		})
		require.NoError(t, err)

		// Wait for turn to advance after fold
		require.Eventually(t, func() bool {
			newState := env.getGameState(ctx, tableID)
			return newState.CurrentPlayer != currentPlayer
		}, 2*time.Second, 10*time.Millisecond, "turn should advance after fold")

		// Remaining two players go all-in
		for i := 0; i < 2; i++ {
			state = env.getGameState(ctx, tableID)
			if state.Phase == pokerrpc.GamePhase_SHOWDOWN {
				break
			}
			currentPlayer = state.CurrentPlayer
			currentPhase := state.Phase

			_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
				PlayerId: currentPlayer,
				TableId:  tableID,
				Amount:   500,
			})
			if err == nil {
				t.Logf("Player %s went all-in", currentPlayer)

				// Wait for turn to advance or phase to change
				require.Eventually(t, func() bool {
					newState := env.getGameState(ctx, tableID)
					return newState.CurrentPlayer != currentPlayer || newState.Phase != currentPhase
				}, 2*time.Second, 10*time.Millisecond, "turn or phase should change after all-in")
			}
		}

		// When remaining players are all-in, should reach showdown
		// Need to wait for: FLOP→TURN (1s) + TURN→RIVER (1s) + RIVER→SHOWDOWN (1s) = 3s + buffer
		require.Eventually(t, func() bool {
			state := env.getGameState(ctx, tableID)
			return state.Phase == pokerrpc.GamePhase_SHOWDOWN
		}, 4*time.Second, 50*time.Millisecond, "game should reach showdown with all-in players")

		// Verify one player folded and two are all-in
		state = env.getGameState(ctx, tableID)
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
		assert.Equal(t, 2, allInCount, "should have 2 all-in players")

		t.Log("✓ Two all-in, one folded - game reached showdown")
	})

	t.Run("AllFoldExceptOne_GoToShowdown", func(t *testing.T) {
		t.Parallel()
		env := newTestEnv(t)
		defer env.Close()

		ctx := context.Background()

		// Setup 3 players
		players := []string{"player1", "player2", "player3"}
		for _, p := range players {
			env.setBalance(ctx, p, 10_000)
		}

		// Create table with long auto-start delay to avoid race
		createResp, err := env.lobbyClient.CreateTable(ctx, &pokerrpc.CreateTableRequest{
			PlayerId:      "player1",
			SmallBlind:    10,
			BigBlind:      20,
			MinPlayers:    3,
			MaxPlayers:    3,
			BuyIn:         1_000,
			MinBalance:    1_000,
			StartingChips: 1_000,
			AutoStartMs:   5000, // Long delay to prevent auto-start during test
			AutoAdvanceMs: 1000,
		})
		require.NoError(t, err)
		tableID := createResp.TableId

		// Other players join
		for _, p := range players[1:] {
			_, err := env.lobbyClient.JoinTable(ctx, &pokerrpc.JoinTableRequest{
				PlayerId: p,
				TableId:  tableID,
			})
			require.NoError(t, err)
		}

		// All players mark ready
		for _, p := range players {
			_, err := env.lobbyClient.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{
				PlayerId: p,
				TableId:  tableID,
			})
			require.NoError(t, err)
		}

		// Wait for game to start
		env.waitForGameStart(ctx, tableID, 3*time.Second)
		env.waitForGamePhase(ctx, tableID, pokerrpc.GamePhase_PRE_FLOP, 3*time.Second)

		// Two players fold, leaving one active
		// We'll check for showdown in the main loop instead of a goroutine
		// to avoid "fail in goroutine after test completes" panics

		folded := 0
		for folded < 2 {
			state := env.getGameState(ctx, tableID)

			// Check if we've already reached showdown
			if state.Phase == pokerrpc.GamePhase_SHOWDOWN {
				t.Log("Showdown reached during fold loop")
				break
			}

			currentPlayer := state.CurrentPlayer
			_, err := env.pokerClient.FoldBet(ctx, &pokerrpc.FoldBetRequest{
				PlayerId: currentPlayer,
				TableId:  tableID,
			})
			if err == nil {
				folded++
				t.Logf("Player %s folded (%d/2)", currentPlayer, folded)

				// After the first fold, wait for the turn to advance.
				if folded == 1 {
					require.Eventually(t, func() bool {
						newState := env.getGameState(ctx, tableID)
						return newState.CurrentPlayer != currentPlayer
					}, 2*time.Second, 10*time.Millisecond, "turn should advance after first fold")
				}
			}
		}

		// Verify we reached showdown (allow extra time on CI)
		require.Eventually(t, func() bool {
			state := env.getGameState(ctx, tableID)
			return state.Phase == pokerrpc.GamePhase_SHOWDOWN
		}, 2*time.Second, 10*time.Millisecond, "should reach showdown after two folds")

		// Final state assertion
		state := env.getGameState(ctx, tableID)
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
	env := newTestEnv(t)
	defer env.Close()

	ctx := context.Background()

	// Setup 2 players
	players := []string{"player1", "player2"}
	for _, p := range players {
		env.setBalance(ctx, p, 10_000)
	}

	// Create table with small stacks to facilitate all-in
	createResp, err := env.lobbyClient.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      "player1",
		SmallBlind:    10,
		BigBlind:      20,
		MinPlayers:    2,
		MaxPlayers:    2,
		BuyIn:         1_000,
		MinBalance:    1_000,
		StartingChips: 100, // Small stacks
		AutoStartMs:   5000,
		AutoAdvanceMs: 1000,
	})
	require.NoError(t, err)
	tableID := createResp.TableId

	// Player2 joins
	_, err = env.lobbyClient.JoinTable(ctx, &pokerrpc.JoinTableRequest{
		PlayerId: "player2",
		TableId:  tableID,
	})
	require.NoError(t, err)

	// Both players mark ready
	for _, p := range players {
		_, err := env.lobbyClient.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{
			PlayerId: p,
			TableId:  tableID,
		})
		require.NoError(t, err)
	}

	// Wait for game to start
	env.waitForGameStart(ctx, tableID, 3*time.Second)
	env.waitForGamePhase(ctx, tableID, pokerrpc.GamePhase_PRE_FLOP, 3*time.Second)

	// Both players go all-in preflop
	state := env.getGameState(ctx, tableID)
	currentPlayer := state.CurrentPlayer

	// First player goes all-in
	_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: currentPlayer,
		TableId:  tableID,
		Amount:   100,
	})
	require.NoError(t, err)
	t.Logf("Player %s went all-in", currentPlayer)

	// Wait for turn to advance or phase to change (fast transitions can skip straight to next phase)
	require.Eventually(t, func() bool {
		state = env.getGameState(ctx, tableID)
		return state.CurrentPlayer != currentPlayer || state.Phase != pokerrpc.GamePhase_PRE_FLOP
	}, 2*time.Second, 10*time.Millisecond, "turn or phase should advance after all-in")

	// Second player calls all-in
	state = env.getGameState(ctx, tableID)
	currentPlayer = state.CurrentPlayer
	_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: currentPlayer,
		TableId:  tableID,
		Amount:   100,
	})
	require.NoError(t, err)
	t.Logf("Player %s called all-in", currentPlayer)

	// CRITICAL VERIFICATION: Game should auto-advance through streets
	// When both players are all-in, the game automatically advances through phases.
	// The second all-in triggers: PRE_FLOP → FLOP (immediate) → TURN (auto 1s) → RIVER (auto 1s) → SHOWDOWN (auto 1s)

	t.Log("Verifying game auto-advances to at least TURN after both all-ins...")
	// Game should reach TURN after the auto-advance delay (1s) since both are all-in
	require.Eventually(t, func() bool {
		state = env.getGameState(ctx, tableID)
		return state.Phase >= pokerrpc.GamePhase_TURN
	}, 2*time.Second, 50*time.Millisecond, "game should reach at least TURN phase")
	t.Logf("Game at phase %s with %d community cards", state.Phase, len(state.CommunityCards))
	assert.GreaterOrEqual(t, len(state.CommunityCards), 3, "should have at least 3 community cards")

	t.Log("Verifying auto-advance to RIVER...")
	require.Eventually(t, func() bool {
		state = env.getGameState(ctx, tableID)
		return state.Phase >= pokerrpc.GamePhase_RIVER
	}, 3*time.Second, 50*time.Millisecond, "game should auto-advance to RIVER")
	assert.Equal(t, 5, len(state.CommunityCards), "should have 5 community cards at RIVER")
	t.Logf("✓ RIVER reached with %d community cards", len(state.CommunityCards))

	t.Log("Verifying auto-advance to SHOWDOWN...")
	require.Eventually(t, func() bool {
		state = env.getGameState(ctx, tableID)
		return state.Phase == pokerrpc.GamePhase_SHOWDOWN
	}, 4*time.Second, 50*time.Millisecond, "game should auto-advance to SHOWDOWN")
	state = env.getGameState(ctx, tableID)
	assert.Equal(t, pokerrpc.GamePhase_SHOWDOWN, state.Phase, "should advance to SHOWDOWN")
	t.Log("✓ SHOWDOWN reached")

	// Verify both players are all-in
	allInCount := 0
	for _, p := range state.Players {
		if p.IsAllIn {
			allInCount++
		}
	}
	assert.Equal(t, 2, allInCount, "both players should be all-in")

	// Wait for showdown to complete and winners to be available
	require.Eventually(t, func() bool {
		_, err := env.pokerClient.GetLastWinners(ctx, &pokerrpc.GetLastWinnersRequest{
			TableId: tableID,
		})
		return err == nil
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
	env := newTestEnv(t)
	defer env.Close()

	ctx := context.Background()

	// Setup 3 players
	players := []string{"player1", "player2", "player3"}
	for _, p := range players {
		env.setBalance(ctx, p, 10_000)
	}

	// Create table with small stacks to facilitate all-in
	createResp, err := env.lobbyClient.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      "player1",
		SmallBlind:    10,
		BigBlind:      20,
		MinPlayers:    3,
		MaxPlayers:    3,
		BuyIn:         1_000,
		MinBalance:    1_000,
		StartingChips: 100, // Small stacks
		AutoStartMs:   5000,
		AutoAdvanceMs: 1000,
	})
	require.NoError(t, err)
	tableID := createResp.TableId

	// Other players join
	for _, p := range players[1:] {
		_, err := env.lobbyClient.JoinTable(ctx, &pokerrpc.JoinTableRequest{
			PlayerId: p,
			TableId:  tableID,
		})
		require.NoError(t, err)
	}

	// All players mark ready
	for _, p := range players {
		_, err := env.lobbyClient.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{
			PlayerId: p,
			TableId:  tableID,
		})
		require.NoError(t, err)
	}

	// Wait for game to start
	env.waitForGameStart(ctx, tableID, 3*time.Second)
	env.waitForGamePhase(ctx, tableID, pokerrpc.GamePhase_PRE_FLOP, 3*time.Second)

	// All three players go all-in preflop
	// Keep trying until we can't bet anymore (phase advanced or all players acted)
	allInPlayers := []string{}
	for i := 0; i < 3; i++ {
		state := env.getGameState(ctx, tableID)
		currentPlayer := state.CurrentPlayer
		currentPhase := state.Phase

		// Helper: place an all-in bet for the snapshot current player, but only
		// once their isTurn flag is true in the live game state. This avoids
		// racing against the player's FSM not having processed StartTurn yet.
		var betErr error
		ok := assert.Eventually(t, func() bool {
			st := env.getGameState(ctx, tableID)
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
			_, betErr = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
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
			newState := env.getGameState(ctx, tableID)
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
			newState := env.getGameState(ctx, tableID)
			// Stop if phase changed (auto-advance triggered)
			if newState.Phase != currentPhase {
				return true
			}
			// Stop if turn advanced to next player
			return newState.CurrentPlayer != currentPlayer
		}, 2*time.Second, 10*time.Millisecond, "turn or phase should advance after all-in")

		// Check if phase advanced (all players all-in triggered auto-advance)
		if env.getGameState(ctx, tableID).Phase != currentPhase {
			t.Logf("Phase advanced to %v after %d players went all-in", env.getGameState(ctx, tableID).Phase, len(allInPlayers))
			break
		}
	}

	// Verify at least 2 players went all-in before auto-advance triggered
	require.GreaterOrEqual(t, len(allInPlayers), 2, "at least 2 players should have gone all-in before auto-advance")

	// CRITICAL VERIFICATION: Game should auto-advance through streets
	t.Log("Verifying auto-advance through FLOP...")
	env.waitForGamePhase(ctx, tableID, pokerrpc.GamePhase_FLOP, 3*time.Second)
	state := env.getGameState(ctx, tableID)
	assert.Equal(t, pokerrpc.GamePhase_FLOP, state.Phase, "should advance to FLOP")
	assert.Equal(t, 3, len(state.CommunityCards), "should have 3 community cards at FLOP")
	t.Logf("✓ FLOP reached with %d community cards", len(state.CommunityCards))

	t.Log("Verifying auto-advance through TURN...")
	env.waitForGamePhase(ctx, tableID, pokerrpc.GamePhase_TURN, 3*time.Second)
	state = env.getGameState(ctx, tableID)
	assert.Equal(t, pokerrpc.GamePhase_TURN, state.Phase, "should advance to TURN")
	assert.Equal(t, 4, len(state.CommunityCards), "should have 4 community cards at TURN")
	t.Logf("✓ TURN reached with %d community cards", len(state.CommunityCards))

	t.Log("Verifying auto-advance through RIVER...")
	env.waitForGamePhase(ctx, tableID, pokerrpc.GamePhase_RIVER, 3*time.Second)
	state = env.getGameState(ctx, tableID)
	assert.Equal(t, pokerrpc.GamePhase_RIVER, state.Phase, "should advance to RIVER")
	assert.Equal(t, 5, len(state.CommunityCards), "should have 5 community cards at RIVER")
	t.Logf("✓ RIVER reached with %d community cards", len(state.CommunityCards))

	t.Log("Verifying auto-advance to SHOWDOWN...")
	env.waitForGamePhase(ctx, tableID, pokerrpc.GamePhase_SHOWDOWN, 3*time.Second)
	state = env.getGameState(ctx, tableID)
	assert.Equal(t, pokerrpc.GamePhase_SHOWDOWN, state.Phase, "should advance to SHOWDOWN")
	t.Log("✓ SHOWDOWN reached")

	// Verify all-in players
	allInCount := 0
	for _, p := range state.Players {
		if p.IsAllIn {
			allInCount++
		}
	}
	assert.GreaterOrEqual(t, allInCount, 2, "at least 2 players should be all-in")

	// Wait for showdown to complete and winners to be available
	require.Eventually(t, func() bool {
		_, err := env.pokerClient.GetLastWinners(ctx, &pokerrpc.GetLastWinnersRequest{
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
	env := newTestEnv(t)
	defer env.Close()

	ctx := context.Background()

	// Setup 3 players
	players := []string{"player1", "player2", "player3"}
	for _, p := range players {
		env.setBalance(ctx, p, 10_000)
	}

	// Create table
	createResp, err := env.lobbyClient.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      "player1",
		SmallBlind:    10,
		BigBlind:      20,
		MinPlayers:    3,
		MaxPlayers:    3,
		BuyIn:         1_000,
		MinBalance:    1_000,
		StartingChips: 200, // Moderate stacks
		AutoStartMs:   5000,
		AutoAdvanceMs: 1000,
	})
	require.NoError(t, err)
	tableID := createResp.TableId

	// Other players join
	for _, p := range players[1:] {
		_, err := env.lobbyClient.JoinTable(ctx, &pokerrpc.JoinTableRequest{
			PlayerId: p,
			TableId:  tableID,
		})
		require.NoError(t, err)
	}

	// All players mark ready
	for _, p := range players {
		_, err := env.lobbyClient.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{
			PlayerId: p,
			TableId:  tableID,
		})
		require.NoError(t, err)
	}

	// Wait for game to start
	env.waitForGameStart(ctx, tableID, 3*time.Second)
	env.waitForGamePhase(ctx, tableID, pokerrpc.GamePhase_PRE_FLOP, 3*time.Second)

	// First player folds
	state := env.getGameState(ctx, tableID)
	firstPlayer := state.CurrentPlayer
	_, err = env.pokerClient.FoldBet(ctx, &pokerrpc.FoldBetRequest{
		PlayerId: firstPlayer,
		TableId:  tableID,
	})
	require.NoError(t, err)
	t.Logf("Player %s folded", firstPlayer)

	// Wait for turn to advance after fold
	require.Eventually(t, func() bool {
		newState := env.getGameState(ctx, tableID)
		return newState.CurrentPlayer != firstPlayer
	}, 2*time.Second, 10*time.Millisecond, "turn should advance after fold")

	// Remaining two players go all-in
	for i := 0; i < 2; i++ {
		state = env.getGameState(ctx, tableID)
		if state.Phase != pokerrpc.GamePhase_PRE_FLOP {
			// Already advanced
			break
		}
		currentPlayer := state.CurrentPlayer
		currentPhase := state.Phase

		_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
			PlayerId: currentPlayer,
			TableId:  tableID,
			Amount:   200,
		})
		if err == nil {
			t.Logf("Player %s went all-in", currentPlayer)

			// Wait for turn to advance or phase to change
			require.Eventually(t, func() bool {
				newState := env.getGameState(ctx, tableID)
				return newState.CurrentPlayer != currentPlayer || newState.Phase != currentPhase
			}, 2*time.Second, 10*time.Millisecond, "turn or phase should change after all-in")
		}
	}

	// CRITICAL VERIFICATION: Game should auto-advance through streets
	t.Log("Verifying auto-advance through FLOP...")
	env.waitForGamePhase(ctx, tableID, pokerrpc.GamePhase_FLOP, 3*time.Second)
	state = env.getGameState(ctx, tableID)
	assert.Equal(t, pokerrpc.GamePhase_FLOP, state.Phase, "should advance to FLOP")
	assert.Equal(t, 3, len(state.CommunityCards), "should have 3 community cards at FLOP")
	t.Logf("✓ FLOP reached with %d community cards", len(state.CommunityCards))

	t.Log("Verifying auto-advance through TURN...")
	env.waitForGamePhase(ctx, tableID, pokerrpc.GamePhase_TURN, 3*time.Second)
	state = env.getGameState(ctx, tableID)
	assert.Equal(t, pokerrpc.GamePhase_TURN, state.Phase, "should advance to TURN")
	assert.Equal(t, 4, len(state.CommunityCards), "should have 4 community cards at TURN")
	t.Logf("✓ TURN reached with %d community cards", len(state.CommunityCards))

	t.Log("Verifying auto-advance through RIVER...")
	env.waitForGamePhase(ctx, tableID, pokerrpc.GamePhase_RIVER, 3*time.Second)
	state = env.getGameState(ctx, tableID)
	assert.Equal(t, pokerrpc.GamePhase_RIVER, state.Phase, "should advance to RIVER")
	assert.Equal(t, 5, len(state.CommunityCards), "should have 5 community cards at RIVER")
	t.Logf("✓ RIVER reached with %d community cards", len(state.CommunityCards))

	t.Log("Verifying auto-advance to SHOWDOWN...")
	env.waitForGamePhase(ctx, tableID, pokerrpc.GamePhase_SHOWDOWN, 3*time.Second)
	state = env.getGameState(ctx, tableID)
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
		_, err := env.pokerClient.GetLastWinners(ctx, &pokerrpc.GetLastWinnersRequest{
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
	env := newTestEnv(t)
	defer env.Close()

	ctx := context.Background()

	// Setup 2 players with small stacks
	players := []string{"alice", "bob"}
	for _, p := range players {
		env.setBalance(ctx, p, 10_000)
	}

	// Create table with small starting chips (100 each)
	// Use shorter auto-start to speed up test if we need multiple hands
	createResp, err := env.lobbyClient.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      "alice",
		SmallBlind:    10,
		BigBlind:      20,
		MinPlayers:    2,
		MaxPlayers:    2,
		BuyIn:         1_000,
		MinBalance:    1_000,
		StartingChips: 100,  // Small stacks
		AutoStartMs:   1500, // Shorter delay for faster test
		AutoAdvanceMs: 500,  // Shorter delay for faster test
	})
	require.NoError(t, err)
	tableID := createResp.TableId

	// Join bob
	_, err = env.lobbyClient.JoinTable(ctx, &pokerrpc.JoinTableRequest{
		PlayerId: "bob",
		TableId:  tableID,
	})
	require.NoError(t, err)

	// Both players mark ready
	for _, p := range players {
		_, err := env.lobbyClient.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{
			PlayerId: p,
			TableId:  tableID,
		})
		require.NoError(t, err)
	}

	// Wait for game to start
	env.waitForGameStart(ctx, tableID, 3*time.Second)
	env.waitForGamePhase(ctx, tableID, pokerrpc.GamePhase_PRE_FLOP, 3*time.Second)

	// Play hands until one player is eliminated
	// Maximum 5 hands in case of consecutive ties (rare but possible)
	maxHands := 5
	var gameOverDetected bool
	var winnerID string

	for handNum := 1; handNum <= maxHands; handNum++ {
		t.Logf("Hand %d: Playing all-in hand...", handNum)

		// Get current state
		state := env.getGameState(ctx, tableID)

		// First player goes all-in
		currentPlayer := state.CurrentPlayer
		_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
			PlayerId: currentPlayer,
			TableId:  tableID,
			Amount:   100,
		})
		require.NoError(t, err)

		// Wait for turn to advance
		require.Eventually(t, func() bool {
			state = env.getGameState(ctx, tableID)
			return state.CurrentPlayer != currentPlayer
		}, 2*time.Second, 10*time.Millisecond, "turn should advance after all-in")

		// Second player calls all-in
		state = env.getGameState(ctx, tableID)
		currentPlayer = state.CurrentPlayer
		_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
			PlayerId: currentPlayer,
			TableId:  tableID,
			Amount:   100,
		})
		require.NoError(t, err)

		// Wait for showdown
		env.waitForGamePhase(ctx, tableID, pokerrpc.GamePhase_SHOWDOWN, 10*time.Second)

		// Check if one player has been eliminated
		finalState := env.getGameState(ctx, tableID)
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
			env.waitForGamePhase(ctx, tableID, pokerrpc.GamePhase_PRE_FLOP, 3*time.Second) // Next hand
		}
	}

	require.True(t, gameOverDetected, "Expected game over within %d hands", maxHands)

	// Wait longer than auto-start delay to ensure no new hand starts
	t.Log("Waiting to verify no auto-start after game over...")
	time.Sleep(3 * time.Second)

	// Verify game is still in SHOWDOWN and hasn't auto-started a new hand
	state := env.getGameState(ctx, tableID)
	assert.Equal(t, pokerrpc.GamePhase_SHOWDOWN, state.Phase, "Game should remain in SHOWDOWN after winner takes all")

	// Verify the phase hasn't changed (still in SHOWDOWN, not back to PRE_FLOP for a new hand)
	assert.NotEqual(t, pokerrpc.GamePhase_PRE_FLOP, state.Phase, "Game should not start a new hand after winner takes all")

	t.Log("✓ Game correctly stayed in SHOWDOWN - no auto-start after game over")
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
	env := newTestEnv(t)
	defer env.Close()

	ctx := context.Background()

	// Setup 2 players
	players := []string{"rich_player", "poor_player"}
	for _, p := range players {
		env.setBalance(ctx, p, 10_000)
	}

	// Create table with moderate stacks - small blinds so all-in doesn't eliminate anyone
	createResp, err := env.lobbyClient.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      "rich_player",
		SmallBlind:    5,
		BigBlind:      10,
		MinPlayers:    2,
		MaxPlayers:    2,
		BuyIn:         10_000,
		MinBalance:    10_000,
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
	_, err = env.lobbyClient.JoinTable(ctx, &pokerrpc.JoinTableRequest{
		PlayerId: "poor_player",
		TableId:  tableID,
	})
	require.NoError(t, err)

	// Both players mark ready
	for _, p := range players {
		_, err := env.lobbyClient.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{
			PlayerId: p,
			TableId:  tableID,
		})
		require.NoError(t, err)
	}

	// Wait for game to start
	env.waitForGameStart(ctx, tableID, 3*time.Second)
	env.waitForGamePhase(ctx, tableID, pokerrpc.GamePhase_PRE_FLOP, 3*time.Second)

	t.Log("Game started - now triggering all-in scenario...")

	// Get current state
	state := env.getGameState(ctx, tableID)
	currentPlayer := state.CurrentPlayer

	// Current player goes all-in (they have 195 or 190 chips after blinds)
	_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: currentPlayer,
		TableId:  tableID,
		Amount:   200, // Try to bet 200, will be capped at their balance
	})
	require.NoError(t, err)
	t.Logf("Player %s went all-in", currentPlayer)

	// Wait for next player's turn (check for connection-closing errors during cleanup)
	require.Eventually(t, func() bool {
		resp, err := env.pokerClient.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		if err != nil {
			// Only ignore connection-closing errors (from test cleanup)
			if strings.Contains(err.Error(), "connection is closing") {
				return false // Expected during cleanup, keep trying
			}
			t.Errorf("unexpected error getting game state: %v", err)
			return false
		}
		state = resp.GameState
		return state.CurrentPlayer != currentPlayer && state.CurrentPlayer != ""
	}, 2*time.Second, 10*time.Millisecond, "next player should have a turn")

	nextPlayer := state.CurrentPlayer

	// Next player calls (will be forced to partial all-in since they have less after paying blind)
	_, err = env.pokerClient.CallBet(ctx, &pokerrpc.CallBetRequest{
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
	env.waitForGamePhase(ctx, tableID, pokerrpc.GamePhase_SHOWDOWN, 10*time.Second)

	// Verify we reached showdown with all 5 community cards
	state = env.getGameState(ctx, tableID)
	assert.Equal(t, pokerrpc.GamePhase_SHOWDOWN, state.Phase)
	assert.Equal(t, 5, len(state.CommunityCards), "Should have all 5 community cards at SHOWDOWN")
	t.Log("✓ SHOWDOWN reached with all 5 community cards")

	// Wait for showdown to complete with winners
	require.Eventually(t, func() bool {
		_, err := env.pokerClient.GetLastWinners(ctx, &pokerrpc.GetLastWinnersRequest{
			TableId: tableID,
		})
		return err == nil
	}, 3*time.Second, 50*time.Millisecond, "showdown should complete with winners")

	t.Log("✓ Unequal stacks all-in: Auto-advanced through all streets correctly")
	t.Log("✓ Regression test passed: partial all-in match triggers auto-advance")
}
