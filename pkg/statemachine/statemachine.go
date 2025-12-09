package statemachine

import (
	"context"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
)

// StateFn is Rob Pike–style: it owns its loop and returns the next state (or nil to stop).
// The input channel carries arbitrary events (typically small structs).
type StateFn[T any] func(*T, <-chan any) StateFn[T]

// Machine runs a single goroutine that owns all transitions and entity mutations.
type Machine[T any] struct {
	entity    *T
	inbox     chan any
	closed    chan struct{}
	stateSnap atomic.Value // stores StateFn[T]
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	closeOnce sync.Once
}

// New creates a machine with an initial state and buffered inbox.
func New[T any](entity *T, initial StateFn[T], inboxSize int) *Machine[T] {
	m := &Machine[T]{
		entity: entity,
		inbox:  make(chan any, inboxSize),
		closed: make(chan struct{}),
	}
	m.stateSnap.Store(initial) // typed-nil is fine
	return m
}

// Start launches the Pike loop.
func (m *Machine[T]) Start(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, m.cancel = context.WithCancel(ctx)
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		state, _ := m.stateSnap.Load().(StateFn[T])
		for state != nil {
			next := state(m.entity, m.inbox) // single owner => no races
			m.stateSnap.Store(next)
			state = next
			select {
			default:
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Stop cancels and waits for the loop to exit.
func (m *Machine[T]) Stop() {
	m.closeOnce.Do(func() {
		// First, signal senders to stop enqueuing before we close the inbox to
		// avoid the send-after-close race reported by the race detector.
		close(m.closed)
		close(m.inbox) // unblock receivers with ok=false
		if m.cancel != nil {
			m.cancel() // optional, keep for outside listeners
		}
	})
	m.wg.Wait()
}

// Send enqueues an event (may block if inbox is full).
func (m *Machine[T]) Send(ev any) {
	select {
	case <-m.closed:
		return // drop silently if machine is stopping/stopped
	default:
		m.inbox <- ev
	}
}

// TrySend enqueues without blocking; returns false if full.
func (m *Machine[T]) TrySend(ev any) bool {
	select {
	case <-m.closed:
		return false
	case m.inbox <- ev:
		return true
	default:
		return false
	}
}

// Current returns the current state function (snapshot).
func (m *Machine[T]) Current() StateFn[T] {
	v := m.stateSnap.Load()
	if v == nil {
		return nil
	}
	fn, _ := v.(StateFn[T])
	return fn
}

// NameOf (optional) is handy for logging transitions.
func NameOf[T any](fn StateFn[T]) string {
	if fn == nil {
		return "<nil>"
	}
	return runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
}
