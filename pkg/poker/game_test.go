package poker

import (
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/decred/slog"
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
		NumPlayers:    2,
		StartingChips: 1000, // Set to 1000 to match the expected balance
		Seed:          42,   // Use a fixed seed for deterministic testing
		Log:           createTestLogger(),
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
		NewUser("player1", "Player 1", 1000, 0),
		NewUser("player2", "Player 2", 1000, 1),
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
		if player.GetCurrentStateString() == "FOLDED" {
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
		NumPlayers:    1,
		StartingChips: 100,
		Log:           createTestLogger(),
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

func TestDealCards(t *testing.T) {
	cfg := GameConfig{
		NumPlayers:    2,
		StartingChips: 100,
		Seed:          42,
		Log:           createTestLogger(),
	}

	game, err := NewGame(cfg)
	require.NoError(t, err)

	// Create test users and set them in the game
	users := []*User{
		NewUser("player1", "Player 1", 100, 0),
		NewUser("player2", "Player 2", 100, 1),
	}
	game.SetPlayers(users)

	// Deal cards manually for testing (since DealCards was removed)
	for _, player := range game.players {
		for i := 0; i < 2; i++ {
			card, ok := game.deck.Draw()
			if !ok {
				t.Fatalf("Failed to draw card from deck")
			}
			player.hand = append(player.hand, card)
		}
	}

	// Check each player has 2 cards
	for i, player := range game.players {
		if len(player.hand) != 2 {
			t.Errorf("Player %d: Expected 2 cards, got %d", i, len(player.hand))
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
		NumPlayers: 2,
		Seed:       42,
		Log:        createTestLogger(),
	}

	game, err := NewGame(cfg)
	require.NoError(t, err)

	// Create test users and set them in the game
	users := []*User{
		NewUser("player1", "Player 1", 100, 0),
		NewUser("player2", "Player 2", 100, 1),
	}
	game.SetPlayers(users)

	// Deal cards manually for testing (since DealCards was removed)
	for _, player := range game.players {
		for i := 0; i < 2; i++ {
			card, ok := game.deck.Draw()
			if !ok {
				t.Fatalf("Failed to draw card from deck")
			}
			player.hand = append(player.hand, card)
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
		NumPlayers: 2,
		Seed:       42,
		Log:        createTestLogger(),
	}

	game, err := NewGame(cfg)
	require.NoError(t, err)

	// Create test users and set them in the game
	users := []*User{
		NewUser("player1", "Player 1", 0, 0), // Start with 0 balance for clean test
		NewUser("player2", "Player 2", 0, 1),
	}
	game.SetPlayers(users)

	// Set up player hands manually
	player1 := game.players[0]
	player2 := game.players[1]

	// Player 1 has a pair of Aces
	player1.hand = []Card{
		{suit: Hearts, value: Ace},
		{suit: Spades, value: Ace},
	}

	// Player 2 has King-Queen
	player2.hand = []Card{
		{suit: Hearts, value: King},
		{suit: Spades, value: Queen},
	}

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
	if player1.balance != 100 {
		t.Errorf("Expected player 1 to win with pot of 100, got %d", player1.balance)
	}

	// Player 2 should not win anything
	if player2.balance != 0 {
		t.Errorf("Expected player 2 to not win anything, got %d", player2.balance)
	}

	// Check hand descriptions
	if !strings.Contains(player1.handDescription, "Pair") {
		t.Errorf("Expected pair description, got %s", player1.handDescription)
	}

	if !strings.Contains(player2.handDescription, "High Card") {
		t.Errorf("Expected high card description, got %s", player2.handDescription)
	}
}

func TestTieBreakerShowdown(t *testing.T) {
	// Create a game with 3 players
	cfg := GameConfig{
		NumPlayers: 3,
		Seed:       42,
		Log:        createTestLogger(),
	}

	game, err := NewGame(cfg)
	require.NoError(t, err)

	// Create test users and set them in the game
	users := []*User{
		NewUser("player1", "Player 1", 0, 0), // Start with 0 balance for clean test
		NewUser("player2", "Player 2", 0, 1),
		NewUser("player3", "Player 3", 0, 2),
	}
	game.SetPlayers(users)

	// Set up player hands manually
	player1 := game.players[0]
	player2 := game.players[1]
	player3 := game.players[2]

	// All players have a pair of Aces but with different kickers
	player1.hand = []Card{
		{suit: Hearts, value: Ace},
		{suit: Spades, value: Ace},
	}

	player2.hand = []Card{
		{suit: Clubs, value: Ace},
		{suit: Diamonds, value: Ace},
	}

	player3.hand = []Card{
		{suit: Hearts, value: King},
		{suit: Spades, value: King}, // Lower pair
	}

	// Set community cards: 2-5-7-9-Jack
	game.communityCards = []Card{
		{suit: Clubs, value: Two},
		{suit: Diamonds, value: Five},
		{suit: Hearts, value: Seven},
		{suit: Spades, value: Nine},
		{suit: Clubs, value: Jack},
	}

	// Mark player 3 as folded
	player3.sm.Send(evFold{})

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
	if player1.balance != 50 {
		t.Errorf("Expected player 1 to win 50 (half pot), got %d", player1.balance)
	}

	if player2.balance != 50 {
		t.Errorf("Expected player 2 to win 50 (half pot), got %d", player2.balance)
	}

	// Player 3 should not win anything (folded)
	if player3.balance != 0 {
		t.Errorf("Expected player 3 to not win anything (folded), got %d", player3.balance)
	}
}

// Split pot: Board makes the best five-card hand for both players.
func TestSplitPotShowdown(t *testing.T) {
	cfg := GameConfig{NumPlayers: 2, Seed: 1, Log: createTestLogger()}
	game, err := NewGame(cfg)
	require.NoError(t, err)

	users := []*User{
		NewUser("p1", "p1", 0, 0),
		NewUser("p2", "p2", 0, 1),
	}
	game.SetPlayers(users)

	// Force hands that don't improve beyond board
	game.players[0].hand = []Card{{suit: Hearts, value: Two}, {suit: Clubs, value: Three}}
	game.players[1].hand = []Card{{suit: Diamonds, value: Four}, {suit: Spades, value: Five}}

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
	res, err := game.handleShowdown()
	require.NoError(t, err)
	require.NotNil(t, res)

	// Both players should split 100 → 50 each
	if game.players[0].balance != 50 {
		t.Fatalf("p1 expected 50, got %d", game.players[0].balance)
	}
	if game.players[1].balance != 50 {
		t.Fatalf("p2 expected 50, got %d", game.players[1].balance)
	}
}

// Side pot: p3 all-in short, p1/p2 create side pot; winners differ per pot.
func TestSidePotShowdown(t *testing.T) {
	cfg := GameConfig{NumPlayers: 3, Seed: 1, Log: createTestLogger()}
	game, err := NewGame(cfg)
	require.NoError(t, err)

	users := []*User{
		NewUser("p1", "p1", 0, 0),
		NewUser("p2", "p2", 0, 1),
		NewUser("p3", "p3", 0, 2),
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
	game.players[0].sm.Send(evStartHand{})
	game.players[1].sm.Send(evStartHand{})
	game.players[2].sm.Send(evStartHand{})

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

	game.players[0].handValue = &hv1
	game.players[1].handValue = &hv2
	game.players[2].handValue = &hv3

	// Pots are automatically built on each bet, no need to call BuildPotsFromTotals

	// Distribute pots
	game.potManager.distributePots(game.players)

	// Expected: p3 gets 90 (main), p1 gets 40 (side)
	if game.players[2].balance != 90 {
		t.Fatalf("p3 expected 90 from main pot, got %d", game.players[2].balance)
	}
	if game.players[0].balance != 40 {
		t.Fatalf("p1 expected 40 from side pot, got %d", game.players[0].balance)
	}
	if game.players[1].balance != 0 {
		t.Fatalf("p2 expected 0, got %d", game.players[1].balance)
	}
}

func TestAutoStartOnNewHandStarted(t *testing.T) {
	cfg := GameConfig{
		NumPlayers:     2,
		StartingChips:  1000,
		SmallBlind:     10,
		BigBlind:       20,
		AutoStartDelay: 10 * time.Millisecond,
		Log:            createTestLogger(),
	}
	game, err := NewGame(cfg)
	require.NoError(t, err)

	// Set players so readyCount >= MinPlayers in timer callback
	users := []*User{
		NewUser("p1", "p1", 0, 0),
		NewUser("p2", "p2", 0, 1),
	}
	game.SetPlayers(users)

	var mu sync.Mutex
	started := false
	callbackCalled := false

	wg := sync.WaitGroup{}
	wg.Add(1)

	// Provide auto-start callbacks _without_ the OnNewHandStarted field.
	game.SetAutoStartCallbacks(&AutoStartCallbacks{
		MinPlayers: func() int { return 2 },
		StartNewHand: func() error {
			mu.Lock()
			started = true
			mu.Unlock()
			return nil
		},
	})

	// Attach the callback via the helper being tested.
	game.SetOnNewHandStartedCallback(func() {
		mu.Lock()
		callbackCalled = true
		mu.Unlock()
		wg.Done()
	})

	// Trigger the timer
	game.ScheduleAutoStart()

	// Wait with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// ok
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for OnNewHandStarted callback")
	}

	mu.Lock()
	if !started {
		t.Fatal("expected StartNewHand to be called")
	}
	if !callbackCalled {
		t.Fatal("expected OnNewHandStarted to be called")
	}
	mu.Unlock()
}

// Ensure that when multiple players are all-in pre-flop, the game
// automatically deals remaining community cards and performs showdown
// without panicking.
func TestPreFlopAllInAutoDealShowdown(t *testing.T) {
	cfg := GameConfig{
		NumPlayers:     2,
		StartingChips:  100,
		SmallBlind:     10,
		BigBlind:       20,
		Seed:           1,
		AutoStartDelay: 0,
		TimeBank:       0,
		Log:            createTestLogger(),
	}
	game, err := NewGame(cfg)
	require.NoError(t, err)

	users := []*User{
		NewUser("p1", "p1", 0, 0),
		NewUser("p2", "p2", 0, 1),
	}
	game.SetPlayers(users)

	// Simulate pre-flop all-in by both players with some bets recorded
	game.phase = pokerrpc.GamePhase_PRE_FLOP
	game.communityCards = nil
	game.potManager = NewPotManager(2)

	// Put some chips in to form a pot
	game.potManager.addBet(0, 50, game.players)
	game.potManager.addBet(1, 50, game.players)

	// Mark both players as all-in and not folded
	game.players[0].sm.Send(evAllIn{})
	game.players[1].sm.Send(evAllIn{})
	game.players[0].lastAction = time.Now()
	game.players[1].lastAction = time.Now()

	// Call showdown; should auto-deal to 5 community cards and not error
	res, err := game.handleShowdown()
	require.NoError(t, err)
	require.NotNil(t, res)

	if got := len(game.communityCards); got != 5 {
		t.Fatalf("expected 5 community cards to be dealt, got %d", got)
	}

	// Total pot equals sum of bets (100)
	require.EqualValues(t, int64(100), res.TotalPot)
}

// Ensure auto-start counts short-stacked players (>0 chips) as eligible and starts a new hand.
func TestAutoStartAllowsShortStackAllIn(t *testing.T) {
	cfg := GameConfig{
		NumPlayers:     2,
		StartingChips:  0,
		SmallBlind:     10,
		BigBlind:       20,
		AutoStartDelay: 10 * time.Millisecond,
		Log:            createTestLogger(),
	}
	game, err := NewGame(cfg)
	require.NoError(t, err)

	users := []*User{
		NewUser("short", "short", 0, 0),
		NewUser("deep", "deep", 0, 1),
	}
	game.SetPlayers(users)

	// Simulate balances: short < big blind, deep >> big blind
	game.players[0].balance = 10   // short stack
	game.players[1].balance = 1990 // deep stack

	startedCh := make(chan struct{}, 1)

	game.SetAutoStartCallbacks(&AutoStartCallbacks{
		MinPlayers: func() int { return 2 },
		StartNewHand: func() error {
			select {
			case startedCh <- struct{}{}:
			default:
			}
			return nil
		},
	})

	game.ScheduleAutoStart()

	select {
	case <-startedCh:
		// ok
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected auto-start to trigger with short-stacked player")
	}
}

// Verify that a short-stacked caller only contributes what they have, and
// their HasBet is NOT force-set to currentBet.
func TestCallShortStackAllInDoesNotForceMatchCurrentBet(t *testing.T) {
	cfg := GameConfig{
		NumPlayers:    2,
		StartingChips: 0,
		SmallBlind:    10,
		BigBlind:      20,
		Log:           createTestLogger(),
	}
	g, err := NewGame(cfg)
	if err != nil {
		t.Fatalf("NewGame error: %v", err)
	}

	users := []*User{
		NewUser("sb", "sb", 0, 0),
		NewUser("bb", "bb", 0, 1),
	}
	g.SetPlayers(users)

	// Simulate pre-flop state:
	// - currentBet is the big blind (20)
	// - SB has already posted 10 and only has 5 left
	// - BB has posted 20
	g.currentBet = 20
	g.players[0].currentBet = 10
	g.players[0].balance = 5
	g.players[1].currentBet = 20
	g.players[1].balance = 1000
	g.currentPlayer = 0 // SB to act

	// Debug: Check player state before call
	t.Logf("Before call - SB state: %s, balance: %d, currentBet: %d",
		g.players[0].GetCurrentStateString(), g.players[0].balance, g.players[0].currentBet)

	// SB tries to call but cannot fully match; should go all-in for +5 only.
	if err := g.handlePlayerCall("sb"); err != nil {
		t.Fatalf("handlePlayerCall error: %v", err)
	}

	// Debug: Check player state after call
	t.Logf("After call - SB state: %s, balance: %d, currentBet: %d",
		g.players[0].GetCurrentStateString(), g.players[0].balance, g.players[0].currentBet)

	// Give the state machine more time to process the evCall event
	time.Sleep(50 * time.Millisecond)
	t.Logf("After sleep - SB state: %s, balance: %d, currentBet: %d",
		g.players[0].GetCurrentStateString(), g.players[0].balance, g.players[0].currentBet)

	if g.players[0].balance != 0 {
		t.Fatalf("SB expected balance 0 after all-in call, got %d", g.players[0].balance)
	}
	if g.players[0].currentBet != 15 {
		t.Fatalf("SB expected currentBet 15 after all-in call, got %d", g.players[0].currentBet)
	}
	if got := g.players[0].GetCurrentStateString(); got != "ALL_IN" {
		t.Fatalf("SB expected state ALL_IN, got %s", got)
	}

	// The table-wide currentBet remains the big blind (20)
	if g.currentBet != 20 {
		t.Fatalf("expected table currentBet to remain 20, got %d", g.currentBet)
	}
}
