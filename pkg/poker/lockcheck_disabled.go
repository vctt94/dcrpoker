//go:build !lockcheck

package poker

import "sync"

type RWLock struct{ sync.RWMutex }
type Mu struct{ sync.Mutex }

func (m *RWLock) RLock()   { m.RWMutex.RLock() }
func (m *RWLock) RUnlock() { m.RWMutex.RUnlock() }
func (m *RWLock) Lock()    { m.RWMutex.Lock() }
func (m *RWLock) Unlock()  { m.RWMutex.Unlock() }

func (m *Mu) Lock()   { m.Mutex.Lock() }
func (m *Mu) Unlock() { m.Mutex.Unlock() }

func mustHeld(_ *RWLock) {}
