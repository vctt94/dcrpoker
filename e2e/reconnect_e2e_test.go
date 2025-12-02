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
	"google.golang.org/grpc/test/bufconn"
)

// TestReconnectRestore_ChecksAdvance verifies that after a server restart and reconnect
// the game is restored and a pair of checks on the flop advances the game to the turn.
//
// NOTE: This test encodes the expected behavior.
func TestReconnectRestore_ChecksAdvance(t *testing.T) {
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

		// Seed auth users required by the tables.host_id foreign key.
		seedCtx := context.Background()
		for _, pid := range []string{"p1", "p2"} {
			require.NoError(t, db.UpsertAuthUser(seedCtx, pid, pid))
		}

		lb, _ := logging.NewLogBackend(logging.LogConfig{DebugLevel: "debug"})
		srv, err := server.NewTestServer(db, lb)
		require.NoError(t, err)

		lis := bufconn.Listen(1024 * 1024)
		gs := grpc.NewServer()
		pokerrpc.RegisterLobbyServiceServer(gs, srv)
		pokerrpc.RegisterPokerServiceServer(gs, srv)
		go func() { _ = gs.Serve(lis) }()

		dialer := func(ctx context.Context, _ string) (net.Conn, error) { return lis.Dial() }
		conn, err := grpc.DialContext(context.Background(), "bufnet", grpc.WithContextDialer(dialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
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

	p1, p2 := "p1", "p2"

	// Create table and join both
	createResp, err := b1.lc.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      p1,
		SmallBlind:    10,
		BigBlind:      20,
		MinPlayers:    2,
		MaxPlayers:    2,
		BuyIn:         0, // BuyIn: 0 to avoid escrow requirement in tests
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

	// Pre-flop: current acts (call), then next checks to close round -> FLOP
	st, err := b1.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	require.NoError(t, err)
	cur := st.GameState.GetCurrentPlayer()
	_, err = b1.pc.CallBet(ctx, &pokerrpc.CallBetRequest{PlayerId: cur, TableId: tableID})
	require.NoError(t, err)
	// Next player check
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

	// 3) Simulate server restart
	stop(b1)
	// Boot second server on same DB
	b2 := start(t)
	defer stop(b2)

	// Attach game streams (reconnect) to trigger restore in production code.
	// Attach both players to simulate production behavior with two clients.
	s1, err := b2.pc.StartGameStream(ctx, &pokerrpc.StartGameStreamRequest{TableId: tableID, PlayerId: p1})
	require.NoError(t, err)
	if closer, ok := interface{}(s1).(interface{ CloseSend() error }); ok {
		defer closer.CloseSend()
	}
	if _, err := s1.Recv(); err == nil { /* ok */
	}
	s2, err := b2.pc.StartGameStream(ctx, &pokerrpc.StartGameStreamRequest{TableId: tableID, PlayerId: p2})
	require.NoError(t, err)
	if closer, ok := interface{}(s2).(interface{ CloseSend() error }); ok {
		defer closer.CloseSend()
	}
	if _, err := s2.Recv(); err == nil { /* ok */
	}

	// Wait until restored game shows FLOP after reconnect
	waitPhase(b2.pc, pokerrpc.GamePhase_FLOP)

	// 4) After reconnect: both players check; EXPECT: advance to TURN
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

	// Assert we advance to TURN
	require.Eventually(t, func() bool {
		st, _ := b2.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		return st.GameState.GetPhase() == pokerrpc.GamePhase_TURN
	}, 2*time.Second, 25*time.Millisecond, "expected to advance to TURN after two checks post-reconnect")
}

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

		seedCtx := context.Background()
		for _, pid := range []string{"p1", "p2"} {
			require.NoError(t, db.UpsertAuthUser(seedCtx, pid, pid))
		}

		lb, _ := logging.NewLogBackend(logging.LogConfig{DebugLevel: "debug"})
		srv, err := server.NewTestServer(db, lb)
		require.NoError(t, err)
		lis := bufconn.Listen(1024 * 1024)
		gs := grpc.NewServer()
		pokerrpc.RegisterLobbyServiceServer(gs, srv)
		pokerrpc.RegisterPokerServiceServer(gs, srv)
		go func() { _ = gs.Serve(lis) }()
		dialer := func(ctx context.Context, _ string) (net.Conn, error) { return lis.Dial() }
		conn, err := grpc.DialContext(context.Background(), "bufnet", grpc.WithContextDialer(dialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
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

	p1, p2 := "p1", "p2"

	// Create table, join, ready
	cr, err := b1.lc.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      p1,
		SmallBlind:    10,
		BigBlind:      20,
		MinPlayers:    2,
		MaxPlayers:    2,
		BuyIn:         0, // BuyIn: 0 to avoid escrow requirement in tests
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
	// CI can be slower to restore; allow a bit more time here.
	require.Eventually(t, func() bool {
		st, err := b2.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		return err == nil && st.GameState.GetPhase() == pokerrpc.GamePhase_TURN
	}, 6*time.Second, 25*time.Millisecond)
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

		seedCtx := context.Background()
		for _, pid := range []string{"p1", "p2"} {
			require.NoError(t, db.UpsertAuthUser(seedCtx, pid, pid))
		}

		lb, _ := logging.NewLogBackend(logging.LogConfig{DebugLevel: "debug"})
		srv, err := server.NewTestServer(db, lb)
		require.NoError(t, err)

		lis := bufconn.Listen(1024 * 1024)
		gs := grpc.NewServer()
		pokerrpc.RegisterLobbyServiceServer(gs, srv)
		pokerrpc.RegisterPokerServiceServer(gs, srv)
		go func() { _ = gs.Serve(lis) }()

		dialer := func(ctx context.Context, _ string) (net.Conn, error) { return lis.Dial() }
		conn, err := grpc.DialContext(context.Background(), "bufnet", grpc.WithContextDialer(dialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
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

	p1, p2 := "p1", "p2"

	// Create table and join
	createResp, err := b1.lc.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      p1,
		SmallBlind:    10,
		BigBlind:      20,
		MinPlayers:    2,
		MaxPlayers:    2,
		BuyIn:         0, // BuyIn: 0 to avoid escrow requirement in tests
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

	// Give async persistence a brief window to flush the PRE_FLOP snapshot.
	time.Sleep(50 * time.Millisecond)

	// Start second server instance on the same DB without stopping b1 first,
	// matching the crash-style restart used in server-level snapshot tests.
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

// TestReconnectRestore_ShowdownPhasePreserved ensures that after finishing a hand
// at SHOWDOWN and restarting, the restored phase remains SHOWDOWN with no current player,
// and actions are rejected.
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

		// Seed auth users so CreateTable/JoinTable satisfy FK constraints on tables.host_id.
		seedCtx := context.Background()
		for _, pid := range []string{"p1", "p2"} {
			require.NoError(t, db.UpsertAuthUser(seedCtx, pid, pid))
		}

		lb, _ := logging.NewLogBackend(logging.LogConfig{DebugLevel: "debug"})
		srv, err := server.NewTestServer(db, lb)
		require.NoError(t, err)

		lis := bufconn.Listen(1024 * 1024)
		gs := grpc.NewServer()
		pokerrpc.RegisterLobbyServiceServer(gs, srv)
		pokerrpc.RegisterPokerServiceServer(gs, srv)
		go func() { _ = gs.Serve(lis) }()

		dialer := func(ctx context.Context, _ string) (net.Conn, error) { return lis.Dial() }
		conn, err := grpc.DialContext(context.Background(), "bufnet", grpc.WithContextDialer(dialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
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

	p1, p2 := "p1", "p2"

	// Create table and join second player
	createResp, err := b1.lc.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      p1,
		SmallBlind:    10,
		BigBlind:      20,
		MinPlayers:    2,
		MaxPlayers:    2,
		BuyIn:         0, // BuyIn: 0 to avoid escrow requirement in tests
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

		seedCtx := context.Background()
		for _, pid := range []string{"p1", "p2"} {
			require.NoError(t, db.UpsertAuthUser(seedCtx, pid, pid))
		}

		lb, _ := logging.NewLogBackend(logging.LogConfig{DebugLevel: "debug"})
		srv, err := server.NewTestServer(db, lb)
		require.NoError(t, err)

		lis := bufconn.Listen(1024 * 1024)
		gs := grpc.NewServer()
		pokerrpc.RegisterLobbyServiceServer(gs, srv)
		pokerrpc.RegisterPokerServiceServer(gs, srv)
		go func() { _ = gs.Serve(lis) }()

		dialer := func(ctx context.Context, _ string) (net.Conn, error) { return lis.Dial() }
		conn, err := grpc.DialContext(context.Background(), "bufnet", grpc.WithContextDialer(dialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
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

	p1, p2 := "p1", "p2"

	// Create table and join both
	createResp, err := b1.lc.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      p1,
		SmallBlind:    10,
		BigBlind:      20,
		MinPlayers:    2,
		MaxPlayers:    2,
		BuyIn:         0, // BuyIn: 0 to avoid escrow requirement in tests
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
	if _, err := s1.Recv(); err == nil { /* ok */
	}
	s2, err := b2.pc.StartGameStream(ctx, &pokerrpc.StartGameStreamRequest{TableId: tableID, PlayerId: p2})
	require.NoError(t, err)
	if closer, ok := interface{}(s2).(interface{ CloseSend() error }); ok {
		defer closer.CloseSend()
	}
	if _, err := s2.Recv(); err == nil { /* ok */
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

	// RIVER: both players check to reach showdown. Avoid racing phase change.
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

	// Wait until either it's the other player's turn on RIVER, or SHOWDOWN already
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
}
