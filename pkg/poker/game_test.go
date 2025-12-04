package poker

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/decred/slog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
)

// createTestLogger creates a simple logger for testing
func createTestLogger() slog.Logger {
	backend := slog.NewBackend(os.Stderr)
	log := backend.Logger("test")
	log.SetLevel(slog.LevelError) // Reduce noise in tests
	return log
}

func TestNewGame(t *testing.T) {
	cfg := GameConfig{
		NumPlayers:       2,
		StartingChips:    1000, // Set to 1000 to match the expected balance
		Seed:             42,   // Use a fixed seed for deterministic testing
		Log:              createTestLogger(),
		AutoAdvanceDelay: 1 * time.Second,
	}

	game, err := NewGame(cfg)
	require.NoError(t, err)

	// After refactor, game starts with empty players slice
	// Table manages players and calls SetPlayers
	if len(game.players) != 0 {
		t.Errorf("Expected 0 players initially, got %d", len(game.players))
	}

	// Create test users and set them in the game
	users := []*User{
		NewUser("player1", nil, nil),
		NewUser("player2", nil, nil),
	}
	game.SetPlayers(users)

	// Now check that players were created correctly
	if len(game.players) != 2 {
		t.Errorf("Expected 2 players after SetPlayers, got %d", len(game.players))
	}

	// Check initial player state
	for i, player := range game.players {
		if player.balance != 1000 {
			t.Errorf("Player %d: Expected 1000 balance, got %d", i, player.balance)
		}
		if player.GetCurrentStateString() == FOLDED_STATE {
			t.Errorf("Player %d: Expected not folded", i)
		}
		if player.currentBet != 0 {
			t.Errorf("Player %d: Expected 0 bet, got %d", i, player.currentBet)
		}
	}

	// Check deck is properly initialized
	if game.deck == nil {
		t.Error("Expected deck to be initialized")
	}
	if game.deck.Size() != 52 {
		t.Errorf("Expected deck size 52, got %d", game.deck.Size())
	}
}

func TestNewGameErrorsOnInvalidPlayers(t *testing.T) {
	cfg := GameConfig{
		NumPlayers:       1,
		StartingChips:    100,
		Log:              createTestLogger(),
		AutoAdvanceDelay: 1 * time.Second,
	}

	_, err := NewGame(cfg)
	if err == nil {
		t.Error("Expected error with < 2 players")
	}

	expectedErr := "poker: must have at least 2 players"
	if err.Error() != expectedErr {
		t.Errorf("Expected error '%s', got '%s'", expectedErr, err.Error())
	}
}

func TestNewGameErrorsOnMissingAutoAdvanceDelay(t *testing.T) {
	cfg := GameConfig{
		NumPlayers:       2,
		StartingChips:    100,
		Log:              createTestLogger(),
		AutoAdvanceDelay: 0, // Invalid - must be > 0
	}

	_, err := NewGame(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "AutoAdvanceDelay must be set")
}

func TestDealCards(t *testing.T) {
	cfg := GameConfig{
		NumPlayers:       2,
		StartingChips:    100,
		Seed:             42,
		Log:              createTestLogger(),
		AutoAdvanceDelay: 1 * time.Second,
	}

	game, err := NewGame(cfg)
	require.NoError(t, err)

	// Create test users and set them in the game
	users := []*User{
		NewUser("player1", nil, nil),
		NewUser("player2", nil, nil),
	}
	game.SetPlayers(users)

	// Initialize Hand for this test
	playerIDs := make([]string, len(game.players))
	for i, p := range game.players {
		playerIDs[i] = p.ID()
	}
	game.currentHand = NewHand(playerIDs)

	// Deal cards manually for testing (since DealCards was removed)
	for _, player := range game.players {
		for i := 0; i < 2; i++ {
			card, ok := game.deck.Draw()
			if !ok {
				t.Fatalf("Failed to draw card from deck")
			}
			if err := game.currentHand.DealCardToPlayer(player.ID(), card); err != nil {
				t.Fatalf("Failed to deal card to player: %v", err)
			}
		}
	}

	// Check each player has 2 cards
	for i, player := range game.players {
		cards := game.currentHand.GetPlayerCards(player.ID(), player.ID())
		if len(cards) != 2 {
			t.Errorf("Player %d: Expected 2 cards, got %d", i, len(cards))
		}
	}

	// Check deck has correct number of cards remaining
	expectedRemaining := 52 - (2 * len(game.players))
	if game.deck.Size() != expectedRemaining {
		t.Errorf("Expected %d cards remaining, got %d", expectedRemaining, game.deck.Size())
	}
}

func TestCommunityCards(t *testing.T) {
	cfg := GameConfig{
		NumPlayers:       2,
		Seed:             42,
		Log:              createTestLogger(),
		AutoAdvanceDelay: 1 * time.Second,
	}

	game, err := NewGame(cfg)
	require.NoError(t, err)

	// Create test users and set them in the game
	users := []*User{
		NewUser("player1", nil, nil),
		NewUser("player2", nil, nil),
	}
	game.SetPlayers(users)

	// Initialize Hand for this test
	playerIDs := make([]string, len(game.players))
	for i, p := range game.players {
		playerIDs[i] = p.ID()
	}
	game.currentHand = NewHand(playerIDs)

	// Deal cards manually for testing (since DealCards was removed)
	for _, player := range game.players {
		for i := 0; i < 2; i++ {
			card, ok := game.deck.Draw()
			if !ok {
				t.Fatalf("Failed to draw card from deck")
			}
			if err := game.currentHand.DealCardToPlayer(player.ID(), card); err != nil {
				t.Fatalf("Failed to deal card to player: %v", err)
			}
		}
	}

	// Check initial community cards
	if len(game.communityCards) != 0 {
		t.Errorf("Expected 0 community cards initially, got %d", len(game.communityCards))
	}

	// Deal flop
	game.StateFlop()
	if len(game.communityCards) != 3 {
		t.Errorf("Expected 3 community cards after flop, got %d", len(game.communityCards))
	}

	// Deal turn
	game.StateTurn()
	if len(game.communityCards) != 4 {
		t.Errorf("Expected 4 community cards after turn, got %d", len(game.communityCards))
	}

	// Deal river
	game.StateRiver()
	if len(game.communityCards) != 5 {
		t.Errorf("Expected 5 community cards after river, got %d", len(game.communityCards))
	}
}

func TestShowdown(t *testing.T) {
	// Create a game with 2 players
	cfg := GameConfig{
		NumPlayers:       2,
		Seed:             42,
		Log:              createTestLogger(),
		AutoAdvanceDelay: 1 * time.Second,
	}

	game, err := NewGame(cfg)
	require.NoError(t, err)

	// Create test users and set them in the game
	users := []*User{
		NewUser("player1", nil, nil), // Start with 0 balance for clean test
		NewUser("player2", nil, nil),
	}
	game.SetPlayers(users)

	// Set up player hands manually
	player1 := game.players[0]
	player2 := game.players[1]

	// Initialize Hand for this test
	game.currentHand = NewHand([]string{player1.ID(), player2.ID()})

	// Player 1 has a pair of Aces
	game.currentHand.DealCardToPlayer(player1.ID(), Card{suit: Hearts, value: Ace})
	game.currentHand.DealCardToPlayer(player1.ID(), Card{suit: Spades, value: Ace})

	// Player 2 has King-Queen
	game.currentHand.DealCardToPlayer(player2.ID(), Card{suit: Hearts, value: King})
	game.currentHand.DealCardToPlayer(player2.ID(), Card{suit: Spades, value: Queen})

	// Set community cards: 2-5-7-9-Jack (no help for either player)
	game.communityCards = []Card{
		{suit: Clubs, value: Two},
		{suit: Diamonds, value: Five},
		{suit: Hearts, value: Seven},
		{suit: Spades, value: Nine},
		{suit: Clubs, value: Jack},
	}

	// Set up pot
	game.potManager = NewPotManager(2)
	game.potManager.addBet(0, 50, game.players) // Player 1 bet 50
	game.potManager.addBet(1, 50, game.players) // Player 2 bet 50

	// Run the showdown
	_, err = game.HandleShowdown()
	if err != nil {
		t.Fatalf("HandleShowdown() error = %v", err)
	}

	// Player 1 should win with pair of Aces
	if player1.Balance() != 100 {
		t.Errorf("Expected player 1 to win with pot of 100, got %d", player1.Balance())
	}

	// Player 2 should not win anything
	if player2.Balance() != 0 {
		t.Errorf("Expected player 2 to not win anything, got %d", player2.Balance())
	}

	// Check hand descriptions
	if !strings.Contains(player1.HandDescription(), "Pair") {
		t.Errorf("Expected pair description, got %s", player1.HandDescription())
	}

	if !strings.Contains(player2.HandDescription(), "High Card") {
		t.Errorf("Expected high card description, got %s", player2.HandDescription())
	}
}

