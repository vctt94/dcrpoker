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
)

// GameEvent represents an event sent from Game FSM to Table
type GameEvent struct {
	Type GameEventType
}

// Game holds the context and data for our poker game
type Game struct {
	// Player management - references to table users converted to players
	players       []*Player // Internal player objects managed by game
	currentPlayer int
	dealer        int

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

	mu sync.RWMutex

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
func (g *Game) sendTableEvent(eventType GameEventType) {
	if g.tableEventChan != nil {
		select {
		case g.tableEventChan <- GameEvent{Type: eventType}:
		default:
			// Channel full or closed, log and continue
			g.log.Warnf("Failed to send game event %v to table (channel full)", eventType)
		}
	}
}

// checkBettingRoundComplete checks if the betting round is complete and sends
// appropriate events to Table. Should be called after each player action.
// Assumes g.mu is held by caller.
func (g *Game) checkBettingRoundComplete() {
	// Count alive (non-folded) and active (non-folded, non-all-in) players
	alivePlayers := 0
	activePlayers := 0
	for _, p := range g.players {
		if p == nil {
			continue
		}
		// Check FSM state - if ALL_IN state function is running, player is all-in
		state := p.GetCurrentStateString()
		if state != "FOLDED" {
			alivePlayers++
			if state != "ALL_IN" {
				activePlayers++
			}
		}
	}

	// If only one alive player remains (others folded), go to showdown
	if alivePlayers <= 1 {
		g.log.Debugf("checkBettingRoundComplete: only %d alive, sending showdown event", alivePlayers)
		g.sendTableEvent(GameEventShowdownReady)
		return
	}

	// If betting is effectively closed (all remaining players are all-in or only one can act)
	if activePlayers <= 1 {
		g.log.Debugf("checkBettingRoundComplete: only %d active players, sending showdown event", activePlayers)
		g.sendTableEvent(GameEventShowdownReady)
		return
	}

	// Check if all active players have had a chance to act and all bets are equal
	if g.actionsInRound < activePlayers {
		g.log.Debugf("checkBettingRoundComplete: waiting for actions: %d/%d", g.actionsInRound, activePlayers)
		return // Not all players have acted yet
	}

	// Check if all active players have matching bets
	unmatchedPlayers := 0
	for _, p := range g.players {
		if p == nil {
			continue
		}
		state := p.GetCurrentStateString()
		if state == "FOLDED" || state == "ALL_IN" {
			continue // Folded and all-in players are considered "matched"
		}
		if p.CurrentBet() < g.currentBet {
			unmatchedPlayers++
		}
	}

	if unmatchedPlayers > 0 {
		g.log.Debugf("checkBettingRoundComplete: %d players with unmatched bets", unmatchedPlayers)
		return // Still waiting for some players to match
	}

	// Betting round is complete — advance phase inside the FSM and notify Table to publish.
	g.log.Debugf("checkBettingRoundComplete: round complete; advancing phase via FSM and notifying table")
	// Perform phase transition, reset counters, and reinitialize current player.
	// This keeps game-state mutations inside the Game FSM instead of the Table.
	g.maybeCompleteBettingRound()

	// Notify the Table that the betting round completed. The Table will
	// inspect current phase and decide whether to publish NEW_ROUND or
	// trigger showdown publishing. No game mutations should occur there.
	g.sendTableEvent(GameEventBettingRoundComplete)
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
	g.winners = nil
	// Reset pot manager for the new hand
	g.potManager = NewPotManager(len(g.players))
	// Deck reseed is done prior to dealing hole cards
	g.phase = pokerrpc.GamePhase_PRE_FLOP

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

		// POST BLINDS while players are still in AT_TABLE state
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

			// Use FSM event to post blind (players are still in AT_TABLE state)
			var applied int64
			if p.sm != nil {
				reply := make(chan int64, 1)
				p.sm.Send(evPostBlind{Amt: delta, Reply: reply})
				applied = <-reply // Wait for FSM to process and return actual amount
			}

			// Reflect into pot manager and table-wide current bet
			g.potManager.addBet(pos, applied, g.players) // contract: g.mu held
			finalBet := already + applied
			if finalBet > g.currentBet {
				g.currentBet = finalBet
			}
		}

		postBlind(sbPos, g.config.SmallBlind)
		postBlind(bbPos, g.config.BigBlind)

		// AFTER blinds are posted, notify all players to start the hand
		// This transitions them from AT_TABLE to IN_GAME (or ALL_IN if they went all-in on blinds)
		for _, p := range g.players {
			if p != nil && p.sm != nil {
				p.sm.Send(evStartHand{})
			}
		}
	}

	g.log.Debugf("statePreDeal: transitioned to PRE_FLOP phase, round=%d, dealer=%d", g.round, g.dealer)
	g.mu.Unlock()
	return stateDeal
}

