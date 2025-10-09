package poker

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/decred/slog"

	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	"github.com/vctt94/pokerbisonrelay/pkg/statemachine"
)

// GameStateFn represents a game state function following Rob Pike's pattern
type GameStateFn = statemachine.StateFn[Game]

// Hole represents a player's hole cards for a single hand
type Hole struct {
	cards    [2]Card
	count    int  // number of cards dealt (0, 1, or 2)
	revealed bool // true if revealed at showdown
	mucked   bool // eligible but not revealed (lost at showdown)
}

// NewHole creates a new empty Hole
func NewHole() *Hole {
	return &Hole{
		cards: [2]Card{},
		count: 0,
	}
}

// AddCard adds a card to the hole (max 2 cards)
func (h *Hole) AddCard(c Card) error {
	if h.count >= 2 {
		return fmt.Errorf("hole already has 2 cards")
	}
	h.cards[h.count] = c
	h.count++
	return nil
}

// GetCards returns the hole cards (visible only per retention policy)
func (h *Hole) GetCards() []Card {
	if h.count == 0 {
		return nil
	}
	return h.cards[:h.count]
}

// Reveal marks the hole as revealed at showdown
func (h *Hole) Reveal() {
	h.revealed = true
}

// Muck marks the hole as mucked at showdown
func (h *Hole) Muck() {
	h.mucked = true
}

// Clear purges the hole cards (called during cleanup)
func (h *Hole) Clear() {
	h.cards = [2]Card{}
	h.count = 0
	// Keep revealed/mucked flags for audit
}

// ActionLog represents a single action taken during a hand
type ActionLog struct {
	Timestamp time.Time
	PlayerID  string
	Action    string // "bet", "call", "check", "fold", "blind"
	Amount    int64
	Epoch     int // optional: action epoch for staleness detection
}

// Settlement represents final payout information for a hand
type Settlement struct {
	PlayerID string
	Amount   int64
	Reason   string // "win", "tie", "refund"
}

// Hand represents all per-hand state, owned by Game
// Created at hand start; destroyed after cleanup
type Hand struct {
	id        string
	phase     string // "PREDEAL", "BETTING", "SHOWDOWN", "SETTLEMENT", "CLEANUP"
	street    pokerrpc.GamePhase
	board     []Card
	hole      map[string]*Hole // playerID -> private hole cards
	actions   []ActionLog
	results   []Settlement
	createdAt time.Time
	finalized bool

	mu sync.RWMutex
}

// NewHand creates a new Hand for the given players
func NewHand(playerIDs []string) *Hand {
	h := &Hand{
		id:        fmt.Sprintf("hand_%d", time.Now().UnixNano()),
		phase:     "PREDEAL",
		hole:      make(map[string]*Hole),
		actions:   make([]ActionLog, 0),
		results:   make([]Settlement, 0),
		createdAt: time.Now(),
	}

	// Initialize empty Hole for each player
	for _, pid := range playerIDs {
		h.hole[pid] = NewHole()
	}

	return h
}

// DealCardToPlayer deals a card to a specific player
func (h *Hand) DealCardToPlayer(playerID string, card Card) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	hole, ok := h.hole[playerID]
	if !ok {
		return fmt.Errorf("player %s not in hand", playerID)
	}

	return hole.AddCard(card)
}

// GetPlayerCards returns the cards for a specific player (respecting visibility)
func (h *Hand) GetPlayerCards(playerID string, requestorID string) []Card {
	h.mu.RLock()
	defer h.mu.RUnlock()

	hole, ok := h.hole[playerID]
	if !ok {
		return nil
	}

	// Visibility rules:
	// 1. Owner can always see their own cards
	// 2. Revealed cards are visible to everyone
	// 3. Mucked cards are only visible to owner
	if playerID == requestorID || hole.revealed {
		return hole.GetCards()
	}

	return nil // Not visible
}

// RevealPlayerCards marks a player's cards as revealed at showdown
func (h *Hand) RevealPlayerCards(playerID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if hole, ok := h.hole[playerID]; ok {
		hole.Reveal()
	}
}

// MuckPlayerCards marks a player's cards as mucked at showdown
func (h *Hand) MuckPlayerCards(playerID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if hole, ok := h.hole[playerID]; ok {
		hole.Muck()
	}
}

// CleanupHoleCards purges all hole card data (called after settlement)
func (h *Hand) CleanupHoleCards() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, hole := range h.hole {
		hole.Clear()
	}

	h.phase = "CLEANUP"
	h.finalized = true
}

// GameConfig holds configuration for a new game
type GameConfig struct {
	NumPlayers     int
	StartingChips  int64         // Fixed number of chips each player starts with
	SmallBlind     int64         // Small blind amount
	BigBlind       int64         // Big blind amount
	Seed           int64         // Optional seed for deterministic games
	AutoStartDelay time.Duration // Delay before automatically starting next hand after showdown
	TimeBank       time.Duration // Time bank for each player
	Log            slog.Logger   // Logger for game events
}

// AutoStartCallbacks defines the callback functions needed for auto-start functionality
type AutoStartCallbacks struct {
	MinPlayers func() int
	// StartNewHand should start a new hand
	StartNewHand func() error
	// OnNewHandStarted is called after a new hand has been successfully started
	OnNewHandStarted func()
}

// GameEventType represents different types of game events sent to Table
type GameEventType int

const (
	GameEventBettingRoundComplete GameEventType = iota // Betting round complete, advance to next street
	GameEventShowdownReady                             // All players all-in or only one active, go to showdown
	GameEventShowdownComplete                          // Showdown processing complete, result available
)

// GameEvent represents an event sent from Game FSM to Table
type GameEvent struct {
	Type           GameEventType
	ShowdownResult *ShowdownResult // Only set for GameEventShowdownComplete
}

// Game holds the context and data for our poker game
type Game struct {
	mu RWLock
	// Player management - references to table users converted to players
	players       []*Player // Internal player objects managed by game
	currentPlayer int
	dealer        int

	// Current hand context (per-hand state)
	currentHand *Hand

	// Cards
	deck           *Deck
	communityCards []Card

	// Game state
	potManager     *potManager
	currentBet     int64
	round          int
	betRound       int // Tracks which betting round (pre-flop, flop, turn, river)
	actionsInRound int // Track actions in current betting round

	// Configuration
	config GameConfig

	// Auto-start management
	autoStartTimer     *time.Timer
	autoStartCanceled  bool
	autoStartCallbacks *AutoStartCallbacks

	// Logger
	log slog.Logger

	// For demonstration purposes
	errorSimulation bool
	maxRounds       int

	// current game phase (pre-flop, flop, turn, river, showdown)
	phase pokerrpc.GamePhase

	// Winner tracking - set after showdown is complete
	winners []string

	// State machine - Rob Pike's pattern
	sm *statemachine.Machine[Game]

	// Lifecycle notifications - states signal when they're reached
	preFlopReached chan struct{}

	// Event channel for sending events to Table (betting round complete, showdown, etc.)
	tableEventChan chan<- GameEvent
}

// SetTableEventChannel sets the channel for sending events to the Table
func (g *Game) SetTableEventChannel(ch chan<- GameEvent) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.tableEventChan = ch
}

// sendTableEvent sends an event to the Table (non-blocking)
func (g *Game) sendTableEvent(event GameEvent) {
	if g.tableEventChan != nil {
		select {
		case g.tableEventChan <- event:
		default:
			// Channel full or closed, log and continue
			g.log.Warnf("Failed to send game event %v to table (channel full)", event.Type)
		}
	}
}

