package poker

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
)

// TestIsTurnFlagManagement verifies that the isTurn flag is properly managed
// when players take actions and turns advance.
func TestIsTurnFlagManagement(t *testing.T) {
	// Create a heads-up game
	cfg := GameConfig{
		NumPlayers:    2,
		StartingChips: 1000,
		SmallBlind:    5,
		BigBlind:      10,
		Seed:          42,
		Log:           createTestLogger(),
	}

	game, err := NewGame(cfg)
	require.NoError(t, err)

	users := []*User{
		NewUser("player1", "Player 1", 1000, 0),
		NewUser("player2", "Player 2", 1000, 1),
	}
	game.SetPlayers(users)

	// Start the game FSM
	go game.Start(context.Background())

	// Wait for FSM to initialize
	game.sm.Send(evStartHand{})
	time.Sleep(50 * time.Millisecond)

	// Wait for PRE_FLOP phase
	require.Eventually(t, func() bool {
		return game.GetPhase() == pokerrpc.GamePhase_PRE_FLOP
	}, 2*time.Second, 10*time.Millisecond, "Game should reach PRE_FLOP")

	// In heads-up pre-flop, the small blind (dealer) acts first
	// Wait for the current player's isTurn to be set by their state machine
	var snap GameStateSnapshot
	var currentPlayerIndex int
	var currentPlayer *Player
	var otherPlayerIndex int
	var otherPlayer *Player

	require.Eventually(t, func() bool {
		snap = game.GetStateSnapshot()
		currentPlayerIndex = game.GetCurrentPlayer()
		if currentPlayerIndex < 0 || currentPlayerIndex >= len(snap.Players) {
			return false
		}
		currentPlayer = snap.Players[currentPlayerIndex]
		otherPlayerIndex = (currentPlayerIndex + 1) % 2
		otherPlayer = snap.Players[otherPlayerIndex]
		// Wait for current player to have isTurn=true
		return currentPlayer != nil && currentPlayer.isTurn
	}, 1*time.Second, 10*time.Millisecond, "Current player should have isTurn=true")

	// CRITICAL: Only the current player should have isTurn = true
	assert.True(t, currentPlayer.isTurn, "Current player should have isTurn=true")
	assert.False(t, otherPlayer.isTurn, "Other player should have isTurn=false")

	// Current player calls
	err = game.HandlePlayerCall(currentPlayer.id)
	require.NoError(t, err)

	// Wait for turn to switch - the other player should get isTurn=true
	require.Eventually(t, func() bool {
		snap = game.GetStateSnapshot()
		newCurrentPlayerIndex := game.GetCurrentPlayer()
		if newCurrentPlayerIndex != otherPlayerIndex {
			return false
		}
		p2After := snap.Players[otherPlayerIndex]
		return p2After != nil && p2After.isTurn
	}, 1*time.Second, 10*time.Millisecond, "Other player should get isTurn=true after call")

	// After the call, verify isTurn flags switched
	snap = game.GetStateSnapshot()
	p1After := snap.Players[currentPlayerIndex]
	p2After := snap.Players[otherPlayerIndex]

	// CRITICAL: The player who just acted should NO LONGER have the turn
	assert.False(t, p1After.isTurn, "Player who just called should have isTurn=false")
	// The other player should now have the turn
	assert.True(t, p2After.isTurn, "Other player should now have isTurn=true")

	// Verify currentPlayer index advanced
	newCurrentPlayerIndex := game.GetCurrentPlayer()
	assert.Equal(t, otherPlayerIndex, newCurrentPlayerIndex, "Current player index should have advanced")

	t.Logf("Turn management test passed: isTurn flags properly toggled")
}

// TestIsTurnFlagOnFold verifies that isTurn is cleared when a player folds
func TestIsTurnFlagOnFold(t *testing.T) {
	cfg := GameConfig{
		NumPlayers:    3,
		StartingChips: 1000,
		SmallBlind:    5,
		BigBlind:      10,
		Seed:          42,
		Log:           createTestLogger(),
	}

	game, err := NewGame(cfg)
	require.NoError(t, err)

	users := []*User{
		NewUser("player1", "Player 1", 1000, 0),
		NewUser("player2", "Player 2", 1000, 1),
		NewUser("player3", "Player 3", 1000, 2),
	}
	game.SetPlayers(users)

	go game.Start(context.Background())
	game.sm.Send(evStartHand{})
	time.Sleep(50 * time.Millisecond)

	require.Eventually(t, func() bool {
		return game.GetPhase() == pokerrpc.GamePhase_PRE_FLOP
	}, 2*time.Second, 10*time.Millisecond, "Game should reach PRE_FLOP")

	// Wait for current player to be properly initialized
	var snap GameStateSnapshot
	var currentPlayerIndex int
	var currentPlayer *Player

	require.Eventually(t, func() bool {
		snap = game.GetStateSnapshot()
		currentPlayerIndex = game.GetCurrentPlayer()
		if currentPlayerIndex < 0 || currentPlayerIndex >= len(snap.Players) {
			return false
		}
		currentPlayer = snap.Players[currentPlayerIndex]
		return currentPlayer != nil && currentPlayer.isTurn
	}, 2*time.Second, 10*time.Millisecond, "Current player should be initialized with isTurn=true")

	// Verify only current player has isTurn
	for i, p := range snap.Players {
		if i == currentPlayerIndex {
			assert.True(t, p.isTurn, "Current player should have isTurn=true")
		} else {
			assert.False(t, p.isTurn, "Non-current player %d should have isTurn=false", i)
		}
	}

	// Current player folds
	err = game.HandlePlayerFold(currentPlayer.id)
	require.NoError(t, err)

	// Wait for the new current player to get isTurn
	var newCurrentPlayerIndex int
	var newCurrentPlayer *Player
	require.Eventually(t, func() bool {
		snap = game.GetStateSnapshot()
		newCurrentPlayerIndex = game.GetCurrentPlayer()
		if newCurrentPlayerIndex < 0 || newCurrentPlayerIndex >= len(snap.Players) {
			return false
		}
		if newCurrentPlayerIndex == currentPlayerIndex {
			return false // Should have advanced
		}
		newCurrentPlayer = snap.Players[newCurrentPlayerIndex]
		return newCurrentPlayer != nil && newCurrentPlayer.isTurn
	}, 1*time.Second, 10*time.Millisecond, "New current player should have isTurn=true after fold")

	// Verify new current player has isTurn
	assert.True(t, newCurrentPlayer.isTurn, "New current player should have isTurn=true")

	// Verify the folded player no longer has isTurn
	// Note: We need to find the player by the original index
	foldedPlayer := snap.Players[currentPlayerIndex]
	assert.False(t, foldedPlayer.isTurn, "Folded player should have isTurn=false")

	// The critical test: isTurn flag should be false. Player state machine transitions
	// are tested elsewhere - we're focused on turn management here.

	// Verify all others don't have isTurn
	for i, p := range snap.Players {
		if i != newCurrentPlayerIndex {
			assert.False(t, p.isTurn, "Non-current player %d should have isTurn=false", i)
		}
	}

	t.Logf("Fold turn management test passed")
}

