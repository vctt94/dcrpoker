package poker

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/decred/slog"

	"github.com/vctt94/bisonbotkit/logging"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	"github.com/vctt94/pokerbisonrelay/pkg/statemachine"
)

// request to enter GAME_ACTIVE (StartGame / startNewHand)
type evStartGameReq struct{}

// force game ended → WAITING_FOR_PLAYERS (endGame / game nil)
type evGameEnded struct{}

// TableEvent represents a table event with type and payload
type TableEvent struct {
	Type    pokerrpc.NotificationType
	TableID string
	Payload interface{}
}

// ActionPayload is a simple payload used by the poker package to pass player IDs
// for auto action events (fold, check).
type ActionPayload struct {
	PlayerID string
}

// GameEndedPayload carries information about the game winner for settlement.
type GameEndedPayload struct {
	WinnerID   string // Player ID of the game winner
	WinnerSeat int32  // Seat index of the winner (for settlement branch)
	MatchID    string // Match ID for referee settlement (table_id)
	TotalPot   int64  // Total pot won (in chips)
}

// TableStateFn represents a table state function following Rob Pike's pattern
type TableStateFn = statemachine.StateFn[Table]

// TableConfig holds configuration for a new poker table
type TableConfig struct {
	ID               string
	Log              slog.Logger
	GameLog          slog.Logger
	HostID           string
	BuyIn            int64 // DCR amount required to join table (in atoms)
	MinPlayers       int
	MaxPlayers       int
	SmallBlind       int64 // Poker chips amount for small blind
	BigBlind         int64 // Poker chips amount for big blind
	MinBalance       int64 // Minimum DCR account balance required (in atoms)
	StartingChips    int64 // Poker chips each player starts with in the game
	TimeBank         time.Duration
	AutoStartDelay   time.Duration // Delay before automatically starting next hand after showdown
	AutoAdvanceDelay time.Duration // Delay between streets when all players are all-in (must be > 0)
}

// TableEventManager handles notifications and state updates for table events
type TableEventManager struct {
	eventChannel chan<- TableEvent
	log          slog.Logger
}

// SetEventChannel sets the event channel for the event manager
func (tem *TableEventManager) SetEventChannel(eventChannel chan<- TableEvent) {
	tem.eventChannel = eventChannel
}

// PublishEvent publishes an event to the channel (non-blocking)
func (tem *TableEventManager) PublishEvent(eventType pokerrpc.NotificationType, tableID string, payload interface{}) {
	if tem.eventChannel == nil {
		// No channel wired; count as dropped and warn once per call site
		IncrementEventDropped()
		if tem.log != nil {
			tem.log.Errorf("TableEvent drop: no event channel (type=%s table=%s)", eventType, tableID)
		}
		return
	}

	select {
	case tem.eventChannel <- TableEvent{
		Type:    eventType,
		TableID: tableID,
		Payload: payload,
	}:
		IncrementEventPublished()
	default:
		// Channel is full or closed, event is dropped
		IncrementEventDropped()
		if tem.log != nil {
			tem.log.Errorf("TableEvent drop: channel full or closed (type=%s table=%s)", eventType, tableID)
		}
	}
}

// SetEventChannel sets the event channel for the table
func (t *Table) SetEventChannel(eventChannel chan<- TableEvent) {
	t.eventManager.SetEventChannel(eventChannel)
}

// PublishEvent publishes an event from the table (non-blocking)
func (t *Table) PublishEvent(eventType pokerrpc.NotificationType, tableID string, payload interface{}) {
	t.eventManager.PublishEvent(eventType, tableID, payload)
}

// Table represents a poker table that manages users and delegates game logic to Game
type Table struct {
	mu RWLock
	// State machine - Rob Pike's pattern
	sm         *statemachine.Machine[Table]
	log        slog.Logger
	logBackend *logging.LogBackend
	config     TableConfig
	users      map[string]*User // Users seated at the table
	game       *Game            // Game logic that handles all player management
	createdAt  time.Time
	lastAction time.Time
	// Event manager for notifications
	eventManager *TableEventManager

	// Persist the last showdown result for retrieval after phase advances
	lastShowdown *ShowdownResult

	// Idempotency guard: track which hand (by game round) has been resolved
	resolvedRound int

	// Timeout management
	timeoutChan chan struct{}
	timeoutStop chan struct{}

	// Channel for receiving events from Game FSM
	gameEventChan chan GameEvent
	gameEventStop chan struct{}

	// Shutdown management
	closeOnce sync.Once
	closed    bool
	wg        sync.WaitGroup
}

// NewTable creates a new poker table
func NewTable(cfg TableConfig) *Table {
	t := &Table{
		log:           cfg.Log,
		config:        cfg,
		users:         make(map[string]*User),
		createdAt:     time.Now(),
		lastAction:    time.Now(),
		eventManager:  &TableEventManager{log: cfg.Log},
		timeoutChan:   make(chan struct{}, 1),
		timeoutStop:   make(chan struct{}),
		gameEventChan: make(chan GameEvent, 10), // Buffered to avoid blocking Game FSM
		gameEventStop: make(chan struct{}),
	}

	// Initialize state machine with first state function
	t.sm = statemachine.New(t, tableStateWaitingForPlayers, 32)
	t.sm.Start(context.Background())

	// Start game event processing goroutine (track with WaitGroup)
	t.wg.Add(1)
	go t.gameEventLoop()

	return t
}

// Close stops all background goroutines and cleans up resources.
// This must be called when a table is no longer needed to prevent goroutine leaks.
// It is safe to call Close() multiple times (idempotent).
func (t *Table) Close() {
	t.closeOnce.Do(func() {
		// Signal stop to all background goroutines
		close(t.timeoutStop)
		close(t.gameEventStop)

		// Wait for all background goroutines to finish
		t.wg.Wait()

		// Grab references while holding lock
		t.mu.Lock()
		game := t.game
		sm := t.sm
		t.game = nil
		t.sm = nil
		t.closed = true
		t.mu.Unlock()

		// Clean up the game (without holding lock to avoid deadlock)
		if game != nil {
			game.Close()
		}

		// Stop the table state machine (without holding lock to avoid deadlock)
		if sm != nil {
			sm.Stop()
		}
	})
}

// gameEventLoop processes events from the Game FSM
func (t *Table) gameEventLoop() {
	defer t.wg.Done()

	for {
		select {
		case event := <-t.gameEventChan:
			t.handleGameEvent(event)
		case <-t.gameEventStop:
			return
		}
	}
}

