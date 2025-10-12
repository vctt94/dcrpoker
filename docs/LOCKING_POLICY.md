# Locking Policy & Concurrency Guidelines

**Date:** 2025-10-09  
**Status:** Active  
**Goal:** Prevent deadlocks and race conditions through strict lock ordering and clear conventions.

---

## 1. Lock Hierarchy & Ordering

To prevent deadlocks, **all code MUST acquire locks in this strict order**:

```
Table.mu → Game.mu → Player.mu
```

**IMPORTANT:** PotManager no longer has its own mutex. It is **FSM-only data** owned by the Game FSM.
All PotManager operations MUST be called with `Game.mu` held (on the Game FSM thread).

**Rules:**
- ✅ **ALLOWED**: Acquire locks left-to-right in the hierarchy
- ❌ **FORBIDDEN**: Acquire locks in reverse order or skip levels and go back
- ⚠️  **NEVER** hold a higher-level lock while attempting to acquire a lower-level lock
- ⚠️  **PotManager methods** assume `Game.mu` is held and will assert this in debug builds

### Examples

**✅ CORRECT:**
```go
// Acquire Table lock first, then Game lock
t.mu.Lock()
game := t.game
t.mu.Unlock()

if game != nil {
    game.mu.Lock()
    // ... do work ...
    game.mu.Unlock()
}
```

**❌ WRONG (Deadlock risk):**
```go
// NEVER: Acquiring Game lock then Table lock
game.mu.Lock()
defer game.mu.Unlock()

t.mu.Lock()  // DEADLOCK: violates hierarchy!
defer t.mu.Unlock()
```

**✅ CORRECT (Game FSM owns PotManager - no separate lock):**
```go
// PotManager methods require Game.mu held (Game FSM thread invariant)
func (g *Game) AddPlayerBet(playerIndex int, amount int64) {
    g.mu.Lock()
    defer g.mu.Unlock()
    
    // PotManager has no lock - it's FSM-only data owned by Game
    g.potManager.addBet(playerIndex, amount, g.players)
}
```

---

## 2. Exported vs Unexported Methods

We follow a strict **naming and locking convention**:

| Method Type | Naming | Locking Responsibility | Comment Required |
|-------------|---------|------------------------|------------------|
| **Exported** | `func (x *X) Method()` | **Acquires lock internally** | Optional |
| **Unexported** | `func (x *X) methodLocked()` | **Assumes lock already held** | `// Requires x.mu held` |

### Naming Convention

- **Exported methods** (`Method`) are **safe to call from anywhere** and handle their own locking
- **Unexported methods** (`methodLocked`) are **internal helpers** that assume the caller holds the lock
- Use suffix `Locked` for unexported methods that require a lock to be held

### Examples

```go
// Exported: safe to call, acquires lock
func (g *Game) GetPhase() pokerrpc.GamePhase {
    g.mu.RLock()
    defer g.mu.RUnlock()
    return g.phase
}

// Unexported: requires g.mu held
// Requires: g.mu held
func (g *Game) getPhaseUnsafe() pokerrpc.GamePhase {
    return g.phase
}

// Exported: safe to call, acquires lock
func (g *Game) MaybeCompleteBettingRound() {
    g.mu.Lock()
    defer g.mu.Unlock()
    g.maybeCompleteBettingRoundLocked()
}

// Unexported: requires g.mu held
// Requires: g.mu held
func (g *Game) maybeCompleteBettingRoundLocked() {
    // ... implementation ...
}
```

---

## 3. Lock Scope Best Practices

### 3.1 Minimize Lock Duration

**✅ DO:**
```go
func (g *Game) ProcessAction() {
    // Snapshot under lock
    g.mu.RLock()
    phase := g.phase
    player := g.currentPlayer
    g.mu.RUnlock()
    
    // Expensive work outside lock
    result := heavyComputation(phase, player)
    
    // Update under lock
    g.mu.Lock()
    g.result = result
    g.mu.Unlock()
}
```

**❌ DON'T:**
```go
func (g *Game) ProcessAction() {
    g.mu.Lock()
    defer g.mu.Unlock()
    
    // WRONG: Holding lock during expensive operation
    result := heavyComputation(g.phase, g.currentPlayer)
    g.result = result
}
```

### 3.2 Never Block on Channels While Holding Locks

**❌ WRONG (Deadlock risk):**
```go
func (p *Player) StartTurn() {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    reply := make(chan error, 1)
    p.handParticipation.Send(evStartTurn{Reply: reply})
    <-reply  // DEADLOCK: waiting while holding lock!
}
```

**✅ CORRECT:**
```go
func (p *Player) StartTurn() {
    // Snapshot under lock
    p.mu.RLock()
    hp := p.handParticipation
    p.mu.RUnlock()
    
    // Send without holding lock
    if hp != nil {
        reply := make(chan error, 1)
        hp.Send(evStartTurn{Reply: reply})
        <-reply
    }
}
```

### 3.3 Release Lock Before Calling Into Other Objects

**✅ DO:**
```go
func (t *Table) Close() {
    // Snapshot references under lock
    t.mu.Lock()
    game := t.game
    sm := t.sm
    t.game = nil
    t.sm = nil
    t.mu.Unlock()
    
    // Call cleanup without holding lock
    if game != nil { game.Close() }
    if sm != nil { sm.Stop() }
}
```

---

## 4. Debug Assertions (Build Tag: `lockcheck`)

For development, we provide optional **runtime lock assertions** enabled with `-tags lockcheck`:

