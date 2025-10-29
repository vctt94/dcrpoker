package poker

import (
	"context"
	"fmt"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/decred/slog"

	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	"github.com/vctt94/pokerbisonrelay/pkg/statemachine"
)

// GameStateFn represents a game state function following Rob Pike's pattern
type GameStateFn = statemachine.StateFn[Game]

// GameConfig holds configuration for a new game
type GameConfig struct {
	NumPlayers       int
	StartingChips    int64         // Fixed number of chips each player starts with
	SmallBlind       int64         // Small blind amount
	BigBlind         int64         // Big blind amount
	Seed             int64         // Optional seed for deterministic games
	AutoStartDelay   time.Duration // Delay before automatically starting next hand after showdown
	AutoAdvanceDelay time.Duration // Delay between streets when all players are all-in (0 = immediate, no sleep)
	TimeBank         time.Duration // Time bank for each player
	Log              slog.Logger   // Logger for game events
}

// GameEventType represents different types of game events sent to Table
type GameEventType int

const (
	GameEventBettingRoundComplete GameEventType = iota // Betting round complete, advance to next street
	GameEventShowdownComplete                          // Showdown processing complete, result available
	GameEventAutoStartTriggered                        // Auto-start timer fired, Table should check conditions and start if ready
	GameEventGameOver                                  // Game has ended, only one player has chips remaining
	GameEventStateUpdated                              // Generic state update (e.g., turn changed)
)