// WireGameEvents connects a Game's event channel to this table so the Game FSM
// can publish events (betting round complete, showdown, etc.) back to the table.
// This is used when restoring a previously running game outside of the normal
// StartGame/startNewHand flow.
func (t *Table) WireGameEvents(g *Game) {
	if g == nil {
		return
	}
	g.SetTableEventChannel(t.gameEventChan)
}

// advanceToNextStreet advances the game to the next betting street
func (t *Table) advanceToNextStreet() error {
	if t.game == nil {
		return fmt.Errorf("no active game")
	}

	phase := t.game.GetPhase()
	tableID := t.config.ID

	// The Game FSM has already advanced the phase internally via maybeCompleteBettingRound.
	// The Table just needs to publish the NEW_ROUND event based on the current phase.
	// Do NOT call state transition methods here as that would cause double advancement.
	switch phase {
	case pokerrpc.GamePhase_FLOP:
		t.log.Debugf("advanceToNextStreet: Publishing NEW_ROUND for FLOP")
		t.PublishEvent(pokerrpc.NotificationType_NEW_ROUND, tableID, nil)
	case pokerrpc.GamePhase_TURN:
		t.log.Debugf("advanceToNextStreet: Publishing NEW_ROUND for TURN")
		t.PublishEvent(pokerrpc.NotificationType_NEW_ROUND, tableID, nil)
	case pokerrpc.GamePhase_RIVER:
		t.log.Debugf("advanceToNextStreet: Publishing NEW_ROUND for RIVER")
		t.PublishEvent(pokerrpc.NotificationType_NEW_ROUND, tableID, nil)
	case pokerrpc.GamePhase_SHOWDOWN:
		// Betting closed early (all-in or single player), already at showdown
		t.log.Debugf("advanceToNextStreet: Already at SHOWDOWN")
		return nil
	default:
		return fmt.Errorf("cannot advance from phase %v", phase)
	}

	return nil
}

// handleGameEvent processes a single event from the Game FSM
func (t *Table) handleGameEvent(event GameEvent) {
	switch event.Type {
	case GameEventBettingRoundComplete:
		t.log.Debugf("Table received GameEventBettingRoundComplete")
		// Advance to next street
		if err := t.advanceToNextStreet(); err != nil {
			t.log.Errorf("Failed to advance to next street: %v", err)
		}
	case GameEventStateUpdated:
		// Publish typed action notification when available to keep clients in sync,
		// then trigger a game state broadcast via the event pipeline.
		switch event.Action {
		case "check":
			t.PublishEvent(pokerrpc.NotificationType_CHECK_MADE, t.config.ID, ActionPayload{PlayerID: event.ActorID})
		case "fold":
			t.PublishEvent(pokerrpc.NotificationType_PLAYER_FOLDED, t.config.ID, ActionPayload{PlayerID: event.ActorID})
		default:
			t.PublishEvent(pokerrpc.NotificationType_UNKNOWN, t.config.ID, nil)
		}
	case GameEventShowdownComplete:
		if err := t.handleShowdownComplete(event.ShowdownResult); err != nil {
			t.log.Errorf("Failed to handle showdown complete: %v", err)
		}
	case GameEventAutoStartTriggered:
		if err := t.handleAutoStart(); err != nil {
			t.log.Debugf("Auto-start check failed: %v, will retry on next trigger", err)
		}
	case GameEventGameOver:
		t.log.Infof("Table received GameEventGameOver - winner: %s", event.WinnerID)
		t.handleGameOver(event.WinnerID)
	default:
		t.log.Warnf("Unknown game event type: %v", event.Type)
	}
}

// handleAutoStart checks conditions and starts a new hand if ready
func (t *Table) handleAutoStart() error {
	t.mu.RLock()
	game := t.game
	minPlayers := t.config.MinPlayers
	t.mu.RUnlock()

	if game == nil {
		return fmt.Errorf("no game active")
	}

	// Count players with chips remaining
	readyCount := 0
	players := game.GetPlayers()
	for _, p := range players {
		if p == nil {
			continue
		}
		// Count players who have any chips left
		if p.Balance() > 0 {
			readyCount++
		}
	}

	if readyCount < minPlayers {
		return fmt.Errorf("not enough players ready: %d < %d", readyCount, minPlayers)
	}

	// Conditions met - start new hand
	t.log.Debugf("Auto-start conditions met: starting new hand")
	return t.startNewHand()
}

// handleGameOver is called when the game has ended (only one player has chips)
func (t *Table) handleGameOver(winnerID string) {
	t.log.Infof("Game over - winner: %s. Game will remain in SHOWDOWN state.", winnerID)

	t.mu.RLock()
	game := t.game
	tableID := t.config.ID
	users := t.users
	t.mu.RUnlock()

	if game != nil {
		game.mu.Lock()
		game.cancelAutoStart()
		game.mu.Unlock()
		t.log.Debugf("Canceled auto-start timer due to game over")
	}

	// Determine winner's seat index for settlement
	var winnerSeat int32 = -1
	for _, u := range users {
		if u.ID == winnerID {
			u.mu.RLock()
			winnerSeat = int32(u.TableSeat)
			u.mu.RUnlock()
			break
		}
	}

	// Calculate total pot from showdown result if available
	var totalPot int64
	t.mu.RLock()
	if t.lastShowdown != nil {
		totalPot = t.lastShowdown.TotalPot
	}
	t.mu.RUnlock()

	// Construct matchID for referee settlement (table_id is the match identifier for SNG)
	matchID := tableID

	t.log.Infof("Game ended - winner: %s, seat: %d, matchID: %s, totalPot: %d",
		winnerID, winnerSeat, matchID, totalPot)

	// Publish GAME_ENDED event with settlement info
	t.PublishEvent(pokerrpc.NotificationType_GAME_ENDED, tableID, GameEndedPayload{
		WinnerID:   winnerID,
		WinnerSeat: winnerSeat,
		MatchID:    matchID,
		TotalPot:   totalPot,
	})
}

