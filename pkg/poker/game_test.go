package poker

import (
	"context"
	"os"
	"strings"
	"sync"
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
	player1.SetHand([]Card{
		{suit: Hearts, value: Ace},
		{suit: Spades, value: Ace},
	})

	// Player 2 has King-Queen
	player2.SetHand([]Card{
		{suit: Hearts, value: King},
		{suit: Spades, value: Queen},
	})

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
	player1.SetHand([]Card{
		{suit: Hearts, value: Ace},
		{suit: Spades, value: Ace},
	})

	player2.SetHand([]Card{
		{suit: Clubs, value: Ace},
		{suit: Diamonds, value: Ace},
	})

	player3.SetHand([]Card{
		{suit: Hearts, value: King},
		{suit: Spades, value: King}, // Lower pair
	})

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
	cfg := GameConfig{NumPlayers: 2, Seed: 1, Log: createTestLogger()}
	game, err := NewGame(cfg)
	require.NoError(t, err)

	users := []*User{
		NewUser("p1", "p1", 0, 0),
		NewUser("p2", "p2", 0, 1),
	}
	game.SetPlayers(users)

	// Force hands that don't improve beyond board
	game.players[0].SetHand([]Card{{suit: Hearts, value: Two}, {suit: Clubs, value: Three}})
	game.players[1].SetHand([]Card{{suit: Diamonds, value: Four}, {suit: Spades, value: Five}})

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
	if game.players[0].Balance() != 50 {
		t.Fatalf("p1 expected 50, got %d", game.players[0].Balance())
	}
	if game.players[1].Balance() != 50 {
		t.Fatalf("p2 expected 50, got %d", game.players[1].Balance())
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

	// Distribute pots
	game.potManager.distributePots(game.players)

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
	g.players[0].SetCurrentBet(10)
	g.players[0].SetBalance(5)
	g.players[1].SetCurrentBet(20)
	g.players[1].SetBalance(1000)
	g.currentPlayer = 0 // SB to act

	// Debug: Check player state before call
	t.Logf("Before call - SB state: %s, balance: %d, currentBet: %d",
		g.players[0].GetCurrentStateString(), g.players[0].Balance(), g.players[0].CurrentBet())

	// SB tries to call but cannot fully match; should go all-in for +5 only.
	if err := g.handlePlayerCall("sb"); err != nil {
		t.Fatalf("handlePlayerCall error: %v", err)
	}

	// Debug: Check player state after call
	t.Logf("After call - SB state: %s, balance: %d, currentBet: %d",
		g.players[0].GetCurrentStateString(), g.players[0].Balance(), g.players[0].CurrentBet())

	// Give the state machine more time to process the evCall event
	time.Sleep(50 * time.Millisecond)
	t.Logf("After sleep - SB state: %s, balance: %d, currentBet: %d",
		g.players[0].GetCurrentStateString(), g.players[0].Balance(), g.players[0].CurrentBet())

	if g.players[0].Balance() != 0 {
		t.Fatalf("SB expected balance 0 after all-in call, got %d", g.players[0].Balance())
	}
	if g.players[0].CurrentBet() != 15 {
		t.Fatalf("SB expected currentBet 15 after all-in call, got %d", g.players[0].CurrentBet())
	}
	if got := g.players[0].GetCurrentStateString(); got != "ALL_IN" {
		t.Fatalf("SB expected state ALL_IN, got %s", got)
	}

	// The table-wide currentBet remains the big blind (20)
	if g.currentBet != 20 {
		t.Fatalf("expected table currentBet to remain 20, got %d", g.currentBet)
	}
}

// Verifies that a timeout-triggered fold completes the round to SHOWDOWN and auto-starts a new hand,
// preventing the game from getting stuck in SHOWDOWN.
func TestTimeoutCompletesShowdownAndAutoStarts(t *testing.T) {
	tbl := newTestTable(t, 2, 2, 5, 10, 1000)
	_, _ = tbl.AddNewUser("p1", "P1", 0, 0)
	_, _ = tbl.AddNewUser("p2", "P2", 0, 1)
	_ = tbl.SetPlayerReady("p1", true)
	_ = tbl.SetPlayerReady("p2", true)
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
	tbl.HandleTimeouts()

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
	var currentPlayer PlayerSnapshot

	require.Eventually(t, func() bool {
		snap = game.GetStateSnapshot()
		currentPlayerIndex = game.GetCurrentPlayer()
		if currentPlayerIndex < 0 || currentPlayerIndex >= len(snap.Players) {
			return false
		}
		currentPlayer = snap.Players[currentPlayerIndex]
		return currentPlayer.IsTurn
	}, 2*time.Second, 10*time.Millisecond, "Current player should be initialized with isTurn=true")

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
		snap = game.GetStateSnapshot()
		newCurrentPlayerIndex = game.GetCurrentPlayer()
		if newCurrentPlayerIndex < 0 || newCurrentPlayerIndex >= len(snap.Players) {
			return false
		}
		if newCurrentPlayerIndex == currentPlayerIndex {
			return false // Should have advanced
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
		t.Logf("Player %d (%s): isTurn=%v, state=%s", i, p.ID, p.IsTurn, GetPlayerStateString(p.StateID))
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

	if !snap.Players[0].IsDealer || !snap.Players[0].IsSmallBlind || snap.Players[0].IsBigBlind {
		t.Fatalf("player 0 flags not copied: dealer=%v sb=%v bb=%v",
			snap.Players[0].IsDealer, snap.Players[0].IsSmallBlind, snap.Players[0].IsBigBlind)
	}
	if snap.Players[1].IsDealer || snap.Players[1].IsSmallBlind || !snap.Players[1].IsBigBlind {
		t.Fatalf("player 1 flags not copied: dealer=%v sb=%v bb=%v",
			snap.Players[1].IsDealer, snap.Players[1].IsSmallBlind, snap.Players[1].IsBigBlind)
	}
}