// GameEvent represents an event sent from Game FSM to Table
type GameEvent struct {
	Type           GameEventType
	ShowdownResult *ShowdownResult // Only set for GameEventShowdownComplete
	WinnerID       string          // Only set for GameEventGameOver - ID of the player who won
	ActorID        string          // Optional: actor who triggered the update (e.g., timebank auto-action)
	Action         string          // Optional: "check" or "fold" for auto-actions
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
	potManager *potManager
	currentBet int64
	round      int

	// Configuration
	config GameConfig

	// Auto-start management
	autoStartTimer    *time.Timer
	autoStartCanceled atomic.Bool

	// Auto-advance management (for all-in scenarios)
	autoAdvanceEnabled  bool
	autoAdvanceTimer    *time.Timer
	autoAdvanceCanceled atomic.Bool

	// Logger
	log slog.Logger

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

	// Player event channel (timebank expired, etc.)
	playerEventChan chan PlayerEvent
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

	if cfg.AutoAdvanceDelay == 0 {
		return nil, fmt.Errorf("poker: AutoAdvanceDelay must be set to a positive duration (e.g., 1s)")
	}

	// Create a new deck with the given seed (or random if not specified)
	var rng *rand.Rand
	if cfg.Seed != 0 {
		rng = rand.New(rand.NewSource(cfg.Seed))
	} else {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	g := &Game{
		players:        make([]*Player, 0, cfg.NumPlayers), // Empty slice, Table will populate
		currentPlayer:  0,
		dealer:         -1, // Start at -1 so first advancement makes it 0
		deck:           newDeck(rng),
		communityCards: nil,
		potManager:     NewPotManager(cfg.NumPlayers),
		currentBet:     0,
		round:          0,
		config:         cfg,
		log:            cfg.Log,
		phase:          pokerrpc.GamePhase_NEW_HAND_DEALING,
	}

	// Initialize state machine with first state function
	g.sm = statemachine.New(g, stateNewHandDealing, 32)
	if g.sm == nil {
		g.log.Errorf("game state machine not running")
		return nil, fmt.Errorf("game state machine not running")
	}

	// Player events channel and loop
	g.playerEventChan = make(chan PlayerEvent, 32)
	go func(ch <-chan PlayerEvent) {
		for ev := range ch {
			switch ev.Type {
			case PlayerEventTimebankExpired:
				g.sm.Send(evTimebankExpiredReq{id: ev.PlayerID})
			}
		}
	}(g.playerEventChan)

	return g, nil
}

func (g *Game) Start(ctx context.Context) { g.sm.Start(ctx) }

// StartFromRestoredSnapshot starts the FSM in a passive restored state that
// does not mutate game fields on entry and only processes events. This keeps
// the restored phase/current bet/community cards intact while still allowing
// evAdvance / timebank / action events to flow through the FSM.
func (g *Game) StartFromRestoredSnapshot(ctx context.Context) {
	g.mu.Lock()
	// Replace the state machine with a restored passive state
	g.sm = statemachine.New(g, stateRestored, 32)
	g.mu.Unlock()
	g.sm.Start(ctx)
}

// TriggerTimebankExpiredFor simulates a player's timebank expiry by emitting
// the corresponding internal event. Intended for tests only.
func (g *Game) TriggerTimebankExpiredFor(playerID string) {
	sm := g.sm
	if sm != nil {
		sm.Send(evTimebankExpiredReq{id: playerID})
	}
}

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
	g.winners = nil

	// Reset auto-advance state for new hand
	g.autoAdvanceEnabled = false
	g.cancelAutoAdvance()

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
			return stateFlop
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

	// Check if auto-advance is enabled (all players all-in)
	autoAdvance := g.autoAdvanceEnabled
	g.log.Debugf("stateFlop: entered, autoAdvanceEnabled=%v", autoAdvance)
	g.mu.Unlock()

	// Emit betting round complete event AFTER dealing and state mutation
	// This allows clients to see the flop cards with the correct phase
	g.sendTableEvent(GameEvent{Type: GameEventBettingRoundComplete})

	// If auto-advance is enabled, schedule the next advance
	if autoAdvance {
		g.log.Debugf("stateFlop: auto-advance enabled, scheduling advance to TURN")
		g.scheduleAutoAdvance()
	} else {
		g.log.Debugf("stateFlop: auto-advance NOT enabled, waiting for player actions")
	}

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
			return stateTurn
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

	// Check if auto-advance is enabled (all players all-in)
	autoAdvance := g.autoAdvanceEnabled
	g.log.Debugf("stateTurn: entered, autoAdvanceEnabled=%v", autoAdvance)
	g.mu.Unlock()

	// Emit betting round complete event AFTER dealing and state mutation
	// This allows clients to see the turn card with the correct phase
	g.sendTableEvent(GameEvent{Type: GameEventBettingRoundComplete})

	// If auto-advance is enabled, schedule the next advance
	if autoAdvance {
		g.log.Debugf("stateTurn: auto-advance enabled, scheduling advance to RIVER")
		g.scheduleAutoAdvance()
	} else {
		g.log.Debugf("stateTurn: auto-advance NOT enabled, waiting for player actions")
	}

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
			return stateRiver
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

	// Check if auto-advance is enabled (all players all-in)
	autoAdvance := g.autoAdvanceEnabled
	g.log.Debugf("stateRiver: entered, autoAdvanceEnabled=%v", autoAdvance)
	g.mu.Unlock()

	// Emit betting round complete event AFTER dealing and state mutation
	// This allows clients to see the river card with the correct phase
	g.sendTableEvent(GameEvent{Type: GameEventBettingRoundComplete})

	// If auto-advance is enabled, schedule advance to showdown
	if autoAdvance {
		g.log.Debugf("stateRiver: auto-advance enabled, scheduling advance to SHOWDOWN")
		// Disable auto-advance for showdown (it's the final state)
		g.mu.Lock()
		g.autoAdvanceEnabled = false
		g.mu.Unlock()

		// Schedule the advance to showdown
		g.scheduleAutoAdvance()
	} else {
		g.log.Debugf("stateRiver: auto-advance NOT enabled, waiting for player actions")
	}

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
			return stateShowdown
		default:
		}
	}
	return nil
}

// stateRestored is a passive state used after restoring a game from a snapshot.
// It does not mutate game state on entry. It forwards generic events via
// handleGameEvent and only transitions on explicit evAdvance/evGotoShowdown.
func stateRestored(g *Game, in <-chan any) GameStateFn {
	for ev := range in {
		// Allow cross-state requests (handle/ack via FSM)
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
			// Advance solely based on current phase
			g.mu.Lock()
			phase := g.phase
			g.mu.Unlock()
			switch phase {
			case pokerrpc.GamePhase_PRE_FLOP:
				return stateFlop
			case pokerrpc.GamePhase_FLOP:
				return stateTurn
			case pokerrpc.GamePhase_TURN:
				return stateRiver
			case pokerrpc.GamePhase_RIVER:
				return stateShowdown
			default:
				// ignore
			}
		default:
			// ignore
		}
	}
	return nil
}