// NewGame creates a new poker game with the given configuration
// Players are managed by the Table, not the Game
func NewGame(cfg GameConfig) (*Game, error) {
	if cfg.NumPlayers < 2 {
		return nil, fmt.Errorf("poker: must have at least 2 players")
	}

	if cfg.Log == nil {
		return nil, fmt.Errorf("poker: log is required")
	}

	// Create a new deck with the given seed (or random if not specified)
	var rng *rand.Rand
	if cfg.Seed != 0 {
		rng = rand.New(rand.NewSource(cfg.Seed))
	} else {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	g := &Game{
		players:         make([]*Player, 0, cfg.NumPlayers), // Empty slice, Table will populate
		currentPlayer:   0,
		dealer:          -1, // Start at -1 so first advancement makes it 0
		deck:            NewDeck(rng),
		communityCards:  nil,
		potManager:      NewPotManager(cfg.NumPlayers),
		currentBet:      0,
		round:           0,
		betRound:        0,
		config:          cfg,
		log:             cfg.Log,
		errorSimulation: false,
		phase:           pokerrpc.GamePhase_NEW_HAND_DEALING,
	}

	// Initialize state machine with first state function
	g.sm = statemachine.New(g, stateNewHandDealing, 32)

	return g, nil
}

func (g *Game) Start(ctx context.Context) { g.sm.Start(ctx) }

// NEW_HAND_DEALING: wait for evStartHand to begin setup
func stateNewHandDealing(g *Game, in <-chan any) GameStateFn {
	g.mu.Lock()
	g.phase = pokerrpc.GamePhase_NEW_HAND_DEALING
	g.mu.Unlock()
	for ev := range in {
		// First, allow cross-state requests (handle/ack via FSM)
		if next, handled := handleGameEvent(g, ev); handled {
			if next != nil {
				return next
			}
			continue
		}
		switch ev.(type) {
		case evStartHand:
			return statePreDeal
		case evGotoShowdown:
			return stateShowdown
		default:
			// no-op for unhandled events in this state
		}
	}
	return nil
}

func statePreDeal(g *Game, in <-chan any) GameStateFn {
	g.mu.Lock()
	g.round++
	// Reset hand-specific state inside FSM (except deck reseed which happens
	// before dealing in ResetForNewHandFromUsers to keep a single deck per hand)
	g.communityCards = nil
	g.currentBet = 0
	g.betRound = 0
	g.actionsInRound = 0 // Reset action counter for new hand
	g.winners = nil
	// Deck reseed is done prior to dealing hole cards
	g.phase = pokerrpc.GamePhase_PRE_FLOP

	// Initialize new Hand for this round (cards will be dealt in stateDeal next)
	if g.currentHand == nil {
		playerIDs := make([]string, 0, len(g.players))
		for _, p := range g.players {
			if p != nil {
				playerIDs = append(playerIDs, p.ID())
			}
		}
		g.currentHand = NewHand(playerIDs)
		g.log.Debugf("statePreDeal: initialized new hand %s with %d players", g.currentHand.id, len(playerIDs))
	}

	// Advance dealer position for the new hand
	numPlayers := len(g.players)
	if numPlayers > 0 {
		if g.dealer < 0 {
			g.dealer = 0 // First hand starts at position 0
		} else {
			g.dealer = (g.dealer + 1) % numPlayers
		}

		// Calculate blind positions
		sbPos := (g.dealer + 1) % numPlayers
		bbPos := (g.dealer + 2) % numPlayers
		if numPlayers == 2 {
			// In heads-up, dealer is small blind
			sbPos = g.dealer
			bbPos = (g.dealer + 1) % numPlayers
		}

		// Set position flags SYNCHRONOUSLY under player locks
		// This MUST happen atomically before any snapshot can be taken
		// to avoid race conditions where UI sees dealer but not blind flags
		for i, p := range g.players {
			if p != nil {
				p.mu.Lock()
				p.isDealer = (i == g.dealer)
				p.isSmallBlind = (i == sbPos)
				p.isBigBlind = (i == bbPos)
				p.mu.Unlock()
				g.log.Debugf("DEBUG POSITIONS: Set Player[%d] %s dealer=%v sb=%v bb=%v",
					i, p.id, i == g.dealer, i == sbPos, i == bbPos)
			}
		}

		// POST BLINDS before starting hand participation
		// This allows HandleStartHand to detect all-in from blinds and start in the correct state
		postBlind := func(pos int, amount int64) {
			p := g.players[pos]
			if p == nil {
				return
			}
			// If already has at least this bet, nothing to do
			p.mu.RLock()
			already := p.currentBet
			balance := p.balance
			p.mu.RUnlock()
			if already >= amount {
				return
			}
			// Calculate delta (capped by stack)
			delta := amount - already
			if delta > balance {
				delta = balance
			}

			// Use player method to post blind (players are still seated at table)
			applied := p.HandlePostBlind(delta)

			// Reflect into pot manager and table-wide current bet
			g.potManager.addBet(pos, applied, g.players) // contract: g.mu held
			finalBet := already + applied
			if finalBet > g.currentBet {
				g.currentBet = finalBet
			}
		}

		postBlind(sbPos, g.config.SmallBlind)
		postBlind(bbPos, g.config.BigBlind)

		// AFTER blinds are posted, start hand participation for all players
		// This starts their hand participation FSM in the appropriate state
		for _, p := range g.players {
			if p != nil {
				if err := p.HandleStartHand(); err != nil {
					g.log.Errorf("Failed to start hand for player %s: %v", p.ID(), err)
				}
			}
		}
	}

	g.log.Debugf("statePreDeal: transitioned to PRE_FLOP phase, round=%d, dealer=%d", g.round, g.dealer)
	g.mu.Unlock()
	return stateDeal
}

func stateDeal(g *Game, in <-chan any) GameStateFn {
	// Deal hole cards to all players
	// This is called after statePreDeal has initialized currentHand and set up dealer/blinds
	if err := g.DealHoleCards(); err != nil {
		g.log.Errorf("stateDeal: failed to deal cards: %v", err)
		// Continue anyway - defensive against deck issues
	}
	return stateBlinds
}

func stateBlinds(g *Game, in <-chan any) GameStateFn {
	// Blinds are now posted in statePreDeal before evStartHand is sent
	// This state just sets up the first player to act
	g.mu.Lock()
	defer g.mu.Unlock()

	numPlayers := len(g.players)
	if numPlayers < 2 {
		return stateEnd
	}

	// Calculate blind positions to determine first player
	sbPos := (g.dealer + 1) % numPlayers
	bbPos := (g.dealer + 2) % numPlayers
	if numPlayers == 2 {
		sbPos = g.dealer
		bbPos = (g.dealer + 1) % numPlayers
	}

	if numPlayers == 2 {
		g.currentPlayer = sbPos
	} else {
		g.currentPlayer = (bbPos + 1) % numPlayers
	}
	if g.currentPlayer >= 0 && g.currentPlayer < len(g.players) {
		if p := g.players[g.currentPlayer]; p != nil {
			p.StartTurn()
		}
	}
	// phase stays PRE_FLOP (already set in statePreDeal)
	return statePreFlop
}

func statePreFlop(g *Game, in <-chan any) GameStateFn {
	g.mu.Lock()
	g.phase = pokerrpc.GamePhase_PRE_FLOP
	ch := g.preFlopReached // Read channel reference under lock
	g.mu.Unlock()

	// Signal that we've reached PRE_FLOP (non-blocking)
	if ch != nil {
		select {
		case ch <- struct{}{}:
		default:
		}
	}

	for ev := range in {
		// Let generic requests run inside any state
		if next, handled := handleGameEvent(g, ev); handled {
			if next != nil {
				return next
			}
			continue
		}
		switch ev.(type) {
		case evGotoShowdown:
			return stateShowdown
		case evAdvance:
			g.mu.Lock()
			can := g.betRound == 0
			if can {
				g.betRound++
			}
			g.mu.Unlock()
			if can {
				return stateFlop
			}
		default:
		}
	}
	return nil
}

func stateFlop(g *Game, in <-chan any) GameStateFn {
	g.mu.Lock()
	// Deal flop on entry; guard against double-deal
	g.dealFlop()
	g.currentBet = 0
	g.phase = pokerrpc.GamePhase_FLOP
	g.mu.Unlock()
	// wait events ...
	for ev := range in {
		if next, handled := handleGameEvent(g, ev); handled {
			if next != nil {
				return next
			}
			continue
		}
		switch ev.(type) {
		case evGotoShowdown:
			return stateShowdown
		case evAdvance:
			g.mu.Lock()
			can := g.betRound == 1
			if can {
				g.betRound++
			}
			g.mu.Unlock()
			if can {
				return stateTurn
			}
		default:
		}
	}
	return nil
}

func stateTurn(g *Game, in <-chan any) GameStateFn {
	g.mu.Lock()
	// Deal turn on entry; guard against double-deal
	g.dealTurn()
	g.currentBet = 0
	g.phase = pokerrpc.GamePhase_TURN
	g.mu.Unlock()

	for ev := range in {
		if next, handled := handleGameEvent(g, ev); handled {
			if next != nil {
				return next
			}
			continue
		}
		switch ev.(type) {
		case evGotoShowdown:
			return stateShowdown
		case evAdvance:
			g.mu.Lock()
			can := g.betRound == 2
			if can {
				g.betRound++
			}
			g.mu.Unlock()
			if can {
				return stateRiver
			}
		default:
		}
	}
	return nil
}

func stateRiver(g *Game, in <-chan any) GameStateFn {
	g.mu.Lock()
	// Deal river on entry; guard against double-deal
	g.dealRiver()
	g.currentBet = 0
	g.phase = pokerrpc.GamePhase_RIVER
	g.mu.Unlock()

	for ev := range in {
		if next, handled := handleGameEvent(g, ev); handled {
			if next != nil {
				return next
			}
			continue
		}
		switch ev.(type) {
		case evGotoShowdown:
			return stateShowdown
		case evAdvance:
			if g.betRound == 3 {
				return stateShowdown
			}
		default:
		}
	}
	return nil
}

func stateShowdown(g *Game, in <-chan any) GameStateFn {
	g.log.Debugf("stateShowdown: entered showdown state")
	g.mu.Lock()
	g.phase = pokerrpc.GamePhase_SHOWDOWN
	// Clear current player's turn flags to avoid accepting further actions
	for _, p := range g.players {
		if p != nil {
			p.EndTurn()
		}
	}
	g.currentPlayer = -1

	// Process showdown logic
	g.log.Debugf("stateShowdown: processing showdown")
	result, err := g.handleShowdown()
	if err != nil {
		g.log.Errorf("stateShowdown: showdown failed: %v", err)
	} else {
		g.log.Debugf("stateShowdown: showdown completed successfully with %d winners", len(result.Winners))
		// Store winners for GetWinners() API
		g.winners = result.Winners
		// Send the showdown result to the table
		g.sendTableEvent(GameEvent{Type: GameEventShowdownComplete, ShowdownResult: result})
	}
	g.mu.Unlock()

	// Schedule auto-start if configured
	if g.HasAutoStartCallbacks() {
		g.log.Debugf("stateShowdown: scheduling auto-start")
		g.ScheduleAutoStart()
	}

	// Stay here until a new hand is started
	for ev := range in {
		// Even in SHOWDOWN, handle request events so callers receive replies
		if next, handled := handleGameEvent(g, ev); handled {
			if next != nil {
				return next
			}
			continue
		}
		switch ev.(type) {
		case evStartHand:
			g.log.Debugf("stateShowdown: received evStartHand, transitioning to statePreDeal")
			return statePreDeal
		default:
			g.log.Debugf("stateShowdown: ignoring event %T while waiting for evStartHand", ev)
		}
	}
	return nil
}

func stateEnd(*Game, <-chan any) GameStateFn { return nil }

// GetPot returns the total pot amount
func (g *Game) GetPot() int64 {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.potManager.getTotalPot()
}

// StateFlop deals the flop (3 community cards)
func (g *Game) StateFlop() {
	g.mu.Lock()
	// Guard: only deal flop if we haven't already dealt it
	if len(g.communityCards) < 3 {
		g.dealFlop()
	}
	g.phase = pokerrpc.GamePhase_FLOP
	g.mu.Unlock()
}

// StateTurn deals the turn (1 community card)
func (g *Game) StateTurn() {
	g.mu.Lock()
	if len(g.communityCards) < 4 {
		g.dealTurn()
	}
	g.phase = pokerrpc.GamePhase_TURN
	g.mu.Unlock()
}

// StateRiver deals the river (1 community card)
func (g *Game) StateRiver() {
	g.mu.Lock()
	if len(g.communityCards) < 5 {
		g.dealRiver()
	}
	g.phase = pokerrpc.GamePhase_RIVER
	g.mu.Unlock()
}

// GetPhase returns the current phase of the game.
func (g *Game) GetPhase() pokerrpc.GamePhase {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.phase
}

// GetCurrentBet returns the current bet amount
func (g *Game) GetCurrentBet() int64 {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.currentBet
}

// AddToPotForPlayer adds the specified amount to the pot for a specific
// player.
func (g *Game) AddToPotForPlayer(playerIndex int, amount int64) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.potManager.addBet(playerIndex, amount, g.players)
}