// Verify that showdown winners carry the evaluated HandRank (not the default/high-card zero value).
func TestShowdownHandRankPropagates(t *testing.T) {
	cfg := GameConfig{
		NumPlayers:       2,
		Seed:             1,
		StartingChips:    1000,
		SmallBlind:       10,
		BigBlind:         20,
		AutoAdvanceDelay: 1 * time.Second,
		Log:              createTestLogger(),
	}

	game, err := NewGame(cfg)
	require.NoError(t, err)

	users := []*User{
		NewUser("p1", nil, nil),
		NewUser("p2", nil, nil),
	}
	game.SetPlayers(users)

	// Build a hand: board A A 10 7 2, p1 = 9c 6s (pair of aces), p2 = 8d 2s (two pair).
	game.currentHand = NewHand([]string{"p1", "p2"})
	require.NoError(t, game.currentHand.DealCardToPlayer("p1", Card{suit: Clubs, value: Nine}))
	require.NoError(t, game.currentHand.DealCardToPlayer("p1", Card{suit: Spades, value: Six}))
	require.NoError(t, game.currentHand.DealCardToPlayer("p2", Card{suit: Diamonds, value: Eight}))
	require.NoError(t, game.currentHand.DealCardToPlayer("p2", Card{suit: Spades, value: Two}))

	game.communityCards = []Card{
		{suit: Spades, value: Ten},
		{suit: Spades, value: Seven},
		{suit: Diamonds, value: Ace},
		{suit: Hearts, value: Two},
		{suit: Hearts, value: Ace},
	}
	game.phase = pokerrpc.GamePhase_RIVER

	// Set up pot contributions.
	game.potManager = NewPotManager(2)
	game.potManager.addBet(0, 40, game.players)
	game.potManager.addBet(1, 40, game.players)

	result, err := game.HandleShowdown()
	require.NoError(t, err)

	require.Len(t, result.WinnerInfo, 1, "expected single winner")
	w := result.WinnerInfo[0]
	assert.Equal(t, "p2", w.PlayerId, "expected p2 to win with two pair")
	assert.Equal(t, pokerrpc.HandRank_TWO_PAIR, w.HandRank, "winner HandRank should reflect evaluated hand")
}

func TestTieBreakerShowdown(t *testing.T) {
	// Create a game with 3 players
	cfg := GameConfig{
		NumPlayers:       3,
		Seed:             42,
		Log:              createTestLogger(),
		AutoAdvanceDelay: 1 * time.Second,
	}

	game, err := NewGame(cfg)
	require.NoError(t, err)

	// Create test users and set them in the game
	users := []*User{
		NewUser("player1", nil, nil), // Start with 0 balance for clean test
		NewUser("player2", nil, nil),
		NewUser("player3", nil, nil),
	}
	game.SetPlayers(users)

	// Set up player hands manually
	player1 := game.players[0]
	player2 := game.players[1]
	player3 := game.players[2]

	// Initialize Hand for this test
	game.currentHand = NewHand([]string{player1.ID(), player2.ID(), player3.ID()})

	// All players have a pair of Aces but with different kickers
	game.currentHand.DealCardToPlayer(player1.ID(), Card{suit: Hearts, value: Ace})
	game.currentHand.DealCardToPlayer(player1.ID(), Card{suit: Spades, value: Ace})

	game.currentHand.DealCardToPlayer(player2.ID(), Card{suit: Clubs, value: Ace})
	game.currentHand.DealCardToPlayer(player2.ID(), Card{suit: Diamonds, value: Ace})

	game.currentHand.DealCardToPlayer(player3.ID(), Card{suit: Hearts, value: King})
	game.currentHand.DealCardToPlayer(player3.ID(), Card{suit: Spades, value: King})

	// Set community cards: 2-5-7-9-Jack
	game.communityCards = []Card{
		{suit: Clubs, value: Two},
		{suit: Diamonds, value: Five},
		{suit: Hearts, value: Seven},
		{suit: Spades, value: Nine},
		{suit: Clubs, value: Jack},
	}

	// Initialize hand participation for all players first
	require.NoError(t, player1.StartHandParticipation())
	require.NoError(t, player2.StartHandParticipation())
	require.NoError(t, player3.StartHandParticipation())

	// Mark player 3 as folded
	reply := make(chan error, 1)
	player3.handParticipation.Send(evFoldReq{Reply: reply})
	<-reply

	// Set up pot
	game.potManager = NewPotManager(3)
	game.potManager.addBet(0, 50, game.players) // Player 1 bet 50
	game.potManager.addBet(1, 50, game.players) // Player 2 bet 50
	// Player 3 folded, no bet

	// Run the showdown
	_, err = game.HandleShowdown()
	if err != nil {
		t.Fatalf("HandleShowdown() error = %v", err)
	}

	// Players 1 and 2 should tie and split the pot (50 each)
	if player1.Balance() != 50 {
		t.Errorf("Expected player 1 to win 50 (half pot), got %d", player1.Balance())
	}

	if player2.Balance() != 50 {
		t.Errorf("Expected player 2 to win 50 (half pot), got %d", player2.Balance())
	}

	// Player 3 should not win anything (folded)
	if player3.Balance() != 0 {
		t.Errorf("Expected player 3 to not win anything (folded), got %d", player3.Balance())
	}
}

// Split pot: Board makes the best five-card hand for both players.
func TestSplitPotShowdown(t *testing.T) {
	cfg := GameConfig{
		NumPlayers:       2,
		Seed:             1,
		Log:              createTestLogger(),
		AutoAdvanceDelay: 1 * time.Second,
	}
	game, err := NewGame(cfg)
	require.NoError(t, err)

	users := []*User{
		NewUser("p1", nil, nil),
		NewUser("p2", nil, nil),
	}
	game.SetPlayers(users)

	// Initialize Hand for this test
	game.currentHand = NewHand([]string{game.players[0].ID(), game.players[1].ID()})

	// Force hands that don't improve beyond board
	game.currentHand.DealCardToPlayer(game.players[0].ID(), Card{suit: Hearts, value: Two})
	game.currentHand.DealCardToPlayer(game.players[0].ID(), Card{suit: Clubs, value: Three})
	game.currentHand.DealCardToPlayer(game.players[1].ID(), Card{suit: Diamonds, value: Four})
	game.currentHand.DealCardToPlayer(game.players[1].ID(), Card{suit: Spades, value: Five})

	// Board: Straight 10-J-Q-K-A (broadway) split; use 10,J,Q,K,A in mixed suits
	game.communityCards = []Card{
		{suit: Hearts, value: Ten},
		{suit: Clubs, value: Jack},
		{suit: Diamonds, value: Queen},
		{suit: Spades, value: King},
		{suit: Hearts, value: Ace},
	}

	game.potManager = NewPotManager(2)
	game.potManager.addBet(0, 50, game.players)
	game.potManager.addBet(1, 50, game.players)

	// Resolve showdown
	game.mu.Lock()
	res, err := game.handleShowdown()
	game.mu.Unlock()
	require.NoError(t, err)
	require.NotNil(t, res)

	// Both players should split 100 → 50 each
	if game.players[0].Balance() != 50 {
		t.Fatalf("p1 expected 50, got %d", game.players[0].Balance())
	}
	if game.players[1].Balance() != 50 {
		t.Fatalf("p2 expected 50, got %d", game.players[1].Balance())
	}
}