func stateDeal(g *Game, in <-chan any) GameStateFn {
	// Table does actual dealing; proceed to preflop
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
	g.potManager.currentBets = make(map[int]int64) // (contract: called with g.mu held)
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
	g.potManager.currentBets = make(map[int]int64)
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
	g.potManager.currentBets = make(map[int]int64)
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
func (g *Game) RefundUncalledBets() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.potManager == nil {
		return fmt.Errorf("potManager is nil")
	}
	if err := g.potManager.returnUncalledBet(g.players); err != nil {
		return fmt.Errorf("failed to refund uncalled bets: %w", err)
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

	// Do NOT reset remaining hand-level state here; FSM will do it in statePreDeal.
	// The Table will send evStartHand after dealing hole cards.
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
		if p != nil && p.StateID() != psFolded {
			n++
		}
	}
	return n
}

// handlePlayerFold is the core logic without locking (for internal use)
func (g *Game) handlePlayerFold(playerID string) error {

	p := g.getPlayerByID(playerID)
	if p == nil {
		return fmt.Errorf("player not found in game")
	}
	if g.currentPlayerID() != playerID {
		return fmt.Errorf("not your turn to act")
	}

	// Send synchronous fold event to ensure player's state is updated before
	// we evaluate betting-round completion (avoids races).
	if p.sm != nil {
		reply := make(chan error, 1)
		p.sm.Send(evFoldReq{Reply: reply})
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
		g.sendTableEvent(GameEventShowdownReady)
		return nil
	}

	// End the player's turn before advancing
	p.EndTurn()

	// Move turn to next alive player
	g.advanceToNextPlayer(time.Now()) // must skip folded players

	// Check if betting round is complete and notify table
	g.checkBettingRoundComplete()

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
func (g *Game) handlePlayerCall(playerID string) error {
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
	if player.sm != nil {
		player.sm.Send(evCallDelta{Amt: delta, Reply: reply})
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

	// Only advance if there are players who can act
	if activePlayers > 1 {
		g.advanceToNextPlayer(time.Now())
	}

	// Check if betting round is complete and notify table
	g.checkBettingRoundComplete()

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
func (g *Game) handlePlayerCheck(playerID string) error {
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
	g.checkBettingRoundComplete()

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
func (g *Game) handlePlayerBet(playerID string, amount int64) error {
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
	if delta > 0 && player.sm != nil {
		reply := make(chan error, 1)
		player.sm.Send(evBet{Amt: delta, Reply: reply})
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
	g.checkBettingRoundComplete()

	return nil
}

// getPlayerByID finds a player by ID
func (g *Game) getPlayerByID(playerID string) *Player {
	for _, p := range g.players {
		if p.ID() == playerID {
			return p
		}
	}
	return nil
}

// currentPlayerID returns the current player's ID
func (g *Game) currentPlayerID() string {
	if g.currentPlayer < 0 || g.currentPlayer >= len(g.players) {
		return ""
	}
	return g.players[g.currentPlayer].ID()
}

// advanceToNextPlayer moves to the next active player
// Note: The caller (action handlers like bet, call, check, fold) is responsible
// for ending the current player's turn before calling this function.
func (g *Game) advanceToNextPlayer(now time.Time) {
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
	// Public API now delegates to the internal locked implementation.
	return g.handleShowdown()
}

// handleShowdown is the core logic without locking (for internal use)
func (g *Game) handleShowdown() (*ShowdownResult, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.log.Debugf("handleShowdown: entered showdown processing")

	// Gather active (non-folded) players
	unfoldedPlayers := make([]*Player, 0, len(g.players))
	for _, player := range g.players {
		if player != nil && player.GetCurrentStateString() != "FOLDED" {
			unfoldedPlayers = append(unfoldedPlayers, player)
		}
	}

	// Prepare result
	result := &ShowdownResult{
		Winners:    make([]string, 0),
		WinnerInfo: make([]*pokerrpc.Winner, 0),
		TotalPot:   0, // Will be set from snapshot BEFORE refunds/distribution for notifications
	}

	// Snapshot pot for notification BEFORE any refunds/distribution so
	// events can reflect the headline amount that was pushed forward,
	// even if some portion is uncalled and will be refunded.
	potForNotification := g.potManager.getTotalPot()
	result.TotalPot = potForNotification

	// Now, ensure any uncalled portion from the last betting action is
	// refunded before resolving pots to maintain correct side-pot structure
	// and winner payouts.
	if err := g.potManager.returnUncalledBet(g.players); err != nil {
		return nil, fmt.Errorf("failed to return uncalled bet: %w", err)
	}

	// --- Uncontested (fold-win): build pots, award total, reset state
	if len(unfoldedPlayers) == 1 {
		winner := unfoldedPlayers[0]
		g.log.Infof("HERE ON ONE ACTIVE PLAYER: %s", winner.id)

		sum := int64(0)
		for _, p := range g.potManager.pots {
			sum += p.amount
		}

		// Total pot for the event already captured for notification

		// --- Use delta accounting to populate result (avoids “empty winners”)
		prev := make(map[string]int64, len(g.players))
		for _, p := range g.players {
			if p != nil {
				p.mu.RLock()
				prev[p.id] = p.balance
				p.mu.RUnlock()
			}
		}

		g.potManager.distributePots(g.players)

		// Fill result from actual balance deltas (handles any future edge cases too)
		totalWinnings := int64(0)
		for _, p := range g.players {
			if p == nil {
				continue
			}
			p.mu.RLock()
			delta := p.balance - prev[p.id]
			p.mu.RUnlock()
			if delta > 0 {
				result.Winners = append(result.Winners, p.id)

				// Best hand (use hole cards if board < 5)
				var best []Card
				if len(p.hand)+len(g.communityCards) >= 5 {
					hv, err := EvaluateHand(p.hand, g.communityCards)
					if err != nil {
						return nil, fmt.Errorf("failed to evaluate hand for player %s: %w", p.id, err)
					}
					p.mu.Lock()
					p.handValue = &hv
					p.handDescription = GetHandDescription(hv)
					p.mu.Unlock()
					best = hv.BestHand
				} else {
					best = p.hand
				}

				result.WinnerInfo = append(result.WinnerInfo, &pokerrpc.Winner{
					PlayerId: p.id,
					BestHand: CreateHandFromCards(best),
					Winnings: delta,
				})
				totalWinnings += delta
			}
		}

		// Now reset for next hand (and clear unswept for clean logs)

		g.phase = pokerrpc.GamePhase_SHOWDOWN
		g.winners = result.Winners
		g.log.Infof("result: %+v", result)
		return result, nil
	}

	// --- True showdown: ensure board is fully dealt if multiple players remain
	if len(unfoldedPlayers) >= 2 && len(g.communityCards) < 5 {
		// Fast-forward dealing based on current phase/board size
		// It's safe to call these as they lock internally.
		dealOne := func() (Card, bool) { return g.deck.Draw() }
		switch g.phase {
		case pokerrpc.GamePhase_PRE_FLOP:
			// flop (3)
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
			// turn (1)
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
			// river (1)
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

	// After auto-deal, validate again for safety
	for _, p := range unfoldedPlayers {
		if len(p.hand)+len(g.communityCards) < 5 {
			return nil, fmt.Errorf("invalid showdown: player %s has insufficient cards (hole=%d, board=%d)",
				p.id, len(p.hand), len(g.communityCards))
		}
	}

	// Evaluate each active player's hand
	for _, p := range unfoldedPlayers {
		hv, err := EvaluateHand(p.hand, g.communityCards)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate hand for player %s: %w", p.id, err)
		}
		p.handValue = &hv
		p.handDescription = GetHandDescription(hv)
		g.log.Debugf("handleShowdown: player %s hand=%v description=%s", p.id, p.hand, p.handDescription)
	}

	// Use the pre-refund snapshot for notification consistency
	result.TotalPot = potForNotification

	g.log.Debugf("handleShowdown: total pots=%d", len(g.potManager.pots))
	for i, pot := range g.potManager.pots {
		g.log.Debugf("handleShowdown: pot %d amount=%d eligible_players=%v", i, pot.amount, pot.eligibility)
	}

	// Snapshot balances to compute exact deltas
	prev := make(map[string]int64, len(g.players))
	for _, p := range g.players {
		if p != nil {
			p.mu.RLock()
			prev[p.id] = p.balance
			g.log.Debugf("handleShowdown: player %s balance before distribution=%d", p.id, p.balance)
			p.mu.RUnlock()
		}
	}

	// Distribute pots
	if err := g.potManager.distributePots(g.players); err != nil {
		g.log.Errorf("Failed to distribute pots: %v", err)
		return nil, err
	}

	// Collect winners by positive delta
	for _, p := range g.players {
		if p == nil {
			continue
		}
		p.mu.RLock()
		delta := p.balance - prev[p.id]
		g.log.Debugf("handleShowdown: player %s balance after distribution=%d delta=%d", p.id, p.balance, delta)
		p.mu.RUnlock()
		if delta > 0 {
			result.Winners = append(result.Winners, p.id)
			var handRank pokerrpc.HandRank
			var best []Card
			p.mu.RLock()
			hv := p.handValue
			p.mu.RUnlock()
			if hv != nil {
				handRank = hv.HandRank
				best = hv.BestHand
			} else {
				best = p.hand
			}
			result.WinnerInfo = append(result.WinnerInfo, &pokerrpc.Winner{
				PlayerId: p.id,
				HandRank: handRank,
				BestHand: CreateHandFromCards(best),
				Winnings: delta,
			})
		}
	}

	// Assertion helper: log pot sums to catch regressions
	totalWinnings := int64(0)
	for _, winner := range result.WinnerInfo {
		totalWinnings += winner.Winnings
	}

	// Mark phase and cache winners
	g.phase = pokerrpc.GamePhase_SHOWDOWN
	g.winners = result.Winners

	return result, nil
}

// maybeAdvancePhase is the core logic without locking (for internal use)
func (g *Game) maybeCompleteBettingRound() {
	// Don't advance during NEW_HAND_DEALING phase - this is managed by setupNewHandLocked()
	// which handles the complete setup sequence and phase transitions internally
	if g.phase == pokerrpc.GamePhase_NEW_HAND_DEALING {
		return
	}

	// Diagnostic: log entry state
	g.log.Debugf("maybeAdvancePhase: phase=%v actionsInRound=%d currentBet=%d",
		g.phase, g.actionsInRound, g.currentBet)

	// Count alive (non-folded) and actionable (non-folded, non-all-in) players
	// Use derived conditions instead of GetCurrentStateString() to avoid races with async FSM.
	alivePlayers := 0
	activePlayers := 0
	for _, p := range g.players {
		isFolded := p.GetCurrentStateString() == "FOLDED"
		isAllIn := p.Balance() == 0 && p.CurrentBet() > 0

		if !isFolded {
			alivePlayers++
			if !isAllIn {
				activePlayers++
			}
		}
	}

	// If only one alive player remains (others folded), finish hand now (uncontested win)
	if alivePlayers <= 1 {
		g.phase = pokerrpc.GamePhase_SHOWDOWN
		g.sm.Send(evGotoShowdown{})
		g.log.Debugf("maybeAdvancePhase: only %d alive players, moving to SHOWDOWN", alivePlayers)
		return
	}

	// If betting is effectively closed (no one or only one player can act), fast-forward and showdown.
	// - activePlayers == 0: all alive players are all-in
	// - activePlayers == 1: only one player could act, but with no opponent able to respond,
	//   further betting isn't possible (e.g., heads-up where one is all-in, or multi-way with only one non-all-in).
	if activePlayers == 0 || activePlayers == 1 {
		// Do not refund uncalled here; handle it inside handleShowdown after
		// capturing the pre-refund snapshot for notifications.
		// Fast‑forward missing streets and set phase before signaling showdown.
		switch g.phase {
		case pokerrpc.GamePhase_PRE_FLOP:
			g.dealFlop()
			g.currentBet = 0
			g.potManager.currentBets = make(map[int]int64)
			g.phase = pokerrpc.GamePhase_FLOP
			g.dealTurn()
			g.currentBet = 0
			g.potManager.currentBets = make(map[int]int64)
			g.phase = pokerrpc.GamePhase_TURN
			g.dealRiver()
			g.currentBet = 0
			g.potManager.currentBets = make(map[int]int64)
			g.phase = pokerrpc.GamePhase_RIVER
		case pokerrpc.GamePhase_FLOP:
			g.dealTurn()
			g.currentBet = 0
			g.potManager.currentBets = make(map[int]int64)
			g.phase = pokerrpc.GamePhase_TURN
			g.dealRiver()
			g.currentBet = 0
			g.potManager.currentBets = make(map[int]int64)
			g.phase = pokerrpc.GamePhase_RIVER
		case pokerrpc.GamePhase_TURN:
			g.dealRiver()
			g.currentBet = 0
			g.potManager.currentBets = make(map[int]int64)
			g.phase = pokerrpc.GamePhase_RIVER
		}
		g.phase = pokerrpc.GamePhase_SHOWDOWN
		g.sm.Send(evGotoShowdown{})
		g.log.Debugf("maybeAdvancePhase: betting closed (alive=%d, active=%d), fast-forward to SHOWDOWN", alivePlayers, activePlayers)
		return
	}

	// Check if all active players have had a chance to act and all bets are equal
	// A betting round is complete when:
	// 1. At least each active player has had one action (actionsInRound >= activePlayers)
	// 2. All active players have matching bets (or have folded)

	if g.actionsInRound < activePlayers {
		g.log.Debugf("maybeAdvancePhase: waiting for actions: %d/%d", g.actionsInRound, activePlayers)
		return // Not all players have acted yet
	}

	// Check if all active players have matching bets
	// All-in players are considered "matched" even if their bet is less than currentBet
	unmatchedPlayers := 0
	for i, p := range g.players {
		if p == nil {
			continue
		}
		state := p.GetCurrentStateString()

		if state == "FOLDED" {
			g.log.Debugf("maybeAdvancePhase: player %d (%s) is FOLDED, skipping", i, p.id)
			continue
		}
		// All-in players are considered matched regardless of their bet amount
		if state == "ALL_IN" {
			p.mu.RLock()
			cb := p.currentBet
			p.mu.RUnlock()
			g.log.Debugf("maybeAdvancePhase: player %d (%s) is ALL_IN with bet %d, considered matched", i, p.id, cb)
			continue
		}
		p.mu.RLock()
		cb := p.currentBet
		p.mu.RUnlock()
		if cb != g.currentBet {
			g.log.Debugf("maybeAdvancePhase: player %d (%s) has unmatched bet: %d != %d", i, p.id, cb, g.currentBet)
			unmatchedPlayers++
		} else {
			g.log.Debugf("maybeAdvancePhase: player %d (%s) has matched bet: %d", i, p.id, cb)
		}
	}

	if unmatchedPlayers > 0 {
		g.log.Debugf("maybeAdvancePhase: %d players have unmatched bets (currentBet=%d)", unmatchedPlayers, g.currentBet)
		return // Still players with unmatched bets
	}

	// Betting round is complete - advance to next phase
	switch g.phase {
	case pokerrpc.GamePhase_PRE_FLOP:
		// Deal flop and transition phase immediately.
		g.dealFlop()
		g.currentBet = 0
		g.potManager.currentBets = make(map[int]int64)
		g.phase = pokerrpc.GamePhase_FLOP
		g.sm.Send(evAdvance{})
	case pokerrpc.GamePhase_FLOP:
		g.dealTurn()
		g.currentBet = 0
		g.potManager.currentBets = make(map[int]int64)
		g.phase = pokerrpc.GamePhase_TURN
		g.sm.Send(evAdvance{})
	case pokerrpc.GamePhase_TURN:
		g.dealRiver()
		g.currentBet = 0
		g.potManager.currentBets = make(map[int]int64)
		g.phase = pokerrpc.GamePhase_RIVER
		g.sm.Send(evAdvance{})
	case pokerrpc.GamePhase_RIVER:
		g.phase = pokerrpc.GamePhase_SHOWDOWN
		g.sm.Send(evGotoShowdown{})
		return
	}

	// Reset for new betting round
	for _, p := range g.players {
		if p != nil {
			p.mu.Lock()
			p.currentBet = 0
			p.mu.Unlock()
			// Clear all turn flags before initializing new current player
			p.EndTurn()
		}
	}
	// table-wide currentBet is reset by FSM state handlers; avoid double-write here
	g.actionsInRound = 0 // safe: we already hold g.mu via wrapper

	// Reset current player for new betting round and update turn flags
	g.initializeCurrentPlayer()
	if g.currentPlayer >= 0 && g.currentPlayer < len(g.players) {
		g.log.Debugf("maybeAdvancePhase: new round currentPlayer=%d id=%s",
			g.currentPlayer, g.players[g.currentPlayer].id)
	}
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
	StateID         PlayerState
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
			Hand:            make([]Card, len(player.hand)),
			HandDescription: player.handDescription,
			HandValue:       player.handValue,
			LastAction:      player.lastAction,
			StateID:         player.StateID(),
		}
		copy(playerCopy.Hand, player.hand)
		player.mu.RUnlock()

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
						Hand:            make([]Card, len(player.hand)),
						HandDescription: player.handDescription,
						HandValue:       player.handValue,
						LastAction:      player.lastAction,
						StateID:         player.StateID(),
					}
					copy(winnerSnapshot.Hand, player.hand)
					player.mu.RUnlock()
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
	need := 3 - len(g.communityCards)
	for i := 0; i < need; i++ {
		if card, ok := g.deck.Draw(); ok {
			g.communityCards = append(g.communityCards, card)
		}
	}
}

// dealTurn adds one community card. Caller MUST hold g.mu.
func (g *Game) dealTurn() {
	if len(g.communityCards) < 4 {
		if card, ok := g.deck.Draw(); ok {
			g.communityCards = append(g.communityCards, card)
		}
	}
}

// dealRiver adds one community card. Caller MUST hold g.mu.
func (g *Game) dealRiver() {
	if len(g.communityCards) < 5 {
		if card, ok := g.deck.Draw(); ok {
			g.communityCards = append(g.communityCards, card)
		}
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
		if g.potManager == nil {
			if e.reply != nil {
				e.reply <- fmt.Errorf("potManager is nil")
			}
		} else {
			err := g.potManager.returnUncalledBet(g.players)
			if e.reply != nil {
				e.reply <- err
			}
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