// Thread-safe readiness check for state fns.
// This checks if all players are ready and escrows are funded (for FSM transitions).
// Does NOT check presign - that's a separate check for game start.
func (t *Table) allPlayersReady() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if len(t.users) < t.config.MinPlayers {
		return false
	}
	requireEscrow := t.config.BuyIn > 0
	for _, u := range t.users {
		u.mu.RLock()
		isReady := u.IsReady
		escrowID := u.EscrowID
		escrowReady := u.EscrowReady
		u.mu.RUnlock()

		if !isReady {
			return false
		}
		if requireEscrow && escrowID == "" {
			return false
		}
		if escrowID != "" && !escrowReady {
			return false
		}
	}
	return true
}

// WAITING_FOR_PLAYERS
func tableStateWaitingForPlayers(t *Table, in <-chan any) TableStateFn {
	for ev := range in {
		switch ev.(type) {
		case evSetUserReady:
			e := ev.(evSetUserReady)
			err := t.applyUserReady(e.userID, e.ready)
			if e.reply != nil {
				e.reply <- err
			}
			if err == nil && t.allPlayersReady() {
				return tableStatePlayersReady
			}
		case evSetUserEscrow:
			e := ev.(evSetUserEscrow)
			err := t.applyUserEscrow(e.userID, e.escrowID, e.ready)
			if e.reply != nil {
				e.reply <- err
			}
			// Escrow change may affect readiness; check transitions
			if err == nil && t.allPlayersReady() {
				return tableStatePlayersReady
			}
		case evSetUserPresignComplete:
			e := ev.(evSetUserPresignComplete)
			err := t.applyUserPresignComplete(e.userID)
			if e.reply != nil {
				e.reply <- err
			}
			// Presign completion may affect readiness; check transitions
			if err == nil && t.allPlayersReady() {
				return tableStatePlayersReady
			}
		case evUsersChanged:
			if t.allPlayersReady() {
				return tableStatePlayersReady
			}
			// stay waiting
		case evStartGameReq:
			// If StartGame was called AND everyone is ready, go ACTIVE immediately.
			if t.allPlayersReady() {
				return tableStateGameActive
			}
			// otherwise remain waiting (server shouldn't call StartGame yet)
		case evGameEnded:
			// already waiting
		default:
		}
	}
	return nil
}

// PLAYERS_READY
func tableStatePlayersReady(t *Table, in <-chan any) TableStateFn {
	for ev := range in {
		switch ev.(type) {
		case evSetUserReady:
			e := ev.(evSetUserReady)
			err := t.applyUserReady(e.userID, e.ready)
			if e.reply != nil {
				e.reply <- err
			}
			if err == nil && !t.allPlayersReady() {
				return tableStateWaitingForPlayers
			}
		case evSetUserEscrow:
			e := ev.(evSetUserEscrow)
			err := t.applyUserEscrow(e.userID, e.escrowID, e.ready)
			if e.reply != nil {
				e.reply <- err
			}
			// Escrow change may affect readiness
			if err == nil && !t.allPlayersReady() {
				return tableStateWaitingForPlayers
			}
		case evSetUserPresignComplete:
			e := ev.(evSetUserPresignComplete)
			err := t.applyUserPresignComplete(e.userID)
			if e.reply != nil {
				e.reply <- err
			}
		case evUsersChanged:
			if !t.allPlayersReady() {
				return tableStateWaitingForPlayers
			}
			// remain ready
		case evStartGameReq:
			// Start the game from the ready state
			return tableStateGameActive
		case evGameEnded:
			return tableStateWaitingForPlayers
		default:
		}
	}
	return nil
}

// GAME_ACTIVE
func tableStateGameActive(t *Table, in <-chan any) TableStateFn {
	for ev := range in {
		switch ev.(type) {
		case evSetUserReady:
			e := ev.(evSetUserReady)
			err := t.applyUserReady(e.userID, e.ready)
			if e.reply != nil {
				e.reply <- err
			}
		case evSetUserEscrow:
			e := ev.(evSetUserEscrow)
			err := t.applyUserEscrow(e.userID, e.escrowID, e.ready)
			if e.reply != nil {
				e.reply <- err
			}
		case evSetUserPresignComplete:
			e := ev.(evSetUserPresignComplete)
			err := t.applyUserPresignComplete(e.userID)
			if e.reply != nil {
				e.reply <- err
			}
		case evGameEnded:
			return tableStateWaitingForPlayers
		default:
		}
	}
	return nil
}

// GetTableStateString returns a string representation of the current table state
func (t *Table) GetTableStateString() string {
	// Check if state machine is nil (table closed)
	if t.sm == nil {
		return "TERMINATED"
	}

	currentState := t.sm.Current()
	if currentState == nil {
		return "TERMINATED"
	}

	// Use function pointer comparison to determine state
	switch fmt.Sprintf("%p", currentState) {
	case fmt.Sprintf("%p", tableStateWaitingForPlayers):
		return "WAITING_FOR_PLAYERS"
	case fmt.Sprintf("%p", tableStatePlayersReady):
		return "PLAYERS_READY"
	case fmt.Sprintf("%p", tableStateGameActive):
		return "GAME_ACTIVE"
	default:
		return "UNKNOWN"
	}
}

// CheckAllPlayersReady checks if all players are ready (and escrows funded for escrow tables).
// Does NOT include presign check - use CheckAllPlayersReadyForGameStart for that.
func (t *Table) CheckAllPlayersReady() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	// Requires: t.mu held
	if len(t.users) < t.config.MinPlayers {
		return false
	}
	requireEscrow := t.config.BuyIn > 0
	for _, u := range t.users {
		u.mu.RLock()
		isReady := u.IsReady
		escrowID := u.EscrowID
		escrowReady := u.EscrowReady
		u.mu.RUnlock()

		if !isReady {
			return false
		}
		if requireEscrow && escrowID == "" {
			return false
		}
		if escrowID != "" && !escrowReady {
			return false
		}
	}
	return true
}

// CheckAllPlayersReadyForGameStart checks if all players are ready to start the game.
// For escrow-backed tables, this also requires presigning to be complete.
func (t *Table) CheckAllPlayersReadyForGameStart() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	// Requires: t.mu held
	if len(t.users) < t.config.MinPlayers {
		return false
	}
	requireEscrow := t.config.BuyIn > 0
	for _, u := range t.users {
		u.mu.RLock()
		isReady := u.IsReady
		escrowID := u.EscrowID
		escrowReady := u.EscrowReady
		presignComplete := u.PresignComplete
		u.mu.RUnlock()

		if !isReady {
			return false
		}
		if requireEscrow && escrowID == "" {
			return false
		}
		if escrowID != "" && !escrowReady {
			return false
		}
		// For escrow-backed tables, presigning must be complete
		if requireEscrow && !presignComplete {
			return false
		}
	}
	return true
}

