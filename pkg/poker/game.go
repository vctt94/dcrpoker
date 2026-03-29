package poker

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
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
	SmallBlind       int64         // Small blind amount (initial level)
	BigBlind         int64         // Big blind amount (initial level)
	Seed             int64         // Optional seed for deterministic games
	AutoStartDelay   time.Duration // Delay before automatically starting next hand after showdown
	AutoAdvanceDelay time.Duration // Delay between streets when all players are all-in (0 = immediate, no sleep)
	TimeBank         time.Duration // Time bank for each player
	Log              slog.Logger   // Logger for game events

	BlindIncreaseInterval time.Duration  // Interval between blind level increases (0 = disabled)
	BlindSchedule         []BlindLevel   // Custom schedule; nil uses DefaultBlindSchedule
}

// GameEventType represents different types of game events sent to Table
type GameEventType int

const (
	GameEventBettingRoundComplete GameEventType = iota // Betting round complete, advance to next street
	GameEventShowdownComplete                          // Showdown processing complete, result available
	GameEventAutoStartTriggered                        // Auto-start timer fired, Table should check conditions and start if ready
	GameEventGameOver                                  // Game has ended, only one player has chips remaining
	GameEventStateUpdated                              // Generic state update (e.g., turn changed)
	GameEventPlayerLost                                // Informational: player has 0 chips after showdown
	GameEventAutoShowCards                             // All players all-in -> reveal cards notification
	GameEventBlindsIncreased                           // Blind level increased (applied at hand boundary)
	GameEventBlindsPending                             // Blind increase is due; will apply next hand
)

// GameEvent represents an event sent from Game FSM to Table
type GameEvent struct {
	Type           GameEventType
	ShowdownResult *ShowdownResult // Only set for GameEventShowdownComplete
	WinnerID       string          // Only set for GameEventGameOver - ID of the player who won
	PlayerID       string          // Only set for GameEventPlayerLost - ID of the player who lost (0 chips)
	ActorID        string          // Optional: actor who triggered the update (e.g., timebank auto-action)
	Action         string          // Optional: "check" or "fold" for auto-actions
	RevealInfo     []AutoRevealPlayer
	NextBlind      *BlindLevel // Only set for GameEventBlindsPending — the upcoming level
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

	// Betting-round bookkeeping
	// lastRaiseAmount tracks the minimum legal raise increment for the current street.
	lastRaiseAmount int64
	// Index of the last player who made an aggressive action that set/increased
	// the street-wide bet (bet or raise). -1 means no aggressor this street.
	lastAggressor int

	// Configuration
	config GameConfig

	// Blind increase management — nil when blind increases are disabled.
	// The BlindManager is a separate FSM; the Game caches the current
	// level at hand boundaries and uses the cached values during play.
	blindManager   *BlindManager
	liveSmallBlind int64 // cached from BlindManager FSM
	liveBigBlind   int64 // cached from BlindManager FSM
	liveBlindLevel int   // cached level index
	liveNextBlindMs int64 // cached next-increase unix ms

	// Auto-start management
	autoStartTimer    *time.Timer
	autoStartCanceled atomic.Bool

	// Auto-advance management (for all-in scenarios)
	autoAdvanceEnabled  bool
	autoAdvanceTimer    *time.Timer
	autoAdvanceCanceled atomic.Bool

	// Shutdown coordination to avoid scheduling new work while stopping
	shuttingDown atomic.Bool

	// Logger
	log slog.Logger

	// current game phase (pre-flop, flop, turn, river, showdown)
	phase pokerrpc.GamePhase

	// Winner tracking - set after showdown is complete
	winners []string

	// Last showdown result - persists across hands for UI review
	lastShowdownResult *ShowdownResult

	// State machine - Rob Pike's pattern
	sm *statemachine.Machine[Game]

	// Lifecycle notifications - states signal when they're reached
	preFlopReached chan struct{}

	// Event channel for sending events to Table (betting round complete, showdown, etc.)
	tableEventChan chan<- GameEvent

	// Player event channel (timebank expired, etc.)
	playerEventChan chan PlayerEvent
}

// SetTableEventChannel sets the channel for sending events to the Table.
// Also wires the BlindManager FSM so pending-increase notifications
// flow through the same channel.
func (g *Game) SetTableEventChannel(ch chan<- GameEvent) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.tableEventChan = ch
	if g.blindManager != nil {
		g.blindManager.SetGameEventChannel(ch)
	}
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

// currentBigBlind returns the active big blind amount. Uses the cached value
// from the BlindManager FSM (updated at hand boundaries in statePreDeal).
// Falls back to config value when blind increases are disabled.
// Requires: g.mu held.
func (g *Game) currentBigBlind() int64 {
	if g.liveBigBlind > 0 {
		return g.liveBigBlind
	}
	return g.config.BigBlind
}

// currentSmallBlind returns the active small blind amount. Uses the cached value
// from the BlindManager FSM (updated at hand boundaries in statePreDeal).
// Falls back to config value when blind increases are disabled.
// Requires: g.mu held.
func (g *Game) currentSmallBlind() int64 {
	if g.liveSmallBlind > 0 {
		return g.liveSmallBlind
	}
	return g.config.SmallBlind
}

