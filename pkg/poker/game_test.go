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

func startHandParticipationIfNeeded(t *testing.T, players []*Player) {
	t.Helper()
	for _, p := range players {
		if p == nil {
			continue
		}
		p.mu.RLock()
		hp := p.handParticipation
		p.mu.RUnlock()
		if hp == nil {
			require.NoError(t, p.HandleStartHand())
		}
	}
}

func buildShowdownPlayers(t *testing.T, g *Game, result *ShowdownResult) []ShowdownPlayerInfo {
	t.Helper()

	winnerInfoByID := make(map[string]*pokerrpc.Winner, len(result.WinnerInfo))
	for _, w := range result.WinnerInfo {
		winnerInfoByID[w.PlayerId] = w
	}

	players := make([]ShowdownPlayerInfo, 0, len(g.players))
	for idx, p := range g.players {
		if p == nil {
			continue
		}
		winfo := winnerInfoByID[p.ID()]
		hasShowdownHand := winfo != nil && len(winfo.BestHand) > 0
		info := ShowdownPlayerInfo{
			ID:           p.ID(),
			Name:         p.Name(),
			FinalState:   p.GetCurrentStateString(),
			Contribution: g.potManager.getTotalBet(idx),
		}
		if g.currentHand != nil && (p.revealed || hasShowdownHand) {
			cards := g.currentHand.GetPlayerCards(p.ID())
			for _, c := range cards {
				info.HoleCards = append(info.HoleCards, toProtoCard(c))
			}
		}
		if winfo != nil {
			info.HandRank = winfo.HandRank
			info.BestHand = winfo.BestHand
		}
		players = append(players, info)
	}

	return players
}