// RefundUncalledBets safely refunds any uncalled portion of the highest bet
// for the current betting round. This guards against scenarios where betting
// closes with one actionable player (e.g., heads-up all-in versus a non-caller)
// and prevents creating an invalid side pot that only the all-in can win.
func (g *Game) refundUncalledBets() error {
	mustHeld(&g.mu)
	if g.potManager == nil {
		return fmt.Errorf("potManager is nil")
	}
	// Compute forced amounts for current street based on blind positions
	forced := make([]int64, len(g.players))
	for i, p := range g.players {
		if p != nil {
			if p.isSmallBlind {
				forced[i] = g.config.SmallBlind
			} else if p.isBigBlind {
				forced[i] = g.config.BigBlind
			}
		}
	}

	hiPlayer, refunded, err := g.potManager.returnUncalledBet(forced)
	if err != nil {
		return fmt.Errorf("failed to refund uncalled bets: %w", err)
	}

	// Rebuild pots after adjusting bets to reflect the refunded amounts
	g.potManager.RebuildPotsIncremental(g.players)

	// Credit the refunded amount to the player
	if hiPlayer >= 0 && refunded > 0 {
		if err := g.players[hiPlayer].credit(refunded); err != nil {
			return fmt.Errorf("failed to credit refunded amount: %w", err)
		}
	}
	return nil
}

// GetCommunityCards returns a copy of the community cards slice.
func (g *Game) GetCommunityCards() []Card {
	g.mu.RLock()
	defer g.mu.RUnlock()
	cards := make([]Card, len(g.communityCards))
	copy(cards, g.communityCards)
	return cards
}

// GetPlayers returns the game players slice
func (g *Game) GetPlayers() []*Player {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.players
}

// GetCurrentPlayer returns the index of the current player to act
func (g *Game) GetCurrentPlayer() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.currentPlayer
}

// GetCurrentPlayerObject returns the current player object
func (g *Game) GetCurrentPlayerObject() *Player {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.currentPlayer >= 0 && g.currentPlayer < len(g.players) {
		return g.players[g.currentPlayer]
	}
	return nil
}

// SetCurrentPlayerByID sets the current player to the seat matching the given
// player ID, if present. If no matching player exists, the call is ignored.
func (g *Game) SetCurrentPlayerByID(playerID string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for i, p := range g.players {
		if p != nil && p.id == playerID {
			// End turn for all other players, start turn for the target player
			for j, cp := range g.players {
				if cp == nil {
					continue
				}
				if j == i {
					if cp.GetCurrentStateString() == "IN_GAME" {
						cp.StartTurn()
					}
				} else {
					cp.EndTurn()
				}
			}
			g.currentPlayer = i
			return
		}
	}
}

// GetWinners returns the winners of the game
func (g *Game) GetWinners() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.winners
}

// GetCurrentHand returns the current hand (with hole cards for all players)
func (g *Game) GetCurrentHand() *Hand {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.currentHand
}

// DealHoleCards deals 2 cards to each player for a new hand.
func (g *Game) DealHoleCards() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.dealHoleCards()
}

// DealHoleCards deals 2 cards to each player for a new hand.
// Called by stateDeal after statePreDeal has initialized currentHand.
func (g *Game) dealHoleCards() error {
	mustHeld(&g.mu)
	if g.deck == nil {
		return fmt.Errorf("deck not initialized")
	}

	if g.currentHand == nil {
		return fmt.Errorf("currentHand not initialized (should be created in statePreDeal)")
	}

	// Deal 2 cards to each player
	for i := 0; i < 2; i++ {
		for _, p := range g.players {
			if p == nil {
				continue
			}
			card, ok := g.deck.Draw()
			if !ok {
				return fmt.Errorf("deck is empty, cannot deal card to player %s", p.ID())
			}
			if err := g.currentHand.DealCardToPlayer(p.ID(), card); err != nil {
				return fmt.Errorf("failed to deal card to player %s: %w", p.ID(), err)
			}
		}
	}

	g.log.Debugf("DealHoleCards: Dealt 2 cards to %d players", len(g.players))

	// Log each player's hole cards for verification
	for _, p := range g.players {
		if p == nil {
			continue
		}
		cards := g.currentHand.GetPlayerCards(p.ID(), p.ID())
		cardStrs := make([]string, len(cards))
		for i, c := range cards {
			cardStrs[i] = c.String()
		}
		g.log.Debugf("CARDS: Player %s hole cards: %v", p.ID(), cardStrs)
	}

	return nil
}

// Close stops all player state machines and cleans up resources.
// This must be called when a game is no longer needed to prevent goroutine leaks.
func (g *Game) Close() {
	// Grab references while holding lock
	g.mu.Lock()
	players := make([]*Player, len(g.players))
	copy(players, g.players)
	sm := g.sm
	timer := g.autoStartTimer
	g.mu.Unlock()

	// Stop all player state machines without holding lock to avoid deadlock
	for _, p := range players {
		if p != nil {
			p.Close()
		}
	}

	// Stop the game state machine without holding lock to avoid deadlock
	// (state machine may need to acquire g.mu during shutdown)
	if sm != nil {
		sm.Stop()
	}

	// Clear references under lock
	g.mu.Lock()
	g.sm = nil
	g.mu.Unlock()

	// Cancel auto-start timer if running (timer.Stop() is safe to call concurrently)
	if timer != nil {
		timer.Stop()
	}
}

// SetPlayers sets the players for this game from table users
func (g *Game) SetPlayers(users []*User) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Convert users to players for game management using proper constructor
	g.players = make([]*Player, len(users))
	for i, user := range users {
		// Create player using constructor to ensure state machine is initialized
		player := NewPlayer(user.ID, user.Name, g.config.StartingChips)

		// Copy table-level state from user
		player.mu.Lock()
		player.tableSeat = user.TableSeat
		player.isReady = user.IsReady
		player.lastAction = time.Now() // Set current time since User doesn't have LastAction
		player.mu.Unlock()

		g.players[i] = player
	}
}

// In game.go

// ResetForNewHandFromUsers rebuilds/reuses players from the given users,
// resets hand state, and kicks the FSM into NEW_HAND_DEALING → PRE_FLOP.
// All mutations happen under g.mu to avoid races with the SM and readers.
func (g *Game) ResetForNewHandFromUsers(users []*User) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Map existing players by id for reuse
	byID := make(map[string]*Player, len(g.players))
	for _, p := range g.players {
		if p != nil {
			byID[p.id] = p
		}
	}

	// Rebuild g.players in users' seat order, reusing objects when possible
	newPlayers := make([]*Player, 0, len(users))
	for _, u := range users {
		if p := byID[u.ID]; p != nil {
			// Reset existing player for new hand while preserving balance
			_ = p.ResetForNewHand(p.balance)
			p.tableSeat = u.TableSeat
			p.isReady = u.IsReady
			newPlayers = append(newPlayers, p)
		} else {
			// New seat joined between hands
			np := NewPlayer(u.ID, u.Name, g.config.StartingChips)
			np.tableSeat = u.TableSeat
			np.isReady = u.IsReady
			newPlayers = append(newPlayers, np)
		}
	}
	g.players = newPlayers

	// Reset pot manager for the new hand BEFORE blinds are posted
	g.potManager = NewPotManager(len(g.players))

	// Prepare a fresh deck for this hand BEFORE dealing hole cards so that
	// community cards come from the same deck. FSM won't reseed again.
	var nextRng *rand.Rand
	if g.config.Seed != 0 {
		derived := g.config.Seed + int64(g.round+1) // lookahead to next hand
		nextRng = rand.New(rand.NewSource(derived))
	} else {
		base := time.Now().UnixNano()
		var mix int64
		if g.deck != nil && g.deck.rng != nil {
			mix = g.deck.rng.Int63()
		}
		nextRng = rand.New(rand.NewSource(base ^ mix ^ int64(g.round+1)))
	}
	g.deck = NewDeck(nextRng)

	// Clear currentHand so statePreDeal creates a fresh one
	g.currentHand = nil

	// Do NOT reset remaining hand-level state here; FSM will do it in statePreDeal.
	// The Table will send evStartHand to trigger FSM: statePreDeal → stateDeal → stateBlinds → statePreFlop.
	return nil
}

// IncrementActionsInRound increments the action counter for the current betting round
func (g *Game) IncrementActionsInRound() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.actionsInRound++
}

// GetActionsInRound returns the current actions count for this betting round
func (g *Game) GetActionsInRound() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.actionsInRound
}

// ResetActionsInRound resets the action counter for a new betting round
func (g *Game) ResetActionsInRound() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.actionsInRound = 0
}