```go
//go:build lockcheck

package poker

import "sync"

// MustHeld panics if the mutex is not held by the caller
func MustHeld(mu *sync.RWMutex) {
    // Try to lock; if we can, it wasn't held
    if mu.TryLock() {
        mu.Unlock()
        panic("lock not held")
    }
}
```

**Usage:**
```go
// Requires: g.mu held
func (g *Game) maybeCompleteBettingRoundLocked() {
    mustHeld(&g.mu)  // Only active with -tags lockcheck
    // ... implementation ...
}
```

**Testing:**
```bash
go test -tags lockcheck ./pkg/poker
```

---

## 5. Common Patterns

### 5.1 Snapshot Pattern (Read-only access)

```go
func (g *Game) GetStateSnapshot() GameStateSnapshot {
    g.mu.RLock()
    
    // Create copies of data we need
    phase := g.phase
    dealer := g.dealer
    pot := g.potManager.getTotalPot()
    
    playersCopy := make([]*PlayerSnapshot, len(g.players))
    for i, p := range g.players {
        if p != nil {
            playersCopy[i] = p.Snapshot()  // p.Snapshot() handles its own lock
        }
    }
    
    g.mu.RUnlock()
    
    return GameStateSnapshot{
        Phase:   phase,
        Dealer:  dealer,
        Pot:     pot,
        Players: playersCopy,
    }
}
```

### 5.2 FSM-Only Data Pattern (PotManager)

**PotManager is FSM-only data with no mutex:**

```go
// REQUIRES: g.mu held (Game FSM thread)
func (pm *PotManager) addBet(playerIndex int, amount int64, players []*Player) {
    // Caller must hold g.mu - we trust this invariant
    // Can safely read player state since caller holds Game.mu
    foldStatus := make([]bool, len(players))
    for i, p := range players {
        if p != nil {
            foldStatus[i] = (p.getCurrentStateString() == "FOLDED")
        }
    }
    
    // Modify pot state directly (no lock needed - owned by Game FSM)
    pm.currentBets[playerIndex] += amount
    pm.totalBets[playerIndex] += amount
    pm.rebuildPotsIncremental(players, foldStatus)
}
```

**Game FSM directly modifies player balances (no synchronous Credit() calls):**

```go
// REQUIRES: g.mu held (Game FSM thread)
func (pm *PotManager) distributePots(players []*Player, foldStatus []bool, handValues []*HandValue) error {
    for _, pot := range pm.pots {
        // ... determine winners ...
        
        // Directly modify player.balance (Game FSM owns authoritative balance)
        for _, idx := range winners {
            players[idx].balance += pot.amount
        }
    }
    return nil
}
```

### 5.3 Two-Phase Update (Atomic swap)

```go
func (g *Game) ResetForNewHand() {
    g.mu.Lock()
    defer g.mu.Unlock()
    
    // Phase 1: Create new state
    newPM := NewPotManager(len(g.players))
    newDeck := NewDeck(g.rng)
    
    // Phase 2: Atomic swap
    g.potManager = newPM
    g.deck = newDeck
    g.phase = pokerrpc.GamePhase_NEW_HAND_DEALING
}
```

---

## 6. Lock Audit Checklist

Before committing code that touches locks:

- [ ] All locks acquired in correct hierarchy order?
- [ ] No channel sends/receives while holding locks?
- [ ] No calls to external objects (e.g., logger, FSM) while holding locks?
- [ ] Exported methods handle their own locking?
- [ ] Unexported methods documented with "Requires: x.mu held"?
- [ ] No `time.Sleep()` or blocking I/O under locks?
- [ ] Lock scope minimized to critical section only?

---

## 7. Testing for Deadlocks

### 7.1 Race Detector

**Always run tests with race detector:**
```bash
go test -race ./pkg/poker/...
go test -race -count=100 ./pkg/poker -run TestTableClose
```

### 7.2 Stress Testing

For concurrency-heavy tests, run multiple iterations:
```bash
go test -race -count=100 ./pkg/poker -run 'Test.*Concurrent'
```

### 7.3 Goroutine Leak Detection

Use `go.uber.org/goleak` in tests:
```go
func TestTableClose(t *testing.T) {
    defer goleak.VerifyNone(t)
    
    table := NewTable(config)
    // ... test ...
    table.Close()
}
```

---

## 8. Common Pitfalls

### ❌ Pitfall 1: Lock Inversion
```go
// Goroutine A:
t.mu.Lock()
g.mu.Lock()  // OK: follows hierarchy

// Goroutine B:
g.mu.Lock()
t.mu.Lock()  // DEADLOCK: violates hierarchy
```

### ❌ Pitfall 2: Holding Lock Across Channel Operations
```go
p.mu.Lock()
p.fsm.Send(event)  // DEADLOCK: FSM might need p.mu
<-reply
p.mu.Unlock()
```

### ❌ Pitfall 3: Recursive Lock Acquisition
```go
func (g *Game) methodA() {
    g.mu.Lock()
    defer g.mu.Unlock()
    g.methodB()  // methodB also tries to lock g.mu → DEADLOCK
}
```

**Solution:** Use `methodBLocked()` that assumes lock held:
```go
func (g *Game) methodA() {
    g.mu.Lock()
    defer g.mu.Unlock()
    g.methodBLocked()  // Assumes g.mu already held
}
```

---

## 9. References

- [Go Blog: Share Memory By Communicating](https://go.dev/blog/codelab-share)
- [Effective Go: Concurrency](https://go.dev/doc/effective_go#concurrency)
- [Go Memory Model](https://go.dev/ref/mem)

---

**Enforcement:** This policy is enforced through:
1. Code reviews
2. `-race` detector in CI
3. Optional `-tags lockcheck` assertions
4. Automated linter checks (future: custom linter for lock order)