// RestoreHoleCards rebuilds the current hand and populates players' hole cards
// from a persisted snapshot of players. It replaces any existing hand.
func (g *Game) RestoreHoleCards(players []PlayerSnapshot) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Build player ID list from current game players to initialize Hand
	ids := make([]string, 0, len(g.players))
	for _, p := range g.players {
		if p != nil {
			ids = append(ids, p.ID())
		}
	}
	h := NewHand(ids)

	// Populate hole cards from snapshot when available
	for _, ps := range players {
		if len(ps.Hand) == 0 {
			continue
		}
		for _, c := range ps.Hand {
			_ = h.DealCardToPlayer(ps.ID, c)
		}
	}

	g.currentHand = h
	return nil
}

// RestorePotFromSnapshot overrides the pot manager with a single pot equal to
// the captured amount and marks all non-folded players as eligible. This is a
// pragmatic restore path that preserves pot value across reconnects when per-
// player contributions are not available in the snapshot.
func (g *Game) RestorePotFromSnapshot(amount int64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.potManager == nil {
		g.potManager = NewPotManager(len(g.players))
	}
	n := len(g.players)
	p := newPot(n)
	p.amount = amount
	for i, pl := range g.players {
		if pl == nil {
			continue
		}
		if pl.GetCurrentStateString() != "FOLDED" {
			p.makeEligible(i)
		}
	}
	g.potManager.pots = []*pot{p}
	g.potManager.currentBets = make(map[int]int64)
	g.potManager.totalBets = make(map[int]int64)
}

