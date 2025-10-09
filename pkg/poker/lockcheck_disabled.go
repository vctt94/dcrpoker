//go:build !lockcheck

package poker

import "sync"

type RWLock struct{ sync.RWMutex }
type Mu struct{ sync.Mutex }

func mustHeld(_ *RWLock)         {}
func mustHeldForWrite(_ *RWLock) {}
func mustHeldMutex(_ *Mu)        {}
