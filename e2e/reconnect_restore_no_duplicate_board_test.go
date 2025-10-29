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

// TestReconnectRestore_NoDuplicateBoardCards ensures that after a server restart
// during PRE_FLOP, the subsequently dealt FLOP does not contain any of the
// players' hole cards (i.e., deck state is correctly restored).
func TestReconnectRestore_NoDuplicateBoardCards(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Reuse DB across restarts
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

	// Boot #1
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

	// Create table and join
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

	// Ready both
	for _, pid := range []string{p1, p2} {
		_, err := b1.lc.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{PlayerId: pid, TableId: tableID})
		require.NoError(t, err)
	}

	// Start streams to capture each player's own hole cards
	s1, err := b1.pc.StartGameStream(ctx, &pokerrpc.StartGameStreamRequest{TableId: tableID, PlayerId: p1})
	require.NoError(t, err)
	s2, err := b1.pc.StartGameStream(ctx, &pokerrpc.StartGameStreamRequest{TableId: tableID, PlayerId: p2})
	require.NoError(t, err)

	// Wait for initial PRE_FLOP snapshots from both
	var st1, st2 *pokerrpc.GameUpdate
	require.Eventually(t, func() bool {
		u, err := s1.Recv()
		if err == nil && u != nil && u.Phase == pokerrpc.GamePhase_PRE_FLOP {
			st1 = u
			return true
		}
		return false
	}, 3*time.Second, 25*time.Millisecond)
	require.Eventually(t, func() bool {
		u, err := s2.Recv()
		if err == nil && u != nil && u.Phase == pokerrpc.GamePhase_PRE_FLOP {
			st2 = u
			return true
		}
		return false
	}, 3*time.Second, 25*time.Millisecond)

	// Extract each player's own hole cards
	getOwn := func(update *pokerrpc.GameUpdate, pid string) []*pokerrpc.Card {
		for _, pl := range update.Players {
			if pl != nil && pl.Id == pid {
				return pl.Hand
			}
		}
		return nil
	}
	p1Hole := getOwn(st1, p1)
	p2Hole := getOwn(st2, p2)
	require.Len(t, p1Hole, 2)
	require.Len(t, p2Hole, 2)

	// Restart before any flop is dealt
	stop(b1)
	b2 := start(t)
	defer stop(b2)

	// Reattach to trigger restore
	rs1, err := b2.pc.StartGameStream(ctx, &pokerrpc.StartGameStreamRequest{TableId: tableID, PlayerId: p1})
	require.NoError(t, err)
	if _, err := rs1.Recv(); err == nil { /* initial snapshot ok */
	}
	rs2, err := b2.pc.StartGameStream(ctx, &pokerrpc.StartGameStreamRequest{TableId: tableID, PlayerId: p2})
	require.NoError(t, err)
	if _, err := rs2.Recv(); err == nil { /* initial snapshot ok */
	}

	// Ensure we are at PRE_FLOP after restore
	require.Eventually(t, func() bool {
		st, err := b2.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		return err == nil && st.GameState.GetPhase() == pokerrpc.GamePhase_PRE_FLOP
	}, 3*time.Second, 25*time.Millisecond)

	// Complete pre-flop: current player calls; next player checks → FLOP
	st, err := b2.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	require.NoError(t, err)
	cur := st.GameState.GetCurrentPlayer()
	_, err = b2.pc.CallBet(ctx, &pokerrpc.CallBetRequest{TableId: tableID, PlayerId: cur})
	require.NoError(t, err)

	// Wait turn to switch, then check
	require.Eventually(t, func() bool {
		st, _ := b2.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		return st.GameState.GetCurrentPlayer() != cur
	}, 2*time.Second, 25*time.Millisecond)
	st, _ = b2.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	next := st.GameState.GetCurrentPlayer()
	_, err = b2.pc.CheckBet(ctx, &pokerrpc.CheckBetRequest{TableId: tableID, PlayerId: next})
	require.NoError(t, err)

	// Wait for FLOP
	require.Eventually(t, func() bool {
		st, _ := b2.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		return st.GameState.GetPhase() == pokerrpc.GamePhase_FLOP && len(st.GameState.CommunityCards) >= 3
	}, 3*time.Second, 25*time.Millisecond)

	// Fetch community cards and assert no duplicates with any hole card
	st, err = b2.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	require.NoError(t, err)
	board := st.GameState.CommunityCards
	require.True(t, len(board) >= 3)
	// Build a set of all hole cards
	key := func(c *pokerrpc.Card) string { return c.Value + "|" + c.Suit }
	holes := map[string]struct{}{
		key(p1Hole[0]): {}, key(p1Hole[1]): {}, key(p2Hole[0]): {}, key(p2Hole[1]): {},
	}
	for i := 0; i < 3; i++ { // only flop
		if _, dup := holes[key(board[i])]; dup {
			t.Fatalf("duplicate card on board after restore: %s of %s appears in hole cards", board[i].GetValue(), board[i].GetSuit())
		}
	}
}
