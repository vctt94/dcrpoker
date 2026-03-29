package server

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	"google.golang.org/grpc/metadata"
)

// mockGameStream implements PokerService_StartGameStreamServer for testing stream lifecycle.
type mockGameStream struct {
	ctx    context.Context
	cancel context.CancelFunc
	sentCh chan *pokerrpc.GameUpdate
}

var _ pokerrpc.PokerService_StartGameStreamServer = (*mockGameStream)(nil)

func newMockGameStream() *mockGameStream {
	ctx, cancel := context.WithCancel(context.Background())
	return &mockGameStream{ctx: ctx, cancel: cancel, sentCh: make(chan *pokerrpc.GameUpdate, 4)}
}

func (m *mockGameStream) Send(upd *pokerrpc.GameUpdate) error {
	select {
	case m.sentCh <- upd:
		return nil
	default:
		return nil
	}
}

// grpc.ServerStream
func (m *mockGameStream) SetHeader(metadata.MD) error  { return nil }
func (m *mockGameStream) SendHeader(metadata.MD) error { return nil }
func (m *mockGameStream) SetTrailer(metadata.MD)       {}
func (m *mockGameStream) Context() context.Context     { return m.ctx }
func (m *mockGameStream) SendMsg(interface{}) error    { return nil }
func (m *mockGameStream) RecvMsg(interface{}) error    { return nil }

func TestStartGameStream_DisconnectRemovesBucket(t *testing.T) {
	s := newBareServer()
	tbl := buildActiveHeadsUpTable(t, "stream-tbl")
	s.tables.Store(tbl.GetConfig().ID, tbl)

	// Two players connect to the same table stream
	p1, p2 := "p1", "p2"
	ms1 := newMockGameStream()
	ms2 := newMockGameStream()

	// Start streams concurrently
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_ = s.StartGameStream(&pokerrpc.StartGameStreamRequest{TableId: tbl.GetConfig().ID, PlayerId: p1}, ms1)
	}()
	go func() {
		defer wg.Done()
		_ = s.StartGameStream(&pokerrpc.StartGameStreamRequest{TableId: tbl.GetConfig().ID, PlayerId: p2}, ms2)
	}()

	// Expect initial updates
	select {
	case <-ms1.sentCh:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting initial update for p1")
	}
	select {
	case <-ms2.sentCh:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting initial update for p2")
	}

	// There should be a bucket registered for the table with count == 2
	bAny, ok := s.gameStreams.Load(tbl.GetConfig().ID)
	if !ok {
		t.Fatal("expected a bucket for table")
	}
	b := bAny.(*bucket)
	if n := int(b.count.Load()); n != 2 {
		t.Fatalf("expected stream count 2, got %d", n)
	}

	// Disconnect first stream
	ms1.cancel()
	time.Sleep(50 * time.Millisecond)
	if n := int(b.count.Load()); n != 1 {
		t.Fatalf("expected stream count 1 after first disconnect, got %d", n)
	}
	if _, still := s.gameStreams.Load(tbl.GetConfig().ID); !still {
		t.Fatal("bucket should remain until last stream disconnects")
	}

	// Disconnect second stream; bucket should be removed
	ms2.cancel()
	wg.Wait()
	time.Sleep(50 * time.Millisecond)
	if _, exists := s.gameStreams.Load(tbl.GetConfig().ID); exists {
		t.Fatal("expected bucket removed after last disconnect")
	}
}

func TestStartGameStream_AllowsWatcherAndHidesHands(t *testing.T) {
	s := newBareServer()
	tbl := buildActiveHeadsUpTable(t, "watch-stream-tbl")
	s.tables.Store(tbl.GetConfig().ID, tbl)

	resp, err := s.WatchTable(context.Background(), &pokerrpc.WatchTableRequest{
		PlayerId: "watcher",
		TableId:  tbl.GetConfig().ID,
	})
	if err != nil {
		t.Fatalf("watch table: %v", err)
	}
	if !resp.Success {
		t.Fatalf("watch table failed: %s", resp.Message)
	}

	ms := newMockGameStream()
	done := make(chan error, 1)
	go func() {
		done <- s.StartGameStream(&pokerrpc.StartGameStreamRequest{
			TableId:  tbl.GetConfig().ID,
			PlayerId: "watcher",
		}, ms)
	}()

	var upd *pokerrpc.GameUpdate
	select {
	case upd = <-ms.sentCh:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting initial update for watcher")
	}

	if upd == nil {
		t.Fatal("expected initial update for watcher")
	}
	if len(upd.Players) != 2 {
		t.Fatalf("expected 2 players, got %d", len(upd.Players))
	}
	for _, pl := range upd.Players {
		if len(pl.Hand) != 0 {
			t.Fatalf("watcher should not see hole cards for %s", pl.Id)
		}
	}

	ms.cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("stream returned error: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting watcher stream shutdown")
	}
}

func TestGetPlayerCurrentTable_ReturnsWatcherTable(t *testing.T) {
	s := newBareServer()
	tbl := buildActiveHeadsUpTable(t, "watch-current-table")
	s.tables.Store(tbl.GetConfig().ID, tbl)

	resp, err := s.WatchTable(context.Background(), &pokerrpc.WatchTableRequest{
		PlayerId: "watcher",
		TableId:  tbl.GetConfig().ID,
	})
	if err != nil {
		t.Fatalf("watch table: %v", err)
	}
	if !resp.Success {
		t.Fatalf("watch table failed: %s", resp.Message)
	}

	current, err := s.GetPlayerCurrentTable(context.Background(), &pokerrpc.GetPlayerCurrentTableRequest{
		PlayerId: "watcher",
	})
	if err != nil {
		t.Fatalf("get current table: %v", err)
	}
	if current.TableId != tbl.GetConfig().ID {
		t.Fatalf("expected watcher current table %q, got %q", tbl.GetConfig().ID, current.TableId)
	}
}

func TestActionRPCsRejectWatcher(t *testing.T) {
	s := newBareServer()
	tbl := buildActiveHeadsUpTable(t, "watcher-actions")
	s.tables.Store(tbl.GetConfig().ID, tbl)

	resp, err := s.WatchTable(context.Background(), &pokerrpc.WatchTableRequest{
		PlayerId: "watcher",
		TableId:  tbl.GetConfig().ID,
	})
	if err != nil {
		t.Fatalf("watch table: %v", err)
	}
	if !resp.Success {
		t.Fatalf("watch table failed: %s", resp.Message)
	}

	_, err = s.CheckBet(context.Background(), &pokerrpc.CheckBetRequest{
		PlayerId: "watcher",
		TableId:  tbl.GetConfig().ID,
	})
	if err == nil {
		t.Fatal("expected CheckBet to reject watcher")
	}

	_, err = s.CallBet(context.Background(), &pokerrpc.CallBetRequest{
		PlayerId: "watcher",
		TableId:  tbl.GetConfig().ID,
	})
	if err == nil {
		t.Fatal("expected CallBet to reject watcher")
	}

	_, err = s.FoldBet(context.Background(), &pokerrpc.FoldBetRequest{
		PlayerId: "watcher",
		TableId:  tbl.GetConfig().ID,
	})
	if err == nil {
		t.Fatal("expected FoldBet to reject watcher")
	}

	_, err = s.MakeBet(context.Background(), &pokerrpc.MakeBetRequest{
		PlayerId: "watcher",
		TableId:  tbl.GetConfig().ID,
		Amount:   50,
	})
	if err == nil {
		t.Fatal("expected MakeBet to reject watcher")
	}
}
