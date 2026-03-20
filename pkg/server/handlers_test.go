package server

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/decred/slog"
	"github.com/stretchr/testify/require"
	"github.com/vctt94/pokerbisonrelay/pkg/poker"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	"google.golang.org/grpc/metadata"
)

// ---------- Stub implementations used across unit tests ---------- //

// mockNotificationStream is a lightweight implementation of the
// LobbyService_StartNotificationStreamServer interface that records the
// notifications sent by server.notifyPlayers.
// It implements only the methods actually used by the code-under-test.

type mockNotificationStream struct {
	mu   sync.RWMutex
	sent []*pokerrpc.Notification
}

// Ensure the mock satisfies the required interface at compile-time.
var _ pokerrpc.LobbyService_StartNotificationStreamServer = (*mockNotificationStream)(nil)

// Send records the notification for inspection.
func (m *mockNotificationStream) Send(n *pokerrpc.Notification) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, n)
	return nil
}

// ----- grpc.ServerStream interface stubs ----- //

func (m *mockNotificationStream) SetHeader(metadata.MD) error  { return nil }
func (m *mockNotificationStream) SendHeader(metadata.MD) error { return nil }
func (m *mockNotificationStream) SetTrailer(metadata.MD)       {}
func (m *mockNotificationStream) Context() context.Context     { return context.TODO() }
func (m *mockNotificationStream) SendMsg(interface{}) error    { return nil }
func (m *mockNotificationStream) RecvMsg(interface{}) error    { return nil }

// ---------- Notification handler tests ---------- //

// TestNotificationHandlerAddsTableOnPlayerJoined ensures lobby notifications
// include the table payload so other clients can refresh immediately.
func TestNotificationHandlerAddsTableOnPlayerJoined(t *testing.T) {
	s := newBareServer()
	mockStream := &mockNotificationStream{}
	s.notificationStreams.Store("listener", &NotificationStream{
		playerID: "listener",
		stream:   mockStream,
		done:     make(chan struct{}),
	})

	snap := &TableSnapshot{
		ID: "tid",
		Config: poker.TableConfig{
			ID:         "tid",
			SmallBlind: 10,
			BigBlind:   20,
			MinPlayers: 2,
			MaxPlayers: 6,
		},
		Players: []*PlayerSnapshot{
			{ID: "host", Name: "Host", TableSeat: 0, Balance: 1000},
			{ID: "p2", Name: "Guest", TableSeat: 1, Balance: 1000},
		},
		State: TableState{PlayerCount: 2},
	}

	nh := NewNotificationHandler(s)
	nh.handlePlayerJoined(&GameEvent{
		Type:          pokerrpc.NotificationType_PLAYER_JOINED,
		TableID:       "tid",
		PlayerIDs:     []string{"host", "p2"},
		Payload:       PlayerJoinedPayload{PlayerID: "p2"},
		TableSnapshot: snap,
	})

	mockStream.mu.RLock()
	sentLen := len(mockStream.sent)
	var ntfn *pokerrpc.Notification
	if sentLen > 0 {
		ntfn = mockStream.sent[0]
	}
	mockStream.mu.RUnlock()
	if sentLen != 1 {
		t.Fatalf("expected 1 notification, got %d", sentLen)
	}
	if ntfn.Table == nil {
		t.Fatalf("notification missing table payload")
	}
	if got := ntfn.Table.CurrentPlayers; got != 2 {
		t.Fatalf("expected CurrentPlayers=2, got %d", got)
	}
	if got := len(ntfn.Table.Players); got != 2 {
		t.Fatalf("expected 2 players in table payload, got %d", got)
	}
}

