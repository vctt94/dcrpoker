package poker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests ensure the state machine preserves fold status correctly.
func TestPlayerStateMachine_FoldRegression(t *testing.T) {
	player := NewPlayer("test-player", "Test Player", 1000)
	t.Cleanup(player.Close)

	require.Equal(t, "AT_TABLE", player.GetCurrentStateString())

	player.sm.Send(evStartHand{}) // move to IN_GAME
	player.sm.Send(evFold{})

	require.Eventually(t, func() bool {
		return player.GetCurrentStateString() == "FOLDED"
	}, 200*time.Millisecond, 10*time.Millisecond)

	assert.Equal(t, "FOLDED", player.GetCurrentStateString())
}

func TestPlayerStateMachine_FoldStateTransition(t *testing.T) {
	player := NewPlayer("test-player", "Test Player", 1000)
	t.Cleanup(player.Close)

	// Move to IN_GAME and wait for FSM
	player.sm.Send(evStartHand{})
	require.Eventually(t, func() bool {
		return player.GetCurrentStateString() == "IN_GAME"
	}, 200*time.Millisecond, 10*time.Millisecond)

	// Fold and wait for FOLDED
	player.sm.Send(evFold{})
	require.Eventually(t, func() bool {
		return player.GetCurrentStateString() == "FOLDED"
	}, 200*time.Millisecond, 10*time.Millisecond)

	assert.Equal(t, "FOLDED", player.GetCurrentStateString())
}

func TestPlayerStateMachine_FoldStatePersistence(t *testing.T) {
	player := NewPlayer("test-player", "Test Player", 1000)
	t.Cleanup(player.Close)

	// Start hand first (folding at table is usually invalid)
	player.sm.Send(evStartHand{})
	require.Eventually(t, func() bool {
		return player.GetCurrentStateString() == "IN_GAME"
	}, 200*time.Millisecond, 10*time.Millisecond)

	// Fold and confirm
	player.sm.Send(evFold{})
	require.Eventually(t, func() bool {
		return player.GetCurrentStateString() == "FOLDED"
	}, 200*time.Millisecond, 10*time.Millisecond)

	// Persistency check (no extra helper needed)
	for i := 0; i < 5; i++ {
		assert.Equalf(t, "FOLDED", player.GetCurrentStateString(),
			"dispatch %d: should remain FOLDED", i+1)
		time.Sleep(5 * time.Millisecond)
	}
}

func TestPlayerStateMachine_UnfoldTransition(t *testing.T) {
	player := NewPlayer("test-player", "Test Player", 1000)
	t.Cleanup(player.Close)

	// Start a hand first
	player.sm.Send(evStartHand{})
	require.Eventually(t, func() bool {
		return player.GetCurrentStateString() == "IN_GAME"
	}, 200*time.Millisecond, 10*time.Millisecond)

	// Fold
	player.sm.Send(evFold{})
	require.Eventually(t, func() bool {
		return player.GetCurrentStateString() == "FOLDED"
	}, 200*time.Millisecond, 10*time.Millisecond)
	assert.Equal(t, "FOLDED", player.GetCurrentStateString())

	// End hand → back to AT_TABLE
	player.sm.Send(evEndHand{})
	require.Eventually(t, func() bool {
		return player.GetCurrentStateString() == "AT_TABLE"
	}, 200*time.Millisecond, 10*time.Millisecond)

	assert.False(t, player.GetCurrentStateString() == "FOLDED", "Player should not be folded")
	assert.Equal(t, "AT_TABLE", player.GetCurrentStateString(), "Player should be back in AT_TABLE state")
}

