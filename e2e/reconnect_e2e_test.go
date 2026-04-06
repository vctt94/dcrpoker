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

// TestReconnectRestore_MidHandReturnsWaiting verifies that restarting during
// FLOP does not restore the in-flight hand. The table comes back in WAITING
// with the seated ready players preserved.
func TestReconnectRestore_MidHandReturnsWaiting(t *testing.T) {
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

		// Seed auth users used by authenticated table creation/join flows.
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

	// Mid-hand restore is intentionally disabled. The table should come back in
	// WAITING with no current player and no active hand to act on.
	require.Eventually(t, func() bool {
		st, err := b2.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		return err == nil && st != nil && st.GameState != nil &&
			st.GameState.GetPhase() == pokerrpc.GamePhase_WAITING
	}, 3*time.Second, 25*time.Millisecond)

	st, err = b2.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	require.NoError(t, err)
	require.Equal(t, pokerrpc.GamePhase_WAITING, st.GameState.GetPhase())
	require.Empty(t, st.GameState.GetCurrentPlayer())
	require.Len(t, st.GameState.GetPlayers(), 2)
	for _, pl := range st.GameState.GetPlayers() {
		require.True(t, pl.GetIsReady(), "ready state should survive mid-hand restart")
	}

	_, err = b2.pc.CheckBet(ctx, &pokerrpc.CheckBetRequest{PlayerId: p1, TableId: tableID})
	require.Error(t, err, "actions must be rejected when no hand is restored")
}

// TestReconnectRestore_TurnReturnsWaiting verifies that restarting during TURN
// drops the in-flight hand instead of restoring street state, pot state, or a
// current actor.
func TestReconnectRestore_TurnReturnsWaiting(t *testing.T) {
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

	// Mid-hand restore is intentionally disabled.
	require.Eventually(t, func() bool {
		st, err := b2.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		return err == nil && st != nil && st.GameState != nil &&
			st.GameState.GetPhase() == pokerrpc.GamePhase_WAITING
	}, 3*time.Second, 25*time.Millisecond)
	stR, err := b2.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	require.NoError(t, err)
	require.Equal(t, pokerrpc.GamePhase_WAITING, stR.GameState.GetPhase())
	require.Zero(t, stR.GameState.GetPot(), "no in-flight pot should survive restart")
	require.Empty(t, stR.GameState.GetCurrentPlayer(), "no current player should remain after dropping the hand")
	require.Len(t, stR.GameState.GetPlayers(), 2)
	for _, pl := range stR.GameState.GetPlayers() {
		require.True(t, pl.GetIsReady(), "ready state should survive mid-hand restart")
	}

	_, err = b2.pc.CheckBet(ctx, &pokerrpc.CheckBetRequest{PlayerId: p2, TableId: tableID})
	require.Error(t, err, "actions must be rejected when TURN is not restored")
}

// TestReconnectRestore_PreFlopClearsHoleCards verifies that restarting during
// PRE_FLOP does not restore the in-flight hand and does not leak previously
// dealt hole cards after reconnect.
func TestReconnectRestore_PreFlopClearsHoleCards(t *testing.T) {
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

	var st *pokerrpc.GetGameStateResponse
	require.Eventually(t, func() bool {
		resp, err := b2.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		if err != nil || resp == nil || resp.GameState == nil {
			return false
		}
		st = resp
		return true
	}, 3*time.Second, 25*time.Millisecond)

	require.Equal(t, pokerrpc.GamePhase_WAITING, st.GameState.GetPhase())
	require.Empty(t, st.GameState.GetCurrentPlayer())
	for _, pl := range st.GameState.GetPlayers() {
		require.Empty(t, pl.GetHand(), "restored WAITING state must not expose old hole cards")
	}

	// Sanity check that cards really were dealt before restart, so this asserts
	// the reconnect path is clearing prior in-flight hand data.
	require.Len(t, p1Hole, 2)
	require.Len(t, p2Hole, 2)

	_, err = b2.pc.CheckBet(ctx, &pokerrpc.CheckBetRequest{PlayerId: p1, TableId: tableID})
	require.Error(t, err, "actions must be rejected when PRE_FLOP is not restored")
}

// TestReconnectRestore_ShowdownStartsNextHand verifies that a restart after
// SHOWDOWN resumes the match from the next-hand boundary instead of restoring
// the completed hand.
func TestReconnectRestore_ShowdownStartsNextHand(t *testing.T) {
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

		// Seed auth users so CreateTable/JoinTable requests are authenticated.
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

	var st2 *pokerrpc.GetGameStateResponse
	require.Eventually(t, func() bool {
		resp, err := b2.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		if err != nil || resp == nil || resp.GameState == nil {
			return false
		}
		st2 = resp
		return resp.GameState.GetPhase() == pokerrpc.GamePhase_PRE_FLOP &&
			resp.GameState.GetCurrentPlayer() != ""
	}, 3*time.Second, 25*time.Millisecond)

	require.NotNil(t, st2)
	require.Equal(t, pokerrpc.GamePhase_PRE_FLOP, st2.GameState.GetPhase())
	require.NotEmpty(t, st2.GameState.GetCurrentPlayer(), "a fresh hand should resume with a current player")

	outOfTurn := p1
	if st2.GameState.GetCurrentPlayer() == p1 {
		outOfTurn = p2
	}
	_, err = b2.pc.CheckBet(ctx, &pokerrpc.CheckBetRequest{PlayerId: outOfTurn, TableId: tableID})
	require.Error(t, err, "out-of-turn actions should still be rejected on the resumed hand")
}

// TestPotRestoration_NoStalePotAfterReconnect verifies that a restart during an
// in-flight hand does not preserve the old pot. Mid-hand restore is disabled,
// so reconnect should return WAITING with a zero pot.
func TestPotRestoration_NoStalePotAfterReconnect(t *testing.T) {
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

	// Wait for game to start before checking phases
	require.Eventually(t, func() bool {
		st, err := b1.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		return err == nil && st.GameState.GetGameStarted()
	}, 3*time.Second, 25*time.Millisecond, "game should start after all players ready")

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

	require.Eventually(t, func() bool {
		st, err := b2.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		return err == nil && st != nil && st.GameState != nil &&
			st.GameState.GetPhase() == pokerrpc.GamePhase_WAITING
	}, 3*time.Second, 25*time.Millisecond)

	st, err = b2.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	require.NoError(t, err)
	require.Equal(t, pokerrpc.GamePhase_WAITING, st.GameState.GetPhase())
	require.Zero(t, st.GameState.GetPot(), "stale pot must not survive mid-hand reconnect")
	require.Empty(t, st.GameState.GetCurrentPlayer())

	_, err = b2.pc.CheckBet(ctx, &pokerrpc.CheckBetRequest{PlayerId: p1, TableId: tableID})
	require.Error(t, err, "actions must be rejected when the old hand is discarded")
}