// ResetForNewHand resets the game state for a new hand while preserving the game instance
func (g *Game) ResetForNewHand(activePlayers []*Player) error {
	// Convert players → users shape minimally (ID/Name/Seat/Ready), preserving chip balances via ResetForNewHandFromUsers implementation.
	users := make([]*User, 0, len(activePlayers))
	for _, p := range activePlayers {
		if p == nil {
			continue
		}
		users = append(users, &User{ID: p.id, Name: p.name, TableSeat: p.tableSeat, IsReady: p.isReady})
	}
	return g.ResetForNewHandFromUsers(users)
}

// HandlePlayerFold handles a player folding in the game (external API)
func (g *Game) HandlePlayerFold(playerID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	// Disallow actions outside betting streets
	switch g.phase {
	case pokerrpc.GamePhase_PRE_FLOP, pokerrpc.GamePhase_FLOP, pokerrpc.GamePhase_TURN, pokerrpc.GamePhase_RIVER:
		// allowed
	default:
		return fmt.Errorf("action not allowed during phase: %s", g.phase)
	}
	return g.handlePlayerFold(playerID)
}

func (g *Game) unfoldsPlayers() int {
	n := 0
	for _, p := range g.players {
		if p != nil && p.GetCurrentStateString() != "FOLDED" {
			n++
		}
	}
	return n
}

// handlePlayerFold is the core logic without locking (for internal use)
// Requires: g.mu held
func (g *Game) handlePlayerFold(playerID string) error {
	mustHeld(&g.mu)

	p := g.getPlayerByID(playerID)
	if p == nil {
		return fmt.Errorf("player not found in game")
	}
	if g.currentPlayerID() != playerID {
		return fmt.Errorf("not your turn to act")
	}

	// Send synchronous fold event to ensure player's state is updated before
	// we evaluate betting-round completion (avoids races).
	if p.handParticipation != nil {
		reply := make(chan error, 1)
		p.handParticipation.Send(evFoldReq{Reply: reply})
		if err := <-reply; err != nil {
			return err
		}
	}

	// Count this action in the round
	g.actionsInRound++

	// If only one player remains, send showdown event to table
	if g.unfoldsPlayers() == 1 {
		// End the player's turn before showdown
		p.EndTurn()
		// Notify the game FSM to enter stateShowdown - it will update phase
		if g.sm != nil {
			g.sm.Send(evGotoShowdown{})
		}
		// Notify table that showdown is ready
		g.sendTableEvent(GameEvent{Type: GameEventShowdownReady})
		return nil
	}

	// End the player's turn before advancing
	p.EndTurn()

	// Move turn to next alive player
	g.advanceToNextPlayer(time.Now()) // must skip folded players

	// Check if betting round is complete and notify table
	g.maybeCompleteBettingRound()

	return nil
}

// HandlePlayerCall handles a player calling in the game (external API)
func (g *Game) HandlePlayerCall(playerID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	// Disallow actions outside betting streets
	switch g.phase {
	case pokerrpc.GamePhase_PRE_FLOP, pokerrpc.GamePhase_FLOP, pokerrpc.GamePhase_TURN, pokerrpc.GamePhase_RIVER:
		// allowed
	default:
		return fmt.Errorf("action not allowed during phase: %s", g.phase)
	}
	return g.handlePlayerCall(playerID)
}

// handlePlayerCall is the core logic without locking (for internal use)
// Requires: g.mu held
func (g *Game) handlePlayerCall(playerID string) error {
	mustHeld(&g.mu)
	player := g.getPlayerByID(playerID)
	if player == nil {
		return fmt.Errorf("player not found in game")
	}
	if g.currentPlayerID() != playerID {
		return fmt.Errorf("not your turn to act")
	}

	// Prefer table-wide current bet; fallback to max if zero.
	maxBet := g.currentBet
	if maxBet == 0 {
		for _, p := range g.players {
			if p != nil {
				p.mu.RLock()
				bet := p.currentBet
				p.mu.RUnlock()
				if bet > maxBet {
					maxBet = bet
				}
			}
		}
	}

	player.mu.RLock()
	currentBet := player.currentBet
	balance := player.balance
	player.mu.RUnlock()

	if maxBet <= currentBet {
		return fmt.Errorf("nothing to call - use check instead")
	}

	// Chips to put in: min(toCall, stack)
	toCall := maxBet - currentBet
	delta := toCall
	if delta > balance {
		g.log.Debugf("Player %s cannot afford to call %d (has %d), contributing remainder all-in",
			player.ID(), delta, balance)
		delta = balance
	}
	if delta <= 0 {
		return fmt.Errorf("invalid call amount")
	}

	// Send call action to player FSM and wait for it to process
	reply := make(chan error, 1)
	if player.handParticipation != nil {
		player.handParticipation.Send(evCallDelta{Amt: delta, Reply: reply})
		// Wait for FSM to process the call
		if err := <-reply; err != nil {
			return err
		}
	} else {
		return fmt.Errorf("player FSM not initialized")
	}

	// Pot bookkeeping uses the calculated delta (not dependent on player state).
	for i, p := range g.players {
		if p != nil && p.ID() == playerID {
			g.potManager.addBet(i, delta, g.players)
			break
		}
	}

	// End the player's turn before advancing
	player.EndTurn()

	g.actionsInRound++

	// Check if we should advance to next player or go to showdown
	// Count active (non-folded, non-all-in) players
	activePlayers := 0
	for _, p := range g.players {
		state := p.GetCurrentStateString()
		if state != "FOLDED" && state != "ALL_IN" {
			activePlayers++
		}
	}

	// Advance to next player if there are any players who can act
	// (including the case where only one player remains who can act,
	// as they still need a turn to respond to the all-in)
	if activePlayers > 0 {
		g.advanceToNextPlayer(time.Now())
	}

	// Check if betting round is complete and notify table
	g.maybeCompleteBettingRound()

	return nil
}

// HandlePlayerCheck handles a player checking in the game (external API)
func (g *Game) HandlePlayerCheck(playerID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	// Disallow actions outside betting streets
	switch g.phase {
	case pokerrpc.GamePhase_PRE_FLOP, pokerrpc.GamePhase_FLOP, pokerrpc.GamePhase_TURN, pokerrpc.GamePhase_RIVER:
		// allowed
	default:
		return fmt.Errorf("action not allowed during phase: %s", g.phase)
	}
	return g.handlePlayerCheck(playerID)
}

// handlePlayerCheck is the core logic without locking (for internal use)
// Requires: g.mu held
func (g *Game) handlePlayerCheck(playerID string) error {
	mustHeld(&g.mu)
	player := g.getPlayerByID(playerID)
	if player == nil {
		return fmt.Errorf("player not found in game")
	}

	if g.currentPlayerID() != playerID {
		return fmt.Errorf("not your turn to act")
	}

	if player.currentBet < g.currentBet {
		return fmt.Errorf("cannot check when there's a bet to call (player bet: %d, current bet: %d)",
			player.currentBet, g.currentBet)
	}

	player.mu.Lock()
	player.lastAction = time.Now()
	player.mu.Unlock()

	// End the player's turn before advancing
	player.EndTurn()

	g.actionsInRound++
	g.advanceToNextPlayer(time.Now())

	// Check if betting round is complete and notify table
	g.maybeCompleteBettingRound()

	return nil
}

// HandlePlayerBet handles a player betting in the game (external API)
func (g *Game) HandlePlayerBet(playerID string, amount int64) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	// Disallow actions outside betting streets
	switch g.phase {
	case pokerrpc.GamePhase_PRE_FLOP, pokerrpc.GamePhase_FLOP, pokerrpc.GamePhase_TURN, pokerrpc.GamePhase_RIVER:
		// allowed
	default:
		return fmt.Errorf("action not allowed during phase: %s", g.phase)
	}
	return g.handlePlayerBet(playerID, amount)
}

// handlePlayerBet is the core logic without locking (for internal use)
// Requires: g.mu held
func (g *Game) handlePlayerBet(playerID string, amount int64) error {
	mustHeld(&g.mu)
	player := g.getPlayerByID(playerID)
	if player == nil {
		return fmt.Errorf("player not found in game")
	}

	if g.currentPlayerID() != playerID {
		return fmt.Errorf("not your turn to act")
	}

	player.mu.RLock()
	currentBet := player.currentBet
	balance := player.balance
	player.mu.RUnlock()

	if amount < currentBet {
		return fmt.Errorf("cannot decrease bet")
	}

	// Determine the effective amount considering player stack (all-in cap)
	delta := amount - currentBet
	if delta > 0 && delta > balance {
		// Player cannot afford the requested raise/bet — cap to all-in
		g.log.Debugf("Player %s cannot afford to bet %d (has %d), contributing remainder all-in", player.ID(), delta, balance)
		delta = balance
		amount = currentBet + delta
	}

	// Compute table-wide current bet (fallback to max player bet if needed)
	tableBet := g.currentBet
	if tableBet == 0 {
		for _, p := range g.players {
			if p != nil {
				p.mu.RLock()
				pBet := p.currentBet
				p.mu.RUnlock()
				if pBet > tableBet {
					tableBet = pBet
				}
			}
		}
	}

	// Server-side validation: disallow betting below the current bet when facing action,
	// except when the player is going all-in for less (short stack call).
	if tableBet > 0 && amount < tableBet {
		// Allow only if this action is an all-in to an amount below the call
		if !(delta > 0 && delta == balance) {
			return fmt.Errorf("bet must be at least current bet (%d)", tableBet)
		}
	}

	// When there is no live bet (opening the betting), enforce minimum opening bet
	// of at least the big blind, unless the player is going all-in for less.
	if tableBet == 0 {
		minOpen := g.config.BigBlind
		if minOpen < 0 {
			minOpen = 0
		}
		if amount < minOpen {
			// Allow short-stack all-in that is less than min open
			if !(delta > 0 && delta == balance) {
				return fmt.Errorf("minimum bet is the big blind (%d)", minOpen)
			}
		}
	}

	// Send bet event to FSM - it will update balance/currentBet and transition to ALL_IN if needed.
	if delta > 0 && player.handParticipation != nil {
		reply := make(chan error, 1)
		player.handParticipation.Send(evBet{Amt: delta, Reply: reply})
		// Wait for FSM to process the bet
		if err := <-reply; err != nil {
			return err
		}
	}

	// Update game-wide current bet
	if amount > g.currentBet {
		g.currentBet = amount
	}

	// Pot bookkeeping uses the calculated delta.
	if delta > 0 {
		for i, p := range g.players {
			if p != nil && p.ID() == playerID {
				g.potManager.addBet(i, delta, g.players)
				break
			}
		}
	}

	// End the player's turn before advancing
	player.EndTurn()

	g.actionsInRound++
	g.advanceToNextPlayer(time.Now())

	// Check if betting round is complete and notify table
	g.maybeCompleteBettingRound()

	return nil
}