func TestPlayerStateMachine_FoldFromDifferentStates(t *testing.T) {
	type tc struct {
		name                 string
		setup                func(p *Player)
		expectPreFoldState   string // state we must reach before sending evFold
		expectStateAfterFold string
	}

	tests := []tc{
		{
			name: "Fold from AT_TABLE (ignored)",
			setup: func(p *Player) {
				// already AT_TABLE; do nothing
			},
			expectPreFoldState:   "AT_TABLE",
			expectStateAfterFold: "AT_TABLE",
		},
		{
			name: "Fold from IN_GAME",
			setup: func(p *Player) {
				p.sm.Send(evStartHand{})
			},
			expectPreFoldState:   "IN_GAME",
			expectStateAfterFold: "FOLDED",
		},
		{
			name: "Fold from ALL_IN (ignored)",
			setup: func(p *Player) {
				// Prepare all-in BEFORE starting the hand so the loop sees it quickly.
				p.balance = 0
				p.currentBet = 100
				p.sm.Send(evStartHand{})
				// FSM will check derived condition (balance==0) on next event
			},
			expectPreFoldState:   "ALL_IN",
			expectStateAfterFold: "ALL_IN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPlayer("test-player", "Test Player", 1000)
			t.Cleanup(p.Close)

			// 1) Drive to the pre-fold state and wait (Pike loop is async)
			tt.setup(p)
			require.Eventually(t, func() bool {
				return p.GetCurrentStateString() == tt.expectPreFoldState
			}, 300*time.Millisecond, 10*time.Millisecond, "did not reach pre-fold state %s (got %s)", tt.expectPreFoldState, p.GetCurrentStateString())

			// 2) Send fold
			p.sm.Send(evFold{})

			// 3) Wait for expected post-fold state
			require.Eventually(t, func() bool {
				return p.GetCurrentStateString() == tt.expectStateAfterFold
			}, 300*time.Millisecond, 10*time.Millisecond, "after fold, wanted %s; got %s", tt.expectStateAfterFold, p.GetCurrentStateString())

			assert.Equal(t, tt.expectStateAfterFold, p.GetCurrentStateString())
			// “has folded” is equivalent to being in FOLDED state:
			assert.Equal(t, tt.expectStateAfterFold == "FOLDED", p.GetCurrentStateString() == "FOLDED")
		})
	}
}

func TestResetForNewHand_ClearsFoldState(t *testing.T) {
	player := NewPlayer("test-player", "Test Player", 1000)
	t.Cleanup(player.Close)

	// Must be IN_GAME before we can fold
	player.sm.Send(evStartHand{})
	require.Eventually(t, func() bool {
		return player.GetCurrentStateString() == "IN_GAME"
	}, 200*time.Millisecond, 10*time.Millisecond)

	// Fold and wait for FOLDED
	player.sm.Send(evFold{})
	require.Eventually(t, func() bool {
		return player.GetCurrentStateString() == "FOLDED"
	}, 200*time.Millisecond, 10*time.Millisecond)
	assert.Equal(t, "FOLDED", player.GetCurrentStateString())

	// Reset for new hand (sends evStartHand internally) and wait for IN_GAME
	require.NoError(t, player.ResetForNewHand(1000))
	require.Eventually(t, func() bool {
		return player.GetCurrentStateString() == "IN_GAME"
	}, 200*time.Millisecond, 10*time.Millisecond)

	assert.NotEqual(t, "FOLDED", player.GetCurrentStateString(), "Player should not be folded after new hand reset")
	assert.Equal(t, "IN_GAME", player.GetCurrentStateString(), "Player should be in IN_GAME state after reset")
}

func TestTryFold_AllowsFoldWhenNotAllIn(t *testing.T) {
	player := NewPlayer("test-player", "Test Player", 1000)
	t.Cleanup(player.Close)

	// Start the hand and wait for IN_GAME
	player.sm.Send(evStartHand{})
	require.Eventually(t, func() bool {
		return player.GetCurrentStateString() == "IN_GAME"
	}, 200*time.Millisecond, 10*time.Millisecond)

	// TryFold should succeed (not all-in)
	success, err := player.TryFold()
	require.NoError(t, err, "TryFold should succeed when not all-in")
	assert.True(t, success, "TryFold should succeed when not all-in")

	// Wait for FOLDED since FSM processes events asynchronously
	require.Eventually(t, func() bool {
		return player.GetCurrentStateString() == "FOLDED"
	}, 200*time.Millisecond, 10*time.Millisecond)

	assert.Equal(t, "FOLDED", player.GetCurrentStateString(), "Player should be in FOLDED state")
}
func TestTryFold_PreventsFoldWhenAllIn(t *testing.T) {
	player := NewPlayer("test-player", "Test Player", 1000)
	t.Cleanup(player.Close)

	// Prepare all-in state: zero stack and some money already in
	player.balance = 0
	player.currentBet = 100

	// Start the hand and wait until we're in-hand
	player.sm.Send(evStartHand{})
	require.Eventually(t, func() bool {
		s := player.GetCurrentStateString()
		return s == "IN_GAME" || s == "ALL_IN"
	}, 300*time.Millisecond, 10*time.Millisecond)

	// FSM checks derived condition (balance==0 && currentBet>0 → ALL_IN) on each loop iteration
	// Wait for FSM to detect the all-in state
	require.Eventually(t, func() bool {
		return player.GetCurrentStateString() == "ALL_IN"
	}, 300*time.Millisecond, 10*time.Millisecond, "Player should transition to ALL_IN")

	// Now TryFold should be rejected for all-in players
	success, err := player.TryFold()
	require.Error(t, err, "TryFold should fail when all-in")
	assert.False(t, success, "TryFold should fail when all-in")
	assert.Equal(t, "ALL_IN", player.GetCurrentStateString(), "Player should remain in ALL_IN state")
}