// Side pot: p3 all-in short, p1/p2 create side pot; winners differ per pot.
func TestSidePotShowdown(t *testing.T) {
	cfg := GameConfig{
		NumPlayers:       3,
		Seed:             1,
		Log:              createTestLogger(),
		AutoAdvanceDelay: 1 * time.Second,
	}
	game, err := NewGame(cfg)
	require.NoError(t, err)

	users := []*User{
		NewUser("p1", nil, nil),
		NewUser("p2", nil, nil),
		NewUser("p3", nil, nil),
	}
	game.SetPlayers(users)

	// Set balances to simulate all-in thresholds via bets recorded in pot manager
	// We control through potManager directly for test.
	game.potManager = NewPotManager(3)

	// Bets: p3 short 30, p1 50, p2 50 → main 90 (all eligible), side 40 (p1,p2)
	game.potManager.addBet(0, 50, game.players)
	game.potManager.addBet(1, 50, game.players)
	game.potManager.addBet(2, 30, game.players)

	// Hand strengths: p3 wins main, p1 wins side
	// Initialize hand participation state machines first
	require.NoError(t, game.players[0].HandleStartHand())
	require.NoError(t, game.players[1].HandleStartHand())
	require.NoError(t, game.players[2].HandleStartHand())

	// Give explicit evaluated values via EvaluateHand semantics
	hv3, err := EvaluateHand([]Card{{suit: Hearts, value: Five}, {suit: Clubs, value: Five}}, []Card{{suit: Diamonds, value: Five}, {suit: Spades, value: Two}, {suit: Hearts, value: Three}, {suit: Clubs, value: Nine}, {suit: Diamonds, value: Queen}}) // trips
	if err != nil {
		t.Fatalf("EvaluateHand() error = %v", err)
	}
	hv1, err := EvaluateHand([]Card{{suit: Hearts, value: Ace}, {suit: Clubs, value: Ace}}, []Card{{suit: Diamonds, value: King}, {suit: Spades, value: Two}, {suit: Hearts, value: Three}, {suit: Clubs, value: Nine}, {suit: Diamonds, value: Queen}}) // pair aces
	if err != nil {
		t.Fatalf("EvaluateHand() error = %v", err)
	}
	hv2, err := EvaluateHand([]Card{{suit: Hearts, value: Ten}, {suit: Clubs, value: Nine}}, []Card{{suit: Diamonds, value: King}, {suit: Spades, value: Two}, {suit: Hearts, value: Three}, {suit: Clubs, value: Nine}, {suit: Diamonds, value: Queen}}) // pair nines
	if err != nil {
		t.Fatalf("EvaluateHand() error = %v", err)
	}

	// Set hand values using lock-protected access
	game.players[0].mu.Lock()
	game.players[0].handValue = &hv1
	game.players[0].mu.Unlock()

	game.players[1].mu.Lock()
	game.players[1].handValue = &hv2
	game.players[1].mu.Unlock()

	game.players[2].mu.Lock()
	game.players[2].handValue = &hv3
	game.players[2].mu.Unlock()

	// Pots are automatically built on each bet, no need to call BuildPotsFromTotals

	// Pre-compute player state
	foldStatus := make([]bool, len(game.players))
	handValues := make([]*HandValue, len(game.players))
	for i, p := range game.players {
		if p != nil {
			p.mu.RLock()
			foldStatus[i] = (p.GetCurrentStateString() == FOLDED_STATE)
			handValues[i] = p.handValue
			p.mu.RUnlock()
		}
	}

	// Distribute pots - now requires game.mu held (Game FSM thread invariant)
	game.mu.Lock()
	game.potManager.distributePots(game.players)
	game.mu.Unlock()

	// Expected: p3 gets 90 (main), p1 gets 40 (side)
	if game.players[2].Balance() != 90 {
		t.Fatalf("p3 expected 90 from main pot, got %d", game.players[2].Balance())
	}
	if game.players[0].Balance() != 40 {
		t.Fatalf("p1 expected 40 from side pot, got %d", game.players[0].Balance())
	}
	if game.players[1].Balance() != 0 {
		t.Fatalf("p2 expected 0, got %d", game.players[1].Balance())
	}
}

func TestAutoStartOnNewHandStarted(t *testing.T) {
	cfg := GameConfig{
		NumPlayers:       2,
		StartingChips:    1000,
		SmallBlind:       10,
		BigBlind:         20,
		AutoStartDelay:   10 * time.Millisecond,
		Log:              createTestLogger(),
		AutoAdvanceDelay: 1 * time.Second,
	}
	game, err := NewGame(cfg)
	require.NoError(t, err)

	// Set players so there are enough to start
	users := []*User{
		NewUser("p1", nil, nil),
		NewUser("p2", nil, nil),
	}
	game.SetPlayers(users)

	// Set up event channel to receive auto-start events
	eventCh := make(chan GameEvent, 10)
	game.SetTableEventChannel(eventCh)

	// Trigger the timer
	game.ScheduleAutoStart()

	// Wait for auto-start event with timeout
	select {
	case event := <-eventCh:
		if event.Type != GameEventAutoStartTriggered {
			t.Fatalf("expected GameEventAutoStartTriggered, got %v", event.Type)
		}
		// ok
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for GameEventAutoStartTriggered")
	}
}

// Ensure that when multiple players are all-in pre-flop, the game
// automatically deals remaining community cards and performs showdown
// without panicking.
func TestPreFlopAllInAutoDealShowdown(t *testing.T) {
	cfg := GameConfig{
		NumPlayers:       2,
		StartingChips:    100,
		SmallBlind:       10,
		BigBlind:         20,
		Seed:             1,
		AutoStartDelay:   0,
		TimeBank:         0,
		Log:              createTestLogger(),
		AutoAdvanceDelay: 1 * time.Second,
	}
	game, err := NewGame(cfg)
	require.NoError(t, err)

	users := []*User{
		NewUser("p1", nil, nil),
		NewUser("p2", nil, nil),
	}
	game.SetPlayers(users)

	// Initialize Hand for this test
	game.currentHand = NewHand([]string{"p1", "p2"})
	// Deal cards to both players
	for i := 0; i < 2; i++ {
		card, _ := game.deck.Draw()
		game.currentHand.DealCardToPlayer("p1", card)
		card, _ = game.deck.Draw()
		game.currentHand.DealCardToPlayer("p2", card)
	}

	// Simulate pre-flop all-in by both players with some bets recorded
	game.phase = pokerrpc.GamePhase_PRE_FLOP
	game.communityCards = nil
	game.potManager = NewPotManager(2)

	// Put some chips in to form a pot
	game.potManager.addBet(0, 50, game.players)
	game.potManager.addBet(1, 50, game.players)

	// Initialize hand participation for both players first
	require.NoError(t, game.players[0].StartHandParticipation())
	require.NoError(t, game.players[1].StartHandParticipation())

	// Mark both players as all-in and not folded
	// Set up all-in state (balance=0, currentBet>0)
	game.players[0].balance = 0
	game.players[0].currentBet = 100
	game.players[1].balance = 0
	game.players[1].currentBet = 100

	// Send events to trigger state machine to detect all-in condition
	game.players[0].handParticipation.Send(evStartTurn{})
	game.players[1].handParticipation.Send(evStartTurn{})

	// Wait for all-in state transitions
	require.Eventually(t, func() bool {
		fmt.Println("player 0 state:", game.players[0].GetCurrentStateString())
		return game.players[0].GetCurrentStateString() == ALL_IN_STATE
	}, 200*time.Millisecond, 10*time.Millisecond)
	require.Eventually(t, func() bool {
		return game.players[1].GetCurrentStateString() == ALL_IN_STATE
	}, 200*time.Millisecond, 10*time.Millisecond)
	game.players[0].lastAction = time.Now()
	game.players[1].lastAction = time.Now()

	// Call showdown; should auto-deal to 5 community cards and not error
	game.mu.Lock()
	res, err := game.handleShowdown()
	game.mu.Unlock()
	require.NoError(t, err)
	require.NotNil(t, res)

	if got := len(game.communityCards); got != 5 {
		t.Fatalf("expected 5 community cards to be dealt, got %d", got)
	}

	// Total pot equals sum of bets (100)
	require.EqualValues(t, int64(100), res.TotalPot)
}

// Ensure auto-start sends event even when players have short stacks (>0 chips).
func TestAutoStartAllowsShortStackAllIn(t *testing.T) {
	cfg := GameConfig{
		NumPlayers:       2,
		StartingChips:    0,
		SmallBlind:       10,
		BigBlind:         20,
		AutoStartDelay:   10 * time.Millisecond,
		Log:              createTestLogger(),
		AutoAdvanceDelay: 1 * time.Second,
	}
	game, err := NewGame(cfg)
	require.NoError(t, err)

	users := []*User{
		NewUser("short", nil, nil),
		NewUser("deep", nil, nil),
	}
	game.SetPlayers(users)

	// Simulate balances: short < big blind, deep >> big blind
	game.players[0].balance = 10   // short stack
	game.players[1].balance = 1990 // deep stack

	// Set up event channel to receive auto-start events
	eventCh := make(chan GameEvent, 10)
	game.SetTableEventChannel(eventCh)

	game.ScheduleAutoStart()

	select {
	case event := <-eventCh:
		if event.Type != GameEventAutoStartTriggered {
			t.Fatalf("expected GameEventAutoStartTriggered, got %v", event.Type)
		}
		// ok - auto-start triggered even with short-stacked player
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected auto-start to trigger with short-stacked player")
	}
}

