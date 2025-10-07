package poker

import (
    "testing"
)

// Verify that GetStateSnapshot copies blind flags for players.
func TestGameStateSnapshotCopiesBlindFlags(t *testing.T) {
    cfg := GameConfig{
        NumPlayers:    2,
        StartingChips: 100,
        SmallBlind:    10,
        BigBlind:      20,
        Seed:          1,
        Log:           createTestLogger(),
    }

    g, err := NewGame(cfg)
    if err != nil {
        t.Fatalf("NewGame error: %v", err)
    }

    users := []*User{
        NewUser("p1", "p1", 0, 0),
        NewUser("p2", "p2", 0, 1),
    }
    g.SetPlayers(users)

    // Manually set positions under locks to simulate a pre-deal setup.
    g.mu.Lock()
    g.dealer = 0
    if p := g.players[0]; p != nil {
        p.mu.Lock()
        p.isDealer = true
        p.isSmallBlind = true
        p.isBigBlind = false
        p.mu.Unlock()
    }
    if p := g.players[1]; p != nil {
        p.mu.Lock()
        p.isDealer = false
        p.isSmallBlind = false
        p.isBigBlind = true
        p.mu.Unlock()
    }
    g.mu.Unlock()

    snap := g.GetStateSnapshot()
    if got, want := len(snap.Players), 2; got != want {
        t.Fatalf("expected %d players in snapshot, got %d", want, got)
    }

    if !snap.Players[0].isDealer || !snap.Players[0].isSmallBlind || snap.Players[0].isBigBlind {
        t.Fatalf("player 0 flags not copied: dealer=%v sb=%v bb=%v",
            snap.Players[0].isDealer, snap.Players[0].isSmallBlind, snap.Players[0].isBigBlind)
    }
    if snap.Players[1].isDealer || snap.Players[1].isSmallBlind || !snap.Players[1].isBigBlind {
        t.Fatalf("player 1 flags not copied: dealer=%v sb=%v bb=%v",
            snap.Players[1].isDealer, snap.Players[1].isSmallBlind, snap.Players[1].isBigBlind)
    }
}