// RestorePotsFromContributions rebuilds the pot manager from a map of total
// chip contributions per player ID. It derives eligibility from live player
// states (non-folded players are eligible) and resets per-street current bets.
func (g *Game) RestorePotsFromContributions(contrib map[string]int64) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Only create new pot manager if it doesn't exist
	if g.potManager == nil {
		g.potManager = NewPotManager(len(g.players))
		// Reset aggregates only when creating new pot manager
		g.potManager.currentBets = make(map[int]int64)
		g.potManager.totalBets = make(map[int]int64)
	}
	// If pot manager already exists, preserve its current state

	// Map player ID -> index
	idxByID := make(map[string]int, len(g.players))
	for i, p := range g.players {
		if p != nil {
			idxByID[p.id] = i
		}
	}

	// Apply contributions by index
	for id, amt := range contrib {
		if amt <= 0 {
			continue
		}
		if idx, ok := idxByID[id]; ok {
			g.potManager.totalBets[idx] = amt
		}
	}

	// Fold status snapshot under Game.mu
	foldStatus := make([]bool, len(g.players))
	for i, p := range g.players {
		if p != nil {
			p.mu.RLock()
			foldStatus[i] = p.hasFolded
			p.mu.RUnlock()
		}
	}

	g.potManager.rebuildPotsIncremental(g.players, foldStatus)
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

	// Check if only one player has chips remaining (game over condition)
	playersWithChips := 0
	var lastPlayerID string
	for _, p := range g.players {
		if p != nil {
			p.mu.RLock()
			balance := p.balance
			id := p.id
			p.mu.RUnlock()
			if balance > 0 {
				playersWithChips++
				lastPlayerID = id
			}
		}
	}

	gameOver := playersWithChips <= 1
	if gameOver && playersWithChips == 1 {
		g.log.Infof("stateShowdown: game over! Player %s wins with all chips", lastPlayerID)
	} else if gameOver {
		g.log.Infof("stateShowdown: game over! No players have chips remaining")
	}

	g.mu.Unlock()

	// Send game over event if game has ended
	if gameOver {
		g.log.Debugf("stateShowdown: sending GameEventGameOver")
		g.sendTableEvent(GameEvent{Type: GameEventGameOver, WinnerID: lastPlayerID})
	}

	// Only schedule auto-start if game is not over and configured
	if !gameOver && g.config.AutoStartDelay > 0 {
		g.log.Debugf("stateShowdown: scheduling auto-start")
		g.ScheduleAutoStart()
	} else if gameOver {
		g.log.Debugf("stateShowdown: game over, not scheduling auto-start")
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

	// Credit the refunded amount directly to player balance (Game FSM owns balance)
	if hiPlayer >= 0 && refunded > 0 {
		g.players[hiPlayer].balance += refunded
		// Notify player FSM about balance change (async, non-blocking)
		g.players[hiPlayer].NotifyBalanceChange(g.players[hiPlayer].balance, refunded, "uncalled_bet_refund")
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
	autoStartTimer := g.autoStartTimer
	autoAdvanceTimer := g.autoAdvanceTimer
	playerCh := g.playerEventChan
	g.mu.Unlock()

	// Cancel all player timebank timers and sever their event channels
	for _, p := range players {
		if p != nil {
			p.cancelTimebank()
			p.mu.Lock()
			p.playerEventChan = nil
			p.mu.Unlock()
		}
	}

	// Close player event channel to stop the forwarder goroutine
	if playerCh != nil {
		close(playerCh)
	}

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
	g.playerEventChan = nil
	g.mu.Unlock()

	// Cancel timers if running (timer.Stop() is safe to call concurrently)
	if autoStartTimer != nil {
		autoStartTimer.Stop()
	}
	if autoAdvanceTimer != nil {
		autoAdvanceTimer.Stop()
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
		// Wire timebank scheduling
		player.timebankDelay = g.config.TimeBank
		player.playerEventChan = g.playerEventChan
		player.mu.Unlock()

		g.players[i] = player
	}
}

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
			p.timebankDelay = g.config.TimeBank
			p.playerEventChan = g.playerEventChan
			newPlayers = append(newPlayers, p)
		} else {
			// New seat joined between hands
			np := NewPlayer(u.ID, u.Name, g.config.StartingChips)
			np.tableSeat = u.TableSeat
			np.isReady = u.IsReady
			np.timebankDelay = g.config.TimeBank
			np.playerEventChan = g.playerEventChan
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
	g.deck = newDeck(nextRng)

	// Clear currentHand so statePreDeal creates a fresh one
	g.currentHand = nil

	return nil
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

	// If only one player remains, send showdown event to table
	if g.unfoldsPlayers() == 1 {
		// End the player's turn before showdown
		p.EndTurn()
		// Notify the game FSM to enter stateShowdown - it will update phase
		if g.sm != nil {
			g.sm.Send(evGotoShowdown{})
		}
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
		// Distribute pots (no actual comparison needed) - directly modifies player.balance under g.mu
		err := g.potManager.distributePots(g.players)
		if err != nil {
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
		g.log.Debugf("CARDS: Player %s evaluated: rank=%v rankValue=%d bestHand=%v description=%s",
			p.id, hv.Rank, hv.RankValue, bestStrs, hv.HandDescription)
	}

	// Distribute pots according to eligibility - directly modifies player.balance under g.mu
	err := g.potManager.distributePots(g.players)
	if err != nil {
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

	// 3) All alive are ALL_IN -> enable auto-advance and let FSM progress streets.
	// Also enable if only one player is active but all others are all-in (no one left to bet against)
	// AND all bets are matched (no unmatched bets).
	allInCount := alive - active
	g.log.Debugf("maybeCompleteBettingRound: alive=%d, active=%d, allInCount=%d, unmatched=%d", alive, active, allInCount, unmatched)
	if active == 0 || (active == 1 && allInCount > 0 && unmatched == 0) {
		// Enable auto-advance mode so state handlers know to schedule timers
		g.autoAdvanceEnabled = true
		if active == 0 {
			g.log.Debugf("maybeCompleteBettingRound: all %d players all-in, enabling auto-advance mode", alive)
		} else {
			g.log.Debugf("maybeCompleteBettingRound: %d player(s) active but %d all-in (no one to bet against), enabling auto-advance mode", active, allInCount)
		}

		// Send evAdvance to progress to the next street
		// The FSM state handlers will deal cards, emit events, and schedule the next advance
		g.sm.Send(evAdvance{})
		return
	}

	// 4) Heads-up special case: if exactly one actionable remains with unmatched bets, they must act.
	// Do NOT auto-advance when active==1 if they still have a bet to respond to AND someone can respond.
	if len(g.players) == 2 && active == 1 && unmatched > 0 && allInCount == 0 {
		return
	}

	// 5) Need all actionable bets matched to be able to advance.
	if unmatched > 0 {
		return
	}

	// 6) Ensure we've completed a full rotation: current player must be the
	// street starter (the player who would act first on this street).
	starter := g.computeStreetStarterIndexLocked()
	if starter < 0 {
		return
	}
	if g.currentPlayer != starter {
		return
	}

	// 7) Street is complete -> signal FSM to advance exactly one street.
	// Dealing and phase updates occur in state handlers.
	g.sm.Send(evAdvance{})

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
	g.initializeCurrentPlayer()
}

// computeStreetStarterIndexLocked returns the index of the first actionable
// player for the current street (skips folded and all-in players).
// Requires: g.mu held.
func (g *Game) computeStreetStarterIndexLocked() int {
	mustHeld(&g.mu)
	n := len(g.players)
	if n == 0 {
		return -1
	}
	start := 0
	if g.phase == pokerrpc.GamePhase_PRE_FLOP {
		if n == 2 {
			start = g.dealer
		} else {
			start = (g.dealer + 3) % n
		}
	} else {
		start = (g.dealer + 1) % n
	}
	for i := 0; i < n; i++ {
		idx := (start + i) % n
		p := g.players[idx]
		if p == nil {
			continue
		}
		st := p.GetCurrentStateString()
		if st != "FOLDED" && st != "ALL_IN" {
			return idx
		}
	}
	return -1
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

		// Use the unified player state directly; only IN_GAME can act.
		if g.players[g.currentPlayer].GetCurrentStateString() == "IN_GAME" {
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

// RestoreDeckState restores the underlying deck from a serialized DeckState.
// It replaces the remaining cards while preserving the existing RNG.
func (g *Game) RestoreDeckState(state *DeckState) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if state == nil {
		return fmt.Errorf("deck state is nil")
	}
	if g.deck == nil {
		// NewGame always initializes a deck, but guard just in case.
		// Use a fresh RNG; order is defined by the provided cards.
		g.deck = newDeck(rand.New(rand.NewSource(time.Now().UnixNano())))
	}
	return g.deck.restoreState(state)
}

// SetGameState allows restoring game state from persistence
func (g *Game) SetGameState(dealer, round int, currentBet, pot int64, phase pokerrpc.GamePhase) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.dealer = dealer
	g.round = round
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
	// Skip setting a current player at SHOWDOWN (no actions allowed).
	if g.phase != pokerrpc.GamePhase_SHOWDOWN {
		g.initializeCurrentPlayer() // uses phase/dealer and skips folded/all-in/disconnected
	} else {
		g.currentPlayer = -1
		for _, p := range g.players {
			if p != nil {
				p.EndTurn()
			}
		}
	}

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

	// Check if auto-start is configured (zero or negative delay disables auto-start)
	if g.config.AutoStartDelay <= 0 {
		g.log.Debugf("scheduleAutoStart: auto-start disabled, delay=%v", g.config.AutoStartDelay)
		return
	}

	// Debug log
	g.log.Debugf("scheduleAutoStart: setting up timer with delay %v", g.config.AutoStartDelay)

	// Mark that auto-start is pending
	g.autoStartCanceled.Store(false)
	g.autoStartTimer = time.AfterFunc(g.config.AutoStartDelay, func() {
		if g.autoStartCanceled.Load() {
			return
		}
		g.mu.Lock()
		if g.autoStartCanceled.Load() {
			g.mu.Unlock()
			return
		}
		g.autoStartTimer = nil
		g.mu.Unlock()
		g.sendTableEvent(GameEvent{Type: GameEventAutoStartTriggered})
	})
}

// cancelAutoStart is the internal implementation (assumes lock is held)
func (g *Game) cancelAutoStart() {
	mustHeld(&g.mu)
	g.autoStartCanceled.Store(true)
	if g.autoStartTimer != nil {
		g.autoStartTimer.Stop()
		g.autoStartTimer = nil
	}
}

// scheduleAutoAdvance schedules automatic advance to next street when all players are all-in
// This function should be called WITHOUT holding g.mu (it will acquire the lock)
func (g *Game) scheduleAutoAdvance() {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Cancel any existing auto-advance timer
	g.cancelAutoAdvance()

	// If auto-advance is disabled (negative delay), don't schedule
	if g.config.AutoAdvanceDelay < 0 {
		g.log.Debugf("scheduleAutoAdvance: auto-advance disabled, delay=%v", g.config.AutoAdvanceDelay)
		return
	}

	// Debug log
	g.log.Debugf("scheduleAutoAdvance: setting up timer with delay %v", g.config.AutoAdvanceDelay)

	// Mark that auto-advance is pending
	g.autoAdvanceCanceled.Store(false)

	// Schedule the auto-advance timer
	// Even if delay is 0, we still use AfterFunc to avoid blocking and maintain event ordering
	g.autoAdvanceTimer = time.AfterFunc(g.config.AutoAdvanceDelay, func() {
		// Fast path: bail without locking if canceled.
		if g.autoAdvanceCanceled.Load() {
			g.log.Debugf("Auto-advance timer was canceled, not sending evAdvance")
			return
		}

		// Optional double-check under lock, and clear pointer.
		g.mu.Lock()
		if g.autoAdvanceCanceled.Load() {
			g.mu.Unlock()
			g.log.Debugf("Auto-advance canceled (post-lock), not sending evAdvance")
			return
		}
		// Snapshot sm again in case it changed.
		localSM := g.sm
		// One-shot timer: clear pointer to avoid stale refs.
		g.autoAdvanceTimer = nil
		g.mu.Unlock()

		if localSM == nil {
			g.log.Debugf("Auto-advance fired but FSM is nil, skipping")
			return
		}

		g.log.Debugf("Auto-advance timer fired, sending evAdvance to FSM")
		// Send without holding g.mu to keep lock order clean.
		localSM.Send(evAdvance{})
	})
}

// cancelAutoAdvance cancels the auto-advance timer
func (g *Game) cancelAutoAdvance() {
	mustHeld(&g.mu)

	g.autoAdvanceCanceled.Store(true)
	// Then stop & clear the one-shot timer.
	if t := g.autoAdvanceTimer; t != nil {
		_ = t.Stop() // AfterFunc: no channel to drain
		g.autoAdvanceTimer = nil
	}
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
	DeckState      *DeckState
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
			IsSmallBlind:    player.isSmallBlind,
			IsBigBlind:      player.isBigBlind,
			IsTurn:          player.isTurn,
			IsDisconnected:  player.isDisconnected,
			Hand:            nil, // Will be populated from currentHand below
			HandDescription: player.handDescription,
			HandValue:       player.handValue,
			LastAction:      player.lastAction,
			StateString:     player.getCurrentStateString(),
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

	// Derive bet round index from phase for snapshot compatibility
	var br int
	switch g.phase {
	case pokerrpc.GamePhase_PRE_FLOP:
		br = 0
	case pokerrpc.GamePhase_FLOP:
		br = 1
	case pokerrpc.GamePhase_TURN:
		br = 2
	case pokerrpc.GamePhase_RIVER:
		br = 3
	}

	return GameStateSnapshot{
		Dealer:         g.dealer,
		CurrentBet:     g.currentBet,
		Pot:            potAmount,
		Round:          g.round,
		BetRound:       br,
		Phase:          g.phase,
		CommunityCards: communityCardsCopy,
		DeckState:      g.deck.GetState(),
		Players:        playersCopy,
		CurrentPlayer:  curID,
		Winners:        winners,
	}
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

// Fired by Player's per-turn timer; request the game to auto-act.
type evTimebankExpiredReq struct{ id string }

// handleGameEvent centralizes cross-state request handling. It returns (nextState, handled).
func handleGameEvent(g *Game, ev any) (GameStateFn, bool) {
	switch e := ev.(type) {
	case evMaybeCompleteReq:
		g.mu.Lock()
		g.maybeCompleteBettingRound()
		g.mu.Unlock()
		return nil, true
	case evTimebankExpiredReq:
		// Decide auto action without holding g.mu while calling external APIs
		var act string
		g.mu.Lock()
		// Validate phase
		actionable := (g.phase == pokerrpc.GamePhase_PRE_FLOP || g.phase == pokerrpc.GamePhase_FLOP || g.phase == pokerrpc.GamePhase_TURN || g.phase == pokerrpc.GamePhase_RIVER)
		if actionable && g.currentPlayer >= 0 && g.currentPlayer < len(g.players) && g.players[g.currentPlayer] != nil && g.players[g.currentPlayer].ID() == e.id {
			p := g.players[g.currentPlayer]
			need := g.currentBet - p.currentBet
			if need <= 0 {
				act = "check"
			} else {
				act = "fold"
			}
		}
		g.mu.Unlock()
		switch act {
		case "check":
			_ = g.HandlePlayerCheck(e.id)
		case "fold":
			_ = g.HandlePlayerFold(e.id)
		}
		// Notify table to push a fresh GameUpdate and optional typed action event
		g.sendTableEvent(GameEvent{Type: GameEventStateUpdated, ActorID: e.id, Action: act})
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