// Verify that a short-stacked caller only contributes what they have, and
// their HasBet is NOT force-set to currentBet.
func TestCallShortStackAllInDoesNotForceMatchCurrentBet(t *testing.T) {
	cfg := GameConfig{
		NumPlayers:       2,
		StartingChips:    15, // SB starts with 15, will post 10 as SB, leaving 5
		SmallBlind:       10,
		BigBlind:         20,
		Log:              createTestLogger(),
		AutoAdvanceDelay: 1 * time.Second,
	}
	g, err := NewGame(cfg)
	require.NoError(t, err)

	users := []*User{
		NewUser("sb", nil, nil),
		NewUser("bb", nil, nil),
	}
	g.SetPlayers(users)

	// Override SB balance to simulate the short stack scenario
	// After SetPlayers, manually adjust balance to create test scenario
	g.players[0].SetBalance(15) // Will post 10 as SB, leaving 5
	g.players[1].SetBalance(1000)

	// Start the game FSM
	go g.Start(context.Background())
	g.sm.Send(evStartHand{})
	time.Sleep(50 * time.Millisecond)

	// Wait for PRE_FLOP phase
	require.Eventually(t, func() bool {
		return g.GetPhase() == pokerrpc.GamePhase_PRE_FLOP
	}, 2*time.Second, 10*time.Millisecond, "Game should reach PRE_FLOP")

	// Wait for SB to have their turn (in heads-up, SB acts first pre-flop)
	var snap GameStateSnapshot
	var sbIndex int
	require.Eventually(t, func() bool {
		snap = g.GetStateSnapshot()
		sbIndex = g.GetCurrentPlayer()
		if sbIndex < 0 || sbIndex >= len(snap.Players) {
			return false
		}
		return snap.Players[sbIndex].IsTurn
	}, 1*time.Second, 10*time.Millisecond, "SB should have isTurn")

	// Verify SB posted blind and has only 5 left
	sbPlayer := snap.Players[sbIndex]
	t.Logf("Before call - SB: balance=%d, currentBet=%d, state=%s",
		sbPlayer.Balance, sbPlayer.CurrentBet, sbPlayer.StateString)

	// SB tries to call to match BB (20) but can only contribute 5 more (total 15)
	err = g.HandlePlayerCall(sbPlayer.ID)
	require.NoError(t, err)

	// Wait a moment for FSM to process
	time.Sleep(50 * time.Millisecond)

	// Get updated snapshot
	snap = g.GetStateSnapshot()
	sbPlayerAfter := snap.Players[sbIndex]

	t.Logf("After call - SB: balance=%d, currentBet=%d, state=%s",
		sbPlayerAfter.Balance, sbPlayerAfter.CurrentBet, sbPlayerAfter.StateString)

	// Verify SB went all-in
	assert.Equal(t, int64(0), sbPlayerAfter.Balance, "SB should have 0 balance after going all-in")
	assert.Equal(t, int64(15), sbPlayerAfter.CurrentBet, "SB should have currentBet of 15 (10 SB + 5 call)")
	assert.Contains(t, sbPlayerAfter.StateString, ALL_IN_STATE, "SB should be in ALL_IN state")

	// With auto-advance enabled, when one player is all-in and can't match,
	// the betting round completes and advances to FLOP, resetting currentBet to 0
	// Wait for FLOP to be reached
	require.Eventually(t, func() bool {
		return g.GetPhase() == pokerrpc.GamePhase_FLOP
	}, 2*time.Second, 10*time.Millisecond, "Game should advance to FLOP")

	// After advancing to FLOP, currentBet is reset to 0
	assert.Equal(t, int64(0), g.GetCurrentBet(), "Table currentBet should be 0 after advancing to FLOP")
}

// Verifies that a timeout-triggered fold completes the round to SHOWDOWN and auto-starts a new hand,
// preventing the game from getting stuck in SHOWDOWN.
func TestTimeoutCompletesShowdownAndAutoStarts(t *testing.T) {
	tbl := newTestTable(t, 2, 2, 5, 10, 1000)
	user1, err := tbl.AddNewUser("p1", nil)
	require.NoError(t, err)
	user2, err := tbl.AddNewUser("p2", nil)
	require.NoError(t, err)
	_ = user1.SendReady()
	_ = user2.SendReady()
	require.True(t, tbl.CheckAllPlayersReady())
	require.NoError(t, tbl.StartGame())

	// Wait until: game exists, phase == PRE_FLOP, current player exists and is facing a bet
	require.Eventually(t, func() bool {
		g := tbl.GetGame()
		if g == nil || g.GetPhase() != pokerrpc.GamePhase_PRE_FLOP {
			return false
		}
		cp := g.GetCurrentPlayerObject()
		if cp == nil {
			return false
		}
		return cp.currentBet < g.GetCurrentBet()
	}, 2*time.Second, 10*time.Millisecond)

	g := tbl.GetGame()
	require.NotNil(t, g)
	oldRound := g.GetRound()

	// Expire timebank of the current player directly on the live object
	cur := g.GetCurrentPlayerObject()
	require.NotNil(t, cur)
	// Avoid data race: mutate lastAction under player lock
	cur.mu.Lock()
	cur.lastAction = time.Now().Add(-2 * tbl.GetConfig().TimeBank)
	cur.mu.Unlock()

	// Trigger timeout handling
	g.TriggerTimebankExpiredFor(cur.ID())

	// First it should reach SHOWDOWN (uncontested fold-win)
	require.Eventually(t, func() bool {
		return g.GetPhase() == pokerrpc.GamePhase_SHOWDOWN
	}, 1*time.Second, 10*time.Millisecond)

	// Then auto-start should kick in and advance to a new hand (round increments and PRE_FLOP again)
	require.Eventually(t, func() bool {
		return g.GetRound() > oldRound && g.GetPhase() == pokerrpc.GamePhase_PRE_FLOP
	}, 2*time.Second, 10*time.Millisecond)
}

// TestIsTurnFlagManagement verifies that the isTurn flag is properly managed
// when players take actions and turns advance.
func TestIsTurnFlagManagement(t *testing.T) {
	// Create a heads-up game
	cfg := GameConfig{
		NumPlayers:       2,
		StartingChips:    1000,
		SmallBlind:       5,
		BigBlind:         10,
		Seed:             42,
		Log:              createTestLogger(),
		AutoAdvanceDelay: 1 * time.Second,
	}

	game, err := NewGame(cfg)
	require.NoError(t, err)

	users := []*User{
		NewUser("player1", nil, nil),
		NewUser("player2", nil, nil),
	}
	for _, u := range users {
		u.SendReady()
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

	// Wait for all players to have hand participation initialized
	require.Eventually(t, func() bool {
		for _, player := range game.players {
			if player == nil {
				continue
			}
			if player.handParticipation == nil {
				return false
			}
		}
		return true
	}, 1*time.Second, 10*time.Millisecond, "All players should have hand participation initialized")

	// In heads-up pre-flop, the small blind (dealer) acts first
	// Wait for the current player's isTurn to be set by their state machine
	var snap GameStateSnapshot
	var currentPlayerIndex int
	var currentPlayer PlayerSnapshot
	var otherPlayerIndex int
	var otherPlayer PlayerSnapshot

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
		return currentPlayer.IsTurn
	}, 1*time.Second, 10*time.Millisecond, "Current player should have isTurn=true")

	// CRITICAL: Only the current player should have isTurn = true
	assert.True(t, currentPlayer.IsTurn, "Current player should have isTurn=true")
	assert.False(t, otherPlayer.IsTurn, "Other player should have isTurn=false")

	// Current player calls
	err = game.HandlePlayerCall(currentPlayer.ID)
	require.NoError(t, err)

	// Wait for turn to switch - the other player should get isTurn=true
	require.Eventually(t, func() bool {
		snap = game.GetStateSnapshot()
		newCurrentPlayerIndex := game.GetCurrentPlayer()
		if newCurrentPlayerIndex != otherPlayerIndex {
			return false
		}
		p2After := snap.Players[otherPlayerIndex]
		return p2After.IsTurn
	}, 1*time.Second, 10*time.Millisecond, "Other player should get isTurn=true after call")

	// After the call, verify isTurn flags switched
	snap = game.GetStateSnapshot()
	p1After := snap.Players[currentPlayerIndex]
	p2After := snap.Players[otherPlayerIndex]

	// CRITICAL: The player who just acted should NO LONGER have the turn
	assert.False(t, p1After.IsTurn, "Player who just called should have isTurn=false")
	// The other player should now have the turn
	assert.True(t, p2After.IsTurn, "Other player should now have isTurn=true")

	// Verify currentPlayer index advanced
	newCurrentPlayerIndex := game.GetCurrentPlayer()
	assert.Equal(t, otherPlayerIndex, newCurrentPlayerIndex, "Current player index should have advanced")

	t.Logf("Turn management test passed: isTurn flags properly toggled")
}