// TestNotificationHandlerAddsTableOnTableCreated collects a fresh snapshot
// when TABLE_CREATED events omit the snapshot payload.
func TestNotificationHandlerAddsTableOnTableCreated(t *testing.T) {
	s := newBareServer()
	mockStream := &mockNotificationStream{}
	s.notificationStreams.Store("listener", &NotificationStream{
		playerID: "listener",
		stream:   mockStream,
		done:     make(chan struct{}),
	})

	cfg := poker.TableConfig{
		ID:         "tid",
		Log:        slog.Disabled,
		GameLog:    slog.Disabled,
		MinPlayers: 2,
		MaxPlayers: 6,
		SmallBlind: 10,
		BigBlind:   20,
	}
	table := poker.NewTable(cfg)
	if _, err := table.AddNewUser("host", &poker.AddUserOptions{DisplayName: "Host"}); err != nil {
		t.Fatalf("add host: %v", err)
	}
	s.tables.Store(cfg.ID, table)

	nh := NewNotificationHandler(s)
	nh.handleTableCreated(&GameEvent{
		Type:    pokerrpc.NotificationType_TABLE_CREATED,
		TableID: cfg.ID,
	})

	mockStream.mu.RLock()
	sentLen := len(mockStream.sent)
	var ntfn *pokerrpc.Notification
	if sentLen > 0 {
		ntfn = mockStream.sent[0]
	}
	mockStream.mu.RUnlock()
	if sentLen != 1 {
		t.Fatalf("expected 1 notification, got %d", sentLen)
	}
	if ntfn.Table == nil {
		t.Fatalf("notification missing table payload")
	}
	if ntfn.Table.CurrentPlayers != 1 {
		t.Fatalf("expected CurrentPlayers=1, got %d", ntfn.Table.CurrentPlayers)
	}
}

func TestLeaveTablePublishesPlayerLeft(t *testing.T) {
	db := NewInMemoryDB()
	defer db.Close()

	logBackend := createTestLogBackend()
	defer logBackend.Close()

	server, err := NewTestServer(db, logBackend)
	require.NoError(t, err)
	defer server.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	hostID := "host"
	guestID := "guest"

	createResp, err := server.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:      hostID,
		SmallBlind:    5,
		BigBlind:      10,
		MinPlayers:    2,
		MaxPlayers:    2,
		BuyIn:         0,
		StartingChips: 1000,
		AutoAdvanceMs: 1000,
	})
	require.NoError(t, err)
	tableID := createResp.TableId

	_, err = server.JoinTable(ctx, &pokerrpc.JoinTableRequest{
		PlayerId: guestID,
		TableId:  tableID,
	})
	require.NoError(t, err)

	mockStream := &mockNotificationStream{}
	server.notificationStreams.Store("listener", &NotificationStream{
		playerID: "listener",
		stream:   mockStream,
		done:     make(chan struct{}),
	})

	_, err = server.LeaveTable(ctx, &pokerrpc.LeaveTableRequest{
		PlayerId: guestID,
		TableId:  tableID,
	})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		mockStream.mu.RLock()
		defer mockStream.mu.RUnlock()
		for _, n := range mockStream.sent {
			if n.Type == pokerrpc.NotificationType_PLAYER_LEFT && n.TableId == tableID {
				return n.Table != nil && n.Table.CurrentPlayers == 1
			}
		}
		return false
	}, 2*time.Second, 10*time.Millisecond, "expected PLAYER_LEFT notification with updated table payload")
}

// TestGameStateHandlerBuildGameStates verifies that game updates are correctly
// built from a table snapshot and that hole cards visibility rules are
// respected.
func TestGameStateHandlerBuildGameStates(t *testing.T) {
	// Build a minimal table snapshot with two players.
	cardA := poker.NewCardFromSuitValue(poker.Spades, poker.Ace)
	cardK := poker.NewCardFromSuitValue(poker.Hearts, poker.King)

	p1Snap := &PlayerSnapshot{
		ID:      "p1",
		Balance: 1000,
		IsReady: true,
		Hand:    []poker.Card{cardA, cardK},
	}
	p2Snap := &PlayerSnapshot{
		ID:      "p2",
		Balance: 1000,
		IsReady: true,
		Hand:    []poker.Card{cardA}, // irrelevant – should be hidden from p1
	}

	gsnap := &poker.GameStateSnapshot{
		Phase:         pokerrpc.GamePhase_PRE_FLOP,
		Pot:           0,
		CurrentBet:    0,
		CurrentPlayer: "p1",
	}

	tsnap := &TableSnapshot{
		ID:           "tid",
		Players:      []*PlayerSnapshot{p1Snap, p2Snap},
		GameSnapshot: gsnap,
		Config:       poker.TableConfig{MinPlayers: 2},
		State:        TableState{GameStarted: true, PlayerCount: 2},
	}

	gsh := NewGameStateHandler(newBareServer())
	updates := gsh.buildGameStatesFromSnapshot(tsnap)

	if len(updates) != 2 {
		t.Fatalf("expected 2 game updates, got %d", len(updates))
	}

	// p1 should see own cards but not p2's.
	up1 := updates["p1"]
	if up1 == nil {
		t.Fatalf("missing update for p1")
	}
	if len(up1.Players) != 2 {
		t.Fatalf("update for p1 should include 2 players, got %d", len(up1.Players))
	}
	var p1HandVisible, p2HandVisible bool
	for _, pl := range up1.Players {
		switch pl.Id {
		case "p1":
			p1HandVisible = len(pl.Hand) == 2
		case "p2":
			p2HandVisible = len(pl.Hand) > 0
		}
	}
	if !p1HandVisible {
		t.Errorf("p1 should see own hand but it's hidden")
	}
	if p2HandVisible {
		t.Errorf("p1 should NOT see p2 hand in preflop phase")
	}
}