// StartGame starts a new game at the table using the state machine
func (t *Table) StartGame() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// 1) Ensure readiness under the table lock.
	// Requires: t.mu held
	if t.GetTableStateString() != "PLAYERS_READY" {
		if len(t.users) < t.config.MinPlayers {
			return fmt.Errorf("cannot start game: not enough ready players")
		}
		requireEscrow := t.config.BuyIn > 0
		for _, u := range t.users {
			u.mu.RLock()
			isReady := u.IsReady
			escrowID := u.EscrowID
			escrowReady := u.EscrowReady
			u.mu.RUnlock()

			if !isReady {
				return fmt.Errorf("cannot start game: not enough ready players")
			}
			if requireEscrow && escrowID == "" {
				return fmt.Errorf("cannot start game: not enough ready players")
			}
			if escrowID != "" && !escrowReady {
				return fmt.Errorf("cannot start game: not enough ready players")
			}
		}
	}
	if len(t.users) < t.config.MinPlayers {
		return fmt.Errorf("not enough players to start game")
	}
	// Clean up any stale game explicitly to prevent goroutine leaks
	if t.game != nil {
		t.game.Close()
		t.game = nil
	}

	// 2) Build the active player list in seat order (pure table concern).
	active := make([]*User, 0, len(t.users))
	for _, u := range t.users {
		active = append(active, u)
	}
	sort.Slice(active, func(i, j int) bool { return active[i].TableSeat < active[j].TableSeat })

	// 3) Create Game (no mutations outside its API).
	gameLog := t.log
	if t.config.GameLog != nil {
		gameLog = t.config.GameLog
	}
	g, err := NewGame(GameConfig{
		NumPlayers:       len(active),
		StartingChips:    t.config.StartingChips,
		SmallBlind:       t.config.SmallBlind,
		BigBlind:         t.config.BigBlind,
		TimeBank:         t.config.TimeBank,
		AutoStartDelay:   t.config.AutoStartDelay,
		AutoAdvanceDelay: t.config.AutoAdvanceDelay,
		Log:              gameLog,
	})
	if err != nil {
		return fmt.Errorf("failed to create game: %w", err)
	}

	// 4) Inject players via API (Game owns its Player objects and SMs).
	g.SetPlayers(active)

	// 5) Wire up game event channel so Game FSM can send events to Table
	g.SetTableEventChannel(t.gameEventChan)

	// 6) Start the game FSM so it's ready to process events
	go g.Start(context.Background())

	// 7) Set up notification to broadcast NEW_HAND_STARTED when FSM reaches PRE_FLOP.
	//    This ensures clients see complete state (blinds posted, current player set).
	preFlopCh := g.SetupPreFlopNotification()
	defer g.ClearPreFlopNotification()

	// 8) Kick off FSM transitions: evStartHand → statePreDeal → stateDeal → stateBlinds → statePreFlop
	g.sm.Send(evStartHand{})

	// 9) Update table state machine to GAME_ACTIVE
	t.sm.Send(evStartGameReq{})

	// NOW after sending updates we can assign the game object
	t.game = g
	// 10) Wait for FSM to reach PRE_FLOP before assigning game object and broadcasting
	//     Only assign t.game after PRE_FLOP is reached - until then, no game object exists
	select {
	case <-preFlopCh:
		t.log.Debugf("StartGame: PRE_FLOP reached via FSM, broadcasting NEW_HAND_STARTED")
		t.PublishEvent(pokerrpc.NotificationType_NEW_HAND_STARTED, t.config.ID, nil)
		t.log.Debugf("StartGame: Hand setup complete")
	case <-time.After(5 * time.Second):
		// Timeout - clean up the game object we created but never assigned
		g.Close()
		t.log.Warnf("StartGame: Timeout waiting for PRE_FLOP FSM transition")
		return fmt.Errorf("timeout waiting for game to reach PRE_FLOP")
	}

	return nil
}

// IsGameStarted returns whether the game has started.
// This checks the table state machine to determine if the game is actually active,
// not just if a game object exists (which may be created before the game transitions to active).
func (t *Table) IsGameStarted() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.isGameActive()
}

// AreAllPlayersReady returns whether all players are ready
func (t *Table) AreAllPlayersReady() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	state := t.GetTableStateString()
	return state == "PLAYERS_READY" || state == "GAME_ACTIVE" || state == "SHOWDOWN"
}

// isGameActive returns true if the game is currently active
func (t *Table) isGameActive() bool {
	return t.game != nil
}

// handleShowdownComplete stores the showdown result received from the game FSM
func (t *Table) handleShowdownComplete(result *ShowdownResult) error {
	if result == nil {
		return fmt.Errorf("showdown result is nil")
	}

	currentRound := t.game.GetRound()

	// Idempotency guard: check if we already processed this round's showdown
	t.mu.RLock()
	alreadyResolved := t.lastShowdown != nil && t.resolvedRound == currentRound
	resolvedRound := t.resolvedRound
	t.mu.RUnlock()
	if alreadyResolved {
		t.log.Debugf("handleShowdownComplete: idempotency guard triggered, round=%d, resolvedRound=%d", currentRound, resolvedRound)
		return nil
	}

	// Store the result in the table for later retrieval
	t.mu.Lock()
	t.lastShowdown = result
	t.resolvedRound = currentRound
	t.mu.Unlock()

	t.log.Debugf("handleShowdownComplete: stored showdown result for round %d with %d winners", currentRound, len(result.Winners))

	// Now publish the SHOWDOWN_RESULT event with the complete payload
	showdownPayload := &pokerrpc.Showdown{
		Winners: result.WinnerInfo,
		Pot:     result.TotalPot,
	}
	t.PublishEvent(pokerrpc.NotificationType_SHOWDOWN_RESULT, t.config.ID, showdownPayload)

	return nil
}