// TestIsTurnFlagOnFold verifies that isTurn is cleared when a player folds
func TestIsTurnFlagOnFold(t *testing.T) {
	cfg := GameConfig{
		NumPlayers:       3,
		StartingChips:    1000,
		SmallBlind:       5,
		BigBlind:         10,
		Seed:             42,
		Log:              createTestLogger(),
		AutoAdvanceDelay: 1 * time.Second,
	}

	game, err := NewGame(cfg)
	require.NoError(t, err)

	users := []*User{
		NewUser("player1", nil, nil),
		NewUser("player2", nil, nil),
		NewUser("player3", nil, nil),
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
	var currentPlayer PlayerSnapshot

	// First, wait until the live current player object reports IsTurn=true.
	require.Eventually(t, func() bool {
		cp := game.GetCurrentPlayerObject()
		if cp == nil {
			return false
		}
		return cp.IsTurn()
	}, 2*time.Second, 10*time.Millisecond, "Current player should be initialized with isTurn=true")

	// Snapshot after turn is known to be active.
	snap = game.GetStateSnapshot()
	currentPlayerIndex = game.GetCurrentPlayer()
	require.GreaterOrEqual(t, currentPlayerIndex, 0)
	require.Less(t, currentPlayerIndex, len(snap.Players))
	currentPlayer = snap.Players[currentPlayerIndex]

	// Verify only current player has isTurn
	for i, p := range snap.Players {
		if i == currentPlayerIndex {
			assert.True(t, p.IsTurn, "Current player should have isTurn=true")
		} else {
			assert.False(t, p.IsTurn, "Non-current player %d should have isTurn=false", i)
		}
	}

	// Current player folds
	err = game.HandlePlayerFold(currentPlayer.ID)
	require.NoError(t, err)

	// Wait for the new current player to get isTurn
	var newCurrentPlayerIndex int
	var newCurrentPlayer PlayerSnapshot

	require.Eventually(t, func() bool {
		// Check live current player object first.
		cp := game.GetCurrentPlayerObject()
		if cp == nil {
			return false
		}
		if cp.ID() == currentPlayer.ID {
			return false // should have advanced
		}
		if !cp.IsTurn() {
			return false
		}

		// Mirror into snapshot for assertions below.
		snap = game.GetStateSnapshot()
		newCurrentPlayerIndex = game.GetCurrentPlayer()
		if newCurrentPlayerIndex < 0 || newCurrentPlayerIndex >= len(snap.Players) {
			return false
		}
		newCurrentPlayer = snap.Players[newCurrentPlayerIndex]
		return newCurrentPlayer.IsTurn
	}, 1*time.Second, 10*time.Millisecond, "New current player should have isTurn=true after fold")

	// Verify new current player has isTurn
	assert.True(t, newCurrentPlayer.IsTurn, "New current player should have isTurn=true")

	// Verify the folded player no longer has isTurn
	// Note: We need to find the player by the original index
	foldedPlayer := snap.Players[currentPlayerIndex]
	assert.False(t, foldedPlayer.IsTurn, "Folded player should have isTurn=false")

	// The critical test: isTurn flag should be false. Player state machine transitions
	// are tested elsewhere - we're focused on turn management here.

	// Verify all others don't have isTurn
	for i, p := range snap.Players {
		if i != newCurrentPlayerIndex {
			assert.False(t, p.IsTurn, "Non-current player %d should have isTurn=false", i)
		}
	}

	t.Logf("Fold turn management test passed")
}

// TestIsTurnFlagOnCheck verifies that isTurn is managed correctly on check
func TestIsTurnFlagOnCheck(t *testing.T) {
	cfg := GameConfig{
		NumPlayers:       2,
		StartingChips:    1000,
		SmallBlind:       5,
		BigBlind:         10,
		Seed:             42,
		Log:              createTestLogger(),
		AutoAdvanceDelay: 1 * time.Second,
	}

	game, err := NewGame(cfg)
	require.NoError(t, err)

	users := []*User{
		NewUser("player1", nil, nil),
		NewUser("player2", nil, nil),
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
		return snap.Players[sbIndex].IsTurn
	}, 1*time.Second, 10*time.Millisecond, "SB should have isTurn initially")

	// SB calls to match BB
	err = game.HandlePlayerCall(snap.Players[sbIndex].ID)
	require.NoError(t, err)

	// Wait for BB to get the turn
	require.Eventually(t, func() bool {
		snap = game.GetStateSnapshot()
		return snap.Players[bbIndex].IsTurn
	}, 1*time.Second, 10*time.Millisecond, "BB should have turn after SB calls")

	// Now BB should have the turn
	snap = game.GetStateSnapshot()
	assert.True(t, snap.Players[bbIndex].IsTurn, "BB should have turn after SB calls")
	assert.False(t, snap.Players[sbIndex].IsTurn, "SB should not have turn after calling")

	// BB checks (they can check since they matched the bet)
	err = game.HandlePlayerCheck(snap.Players[bbIndex].ID)
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
		return snap.Players[currentPlayerIndex].IsTurn
	}, 1*time.Second, 10*time.Millisecond, "Current player should have isTurn after check")

	// After both players act, should advance to FLOP
	// The turn should advance and isTurn should be managed properly
	snap = game.GetStateSnapshot()

	// Verify we advanced beyond PRE_FLOP
	phase := game.GetPhase()
	t.Logf("Phase after both players acted: %v", phase)

	// Verify only the current player has isTurn
	currentPlayerIndex := game.GetCurrentPlayer()
	t.Logf("Current player index: %d", currentPlayerIndex)
	for i, p := range snap.Players {
		t.Logf("Player %d (%s): isTurn=%v, state=%s", i, p.ID, p.IsTurn, p.StateString)
	}

	assert.True(t, snap.Players[currentPlayerIndex].IsTurn, "Current player should have isTurn")
	for i, p := range snap.Players {
		if i != currentPlayerIndex {
			assert.False(t, p.IsTurn, "Non-current player should not have isTurn")
		}
	}

	t.Logf("Check turn management test passed")
}