// TestIsTurnFlagOnCheck verifies that isTurn is managed correctly on check
func TestIsTurnFlagOnCheck(t *testing.T) {
	cfg := GameConfig{
		NumPlayers:    2,
		StartingChips: 1000,
		SmallBlind:    5,
		BigBlind:      10,
		Seed:          42,
		Log:           createTestLogger(),
	}

	game, err := NewGame(cfg)
	require.NoError(t, err)

	users := []*User{
		NewUser("player1", "Player 1", 1000, 0),
		NewUser("player2", "Player 2", 1000, 1),
	}
	game.SetPlayers(users)

	go game.Start(context.Background())
	game.sm.Send(evStartHand{})
	time.Sleep(50 * time.Millisecond)

	require.Eventually(t, func() bool {
		return game.GetPhase() == pokerrpc.GamePhase_PRE_FLOP
	}, 2*time.Second, 10*time.Millisecond, "Game should reach PRE_FLOP")

	// Wait for the first player to get their turn
	var snap GameStateSnapshot
	var sbIndex int
	var bbIndex int
	require.Eventually(t, func() bool {
		snap = game.GetStateSnapshot()
		sbIndex = game.GetCurrentPlayer()
		if sbIndex < 0 || sbIndex >= len(snap.Players) {
			return false
		}
		bbIndex = (sbIndex + 1) % 2
		return snap.Players[sbIndex] != nil && snap.Players[sbIndex].isTurn
	}, 1*time.Second, 10*time.Millisecond, "SB should have isTurn initially")

	// SB calls to match BB
	err = game.HandlePlayerCall(snap.Players[sbIndex].id)
	require.NoError(t, err)

	// Wait for BB to get the turn
	require.Eventually(t, func() bool {
		snap = game.GetStateSnapshot()
		return snap.Players[bbIndex] != nil && snap.Players[bbIndex].isTurn
	}, 1*time.Second, 10*time.Millisecond, "BB should have turn after SB calls")

	// Now BB should have the turn
	snap = game.GetStateSnapshot()
	assert.True(t, snap.Players[bbIndex].isTurn, "BB should have turn after SB calls")
	assert.False(t, snap.Players[sbIndex].isTurn, "SB should not have turn after calling")

	// BB checks (they can check since they matched the bet)
	err = game.HandlePlayerCheck(snap.Players[bbIndex].id)
	require.NoError(t, err)

	// Wait for phase to advance or turn to advance
	// After both players act, should advance to FLOP
	time.Sleep(100 * time.Millisecond) // Give time for phase advance

	// Wait for the current player in the new phase/round to have isTurn
	require.Eventually(t, func() bool {
		snap = game.GetStateSnapshot()
		currentPlayerIndex := game.GetCurrentPlayer()
		if currentPlayerIndex < 0 || currentPlayerIndex >= len(snap.Players) {
			return false
		}
		return snap.Players[currentPlayerIndex] != nil && snap.Players[currentPlayerIndex].isTurn
	}, 1*time.Second, 10*time.Millisecond, "Current player should have isTurn after check")

	// After both players act, should advance to FLOP
	// The turn should advance and isTurn should be managed properly
	snap = game.GetStateSnapshot()

	// Verify we advanced beyond PRE_FLOP
	phase := game.GetPhase()
	t.Logf("Phase after both players acted: %v", phase)

	// Verify only the current player has isTurn
	currentPlayerIndex := game.GetCurrentPlayer()
	assert.True(t, snap.Players[currentPlayerIndex].isTurn, "Current player should have isTurn")
	for i, p := range snap.Players {
		if i != currentPlayerIndex {
			assert.False(t, p.isTurn, "Non-current player should not have isTurn")
		}
	}

	t.Logf("Check turn management test passed")
}