// startNewHand starts a fresh hand atomically (acquires the table lock internally)
// startNewHand starts a fresh hand atomically from the table’s perspective.
// It builds/sorts the users list under t.mu, then lets the Game do all state mutations.
func (t *Table) startNewHand() error {
	t.log.Debugf("startNewHand: Starting new hand")

	// Build the sorted users list under table lock only.
	t.mu.Lock()
	if t.game == nil {
		t.mu.Unlock()
		return fmt.Errorf("startNewHand called but game is nil - this should not happen")
	}

	playersAtTable := len(t.users)
	minRequired := t.config.MinPlayers
	if playersAtTable >= 2 && playersAtTable < t.config.MinPlayers {
		minRequired = 2 // allow heads-up
	}
	if playersAtTable < minRequired {
		t.mu.Unlock()
		return fmt.Errorf("not enough players to start new hand: %d < %d", playersAtTable, minRequired)
	}

	activeUsers := make([]*User, 0, len(t.users))
	for _, u := range t.users {
		activeUsers = append(activeUsers, u)
	}
	sort.Slice(activeUsers, func(i, j int) bool {
		return activeUsers[i].TableSeat < activeUsers[j].TableSeat
	})
	// For logs, snapshot balances safely via Game (outside t.mu)
	g := t.game
	t.mu.Unlock()

	// Set up notification to broadcast NEW_HAND_STARTED when PRE_FLOP is reached.
	// Create the notification channel BEFORE triggering FSM transitions.
	preFlopCh := g.SetupPreFlopNotification()
	defer g.ClearPreFlopNotification()

	// Rebuild/reuse players (no hand-state mutation here; FSM will do that).
	if err := g.ResetForNewHandFromUsers(activeUsers); err != nil {
		return fmt.Errorf("failed to setup new hand: %w", err)
	}

	// Update table state / bookkeeping
	t.mu.Lock()
	t.lastShowdown = nil
	t.resolvedRound = -1
	t.sm.Send(evStartGameReq{})
	t.lastAction = time.Now()
	t.mu.Unlock()

	// Kick the FSM after cards are dealt so deck reseed in statePreDeal does
	// not race with dealing. This flows: evStartHand → PRE_DEAL → BLINDS → PRE_FLOP.
	g.sm.Send(evStartHand{})

	// Wait for PRE_FLOP to be reached before broadcasting.
	// This ensures clients see the complete game state with blinds posted and current player set.
	t.log.Debugf("startNewHand: Waiting for PRE_FLOP FSM transition...")
	select {
	case <-preFlopCh:
		t.log.Debugf("startNewHand: PRE_FLOP reached via FSM, broadcasting NEW_HAND_STARTED")
		t.PublishEvent(pokerrpc.NotificationType_NEW_HAND_STARTED, t.config.ID, nil)
		t.log.Debugf("startNewHand: Hand setup complete")
	case <-time.After(5 * time.Second):
		// Safety timeout - should never happen in normal operation
		t.log.Warnf("startNewHand: Timeout waiting for PRE_FLOP FSM transition")
	}

	return nil
}

// GetStatus returns the current status of the table
func (t *Table) GetStatus() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	status := fmt.Sprintf("Table %s:\n", t.config.ID)
	status += fmt.Sprintf("Players: %d/%d\n", len(t.users), t.config.MaxPlayers)
	status += fmt.Sprintf("Buy-in: %.8f DCR\n", float64(t.config.BuyIn)/1e8)
	status += fmt.Sprintf("Starting Chips: %d chips\n", t.config.StartingChips)
	status += fmt.Sprintf("Blinds: %d/%d chips\n", t.config.SmallBlind, t.config.BigBlind)

	if t.game != nil {
		status += "Game in progress\n"
	} else {
		status += "Waiting for players\n"
	}

	return status
}

// GetUsers returns all users at the table
func (t *Table) GetUsers() []*User {
	t.mu.RLock()
	defer t.mu.RUnlock()

	users := make([]*User, 0, len(t.users))
	for _, u := range t.users {
		users = append(users, u)
	}

	// Sort by TableSeat to ensure consistent ordering
	sort.Slice(users, func(i, j int) bool {
		return users[i].TableSeat < users[j].TableSeat
	})

	return users
}

// GetBigBlind returns the big blind value for the table
func (t *Table) GetBigBlind() int64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.config.BigBlind
}

// MakeBet handles betting by delegating to the Game layer
func (t *Table) MakeBet(userID string, amount int64) error {
	if amount < 0 {
		return fmt.Errorf("amount cannot be negative")
	}

	t.mu.Lock()

	user := t.users[userID]
	if user == nil {
		t.mu.Unlock()
		return fmt.Errorf("user not found")
	}

	// Validate that it's this player's turn to act. Allow routing as soon as a
	// game exists; table FSM may not have switched to GAME_ACTIVE yet after a restore.
	if t.game != nil {
		// Disallow actions outside betting streets
		switch t.game.GetPhase() {
		case pokerrpc.GamePhase_PRE_FLOP, pokerrpc.GamePhase_FLOP, pokerrpc.GamePhase_TURN, pokerrpc.GamePhase_RIVER:
			// allowed
		default:
			t.mu.Unlock()
			return fmt.Errorf("action not allowed during phase: %s", t.game.GetPhase())
		}
		currentPlayerID := t.currentPlayerID()
		t.log.Debugf("MakeBet: userID=%s, currentPlayerID=%s, currentPlayer=%d, gamePhase=%v, amount=%d",
			userID, currentPlayerID, t.game.GetCurrentPlayer(), t.game.GetPhase(), amount)
		if currentPlayerID != userID {
			t.mu.Unlock()
			return fmt.Errorf("not your turn to act")
		}

		// Disallow actions when current player is not actively IN_GAME (e.g., ALL_IN)
		if cp := t.game.GetCurrentPlayerObject(); cp != nil {
			if cp.GetCurrentStateString() != IN_GAME_STATE {
				t.mu.Unlock()
				return fmt.Errorf("player cannot act in current state")
			}
		}

		// Route through Game FSM; no direct fallback
		if t.game.sm == nil {
			t.mu.Unlock()
			return fmt.Errorf("game state machine not running")
		}
		reply := make(chan error, 1)
		t.game.sm.Send(evHandleBetReq{id: userID, amount: amount, reply: reply})
		t.lastAction = time.Now()
		t.mu.Unlock()
		if err := <-reply; err != nil {
			return err
		}
		return nil
	}

	t.lastAction = time.Now()
	t.mu.Unlock()

	// Betting round completion is now handled by Game FSM sending events to Table
	return nil
}

// GetMinPlayers returns the minimum number of players required
func (t *Table) GetMinPlayers() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.config.MinPlayers
}

// GetMaxPlayers returns the maximum number of players allowed
func (t *Table) GetMaxPlayers() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.config.MaxPlayers
}