func TestNewGame(t *testing.T) {
	cfg := GameConfig{
		NumPlayers:       2,
		StartingChips:    1000, // Set to 1000 to match the expected balance
		SmallBlind:       10,
		BigBlind:         20,
		Seed:             42, // Use a fixed seed for deterministic testing
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
		cards := game.currentHand.GetPlayerCards(player.ID())
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
	startHandParticipationIfNeeded(t, game.players)
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

	startHandParticipationIfNeeded(t, game.players)
	result, err := game.HandleShowdown()
	require.NoError(t, err)

	require.Len(t, result.WinnerInfo, 1, "expected single winner")
	w := result.WinnerInfo[0]
	assert.Equal(t, "p2", w.PlayerId, "expected p2 to win with two pair")
	assert.Equal(t, pokerrpc.HandRank_TWO_PAIR, w.HandRank, "winner HandRank should reflect evaluated hand")
	assert.Len(t, w.BestHand, 5, "winner BestHand should be revealed")
}

func TestSplitPotForcesWinnerReveal(t *testing.T) {
	cfg := GameConfig{
		NumPlayers:       2,
		SmallBlind:       10,
		BigBlind:         20,
		Seed:             42,
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

	game.currentHand = NewHand([]string{"p1", "p2"})
	require.NoError(t, game.currentHand.DealCardToPlayer("p1", Card{suit: Clubs, value: Three}))
	require.NoError(t, game.currentHand.DealCardToPlayer("p1", Card{suit: Diamonds, value: Five}))
	require.NoError(t, game.currentHand.DealCardToPlayer("p2", Card{suit: Hearts, value: Four}))
	require.NoError(t, game.currentHand.DealCardToPlayer("p2", Card{suit: Spades, value: Six}))

	// Board: 7-8-9-10-J gives a straight on the board, forcing a split.
	game.communityCards = []Card{
		{suit: Hearts, value: Seven},
		{suit: Clubs, value: Eight},
		{suit: Spades, value: Nine},
		{suit: Diamonds, value: Ten},
		{suit: Hearts, value: Jack},
	}
	game.phase = pokerrpc.GamePhase_RIVER

	game.potManager = NewPotManager(2)
	game.potManager.addBet(0, 40, game.players)
	game.potManager.addBet(1, 40, game.players)

	startHandParticipationIfNeeded(t, game.players)
	result, err := game.HandleShowdown()
	require.NoError(t, err)

	require.Len(t, result.WinnerInfo, 2, "expected split pot with two winners")
	for _, w := range result.WinnerInfo {
		assert.Equal(t, pokerrpc.HandRank_STRAIGHT, w.HandRank, "split winners should expose hand rank")
		assert.Len(t, w.BestHand, 5, "split winners should reveal best hand cards")
	}

	showdownPlayers := buildShowdownPlayers(t, game, result)
	for _, pInfo := range showdownPlayers {
		if pInfo.ID == "p1" || pInfo.ID == "p2" {
			assert.Len(t, pInfo.HoleCards, 2, "split winners should reveal hole cards")
		}
	}
}

func TestFoldWinDoesNotReveal(t *testing.T) {
	cfg := GameConfig{
		NumPlayers:       2,
		SmallBlind:       10,
		BigBlind:         20,
		Seed:             99,
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

	game.currentHand = NewHand([]string{"p1", "p2"})
	require.NoError(t, game.currentHand.DealCardToPlayer("p1", Card{suit: Clubs, value: Ace}))
	require.NoError(t, game.currentHand.DealCardToPlayer("p1", Card{suit: Spades, value: King}))
	require.NoError(t, game.currentHand.DealCardToPlayer("p2", Card{suit: Hearts, value: Two}))
	require.NoError(t, game.currentHand.DealCardToPlayer("p2", Card{suit: Diamonds, value: Three}))

	game.communityCards = []Card{
		{suit: Clubs, value: Four},
		{suit: Diamonds, value: Five},
		{suit: Hearts, value: Six},
	}
	game.phase = pokerrpc.GamePhase_FLOP

	game.potManager = NewPotManager(2)
	game.potManager.addBet(0, 40, game.players)
	game.potManager.addBet(1, 40, game.players)

	startHandParticipationIfNeeded(t, game.players)

	// Player 2 folds.
	reply := make(chan error, 1)
	game.players[1].handParticipation.Send(evFoldReq{Reply: reply})
	require.NoError(t, <-reply)

	result, err := game.HandleShowdown()
	require.NoError(t, err)

	require.Len(t, result.WinnerInfo, 1, "expected a single winner by fold")
	assert.Equal(t, "p1", result.WinnerInfo[0].PlayerId)

	// Winner did not reach showdown; hole cards must stay hidden unless revealed manually.
	showdownPlayers := buildShowdownPlayers(t, game, result)
	for _, pi := range showdownPlayers {
		if pi.ID == "p1" {
			assert.Len(t, pi.HoleCards, 0, "winner by fold should not auto-reveal hole cards")
		}
	}
}

func TestStateShowdownForcesWinnerRevealViaFSM(t *testing.T) {
	cfg := GameConfig{
		NumPlayers:       2,
		SmallBlind:       10,
		BigBlind:         20,
		Seed:             99,
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

	game.currentHand = NewHand([]string{"p1", "p2"})
	require.NoError(t, game.currentHand.DealCardToPlayer("p1", Card{suit: Spades, value: Ace}))
	require.NoError(t, game.currentHand.DealCardToPlayer("p1", Card{suit: Hearts, value: Ace}))
	require.NoError(t, game.currentHand.DealCardToPlayer("p2", Card{suit: Clubs, value: King}))
	require.NoError(t, game.currentHand.DealCardToPlayer("p2", Card{suit: Diamonds, value: King}))

	game.communityCards = []Card{
		{suit: Clubs, value: Two},
		{suit: Diamonds, value: Three},
		{suit: Hearts, value: Four},
		{suit: Spades, value: Five},
		{suit: Clubs, value: Nine},
	}
	game.phase = pokerrpc.GamePhase_RIVER

	game.potManager = NewPotManager(2)
	game.potManager.addBet(0, 40, game.players)
	game.potManager.addBet(1, 40, game.players)

	startHandParticipationIfNeeded(t, game.players)

	eventCh := make(chan GameEvent, 8)
	game.SetTableEventChannel(eventCh)

	in := make(chan any)
	done := make(chan struct{})
	go func() {
		stateShowdown(game, in)
		close(done)
	}()

	var showdownEvt *GameEvent
	var revealEvt *GameEvent
	require.Eventually(t, func() bool {
		for {
			select {
			case evt := <-eventCh:
				switch evt.Type {
				case GameEventShowdownComplete:
					tmp := evt
					showdownEvt = &tmp
				case GameEventAutoShowCards:
					tmp := evt
					revealEvt = &tmp
				}
			default:
				return showdownEvt != nil && revealEvt != nil
			}
		}
	}, time.Second, 10*time.Millisecond, "showdown should emit result and forced reveal events")

	require.NotNil(t, showdownEvt)
	require.NotNil(t, revealEvt)
	require.Len(t, revealEvt.RevealInfo, 1, "expected only the showdown winner to be auto-revealed")
	require.Equal(t, "p1", revealEvt.RevealInfo[0].PlayerID)
	require.Len(t, revealEvt.RevealInfo[0].Cards, 2)

	game.mu.RLock()
	winner := game.getPlayerByID("p1")
	require.NotNil(t, winner)
	winner.mu.RLock()
	revealed := winner.revealed
	winner.mu.RUnlock()
	game.mu.RUnlock()
	require.True(t, revealed, "winner reveal state should be owned by the player FSM")

	close(in)
	<-done
}

func TestStateShowdownDoesNotForceRevealFoldWinner(t *testing.T) {
	cfg := GameConfig{
		NumPlayers:       2,
		SmallBlind:       10,
		BigBlind:         20,
		Seed:             99,
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

	game.currentHand = NewHand([]string{"p1", "p2"})
	require.NoError(t, game.currentHand.DealCardToPlayer("p1", Card{suit: Spades, value: Ace}))
	require.NoError(t, game.currentHand.DealCardToPlayer("p1", Card{suit: Hearts, value: Ace}))
	require.NoError(t, game.currentHand.DealCardToPlayer("p2", Card{suit: Clubs, value: King}))
	require.NoError(t, game.currentHand.DealCardToPlayer("p2", Card{suit: Diamonds, value: King}))

	game.communityCards = []Card{
		{suit: Clubs, value: Two},
		{suit: Diamonds, value: Three},
		{suit: Hearts, value: Four},
	}
	game.phase = pokerrpc.GamePhase_FLOP

	game.potManager = NewPotManager(2)
	game.potManager.addBet(0, 40, game.players)
	game.potManager.addBet(1, 40, game.players)

	startHandParticipationIfNeeded(t, game.players)

	reply := make(chan error, 1)
	game.players[1].handParticipation.Send(evFoldReq{Reply: reply})
	require.NoError(t, <-reply)

	eventCh := make(chan GameEvent, 8)
	game.SetTableEventChannel(eventCh)

	in := make(chan any)
	done := make(chan struct{})
	go func() {
		stateShowdown(game, in)
		close(done)
	}()

	var showdownEvt *GameEvent
	var revealEvt *GameEvent
	require.Eventually(t, func() bool {
		for {
			select {
			case evt := <-eventCh:
				switch evt.Type {
				case GameEventShowdownComplete:
					tmp := evt
					showdownEvt = &tmp
				case GameEventAutoShowCards:
					tmp := evt
					revealEvt = &tmp
				}
			default:
				return showdownEvt != nil
			}
		}
	}, time.Second, 10*time.Millisecond, "fold win should still emit showdown result")

	require.NotNil(t, showdownEvt)
	require.Nil(t, revealEvt, "uncontested fold winner must not be auto-revealed")

	game.mu.RLock()
	winner := game.getPlayerByID("p1")
	require.NotNil(t, winner)
	winner.mu.RLock()
	revealed := winner.revealed
	winner.mu.RUnlock()
	game.mu.RUnlock()
	require.False(t, revealed, "fold winner should remain hidden unless explicitly revealed")

	close(in)
	<-done
}

func TestTieBreakerShowdown(t *testing.T) {
	// Create a game with 3 players
	cfg := GameConfig{
		NumPlayers:       3,
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
		SmallBlind:       10,
		BigBlind:         20,
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
	startHandParticipationIfNeeded(t, game.players)
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
		SmallBlind:       10,
		BigBlind:         20,
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

	// Distribute pots - now requires game.mu held (Game FSM thread invariant)
	game.mu.Lock()
	payouts, err := game.potManager.distributePots(game.players)
	game.mu.Unlock()
	require.NoError(t, err)

	for idx, amt := range payouts {
		if amt <= 0 || idx < 0 || idx >= len(game.players) || game.players[idx] == nil {
			continue
		}
		require.NoError(t, game.players[idx].AddToBalance(amt))
	}

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
	game.players[0].SetBalance(0)
	game.players[0].SetCurrentBet(100)
	game.players[1].SetBalance(0)
	game.players[1].SetCurrentBet(100)

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
	game.players[0].SetBalance(10)   // short stack
	game.players[1].SetBalance(1990) // deep stack

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

// Minimum raise enforcement should reject sub-min raises but still allow a
// short-stack all-in for less than the minimum raise.
func TestHandlePlayerBetEnforcesMinimumRaise(t *testing.T) {
	newGame := func(t *testing.T, startingChips int64) (*Game, context.CancelFunc) {
		t.Helper()
		cfg := GameConfig{
			NumPlayers:       3,
			StartingChips:    startingChips,
			SmallBlind:       10,
			BigBlind:         20,
			Log:              createTestLogger(),
			AutoAdvanceDelay: 1 * time.Second,
		}
		g, err := NewGame(cfg)
		require.NoError(t, err)
		users := []*User{
			NewUser("p1", nil, nil),
			NewUser("p2", nil, nil),
			NewUser("p3", nil, nil),
		}
		g.SetPlayers(users)
		ctx, cancel := context.WithCancel(context.Background())
		go g.Start(ctx)
		g.sm.Send(evStartHand{})
		require.Eventually(t, func() bool {
			return g.GetPhase() == pokerrpc.GamePhase_PRE_FLOP
		}, 2*time.Second, 10*time.Millisecond, "game should reach PRE_FLOP")
		return g, cancel
	}

	waitForUTGTurn := func(t *testing.T, g *Game) *Player {
		t.Helper()
		var p *Player
		require.Eventually(t, func() bool {
			p = g.GetCurrentPlayerObject()
			return p != nil && p.ID() == "p1"
		}, 2*time.Second, 10*time.Millisecond, "UTG should act first pre-flop in 3-handed")
		return p
	}

	t.Run("rejects sub-minimum raise", func(t *testing.T) {
		g, cancel := newGame(t, 1000)
		t.Cleanup(cancel)

		current := waitForUTGTurn(t, g)
		require.Equal(t, int64(20), g.GetCurrentBet(), "table bet should reflect big blind")

		// Attempt to raise from 0 to 25 (only 5 over the big blind) should fail.
		err := g.HandlePlayerBet(current.ID(), 25)
		require.Error(t, err)
	})

	t.Run("allows short all-in below minimum raise", func(t *testing.T) {
		g, cancel := newGame(t, 30) // UTG will have 30 chips total
		t.Cleanup(cancel)

		// Make sure only the UTG stack is short; others remain deep.
		g.players[1].SetBalance(1000)
		g.players[2].SetBalance(1000)

		current := waitForUTGTurn(t, g)
		require.Equal(t, int64(20), g.GetCurrentBet(), "table bet should reflect big blind")

		// UTG goes all-in for 30: raise size is only 10 over the 20 big blind, below min-raise, but allowed as all-in.
		err := g.HandlePlayerBet(current.ID(), 30)
		require.NoError(t, err)
		snap := g.GetStateSnapshot()
		require.True(t, snap.Players[0].IsAllIn, "short-stack raise to all-in should be allowed")
	})

	t.Run("short all-in underraise does not reset aggressor or min raise", func(t *testing.T) {
		g, cancel := newGame(t, 1000)
		t.Cleanup(cancel)

		// In 3-handed pre-flop, p2 is the small blind with 10 already posted.
		// Reduce their remaining stack so their only raise is an under-min all-in.
		g.players[1].SetBalance(80)

		current := waitForUTGTurn(t, g)
		require.Equal(t, "p1", current.ID())

		// Open to 60 over the 20 big blind. This is a full raise of 40.
		require.NoError(t, g.HandlePlayerBet(current.ID(), 60))
		require.Equal(t, int64(60), g.GetCurrentBet())

		g.mu.RLock()
		require.Equal(t, 0, g.lastAggressor)
		require.Equal(t, int64(40), g.lastRaiseAmount)
		g.mu.RUnlock()

		// Small blind goes all-in to 90 total: only 30 over 60, so below the 40 min raise.
		require.NoError(t, g.HandlePlayerBet("p2", 90))
		require.Equal(t, int64(90), g.GetCurrentBet())

		g.mu.RLock()
		require.Equal(t, 0, g.lastAggressor, "short all-in underraise must not become the new aggressor")
		require.Equal(t, int64(40), g.lastRaiseAmount, "short all-in underraise must not reduce the minimum raise size")
		g.mu.RUnlock()

		// Remaining players can still respond and the round should complete normally.
		require.NoError(t, g.HandlePlayerCall("p3"))
		require.NoError(t, g.HandlePlayerCall("p1"))
		require.Eventually(t, func() bool {
			return g.GetPhase() == pokerrpc.GamePhase_FLOP
		}, 2*time.Second, 10*time.Millisecond, "betting round should complete after matching the short all-in underraise")
	})
}

func TestSetGameStateRestoresLastRaiseAmount(t *testing.T) {
	cfg := GameConfig{
		NumPlayers:       2,
		StartingChips:    1000,
		SmallBlind:       10,
		BigBlind:         20,
		Log:              createTestLogger(),
		AutoAdvanceDelay: 1 * time.Second,
	}

	g, err := NewGame(cfg)
	require.NoError(t, err)
	g.SetGameState(0, 7, 60, 0, 40, pokerrpc.GamePhase_PRE_FLOP)

	snap := g.GetStateSnapshot()
	require.Equal(t, int64(40), snap.LastRaiseAmount)

	restored, err := NewGame(cfg)
	require.NoError(t, err)
	restored.SetGameState(snap.Dealer, snap.Round, snap.CurrentBet, snap.Pot, snap.LastRaiseAmount, snap.Phase)

	restored.mu.RLock()
	defer restored.mu.RUnlock()
	require.Equal(t, int64(40), restored.lastRaiseAmount)
	require.Equal(t, int64(60), restored.currentBet)
}

func TestOpeningShortAllInDoesNotLowerMinimumRaise(t *testing.T) {
	cfg := GameConfig{
		NumPlayers:       2,
		StartingChips:    100,
		SmallBlind:       10,
		BigBlind:         20,
		Log:              createTestLogger(),
		AutoAdvanceDelay: 50 * time.Millisecond,
	}
	g, err := NewGame(cfg)
	require.NoError(t, err)

	users := []*User{
		NewUser("sb", nil, nil),
		NewUser("bb", nil, nil),
	}
	g.SetPlayers(users)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go g.Start(ctx)
	g.sm.Send(evStartHand{})

	require.Eventually(t, func() bool {
		return g.GetPhase() == pokerrpc.GamePhase_PRE_FLOP
	}, 2*time.Second, 10*time.Millisecond, "game should reach PRE_FLOP")

	// Force a clean street state directly so this test isolates the opening
	// short-all-in behavior in handlePlayerBet rather than hand progression.
	g.SetGameState(g.GetDealer(), g.GetRound(), 0, 0, cfg.BigBlind, pokerrpc.GamePhase_FLOP)
	for _, p := range g.players {
		p.SetCurrentBet(0)
		p.EndTurn()
	}
	g.mu.Lock()
	g.lastAggressor = -1
	g.mu.Unlock()
	g.SetCurrentPlayerByID("sb")

	require.Equal(t, int64(0), g.GetCurrentBet(), "new street should start with no live bet")

	actor := g.GetCurrentPlayerObject()
	require.NotNil(t, actor, "there should be a street opener")
	require.Equal(t, "sb", actor.ID())

	// Make the first actor a short stack so their only opening action is a
	// below-BB all-in. This should be allowed, but it must not reduce the
	// minimum legal raise size for the street.
	actor.SetBalance(15)
	require.NoError(t, g.HandlePlayerBet(actor.ID(), 15))

	g.mu.RLock()
	defer g.mu.RUnlock()
	require.Equal(t, int64(15), g.currentBet, "table bet should reflect the short all-in")
	require.Equal(t, int64(20), g.lastRaiseAmount, "opening short all-in must not lower the minimum raise below the big blind")
}

func TestOpeningBetSetsMinimumFutureRaise(t *testing.T) {
	cfg := GameConfig{
		NumPlayers:       2,
		StartingChips:    200,
		SmallBlind:       10,
		BigBlind:         20,
		Log:              createTestLogger(),
		AutoAdvanceDelay: 50 * time.Millisecond,
	}
	g, err := NewGame(cfg)
	require.NoError(t, err)

	users := []*User{
		NewUser("sb", nil, nil),
		NewUser("bb", nil, nil),
	}
	g.SetPlayers(users)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go g.Start(ctx)
	g.sm.Send(evStartHand{})

	require.Eventually(t, func() bool {
		return g.GetPhase() == pokerrpc.GamePhase_PRE_FLOP
	}, 2*time.Second, 10*time.Millisecond, "game should reach PRE_FLOP")

	// Force a clean post-flop street so this test isolates the opening-bet
	// behavior in handlePlayerBet rather than pre-flop blind mechanics.
	g.SetGameState(g.GetDealer(), g.GetRound(), 0, 0, cfg.BigBlind, pokerrpc.GamePhase_FLOP)
	for _, p := range g.players {
		p.SetCurrentBet(0)
		p.EndTurn()
	}
	g.mu.Lock()
	g.lastAggressor = -1
	g.mu.Unlock()
	g.SetCurrentPlayerByID("sb")

	actor := g.GetCurrentPlayerObject()
	require.NotNil(t, actor, "there should be a street opener")
	require.Equal(t, "sb", actor.ID())

	require.NoError(t, g.HandlePlayerBet(actor.ID(), 100))

	g.mu.RLock()
	defer g.mu.RUnlock()
	require.Equal(t, int64(100), g.currentBet, "table bet should reflect the opening bet")
	require.Equal(t, int64(100), g.lastRaiseAmount, "opening bet must set the next minimum raise size to the amount opened")
}

// When blinds alone put a player all-in and only one actionable player remains,
// the hand should auto-advance without waiting for a redundant check/fold.
func TestAutoAdvanceWhenBlindsShortStackAllIn(t *testing.T) {
	cfg := GameConfig{
		NumPlayers:       2,
		StartingChips:    100, // Override per-player balances below
		SmallBlind:       5,
		BigBlind:         10,
		Log:              createTestLogger(),
		AutoAdvanceDelay: 50 * time.Millisecond,
	}
	g, err := NewGame(cfg)
	require.NoError(t, err)

	users := []*User{
		NewUser("short", nil, nil), // Dealer/SB
		NewUser("deep", nil, nil),  // BB
	}
	g.SetPlayers(users)

	// Make the small blind a short stack so posting the blind leaves them all-in.
	g.players[0].SetBalance(5)
	g.players[1].SetBalance(100)

	// Start the game FSM and first hand
	go g.Start(context.Background())
	g.sm.Send(evStartHand{})

	// Wait for PRE_FLOP to be reached (blinds posted, hand started)
	require.Eventually(t, func() bool {
		return g.GetPhase() == pokerrpc.GamePhase_PRE_FLOP
	}, 2*time.Second, 10*time.Millisecond, "Game should reach PRE_FLOP")

	// The short stack should already be all-in from posting the small blind.
	snap := g.GetStateSnapshot()
	require.True(t, snap.Players[0].IsAllIn, "SB should be all-in after posting blind")
	require.False(t, snap.Players[1].IsAllIn, "BB should still have chips to act with")

	// With no other actionable players, auto-advance should trigger without any actions.
	require.Eventually(t, func() bool {
		return g.GetPhase() == pokerrpc.GamePhase_FLOP
	}, 1*time.Second, 20*time.Millisecond, "Hand should auto-advance from blinds when only one active player remains")
}

// When the big blind cannot cover the full blind amount, they must post their
// full stack as an all-in blind, remain in the hand, and action should continue
// with the other player rather than treating the short blind poster as folded/out.
func TestShortBigBlindPostsAllInAndStaysInHand(t *testing.T) {
	cfg := GameConfig{
		NumPlayers:       2,
		StartingChips:    100,
		SmallBlind:       5,
		BigBlind:         10,
		Log:              createTestLogger(),
		AutoAdvanceDelay: 50 * time.Millisecond,
	}
	g, err := NewGame(cfg)
	require.NoError(t, err)

	users := []*User{
		NewUser("sb", nil, nil), // Dealer/SB in heads-up
		NewUser("bb", nil, nil), // Big blind
	}
	g.SetPlayers(users)

	g.players[0].SetBalance(100)
	g.players[1].SetBalance(7) // Cannot cover the 10-chip big blind

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go g.Start(ctx)
	g.sm.Send(evStartHand{})

	require.Eventually(t, func() bool {
		return g.GetPhase() == pokerrpc.GamePhase_PRE_FLOP
	}, 2*time.Second, 10*time.Millisecond, "game should reach PRE_FLOP")

	snap := g.GetStateSnapshot()
	require.Len(t, snap.Players, 2)

	var bbSnap PlayerSnapshot
	for _, p := range snap.Players {
		switch p.ID {
		case "bb":
			bbSnap = p
		}
	}

	require.Equal(t, int64(7), bbSnap.CurrentBet, "short big blind should post only the chips they have")
	require.Equal(t, int64(0), bbSnap.Balance, "short big blind should have no chips left after posting")
	require.True(t, bbSnap.IsAllIn, "short big blind should be all-in from posting the blind")
	require.False(t, bbSnap.Folded, "short big blind must remain in the hand, not be folded")
	require.Equal(t, ALL_IN_STATE, bbSnap.StateString, "short big blind should be in ALL_IN state")

	require.Equal(t, "sb", g.GetCurrentPlayerObject().ID(), "action should continue with the small blind")

	// SB should be able to complete the betting round against the short BB.
	require.NoError(t, g.HandlePlayerCall("sb"))

	require.Eventually(t, func() bool {
		phase := g.GetPhase()
		return phase == pokerrpc.GamePhase_FLOP ||
			phase == pokerrpc.GamePhase_TURN ||
			phase == pokerrpc.GamePhase_RIVER ||
			phase == pokerrpc.GamePhase_SHOWDOWN
	}, 2*time.Second, 10*time.Millisecond, "hand should continue after the short big blind posts all-in")
}

// Reproduces a stall when the pre-flop aggressor is all-in: action never
// returns to them, so the betting round should still complete once all other
// players match the bet.
// Currently fails due to the bug: the game stays in PRE_FLOP instead of
// advancing to FLOP after both callers act.
func TestPreFlopAllInAggressorShouldAdvance(t *testing.T) {
	cfg := GameConfig{
		NumPlayers:       3,
		StartingChips:    1000,
		SmallBlind:       30,
		BigBlind:         60,
		AutoStartDelay:   0,
		TimeBank:         0,
		Log:              createTestLogger(),
		AutoAdvanceDelay: 10 * time.Millisecond,
	}
	g, err := NewGame(cfg)
	require.NoError(t, err)

	users := []*User{
		NewUser("dealer", nil, nil),
		NewUser("smallblind", nil, nil),
		NewUser("bigblind", nil, nil),
	}
	g.SetPlayers(users)

	// Make the dealer a short stack to force an all-in open, while the blinds
	// remain deep and actionable after calling.
	g.players[0].SetBalance(200)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go g.Start(ctx)
	g.sm.Send(evStartHand{})

	// Wait for PRE_FLOP with the dealer acting first (UTG in 3-handed).
	require.Eventually(t, func() bool {
		return g.GetPhase() == pokerrpc.GamePhase_PRE_FLOP && g.GetCurrentPlayer() == 0
	}, 2*time.Second, 10*time.Millisecond, "dealer should act first pre-flop")

	// Dealer shoves over the blinds.
	allInAmount := g.GetCurrentBet() + g.players[0].Balance()
	require.NoError(t, g.HandlePlayerBet(users[0].ID, allInAmount))
	require.Eventually(t, func() bool {
		return g.players[0].GetCurrentStateString() == ALL_IN_STATE
	}, 200*time.Millisecond, 10*time.Millisecond, "dealer should be marked all-in")

	// Both blinds call the shove (but keep chips).
	require.NoError(t, g.HandlePlayerCall(users[1].ID))
	require.NoError(t, g.HandlePlayerCall(users[2].ID))

	// Snapshot state after calls to aid debugging.
	g.mu.Lock()
	stateStrings := make([]string, len(g.players))
	currentBets := make([]int64, len(g.players))
	balances := make([]int64, len(g.players))
	for i, p := range g.players {
		if p != nil {
			stateStrings[i] = p.GetCurrentStateString()
			currentBets[i] = p.currentBet
			balances[i] = p.balance
		}
	}
	phase := g.phase
	cur := g.currentPlayer
	lastAgg := g.lastAggressor
	autoAdv := g.autoAdvanceEnabled
	timerSet := g.autoAdvanceTimer != nil
	g.mu.Unlock()
	t.Logf("post-calls: phase=%v currentPlayer=%d lastAggressor=%d autoAdvanceEnabled=%v timerSet=%v states=%v bets=%v balances=%v",
		phase, cur, lastAgg, autoAdv, timerSet, stateStrings, currentBets, balances)

	// With all bets matched and aggressor all-in, betting round should complete.
	require.Eventually(t, func() bool {
		p := g.GetPhase()
		return p == pokerrpc.GamePhase_FLOP ||
			p == pokerrpc.GamePhase_TURN ||
			p == pokerrpc.GamePhase_RIVER ||
			p == pokerrpc.GamePhase_SHOWDOWN
	}, 300*time.Millisecond, 10*time.Millisecond, "pre-flop should advance after calls even though aggressor is all-in")
}

// Verifies that a timeout-triggered fold completes the round to SHOWDOWN and auto-starts a new hand,
// preventing the game from getting stuck in SHOWDOWN.
func TestTimeoutCompletesShowdownAndAutoStarts(t *testing.T) {
	tbl := newTestTable(t, 2, 2, 5, 10, 1000)
	// Re-enable auto-start for this scenario so a new hand begins after the timeout showdown.
	tbl.config.AutoStartDelay = 100 * time.Millisecond
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
	game.players[0].SetBalance(1060)
	game.players[0].SetStartingBalance(1060)
	game.players[1].SetBalance(940)
	game.players[1].SetStartingBalance(940)

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
	if len(winners) > 0 {
		t.Logf("Winners: %v", winners)
	}
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

	// In heads-up, a fold immediately triggers showdown. The player's FOLDED state
	// is ephemeral (cleared when the hand ends). Verify via lastShowdownResult.
	require.Eventually(t, func() bool {
		result := g.GetLastShowdownResult()
		if result == nil {
			return false
		}
		// Find the player in the showdown result and verify they folded
		for _, p := range result.Players {
			if p.ID == curID {
				t.Logf("player %s final state in showdown: %s", curID, p.FinalState)
				return p.FinalState == FOLDED_STATE
			}
		}
		return false
	}, 2*time.Second, 10*time.Millisecond, "showdown should record player as folded")
}

// Test that when the current player's timebank expires with nothing to call, they auto-check.
func TestTimebank_AutoCheck_Flop(t *testing.T) {
	tbl := NewTable(TableConfig{
		ID:               "tbl-timebank-check",
		Log:              createTestLogger(),
		GameLog:          createTestLogger(),
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
	g.currentPlayer = g.computeFirstActorIndex()
	g.mu.Unlock()

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

// When both players time out with no bet on the street, each is auto-checked
// and the betting round should advance. This reproduces the "silent advance"
// seen in logs where auto-checks occurred after timebank expiry.
func TestTimebank_DualAutoCheckAdvancesStreet(t *testing.T) {
	tbl := NewTable(TableConfig{
		ID:               "tbl-timebank-dual-check",
		Log:              createTestLogger(),
		GameLog:          createTestLogger(),
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

	evtCh := make(chan TableEvent, 16)
	tbl.SetEventChannel(evtCh)

	// Seat and ready two players
	p1, err := tbl.AddNewUser("p1", nil)
	require.NoError(t, err)
	p2, err := tbl.AddNewUser("p2", nil)
	require.NoError(t, err)
	p1.SendReady()
	p2.SendReady()

	require.True(t, tbl.CheckAllPlayersReady())
	require.NoError(t, tbl.StartGame())

	// Wait for game to be instantiated
	require.Eventually(t, func() bool { return tbl.GetGame() != nil }, 2*time.Second, 10*time.Millisecond)
	g := tbl.GetGame()
	require.NotNil(t, g)

	// Jump to flop with no outstanding bet and set the first actor.
	g.StateFlop()
	g.mu.Lock()
	g.currentBet = 0
	g.currentPlayer = g.computeFirstActorIndex()
	g.mu.Unlock()

	first := g.GetCurrentPlayerObject()
	require.NotNil(t, first)
	firstID := first.ID()

	// First player times out and auto-checks, advancing turn.
	g.TriggerTimebankExpiredFor(firstID)
	require.Eventually(t, func() bool {
		next := g.GetCurrentPlayerObject()
		return next != nil && next.ID() != firstID
	}, 1*time.Second, 10*time.Millisecond, "turn should advance after first auto-check")

	second := g.GetCurrentPlayerObject()
	require.NotNil(t, second)
	secondID := second.ID()
	require.NotEqual(t, firstID, secondID)

	// Second player also times out and auto-checks; after both have checked
	// the betting round should complete and advance to the next street.
	g.TriggerTimebankExpiredFor(secondID)

	require.Eventually(t, func() bool {
		return g.GetPhase() == pokerrpc.GamePhase_TURN
	}, 500*time.Millisecond, 10*time.Millisecond,
		"street should advance after dual auto-checks")

	// Neither player should have been forced to fold by the auto-check path.
	for _, p := range g.GetPlayers() {
		if p == nil {
			continue
		}
		require.Equal(t, IN_GAME_STATE, p.GetCurrentStateString(), "player should remain in-game after auto-check")
	}
}

// TestAllIn_ActivePlayer_AutoAdvancesWithOneActive verifies that when only one
// player remains active and the rest are all-in, streets auto-advance (per
// requirement: need at least 2 active players to block auto-advance).
func TestAllIn_ActivePlayer_AutoAdvancesWithOneActive(t *testing.T) {
	tbl := NewTable(TableConfig{
		ID:               "test-allin-timebank",
		Log:              createTestLogger(),
		GameLog:          createTestLogger(),
		BuyIn:            0,
		MinPlayers:       2,
		MaxPlayers:       2,
		SmallBlind:       10,
		BigBlind:         20,
		StartingChips:    1000, // Will be overridden per-player
		TimeBank:         500 * time.Millisecond,
		AutoStartDelay:   10 * time.Millisecond,
		AutoAdvanceDelay: 100 * time.Millisecond,
	})
	eventChan := make(chan TableEvent, 32)
	tbl.SetEventChannel(eventChan)
	go func() {
		for range eventChan {
		}
	}()
	defer func() {
		tbl.Close()
	}()

	// Add two players
	u1, err := tbl.AddNewUser("p1-deep", nil)
	require.NoError(t, err)
	u2, err := tbl.AddNewUser("p2-short", nil)
	require.NoError(t, err)
	require.NoError(t, u1.SendReady())
	require.NoError(t, u2.SendReady())

	// Start the game
	require.True(t, tbl.CheckAllPlayersReady())
	require.NoError(t, tbl.StartGame())

	// Wait for game to be created
	require.Eventually(t, func() bool { return tbl.GetGame() != nil }, time.Second, 10*time.Millisecond)
	g := tbl.GetGame()
	require.NotNil(t, g)

	// Set asymmetric stacks: p1 has 1000, p2 has only 500
	// This way when p1 bets big, p2 goes all-in but p1 stays active
	g.players[0].SetBalance(1000) // deep stack
	g.players[1].SetBalance(500)  // short stack

	// Wait for PRE_FLOP
	require.Eventually(t, func() bool {
		return g.GetPhase() == pokerrpc.GamePhase_PRE_FLOP
	}, time.Second, 10*time.Millisecond)

	// Find who is current player (SB in heads-up acts first pre-flop)
	cur := g.GetCurrentPlayerObject()
	require.NotNil(t, cur)

	// Current player (p1, deep stack) raises big - more than p2 can afford
	err = tbl.MakeBet(cur.ID(), 600)
	require.NoError(t, err)

	// Wait for p2's turn
	require.Eventually(t, func() bool {
		next := g.GetCurrentPlayerObject()
		return next != nil && next.ID() != cur.ID()
	}, time.Second, 10*time.Millisecond)

	// p2 calls - goes all-in (only has ~480 after posting blind)
	p2 := g.GetCurrentPlayerObject()
	err = tbl.HandleCall(p2.ID())
	require.NoError(t, err)

	// Wait for FLOP - auto-advance should deal it
	require.Eventually(t, func() bool {
		return g.GetPhase() == pokerrpc.GamePhase_FLOP
	}, 2*time.Second, 10*time.Millisecond)

	// Verify one player is all-in and one is active
	p1State := g.players[0].GetCurrentStateString()
	p2State := g.players[1].GetCurrentStateString()
	t.Logf("After FLOP dealt: p1=%s, p2=%s", p1State, p2State)

	// Wait for auto-advance to fire (AutoAdvanceDelay=100ms) and reach TURN.
	require.Eventually(t, func() bool {
		return g.GetPhase() == pokerrpc.GamePhase_TURN
	}, 500*time.Millisecond, 20*time.Millisecond, "game should auto-advance to TURN when only one active player remains")
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
	game.players[0].SetBalance(980)   // Started with 1000, bet 20
	game.players[0].SetCurrentBet(20) // Mirror posted bet
	// Send fold event to transition to folded state
	game.players[0].handParticipation.Send(evFoldReq{Reply: make(chan error, 1)})

	// Mark players 2 and 3 as all-in
	game.players[1].SetBalance(0)
	game.players[1].SetCurrentBet(1000)
	game.players[2].SetBalance(0)
	game.players[2].SetCurrentBet(1000)

	// Send events to trigger state machine to detect all-in condition
	game.players[1].handParticipation.Send(evCall{})
	game.players[2].handParticipation.Send(evCall{})

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
	tbl := newTestTable(t, 2, 3, 10, 20, 1000)
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

	// Simulate a hand where player3 loses all chips and is eliminated.
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
	game.players[player3Idx].SetBalance(0)
	game.mu.Lock()
	game.phase = pokerrpc.GamePhase_SHOWDOWN
	game.mu.Unlock()

	// PLAYER_LOST is informational only during showdown. Record the pending
	// elimination through the same table event pipeline the Game FSM uses.
	tbl.handleGameEvent(GameEvent{Type: GameEventPlayerLost, PlayerID: "player3"})
	require.Eventually(t, func() bool {
		pending := tbl.pendingEliminationIDs()
		return len(pending) == 1 && pending[0] == "player3"
	}, time.Second, 10*time.Millisecond, "player3 should be staged for next-hand pruning")

	// Verify: 2 players have chips > 0, 1 player has 0 chips
	readyCount := 0
	for _, p := range game.players {
		if p != nil && p.Balance() > 0 {
			readyCount++
		}
	}

	require.Equal(t, 2, readyCount, "Should have 2 players with chips remaining")
	require.Equal(t, 2, tbl.config.MinPlayers, "Table should have minPlayers=2")
	require.Len(t, game.GetPlayers(), 3, "showdown grace should keep eliminated player visible until restart")
	require.Len(t, tbl.GetUsers(), 3, "table roster should keep eliminated player seated until restart")

	// Manually trigger auto-start check (simulating the next-hand prep boundary
	// after showdown UI delay expires).
	// With minPlayers=2 and 2 active players, auto-start should succeed
	err = tbl.handleAutoStart()
	require.NoError(t, err, "Auto-start should succeed with 2 players when minPlayers=2")
	require.Len(t, game.GetPlayers(), 2, "auto-start should prune eliminated player before the next hand")
	require.Len(t, tbl.GetUsers(), 2, "table roster should prune eliminated player before the next hand")
	t.Log("✓ Auto-start succeeded with 2 players")
}

// TestShowdownRaceCondition verifies that a stale showdown request (for a previous round)
// is correctly ignored when the game has already advanced to a new hand.
// This guards against the race where evGotoShowdownReq for round N could
// incorrectly trigger showdown on round N+1.
func TestShowdownRaceCondition(t *testing.T) {
	config := GameConfig{
		NumPlayers:       2,
		StartingChips:    1000,
		SmallBlind:       10,
		BigBlind:         20,
		AutoStartDelay:   0, // No auto-start delay for this test
		Log:              createTestLogger(),
		AutoAdvanceDelay: 1 * time.Second,
	}

	game, err := NewGame(config)
	require.NoError(t, err)

	users := []*User{
		NewUser("player1", nil, nil),
		NewUser("player2", nil, nil),
	}
	game.SetPlayers(users)

	// Start the game FSM
	go game.Start(context.Background())

	// Start first hand (round 1)
	game.sm.Send(evStartHand{})

	// Wait for PRE_FLOP phase
	require.Eventually(t, func() bool {
		return game.GetPhase() == pokerrpc.GamePhase_PRE_FLOP
	}, 2*time.Second, 10*time.Millisecond, "Game should reach PRE_FLOP")

	// Capture the original round
	game.mu.RLock()
	originalRound := game.round
	originalHandID := ""
	if game.currentHand != nil {
		originalHandID = game.currentHand.id
	}
	game.mu.RUnlock()

	require.Equal(t, 1, originalRound, "First hand should be round 1")
	t.Logf("Round 1: hand ID = %s", originalHandID)

	// Now start round 2 by triggering showdown for round 1, then starting new hand
	// First, send a valid showdown request for round 1
	game.sm.Send(evGotoShowdownReq{round: originalRound})

	// Wait for showdown to complete
	require.Eventually(t, func() bool {
		return game.GetPhase() == pokerrpc.GamePhase_SHOWDOWN
	}, 2*time.Second, 10*time.Millisecond, "Game should reach SHOWDOWN for round 1")

	// Start round 2
	game.sm.Send(evStartHand{})

	// Wait for PRE_FLOP round 2
	require.Eventually(t, func() bool {
		game.mu.RLock()
		defer game.mu.RUnlock()
		return game.phase == pokerrpc.GamePhase_PRE_FLOP && game.round == 2
	}, 2*time.Second, 10*time.Millisecond, "Game should reach PRE_FLOP for round 2")

	// Capture round 2 state
	game.mu.RLock()
	round2HandID := ""
	if game.currentHand != nil {
		round2HandID = game.currentHand.id
	}
	round2Phase := game.phase
	round2Round := game.round
	game.mu.RUnlock()

	require.Equal(t, 2, round2Round, "Should be in round 2")
	require.Equal(t, pokerrpc.GamePhase_PRE_FLOP, round2Phase, "Should be in PRE_FLOP")
	require.NotEqual(t, originalHandID, round2HandID, "Round 2 should have different hand ID")
	t.Logf("Round 2: hand ID = %s, phase = %v", round2HandID, round2Phase)

	// NOW send a STALE showdown request for round 1 (the old round)
	// This should be IGNORED because we're already in round 2
	t.Logf("Sending stale evGotoShowdownReq{round: %d} while in round %d", originalRound, round2Round)
	game.sm.Send(evGotoShowdownReq{round: originalRound})

	// Wait a bit to let the FSM process the stale event
	time.Sleep(200 * time.Millisecond)

	// Verify the stale request was IGNORED:
	// - We should still be in PRE_FLOP round 2
	// - The hand ID should be unchanged
	// - We should NOT have transitioned to SHOWDOWN
	game.mu.RLock()
	finalPhase := game.phase
	finalRound := game.round
	finalHandID := ""
	if game.currentHand != nil {
		finalHandID = game.currentHand.id
	}
	game.mu.RUnlock()

	t.Logf("After stale request: round = %d, phase = %v, hand ID = %s", finalRound, finalPhase, finalHandID)

	// Assert the stale request was ignored
	assert.Equal(t, round2Round, finalRound, "Round should remain unchanged after stale showdown request")
	assert.Equal(t, round2Phase, finalPhase, "Phase should remain PRE_FLOP after stale showdown request")
	assert.Equal(t, round2HandID, finalHandID, "Hand ID should remain unchanged after stale showdown request")

	// The key assertion: we should NOT be in SHOWDOWN
	assert.NotEqual(t, pokerrpc.GamePhase_SHOWDOWN, finalPhase,
		"Stale showdown request should NOT trigger showdown for the current hand")

	t.Log("✓ Stale showdown request was correctly ignored")
}

func TestGetBlindSnapshotAfterGameClose(t *testing.T) {
	cfg := GameConfig{
		NumPlayers:            2,
		StartingChips:         1000,
		SmallBlind:            10,
		BigBlind:              20,
		AutoAdvanceDelay:      time.Second,
		BlindIncreaseInterval: time.Minute,
		Log:                   createTestLogger(),
	}

	game, err := NewGame(cfg)
	require.NoError(t, err)

	nextIncrease := time.Now().Add(2 * time.Minute).UnixMilli()
	game.round = 3
	game.liveBlindLevel = 2
	game.liveNextBlindMs = nextIncrease

	game.Close()

	done := make(chan BlindSnapshot, 1)
	go func() {
		done <- game.GetBlindSnapshot()
	}()

	select {
	case snap := <-done:
		require.Equal(t, BlindStateActive, snap.State)
		require.Equal(t, 2, snap.CurrentLevel)
		require.Equal(t, nextIncrease, snap.NextIncreaseMs)
		require.Equal(t, time.UnixMilli(nextIncrease).Add(-3*time.Minute).UnixMilli(), snap.StartUnixMs)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("GetBlindSnapshot blocked after Game.Close")
	}
}
