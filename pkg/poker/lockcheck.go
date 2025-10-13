//go:build lockcheck

package poker

import (
	"sync"
	"sync/atomic"
)

type RWLock struct {
	sync.RWMutex
	r int32
	w int32
}

func (m *RWLock) RLock()   { m.RWMutex.RLock(); atomic.AddInt32(&m.r, 1) }
func (m *RWLock) RUnlock() { atomic.AddInt32(&m.r, -1); m.RWMutex.RUnlock() }
func (m *RWLock) Lock()    { m.RWMutex.Lock(); atomic.AddInt32(&m.w, 1) }
func (m *RWLock) Unlock()  { atomic.AddInt32(&m.w, -1); m.RWMutex.Unlock() }

type Mu struct {
	sync.Mutex
	w int32
}

func (m *Mu) Lock()   { m.Mutex.Lock(); atomic.AddInt32(&m.w, 1) }
func (m *Mu) Unlock() { atomic.AddInt32(&m.w, -1); m.Mutex.Unlock() }

func mustHeld(mu *RWLock) {
	if atomic.LoadInt32(&mu.r) == 0 && atomic.LoadInt32(&mu.w) == 0 {
		panic("lockcheck: RW lock not held")
	}
}
func mustHeldForWrite(mu *RWLock) {
	if atomic.LoadInt32(&mu.w) == 0 {
		panic("lockcheck: RW write lock not held")
	}
}
func mustHeldMutex(mu *Mu) {
	if atomic.LoadInt32(&mu.w) == 0 {
		panic("lockcheck: mutex not held")
	}
}