// GetConfig returns the table configuration
func (t *Table) GetConfig() TableConfig {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.config
}

// GetSettlementMatchID returns the matchID to use for Schnorr settlement.
// For WTA poker, the tableID itself is the matchID (a random 16-byte hex string).
func (t *Table) GetSettlementMatchID() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.config.ID
}

// GetGamePhase returns the current phase of the active game, or WAITING.
func (t *Table) GetGamePhase() pokerrpc.GamePhase {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.game == nil {
		return pokerrpc.GamePhase_WAITING
	}
	return t.game.GetPhase()
}

// GetGame returns the current game (can be nil)
func (t *Table) GetGame() *Game {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.game
}

// GetLastShowdown returns the last recorded showdown result (if any).
func (t *Table) GetLastShowdown() *ShowdownResult {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.lastShowdown
}

// GetCurrentBet returns the current highest bet for the ongoing betting round.
// If no game is active it returns zero.
func (t *Table) GetCurrentBet() int64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.game == nil {
		return 0
	}
	return t.game.GetCurrentBet()
}

// GetCurrentPlayerID returns the ID of the player whose turn it is
func (t *Table) GetCurrentPlayerID() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	// Get current player directly from game
	if t.game == nil {
		return ""
	}
	p := t.game.GetCurrentPlayerObject()
	if p == nil {
		return ""
	}
	return p.id
}

// currentPlayerID returns the current player ID
func (t *Table) currentPlayerID() string {
	mustHeld(&t.mu)
	if t.game == nil {
		return ""
	}
	p := t.game.GetCurrentPlayerObject()
	if p == nil {
		return ""
	}
	return p.id
}

func (t *Table) HandleFold(userID string) error {
	t.mu.Lock()

	user := t.users[userID]
	if user == nil {
		t.mu.Unlock()
		return fmt.Errorf("user not found")
	}
	// Allow actions as soon as a game exists; table FSM may lag after restore.
	if t.game == nil {
		t.mu.Unlock()
		return nil
	}

	// Disallow actions outside betting streets
	switch t.game.GetPhase() {
	case pokerrpc.GamePhase_PRE_FLOP, pokerrpc.GamePhase_FLOP, pokerrpc.GamePhase_TURN, pokerrpc.GamePhase_RIVER:
		// allowed
	default:
		t.mu.Unlock()
		return fmt.Errorf("action not allowed during phase: %s", t.game.GetPhase())
	}
	curr := t.currentPlayerID()
	t.log.Debugf("HandleFold: userID=%s, currentPlayerID=%s, currentPlayer=%d, gamePhase=%v",
		userID, curr, t.game.GetCurrentPlayer(), t.game.GetPhase())
	if curr != userID {
		t.mu.Unlock()
		return fmt.Errorf("not your turn to act")
	}

	// Disallow actions when current player is not actively IN_GAME (e.g., ALL_IN)
	if cp := t.game.GetCurrentPlayerObject(); cp != nil {
		if cp.GetCurrentStateString() != IN_GAME_STATE {
			t.mu.Unlock()
			return fmt.Errorf("player cannot act in current state")
		}
	}

	if t.game.sm == nil {
		t.mu.Unlock()
		return fmt.Errorf("game state machine not running")
	}
	reply := make(chan error, 1)
	t.game.sm.Send(evHandleFoldReq{id: userID, reply: reply})
	t.lastAction = time.Now()
	t.mu.Unlock()
	if err := <-reply; err != nil {
		return err
	}
	return nil
}

// HandleCall handles call actions by delegating to the Game layer
func (t *Table) HandleCall(userID string) error {
	t.mu.Lock()

	user := t.users[userID]
	if user == nil {
		t.mu.Unlock()
		return fmt.Errorf("user not found")
	}
	// Allow actions as soon as a game exists; table FSM may lag after restore.
	if t.game == nil {
		t.mu.Unlock()
		return nil
	}

	// Disallow actions outside betting streets
	switch t.game.GetPhase() {
	case pokerrpc.GamePhase_PRE_FLOP, pokerrpc.GamePhase_FLOP, pokerrpc.GamePhase_TURN, pokerrpc.GamePhase_RIVER:
		// allowed
	default:
		t.mu.Unlock()
		return fmt.Errorf("action not allowed during phase: %s", t.game.GetPhase())
	}
	curr := t.currentPlayerID()
	t.log.Debugf("HandleCall: userID=%s, currentPlayerID=%s, currentPlayer=%d, gamePhase=%v",
		userID, curr, t.game.GetCurrentPlayer(), t.game.GetPhase())
	if curr != userID {
		t.mu.Unlock()
		return fmt.Errorf("not your turn to act")
	}

	// Route through the Game FSM with a reply channel to ensure the call
	// has been fully processed (balance/currentBet/stateID updated) before
	// we return, but avoid synchronously completing the betting round here.
	var reply chan error
	shouldWaitForReply := false

	if t.game.sm == nil {
		t.mu.Unlock()
		return fmt.Errorf("game state machine not running")
	}
	reply = make(chan error, 1)
	t.game.sm.Send(evHandleCallReq{id: userID, reply: reply})
	shouldWaitForReply = true

	t.lastAction = time.Now()
	t.mu.Unlock()

	if shouldWaitForReply {
		if err := <-reply; err != nil {
			return err
		}
	}

	return nil
}

// HandleCheck handles check actions by delegating to the Game layer
func (t *Table) HandleCheck(userID string) error {
	t.mu.Lock()

	user := t.users[userID]
	if user == nil {
		t.mu.Unlock()
		return fmt.Errorf("user not found")
	}

	// Validate that it's this player's turn to act. Allow routing as soon as a
	// game exists; table FSM may lag after a restore.
	if t.game != nil {
		// Disallow actions outside betting streets
		switch t.game.GetPhase() {
		case pokerrpc.GamePhase_PRE_FLOP, pokerrpc.GamePhase_FLOP, pokerrpc.GamePhase_TURN, pokerrpc.GamePhase_RIVER:
			// allowed
		default:
			t.mu.Unlock()
			return fmt.Errorf("action not allowed during phase: %s", t.game.GetPhase())
		}
		currentPlayerID := t.currentPlayerID()
		t.log.Debugf("HandleCheck: userID=%s, currentPlayerID=%s, currentPlayer=%d, gamePhase=%v",
			userID, currentPlayerID, t.game.GetCurrentPlayer(), t.game.GetPhase())
		if currentPlayerID != userID {
			t.mu.Unlock()
			return fmt.Errorf("not your turn to act")
		}

		// Require FSM to be running; do not fallback
		if t.game.sm == nil {
			t.mu.Unlock()
			return fmt.Errorf("game state machine not running")
		}

		// Disallow actions when current player is not actively IN_GAME (e.g., ALL_IN)
		if cp := t.game.GetCurrentPlayerObject(); cp != nil {
			if cp.GetCurrentStateString() != IN_GAME_STATE {
				t.mu.Unlock()
				return fmt.Errorf("player cannot act in current state")
			}
		}
		reply := make(chan error, 1)
		t.game.sm.Send(evHandleCheckReq{id: userID, reply: reply})

		t.lastAction = time.Now()
		t.mu.Unlock()

		if err := <-reply; err != nil {
			return err
		}
		return nil

	}

	t.lastAction = time.Now()
	t.mu.Unlock()

	// Betting round completion is now handled by Game FSM sending events to Table
	return nil
}

