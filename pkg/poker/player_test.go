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

	require.NoError(t, player.StartHandParticipation()) // move to IN_GAME
	reply := make(chan error, 1)
	player.handParticipation.Send(evFoldReq{Reply: reply})
	require.NoError(t, <-reply)

	require.Equal(t, "FOLDED", player.GetCurrentStateString())

	assert.Equal(t, "FOLDED", player.GetCurrentStateString())
}

func TestPlayerStateMachine_FoldStateTransition(t *testing.T) {
	player := NewPlayer("test-player", "Test Player", 1000)
	t.Cleanup(player.Close)

	// Move to IN_GAME and wait for FSM
	require.NoError(t, player.StartHandParticipation())
	require.Eventually(t, func() bool {
		return player.GetCurrentStateString() == "IN_GAME"
	}, 200*time.Millisecond, 10*time.Millisecond)

	// Fold and wait for FOLDED
	reply := make(chan error, 1)
	player.handParticipation.Send(evFoldReq{Reply: reply})
	require.NoError(t, <-reply)
	require.Equal(t, "FOLDED", player.GetCurrentStateString())

	assert.Equal(t, "FOLDED", player.GetCurrentStateString())
}

func TestPlayerStateMachine_FoldStatePersistence(t *testing.T) {
	player := NewPlayer("test-player", "Test Player", 1000)
	t.Cleanup(player.Close)

	// Start hand first (folding at table is usually invalid)
	require.NoError(t, player.StartHandParticipation())
	require.Eventually(t, func() bool {
		return player.GetCurrentStateString() == "IN_GAME"
	}, 200*time.Millisecond, 10*time.Millisecond)

	// Fold and confirm
	reply := make(chan error, 1)
	player.handParticipation.Send(evFoldReq{Reply: reply})
	require.NoError(t, <-reply)
	require.Equal(t, "FOLDED", player.GetCurrentStateString())

	// Persistency check (no extra helper needed)
	for i := 0; i < 5; i++ {
		assert.Equalf(t, "FOLDED", player.GetCurrentStateString(),
			"dispatch %d: should remain FOLDED", i+1)
		time.Sleep(5 * time.Millisecond)
	}
}

// TestPlayerStateMachine_FoldTransition verifies that a player can transition from IN_GAME to FOLDED state.
func TestPlayerStateMachine_FoldTransition(t *testing.T) {
	player := NewPlayer("test-player", "Test Player", 1000)
	t.Cleanup(player.Close)

	// Start a hand first
	require.NoError(t, player.StartHandParticipation())
	require.Eventually(t, func() bool {
		return player.GetCurrentStateString() == "IN_GAME"
	}, 200*time.Millisecond, 10*time.Millisecond)

	// Fold
	reply := make(chan error, 1)
	player.handParticipation.Send(evFoldReq{Reply: reply})
	require.NoError(t, <-reply)
	require.Equal(t, "FOLDED", player.GetCurrentStateString())

	// Verify player is in folded state
	assert.Equal(t, "FOLDED", player.GetCurrentStateString())
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
				p.StartHandParticipation()
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
				p.HandleStartHand() // Use HandleStartHand to check all-in condition
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
			// Give a bit more time for state machine to initialize
			time.Sleep(50 * time.Millisecond)
			require.Eventually(t, func() bool {
				return p.GetCurrentStateString() == tt.expectPreFoldState
			}, 500*time.Millisecond, 10*time.Millisecond, "did not reach pre-fold state %s (got %s)", tt.expectPreFoldState, p.GetCurrentStateString())

			// 2) Send fold (only if hand participation is active)
			if p.handParticipation != nil {
				reply := make(chan error, 1)
				p.handParticipation.Send(evFoldReq{Reply: reply})
				<-reply
			}

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

	// Must start hand participation and be IN_GAME before we can fold
	require.NoError(t, player.StartHandParticipation())
	require.Eventually(t, func() bool {
		return player.GetCurrentStateString() == "IN_GAME"
	}, 200*time.Millisecond, 10*time.Millisecond)

	// Fold and wait for FOLDED
	reply := make(chan error, 1)
	player.handParticipation.Send(evFoldReq{Reply: reply})
	require.NoError(t, <-reply)
	require.Equal(t, "FOLDED", player.GetCurrentStateString())
	assert.Equal(t, "FOLDED", player.GetCurrentStateString())

	// Reset for new hand and start hand participation again
	require.NoError(t, player.ResetForNewHand(1000))
	require.NoError(t, player.StartHandParticipation())
	require.Eventually(t, func() bool {
		return player.GetCurrentStateString() == "IN_GAME"
	}, 200*time.Millisecond, 10*time.Millisecond)

	assert.NotEqual(t, "FOLDED", player.GetCurrentStateString(), "Player should not be folded after new hand reset")
	assert.Equal(t, "IN_GAME", player.GetCurrentStateString(), "Player should be in IN_GAME state after reset")
}