// resetBettingRound resets per-street betting bookkeeping.
// Requires: g.mu held.
func (g *Game) resetBettingRound() {
	mustHeld(&g.mu)
	g.currentBet = 0
	g.lastAggressor = -1
	g.lastRaiseAmount = g.currentBigBlind()
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

	if cfg.BigBlind <= 0 {
		return nil, fmt.Errorf("poker: BigBlind must be greater than zero")
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
		deck:            newDeck(rng),
		communityCards:  nil,
		potManager:      NewPotManager(cfg.NumPlayers),
		currentBet:      0,
		round:           0,
		config:          cfg,
		log:             cfg.Log,
		phase:           pokerrpc.GamePhase_NEW_HAND_DEALING,
		lastRaiseAmount: cfg.BigBlind, // will be updated after blindManager is created
		lastAggressor:   -1,
	}

	// Initialize blind manager if blind increases are configured
	if cfg.BlindIncreaseInterval > 0 {
		schedule := cfg.BlindSchedule
		if schedule == nil {
			schedule = DefaultBlindSchedule
		}
		g.blindManager = NewBlindManager(schedule, cfg.BlindIncreaseInterval, cfg.Log)
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

func (g *Game) Start(ctx context.Context) {
	if g.blindManager != nil {
		g.blindManager.Start(ctx)
	}
	g.sm.Start(ctx)
}

// StartFromRestoredSnapshot starts the FSM in the correct state based on the
// restored phase. This allows the game to continue from where it was saved.
func (g *Game) StartFromRestoredSnapshot(ctx context.Context) {
	g.mu.Lock()
	phase := g.phase
	g.mu.Unlock()

	// Select the correct starting state based on the restored phase
	var initialState GameStateFn
	switch phase {
	case pokerrpc.GamePhase_NEW_HAND_DEALING:
		initialState = stateNewHandDealing
	case pokerrpc.GamePhase_PRE_FLOP:
		initialState = statePreFlop
	case pokerrpc.GamePhase_FLOP:
		initialState = stateFlop
	case pokerrpc.GamePhase_TURN:
		initialState = stateTurn
	case pokerrpc.GamePhase_RIVER:
		initialState = stateRiver
	case pokerrpc.GamePhase_SHOWDOWN:
		initialState = stateShowdown
	default:
		// Fallback to restored state for unknown phases
		initialState = stateRestored
	}

	g.mu.Lock()
	// Replace the state machine with the correct starting state
	g.sm = statemachine.New(g, initialState, 32)
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
	// before dealing to keep a single deck per hand)
	g.resetBettingRound()
	g.communityCards = nil
	g.winners = nil

	// Cancel any pending auto-start from the previous hand.
	g.cancelAutoStart()

	// Reset auto-advance state for new hand
	g.cancelAutoAdvance()
	g.autoAdvanceEnabled = false

	// Reset pot manager for the new hand BEFORE blinds are posted.
	g.potManager = NewPotManager(len(g.players))

	// Prepare a fresh deck for this hand BEFORE dealing hole cards so that
	// community cards come from the same deck for the entire hand.
	var nextRng *rand.Rand
	if g.config.Seed != 0 {
		// Derive a per-hand seed from the base seed and current round to keep
		// deterministic shuffles across hands for the same config.
		derived := g.config.Seed + int64(g.round)
		nextRng = rand.New(rand.NewSource(derived))
	} else {
		base := time.Now().UnixNano()
		var mix int64
		if g.deck != nil && g.deck.rng != nil {
			mix = g.deck.rng.Int63()
		}
		nextRng = rand.New(rand.NewSource(base ^ mix ^ int64(g.round)))
	}
	g.deck = newDeck(nextRng)

	g.phase = pokerrpc.GamePhase_PRE_FLOP

	// Initialize new Hand for this round (cards will be dealt in stateDeal next).
	// We always create a fresh Hand object per round so that hole cards and
	// board are isolated between hands.
	playerIDs := make([]string, 0, len(g.players))
	for _, p := range g.players {
		if p != nil {
			playerIDs = append(playerIDs, p.ID())
		}
	}
	g.currentHand = NewHand(playerIDs)
	g.log.Debugf("statePreDeal: initialized new hand %s with %d players", g.currentHand.id, len(playerIDs))

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

		// Reset reveal intent for all players at hand start
		for _, p := range g.players {
			if p != nil {
				p.mu.Lock()
				p.revealed = false
				p.mu.Unlock()
			}
		}

		// Start hand participation FSMs for this hand BEFORE posting blinds so
		// blind deductions flow through player state machines.
		for _, p := range g.players {
			if p != nil {
				if err := p.HandleStartHand(); err != nil {
					g.log.Errorf("Failed to start hand for player %s: %v", p.ID(), err)
				}
			}
		}

		// Apply any pending blind increase from the BlindManager FSM.
		// The timer fires between hands but the FSM defers the actual
		// mutation until we call Apply() here at the hand boundary.
		if g.blindManager != nil {
			if g.round == 1 {
				g.mu.Unlock()
				g.blindManager.SendStart(time.Time{})
				g.mu.Lock()
			}
			g.mu.Unlock()
			result := g.blindManager.Apply()
			info := g.blindManager.GetInfo()
			g.mu.Lock()
			g.liveSmallBlind = result.Level.SmallBlind
			g.liveBigBlind = result.Level.BigBlind
			g.liveBlindLevel = result.Index
			g.liveNextBlindMs = info.NextIncreaseUnixMs
			if result.Changed {
				g.log.Infof("statePreDeal: %s", result.Message)
				g.sendTableEvent(GameEvent{Type: GameEventBlindsIncreased})
			}
		}

		// Post blinds through player FSMs so short stacks become ALL_IN immediately
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
			if delta <= 0 {
				return
			}

			// Route blind payment through the player's hand FSM
			paid, err := p.DeductBlind(delta)
			if err != nil {
				g.log.Errorf("Failed to post blind for player %s: %v", p.ID(), err)
				return
			}

			// Reflect into pot manager and table-wide current bet
			g.potManager.addBet(pos, paid, g.players) // contract: g.mu held
			finalBet := already + paid
			if finalBet > g.currentBet {
				g.currentBet = finalBet
			}
		}

		postBlind(sbPos, g.currentSmallBlind())
		postBlind(bbPos, g.currentBigBlind())

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
			// If initial player is ALL_IN from posting blinds, skip to next active player
			if p.GetCurrentStateString() == ALL_IN_STATE {
				g.advanceToNextPlayer(time.Now())
			} else if g.shouldSkipTurnTimer() {
				// Auto-advance scenario (e.g. everyone else all-in)
				g.maybeCompleteBettingRound()
			} else {
				p.StartTurn()
			}
		}
	}
	// phase stays PRE_FLOP (already set in statePreDeal)
	return statePreFlop
}

// countActivePlayersLocked returns how many players are still actionable
// (i.e. in the hand and not all-in). g.mu MUST be held.
func (g *Game) countActivePlayers() int {
	mustHeld(&g.mu)

	active := 0
	for _, p := range g.players {
		if p == nil {
			continue
		}
		state := p.GetCurrentStateString()
		// Folded and all-in are not actionable
		if state == FOLDED_STATE || state == ALL_IN_STATE {
			continue
		}
		active++
	}
	return active
}

func statePreFlop(g *Game, in <-chan any) GameStateFn {
	g.mu.Lock()
	g.phase = pokerrpc.GamePhase_PRE_FLOP
	// Reset aggressor at the start of a new street
	g.lastAggressor = -1
	ch := g.preFlopReached // Read channel reference under lock
	g.mu.Unlock()

	// Evaluate immediate betting completion (e.g., blind all-ins) now that phase is set.
	g.mu.Lock()
	g.maybeCompleteBettingRound()
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
	g.resetBettingRound()
	g.phase = pokerrpc.GamePhase_FLOP

	// Snapshot auto-advance flag and count actionable players
	autoAdvance := g.autoAdvanceEnabled
	activePlayers := g.countActivePlayers()

	// Decide current player & whether to start a turn timer
	if activePlayers == 0 {
		// Nobody can act; don't assign a current player
		g.currentPlayer = -1
	} else if !autoAdvance || activePlayers >= 2 {
		// Normal betting round (no auto-advance, or multi-way street):
		// pick first actor and start their turn.
		g.currentPlayer = g.computeFirstActorIndex()
		if g.currentPlayer >= 0 {
			g.players[g.currentPlayer].StartTurn()
		}
	} else {
		// autoAdvance == true && activePlayers == 1:
		// streets will auto-advance; DO NOT start a turn timer.
		g.currentPlayer = -1
	}

	g.log.Debugf("stateFlop: entered, autoAdvanceEnabled=%v, activePlayers=%d",
		autoAdvance, activePlayers)
	g.mu.Unlock()

	// Emit betting round complete event AFTER dealing and state mutation
	// This allows clients to see the flop cards with the correct phase
	g.sendTableEvent(GameEvent{Type: GameEventBettingRoundComplete})

	// If auto-advance is enabled, schedule the next advance unless there are
	// 2+ active players. With 0 or 1 active player, we auto-advance streets.
	if autoAdvance && activePlayers <= 1 {
		g.log.Debugf("stateFlop: auto-advance enabled (active=%d), scheduling advance to TURN",
			activePlayers)
		g.ScheduleAutoAdvance()
	}

	// Wait for events...
	for ev := range in {
		if next, handled := handleGameEvent(g, ev); handled {
			if next != nil {
				return next
			}
			continue
		}
		switch ev.(type) {
		case evAdvance:
			return stateTurn
		default:
		}
	}
	return nil
}