// (removed) postBlindsFromGame: blinds are posted exclusively inside the Game FSM (stateBlinds).

// AddUser adds a user to the table
func (t *Table) AddUser(user *User) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Check if table is full
	if len(t.users) >= t.config.MaxPlayers {
		return fmt.Errorf("table is full")
	}

	// Check if user already at table
	if _, exists := t.users[user.ID]; exists {
		return fmt.Errorf("user already at table")
	}

	// Auto-assign seat if TableSeat is -1 (unseated)
	user.mu.RLock()
	userTableSeat := user.TableSeat
	user.mu.RUnlock()
	if userTableSeat < 0 {
		// Find the next available seat
		occupied := make(map[int]bool)
		for _, u := range t.users {
			u.mu.RLock()
			uTableSeat := u.TableSeat
			u.mu.RUnlock()
			if uTableSeat >= 0 {
				occupied[uTableSeat] = true
			}
		}
		seat := -1
		for i := 0; i < t.config.MaxPlayers; i++ {
			if !occupied[i] {
				seat = i
				break
			}
		}
		if seat == -1 {
			return fmt.Errorf("no available seats")
		}
		user.mu.Lock()
		user.TableSeat = seat
		user.mu.Unlock()
	}

	t.users[user.ID] = user
	t.lastAction = time.Now()

	// Trigger state machine update to check if we should transition to PLAYERS_READY
	t.sm.Send(evUsersChanged{})

	return nil
}

// AddNewUser creates and adds a new user to the table in one operation.
// Only the zk identity (id) and seat are required; optional metadata such as
// escrow bindings can be provided via opts.
func (t *Table) AddNewUser(id string, opts *AddUserOptions) (*User, error) {
	user := NewUser(id, t, opts)
	err := t.AddUser(user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// SetSeatEscrowState binds an escrow to a seat and records whether funding is valid.
// Returns the player ID seated there and whether anything changed.
// This method routes through the Table FSM for proper state management.
func (t *Table) SetSeatEscrowState(seat int, escrowID string, ready bool) (string, bool, error) {
	// First find the user at the seat to get their ID and check if state would change
	t.mu.RLock()
	var user *User
	for _, u := range t.users {
		u.mu.RLock()
		uTableSeat := u.TableSeat
		u.mu.RUnlock()
		if uTableSeat == seat {
			user = u
			break
		}
	}
	if user == nil {
		t.mu.RUnlock()
		return "", false, fmt.Errorf("no player at seat %d", seat)
	}
	userID := user.ID
	user.mu.RLock()
	changed := user.EscrowID != escrowID || user.EscrowReady != ready
	user.mu.RUnlock()
	t.mu.RUnlock()

	if !changed {
		return userID, false, nil
	}

	// Route through FSM
	if err := t.SetPlayerEscrow(userID, escrowID, ready); err != nil {
		return "", false, err
	}
	return userID, true, nil
}

// GetUserAtSeat returns the user seated at the specified seat, or nil if no one is there.
func (t *Table) GetUserAtSeat(seat int) *User {
	t.mu.RLock()
	defer t.mu.RUnlock()

	for _, u := range t.users {
		u.mu.RLock()
		uTableSeat := u.TableSeat
		u.mu.RUnlock()
		if uTableSeat == seat {
			return u
		}
	}
	return nil
}

// RemoveUser removes a user from the table
func (t *Table) RemoveUser(userID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, exists := t.users[userID]; !exists {
		return fmt.Errorf("user not at table")
	}

	delete(t.users, userID)
	t.lastAction = time.Now()

	// Trigger state machine update to check if we should transition back to WAITING_FOR_PLAYERS
	t.sm.Send(evUsersChanged{})

	return nil
}

// GetUser returns a user by ID
func (t *Table) GetUser(userID string) *User {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.users[userID]
}

// SetHost transfers host ownership to a new user
func (t *Table) SetHost(newHostID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Verify the new host is actually at the table
	if _, exists := t.users[newHostID]; !exists {
		return fmt.Errorf("new host %s is not at the table", newHostID)
	}

	// Update the host ID in the config
	t.config.HostID = newHostID
	t.lastAction = time.Now()

	return nil
}

func (t *Table) sendTableEvent(ev any) (err error) {
	t.mu.RLock()
	closed := t.closed
	sm := t.sm
	t.mu.RUnlock()
	if closed || sm == nil {
		return fmt.Errorf("table closed")
	}
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("table closed")
		}
	}()
	sm.Send(ev)
	return nil
}

// applyUserReady mutates readiness under lock; caller must trigger FSM transitions.
func (t *Table) applyUserReady(userID string, ready bool) error {
	t.mu.Lock()
	user := t.users[userID]
	if user == nil {
		t.mu.Unlock()
		return fmt.Errorf("user not found at table")
	}
	// Acquire user lock to write field
	user.mu.Lock()
	user.IsReady = ready
	user.mu.Unlock()
	t.lastAction = time.Now()
	t.mu.Unlock()
	return nil
}

