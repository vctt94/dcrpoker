package e2e

import (
	"context"
	"net"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"github.com/vctt94/bisonbotkit/logging"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	"github.com/vctt94/pokerbisonrelay/pkg/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TestPotRestoration_AfterReconnect verifies that after a server restart and reconnect,
// the pot amount is correctly restored and distributed to the winner during showdown.
//
// This test reproduces the bug where pot=0 is restored from snapshot, causing
// "0 winners" in showdown despite correct hand evaluation.
func TestPotRestoration_AfterReconnect(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// 1) Create temp DB path we can reuse across restarts
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "poker.sqlite")

	// Helper to boot a server + client pair on a given DB path
	type boot struct {
		db   server.Database
		srv  *server.Server
		grpc *grpc.Server
		conn *grpc.ClientConn
		lc   pokerrpc.LobbyServiceClient
		pc   pokerrpc.PokerServiceClient
	}
	start := func(t *testing.T) *boot {
		db, err := server.NewDatabase(dbPath)
		require.NoError(t, err)

		lb, _ := logging.NewLogBackend(logging.LogConfig{DebugLevel: "debug"})
		srv := server.NewServer(db, lb)

		lis, err := net.Listen("tcp", ":0")
		require.NoError(t, err)
		gs := grpc.NewServer()
		pokerrpc.RegisterLobbyServiceServer(gs, srv)
		pokerrpc.RegisterPokerServiceServer(gs, srv)
		go func() { _ = gs.Serve(lis) }()

		conn, err := grpc.Dial(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)

		return &boot{
			db:   db,
			srv:  srv,
			grpc: gs,
			conn: conn,
			lc:   pokerrpc.NewLobbyServiceClient(conn),
			pc:   pokerrpc.NewPokerServiceClient(conn),
		}
	}
	stop := func(b *boot) {
		if b == nil {
			return
		}
		if b.conn != nil {
			_ = b.conn.Close()
		}
		if b.srv != nil {
			b.srv.Stop()
		}
		if b.grpc != nil {
			b.grpc.Stop()
		}
		if b.db != nil {
			_ = b.db.Close()
		}
	}

	// 2) Boot first server
	b1 := start(t)
	defer stop(b1)

	// Seed balances
	setBalance := func(lc pokerrpc.LobbyServiceClient, pid string, want int64) {
		rb, _ := lc.GetBalance(ctx, &pokerrpc.GetBalanceRequest{PlayerId: pid})
		var cur int64
		if rb != nil {
			cur = rb.Balance
		}
		delta := want - cur
		if delta != 0 {
			_, err := lc.UpdateBalance(ctx, &pokerrpc.UpdateBalanceRequest{PlayerId: pid, Amount: delta, Description: "seed"})
			require.NoError(t, err)
		}
	}
	p1, p2 := "p1", "p2"
	setBalance(b1.lc, p1, 10_000)
	setBalance(b1.lc, p2, 10_000)

	// Create table and join both
	createResp, err := b1.lc.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      p1,
		SmallBlind:    10,
		BigBlind:      20,
		MinPlayers:    2,
		MaxPlayers:    2,
		BuyIn:         1_000,
		MinBalance:    1_000,
		StartingChips: 1_000,
		AutoAdvanceMs: 1_000,
	})
	require.NoError(t, err)
	tableID := createResp.TableId

	_, err = b1.lc.JoinTable(ctx, &pokerrpc.JoinTableRequest{PlayerId: p2, TableId: tableID})
	require.NoError(t, err)

	// Ready up both
	for _, pid := range []string{p1, p2} {
		_, err := b1.lc.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{PlayerId: pid, TableId: tableID})
		require.NoError(t, err)
	}

	// Wait for PRE_FLOP
	waitPhase := func(pc pokerrpc.PokerServiceClient, phase pokerrpc.GamePhase) {
		require.Eventually(t, func() bool {
			st, err := pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
			return err == nil && st.GameState.GetPhase() == phase
		}, 3*time.Second, 25*time.Millisecond)
	}
	waitPhase(b1.pc, pokerrpc.GamePhase_PRE_FLOP)

	// Pre-flop: first player calls big blind, second player checks to build a pot
	st, err := b1.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	require.NoError(t, err)
	cur := st.GameState.GetCurrentPlayer()
	_, err = b1.pc.CallBet(ctx, &pokerrpc.CallBetRequest{PlayerId: cur, TableId: tableID})
	require.NoError(t, err)

	// Next player checks (no bet to call)
	require.Eventually(t, func() bool {
		st, _ := b1.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		return st.GameState.GetCurrentPlayer() != cur
	}, 2*time.Second, 25*time.Millisecond)
	st, _ = b1.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	next := st.GameState.GetCurrentPlayer()
	_, err = b1.pc.CheckBet(ctx, &pokerrpc.CheckBetRequest{PlayerId: next, TableId: tableID})
	require.NoError(t, err)

	// We should now be on FLOP
	waitPhase(b1.pc, pokerrpc.GamePhase_FLOP)

	// FLOP: both players check to advance to TURN
	st, err = b1.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	require.NoError(t, err)
	cur = st.GameState.GetCurrentPlayer()
	_, err = b1.pc.CheckBet(ctx, &pokerrpc.CheckBetRequest{PlayerId: cur, TableId: tableID})
	require.NoError(t, err)

	// Next player checks
	require.Eventually(t, func() bool {
		st, _ := b1.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		return st.GameState.GetCurrentPlayer() != cur
	}, 2*time.Second, 25*time.Millisecond)
	st, _ = b1.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	next = st.GameState.GetCurrentPlayer()
	_, err = b1.pc.CheckBet(ctx, &pokerrpc.CheckBetRequest{PlayerId: next, TableId: tableID})
	require.NoError(t, err)

	// We should now be on TURN
	waitPhase(b1.pc, pokerrpc.GamePhase_TURN)

	// 3) Simulate server restart at TURN phase (like in the logs)
	stop(b1)
	// Boot second server on same DB
	b2 := start(t)
	defer stop(b2)

	// Attach game streams (reconnect) to trigger restore in production code.
	s1, err := b2.pc.StartGameStream(ctx, &pokerrpc.StartGameStreamRequest{TableId: tableID, PlayerId: p1})
	require.NoError(t, err)
	if closer, ok := interface{}(s1).(interface{ CloseSend() error }); ok {
		defer closer.CloseSend()
	}
	// Read the initial snapshot to ensure the stream is established
	if _, err := s1.Recv(); err == nil {
		// ok
	}
	s2, err := b2.pc.StartGameStream(ctx, &pokerrpc.StartGameStreamRequest{TableId: tableID, PlayerId: p2})
	require.NoError(t, err)
	if closer, ok := interface{}(s2).(interface{ CloseSend() error }); ok {
		defer closer.CloseSend()
	}
	if _, err := s2.Recv(); err == nil {
		// ok
	}

	// Wait until restored game shows TURN after reconnect
	waitPhase(b2.pc, pokerrpc.GamePhase_TURN)

	// 4) After reconnect: both players check through TURN and RIVER to reach showdown
	st, err = b2.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	require.NoError(t, err)
	cur = st.GameState.GetCurrentPlayer()
	_, err = b2.pc.CheckBet(ctx, &pokerrpc.CheckBetRequest{PlayerId: cur, TableId: tableID})
	require.NoError(t, err)

	// Next player checks
	require.Eventually(t, func() bool {
		st, err := b2.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		if err != nil || st == nil || st.GameState == nil {
			return false
		}
		return st.GameState.GetCurrentPlayer() != cur
	}, 2*time.Second, 25*time.Millisecond)
	st, _ = b2.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	next = st.GameState.GetCurrentPlayer()
	_, err = b2.pc.CheckBet(ctx, &pokerrpc.CheckBetRequest{PlayerId: next, TableId: tableID})
	require.NoError(t, err)

	// We should now be on RIVER
	waitPhase(b2.pc, pokerrpc.GamePhase_RIVER)

	// RIVER: both players check to reach showdown. Avoid racing phase change by
	// precomputing the other player's ID and waiting for turn ownership while
	// still on RIVER. If we've already advanced to SHOWDOWN, skip sending a
	// second action.
	st, err = b2.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	require.NoError(t, err)
	cur = st.GameState.GetCurrentPlayer()
	_, err = b2.pc.CheckBet(ctx, &pokerrpc.CheckBetRequest{PlayerId: cur, TableId: tableID})
	require.NoError(t, err)

	// Determine the other player's ID (heads-up)
	other := p1
	if cur == p1 {
		other = p2
	}

	// Wait until either it's the other player's turn on RIVER, or we've
	// already advanced to SHOWDOWN (in which case no second action is needed).
	require.Eventually(t, func() bool {
		st, err := b2.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		if err != nil || st == nil || st.GameState == nil {
			return false
		}
		ph := st.GameState.GetPhase()
		if ph == pokerrpc.GamePhase_SHOWDOWN {
			return true
		}
		return ph == pokerrpc.GamePhase_RIVER && st.GameState.GetCurrentPlayer() == other
	}, 2*time.Second, 25*time.Millisecond)

	st, _ = b2.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	if st.GameState.GetPhase() == pokerrpc.GamePhase_RIVER {
		_, err = b2.pc.CheckBet(ctx, &pokerrpc.CheckBetRequest{PlayerId: other, TableId: tableID})
		require.NoError(t, err)
	}

	// 5) Wait for showdown to complete
	require.Eventually(t, func() bool {
		st, err := b2.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		return err == nil && st.GameState.GetPhase() == pokerrpc.GamePhase_SHOWDOWN
	}, 3*time.Second, 25*time.Millisecond)

	// 6) Verify that the pot was correctly distributed (in chips)
	winners, err := b2.pc.GetLastWinners(ctx, &pokerrpc.GetLastWinnersRequest{TableId: tableID})
	require.NoError(t, err)
	t.Logf("Showdown winners: %+v", winners)

	// At least one winner should be present and have non-zero winnings
	require.GreaterOrEqual(t, len(winners.Winners), 1)
	nonZero := false
	for _, w := range winners.Winners {
		if w.Winnings > 0 {
			nonZero = true
			break
		}
	}
	require.True(t, nonZero, "expected non-zero pot to be distributed to at least one winner")
}
