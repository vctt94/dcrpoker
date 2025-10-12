//go:build lockcheck

package poker

import (
	"testing"
	"time"
)

// Test_LockOrdering_Assertions tests that mustHeld panics when lock is not held
func Test_LockOrdering_Assertions(t *testing.T) {
	t.Run("mustHeld panics when lock not held", func(t *testing.T) {
		var mu RWLock

		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic when lock not held, but didn't panic")
			}
		}()

		mustHeld(&mu) // Should panic
	})

	t.Run("mustHeld succeeds when write lock held", func(t *testing.T) {
		var mu RWLock
		mu.Lock()
		defer mu.Unlock()

		// Should not panic
		mustHeld(&mu)
	})

	t.Run("mustHeld succeeds when read lock held", func(t *testing.T) {
		var mu RWLock
		mu.RLock()
		defer mu.RUnlock()

		// Should not panic (TryLock fails when read lock is held)
		mustHeld(&mu)
	})
}

// Test_BettingRound_MustHoldLock tests that maybeCompleteBettingRound requires g.mu
func Test_BettingRound_MustHoldLock(t *testing.T) {
	cfg := GameConfig{
		NumPlayers:       2,
		SmallBlind:       10,
		BigBlind:         20,
		StartingChips:    1000,
		Log:              createTestLogger(),
		AutoAdvanceDelay: 1 * time.Second,
	}

	game, err := NewGame(cfg)
	if err != nil {
		t.Fatalf("Failed to create game: %v", err)
	}
	defer game.Close()

	// Try calling maybeCompleteBettingRound without holding lock - should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when calling maybeCompleteBettingRound without lock, but didn't panic")
		}
	}()

	game.maybeCompleteBettingRound() // Should panic
}

// Test_HandleMethods_HoldLock tests that handle* methods require g.mu
func Test_HandleMethods_HoldLock(t *testing.T) {
	cfg := GameConfig{
		NumPlayers:       2,
		SmallBlind:       10,
		BigBlind:         20,
		StartingChips:    1000,
		Log:              createTestLogger(),
		AutoAdvanceDelay: 1 * time.Second,
	}

	game, err := NewGame(cfg)
	if err != nil {
		t.Fatalf("Failed to create game: %v", err)
	}
	defer game.Close()

	// Add a player
	player := NewPlayer("player1", "Alice", 1000)
	defer player.Close()
	game.players = []*Player{player}

	t.Run("handlePlayerFold requires lock", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic when calling handlePlayerFold without lock")
			}
		}()
		_ = game.handlePlayerFold("player1") // Should panic
	})

	t.Run("handlePlayerCall requires lock", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic when calling handlePlayerCall without lock")
			}
		}()
		_ = game.handlePlayerCall("player1") // Should panic
	})

	t.Run("handlePlayerCheck requires lock", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic when calling handlePlayerCheck without lock")
			}
		}()
		_ = game.handlePlayerCheck("player1") // Should panic
	})

	t.Run("handlePlayerBet requires lock", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic when calling handlePlayerBet without lock")
			}
		}()
		_ = game.handlePlayerBet("player1", 100) // Should panic
	})
}