// Verify that GetStateSnapshot copies blind flags for players.
func TestGameStateSnapshotCopiesBlindFlags(t *testing.T) {
	cfg := GameConfig{
		NumPlayers:       2,
		StartingChips:    100,
		SmallBlind:       10,
		BigBlind:         20,
		Seed:             1,
		Log:              createTestLogger(),
		AutoAdvanceDelay: 1 * time.Second,
	}

	g, err := NewGame(cfg)
	if err != nil {
		t.Fatalf("NewGame error: %v", err)
	}

	users := []*User{
		NewUser("p1", nil, nil),
		NewUser("p2", nil, nil),
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

	if !snap.Players[0].IsDealer || !snap.Players[0].IsSmallBlind || snap.Players[0].IsBigBlind {
		t.Fatalf("player 0 flags not copied: dealer=%v sb=%v bb=%v",
			snap.Players[0].IsDealer, snap.Players[0].IsSmallBlind, snap.Players[0].IsBigBlind)
	}
	if snap.Players[1].IsDealer || snap.Players[1].IsSmallBlind || !snap.Players[1].IsBigBlind {
		t.Fatalf("player 1 flags not copied: dealer=%v sb=%v bb=%v",
			snap.Players[1].IsDealer, snap.Players[1].IsSmallBlind, snap.Players[1].IsBigBlind)
	}
}

// TestShowdownBugReproduction reproduces the exact bug scenario from the logs
// where players lose chips but no one wins the pot during showdown
func TestShowdownBugReproduction(t *testing.T) {
	// This test focuses on the core issue: pot distribution bug
	// We'll create a simple scenario where we manually set up the pot
	// and verify that the bug is fixed
	// Create a game with the exact scenario from the logs
	config := GameConfig{
		NumPlayers:       2,
		StartingChips:    1000, // Will be overridden per player
		SmallBlind:       10,
		BigBlind:         20,
		AutoStartDelay:   0, // Start immediately, no delay
		Log:              createTestLogger(),
		AutoAdvanceDelay: 1 * time.Second,
	}

	// Create game first
	game, err := NewGame(config)
	require.NoError(t, err)

	// Create users with explicit seat assignments for heads-up
	// In HU: dealer/SB = seat 0, BB = seat 1
	users := []*User{
		NewUser("player1", nil, nil), // Player 1 = dealer/SB
		NewUser("player2", nil, nil), // Player 2 = BB
	}
	game.SetPlayers(users)

	// Start the game FSM
	go game.Start(context.Background())

	// Set custom balances BEFORE starting the hand (after FSM is started but before evStartHand)
	game.mu.Lock()
	game.players[0].mu.Lock()
	game.players[0].balance = 1060
	game.players[0].startingBalance = 1060
	game.players[0].mu.Unlock()
	game.players[1].mu.Lock()
	game.players[1].balance = 940
	game.players[1].startingBalance = 940
	game.players[1].mu.Unlock()
	game.mu.Unlock()

	// Start the hand - the FSM will handle creating currentHand and dealing cards
	// Flow: evStartHand → statePreDeal (creates Hand, posts blinds) → stateDeal (deals cards) → stateBlinds → statePreFlop
	game.sm.Send(evStartHand{})

	// Wait for PRE_FLOP phase and hand participation to be ready
	require.Eventually(t, func() bool {
		return game.GetPhase() == pokerrpc.GamePhase_PRE_FLOP
	}, 2*time.Second, 10*time.Millisecond, "Game should reach PRE_FLOP")

	// Wait for hand participation FSMs to be initialized
	require.Eventually(t, func() bool {
		return game.players[0].handParticipation != nil && game.players[1].handParticipation != nil
	}, 1*time.Second, 10*time.Millisecond, "Hand participation FSMs should be initialized")

	// Verify initial balances
	player1Balance := game.players[0].Balance()
	player2Balance := game.players[1].Balance()
	t.Logf("Initial balances - Player 1: %d, Player 2: %d", player1Balance, player2Balance)

	// Debug: Check if hand participation FSMs are initialized
	t.Logf("Player 1 handParticipation FSM: %v", game.players[0].handParticipation != nil)
	t.Logf("Player 2 handParticipation FSM: %v", game.players[1].handParticipation != nil)

	// After blinds are posted: Player 1 (SB) posted 10, Player 2 (BB) posted 20
	require.Equal(t, int64(1050), player1Balance, "Player 1 balance after posting small blind should be 1050")
	require.Equal(t, int64(920), player2Balance, "Player 2 balance after posting big blind should be 920")

	// Simulate the exact betting sequence for heads-up:
	// 1. Player 1 (SB/dealer) calls to 20 (big blind)
	// 2. Player 2 (BB) raises to 100
	// 3. Player 1 (SB) calls to 100

	// Player 1 (SB) calls to 20 (big blind) - first action in HU
	currentPlayer := game.GetCurrentPlayerObject()
	require.NotNil(t, currentPlayer, "Current player should not be nil")
	require.Equal(t, "player1", currentPlayer.ID(), "Player 1 should be first to act")
	err = game.HandlePlayerCall("player1")
	require.NoError(t, err, "Player 1 call should succeed")

	// Player 2 (BB) raises to 100
	currentPlayer = game.GetCurrentPlayerObject()
	require.NotNil(t, currentPlayer, "Current player should not be nil")
	require.Equal(t, "player2", currentPlayer.ID(), "Player 2 should be next to act")
	err = game.HandlePlayerBet("player2", 100)
	require.NoError(t, err, "Player 2 bet should succeed")

	// Player 1 (SB) calls to 100
	currentPlayer = game.GetCurrentPlayerObject()
	require.NotNil(t, currentPlayer, "Current player should not be nil")
	require.Equal(t, "player1", currentPlayer.ID(), "Player 1 should be next to act")
	err = game.HandlePlayerCall("player1")
	require.NoError(t, err, "Player 1 call should succeed")

	// Check pot amount - should be 200
	// In HU 10/20: preflop blinds = 30; BB "bet to 100" adds 80; SB calls to 100 adds 80 (after first call to 20 added 10)
	// Totals: SB −90 (10 call to 20 + 80 call to 100), BB −80 (raise to 100) ⇒ pot 200
	totalPot := game.GetPot()
	t.Logf("Total pot after betting: %d", totalPot)
	require.Equal(t, int64(200), totalPot, "Expected pot to be 200")

	// Check player balances after betting
	player1BalanceAfterBetting := game.players[0].Balance()
	player2BalanceAfterBetting := game.players[1].Balance()
	t.Logf("Balances after betting - Player 1: %d, Player 2: %d", player1BalanceAfterBetting, player2BalanceAfterBetting)

	// Expected balances based on HU betting sequence:
	// Player 1 (SB): 1050 - 10 (call to 20) - 80 (call to 100) = 960
	// Player 2 (BB): 920 - 80 (raise to 100) = 840
	expectedPlayer1Balance := int64(1050 - 10 - 80) // 90 chips total
	expectedPlayer2Balance := int64(920 - 80)       // 80 chips total

	require.Equal(t, expectedPlayer1Balance, player1BalanceAfterBetting, "Player 1 balance after betting")
	require.Equal(t, expectedPlayer2Balance, player2BalanceAfterBetting, "Player 2 balance after betting")

	// Now simulate checking through all streets (FLOP, TURN, RIVER)
	// This should trigger the bug where currentBets gets cleared

	// Wait for FLOP phase - this should happen automatically after the last call
	require.Eventually(t, func() bool {
		return game.GetPhase() == pokerrpc.GamePhase_FLOP
	}, 2*time.Second, 10*time.Millisecond, "Game should reach FLOP")

	// Player 2 checks
	currentPlayer = game.GetCurrentPlayerObject()
	require.NotNil(t, currentPlayer, "Current player should not be nil")
	require.Equal(t, "player2", currentPlayer.ID(), "Player 2 should be first to act on flop")
	err = game.HandlePlayerCheck("player2")
	require.NoError(t, err, "Player 2 check should succeed")

	// Player 1 checks
	currentPlayer = game.GetCurrentPlayerObject()
	require.NotNil(t, currentPlayer, "Current player should not be nil")
	require.Equal(t, "player1", currentPlayer.ID(), "Player 1 should be next to act on flop")
	err = game.HandlePlayerCheck("player1")
	require.NoError(t, err, "Player 1 check should succeed")

	// Wait for TURN phase
	require.Eventually(t, func() bool {
		return game.GetPhase() == pokerrpc.GamePhase_TURN
	}, 2*time.Second, 10*time.Millisecond, "Game should reach TURN")

	// Player 2 checks
	currentPlayer = game.GetCurrentPlayerObject()
	require.NotNil(t, currentPlayer, "Current player should not be nil")
	require.Equal(t, "player2", currentPlayer.ID(), "Player 2 should be first to act on turn")
	err = game.HandlePlayerCheck("player2")
	require.NoError(t, err, "Player 2 check should succeed")

	// Player 1 checks
	currentPlayer = game.GetCurrentPlayerObject()
	require.NotNil(t, currentPlayer, "Current player should not be nil")
	require.Equal(t, "player1", currentPlayer.ID(), "Player 1 should be next to act on turn")
	err = game.HandlePlayerCheck("player1")
	require.NoError(t, err, "Player 1 check should succeed")

	// Wait for RIVER phase
	require.Eventually(t, func() bool {
		return game.GetPhase() == pokerrpc.GamePhase_RIVER
	}, 2*time.Second, 10*time.Millisecond, "Game should reach RIVER")

	// Player 2 checks
	currentPlayer = game.GetCurrentPlayerObject()
	require.NotNil(t, currentPlayer, "Current player should not be nil")
	require.Equal(t, "player2", currentPlayer.ID(), "Player 2 should be first to act on river")
	err = game.HandlePlayerCheck("player2")
	require.NoError(t, err, "Player 2 check should succeed")

	// Player 1 checks - this should trigger showdown
	currentPlayer = game.GetCurrentPlayerObject()
	require.NotNil(t, currentPlayer, "Current player should not be nil")
	require.Equal(t, "player1", currentPlayer.ID(), "Player 1 should be next to act on river")
	err = game.HandlePlayerCheck("player1")
	require.NoError(t, err, "Player 1 check should succeed")

	// Wait for showdown to be triggered by FSM
	require.Eventually(t, func() bool {
		return game.GetPhase() == pokerrpc.GamePhase_SHOWDOWN
	}, 2*time.Second, 10*time.Millisecond, "Game should reach SHOWDOWN")

	// Check pot amount before showdown - should still be 200
	potBeforeShowdown := game.GetPot()
	t.Logf("Pot before showdown: %d", potBeforeShowdown)
	// Note: Pot might be 0 if showdown has already processed it

	// Check what phase we're in after the last check
	phase := game.GetPhase()
	t.Logf("Phase after river checks: %v", phase)

	// If we're still in RIVER phase, we may need to manually trigger showdown
	if phase == pokerrpc.GamePhase_RIVER {
		t.Logf("Still in RIVER phase, attempting to trigger showdown manually")
		// Try to trigger showdown manually
		result, err := game.HandleShowdown()
		if err != nil {
			t.Logf("Manual showdown failed: %v", err)
		} else {
			t.Logf("Manual showdown succeeded: %+v", result)
		}
	} else {
		t.Logf("Phase is %v, showdown should have been triggered automatically", phase)
	}

	// Wait for showdown to complete
	require.Eventually(t, func() bool {
		phase := game.GetPhase()
		return phase == pokerrpc.GamePhase_SHOWDOWN || phase == pokerrpc.GamePhase_NEW_HAND_DEALING || phase == pokerrpc.GamePhase_WAITING
	}, 3*time.Second, 10*time.Millisecond, "Showdown should complete")

	// Check final phase
	finalPhase := game.GetPhase()
	t.Logf("Final phase: %v", finalPhase)

	// The FSM should have handled showdown automatically
	// We can verify the results by checking final balances and pot state

	// Check final balances
	player1FinalBalance := game.players[0].Balance()
	player2FinalBalance := game.players[1].Balance()
	t.Logf("Final balances - Player 1: %d, Player 2: %d", player1FinalBalance, player2FinalBalance)

	// Check final pot amount
	finalPot := game.GetPot()
	t.Logf("Final pot amount: %d", finalPot)

	// Check if there are any winners recorded
	winners := game.GetWinners()
	t.Logf("Winners: %v", winners)

	// The sum of final balances should equal the sum of initial balances
	// (no chips should be lost to the void)
	totalInitialBalance := int64(1060 + 940)
	totalFinalBalance := player1FinalBalance + player2FinalBalance

	require.Equal(t, totalInitialBalance, totalFinalBalance,
		"Chip conservation violated! Initial total: %d, Final total: %d", totalInitialBalance, totalFinalBalance)

	// At least one player should have more chips than they started with
	// (someone should have won the pot)
	player1Won := player1FinalBalance > player1Balance
	player2Won := player2FinalBalance > player2Balance

	require.True(t, player1Won || player2Won,
		"Neither player won chips - this indicates the pot distribution bug!")
}

// TestShowdownTotalPotBug reproduces the bug where TotalPot in showdown result
// reflects the pre-refund amount instead of the actual pot after refunds.
// Scenario: P1 raises to 60, P2 folds, P1 should win 30 (SB 10 + BB 20)
// but TotalPot shows 80 (60 + 20) instead of 30.
func TestShowdownTotalPotBug(t *testing.T) {
	// Create a game with 2 players
	cfg := GameConfig{
		NumPlayers:       2,
		StartingChips:    1000,
		SmallBlind:       10,
		BigBlind:         20,
		Seed:             42,
		Log:              createTestLogger(),
		AutoAdvanceDelay: 1 * time.Second,
	}

	game, err := NewGame(cfg)
	require.NoError(t, err)

	// Create test users and set them in the game
	users := []*User{
		NewUser("player1", nil, nil), // SB/Dealer
		NewUser("player2", nil, nil), // BB
	}
	game.SetPlayers(users)

	// Properly simulate the scenario: P1 (SB) raises to 60, P2 (BB) folds
	// This must be in PRE_FLOP to trigger the "no-call preflop" refund rule

	// Start hand participation for both players
	err = game.players[0].StartHandParticipation()
	require.NoError(t, err)
	err = game.players[1].StartHandParticipation()
	require.NoError(t, err)

	// Set up blinds properly - this is crucial for the refund rule
	game.players[0].mu.Lock()
	game.players[0].isSmallBlind = true
	game.players[0].isDealer = true
	game.players[0].mu.Unlock()

	game.players[1].mu.Lock()
	game.players[1].isBigBlind = true
	game.players[1].mu.Unlock()

	// Set game phase to PRE_FLOP (required for the specific refund rule)
	game.mu.Lock()
	game.phase = pokerrpc.GamePhase_PRE_FLOP
	game.mu.Unlock()

	// Set up pot manager with the scenario:
	// - P1 (SB) bets 60 total (SB 10 + raise 50)
	// - P2 (BB) bets 20 total (BB)
	// - Total pot before refund: 80
	game.potManager = NewPotManager(2)
	game.potManager.addBet(0, 60, game.players) // P1 bets 60
	game.potManager.addBet(1, 20, game.players) // P2 bets 20 (BB)

	// Set player states: P1 active, P2 folded
	// P1 stays in active state (default)
	// P2 folds - use synchronous fold to ensure state is updated
	if game.players[1].handParticipation != nil {
		reply := make(chan error, 1)
		game.players[1].handParticipation.Send(evFoldReq{Reply: reply})
		err = <-reply
		require.NoError(t, err)
	}

	// Set up hands for the players (required for showdown)
	// Initialize Hand if not already present
	if game.currentHand == nil {
		playerIDs := make([]string, len(game.players))
		for i, p := range game.players {
			playerIDs[i] = p.ID()
		}
		game.currentHand = NewHand(playerIDs)
	}

	// Deal cards to player 0
	game.currentHand.DealCardToPlayer(game.players[0].ID(), Card{suit: Hearts, value: Ace})
	game.currentHand.DealCardToPlayer(game.players[0].ID(), Card{suit: Spades, value: King})

	// Deal cards to player 1
	game.currentHand.DealCardToPlayer(game.players[1].ID(), Card{suit: Clubs, value: Queen})
	game.currentHand.DealCardToPlayer(game.players[1].ID(), Card{suit: Diamonds, value: Jack})

	// Set up community cards (required for showdown)
	game.mu.Lock()
	game.communityCards = []Card{
		{suit: Hearts, value: Ten},
		{suit: Spades, value: Nine},
		{suit: Clubs, value: Eight},
		{suit: Diamonds, value: Seven},
		{suit: Hearts, value: Six},
	}
	game.mu.Unlock()

	// Verify initial pot is 80 (pre-refund)
	require.Equal(t, int64(80), game.GetPot())

	// Run showdown
	result, err := game.HandleShowdown()
	require.NoError(t, err)

	// The bug: TotalPot should be 30 (actual pot after refunds)
	// but it shows 80 (pre-refund amount)
	require.Equal(t, int64(30), result.TotalPot,
		"TotalPot should reflect actual pot after refunds, not pre-refund amount")

	// Verify actual winnings match the pot
	require.Len(t, result.Winners, 1)
	require.Equal(t, "player1", result.Winners[0])
	require.Len(t, result.WinnerInfo, 1)
	require.Equal(t, int64(30), result.WinnerInfo[0].Winnings)

	// Note: We're not checking final balances here because this is a unit test
	// that focuses on the TotalPot bug. The actual balance accounting would
	// be handled by the full game flow with proper bet/debit mechanisms.
}

// Test that when the current player's timebank expires facing a bet, they auto-fold.
func TestTimebank_AutoFold_Preflop(t *testing.T) {
	tbl := NewTable(TableConfig{
		ID:               "tbl-timebank-fold",
		Log:              createTestLogger(),
		GameLog:          createTestLogger(),
		HostID:           "host",
		BuyIn:            0,
		MinPlayers:       2,
		MaxPlayers:       2,
		SmallBlind:       5,
		BigBlind:         10,
		StartingChips:    100,
		TimeBank:         50 * time.Millisecond,
		AutoStartDelay:   10 * time.Millisecond,
		AutoAdvanceDelay: 10 * time.Millisecond,
	})
	defer tbl.Close()
	// Wire a buffered event channel to avoid global drop metrics noise.
	// We intentionally do not close this channel; tbl.Close() will stop
	// publishing and goroutines cleanly.
	evtChFold := make(chan TableEvent, 16)
	tbl.SetEventChannel(evtChFold)

	// Seat two users and ready
	user1, err := tbl.AddNewUser("p1", nil)
	require.NoError(t, err)
	user2, err := tbl.AddNewUser("p2", nil)
	require.NoError(t, err)

	// Mark players ready (SendReady now notifies table state machine)
	user1.SendReady()
	user2.SendReady()

	require.True(t, tbl.CheckAllPlayersReady())
	require.NoError(t, tbl.StartGame())

	// Wait until: PRE_FLOP and current player is facing action (need > 0)
	require.Eventually(t, func() bool {
		g := tbl.GetGame()
		if g == nil {
			return false
		}
		if g.GetPhase() != pokerrpc.GamePhase_PRE_FLOP {
			return false
		}
		cp := g.GetCurrentPlayerObject()
		if cp == nil {
			return false
		}
		cp.mu.RLock()
		need := g.GetCurrentBet() - cp.currentBet
		cpid := cp.id
		cp.mu.RUnlock()
		return need > 0 && cpid != ""
	}, 2*time.Second, 10*time.Millisecond)

	g := tbl.GetGame()
	require.NotNil(t, g)
	cur := g.GetCurrentPlayerObject()
	require.NotNil(t, cur)
	curID := cur.ID()

	// Simulate timebank expiration
	g.TriggerTimebankExpiredFor(curID)

	// Player should auto-fold
	require.Eventually(t, func() bool {
		// Find the player by id and check state
		for _, p := range g.GetPlayers() {
			if p == nil {
				continue
			}
			if p.ID() == curID {
				fmt.Println("player state:", p.GetCurrentStateString())
				return p.GetCurrentStateString() == FOLDED_STATE
			}
		}
		return false
	}, 1*time.Second, 10*time.Millisecond)
}

// Test that when the current player's timebank expires with nothing to call, they auto-check.
func TestTimebank_AutoCheck_Flop(t *testing.T) {
	tbl := NewTable(TableConfig{
		ID:               "tbl-timebank-check",
		Log:              createTestLogger(),
		GameLog:          createTestLogger(),
		HostID:           "host",
		BuyIn:            0,
		MinPlayers:       2,
		MaxPlayers:       2,
		SmallBlind:       5,
		BigBlind:         10,
		StartingChips:    100,
		TimeBank:         50 * time.Millisecond,
		AutoStartDelay:   10 * time.Millisecond,
		AutoAdvanceDelay: 10 * time.Millisecond,
	})
	defer tbl.Close()
	evtChCheck := make(chan TableEvent, 16)
	tbl.SetEventChannel(evtChCheck)

	// Seat two users and ready
	user1, err := tbl.AddNewUser("p1", nil)
	require.NoError(t, err)
	user2, err := tbl.AddNewUser("p2", nil)
	require.NoError(t, err)

	// Mark players ready (SendReady now notifies table state machine)
	user1.SendReady()
	user2.SendReady()

	require.True(t, tbl.CheckAllPlayersReady())
	require.NoError(t, tbl.StartGame())

	// Wait until game exists
	require.Eventually(t, func() bool { return tbl.GetGame() != nil }, 2*time.Second, 10*time.Millisecond)
	g := tbl.GetGame()
	require.NotNil(t, g)

	// Force move to flop and reset table current bet; start turn for post-flop actor
	g.StateFlop()
	g.mu.Lock()
	g.currentBet = 0
	g.mu.Unlock()
	g.InitializeCurrentPlayer()

	// Capture current player id
	cp := g.GetCurrentPlayerObject()
	require.NotNil(t, cp)
	curID := cp.ID()

	// Sanity: need <= 0 to favor auto-check path
	cp.mu.RLock()
	need := g.GetCurrentBet() - cp.currentBet
	cp.mu.RUnlock()
	require.LessOrEqual(t, need, int64(0))

	// Trigger timebank expiration and expect a check (not folded) and turn advancement
	g.TriggerTimebankExpiredFor(curID)

	require.Eventually(t, func() bool {
		// Current player should have advanced to someone else
		next := g.GetCurrentPlayerObject()
		return next != nil && next.ID() != curID
	}, 1*time.Second, 10*time.Millisecond)

	// Ensure the player did not fold
	var state string
	for _, p := range g.GetPlayers() {
		if p == nil {
			continue
		}
		if p.ID() == curID {
			state = p.GetCurrentStateString()
			break
		}
	}
	require.Equal(t, IN_GAME_STATE, state)
}

// TestShowdownWithFoldedPlayer verifies that showdown works correctly when
// a folded player is present (folded players have nil handValue and should be skipped).
// Scenario: 3 players, one folds, two go all-in, showdown should complete successfully.
// This test verifies the fix for the bug where folded players with nil handValue
// caused showdown to fail.
func TestShowdownWithFoldedPlayer(t *testing.T) {
	cfg := GameConfig{
		NumPlayers:       3,
		StartingChips:    1000,
		SmallBlind:       10,
		BigBlind:         20,
		Seed:             42,
		Log:              createTestLogger(),
		AutoAdvanceDelay: 1 * time.Second,
	}

	game, err := NewGame(cfg)
	require.NoError(t, err)

	// Create 3 players
	users := []*User{
		NewUser("player1", nil, nil),
		NewUser("player2", nil, nil),
		NewUser("player3", nil, nil),
	}
	game.SetPlayers(users)

	// Initialize Hand for this test
	game.currentHand = NewHand([]string{"player1", "player2", "player3"})

	// Deal hole cards to all players
	for i := 0; i < 3; i++ {
		for j := 0; j < 2; j++ {
			card, ok := game.deck.Draw()
			require.True(t, ok, "should be able to draw card")
			game.currentHand.DealCardToPlayer(game.players[i].ID(), card)
		}
	}

	// Set up community cards (flop, turn, river)
	game.communityCards = []Card{
		{suit: Spades, value: Five},
		{suit: Diamonds, value: King},
		{suit: Hearts, value: Four},
		{suit: Hearts, value: Five},
		{suit: Clubs, value: Ten},
	}

	// Set up pot with bets from all 3 players
	game.potManager = NewPotManager(3)
	game.potManager.addBet(0, 20, game.players)   // Player 1 (folded) bet 20
	game.potManager.addBet(1, 1000, game.players) // Player 2 (all-in) bet 1000
	game.potManager.addBet(2, 1000, game.players) // Player 3 (all-in) bet 1000

	// Set up player states:
	// Player 1: FOLDED (has nil handValue - this is the bug)
	// Player 2: ALL_IN
	// Player 3: ALL_IN

	// Initialize hand participation for all players
	require.NoError(t, game.players[0].StartHandParticipation())
	require.NoError(t, game.players[1].StartHandParticipation())
	require.NoError(t, game.players[2].StartHandParticipation())

	// Mark player 1 as folded
	game.players[0].mu.Lock()
	game.players[0].hasFolded = true
	game.players[0].balance = 980 // Started with 1000, bet 20
	game.players[0].currentBet = 20
	game.players[0].mu.Unlock()
	// Send fold event to transition to folded state
	game.players[0].handParticipation.Send(evFoldReq{Reply: make(chan error, 1)})

	// Mark players 2 and 3 as all-in
	game.players[1].mu.Lock()
	game.players[1].balance = 0
	game.players[1].currentBet = 1000
	game.players[1].isAllIn = true // Set flag
	game.players[1].mu.Unlock()

	game.players[2].mu.Lock()
	game.players[2].balance = 0
	game.players[2].currentBet = 1000
	game.players[2].isAllIn = true // Set flag
	game.players[2].mu.Unlock()

	// Send events to trigger state machine to detect all-in condition
	game.players[1].handParticipation.Send(evStartTurn{})
	game.players[2].handParticipation.Send(evStartTurn{})

	// Wait for state transitions
	require.Eventually(t, func() bool {
		return game.players[0].GetCurrentStateString() == FOLDED_STATE
	}, 200*time.Millisecond, 10*time.Millisecond, "Player 1 should be folded")

	require.Eventually(t, func() bool {
		return game.players[1].GetCurrentStateString() == ALL_IN_STATE
	}, 200*time.Millisecond, 10*time.Millisecond, "Player 2 should be all-in")

	require.Eventually(t, func() bool {
		return game.players[2].GetCurrentStateString() == ALL_IN_STATE
	}, 200*time.Millisecond, 10*time.Millisecond, "Player 3 should be all-in")

	// Set phase to RIVER (ready for showdown)
	game.mu.Lock()
	game.phase = pokerrpc.GamePhase_RIVER

	// Call showdown - should succeed even with folded player (nil handValue)
	// The fix should skip folded players when checking handValue
	res, err := game.handleShowdown()
	game.mu.Unlock()

	// Showdown should succeed - folded players are now skipped
	require.NoError(t, err, "Showdown should succeed with folded player present")
	require.NotNil(t, res, "Showdown should return a result")

	// Verify that we have winners (the two all-in players)
	require.Greater(t, len(res.Winners), 0, "Should have at least one winner")
	t.Logf("✓ Showdown completed successfully with %d winners (folded player correctly skipped)", len(res.Winners))
}

// TestAutoStartAfterElimination verifies that auto-start works after a player is eliminated.
// Scenario: 3 players start, one eliminated (0 chips), auto-start should work with 2 players remaining.
// The table is created with minPlayers=2 so it can continue with 2 active players.
func TestAutoStartAfterElimination(t *testing.T) {
	// Create a table with minPlayers=2
	tbl := newTestTable(t, 2, 6, 10, 20, 1000)
	tbl.config.AutoStartDelay = 50 * time.Millisecond

	// Add 3 players
	user1, err := tbl.AddNewUser("player1", nil)
	require.NoError(t, err)
	user2, err := tbl.AddNewUser("player2", nil)
	require.NoError(t, err)
	user3, err := tbl.AddNewUser("player3", nil)
	require.NoError(t, err)

	// Mark all players ready
	user1.SendReady()
	user2.SendReady()
	user3.SendReady()

	// Start the game
	err = tbl.StartGame()
	require.NoError(t, err, "Should be able to start game with 3 players")

	// Wait for game to be active
	require.Eventually(t, func() bool {
		return tbl.GetTableStateString() == "GAME_ACTIVE"
	}, 2*time.Second, 10*time.Millisecond, "Game should be active")

	// Get the game
	game := tbl.GetGame()
	require.NotNil(t, game, "Game should exist")

	// Wait for PRE_FLOP phase
	require.Eventually(t, func() bool {
		return game.GetPhase() == pokerrpc.GamePhase_PRE_FLOP
	}, 2*time.Second, 10*time.Millisecond, "Game should reach PRE_FLOP")

	// Simulate a hand where player3 loses all chips and is eliminated
	// We'll manually set up the game state after a showdown where player3 is eliminated
	game.mu.Lock()

	// Find player3 and set balance to 0 (eliminated after losing)
	var player3Idx int = -1
	for i, p := range game.players {
		if p != nil && p.ID() == "player3" {
			player3Idx = i
			break
		}
	}
	require.GreaterOrEqual(t, player3Idx, 0, "Should find player3")

	// Set player3 balance to 0 (eliminated)
	game.players[player3Idx].mu.Lock()
	game.players[player3Idx].balance = 0
	game.players[player3Idx].mu.Unlock()

	// Verify: 2 players have chips > 0, 1 player has 0 chips
	readyCount := 0
	for _, p := range game.players {
		if p != nil && p.Balance() > 0 {
			readyCount++
		}
	}
	game.mu.Unlock()

	require.Equal(t, 2, readyCount, "Should have 2 players with chips remaining")
	require.Equal(t, 2, tbl.config.MinPlayers, "Table should have minPlayers=2")

	// Manually trigger auto-start check (simulating what happens after showdown)
	// With minPlayers=2 and 2 active players, auto-start should succeed
	err = tbl.handleAutoStart()
	require.NoError(t, err, "Auto-start should succeed with 2 players when minPlayers=2")
	t.Log("✓ Auto-start succeeded with 2 players")
}