// getPlayerByID finds a player by ID
func (g *Game) getPlayerByID(playerID string) *Player {
	mustHeld(&g.mu)
	for _, p := range g.players {
		if p.ID() == playerID {
			return p
		}
	}
	return nil
}

// currentPlayerID returns the current player's ID
func (g *Game) currentPlayerID() string {
	mustHeld(&g.mu)
	if g.currentPlayer < 0 || g.currentPlayer >= len(g.players) {
		return ""
	}
	return g.players[g.currentPlayer].ID()
}

// advanceToNextPlayer moves to the next active player
// Note: The caller (action handlers like bet, call, check, fold) is responsible
// for ending the current player's turn before calling this function.
func (g *Game) advanceToNextPlayer(now time.Time) {
	mustHeld(&g.mu)
	if len(g.players) == 0 {
		return
	}

	playersChecked := 0
	maxPlayers := len(g.players)

	for {
		g.currentPlayer = (g.currentPlayer + 1) % len(g.players)
		playersChecked++

		if playersChecked >= maxPlayers {
			break
		}

		// Only start turn for players that are actively in the hand.
		// This avoids blocking StartTurn on players whose FSM is in AT_TABLE.
		if g.players[g.currentPlayer].GetCurrentStateString() == "IN_GAME" {
			g.players[g.currentPlayer].StartTurn()
			break
		}
		// Skip FOLDED, ALL_IN, AT_TABLE, LEFT, etc.
	}
}

// ShowdownResult contains the results of a showdown for table notifications
type ShowdownResult struct {
	Winners    []string
	WinnerInfo []*pokerrpc.Winner
	TotalPot   int64
}

// HandleShowdown processes the showdown logic and returns results (external API)
func (g *Game) HandleShowdown() (*ShowdownResult, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	// Public API now delegates to the internal locked implementation.
	return g.handleShowdown()
}

// handleShowdown is the core logic without locking (for internal use)
// handleShowdown processes the end of a hand and returns the showdown result.
// Invariants enforced:
//  1. postRefundPot := potManager.getTotalPot()  (after refund+rebuild)
//  2. sum(winner deltas) == postRefundPot (or we warn and self-heal)
//  3. result.TotalPot == postRefundPot
func (g *Game) handleShowdown() (*ShowdownResult, error) {
	mustHeld(&g.mu)
	g.log.Debugf("handleShowdown: entered")

	// DEBUG: Log community cards at showdown
	// Log community cards at showdown
	communityStrs := make([]string, len(g.communityCards))
	for i, c := range g.communityCards {
		communityStrs[i] = c.String()
	}
	g.log.Debugf("CARDS: Showdown community cards: %v", communityStrs)

	// Collect non-folded players
	unfolded := make([]*Player, 0, len(g.players))
	for _, p := range g.players {
		if p != nil && p.GetCurrentStateString() != "FOLDED" {
			unfolded = append(unfolded, p)
		}
	}

	// DEBUG: Log all players' hole cards at showdown
	// Log all players' hole cards at showdown
	for _, p := range g.players {
		if p == nil {
			continue
		}
		hole := g.currentHand.GetPlayerCards(p.id, p.id)
		holeStrs := make([]string, len(hole))
		for i, c := range hole {
			holeStrs[i] = c.String()
		}
		g.log.Debugf("CARDS: Showdown player %s hole cards: %v (state: %s)",
			p.id, holeStrs, p.GetCurrentStateString())
	}

	if len(unfolded) == 0 {
		return nil, fmt.Errorf("invalid showdown: no active players")
	}

	result := &ShowdownResult{
		Winners:    make([]string, 0, len(g.players)),
		WinnerInfo: make([]*pokerrpc.Winner, 0, len(g.players)),
		TotalPot:   0,
	}

	// --- Normalize pots for both branches (refund uncalled bets, rebuild pots)
	// This guarantees post-refund pot accounting regardless of whether it's fold-win or multi-way.
	g.refundUncalledBets()
	g.potManager.RebuildPotsIncremental(g.players)
	postRefundPot := g.potManager.getTotalPot()

	// Helper: snapshot balances
	prev := make(map[string]int64, len(g.players))
	for _, p := range g.players {
		if p != nil {
			p.mu.RLock()
			prev[p.id] = p.balance
			p.mu.RUnlock()
		}
	}

	// --- Branch A: uncontested (exactly one non-folded player)
	if len(unfolded) == 1 {

		// Distribute pots (no actual comparison needed)
		if err := g.potManager.distributePots(g.players); err != nil {
			g.log.Errorf("distributePots (uncontested) failed: %v", err)
			return nil, err
		}

		// Compute deltas, populate winners
		totalWinnings := int64(0)
		for _, p := range g.players {
			if p == nil {
				continue
			}
			p.mu.RLock()
			delta := p.balance - prev[p.id]
			p.mu.RUnlock()
			if delta > 0 {
				// Get player's hole cards from Hand
				hole := g.currentHand.GetPlayerCards(p.id, p.id)

				// Best hand: if board+hole < 5 just report hole cards
				var best []Card
				if len(hole)+len(g.communityCards) >= 5 {
					hv, err := EvaluateHand(hole, g.communityCards)
					if err != nil {
						return nil, fmt.Errorf("evaluate hand (uncontested) for %s: %w", p.id, err)
					}
					p.mu.Lock()
					p.handValue = &hv
					p.handDescription = GetHandDescription(hv)
					p.mu.Unlock()
					best = hv.BestHand
				} else {
					best = hole
				}

				result.Winners = append(result.Winners, p.id)
				result.WinnerInfo = append(result.WinnerInfo, &pokerrpc.Winner{
					PlayerId: p.id,
					BestHand: CreateHandFromCards(best),
					Winnings: delta,
				})
				totalWinnings += delta
			}
		}

		// Invariants & finalization
		if totalWinnings != postRefundPot {
			g.log.Warnf("showdown invariant (uncontested) violated: postRefundPot=%d, distributed=%d",
				postRefundPot, totalWinnings)
			// Self-heal to what we actually paid
			postRefundPot = totalWinnings
		}
		result.TotalPot = postRefundPot
		return result, nil
	}

	// --- Branch B: multi-way showdown (>=2 non-folded players)

	// Ensure board is fully dealt (fast-forward safely based on current phase)
	if len(g.communityCards) < 5 {
		dealOne := func() (Card, bool) { return g.deck.Draw() }
		switch g.phase {
		case pokerrpc.GamePhase_PRE_FLOP:
			for i := 0; i < 3; i++ {
				if c, ok := dealOne(); ok {
					g.communityCards = append(g.communityCards, c)
				} else {
					return nil, fmt.Errorf("deck underflow on flop")
				}
			}
			g.phase = pokerrpc.GamePhase_FLOP
			fallthrough
		case pokerrpc.GamePhase_FLOP:
			if len(g.communityCards) < 4 {
				if c, ok := dealOne(); ok {
					g.communityCards = append(g.communityCards, c)
				} else {
					return nil, fmt.Errorf("deck underflow on turn")
				}
			}
			g.phase = pokerrpc.GamePhase_TURN
			fallthrough
		case pokerrpc.GamePhase_TURN:
			if len(g.communityCards) < 5 {
				if c, ok := dealOne(); ok {
					g.communityCards = append(g.communityCards, c)
				} else {
					return nil, fmt.Errorf("deck underflow on river")
				}
			}
			g.phase = pokerrpc.GamePhase_RIVER
		case pokerrpc.GamePhase_RIVER:
			// nothing
		default:
			return nil, fmt.Errorf("invalid showdown: unexpected phase %s", g.phase)
		}
	}

	// Validate players have enough cards for evaluation
	for _, p := range unfolded {
		hole := g.currentHand.GetPlayerCards(p.id, p.id)
		if len(hole)+len(g.communityCards) < 5 {
			return nil, fmt.Errorf("invalid showdown: player %s has insufficient cards (hole=%d, board=%d)",
				p.id, len(hole), len(g.communityCards))
		}
	}

	// Evaluate hands for all unfolded players
	for _, p := range unfolded {
		hole := g.currentHand.GetPlayerCards(p.id, p.id)
		hv, err := EvaluateHand(hole, g.communityCards)
		if err != nil {
			return nil, fmt.Errorf("evaluate hand (multi-way) for %s: %w", p.id, err)
		}
		p.mu.Lock()
		p.handValue = &hv
		p.handDescription = GetHandDescription(hv)
		p.mu.Unlock()

		// Log evaluation results for debugging
		bestStrs := make([]string, len(hv.BestHand))
		for i, c := range hv.BestHand {
			bestStrs[i] = c.String()
		}
		g.log.Debugf("CARDS: Player %s evaluated: rank=%s rankValue=%d bestHand=%v description=%s",
			p.id, hv.Rank, hv.RankValue, bestStrs, hv.HandDescription)
	}

	// Distribute pots according to eligibility
	if err := g.potManager.distributePots(g.players); err != nil {
		g.log.Errorf("distributePots (multi-way) failed: %v", err)
		return nil, err
	}

	g.log.Debugf("CARDS: Pot distribution complete, checking winners...")

	// Build result from positive deltas
	totalWinnings := int64(0)
	for _, p := range g.players {
		if p == nil {
			continue
		}
		p.mu.RLock()
		delta := p.balance - prev[p.id]
		hv := p.handValue
		p.mu.RUnlock()

		if delta > 0 {
			var handRank pokerrpc.HandRank
			var best []Card
			var rankValue int
			var description string
			if hv != nil {
				handRank = hv.HandRank
				best = hv.BestHand
				rankValue = hv.RankValue
				description = hv.HandDescription
			} else {
				// Get player's hole cards
				hole := g.currentHand.GetPlayerCards(p.id, p.id)
				best = hole
			}

			g.log.Debugf("CARDS: Winner %s with delta=%d rankValue=%d description=%s",
				p.id, delta, rankValue, description)

			result.Winners = append(result.Winners, p.id)
			result.WinnerInfo = append(result.WinnerInfo, &pokerrpc.Winner{
				PlayerId: p.id,
				HandRank: handRank,
				BestHand: CreateHandFromCards(best),
				Winnings: delta,
			})
			totalWinnings += delta
		}
	}

	// Invariants & finalization
	if totalWinnings != postRefundPot {
		g.log.Warnf("showdown invariant (multi-way) violated: postRefundPot=%d, distributed=%d",
			postRefundPot, totalWinnings)
		// Self-heal to what we actually paid
		postRefundPot = totalWinnings
	}
	result.TotalPot = postRefundPot
	return result, nil
}

