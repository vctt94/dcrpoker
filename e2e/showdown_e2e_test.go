package e2e

import (
	"context"
	"fmt"
	"net"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"github.com/vctt94/bisonbotkit/logging"
	"github.com/vctt94/pokerbisonrelay/pkg/poker"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	"github.com/vctt94/pokerbisonrelay/pkg/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

// TestShowdownRestoreBug_HandEvaluationCorrectness verifies that after a server restart
// during a hand, the showdown correctly determines the winner based on hand strength.
func TestShowdownRestoreBug_HandEvaluationCorrectness(t *testing.T) {
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
		srv, err := server.NewTestServer(db, lb)
		require.NoError(t, err)

		lis := bufconn.Listen(1024 * 1024)

		grpcSrv := grpc.NewServer()
		pokerrpc.RegisterLobbyServiceServer(grpcSrv, srv)
		pokerrpc.RegisterPokerServiceServer(grpcSrv, srv)

		go grpcSrv.Serve(lis)

		dialer := func(ctx context.Context, _ string) (net.Conn, error) { return lis.Dial() }
		conn, err := grpc.DialContext(context.Background(), "bufnet", grpc.WithContextDialer(dialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)

		lc := pokerrpc.NewLobbyServiceClient(conn)
		pc := pokerrpc.NewPokerServiceClient(conn)

		return &boot{db, srv, grpcSrv, conn, lc, pc}
	}

	// 2) Start first server instance
	boot1 := start(t)
	// Ensure cleanup order: stop gRPC client/server before stopping the Server,
	// so stream handlers can exit before Server.Stop waits on them.
	defer boot1.db.Close()
	defer boot1.srv.Stop()
	defer boot1.conn.Close()
	defer boot1.grpc.Stop()

	p1, p2 := "player1", "player2"
	// Create table and join players
	tableResp, err := boot1.lc.CreateTable(ctx, &pokerrpc.CreateTableRequest{
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
	tableID := tableResp.TableId

	// Join second player
	_, err = boot1.lc.JoinTable(ctx, &pokerrpc.JoinTableRequest{TableId: tableID, PlayerId: p2})
	require.NoError(t, err)

	// Ready up both players
	for _, pid := range []string{p1, p2} {
		_, err := boot1.lc.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{PlayerId: pid, TableId: tableID})
		require.NoError(t, err)
	}

	// Start streams
	s1, err := boot1.pc.StartGameStream(ctx, &pokerrpc.StartGameStreamRequest{TableId: tableID, PlayerId: p1})
	require.NoError(t, err)
	if closer, ok := interface{}(s1).(interface{ CloseSend() error }); ok {
		defer closer.CloseSend()
	}
	s2, err := boot1.pc.StartGameStream(ctx, &pokerrpc.StartGameStreamRequest{TableId: tableID, PlayerId: p2})
	require.NoError(t, err)
	if closer, ok := interface{}(s2).(interface{ CloseSend() error }); ok {
		defer closer.CloseSend()
	}

	// Helper: receive until player's own hand is visible or timeout.
	waitOwnHand := func(s pokerrpc.PokerService_StartGameStreamClient, pid string, d time.Duration) *pokerrpc.GameUpdate {
		deadline := time.After(d)
		for {
			select {
			case <-deadline:
				t.Fatalf("timeout waiting for own hand for %s", pid)
				return nil
			default:
				st, err := s.Recv()
				require.NoError(t, err)
				if st == nil {
					continue
				}
				// Find own player entry and check hand length
				for _, pl := range st.Players {
					if pl != nil && pl.Id == pid && len(pl.Hand) >= 2 {
						return st
					}
				}
			}
		}
	}

	// Wait for snapshots where each player sees their own 2 hole cards
	u1 := waitOwnHand(s1, p1, 5*time.Second)
	u2 := waitOwnHand(s2, p2, 5*time.Second)
	require.Equal(t, pokerrpc.GamePhase_PRE_FLOP, u1.Phase)
	require.Equal(t, pokerrpc.GamePhase_PRE_FLOP, u2.Phase)

	// Extract hole cards
	getOwn := func(update *pokerrpc.GameUpdate, pid string) []*pokerrpc.Card {
		for _, pl := range update.Players {
			if pl != nil && pl.Id == pid {
				return pl.Hand
			}
		}
		return nil
	}
	p1HolePB := getOwn(u1, p1)
	p2HolePB := getOwn(u2, p2)
	require.Len(t, p1HolePB, 2)
	require.Len(t, p2HolePB, 2)

	// Drive streets dynamically (preflop close → flop, flop checks → turn, turn checks → river)
	waitPhase := func(phase pokerrpc.GamePhase) {
		require.Eventually(t, func() bool {
			st, err := boot1.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
			return err == nil && st.GameState.GetPhase() == phase
		}, 3*time.Second, 25*time.Millisecond)
	}
	st, err := boot1.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	require.NoError(t, err)
	cur := st.GameState.GetCurrentPlayer()
	_, err = boot1.pc.CallBet(ctx, &pokerrpc.CallBetRequest{TableId: tableID, PlayerId: cur})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		st, _ := boot1.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		return st.GameState.GetCurrentPlayer() != cur
	}, 2*time.Second, 25*time.Millisecond)
	st, _ = boot1.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	next := st.GameState.GetCurrentPlayer()
	_, err = boot1.pc.CheckBet(ctx, &pokerrpc.CheckBetRequest{TableId: tableID, PlayerId: next})
	require.NoError(t, err)
	waitPhase(pokerrpc.GamePhase_FLOP)

	st, _ = boot1.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	cur = st.GameState.GetCurrentPlayer()
	_, err = boot1.pc.CheckBet(ctx, &pokerrpc.CheckBetRequest{TableId: tableID, PlayerId: cur})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		st, _ := boot1.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		return st.GameState.GetCurrentPlayer() != cur
	}, 2*time.Second, 25*time.Millisecond)
	st, _ = boot1.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	next = st.GameState.GetCurrentPlayer()
	_, err = boot1.pc.CheckBet(ctx, &pokerrpc.CheckBetRequest{TableId: tableID, PlayerId: next})
	require.NoError(t, err)
	waitPhase(pokerrpc.GamePhase_TURN)

	st, _ = boot1.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	cur = st.GameState.GetCurrentPlayer()
	_, err = boot1.pc.CheckBet(ctx, &pokerrpc.CheckBetRequest{TableId: tableID, PlayerId: cur})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		st, _ := boot1.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		return st.GameState.GetCurrentPlayer() != cur
	}, 2*time.Second, 25*time.Millisecond)
	st, _ = boot1.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	next = st.GameState.GetCurrentPlayer()
	_, err = boot1.pc.CheckBet(ctx, &pokerrpc.CheckBetRequest{TableId: tableID, PlayerId: next})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		st, _ := boot1.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		return st.GameState.GetPhase() == pokerrpc.GamePhase_RIVER
	}, 3*time.Second, 25*time.Millisecond)

	// Capture board at RIVER
	r, err := boot1.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	require.NoError(t, err)
	boardPB := r.GameState.CommunityCards

	// Convert pb cards to internal poker.Card
	toSuit := func(s string) (poker.Suit, error) {
		switch s {
		case "♠", "s", "S", "spades", "Spades":
			return poker.Spades, nil
		case "♥", "h", "H", "hearts", "Hearts":
			return poker.Hearts, nil
		case "♦", "d", "D", "diamonds", "Diamonds":
			return poker.Diamonds, nil
		case "♣", "c", "C", "clubs", "Clubs":
			return poker.Clubs, nil
		default:
			return "", fmt.Errorf("invalid suit: %s", s)
		}
	}
	toValue := func(v string) (poker.Value, error) {
		switch v {
		case "A", "a", "ace", "Ace":
			return poker.Ace, nil
		case "K", "k", "king", "King":
			return poker.King, nil
		case "Q", "q", "queen", "Queen":
			return poker.Queen, nil
		case "J", "j", "jack", "Jack":
			return poker.Jack, nil
		case "10", "T", "t", "ten", "Ten":
			return poker.Ten, nil
		case "9", "nine", "Nine":
			return poker.Nine, nil
		case "8", "eight", "Eight":
			return poker.Eight, nil
		case "7", "seven", "Seven":
			return poker.Seven, nil
		case "6", "six", "Six":
			return poker.Six, nil
		case "5", "five", "Five":
			return poker.Five, nil
		case "4", "four", "Four":
			return poker.Four, nil
		case "3", "three", "Three":
			return poker.Three, nil
		case "2", "two", "Two":
			return poker.Two, nil
		default:
			return "", fmt.Errorf("invalid value: %s", v)
		}
	}
	toInternal := func(cs []*pokerrpc.Card) ([]poker.Card, error) {
		out := make([]poker.Card, 0, len(cs))
		for _, c := range cs {
			if c == nil {
				return nil, fmt.Errorf("nil card")
			}
			s, err := toSuit(c.Suit)
			if err != nil {
				return nil, err
			}
			v, err := toValue(c.Value)
			if err != nil {
				return nil, err
			}
			out = append(out, poker.NewCardFromSuitValue(s, v))
		}
		return out, nil
	}

	p1Hole, err := toInternal(p1HolePB)
	require.NoError(t, err)
	p2Hole, err := toInternal(p2HolePB)
	require.NoError(t, err)
	board, err := toInternal(boardPB)
	require.NoError(t, err)

	hv1, err := poker.EvaluateHand(p1Hole, board)
	require.NoError(t, err)
	hv2, err := poker.EvaluateHand(p2Hole, board)
	require.NoError(t, err)

	cmp := poker.CompareHands(hv1, hv2)

	// 4) Server restart simulation - stop first server
	boot1.grpc.Stop()
	boot1.conn.Close()

	// 5) Start second server instance (restore from snapshot)
	boot2 := start(t)
	defer boot2.db.Close()
	defer boot2.srv.Stop()
	defer boot2.conn.Close()
	defer boot2.grpc.Stop()

	// Reconnect both players (read a single initial snapshot to ensure stream is established)
	s1r, err := boot2.pc.StartGameStream(ctx, &pokerrpc.StartGameStreamRequest{TableId: tableID, PlayerId: p1})
	require.NoError(t, err)
	if closer, ok := interface{}(s1r).(interface{ CloseSend() error }); ok {
		defer closer.CloseSend()
	}
	s2r, err := boot2.pc.StartGameStream(ctx, &pokerrpc.StartGameStreamRequest{TableId: tableID, PlayerId: p2})
	require.NoError(t, err)
	if closer, ok := interface{}(s2r).(interface{ CloseSend() error }); ok {
		defer closer.CloseSend()
	}
	// Reuse a lightweight one-shot receive with timeout
	recvOne := func(s pokerrpc.PokerService_StartGameStreamClient, d time.Duration) *pokerrpc.GameUpdate {
		ch := make(chan *pokerrpc.GameUpdate, 1)
		errCh := make(chan error, 1)
		go func() {
			st, err := s.Recv()
			if err != nil {
				errCh <- err
				return
			}
			ch <- st
		}()
		select {
		case st := <-ch:
			return st
		case err := <-errCh:
			require.NoError(t, err)
			return nil
		case <-time.After(5 * time.Second):
			t.Fatalf("timeout waiting for restored stream update")
			return nil
		}
	}
	_ = recvOne(s1r, 5*time.Second)
	_ = recvOne(s2r, 5*time.Second)

	// Verify we're still in RIVER phase
	stR, err := boot2.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	require.NoError(t, err)
	require.Equal(t, pokerrpc.GamePhase_RIVER, stR.GameState.GetPhase())

	// Complete the hand on RIVER using current player order (post-restore)
	curR := stR.GameState.GetCurrentPlayer()
	require.NotEmpty(t, curR)
	_, err = boot2.pc.CheckBet(ctx, &pokerrpc.CheckBetRequest{TableId: tableID, PlayerId: curR})
	require.NoError(t, err)
	other := p1
	if curR == p1 {
		other = p2
	}
	require.Eventually(t, func() bool {
		st, err := boot2.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		if err != nil || st == nil || st.GameState == nil {
			return false
		}
		ph := st.GameState.GetPhase()
		if ph == pokerrpc.GamePhase_SHOWDOWN {
			return true
		}
		return ph == pokerrpc.GamePhase_RIVER && st.GameState.GetCurrentPlayer() == other
	}, 2*time.Second, 25*time.Millisecond)
	stAfter, _ := boot2.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
	if stAfter.GameState.GetPhase() == pokerrpc.GamePhase_RIVER {
		_, err = boot2.pc.CheckBet(ctx, &pokerrpc.CheckBetRequest{TableId: tableID, PlayerId: other})
		require.NoError(t, err)
	}

	// Wait for showdown to complete
	require.Eventually(t, func() bool {
		st, err := boot2.pc.GetGameState(ctx, &pokerrpc.GetGameStateRequest{TableId: tableID})
		return err == nil && st.GameState.GetPhase() == pokerrpc.GamePhase_SHOWDOWN
	}, 3*time.Second, 25*time.Millisecond)

	// Winners must match evaluator
	winners, err := boot2.pc.GetLastWinners(ctx, &pokerrpc.GetLastWinnersRequest{TableId: tableID})
	require.NoError(t, err)
	switch cmp {
	case 0:
		require.Len(t, winners.Winners, 2)
		got := map[string]bool{}
		for _, w := range winners.Winners {
			got[w.PlayerId] = true
		}
		require.True(t, got[p1] && got[p2])
	case 1:
		require.GreaterOrEqual(t, len(winners.Winners), 1)
		require.Equal(t, p1, winners.Winners[0].PlayerId)
	case -1:
		require.GreaterOrEqual(t, len(winners.Winners), 1)
		require.Equal(t, p2, winners.Winners[0].PlayerId)
	}
}
