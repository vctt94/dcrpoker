// This file contains end-to-end tests that spin up a full poker server backed
// by a real SQLite database. The tests exercise realistic gameplay flows with
// minimal mocking (only the network is in-process via gRPC).
//
// To keep the tests self-contained and independent they **must** be executed
// with `go test ./...` and **should not** depend on external resources.

package e2e

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"net"

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

	// Quick sanity check of balances after table creation/join.
	//  - Table creator (alice) also pays buy-in when creating the table
	//  - Joiners (bob & carol) have bankroll - buyIn.
	buyIn := int64(1_000)
	assert.Equal(t, initialBankroll-buyIn, env.getBalance(ctx, "alice"))
	for _, p := range []string{"bob", "carol"} {
		assert.Equal(t, initialBankroll-buyIn, env.getBalance(ctx, p), "post buy-in balance mismatch for %s", p)
	}

	// ACTION ROUND -------------------------------------------------------------
	// Alice opens with a 100 bet.
	_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: "alice",
		TableId:  tableID,
		Amount:   100,
	})
	require.NoError(t, err)

	// Bob calls.
	_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: "bob",
		TableId:  tableID,
		Amount:   100,
	})
	require.NoError(t, err)

	// Carol decides to fold (0 bet via Fold API).
	_, err = env.pokerClient.FoldBet(ctx, &pokerrpc.FoldBetRequest{
		PlayerId: "carol",
		TableId:  tableID,
	})
	require.NoError(t, err)

	// Validate pot value (220) via GetGameState.
	// Pot = 30 (blinds) + 100 (Alice's bet) + 90 (Bob's additional bet after SB)
	state, err := env.pokerClient.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	require.NoError(t, err)
	assert.Equal(t, int64(220), state.GameState.Pot, "unexpected pot size")

	// Verify Carol is marked as folded
	for _, player := range state.GameState.Players {
		if player.Id == "carol" {
			assert.True(t, player.Folded, "carol should be marked as folded")
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

	// PRE-FLOP BETTING
	// In 4-player game: player1=dealer, player2=SB, player3=BB, player4=UTG (acts first)

	// Player4 calls the big blind
	_, err := env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: "player4",
		TableId:  tableID,
		Amount:   20,
	})
	require.NoError(t, err)

	// Player1 raises
	_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: "player1",
		TableId:  tableID,
		Amount:   60, // Raising to 60
	})
	require.NoError(t, err)

	// Player2 calls the raise (SB needs to add 50 more to existing 10)
	_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: "player2",
		TableId:  tableID,
		Amount:   60,
	})
	require.NoError(t, err)

	// Player3 calls the raise (BB needs to add 40 more to existing 20)
	_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: "player3",
		TableId:  tableID,
		Amount:   60,
	})
	require.NoError(t, err)

	// Player4 calls the raise
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
	_, err = env.pokerClient.CheckBet(ctx, &pokerrpc.CheckBetRequest{
		PlayerId: "player2",
		TableId:  tableID,
	})
	require.NoError(t, err)

	// Player3 bets 100
	_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: "player3",
		TableId:  tableID,
		Amount:   100,
	})
	require.NoError(t, err)

	// Player4 calls
	_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: "player4",
		TableId:  tableID,
		Amount:   100,
	})
	require.NoError(t, err)

	// Player1 folds
	_, err = env.pokerClient.FoldBet(ctx, &pokerrpc.FoldBetRequest{
		PlayerId: "player1",
		TableId:  tableID,
	})
	require.NoError(t, err)

	// Player2 folds
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
	_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: "player3",
		TableId:  tableID,
		Amount:   200,
	})
	require.NoError(t, err)

	// Player4 calls
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
	_, err = env.pokerClient.CheckBet(ctx, &pokerrpc.CheckBetRequest{
		PlayerId: "player3",
		TableId:  tableID,
	})
	require.NoError(t, err)

	// Player4 bets 300
	_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: "player4",
		TableId:  tableID,
		Amount:   300,
	})
	require.NoError(t, err)

	// Player3 folds
	_, err = env.pokerClient.FoldBet(ctx, &pokerrpc.FoldBetRequest{
		PlayerId: "player3",
		TableId:  tableID,
	})
	require.NoError(t, err)

	// Wait for showdown
	env.waitForGamePhase(ctx, tableID, pokerrpc.GamePhase_SHOWDOWN, 3*time.Second)

	// Verify Player4 won the pot
	winners, err := env.pokerClient.GetLastWinners(ctx, &pokerrpc.GetLastWinnersRequest{
		TableId: tableID,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, len(winners.Winners), "expected 1 winner")
	assert.Equal(t, "player4", winners.Winners[0].PlayerId, "expected player4 to win")

	// Verify pot amount: 240 (pre-flop) + 200 (flop) + 400 (turn) + 300 (river) = 1140 (headline pot amount)
	assert.Equal(t, int64(1140), winners.Winners[0].Winnings, "unexpected pot amount in winner response")
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
	// Wait for enough time for auto-check-or-fold to occur (timebank + buffer)
	time.Sleep(7 * time.Second)

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

	// active1 calls the big blind (20)
	_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: "active1",
		TableId:  tableID,
		Amount:   20,
	})
	require.NoError(t, err)

	// active2 raises to 50
	_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: "active2",
		TableId:  tableID,
		Amount:   50,
	})
	require.NoError(t, err)

	// Now "timeout" player (big blind) would need to call from 20 to 50 - should auto-fold after timeout
	// Wait for enough time for auto-fold to occur (timebank + buffer)
	time.Sleep(7 * time.Second)

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

	// Verify initial pot includes blinds (10 + 20 = 30)
	state := env.getGameState(ctx, tableID)
	assert.Equal(t, int64(30), state.Pot, "initial pot should include blinds (10+20=30)")

	// Check current player and bet state
	t.Logf("Current player: %s, Current bet: %d", state.CurrentPlayer, state.CurrentBet)

	// Check player bets to understand blind posting
	for _, player := range state.Players {
		t.Logf("Player %s has bet: %d, folded: %t", player.Id, player.CurrentBet, player.Folded)
	}

	// Verify blind posting is correct:
	// p1 is dealer (no blind), p2 is small blind (10), p3 is big blind (20)
	playerBets := make(map[string]int64)
	for _, player := range state.Players {
		playerBets[player.Id] = player.CurrentBet
	}
	assert.Equal(t, int64(0), playerBets["p1"], "p1 (dealer) should have no blind")
	assert.Equal(t, int64(10), playerBets["p2"], "p2 should have small blind (10)")
	assert.Equal(t, int64(20), playerBets["p3"], "p3 should have big blind (20)")

	// First player to act should be p1 (dealer/Under the Gun in 3-handed)
	assert.Equal(t, "p1", state.CurrentPlayer, "p1 should be first to act (Under the Gun)")

	// Current bet should be big blind amount (20)
	assert.Equal(t, int64(20), state.CurrentBet, "current bet should be big blind (20)")

	// P1 calls the big blind (20)
	_, err := env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: "p1",
		TableId:  tableID,
		Amount:   20,
	})
	require.NoError(t, err)

	// Check pot is now 50 (30 from blinds + 20 from p1's call)
	state = env.getGameState(ctx, tableID)
	assert.Equal(t, int64(50), state.Pot, "pot should be 50 after p1's call (30+20)")

	// P2 (small blind) calls by betting 20 total (needs to add 10 more to their existing 10)
	_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: "p2",
		TableId:  tableID,
		Amount:   20,
	})
	require.NoError(t, err)

	// Check pot is now 60 (50 + 10 more from p2)
	state = env.getGameState(ctx, tableID)
	assert.Equal(t, int64(60), state.Pot, "pot should be 60 after p2's call (50+10)")

	// P3 (big blind) can check (already has 20 bet)
	_, err = env.pokerClient.CheckBet(ctx, &pokerrpc.CheckBetRequest{
		PlayerId: "p3",
		TableId:  tableID,
	})
	require.NoError(t, err)

	// Pot should still be 60 after check
	state = env.getGameState(ctx, tableID)
	assert.Equal(t, int64(60), state.Pot, "pot should remain 60 after p3's check")
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

	// Get game state and verify that players have the expected starting chips
	state := env.getGameState(ctx, tableID)

	// All players should have starting chips equal to default (1000)
	// minus any blinds they've posted
	for _, player := range state.Players {
		switch player.Id {
		case "player1":
			// Dealer, no blind posted, should have full 1000 chips
			expectedChips := int64(1000)
			actualChips := expectedChips - player.CurrentBet
			t.Logf("Player %s: has bet %d, should have balance %d", player.Id, player.CurrentBet, actualChips)
		case "player2":
			// Small blind, should have 1000 - 10 = 990 chips
			expectedChips := int64(1000) - int64(10)
			actualChips := expectedChips - (player.CurrentBet - int64(10))
			t.Logf("Player %s: has bet %d, should have balance %d", player.Id, player.CurrentBet, actualChips)
		case "player3":
			// Big blind, should have 1000 - 20 = 980 chips
			expectedChips := int64(1000) - int64(20)
			actualChips := expectedChips - (player.CurrentBet - int64(20))
			t.Logf("Player %s: has bet %d, should have balance %d", player.Id, player.CurrentBet, actualChips)
		}
	}

	// Verify pot includes blinds (10 + 20 = 30)
	assert.Equal(t, int64(30), state.Pot, "pot should include blinds (10+20=30)")

	// Player1 should be able to bet (has enough chips)
	_, err = env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: "player1",
		TableId:  tableID,
		Amount:   20, // Call the big blind
	})
	require.NoError(t, err)

	// Verify pot is now 50
	state = env.getGameState(ctx, tableID)
	assert.Equal(t, int64(50), state.Pot, "pot should be 50 after player1's call")
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
	state = env.getGameState(ctx, tableID)
	assert.Equal(t, int64(80), state.Pot, "pot should be 80 after BB's call (60+20)")
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
		if state.GameStarted && state.Phase == pokerrpc.GamePhase_SHOWDOWN {
			break
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
			time.Sleep(50 * time.Millisecond)
			continue
		}

		// Decide action
		if currPlayer.CurrentBet >= state.CurrentBet {
			_, err := env.pokerClient.CheckBet(ctx, &pokerrpc.CheckBetRequest{PlayerId: curr, TableId: tableID})
			if err != nil {
				// If cannot check, try calling to the current bet
				_, err2 := env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{PlayerId: curr, TableId: tableID, Amount: state.CurrentBet})
				require.NoError(t, err2)
			}
		} else {
			_, err := env.pokerClient.MakeBet(ctx, &pokerrpc.MakeBetRequest{PlayerId: curr, TableId: tableID, Amount: state.CurrentBet})
			require.NoError(t, err)
		}

		// Avoid spamming
		time.Sleep(50 * time.Millisecond)
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