// TestGameStateHandlerShowsOwnCardsDuringNewHandDealing asserts that a player
// sees their own hole cards in updates even during NEW_HAND_DEALING, while
// opponents' cards remain hidden.
func TestGameStateHandlerShowsOwnCardsDuringNewHandDealing(t *testing.T) {
	// Arrange: snapshot with phase NEW_HAND_DEALING and dealt hands
	cardA := poker.NewCardFromSuitValue(poker.Spades, poker.Ace)
	cardK := poker.NewCardFromSuitValue(poker.Hearts, poker.King)

	p1Snap := &PlayerSnapshot{
		ID:      "p1",
		Balance: 1000,
		IsReady: true,
		Hand:    []poker.Card{cardA, cardK},
	}
	p2Snap := &PlayerSnapshot{
		ID:      "p2",
		Balance: 1000,
		IsReady: true,
		Hand:    []poker.Card{cardA, cardK},
	}

	tsnap := &TableSnapshot{
		ID:      "tid",
		Players: []*PlayerSnapshot{p1Snap, p2Snap},
		GameSnapshot: &poker.GameStateSnapshot{
			Phase:         pokerrpc.GamePhase_NEW_HAND_DEALING,
			Pot:           0,
			CurrentBet:    0,
			CurrentPlayer: "",
		},
		Config: poker.TableConfig{MinPlayers: 2},
		State:  TableState{GameStarted: true, PlayerCount: 2},
	}

	gsh := NewGameStateHandler(newBareServer())
	updates := gsh.buildGameStatesFromSnapshot(tsnap)
	if len(updates) != 2 {
		t.Fatalf("expected updates for 2 players, got %d", len(updates))
	}

	// When building an update for p1, p1 must see their 2 cards; p2's must be hidden.
	up1 := updates["p1"]
	if up1 == nil {
		t.Fatalf("missing update for p1")
	}
	var p1HandCnt, p2HandCnt int
	for _, pl := range up1.Players {
		switch pl.Id {
		case "p1":
			p1HandCnt = len(pl.Hand)
		case "p2":
			p2HandCnt = len(pl.Hand)
		}
	}
	if p1HandCnt != 2 {
		t.Errorf("p1 should see own 2 hole cards during NEW_HAND_DEALING; got %d", p1HandCnt)
	}
	if p2HandCnt != 0 {
		t.Errorf("p1 should NOT see p2's hand during NEW_HAND_DEALING; got %d", p2HandCnt)
	}

	// Symmetric check for p2's perspective
	up2 := updates["p2"]
	if up2 == nil {
		t.Fatalf("missing update for p2")
	}
	p1HandCnt, p2HandCnt = 0, 0
	for _, pl := range up2.Players {
		switch pl.Id {
		case "p1":
			p1HandCnt = len(pl.Hand)
		case "p2":
			p2HandCnt = len(pl.Hand)
		}
	}
	if p2HandCnt != 2 {
		t.Errorf("p2 should see own 2 hole cards during NEW_HAND_DEALING; got %d", p2HandCnt)
	}
	if p1HandCnt != 0 {
		t.Errorf("p2 should NOT see p1's hand during NEW_HAND_DEALING; got %d", p1HandCnt)
	}
}