// maybeCompleteBettingRound checks if the betting round is complete and advances to next phase.
// Goal: advance street only when the acting turn finished AND all actionable players are matched.
// - Do NOT touch per-player currentBet (used for pot building).
// - Reset ONLY table-wide currentBet when a new street starts.
// - do not auto-showdown when active==1 (the other player must decide).
func (g *Game) maybeCompleteBettingRound() {
	mustHeld(&g.mu)

	// 0) Phases we never auto-advance from here.
	if g.phase == pokerrpc.GamePhase_NEW_HAND_DEALING {
		return
	}

	// 1) Tally round status (REMOVED turn check - we hold g.mu so no TOCTOU possible)
	alive, active, unmatched := 0, 0, 0
	for _, p := range g.players {
		if p == nil {
			continue
		}
		state := p.GetCurrentStateString()

		// Skip folded entirely.
		if state == "FOLDED" {
			continue
		}

		// Player is alive.
		alive++

		// ALL_IN counts as matched but is not actionable.
		if state == "ALL_IN" {
			continue
		}

		// Actionable: must match table currentBet.
		active++
		p.mu.RLock()
		cb := p.currentBet
		p.mu.RUnlock()
		if cb != g.currentBet {
			unmatched++
		}
	}

	// 2) Terminal: only one alive -> uncontested; go to showdown.
	if alive <= 1 {
		g.phase = pokerrpc.GamePhase_SHOWDOWN
		g.sm.Send(evGotoShowdown{})
		return
	}

	// 3) All alive are ALL_IN -> deal out remaining streets then showdown.
	if active == 0 {
		// Fast-forward community cards without zeroing player bets.
		switch g.phase {
		case pokerrpc.GamePhase_PRE_FLOP:
			g.dealFlop()
			g.currentBet = 0
			g.phase = pokerrpc.GamePhase_FLOP
			g.dealTurn()
			g.currentBet = 0
			g.phase = pokerrpc.GamePhase_TURN
			g.dealRiver()
			g.currentBet = 0
			g.phase = pokerrpc.GamePhase_RIVER
		case pokerrpc.GamePhase_FLOP:
			g.dealTurn()
			g.currentBet = 0
			g.phase = pokerrpc.GamePhase_TURN
			g.dealRiver()
			g.currentBet = 0
			g.phase = pokerrpc.GamePhase_RIVER
		case pokerrpc.GamePhase_TURN:
			g.dealRiver()
			g.currentBet = 0
			g.phase = pokerrpc.GamePhase_RIVER
		}
		g.phase = pokerrpc.GamePhase_SHOWDOWN
		g.sm.Send(evGotoShowdown{})
		return
	}

	// 4) Heads-up special case: if exactly one actionable remains, the other player must still act.
	// Do NOT auto-advance or showdown just because active==1 in HU.
	if len(g.players) == 2 && active == 1 {
		return
	}

	// 5) Need everybody to have acted once this street.
	if g.actionsInRound < active {
		return
	}
	// 6) Need all actionable bets matched.
	if unmatched > 0 {
		return
	}

	// 7) Street is complete -> advance exactly one street (or showdown on river).
	switch g.phase {
	case pokerrpc.GamePhase_PRE_FLOP:
		g.dealFlop()
		g.currentBet = 0
		g.phase = pokerrpc.GamePhase_FLOP
		g.sm.Send(evAdvance{})
	case pokerrpc.GamePhase_FLOP:
		g.dealTurn()
		g.currentBet = 0
		g.phase = pokerrpc.GamePhase_TURN
		g.sm.Send(evAdvance{})
	case pokerrpc.GamePhase_TURN:
		g.dealRiver()
		g.currentBet = 0
		g.phase = pokerrpc.GamePhase_RIVER
		g.sm.Send(evAdvance{})
	case pokerrpc.GamePhase_RIVER:
		g.phase = pokerrpc.GamePhase_SHOWDOWN
		g.sm.Send(evGotoShowdown{})
	}

	// 8) Prepare next betting round.
	// Clear turn flags, zero player currentBet (since table currentBet was zeroed above),
	// and re-init who acts next. Pot manager retains cumulative contributions.
	for _, p := range g.players {
		if p != nil {
			p.EndTurn()
			p.mu.Lock()
			p.currentBet = 0 // Reset for new street (pot manager tracks cumulative)
			p.mu.Unlock()
		}
	}
	g.actionsInRound = 0
	g.initializeCurrentPlayer()
}

// AdvanceToNextPlayer moves to the next active player (external API)
func (g *Game) AdvanceToNextPlayer(now time.Time) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.advanceToNextPlayer(now)
}

// InitializeCurrentPlayer sets the current player with proper locking (external API)
func (g *Game) InitializeCurrentPlayer() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.initializeCurrentPlayer()
}

// initializeCurrentPlayer sets the current player based on game phase and rules
func (g *Game) initializeCurrentPlayer() {
	mustHeld(&g.mu)
	if len(g.players) == 0 {
		g.currentPlayer = -1
		return
	}

	numPlayers := len(g.players)

	// In pre-flop, start with Under the Gun (player after big blind)
	if g.phase == pokerrpc.GamePhase_PRE_FLOP {
		if numPlayers == 2 {
			// In heads-up, after blinds are posted, small blind acts first
			// The small blind IS the dealer in heads-up
			g.currentPlayer = g.dealer
		} else {
			// In multi-way, Under the Gun acts first (after big blind)
			g.currentPlayer = (g.dealer + 3) % numPlayers
		}
	} else {
		// Post-flop action order:
		// - Heads-up: Big blind acts first (dealer is SB, so dealer+1 = BB acts first)
		// - Multi-way: Small blind acts first (dealer+1 = SB acts first)
		//
		// In both cases, the formula is the same: (dealer + 1) % numPlayers
		g.currentPlayer = (g.dealer + 1) % numPlayers
	}

	// Ensure we start with an active player and handle edge cases
	playersChecked := 0
	maxPlayers := len(g.players)

	for {
		// Validate currentPlayer is within bounds
		if g.currentPlayer < 0 || g.currentPlayer >= len(g.players) {
			g.currentPlayer = 0 // Reset to first player if out of bounds
		}

		// Use the unified player state directly
		if g.players[g.currentPlayer].GetCurrentStateString() != "FOLDED" {
			break
		}

		g.currentPlayer = (g.currentPlayer + 1) % len(g.players)
		playersChecked++

		// Prevent infinite loop by checking all players at most once
		if playersChecked >= maxPlayers {
			// All players have folded - this shouldn't happen during initialization
			// Default to first player
			g.currentPlayer = 0
			break
		}
	}

	// Start turn for the new current player only if actively in the hand
	if g.currentPlayer >= 0 && g.currentPlayer < len(g.players) {
		if g.players[g.currentPlayer].GetCurrentStateString() == "IN_GAME" {
			g.players[g.currentPlayer].StartTurn()
		}
	}
}

// GetRound returns the current round number
func (g *Game) GetRound() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.round
}

// GetBetRound returns the current betting round
func (g *Game) GetBetRound() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.betRound
}

// GetDealer returns the dealer position
func (g *Game) GetDealer() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.dealer
}

// GetDeckState returns the current deck state for persistence
func (g *Game) GetDeckState() interface{} {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.deck == nil {
		return nil
	}
	// Return the remaining cards in the deck
	return g.deck.cards
}

// SetGameState allows restoring game state from persistence
func (g *Game) SetGameState(dealer, round, betRound int, currentBet, pot int64, phase pokerrpc.GamePhase) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.dealer = dealer
	g.round = round
	g.betRound = betRound
	g.currentBet = currentBet
	g.phase = phase
	// Note: Pot will be restored through the potManager when restoring player bets
	// We can't directly set the pot value, but it will be calculated from player bets

	g.initializeCurrentPlayer()
}

