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

// Reproduces a restore bug: after completing SHOWDOWN, the server restarts and
// StartGameStream should restore the game at SHOWDOWN. Current behavior demotes
// the phase back to RIVER because restore derives phase from board size (5 cards)
// instead of trusting the snapshot phase.
func TestReconnectRestore_ShowdownPhasePreserved(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "poker.sqlite")

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

	// Boot first server
	b1 := start(t)
	defer stop(b1)

	// Seed wallet balances
	setBalance := func(pid string, want int64) {
		rb, _ := b1.lc.GetBalance(ctx, &pokerrpc.GetBalanceRequest{PlayerId: pid})
		var cur int64
		if rb != nil {
			cur = rb.Balance
		}
		delta := want - cur
		if delta != 0 {
			_, err := b1.lc.UpdateBalance(ctx, &pokerrpc.UpdateBalanceRequest{PlayerId: pid, Amount: delta, Description: "seed"})
			require.NoError(t, err)
		}
	}
	p1, p2 := "p1", "p2"
	setBalance(p1, 10_000)
	setBalance(p2, 10_000)

	// Create table and join second player
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

	// Ready both players
	for _, pid := range []string{p1, p2} {
		_, err := b1.lc.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{PlayerId: pid, TableId: tableID})
		require.NoError(t, err)
	}

	waitPhase := func(phase pokerrpc.GamePhase) {
		require.Eventually(t, func() bool {
			st, err := b1.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
			return err == nil && st.GameState.GetPhase() == phase
		}, 3*time.Second, 25*time.Millisecond)
	}

	// Reach FLOP: call + check pre-flop
	waitPhase(pokerrpc.GamePhase_PRE_FLOP)
	st, err := b1.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	require.NoError(t, err)
	cur := st.GameState.GetCurrentPlayer()
	_, err = b1.pc.CallBet(ctx, &pokerrpc.CallBetRequest{PlayerId: cur, TableId: tableID})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		st, _ := b1.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		return st.GameState.GetCurrentPlayer() != cur
	}, 2*time.Second, 25*time.Millisecond)
	st, _ = b1.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	next := st.GameState.GetCurrentPlayer()
	_, err = b1.pc.CheckBet(ctx, &pokerrpc.CheckBetRequest{PlayerId: next, TableId: tableID})
	require.NoError(t, err)
	waitPhase(pokerrpc.GamePhase_FLOP)

	// FLOP: both check → TURN
	st, _ = b1.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	cur = st.GameState.GetCurrentPlayer()
	_, err = b1.pc.CheckBet(ctx, &pokerrpc.CheckBetRequest{PlayerId: cur, TableId: tableID})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		st, _ := b1.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		return st.GameState.GetCurrentPlayer() != cur
	}, 2*time.Second, 25*time.Millisecond)
	st, _ = b1.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	next = st.GameState.GetCurrentPlayer()
	_, err = b1.pc.CheckBet(ctx, &pokerrpc.CheckBetRequest{PlayerId: next, TableId: tableID})
	require.NoError(t, err)
	waitPhase(pokerrpc.GamePhase_TURN)

	// TURN: both check → RIVER
	st, _ = b1.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	cur = st.GameState.GetCurrentPlayer()
	_, err = b1.pc.CheckBet(ctx, &pokerrpc.CheckBetRequest{PlayerId: cur, TableId: tableID})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		st, _ := b1.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		return st.GameState.GetCurrentPlayer() != cur
	}, 2*time.Second, 25*time.Millisecond)
	st, _ = b1.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	next = st.GameState.GetCurrentPlayer()
	_, err = b1.pc.CheckBet(ctx, &pokerrpc.CheckBetRequest{PlayerId: next, TableId: tableID})
	require.NoError(t, err)
	waitPhase(pokerrpc.GamePhase_RIVER)

	// RIVER: both check → SHOWDOWN (avoid racing the second action after phase change)
	st, _ = b1.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	cur = st.GameState.GetCurrentPlayer()
	_, err = b1.pc.CheckBet(ctx, &pokerrpc.CheckBetRequest{PlayerId: cur, TableId: tableID})
	require.NoError(t, err)

	// Determine the other player's ID
	other := p1
	if cur == p1 {
		other = p2
	}

	// Wait for either other's turn on RIVER or SHOWDOWN reached already
	require.Eventually(t, func() bool {
		st, err := b1.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		if err != nil || st == nil || st.GameState == nil {
			return false
		}
		ph := st.GameState.GetPhase()
		if ph == pokerrpc.GamePhase_SHOWDOWN {
			return true
		}
		return ph == pokerrpc.GamePhase_RIVER && st.GameState.GetCurrentPlayer() == other
	}, 2*time.Second, 25*time.Millisecond)
	st, _ = b1.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	if st.GameState.GetPhase() == pokerrpc.GamePhase_RIVER {
		_, err = b1.pc.CheckBet(ctx, &pokerrpc.CheckBetRequest{PlayerId: other, TableId: tableID})
		require.NoError(t, err)
	}

	// Hand should be at SHOWDOWN
	require.Eventually(t, func() bool {
		st, _ := b1.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		return st.GameState.GetPhase() == pokerrpc.GamePhase_SHOWDOWN
	}, 3*time.Second, 25*time.Millisecond)

	// Simulate server restart
	stop(b1)
	b2 := start(t)
	defer stop(b2)

	// Attach streams to trigger restore
	s1, err := b2.pc.StartGameStream(ctx, &pokerrpc.StartGameStreamRequest{TableId: tableID, PlayerId: p1})
	require.NoError(t, err)
	if closer, ok := interface{}(s1).(interface{ CloseSend() error }); ok {
		defer closer.CloseSend()
	}
	_, _ = s1.Recv()
	s2, err := b2.pc.StartGameStream(ctx, &pokerrpc.StartGameStreamRequest{TableId: tableID, PlayerId: p2})
	require.NoError(t, err)
	if closer, ok := interface{}(s2).(interface{ CloseSend() error }); ok {
		defer closer.CloseSend()
	}
	_, _ = s2.Recv()

	// Restored phase must remain SHOWDOWN and there must be no current player
	st2, err := b2.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	require.NoError(t, err)
	require.Equal(t, pokerrpc.GamePhase_SHOWDOWN, st2.GameState.GetPhase(), "restored phase should remain SHOWDOWN, not regress")
	require.Equal(t, "", st2.GameState.GetCurrentPlayer(), "no current player at SHOWDOWN")

	// Any action at SHOWDOWN must be rejected and phase remain unchanged
	_, err = b2.pc.CheckBet(ctx, &pokerrpc.CheckBetRequest{PlayerId: p1, TableId: tableID})
	require.Error(t, err, "actions must be rejected at SHOWDOWN")
	st3, _ := b2.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	require.Equal(t, pokerrpc.GamePhase_SHOWDOWN, st3.GameState.GetPhase())
}
