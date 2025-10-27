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