// RestoreGameState rebuilds all derived state from persisted player data
// This should be called after SetGameState and after all players have been restored
func (g *Game) RestoreGameState(tableID string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// 1) Rebuild pot manager from players
	g.potManager = NewPotManager(len(g.players)) // fresh for safety

	maxBet := int64(0)
	for i, p := range g.players {
		if p == nil {
			continue
		}
		if p.currentBet > 0 {
			g.potManager.addBet(i, p.currentBet, g.players) // contract: g.mu held
		}
		if p.currentBet > maxBet {
			maxBet = p.currentBet
		}
	}
	g.currentBet = maxBet

	// 2) Ensure phase is coherent with what's on the table
	// If blinds are posted or any bet exists, we're not NEW_HAND_DEALING
	if g.currentBet > 0 && g.phase == pokerrpc.GamePhase_NEW_HAND_DEALING {
		g.phase = pokerrpc.GamePhase_PRE_FLOP
	}

	// 3) (re)choose a valid current player from rules, not from snapshot
	g.initializeCurrentPlayer() // uses phase/dealer and skips folded/all-in/disconnected

	// 4) If nobody is actionable (e.g., only one alive), push to showdown
	alive := 0
	for _, p := range g.players {
		if p != nil && p.GetCurrentStateString() != "FOLDED" {
			alive++
		}
	}
	if alive <= 1 {
		g.phase = pokerrpc.GamePhase_SHOWDOWN
	}
}

// SetCommunityCards allows restoring community cards from persistence
func (g *Game) SetCommunityCards(cards []Card) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.communityCards = make([]Card, len(cards))
	copy(g.communityCards, cards)
}

// SetAutoStartCallbacks sets the callback functions for auto-start functionality
func (g *Game) SetAutoStartCallbacks(callbacks *AutoStartCallbacks) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.autoStartCallbacks = callbacks
}

// HasAutoStartCallbacks reports whether auto-start callbacks are configured.
func (g *Game) HasAutoStartCallbacks() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.autoStartCallbacks != nil
}

// SetupPreFlopNotification creates a notification channel that will be signaled
// when the PRE_FLOP state is reached. Returns the channel. The channel has a
// buffer of 1 to prevent blocking if no one is listening.
func (g *Game) SetupPreFlopNotification() <-chan struct{} {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.preFlopReached = make(chan struct{}, 1)
	return g.preFlopReached
}

// ClearPreFlopNotification removes the notification channel.
func (g *Game) ClearPreFlopNotification() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.preFlopReached != nil {
		close(g.preFlopReached)
		g.preFlopReached = nil
	}
}

// scheduleAutoStart schedules automatic start of next hand after configured delay
func (g *Game) ScheduleAutoStart() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.scheduleAutoStart()
}

// scheduleAutoStart is the internal implementation
func (g *Game) scheduleAutoStart() {
	mustHeld(&g.mu)
	// Cancel any existing auto-start timer
	g.cancelAutoStart()

	// Check if auto-start is configured (negative delay disables auto-start)
	if g.config.AutoStartDelay < 0 || g.autoStartCallbacks == nil {
		g.log.Debugf("scheduleAutoStart: auto-start disabled, delay=%v, callbacks=%v", g.config.AutoStartDelay, g.autoStartCallbacks != nil)
		return
	}

	// Use a minimal delay if configured delay is 0 to allow notifications to propagate
	effectiveDelay := g.config.AutoStartDelay
	if effectiveDelay == 0 {
		effectiveDelay = 100 * time.Millisecond
	}

	// Debug log
	g.log.Debugf("scheduleAutoStart: setting up timer with delay %v (configured=%v)", effectiveDelay, g.config.AutoStartDelay)

	// Mark that auto-start is pending
	g.autoStartCanceled = false

	// Schedule the auto-start with self-rescheduling if conditions aren't met
	var scheduleCheckLocked func() // requires g.mu held
	var scheduleCheck func()       // acquires g.mu internally

	scheduleCheck = func() {
		g.mu.Lock()
		defer g.mu.Unlock()
		scheduleCheckLocked()
	}

	scheduleCheckLocked = func() {
		// Avoid scheduling if canceled or callbacks missing (g.mu held)
		if g.autoStartCanceled || g.autoStartCallbacks == nil {
			return
		}
		t := time.AfterFunc(effectiveDelay, func() {
			// Snapshot required fields under lock to avoid races
			g.mu.RLock()
			canceled := g.autoStartCanceled
			callbacks := g.autoStartCallbacks
			log := g.log
			cfg := g.config
			players := make([]*Player, len(g.players))
			copy(players, g.players)
			g.mu.RUnlock()

			if canceled || callbacks == nil {
				return
			}

			readyCount := 0
			for _, player := range players {
				if player == nil {
					continue
				}
				// Read player fields under the player's lock to avoid races
				player.mu.RLock()
				bal := player.balance
				pid := player.id
				player.mu.RUnlock()

				// Count players who have any chips left. Short stacks will auto-post
				// blinds all-in when needed during hand setup.
				if bal > 0 {
					readyCount++
					// Log explicitly that short stacks are still eligible for auto-start.
					if bal < cfg.BigBlind {
						log.Debugf("Player %s ready for auto-start (short stack all-in): balance=%d < bigBlind=%d", pid, bal, cfg.BigBlind)
					} else {
						log.Debugf("Player %s ready for auto-start: balance=%d >= bigBlind=%d", pid, bal, cfg.BigBlind)
					}
				} else {
					log.Debugf("Player %s not ready for auto-start: balance=0", pid)
				}
			}

			minRequired := callbacks.MinPlayers()
			log.Debugf("Auto-start check: readyCount=%d, minRequired=%d", readyCount, minRequired)
			if readyCount >= minRequired {
				err := callbacks.StartNewHand()
				if err != nil {
					log.Debugf("Auto-start new hand failed: %v", err)
					// Reschedule on failure
					scheduleCheck()
				} else {
					if callbacks.OnNewHandStarted != nil {
						// Invoke the callback
						go callbacks.OnNewHandStarted()
					}
				}
			} else {
				// Not enough players yet - reschedule to check again
				log.Debugf("Not enough players for auto-start: %d < %d, will check again", readyCount, minRequired)
				scheduleCheck()
			}
		})
		// Assign the timer while holding the lock to serialize writes (g.mu held)
		g.autoStartTimer = t
	}
	// We are currently called with g.mu held by the public wrapper, so call the locked variant.
	scheduleCheckLocked()
}

// CancelAutoStart cancels any pending auto-start timer
func (g *Game) CancelAutoStart() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.cancelAutoStart()
}

// cancelAutoStart is the internal implementation (assumes lock is held)
func (g *Game) cancelAutoStart() {
	mustHeld(&g.mu)
	if g.autoStartTimer != nil {
		g.autoStartTimer.Stop()
		g.autoStartTimer = nil
	}
	g.autoStartCanceled = true
}

// PlayerSnapshot represents a snapshot of player state without mutex for safe concurrent access
type PlayerSnapshot struct {
	ID              string
	Name            string
	TableSeat       int
	IsReady         bool
	IsDisconnected  bool
	LastAction      time.Time
	Balance         int64
	StartingBalance int64
	Hand            []Card
	CurrentBet      int64
	IsDealer        bool
	IsSmallBlind    bool
	IsBigBlind      bool
	IsTurn          bool
	StateString     string
	HandValue       *HandValue
	HandDescription string
}

// GameStateSnapshot represents a point-in-time snapshot of game state for safe concurrent access
type GameStateSnapshot struct {
	Dealer         int
	CurrentBet     int64
	Pot            int64
	Round          int
	BetRound       int
	Phase          pokerrpc.GamePhase
	CommunityCards []Card
	DeckState      interface{}
	Players        []PlayerSnapshot
	CurrentPlayer  string
	Winners        []PlayerSnapshot
}

// GetStateSnapshot returns an atomic snapshot of the game state for safe concurrent access
func (g *Game) GetStateSnapshot() GameStateSnapshot {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Create a deep copy of players to avoid race conditions
	playersCopy := make([]PlayerSnapshot, len(g.players))
	for i, player := range g.players {
		if player == nil {
			// Create a zero-value PlayerSnapshot for nil entries
			playersCopy[i] = PlayerSnapshot{}
			continue
		}
		// Lock the player's own mutex while copying fields that are mutated by the FSM
		player.mu.RLock()
		playerCopy := PlayerSnapshot{
			ID:              player.id,
			Name:            player.name,
			TableSeat:       player.tableSeat,
			IsReady:         player.isReady,
			Balance:         player.balance,
			StartingBalance: player.startingBalance,
			CurrentBet:      player.currentBet,
			IsDealer:        player.isDealer,
			// Preserve blind flags in snapshot so downstream snapshots and UIs
			// see correct SB/BB assignments (e.g., dealer==SB in heads-up).
			IsSmallBlind:    player.isSmallBlind,
			IsBigBlind:      player.isBigBlind,
			IsTurn:          player.isTurn,
			IsDisconnected:  player.isDisconnected,
			Hand:            nil, // Will be populated from currentHand below
			HandDescription: player.handDescription,
			HandValue:       player.handValue,
			LastAction:      player.lastAction,
			StateString:     player.getCurrentStateString(), // Use locked version to avoid reentrant lock
		}
		player.mu.RUnlock()

		// Retrieve hole cards from g.currentHand if it exists
		// Note: Snapshot includes all cards; visibility filtering happens at server level
		if g.currentHand != nil {
			// Get player's cards from the current hand
			// Use player's own ID as requestor to get their cards
			cards := g.currentHand.GetPlayerCards(player.id, player.id)
			if len(cards) > 0 {
				// Deep copy the cards to avoid reference issues
				playerCopy.Hand = make([]Card, len(cards))
				copy(playerCopy.Hand, cards)
			}
		}

		playersCopy[i] = playerCopy
	}

	// Copy community cards
	communityCardsCopy := make([]Card, len(g.communityCards))
	copy(communityCardsCopy, g.communityCards)

	// Resolve current player ID from live index if valid
	curID := ""
	if g.currentPlayer >= 0 && g.currentPlayer < len(g.players) && g.players[g.currentPlayer] != nil {
		curID = g.players[g.currentPlayer].ID()
	}

	// Calculate pot amount based on game phase
	var potAmount int64
	// During showdown, use getTotalPot() after pots have been built
	potAmount = g.potManager.getTotalPot()

	winners := make([]PlayerSnapshot, 0)
	// Get winners if available
	if len(g.winners) > 0 {
		for _, winnerID := range g.winners {
			// Find player by ID
			for _, player := range g.players {
				if player != nil && player.id == winnerID {
					// Create a PlayerSnapshot for the winner
					player.mu.RLock()
					winnerSnapshot := PlayerSnapshot{
						ID:              player.id,
						Name:            player.name,
						TableSeat:       player.tableSeat,
						IsReady:         player.isReady,
						Balance:         player.balance,
						StartingBalance: player.startingBalance,
						CurrentBet:      player.currentBet,
						IsDealer:        player.isDealer,
						IsSmallBlind:    player.isSmallBlind,
						IsBigBlind:      player.isBigBlind,
						IsTurn:          player.isTurn,
						IsDisconnected:  player.isDisconnected,
						Hand:            nil, // Will be populated from currentHand below
						HandDescription: player.handDescription,
						HandValue:       player.handValue,
						LastAction:      player.lastAction,
					}
					player.mu.RUnlock()

					// Retrieve revealed winner cards from g.currentHand
					if g.currentHand != nil {
						// Winners' cards are always visible at showdown
						// Use the winner's own ID to get their revealed cards
						cards := g.currentHand.GetPlayerCards(player.id, player.id)
						if len(cards) > 0 {
							winnerSnapshot.Hand = make([]Card, len(cards))
							copy(winnerSnapshot.Hand, cards)
						}
					}

					winners = append(winners, winnerSnapshot)
					break
				}
			}
		}
	}

	return GameStateSnapshot{
		Dealer:         g.dealer,
		CurrentBet:     g.currentBet,
		Pot:            potAmount,
		Round:          g.round,
		BetRound:       g.betRound,
		Phase:          g.phase,
		CommunityCards: communityCardsCopy,
		DeckState:      g.deck.GetState(),
		Players:        playersCopy,
		CurrentPlayer:  curID,
		Winners:        winners,
	}
}