// applyUserEscrow mutates escrow binding under lock; caller must trigger FSM transitions.
func (t *Table) applyUserEscrow(userID string, escrowID string, ready bool) error {
	t.mu.Lock()
	user := t.users[userID]
	if user == nil {
		t.mu.Unlock()
		return fmt.Errorf("user not found at table")
	}
	game := t.game
	t.mu.Unlock()

	// Acquire user lock to read and write fields
	user.mu.Lock()
	// Allow changing escrow only when the game has not started and
	// presigning has not completed for this player. This prevents
	// altering settlement inputs mid-presign or during an active game.
	if user.EscrowID != "" && user.EscrowID != escrowID {
		if game == nil && !user.PresignComplete {
			user.mu.Unlock()
			return fmt.Errorf("user %s already bound to escrow %s", userID, user.EscrowID)
		}
	}

	user.EscrowID = escrowID
	user.EscrowReady = ready
	user.mu.Unlock()

	t.mu.Lock()
	t.lastAction = time.Now()
	t.mu.Unlock()
	return nil
}

// applyUserPresignComplete marks presigning complete under lock.
func (t *Table) applyUserPresignComplete(userID string) error {
	t.mu.Lock()
	user := t.users[userID]
	if user == nil {
		t.mu.Unlock()
		return fmt.Errorf("user not found at table")
	}
	// Acquire user lock to write field
	user.mu.Lock()
	user.PresignComplete = true
	user.mu.Unlock()
	t.lastAction = time.Now()
	t.mu.Unlock()
	return nil
}

// SetPlayerReady sets the ready status for a player via the table FSM.
func (t *Table) SendPlayerReady(userID string, ready bool) error {
	reply := make(chan error, 1)
	if err := t.sendTableEvent(evSetUserReady{
		userID: userID,
		ready:  ready,
		reply:  reply,
	}); err != nil {
		return err
	}

	return <-reply
}

// SetPlayerEscrow sets escrow binding for a player via the table FSM.
func (t *Table) SetPlayerEscrow(userID string, escrowID string, ready bool) error {
	reply := make(chan error, 1)
	if err := t.sendTableEvent(evSetUserEscrow{
		userID:   userID,
		escrowID: escrowID,
		ready:    ready,
		reply:    reply,
	}); err != nil {
		return err
	}

	return <-reply
}

// SetPlayerPresignComplete marks presigning complete for a player via the table FSM.
func (t *Table) SetPlayerPresignComplete(userID string) error {
	reply := make(chan error, 1)
	if err := t.sendTableEvent(evSetUserPresignComplete{
		userID: userID,
		reply:  reply,
	}); err != nil {
		return err
	}

	return <-reply
}

// TableStateSnapshot represents a point-in-time snapshot of table state for safe concurrent access
type TableStateSnapshot struct {
	Config TableConfig
	Users  []UserSnapshot
	Game   GameStateSnapshot // Nested game state snapshot if game is active
}

// GetStateSnapshot returns an atomic snapshot of the table state for safe concurrent access
func (t *Table) GetStateSnapshot() TableStateSnapshot {
	// Grab table data while holding lock
	t.mu.RLock()

	// Create a deep copy of users to avoid race conditions
	usersCopy := make([]UserSnapshot, 0, len(t.users))
	for _, user := range t.users {
		user.mu.RLock()
		userCopy := UserSnapshot{
			ID:              user.ID,
			Name:            user.Name,
			TableSeat:       user.TableSeat,
			IsReady:         user.IsReady,
			IsDisconnected:  user.IsDisconnected,
			JoinedAt:        user.JoinedAt,
			EscrowID:        user.EscrowID,
			EscrowReady:     user.EscrowReady,
			PresignComplete: user.PresignComplete,
		}
		user.mu.RUnlock()
		usersCopy = append(usersCopy, userCopy)
	}

	// Grab references we need without holding lock during expensive operations
	config := t.config
	game := t.game
	t.mu.RUnlock()

	// Sort by TableSeat to ensure consistent ordering
	sort.Slice(usersCopy, func(i, j int) bool {
		return usersCopy[i].TableSeat < usersCopy[j].TableSeat
	})

	// Get game state snapshot WITHOUT holding table lock to avoid nested lock deadlock
	// (game.GetStateSnapshot may need to acquire player locks)
	var gameSnapshot GameStateSnapshot
	if game != nil {
		gameSnapshot = game.GetStateSnapshot()
	}

	return TableStateSnapshot{
		Config: config,
		Users:  usersCopy,
		Game:   gameSnapshot,
	}
}

// XX We need to properly fix this restore for clients. and properly restore game state from sm
func (t *Table) RestoreGame(tableID string) (*Game, error) {
	t.mu.Lock()

	tblCfg := t.config
	// Build new game for currently seated users
	gameLog := t.log
	if tblCfg.GameLog != nil {
		gameLog = tblCfg.GameLog
	}
	t.log.Debugf("RestoreGame: NumPlayers=%d, StartingChips=%d, SmallBlind=%d, BigBlind=%d, TimeBank=%v, AutoStartDelay=%v, AutoAdvanceDelay=%v",
		len(t.users), tblCfg.StartingChips, tblCfg.SmallBlind, tblCfg.BigBlind, tblCfg.TimeBank, tblCfg.AutoStartDelay, tblCfg.AutoAdvanceDelay)

	gCfg := GameConfig{
		NumPlayers:       len(t.users),
		StartingChips:    tblCfg.StartingChips,
		SmallBlind:       tblCfg.SmallBlind,
		BigBlind:         tblCfg.BigBlind,
		TimeBank:         tblCfg.TimeBank,
		AutoStartDelay:   tblCfg.AutoStartDelay,
		AutoAdvanceDelay: tblCfg.AutoAdvanceDelay,
		Log:              gameLog,
	}
	game, err := NewGame(gCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create game: %v", err)
	}

	// Build active users list in seat order
	active := make([]*User, 0, len(t.users))
	for _, u := range t.users {
		active = append(active, u)
	}
	sort.Slice(active, func(i, j int) bool { return active[i].TableSeat < active[j].TableSeat })
	// Initialize players for this game
	game.SetPlayers(active)

	readyIDs := make([]string, 0, len(active))
	for _, u := range active {
		readyIDs = append(readyIDs, u.ID)
	}

	// Attach game to table
	t.game = game

	// Release lock before sending event to avoid deadlock (state machine
	// processing evStartGameReq may need table lock)
	t.mu.Unlock()

	// Send event to state machine to transition to GAME_ACTIVE (non-blocking)
	// The state machine will check allPlayersReady() when processing this event,
	// and since we just set all players to ready, it will transition to GAME_ACTIVE
	if !t.sm.TrySend(evStartGameReq{}) {
		t.log.Warnf("RestoreGame: failed to send evStartGameReq (inbox full)")
	}

	return game, nil
}