func stateTurn(g *Game, in <-chan any) GameStateFn {
	g.mu.Lock()
	g.dealTurn()
	g.resetBettingRound()
	g.phase = pokerrpc.GamePhase_TURN

	autoAdvance := g.autoAdvanceEnabled
	activePlayers := g.countActivePlayers()

	if !autoAdvance || activePlayers >= 2 {
		g.currentPlayer = g.computeFirstActorIndex()
		if g.currentPlayer >= 0 {
			g.players[g.currentPlayer].StartTurn()
		}
	} else {
		g.currentPlayer = -1
	}

	g.log.Debugf("stateTurn: entered, autoAdvanceEnabled=%v, activePlayers=%d", autoAdvance, activePlayers)
	g.mu.Unlock()

	g.sendTableEvent(GameEvent{Type: GameEventBettingRoundComplete})

	if autoAdvance && activePlayers <= 1 {
		g.log.Debugf("stateTurn: auto-advance enabled (active=%d), scheduling advance to RIVER", activePlayers)
		g.ScheduleAutoAdvance()
	}

	for ev := range in {
		if next, handled := handleGameEvent(g, ev); handled {
			if next != nil {
				return next
			}
			continue
		}
		switch ev.(type) {
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
	g.resetBettingRound()
	g.phase = pokerrpc.GamePhase_RIVER

	// Check if auto-advance is enabled and count actionable players
	autoAdvance := g.autoAdvanceEnabled
	activePlayers := g.countActivePlayers()

	// Decide who (if anyone) is the current actor and whether to start a turn
	if activePlayers == 0 {
		// Nobody can act
		g.currentPlayer = -1
	} else if !autoAdvance || activePlayers >= 2 {
		// Normal betting round: pick first actor and start their turn
		g.currentPlayer = g.computeFirstActorIndex()
		if g.currentPlayer >= 0 {
			g.players[g.currentPlayer].StartTurn()
		}
	} else {
		// autoAdvance == true && activePlayers == 1:
		// Streets will auto-advance; DO NOT start a timed turn on the last active player.
		g.currentPlayer = -1
	}

	g.log.Debugf("stateRiver: entered, autoAdvanceEnabled=%v, activePlayers=%d",
		autoAdvance, activePlayers)
	g.mu.Unlock()

	// Emit betting round complete event AFTER dealing and state mutation
	// This allows clients to see the river card with the correct phase
	g.sendTableEvent(GameEvent{Type: GameEventBettingRoundComplete})

	// If auto-advance is enabled, schedule advance to showdown unless there are
	// 2+ active players. With 0 or 1 active player, we auto-advance streets.
	if autoAdvance {
		if activePlayers <= 1 {
			g.log.Debugf("stateRiver: auto-advance enabled (active=%d), scheduling advance to SHOWDOWN", activePlayers)

			// Disable auto-advance for showdown (it's the final state)
			g.mu.Lock()
			g.autoAdvanceEnabled = false
			g.mu.Unlock()

			// Schedule the advance to showdown
			g.ScheduleAutoAdvance()
		} else {
			g.log.Debugf("stateRiver: auto-advance enabled but %d active players; waiting for action", activePlayers)
		}
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
		if pl.GetCurrentStateString() != FOLDED_STATE {
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

		// Capture hand identification
		if g.currentHand != nil {
			result.HandID = g.currentHand.id
		}
		result.Round = g.round

		// Capture per-player final states BEFORE EndHandParticipation clears them
		winnerInfoByID := make(map[string]*pokerrpc.Winner, len(result.WinnerInfo))
		for _, w := range result.WinnerInfo {
			winnerInfoByID[w.PlayerId] = w
		}
		showdownContested := false
		activeAtShowdown := 0
		for _, p := range g.players {
			if p == nil {
				continue
			}
			if p.GetCurrentStateString() == FOLDED_STATE {
				continue
			}
			activeAtShowdown++
			if activeAtShowdown > 1 {
				showdownContested = true
				break
			}
		}
		forcedReveals := make([]AutoRevealPlayer, 0, len(result.WinnerInfo))
		if showdownContested {
			for _, w := range result.WinnerInfo {
				if len(w.BestHand) == 0 {
					continue
				}
				cards, err := g.forceRevealShowdownWinner(w.PlayerId)
				if err != nil {
					g.log.Warnf("stateShowdown: failed to reveal showdown winner %s: %v", w.PlayerId, err)
					continue
				}
				if len(cards) == 0 {
					continue
				}
				forcedReveals = append(forcedReveals, AutoRevealPlayer{
					PlayerID: w.PlayerId,
					Cards:    cards,
				})
			}
		}

		result.Players = make([]ShowdownPlayerInfo, 0, len(g.players))
		for idx, p := range g.players {
			if p == nil {
				continue
			}
			winfo := winnerInfoByID[p.ID()]
			info := ShowdownPlayerInfo{
				ID:           p.ID(),
				Name:         p.Name(),
				FinalState:   p.GetCurrentStateString(),
				Contribution: g.potManager.getTotalBet(idx),
			}
			// Reveal winners that reached showdown (have a hand value) or any player who toggled reveal.
			if g.currentHand != nil && (p.revealed || (winfo != nil && len(winfo.BestHand) > 0)) {
				cards := g.currentHand.GetPlayerCards(p.ID())
				for _, c := range cards {
					info.HoleCards = append(info.HoleCards, toProtoCard(c))
				}
			}
			// Find this player's winning info if they won
			if winfo != nil {
				info.HandRank = winfo.HandRank
				info.BestHand = winfo.BestHand
			}
			result.Players = append(result.Players, info)
		}

		// Store for later retrieval (persists across hands)
		g.lastShowdownResult = result

		// Send the showdown result to the table
		g.sendTableEvent(GameEvent{Type: GameEventShowdownComplete, ShowdownResult: result})
		if len(forcedReveals) > 0 {
			g.sendTableEvent(GameEvent{Type: GameEventAutoShowCards, RevealInfo: forcedReveals})
		}
	}

	// Snapshot players slice for post-settlement notifications.
	players := make([]*Player, len(g.players))
	copy(players, g.players)

	// Check for players with 0 chips and send GameEventPlayerLost events
	// This must happen after showdown so winnings are distributed
	for _, p := range g.players {
		if p != nil {
			p.mu.RLock()
			balance := p.balance
			id := p.id
			p.mu.RUnlock()
			if balance <= 0 {
				g.log.Debugf("stateShowdown: player %s has 0 chips, sending GameEventPlayerLost", id)
				g.sendTableEvent(GameEvent{Type: GameEventPlayerLost, PlayerID: id})
			}
		}
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

	// Mark phase as SHOWDOWN now that settlement is complete.
	g.phase = pokerrpc.GamePhase_SHOWDOWN
	g.mu.Unlock()

	// Send game over event if game has ended
	if gameOver {
		g.log.Debugf("stateShowdown: sending GameEventGameOver")
		g.sendTableEvent(GameEvent{Type: GameEventGameOver, WinnerID: lastPlayerID})
	}

	// Regardless of game over or continuation, signal all players that the
	// hand has ended so their per-hand FSMs can reset internal flags. This
	// is done outside of g.mu to avoid lock-ordering issues.
	for _, p := range players {
		if p != nil {
			p.EndHandParticipation()
		}
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
			p.mu.RLock()
			isSmallBlind := p.isSmallBlind
			isBigBlind := p.isBigBlind
			p.mu.RUnlock()
			if isSmallBlind {
				forced[i] = g.currentSmallBlind()
			} else if isBigBlind {
				forced[i] = g.currentBigBlind()
			}
		}
	}

	hiPlayer, refunded, err := g.potManager.returnUncalledBet(forced)
	if err != nil {
		return fmt.Errorf("failed to refund uncalled bets: %w", err)
	}

	// Rebuild pots after adjusting bets to reflect the refunded amounts
	g.potManager.RebuildPotsIncremental(g.players)

	// Credit the refunded amount via the Player FSM.
	if hiPlayer >= 0 && refunded > 0 {
		if hiPlayer >= len(g.players) || g.players[hiPlayer] == nil {
			return fmt.Errorf("invalid refund target %d", hiPlayer)
		}
		if err := g.players[hiPlayer].AddToBalance(refunded); err != nil {
			return fmt.Errorf("refund uncalled bet: adjust balance: %w", err)
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
					if cp.GetCurrentStateString() == IN_GAME_STATE {
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

// GetLastShowdownResult returns the result of the most recently completed hand.
// This persists across hands, allowing players to review what happened.
// Returns nil if no hand has completed yet.
func (g *Game) GetLastShowdownResult() *ShowdownResult {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.lastShowdownResult
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

	return nil
}

// Close stops all player state machines and cleans up resources.
// This must be called when a game is no longer needed to prevent goroutine leaks.
func (g *Game) Close() {
	// Mark shutdown early to prevent new timers/events from scheduling
	g.shuttingDown.Store(true)

	// Grab references while holding lock and cancel timers
	g.mu.Lock()
	players := make([]*Player, len(g.players))
	copy(players, g.players)
	sm := g.sm
	playerCh := g.playerEventChan
	// Cancel timers under lock so callbacks bail before FSM is stopped
	g.cancelAutoAdvance()
	g.cancelAutoStart()
	// Cancel auto-advance under lock so the timer callback sees the canceled flag
	// before we stop the state machine (prevents send on closed inbox).
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

	// Stop the blind manager FSM first (it may send events to the game)
	if g.blindManager != nil {
		g.blindManager.Stop()
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
}

// SetPlayers sets the players for this game from table users
func (g *Game) SetPlayers(users []*User) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Convert users to players for game management using proper constructor
	g.players = make([]*Player, len(users))
	for i, user := range users {
		// Copy table-level state from user (acquire user lock to read fields)
		user.mu.RLock()
		userName := user.Name
		tableSeat := user.TableSeat
		isReady := user.IsReady
		user.mu.RUnlock()

		// Create player using constructor to ensure state machine is initialized
		player := NewPlayer(user.ID, userName, g.config.StartingChips)

		player.mu.Lock()
		player.tableSeat = tableSeat
		player.isReady = isReady
		player.lastAction = time.Now() // Set current time since User doesn't have LastAction
		// Wire timebank scheduling
		player.timebankDelay = g.config.TimeBank
		player.playerEventChan = g.playerEventChan
		player.mu.Unlock()

		g.players[i] = player
	}
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
		if p != nil && p.GetCurrentStateString() != FOLDED_STATE {
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

	// If only one player remains, send showdown event for the current round.
	if g.unfoldsPlayers() == 1 {
		// End the player's turn before showdown
		p.EndTurn()
		// Notify the game FSM to enter stateShowdown for this round.
		if g.sm != nil {
			g.sm.Send(evGotoShowdownReq{round: g.round})
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
		// Player has no chips to contribute - they're already all-in from a previous action
		// (e.g., posting blinds). Treat as a no-op: end their turn and advance.
		g.log.Debugf("Player %s has 0 balance and is already all-in, advancing turn", player.ID())
		player.EndTurn()
		g.advanceToNextPlayer(time.Now())
		g.maybeCompleteBettingRound()
		return nil
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
		if state != FOLDED_STATE && state != ALL_IN_STATE {
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

	minRaise := g.lastRaiseAmount
	isShortAllIn := delta > 0 && delta == balance

	if tableBet == 0 {
		if amount < minRaise && !isShortAllIn {
			return fmt.Errorf("minimum bet is %d", minRaise)
		}
	} else if amount > tableBet {
		raiseSize := amount - tableBet
		if raiseSize < minRaise && !isShortAllIn {
			return fmt.Errorf("minimum raise is %d (current bet: %d, attempted: %d)", minRaise, tableBet, amount)
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

	if amount > g.currentBet {
		prevTableBet := g.currentBet
		g.currentBet = amount

		raiseSize := amount - prevTableBet
		isOpeningBet := prevTableBet == 0
		isFullRaise := !isOpeningBet && raiseSize >= minRaise

		if g.currentPlayer >= 0 && g.currentPlayer < len(g.players) {
			if isOpeningBet || isFullRaise {
				g.lastAggressor = g.currentPlayer
			}
		}

		if isOpeningBet {
			if !isShortAllIn {
				g.lastRaiseAmount = amount
			}
		} else if isFullRaise {
			g.lastRaiseAmount = raiseSize
		}
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

// advanceToNextPlayer moves to the next active player.
// Note: The caller (action handlers like bet, call, check, fold) is responsible
// for ending the current player's turn before calling this function.
//
// This function checks shouldSkipTurnTimer() before starting a turn timer.
// If auto-advance conditions are met (e.g., all other players are all-in),
// we update currentPlayer but do NOT start a timer—maybeCompleteBettingRound
// will handle scheduling auto-advance instead.
func (g *Game) advanceToNextPlayer(now time.Time) {
	mustHeld(&g.mu)
	if len(g.players) == 0 {
		return
	}

	maxPlayers := len(g.players)
	foundNext := false

	// Find the next IN_GAME player
	for i := 0; i < maxPlayers; i++ {
		g.currentPlayer = (g.currentPlayer + 1) % maxPlayers

		if g.players[g.currentPlayer].GetCurrentStateString() == IN_GAME_STATE {
			foundNext = true
			break
		}
		// Skip FOLDED, ALL_IN, AT_TABLE, LEFT, etc.
	}

	if !foundNext {
		// No actionable player found
		return
	}

	// Check if we should skip starting the turn timer (auto-advance scenario)
	if g.shouldSkipTurnTimer() {
		// Don't start timer - maybeCompleteBettingRound will handle auto-advance
		return
	}

	// Normal case: start the player's turn timer
	g.players[g.currentPlayer].StartTurn()
}

// ShowdownPlayerInfo captures each player's state at showdown
type ShowdownPlayerInfo struct {
	ID           string            // Player ID
	Name         string            // Player nickname (for showdown display)
	HoleCards    []*pokerrpc.Card  // Revealed hole cards
	FinalState   string            // "PLAYER_STATE_FOLDED", "PLAYER_STATE_ALL_IN", "PLAYER_STATE_IN_GAME"
	Contribution int64             // Total contributed to pot this hand
	HandRank     pokerrpc.HandRank // Best hand rank (if not folded)
	BestHand     []*pokerrpc.Card  // Best 5 cards (if not folded)
}

// ShowdownResult contains the results of a showdown for table notifications
type ShowdownResult struct {
	// Existing fields
	Winners    []string
	WinnerInfo []*pokerrpc.Winner
	TotalPot   int64
	Board      []*pokerrpc.Card

	// Hand identification
	HandID string // Unique hand identifier
	Round  int    // Round number

	// Per-player final states and cards
	Players []ShowdownPlayerInfo
}

// AutoRevealPlayer captures cards revealed automatically by the game FSM.
type AutoRevealPlayer struct {
	PlayerID string
	Cards    []*pokerrpc.Card
}

// forceRevealShowdownWinner reveals an obligated showdown winner via the player FSM.
// Requires: g.mu held.
func (g *Game) forceRevealShowdownWinner(playerID string) ([]*pokerrpc.Card, error) {
	mustHeld(&g.mu)

	if g.currentHand == nil {
		return nil, fmt.Errorf("no active hand")
	}
	player := g.getPlayerByID(playerID)
	if player == nil {
		return nil, fmt.Errorf("player %s not found", playerID)
	}

	player.mu.RLock()
	if player.revealed {
		player.mu.RUnlock()
		return nil, nil
	}
	hp := player.handParticipation
	player.mu.RUnlock()
	if hp == nil {
		return nil, fmt.Errorf("player %s hand participation not started", playerID)
	}

	reply := make(chan error, 1)
	hp.Send(evRevealCards{Reply: reply})
	if err := <-reply; err != nil {
		return nil, err
	}

	cards := g.currentHand.GetPlayerCards(playerID)
	if len(cards) == 0 {
		return nil, nil
	}
	revealed := make([]*pokerrpc.Card, 0, len(cards))
	for _, c := range cards {
		revealed = append(revealed, toProtoCard(c))
	}
	return revealed, nil
}

// toProtoCard converts an internal Card to a pokerrpc.Card
func toProtoCard(c Card) *pokerrpc.Card {
	return &pokerrpc.Card{
		Suit:  c.GetSuit(),
		Value: c.GetValue(),
	}
}

// HandleShowdown processes the showdown logic and returns results (external API)
func (g *Game) HandleShowdown() (*ShowdownResult, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	// Public API now delegates to the internal locked implementation.
	return g.handleShowdown()
}

func (g *Game) handleShowdown() (*ShowdownResult, error) {
	mustHeld(&g.mu)

	type snap struct {
		id        string
		balance   int64
		folded    bool
		revealed  bool
		handValue *HandValue
		// optional: stateStr for logs only
	}

	// Use a stable order for deterministic results (and predictable lock acquisition).
	players := make([]*Player, 0, len(g.players))
	for _, p := range g.players {
		if p != nil {
			players = append(players, p)
		}
	}
	sort.Slice(players, func(i, j int) bool { return players[i].id < players[j].id })

	idxByID := make(map[string]int, len(g.players))
	for i, p := range g.players {
		if p != nil {
			idxByID[p.id] = i
		}
	}

	// --- Normalize pots (refund + rebuild) BEFORE taking prev balances.
	if err := g.refundUncalledBets(); err != nil {
		return nil, err
	}
	g.potManager.RebuildPotsIncremental(g.players)
	postRefundPot := g.potManager.getTotalPot()

	// Snapshot hole cards once (no player locks needed; g.mu is held).
	// Owners can always see their own cards, so pass true for revealed
	holeCards := make(map[string][]Card, len(players))
	if g.currentHand == nil {
		return nil, fmt.Errorf("invalid showdown: nil currentHand")
	}
	for _, p := range players {
		holeCards[p.id] = g.currentHand.GetPlayerCards(p.id)
	}

	// --- PASS 1: snapshot player state/balances once
	snaps := make(map[string]snap, len(players))
	for _, p := range players {
		// choose ONE fold source of truth; if you trust p.hasFolded, use it.
		// If getCurrentState() is canonical, derive folded from that and ignore p.hasFolded.
		p.mu.RLock()
		state := p.getCurrentState()
		snaps[p.id] = snap{
			id:       p.id,
			balance:  p.balance,
			folded:   state.String() == FOLDED_STATE,
			revealed: p.revealed,
		}
		p.mu.RUnlock()
	}

	// Build list of unfolded from snapshots (no extra locks)
	unfolded := make([]*Player, 0, len(players))
	for _, p := range players {
		if !snaps[p.id].folded {
			unfolded = append(unfolded, p)
		}
	}
	if len(unfolded) == 0 {
		return nil, fmt.Errorf("invalid showdown: no active players")
	}

	// Ensure board is complete only for multi-way (>=2 unfolded)
	if len(unfolded) >= 2 && len(g.communityCards) < 5 {
		dealOne := func() (Card, bool) { return g.deck.Draw() }
		switch g.phase {
		case pokerrpc.GamePhase_PRE_FLOP:
			for i := 0; i < 3; i++ {
				c, ok := dealOne()
				if !ok {
					return nil, fmt.Errorf("deck underflow on flop")
				}
				g.communityCards = append(g.communityCards, c)
			}
			g.phase = pokerrpc.GamePhase_FLOP
			fallthrough
		case pokerrpc.GamePhase_FLOP:
			if len(g.communityCards) < 4 {
				c, ok := dealOne()
				if !ok {
					return nil, fmt.Errorf("deck underflow on turn")
				}
				g.communityCards = append(g.communityCards, c)
			}
			g.phase = pokerrpc.GamePhase_TURN
			fallthrough
		case pokerrpc.GamePhase_TURN:
			if len(g.communityCards) < 5 {
				c, ok := dealOne()
				if !ok {
					return nil, fmt.Errorf("deck underflow on river")
				}
				g.communityCards = append(g.communityCards, c)
			}
			g.phase = pokerrpc.GamePhase_RIVER
		case pokerrpc.GamePhase_RIVER:
		default:
			return nil, fmt.Errorf("invalid showdown: unexpected phase %s", g.phase)
		}
	}

	// Build result now that board is final.
	result := &ShowdownResult{
		Winners:    make([]string, 0, len(players)),
		WinnerInfo: make([]*pokerrpc.Winner, 0, len(players)),
		Board:      make([]*pokerrpc.Card, 0, len(g.communityCards)),
	}
	for _, c := range g.communityCards {
		result.Board = append(result.Board, &pokerrpc.Card{Suit: c.GetSuit(), Value: c.GetValue()})
	}

	// --- Evaluate hands without locks (multi-way only)
	if len(unfolded) >= 2 {
		for _, p := range unfolded {
			hole := holeCards[p.id]
			if len(hole)+len(g.communityCards) < 5 {
				return nil, fmt.Errorf("invalid showdown: player %s has insufficient cards (hole=%d, board=%d)",
					p.id, len(hole), len(g.communityCards))
			}
			hv, err := EvaluateHand(hole, g.communityCards)
			if err != nil {
				return nil, fmt.Errorf("evaluate hand (multi-way) for %s: %w", p.id, err)
			}
			s := snaps[p.id]
			s.handValue = &hv
			snaps[p.id] = s
		}

		// --- PASS 2 (multi-way): publish handValue/handDescription via Player FSM
		for _, p := range unfolded {
			if hv := snaps[p.id].handValue; hv != nil {
				if err := p.SetShowdownHand(hv, GetHandDescription(*hv)); err != nil {
					return nil, fmt.Errorf("set showdown hand for %s: %w", p.id, err)
				}
			}
		}
	}

	// Compute pot payouts and settle pots (no player mutation here).
	payouts, err := g.potManager.distributePots(g.players)
	if err != nil {
		return nil, err
	}

	// Apply payouts via Player FSM.
	var totalWinnings int64
	for idx, amt := range payouts {
		if amt <= 0 {
			continue
		}
		if idx < 0 || idx >= len(g.players) || g.players[idx] == nil {
			return nil, fmt.Errorf("invalid payout target %d", idx)
		}
		if err := g.players[idx].AddToBalance(amt); err != nil {
			return nil, fmt.Errorf("apply payout to %s: %w", g.players[idx].id, err)
		}
		totalWinnings += amt
	}

	// Build winners from payouts (no locks)
	for _, p := range players {
		s := snaps[p.id]
		if s.folded {
			continue
		}
		idx, ok := idxByID[p.id]
		if !ok {
			continue
		}
		winnings := payouts[idx]
		if winnings <= 0 {
			continue
		}

		var handRank pokerrpc.HandRank
		var best []Card

		if s.handValue != nil {
			handRank = s.handValue.HandRank
			best = s.handValue.BestHand
		}

		result.Winners = append(result.Winners, p.id)
		result.WinnerInfo = append(result.WinnerInfo, &pokerrpc.Winner{
			PlayerId: p.id,
			HandRank: handRank,
			BestHand: CreateHandFromCards(best),
			Winnings: winnings,
		})
	}

	if totalWinnings != postRefundPot {
		g.log.Warnf("showdown invariant violated: postRefundPot=%d, distributed=%d", postRefundPot, totalWinnings)
		postRefundPot = totalWinnings
	}
	result.TotalPot = postRefundPot
	return result, nil
}

// shouldSkipTurnTimer returns true if auto-advance conditions are met and we should
// NOT start a player's turn timer. This happens when:
// - All remaining players are all-in/folded (active == 0), OR
// - Only one can act but all others are all-in with matched bets
//
// This is a pure check function that does NOT mutate any state.
// Requires: g.mu held.
func (g *Game) shouldSkipTurnTimer() bool {
	mustHeld(&g.mu)

	// Find the effective bet and count player states
	maxAllInBet := int64(0)
	alive, active, unmatched := 0, 0, 0

	for _, p := range g.players {
		if p == nil {
			continue
		}
		state := p.GetCurrentStateString()

		// Skip folded entirely
		if state == FOLDED_STATE {
			continue
		}

		alive++

		if state == ALL_IN_STATE {
			p.mu.RLock()
			allInBet := p.currentBet
			p.mu.RUnlock()
			if allInBet > maxAllInBet {
				maxAllInBet = allInBet
			}
			continue
		}

		active++
	}

	// Calculate effective bet
	effectiveBet := g.currentBet
	if maxAllInBet > 0 && maxAllInBet < effectiveBet {
		effectiveBet = maxAllInBet
	}

	// Count unmatched active players
	for _, p := range g.players {
		if p == nil {
			continue
		}
		state := p.GetCurrentStateString()
		if state == FOLDED_STATE || state == ALL_IN_STATE {
			continue
		}
		p.mu.RLock()
		cb := p.currentBet
		p.mu.RUnlock()
		if cb < effectiveBet {
			unmatched++
		}
	}

	allInCount := alive - active

	// Skip turn timer if:
	// - No active players (all are all-in or folded), OR
	// - Only one active player but others are all-in with matched bets
	return active == 0 || (active == 1 && allInCount > 0 && unmatched == 0)
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
	// First, find the effective bet: when a short-stack all-in occurs, the effective bet
	// is the maximum of all all-in amounts, not the table currentBet.
	maxAllInBet := int64(0)
	alive, active, unmatched := 0, 0, 0
	for _, p := range g.players {
		if p == nil {
			continue
		}
		state := p.GetCurrentStateString()

		// Skip folded entirely.
		if state == FOLDED_STATE {
			continue
		}

		// Player is alive.
		alive++

		// ALL_IN counts as matched but is not actionable.
		if state == ALL_IN_STATE {
			p.mu.RLock()
			allInBet := p.currentBet
			p.mu.RUnlock()
			if allInBet > maxAllInBet {
				maxAllInBet = allInBet
			}
			continue
		}

		// Count active players
		active++
	}

	// Calculate effective bet: when someone goes all-in for less than the table currentBet,
	// the effective bet becomes the all-in amount (other players can't bet more than that).
	effectiveBet := g.currentBet
	if maxAllInBet > 0 && maxAllInBet < effectiveBet {
		effectiveBet = maxAllInBet
	}

	// Now check if active players are matched to the effective bet
	for _, p := range g.players {
		if p == nil {
			continue
		}
		state := p.GetCurrentStateString()
		if state == FOLDED_STATE || state == ALL_IN_STATE {
			continue
		}
		// Actionable player: must have at least the effective bet
		p.mu.RLock()
		cb := p.currentBet
		p.mu.RUnlock()
		if cb < effectiveBet {
			unmatched++
		}
	}

	// 2) Terminal: only one alive -> uncontested; go to showdown.
	if alive <= 1 {
		if g.sm != nil {
			g.sm.Send(evGotoShowdownReq{round: g.round})
		}
		return
	}

	// 3) All alive are ALL_IN -> enable auto-advance and let FSM progress streets.
	// Also enable if only one player is active but all others are all-in (no one left to bet against)
	// AND all bets are matched (no unmatched bets).
	allInCount := alive - active
	g.log.Debugf("maybeCompleteBettingRound: alive=%d, active=%d, allInCount=%d, unmatched=%d, effectiveBet=%d, tableCurrentBet=%d, maxAllInBet=%d", alive, active, allInCount, unmatched, effectiveBet, g.currentBet, maxAllInBet)
	if active == 0 || (active == 1 && allInCount > 0 && unmatched == 0) {
		// Enable auto-advance mode so state handlers know to schedule timers
		alreadyAutoAdvance := g.autoAdvanceEnabled
		g.autoAdvanceEnabled = true
		if active == 0 {
			g.log.Debugf("maybeCompleteBettingRound: all %d players all-in, enabling auto-advance mode", alive)
		} else {
			g.log.Debugf("maybeCompleteBettingRound: %d player(s) active but %d all-in (no one to bet against), enabling auto-advance mode", active, allInCount)
		}

		// Auto-reveal hole cards once when entering auto-advance so everyone can see them.
		if !alreadyAutoAdvance && g.currentHand != nil {
			playerIDs := make([]string, 0, len(g.players))
			for _, p := range g.players {
				if p == nil {
					continue
				}
				if p.GetCurrentStateString() == FOLDED_STATE {
					continue
				}
				playerIDs = append(playerIDs, p.id)
			}
			if len(playerIDs) > 0 && g.sm != nil {
				g.sm.Send(evAutoRevealReq{playerIDs: playerIDs})
			}
		}

		// Schedule auto-advance with configured delay (gives UI time to show countdown)
		g.scheduleAutoAdvance()
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

	// 6) Once all actionable bets are matched, the street closes when action
	// returns to the last full aggressor, or to the street starter if everyone checked.
	if !g.actionReturnedToCloser(effectiveBet) {
		return
	}

	// 7) Street is complete -> signal FSM to advance exactly one street.
	// Dealing and phase updates occur in state handlers.
	g.sm.Send(evAdvance{})

	// 8) Prepare next betting round.
	// Clear turn flags and zero player currentBet (since table currentBet was
	// zeroed above). The next state's entry will initialize the first actor for
	// the new street based on the updated phase.
	for _, p := range g.players {
		if p != nil {
			p.EndTurn()
			p.mu.Lock()
			p.currentBet = 0 // Reset for new street (pot manager tracks cumulative)
			p.mu.Unlock()
		}
	}
	// Reset aggressor marker eagerly; state entry also resets this defensively.
	g.lastAggressor = -1
}

// nextActionablePlayerAfter returns the next IN_GAME player after idx, wrapping
// around the table. Returns -1 if none are actionable. Requires: g.mu held.
func (g *Game) nextActionablePlayerAfter(idx int) int {
	mustHeld(&g.mu)

	n := len(g.players)
	if n < 2 {
		return -1
	}

	for step := 1; step < n; step++ {
		next := (idx + step) % n
		if p := g.players[next]; p != nil && p.GetCurrentStateString() == IN_GAME_STATE {
			return next
		}
	}
	return -1
}

// actionReturnedToCloser reports whether betting has come back to the player
// who would close the street: the last full aggressor when there was a bet or
// raise, otherwise the street starter after a round of checks.
//
// A short all-in underraise does not become the closer. If it forces the last
// aggressor to respond and action has already advanced past them again, allow
// the street to close once their bet is matched.
//
// Requires: g.mu held.
func (g *Game) actionReturnedToCloser(effectiveBet int64) bool {
	mustHeld(&g.mu)

	if g.lastAggressor < 0 {
		starter := g.computeStreetStarterIndex()
		return starter >= 0 && g.currentPlayer == starter
	}

	if g.lastAggressor >= len(g.players) {
		return true
	}

	aggressor := g.players[g.lastAggressor]
	if aggressor == nil || aggressor.GetCurrentStateString() != IN_GAME_STATE {
		return true
	}

	if g.currentPlayer == g.lastAggressor {
		return true
	}

	aggressor.mu.RLock()
	aggressorBet := aggressor.currentBet
	aggressor.mu.RUnlock()

	nextAfterAggressor := g.nextActionablePlayerAfter(g.lastAggressor)
	return g.currentPlayer == nextAfterAggressor && aggressorBet >= effectiveBet
}

// computeStreetStarterIndexLocked returns the index of the first actionable
// player for the current street (skips folded and all-in players).
// Requires: g.mu held.
func (g *Game) computeStreetStarterIndex() int {
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
		if st != FOLDED_STATE && st != ALL_IN_STATE {
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

// computeFirstActorIndex returns the index of the first player who should act
// on the current street, or -1 if no player can act.
// This is a pure function that does NOT mutate any state.
// Requires: g.mu held.
func (g *Game) computeFirstActorIndex() int {
	mustHeld(&g.mu)

	if len(g.players) == 0 {
		return -1
	}

	numPlayers := len(g.players)
	var startIdx int

	if g.phase == pokerrpc.GamePhase_PRE_FLOP {
		if numPlayers == 2 {
			startIdx = g.dealer
		} else {
			startIdx = (g.dealer + 3) % numPlayers
		}
	} else {
		startIdx = (g.dealer + 1) % numPlayers
	}

	// Find first IN_GAME player starting from startIdx
	idx := startIdx
	for i := 0; i < numPlayers; i++ {
		if idx < 0 || idx >= numPlayers {
			idx = 0
		}
		if g.players[idx].GetCurrentStateString() == IN_GAME_STATE {
			return idx
		}
		idx = (idx + 1) % numPlayers
	}

	return -1
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
func (g *Game) SetGameState(dealer, round int, currentBet, pot, lastRaiseAmount int64, phase pokerrpc.GamePhase) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.dealer = dealer
	g.round = round
	g.currentBet = currentBet
	if lastRaiseAmount > 0 {
		g.lastRaiseAmount = lastRaiseAmount
	} else {
		g.lastRaiseAmount = g.currentBigBlind()
	}
	g.phase = phase
	// Note: Pot will be restored through the potManager when restoring player bets
	// We can't directly set the pot value, but it will be calculated from player bets
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
		g.currentPlayer = g.computeFirstActorIndex() // uses phase/dealer and skips folded/all-in/disconnected
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
		if p != nil && p.GetCurrentStateString() != FOLDED_STATE {
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
	if g.shuttingDown.Load() {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.shuttingDown.Load() {
		return
	}
	g.scheduleAutoStart()
}

// scheduleAutoStart is the internal implementation
func (g *Game) scheduleAutoStart() {
	mustHeld(&g.mu)
	if g.shuttingDown.Load() {
		return
	}
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
		if g.shuttingDown.Load() || g.autoStartCanceled.Load() {
			return
		}
		g.mu.Lock()
		if g.shuttingDown.Load() || g.autoStartCanceled.Load() {
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

// ScheduleAutoAdvance schedules automatic advance to next street when all players are all-in.
// This function acquires g.mu internally - safe to call without holding the lock.
func (g *Game) ScheduleAutoAdvance() {
	if g.shuttingDown.Load() {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.shuttingDown.Load() {
		return
	}
	g.scheduleAutoAdvance()
}

// cancelAutoAdvance cancels the auto-advance timer.
// Requires: g.mu held
func (g *Game) cancelAutoAdvance() {
	mustHeld(&g.mu)

	g.autoAdvanceCanceled.Store(true)
	if t := g.autoAdvanceTimer; t != nil {
		_ = t.Stop()
		g.autoAdvanceTimer = nil
	}
}

// scheduleAutoAdvance is the core implementation that schedules auto-advance.
// Requires: g.mu held
func (g *Game) scheduleAutoAdvance() {
	mustHeld(&g.mu)
	if g.shuttingDown.Load() {
		return
	}

	// Cancel any existing auto-advance timer
	g.cancelAutoAdvance()

	// If auto-advance is disabled (negative delay), don't schedule
	if g.config.AutoAdvanceDelay < 0 {
		g.log.Debugf("scheduleAutoAdvance: auto-advance disabled, delay=%v", g.config.AutoAdvanceDelay)
		return
	}

	g.log.Debugf("scheduleAutoAdvance: setting up timer with delay %v", g.config.AutoAdvanceDelay)

	// Mark that auto-advance is pending
	g.autoAdvanceCanceled.Store(false)

	// Schedule the auto-advance timer
	g.autoAdvanceTimer = time.AfterFunc(g.config.AutoAdvanceDelay, func() {
		// Fast path: bail without locking if canceled
		if g.autoAdvanceCanceled.Load() {
			g.log.Debugf("Auto-advance timer was canceled, not sending evAdvance")
			return
		}

		// Double-check under lock, and clear pointer
		g.mu.Lock()
		if g.autoAdvanceCanceled.Load() {
			g.mu.Unlock()
			g.log.Debugf("Auto-advance canceled (post-lock), not sending evAdvance")
			return
		}
		localSM := g.sm
		g.autoAdvanceTimer = nil
		g.mu.Unlock()

		if localSM == nil {
			g.log.Debugf("Auto-advance fired but FSM is nil, skipping")
			return
		}

		g.log.Debugf("Auto-advance timer fired, sending evAdvance to FSM")
		localSM.Send(evAdvance{})
	})
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
	Folded          bool
	IsAllIn         bool
	Hand            []Card
	CardsRevealed   bool
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
	Dealer          int
	CurrentBet      int64
	LastRaiseAmount int64
	Pot             int64
	Round           int
	BetRound        int
	Phase           pokerrpc.GamePhase
	CommunityCards  []Card
	DeckState       *DeckState
	Players         []PlayerSnapshot
	CurrentPlayer   string
	Winners         []PlayerSnapshot

	SmallBlind              int64 // Current small blind (may differ from config if blind increases are active)
	BigBlind                int64 // Current big blind (may differ from config if blind increases are active)
	BlindLevel              int   // Current blind level index (0-based)
	NextBlindIncreaseUnixMs int64 // Unix ms timestamp of next blind increase (0 if disabled/max)
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
		// Use getCurrentState() directly since we already hold the lock (avoids nested RLock)
		stateStr := player.getCurrentState().String()
		playerCopy := PlayerSnapshot{
			ID:              player.id,
			Name:            player.name,
			TableSeat:       player.tableSeat,
			IsReady:         player.isReady,
			Folded:          player.hasFolded,
			IsAllIn:         player.isAllIn,
			Balance:         player.balance,
			StartingBalance: player.startingBalance,
			CurrentBet:      player.currentBet,
			IsDealer:        player.isDealer,
			IsSmallBlind:    player.isSmallBlind,
			IsBigBlind:      player.isBigBlind,
			IsTurn:          player.isTurn,
			IsDisconnected:  player.isDisconnected,
			Hand:            nil, // Will be populated from currentHand below
			CardsRevealed:   player.revealed,
			HandDescription: player.handDescription,
			HandValue:       player.handValue,
			LastAction:      player.lastAction,
			StateString:     stateStr,
		}
		player.mu.RUnlock()

		// Retrieve hole cards from g.currentHand if it exists
		// Note: Snapshot includes all cards; visibility filtering happens at server level
		if g.currentHand != nil {
			// Get revealed state first

			// Get player's cards from the current hand, respecting reveal state
			// Owner can always see their own cards, but we pass revealed state for consistency
			cards := g.currentHand.GetPlayerCards(playerCopy.ID)
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
	// During showdown, use getTotalPot() after pots have been built
	potAmount := g.potManager.getTotalPot()

	// If we're in SHOWDOWN and per-hand flags have been cleared, hydrate folded/all-in
	// from the persisted lastShowdownResult so observers (UI/tests) can still see
	// who folded or was all-in for the completed hand.
	if g.phase == pokerrpc.GamePhase_SHOWDOWN && g.lastShowdownResult != nil {
		byID := make(map[string]ShowdownPlayerInfo, len(g.lastShowdownResult.Players))
		for _, pi := range g.lastShowdownResult.Players {
			byID[pi.ID] = pi
		}
		for i := range playersCopy {
			id := playersCopy[i].ID
			if id == "" {
				continue
			}
			if pi, ok := byID[id]; ok {
				playersCopy[i].Folded = (pi.FinalState == FOLDED_STATE)
				playersCopy[i].IsAllIn = (pi.FinalState == ALL_IN_STATE)
			}
		}
	}

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
						// Owner can always see their own cards
						cards := g.currentHand.GetPlayerCards(player.id)
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
		Dealer:                  g.dealer,
		CurrentBet:              g.currentBet,
		LastRaiseAmount:         g.lastRaiseAmount,
		Pot:                     potAmount,
		Round:                   g.round,
		BetRound:                br,
		Phase:                   g.phase,
		CommunityCards:          communityCardsCopy,
		DeckState:               g.deck.GetState(),
		Players:                 playersCopy,
		CurrentPlayer:           curID,
		Winners:                 winners,
		SmallBlind:              g.currentSmallBlind(),
		BigBlind:                g.currentBigBlind(),
		BlindLevel:              g.liveBlindLevel,
		NextBlindIncreaseUnixMs: g.liveNextBlindMs,
	}
}

// dealFlop adds three community cards. Caller MUST hold g.mu.
func (g *Game) dealFlop() {
	mustHeld(&g.mu)
	// Burn one card before dealing the flop (standard poker rule)
	if _, ok := g.deck.Draw(); ok {
		g.log.Debugf("CARDS: Burn before flop")
	}
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
	// Burn one card before dealing the turn
	if _, ok := g.deck.Draw(); ok {
		g.log.Debugf("CARDS: Burn before turn")
	}
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
	// Burn one card before dealing the river
	if _, ok := g.deck.Draw(); ok {
		g.log.Debugf("CARDS: Burn before river")
	}
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

// RevealPlayerCards marks a player's hole cards as revealed and returns them.
func (g *Game) RevealPlayerCards(playerID string) ([]*pokerrpc.Card, error) {
	reply := make(chan revealCardsResp, 1)
	sm := g.sm
	if sm == nil {
		return nil, fmt.Errorf("state machine not initialized")
	}
	sm.Send(evRevealCardsReq{playerID: playerID, reply: reply})
	res := <-reply
	return res.cards, res.err
}

// HidePlayerCards marks a player's hole cards as hidden (unrevealed).
func (g *Game) HidePlayerCards(playerID string) error {
	reply := make(chan error, 1)
	sm := g.sm
	if sm == nil {
		return fmt.Errorf("state machine not initialized")
	}
	sm.Send(evHideCardsReq{playerID: playerID, reply: reply})
	return <-reply
}

// handleToggleCards updates Player.revealed via FSM and Hand.revealed at showdown.
// Returns cards if revealing, error if hiding.
func (g *Game) handleToggleCards(playerID string, reveal bool) revealCardsResp {
	g.mu.Lock()
	player := g.getPlayerByID(playerID)
	var hp *statemachine.Machine[Player]
	if player != nil {
		hp = player.handParticipation
	}
	autoAdvance := false
	if !reveal && g.shouldSkipTurnTimer() {
		autoAdvance = true
	}
	g.mu.Unlock()

	var resp revealCardsResp
	if g.currentHand == nil {
		resp.err = fmt.Errorf("no active hand")
		return resp
	}
	if player == nil {
		resp.err = fmt.Errorf("player not found")
		return resp
	}
	if autoAdvance {
		resp.err = fmt.Errorf("cannot hide cards during auto-advance")
		return resp
	}

	// Send event to Player FSM to update Player.revealed
	var err error
	if hp != nil {
		reply := make(chan error, 1)
		if reveal {
			hp.Send(evRevealCards{Reply: reply})
		} else {
			hp.Send(evHideCards{Reply: reply})
		}
		err = <-reply
	} else {
		// Allow toggling reveal state after hand participation ends (e.g., during showdown).
		player.mu.Lock()
		player.revealed = reveal
		player.mu.Unlock()
	}

	if err != nil {
		resp.err = err
		return resp
	}

	// Get the cards (player.revealed is already updated by FSM)
	g.mu.Lock()
	cards := g.currentHand.GetPlayerCards(playerID)
	for _, c := range cards {
		resp.cards = append(resp.cards, &pokerrpc.Card{
			Suit:  c.GetSuit(),
			Value: c.GetValue(),
		})
	}
	g.mu.Unlock()

	return resp
}

type evAdvance struct{} // advance current betting/phase when conditions met

// evGotoShowdownReq requests an immediate showdown for the specified round.
// This MUST be scoped to a specific round so that stale requests (for a hand
// that has already completed) can be safely ignored once a new hand starts.
type evGotoShowdownReq struct {
	round int
}

type evMaybeCompleteReq struct{}

type evAdvanceToNextPlayerReq struct{ now time.Time }

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

type revealCardsResp struct {
	cards []*pokerrpc.Card
	err   error
}

type evRevealCardsReq struct {
	playerID string
	reply    chan revealCardsResp
}

type prunePlayersResp struct {
	removed []string
	err     error
}

type evPrunePlayersReq struct {
	playerIDs []string
	reply     chan<- prunePlayersResp
}

type evAutoRevealReq struct {
	playerIDs []string
}

type evHideCardsReq struct {
	playerID string
	reply    chan error
}

// handleGameEvent centralizes cross-state request handling. It returns (nextState, handled).
func handleGameEvent(g *Game, ev any) (GameStateFn, bool) {
	switch e := ev.(type) {
	case evGotoShowdownReq:
		// Guard against stale showdown requests: if the round has already
		// advanced, ignore this event instead of show-downing the wrong hand.
		g.mu.RLock()
		curRound := g.round
		curPhase := g.phase
		g.mu.RUnlock()

		if e.round != curRound {
			g.log.Debugf("handleGameEvent: ignoring stale evGotoShowdownReq for round=%d (current=%d)", e.round, curRound)
			return nil, true
		}

		switch curPhase {
		case pokerrpc.GamePhase_PRE_FLOP,
			pokerrpc.GamePhase_FLOP,
			pokerrpc.GamePhase_TURN,
			pokerrpc.GamePhase_RIVER:
			// Valid betting streets – transition into showdown.
			return stateShowdown, true
		default:
			// In any other phase (NEW_HAND_DEALING, SHOWDOWN, etc.), just ignore.
			g.log.Debugf("handleGameEvent: evGotoShowdownReq ignored in phase=%v", curPhase)
			return nil, true
		}
	case evMaybeCompleteReq:
		g.mu.Lock()
		g.maybeCompleteBettingRound()
		g.mu.Unlock()
		return nil, true
	case evPrunePlayersReq:
		g.mu.Lock()
		removed, err := g.prunePlayers(e.playerIDs)
		g.mu.Unlock()
		if e.reply != nil {
			e.reply <- prunePlayersResp{removed: removed, err: err}
		}
		return nil, true
	case evTimebankExpiredReq:
		// Decide auto action without holding g.mu while calling external APIs
		var act string
		var allowAdvance bool

		g.mu.Lock()
		// Validate phase
		actionable := (g.phase == pokerrpc.GamePhase_PRE_FLOP ||
			g.phase == pokerrpc.GamePhase_FLOP ||
			g.phase == pokerrpc.GamePhase_TURN ||
			g.phase == pokerrpc.GamePhase_RIVER)

		if actionable &&
			g.currentPlayer >= 0 && g.currentPlayer < len(g.players) &&
			g.players[g.currentPlayer] != nil &&
			g.players[g.currentPlayer].ID() == e.id {

			p := g.players[g.currentPlayer]
			need := g.currentBet - p.currentBet

			if need <= 0 {
				// Auto-check path when there's nothing to call.
				// Treat exactly like a normal check so betting-round completion
				// logic runs and the street can advance after everyone checks.
				act = "check"
				allowAdvance = true
			} else {
				// Facing a bet: auto-fold.
				act = "fold"
				allowAdvance = true
			}
		}
		g.mu.Unlock()

		switch act {
		case "check":
			if allowAdvance {
				_ = g.HandlePlayerCheck(e.id)
			}
		case "fold":
			_ = g.HandlePlayerFold(e.id)
		}
		// Notify table to push a fresh GameUpdate and optional typed action event
		g.sendTableEvent(GameEvent{Type: GameEventStateUpdated, ActorID: e.id, Action: act})
		return nil, true
	case evRevealCardsReq:
		resp := g.handleToggleCards(e.playerID, true)
		if e.reply != nil {
			e.reply <- resp
		}
		return nil, true
	case evAutoRevealReq:
		if len(e.playerIDs) == 0 {
			return nil, true
		}
		revealed := make([]AutoRevealPlayer, 0, len(e.playerIDs))
		for _, playerID := range e.playerIDs {
			resp := g.handleToggleCards(playerID, true)
			if resp.err != nil || len(resp.cards) == 0 {
				continue
			}
			revealed = append(revealed, AutoRevealPlayer{
				PlayerID: playerID,
				Cards:    resp.cards,
			})
		}
		if len(revealed) > 0 {
			g.sendTableEvent(GameEvent{
				Type:       GameEventAutoShowCards,
				RevealInfo: revealed,
			})
		}
		return nil, true
	case evHideCardsReq:
		err := g.handleToggleCards(e.playerID, false).err
		if e.reply != nil {
			e.reply <- err
		}
		return nil, true
	case evAdvanceToNextPlayerReq:
		g.mu.Lock()
		g.advanceToNextPlayer(e.now)
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
			err = g.refundUncalledBets()
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

// PrunePlayersForNextHand requests the Game FSM to remove eliminated
// participants at the next-hand preparation boundary.
func (g *Game) PrunePlayersForNextHand(playerIDs []string) ([]string, error) {
	if len(playerIDs) == 0 {
		return nil, nil
	}

	g.mu.RLock()
	sm := g.sm
	g.mu.RUnlock()
	if sm == nil {
		return nil, fmt.Errorf("game state machine not running")
	}

	ids := append([]string(nil), playerIDs...)
	reply := make(chan prunePlayersResp, 1)
	if !sm.TrySend(evPrunePlayersReq{playerIDs: ids, reply: reply}) {
		return nil, fmt.Errorf("game state machine not accepting prune request")
	}
	res := <-reply
	return res.removed, res.err
}

func (g *Game) prunePlayers(playerIDs []string) ([]string, error) {
	mustHeld(&g.mu)

	if g.phase != pokerrpc.GamePhase_SHOWDOWN && g.phase != pokerrpc.GamePhase_NEW_HAND_DEALING {
		return nil, fmt.Errorf("cannot prune players during phase: %s", g.phase)
	}
	if len(playerIDs) == 0 || len(g.players) == 0 {
		return nil, nil
	}

	toRemove := make(map[string]struct{}, len(playerIDs))
	for _, playerID := range playerIDs {
		if playerID == "" {
			continue
		}
		toRemove[playerID] = struct{}{}
	}
	if len(toRemove) == 0 {
		return nil, nil
	}

	removed := make([]string, 0, len(toRemove))
	kept := make([]*Player, 0, len(g.players))
	for _, p := range g.players {
		if p == nil {
			continue
		}
		if _, ok := toRemove[p.ID()]; ok {
			removed = append(removed, p.ID())
			continue
		}
		kept = append(kept, p)
	}
	if len(removed) == 0 {
		return nil, nil
	}

	g.players = kept
	return removed, nil
}
