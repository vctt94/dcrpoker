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

	require.Equal(t, AT_TABLE_STATE, player.GetCurrentStateString())

	require.NoError(t, player.StartHandParticipation()) // move to IN_GAME
	reply := make(chan error, 1)
	player.handParticipation.Send(evFoldReq{Reply: reply})
	require.NoError(t, <-reply)

	require.Equal(t, FOLDED_STATE, player.GetCurrentStateString())

	assert.Equal(t, FOLDED_STATE, player.GetCurrentStateString())
}

func TestPlayerStateMachine_FoldStateTransition(t *testing.T) {
	player := NewPlayer("test-player", "Test Player", 1000)
	t.Cleanup(player.Close)

	// Move to IN_GAME and wait for FSM
	require.NoError(t, player.StartHandParticipation())
	require.Eventually(t, func() bool {
		return player.GetCurrentStateString() == IN_GAME_STATE
	}, 200*time.Millisecond, 10*time.Millisecond)

	// Fold and wait for FOLDED
	reply := make(chan error, 1)
	player.handParticipation.Send(evFoldReq{Reply: reply})
	require.NoError(t, <-reply)
	require.Equal(t, FOLDED_STATE, player.GetCurrentStateString())

	assert.Equal(t, FOLDED_STATE, player.GetCurrentStateString())
}

func TestPlayerStateMachine_FoldStatePersistence(t *testing.T) {
	player := NewPlayer("test-player", "Test Player", 1000)
	t.Cleanup(player.Close)

	// Start hand first (folding at table is usually invalid)
	require.NoError(t, player.StartHandParticipation())
	require.Eventually(t, func() bool {
		return player.GetCurrentStateString() == IN_GAME_STATE
	}, 200*time.Millisecond, 10*time.Millisecond)

	// Fold and confirm
	reply := make(chan error, 1)
	player.handParticipation.Send(evFoldReq{Reply: reply})
	require.NoError(t, <-reply)
	require.Equal(t, FOLDED_STATE, player.GetCurrentStateString())

	// Persistency check (no extra helper needed)
	for i := 0; i < 5; i++ {
		assert.Equalf(t, FOLDED_STATE, player.GetCurrentStateString(),
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
		return player.GetCurrentStateString() == IN_GAME_STATE
	}, 200*time.Millisecond, 10*time.Millisecond)

	// Fold
	reply := make(chan error, 1)
	player.handParticipation.Send(evFoldReq{Reply: reply})
	require.NoError(t, <-reply)
	require.Equal(t, FOLDED_STATE, player.GetCurrentStateString())

	// Verify player is in folded state
	assert.Equal(t, FOLDED_STATE, player.GetCurrentStateString())
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
			expectPreFoldState:   AT_TABLE_STATE,
			expectStateAfterFold: AT_TABLE_STATE,
		},
		{
			name: "Fold from IN_GAME",
			setup: func(p *Player) {
				p.StartHandParticipation()
			},
			expectPreFoldState:   IN_GAME_STATE,
			expectStateAfterFold: FOLDED_STATE,
		},
		{
			name: "Fold from ALL_IN (ignored)",
			setup: func(p *Player) {
				// Start hand, then go all-in via DeductBlind (mirrors real flow).
				p.balance = 100
				p.HandleStartHand()
				p.DeductBlind(100) // goes all-in: balance=0, currentBet=100
			},
			expectPreFoldState:   ALL_IN_STATE,
			expectStateAfterFold: ALL_IN_STATE,
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
			assert.Equal(t, tt.expectStateAfterFold == FOLDED_STATE, p.GetCurrentStateString() == FOLDED_STATE)
		})
	}
}

func TestResetForNewHand_ClearsFoldState(t *testing.T) {
	player := NewPlayer("test-player", "Test Player", 1000)
	t.Cleanup(player.Close)

	// Must start hand participation and be IN_GAME before we can fold
	require.NoError(t, player.StartHandParticipation())
	require.Eventually(t, func() bool {
		return player.GetCurrentStateString() == IN_GAME_STATE
	}, 200*time.Millisecond, 10*time.Millisecond)

	// Fold and wait for FOLDED
	reply := make(chan error, 1)
	player.handParticipation.Send(evFoldReq{Reply: reply})
	require.NoError(t, <-reply)
	require.Equal(t, FOLDED_STATE, player.GetCurrentStateString())
	assert.Equal(t, FOLDED_STATE, player.GetCurrentStateString())

	// Reset for new hand and start hand participation again
	player.ResetForNewHand(1000)
	require.NoError(t, player.StartHandParticipation())
	require.Eventually(t, func() bool {
		return player.GetCurrentStateString() == IN_GAME_STATE
	}, 200*time.Millisecond, 10*time.Millisecond)

	assert.NotEqual(t, FOLDED_STATE, player.GetCurrentStateString(), "Player should not be folded after new hand reset")
	assert.Equal(t, IN_GAME_STATE, player.GetCurrentStateString(), "Player should be in IN_GAME state after reset")
}

func TestDeductBlindIsIdempotentPerHand(t *testing.T) {
	player := NewPlayer("test-player", "Test Player", 1000)
	t.Cleanup(player.Close)

	require.NoError(t, player.HandleStartHand())

	first, err := player.DeductBlind(50)
	require.NoError(t, err)
	require.Equal(t, int64(50), first)
	require.Equal(t, int64(950), player.Balance())
	require.Equal(t, int64(50), player.CurrentBet())

	second, err := player.DeductBlind(50)
	require.NoError(t, err)
	require.Equal(t, int64(0), second)
	require.Equal(t, int64(950), player.Balance())
	require.Equal(t, int64(50), player.CurrentBet())
}

func TestDeductBlindResetsForNextHand(t *testing.T) {
	player := NewPlayer("test-player", "Test Player", 1000)
	t.Cleanup(player.Close)

	require.NoError(t, player.HandleStartHand())

	paid, err := player.DeductBlind(50)
	require.NoError(t, err)
	require.Equal(t, int64(50), paid)

	player.EndHandParticipation()
	require.Eventually(t, func() bool {
		return player.GetCurrentStateString() == AT_TABLE_STATE && player.CurrentBet() == 0
	}, 200*time.Millisecond, 10*time.Millisecond)

	require.NoError(t, player.HandleStartHand())

	paid, err = player.DeductBlind(50)
	require.NoError(t, err)
	require.Equal(t, int64(50), paid)
	require.Equal(t, int64(900), player.Balance())
	require.Equal(t, int64(50), player.CurrentBet())
}
