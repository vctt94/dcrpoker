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

// Ensures restoring at TURN preserves a non-zero pot and a valid current player,
// and that out-of-turn actions are rejected after restore (no false positives).
func TestReconnectRestore_TurnPotPreserved(t *testing.T) {
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
		return &boot{db, srv, gs, conn, pokerrpc.NewLobbyServiceClient(conn), pokerrpc.NewPokerServiceClient(conn)}
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

	b1 := start(t)
	defer stop(b1)

	// Seed balances
	setBalance := func(pid string, want int64) {
		rb, _ := b1.lc.GetBalance(ctx, &pokerrpc.GetBalanceRequest{PlayerId: pid})
		cur := int64(0)
		if rb != nil {
			cur = rb.Balance
		}
		if d := want - cur; d != 0 {
			_, err := b1.lc.UpdateBalance(ctx, &pokerrpc.UpdateBalanceRequest{PlayerId: pid, Amount: d, Description: "seed"})
			require.NoError(t, err)
		}
	}
	p1, p2 := "p1", "p2"
	setBalance(p1, 10_000)
	setBalance(p2, 10_000)

	// Create table, join, ready
	cr, err := b1.lc.CreateTable(ctx, &pokerrpc.CreateTableRequest{
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
	tableID := cr.TableId
	_, err = b1.lc.JoinTable(ctx, &pokerrpc.JoinTableRequest{PlayerId: p2, TableId: tableID})
	require.NoError(t, err)
	_, err = b1.lc.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{PlayerId: p1, TableId: tableID})
	require.NoError(t, err)
	_, err = b1.lc.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{PlayerId: p2, TableId: tableID})
	require.NoError(t, err)

	waitPhase := func(pc pokerrpc.PokerServiceClient, phase pokerrpc.GamePhase) {
		require.Eventually(t, func() bool {
			st, err := pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
			return err == nil && st.GameState.GetPhase() == phase
		}, 3*time.Second, 25*time.Millisecond)
	}
	waitPhase(b1.pc, pokerrpc.GamePhase_PRE_FLOP)

	// Preflop: current calls -> next checks → FLOP
	st, _ := b1.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
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
	waitPhase(b1.pc, pokerrpc.GamePhase_FLOP)

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
	waitPhase(b1.pc, pokerrpc.GamePhase_TURN)

	// Restart server at TURN
	stop(b1)
	b2 := start(t)
	defer stop(b2)

	// Attach both streams to trigger restore
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

	// Verify restored TURN with non-zero pot and valid current player not ALL_IN
	waitPhase(b2.pc, pokerrpc.GamePhase_TURN)
	stR, err := b2.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	require.NoError(t, err)
	require.Equal(t, pokerrpc.GamePhase_TURN, stR.GameState.GetPhase())
	require.Greater(t, stR.GameState.GetPot(), int64(0), "expected non-zero pot at TURN after restore")
	curID := stR.GameState.GetCurrentPlayer()
	require.NotEmpty(t, curID, "expected a current player at TURN")
	// Ensure current player is not all-in
	isAllIn := false
	for _, pl := range stR.GameState.GetPlayers() {
		if pl.GetId() == curID {
			isAllIn = pl.GetIsAllIn()
			break
		}
	}
	require.False(t, isAllIn, "current player should not be ALL_IN at TURN")

	// Out-of-turn action should be rejected
	other := p1
	if curID == p1 {
		other = p2
	}
	_, err = b2.pc.CheckBet(ctx, &pokerrpc.CheckBetRequest{PlayerId: other, TableId: tableID})
	require.Error(t, err, "out-of-turn action should be rejected after restore at TURN")
}