// ModifyPlayers executes the provided function while holding the game's write
// lock, giving callers safe, exclusive access to the underlying slice of
// players. This is useful for code that needs to mutate player state outside
// of the poker package (for example, when restoring snapshots) while still
// guaranteeing there are no data races with concurrent reads performed via
// GetStateSnapshot.
func (g *Game) ModifyPlayers(fn func(players []*Player)) {
	g.mu.Lock()
	defer g.mu.Unlock()
	fn(g.players)
}

// ForceSetPot sets the amount of the main pot directly. This is intended to
// be used only during server-side restoration when rebuilding a game from a
// persisted snapshot where the individual betting history is not available.
func (g *Game) ForceSetPot(amount int64) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.potManager == nil {
		g.potManager = NewPotManager(len(g.players))
	}

	// Ensure there is at least a main pot.
	if len(g.potManager.pots) == 0 {
		g.potManager.pots = []*pot{newPot(0)}
	}

	// Set the amount on the main pot directly.
	g.potManager.pots[0].amount = amount
}

// SetOnNewHandStartedCallback registers a callback to be executed each time a
// new hand is successfully auto-started. The callback will be invoked from the
// auto-start timer goroutine, so it MUST be thread-safe and return quickly.
func (g *Game) SetOnNewHandStartedCallback(cb func()) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.autoStartCallbacks == nil {
		g.autoStartCallbacks = &AutoStartCallbacks{}
	}
	g.autoStartCallbacks.OnNewHandStarted = cb
}

// dealFlop adds three community cards. Caller MUST hold g.mu.
func (g *Game) dealFlop() {
	mustHeld(&g.mu)
	need := 3 - len(g.communityCards)
	for i := 0; i < need; i++ {
		if card, ok := g.deck.Draw(); ok {
			g.communityCards = append(g.communityCards, card)
		}
	}

	// Log flop cards for verification
	if len(g.communityCards) >= 3 {
		cardStrs := make([]string, len(g.communityCards))
		for i, c := range g.communityCards {
			cardStrs[i] = c.String()
		}
		g.log.Debugf("CARDS: Flop dealt, community cards: %v", cardStrs)
	}
}

// dealTurn adds one community card. Caller MUST hold g.mu.
func (g *Game) dealTurn() {
	mustHeld(&g.mu)
	if len(g.communityCards) < 4 {
		if card, ok := g.deck.Draw(); ok {
			g.communityCards = append(g.communityCards, card)
		}
	}

	// Log turn card for verification
	if len(g.communityCards) >= 4 {
		cardStrs := make([]string, len(g.communityCards))
		for i, c := range g.communityCards {
			cardStrs[i] = c.String()
		}
		g.log.Debugf("CARDS: Turn dealt, community cards: %v", cardStrs)
	}
}

// dealRiver adds one community card. Caller MUST hold g.mu.
func (g *Game) dealRiver() {
	mustHeld(&g.mu)
	if len(g.communityCards) < 5 {
		if card, ok := g.deck.Draw(); ok {
			g.communityCards = append(g.communityCards, card)
		}
	}

	// Log river card for verification
	if len(g.communityCards) >= 5 {
		cardStrs := make([]string, len(g.communityCards))
		for i, c := range g.communityCards {
			cardStrs[i] = c.String()
		}
		g.log.Debugf("CARDS: River dealt, community cards: %v", cardStrs)
	}
}

type evAdvance struct{} // advance current betting/phase when conditions met

type evGotoShowdown struct{} // force immediate showdown (e.g., only one alive)

// Request/command events processed by the Game FSM
type evResetForNewHandReq struct {
	users []*User
	reply chan error
}

type evMaybeCompleteReq struct{}

type evAdvanceToNextPlayerReq struct{ now time.Time }

type evInitializeCurrentPlayerReq struct{}

type evSetCurrentPlayerByIDReq struct{ id string }

type evStateFlopReq struct{ reply chan struct{} }
type evStateTurnReq struct{ reply chan struct{} }
type evStateRiverReq struct{ reply chan struct{} }

type evAddToPotForPlayerReq struct {
	idx    int
	amount int64
}

type evRefundUncalledBetsReq struct{ reply chan error }

type evHandleFoldReq struct {
	id    string
	reply chan error
}
type evHandleCallReq struct {
	id    string
	reply chan error
}
type evHandleCheckReq struct {
	id    string
	reply chan error
}
type evHandleBetReq struct {
	id     string
	amount int64
	reply  chan error
}

// handleGameEvent centralizes cross-state request handling. It returns (nextState, handled).
func handleGameEvent(g *Game, ev any) (GameStateFn, bool) {
	switch e := ev.(type) {
	case evMaybeCompleteReq:
		g.mu.Lock()
		g.maybeCompleteBettingRound()
		g.mu.Unlock()
		return nil, true
	case evAdvanceToNextPlayerReq:
		g.mu.Lock()
		g.advanceToNextPlayer(e.now)
		g.mu.Unlock()
		return nil, true
	case evInitializeCurrentPlayerReq:
		g.mu.Lock()
		g.initializeCurrentPlayer()
		g.mu.Unlock()
		return nil, true
	case evSetCurrentPlayerByIDReq:
		g.mu.Lock()
		for i, p := range g.players {
			if p != nil && p.id == e.id {
				for j, cp := range g.players {
					if cp == nil {
						continue
					}
					if j == i {
						cp.StartTurn()
					} else {
						cp.EndTurn()
					}
				}
				g.currentPlayer = i
				break
			}
		}
		g.mu.Unlock()
		return nil, true
	case evAddToPotForPlayerReq:
		g.mu.Lock()
		g.potManager.addBet(e.idx, e.amount, g.players)
		g.mu.Unlock()
		return nil, true
	case evRefundUncalledBetsReq:
		g.mu.Lock()
		var err error
		if g.potManager == nil {
			err = fmt.Errorf("potManager is nil")
		} else {
			g.refundUncalledBets()
		}
		if e.reply != nil {
			e.reply <- err
		}
		g.mu.Unlock()
		return nil, true
	case evStateFlopReq:
		g.mu.Lock()
		if len(g.communityCards) < 3 {
			g.dealFlop()
		}
		g.phase = pokerrpc.GamePhase_FLOP
		g.mu.Unlock()
		if e.reply != nil {
			e.reply <- struct{}{}
		}
		return nil, true
	case evStateTurnReq:
		g.mu.Lock()
		if len(g.communityCards) < 4 {
			g.dealTurn()
		}
		g.phase = pokerrpc.GamePhase_TURN
		g.mu.Unlock()
		if e.reply != nil {
			e.reply <- struct{}{}
		}
		return nil, true
	case evStateRiverReq:
		g.mu.Lock()
		if len(g.communityCards) < 5 {
			g.dealRiver()
		}
		g.phase = pokerrpc.GamePhase_RIVER
		g.mu.Unlock()
		if e.reply != nil {
			e.reply <- struct{}{}
		}
		return nil, true
	case evHandleFoldReq:
		// Use public API to ensure proper locking while mutating game state
		err := g.HandlePlayerFold(e.id)
		if e.reply != nil {
			e.reply <- err
		}
		return nil, true
	case evHandleCallReq:
		// Use public API to ensure proper locking while mutating game state
		err := g.HandlePlayerCall(e.id)
		if e.reply != nil {
			e.reply <- err
		}
		return nil, true
	case evHandleCheckReq:
		// Use public API to ensure proper locking while mutating game state
		err := g.HandlePlayerCheck(e.id)
		if e.reply != nil {
			e.reply <- err
		}
		return nil, true
	case evHandleBetReq:
		// Use public API to ensure proper locking while mutating game state
		err := g.HandlePlayerBet(e.id, e.amount)
		if e.reply != nil {
			e.reply <- err
		}
		return nil, true
	default:
		return nil, false
	}
}
