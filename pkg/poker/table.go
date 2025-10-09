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

// fired when users join/leave or toggle ready; state may move to/from PLAYERS_READY
type evUsersChanged struct{}

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

// TableStateFn represents a table state function following Rob Pike's pattern
type TableStateFn = statemachine.StateFn[Table]

// User represents someone seated at the table (not necessarily playing)
type User struct {
	ID                string
	Name              string
	DCRAccountBalance int64 // DCR account balance (in atoms)
	TableSeat         int   // Seat position at the table
	IsReady           bool  // Ready to start/continue games
	JoinedAt          time.Time
	IsDisconnected    bool // Whether the user is disconnected
}

// NewUser creates a new user
func NewUser(id, name string, dcrAccountBalance int64, seat int) *User {
	return &User{
		ID:                id,
		Name:              name,
		DCRAccountBalance: dcrAccountBalance,
		TableSeat:         seat,
		IsReady:           false,
		JoinedAt:          time.Now(),
	}
}

// TableConfig holds configuration for a new poker table
type TableConfig struct {
	ID             string
	Log            slog.Logger
	GameLog        slog.Logger
	HostID         string
	BuyIn          int64 // DCR amount required to join table (in atoms)
	MinPlayers     int
	MaxPlayers     int
	SmallBlind     int64 // Poker chips amount for small blind
	BigBlind       int64 // Poker chips amount for big blind
	MinBalance     int64 // Minimum DCR account balance required (in atoms)
	StartingChips  int64 // Poker chips each player starts with in the game
	TimeBank       time.Duration
	AutoStartDelay time.Duration // Delay before automatically starting next hand after showdown
}

// TableEventManager handles notifications and state updates for table events
type TableEventManager struct {
	eventChannel chan<- TableEvent
}

// SetEventChannel sets the event channel for the event manager
func (tem *TableEventManager) SetEventChannel(eventChannel chan<- TableEvent) {
	tem.eventChannel = eventChannel
}

// PublishEvent publishes an event to the channel (non-blocking)
func (tem *TableEventManager) PublishEvent(eventType pokerrpc.NotificationType, tableID string, payload interface{}) {
	if tem.eventChannel != nil {
		select {
		case tem.eventChannel <- TableEvent{
			Type:    eventType,
			TableID: tableID,
			Payload: payload,
		}:
		default:
			// Channel is full or closed, event is dropped
			// In production, you might want to log this
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
	log        slog.Logger
	logBackend *logging.LogBackend
	config     TableConfig
	users      map[string]*User // Users seated at the table
	game       *Game            // Game logic that handles all player management
	mu         RWLock
	createdAt  time.Time
	lastAction time.Time
	// Event manager for notifications
	eventManager *TableEventManager

	// Persist the last showdown result for retrieval after phase advances
	lastShowdown *ShowdownResult

	// Idempotency guard: track which hand (by game round) has been resolved
	resolvedRound int

	// State machine - Rob Pike's pattern
	sm *statemachine.Machine[Table]

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
		eventManager:  &TableEventManager{},
		timeoutChan:   make(chan struct{}, 1),
		timeoutStop:   make(chan struct{}),
		gameEventChan: make(chan GameEvent, 10), // Buffered to avoid blocking Game FSM
		gameEventStop: make(chan struct{}),
	}

	// Initialize state machine with first state function
	t.sm = statemachine.New(t, tableStateWaitingForPlayers, 32)
	t.sm.Start(context.Background())

	// Start timeout goroutine (track with WaitGroup)
	t.wg.Add(1)
	go t.timeoutLoop()

	// Start game event processing goroutine (track with WaitGroup)
	t.wg.Add(1)
	go t.gameEventLoop()

	return t
}

// timeoutLoop runs a periodic timeout check every 200ms
func (t *Table) timeoutLoop() {
	defer t.wg.Done()

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check for timeouts
			t.HandleTimeouts()
		case <-t.timeoutStop:
			return
		}
	}
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
	case GameEventShowdownReady:
		t.log.Debugf("Table received GameEventShowdownReady")
		// Game FSM will process showdown and send us the complete result
	case GameEventShowdownComplete:
		t.log.Debugf("Table received GameEventShowdownComplete")
		// Store the showdown result
		if err := t.handleShowdownComplete(event.ShowdownResult); err != nil {
			t.log.Errorf("Failed to handle showdown complete: %v", err)
		}
	default:
		t.log.Warnf("Unknown game event type: %v", event.Type)
	}
}

// Thread-safe readiness check for state fns.
func (t *Table) allPlayersReady() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if len(t.users) < t.config.MinPlayers {
		return false
	}
	for _, u := range t.users {
		if !u.IsReady {
			return false
		}
	}
	return true
}

// WAITING_FOR_PLAYERS
func tableStateWaitingForPlayers(t *Table, in <-chan any) TableStateFn {
	for ev := range in {
		switch ev.(type) {
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
			// otherwise remain waiting (server shouldn’t call StartGame yet)
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
		fmt.Println("tableStatePlayersReady")
		switch ev.(type) {
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
		case evGameEnded:
			return tableStateWaitingForPlayers
		default:
		}
	}
	return nil
}

// GetTableStateString returns a string representation of the current table state
func (t *Table) GetTableStateString() string {
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

// CheckAllPlayersReady checks if all players are ready without triggering state machine updates
func (t *Table) CheckAllPlayersReady() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Just check the current readiness status without triggering state machine updates
	return t.allPlayersReadyLocked()
}

func (t *Table) allPlayersReadyLocked() bool {
	mustHeld(&t.mu)
	if len(t.users) < t.config.MinPlayers {
		return false
	}
	for _, u := range t.users {
		if !u.IsReady {
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
	if t.GetTableStateString() != "PLAYERS_READY" && !t.allPlayersReadyLocked() {
		return fmt.Errorf("cannot start game: not enough ready players")
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
		NumPlayers:     len(active),
		StartingChips:  t.config.StartingChips,
		SmallBlind:     t.config.SmallBlind,
		BigBlind:       t.config.BigBlind,
		AutoStartDelay: t.config.AutoStartDelay,
		Log:            gameLog,
	})
	if err != nil {
		return fmt.Errorf("failed to create game: %w", err)
	}

	// 4) Wire auto-start callbacks (pure callbacks, no direct field writes).
	g.SetAutoStartCallbacks(&AutoStartCallbacks{
		MinPlayers: func() int { return t.config.MinPlayers },
		StartNewHand: func() error {
			return t.startNewHand() // should only call g.ResetForNewHand via API; see below
		},
		OnNewHandStarted: nil,
	})

	// 5) Inject players via API (Game owns its Player objects and SMs).
	g.SetPlayers(active)

	// 6) Publish the game on the table so helpers can reference t.game safely.
	t.game = g

	// 6a) Wire up game event channel so Game FSM can send events to Table
	g.SetTableEventChannel(t.gameEventChan)

	// 8) Start the game FSM so it's ready to process events
	go g.Start(context.Background())

	// 9) Set up notification to broadcast NEW_HAND_STARTED when FSM reaches PRE_FLOP.
	//    This ensures clients see complete state (blinds posted, current player set).
	preFlopCh := g.SetupPreFlopNotification()
	defer g.ClearPreFlopNotification()

	// 10) Kick off FSM transitions: evStartHand → statePreDeal → stateDeal → stateBlinds → statePreFlop
	g.sm.Send(evStartHand{})

	// 11) Update table state machine to GAME_ACTIVE
	t.sm.Send(evStartGameReq{})

	// Wait for FSM to reach PRE_FLOP before broadcasting and returning
	select {
	case <-preFlopCh:
		t.log.Debugf("StartGame: PRE_FLOP reached via FSM, broadcasting NEW_HAND_STARTED")
		t.PublishEvent(pokerrpc.NotificationType_NEW_HAND_STARTED, t.config.ID, nil)
		t.log.Debugf("StartGame: Hand setup complete")
	case <-time.After(5 * time.Second):
		t.log.Warnf("StartGame: Timeout waiting for PRE_FLOP FSM transition")
	}

	return nil
}

// IsGameStarted returns whether the game has started
func (t *Table) IsGameStarted() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.game != nil
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
	state := t.GetTableStateString()
	return state == "GAME_ACTIVE"
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

	// Validate that it's this player's turn to act
	if t.isGameActive() && t.game != nil {
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

		// Delegate to Game layer - this handles all the betting logic (locks internally)
		if err := t.game.HandlePlayerBet(userID, amount); err != nil {
			t.mu.Unlock()
			return err
		}
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

// GetGamePhase returns the current phase of the active game, or WAITING.
func (t *Table) GetGamePhase() pokerrpc.GamePhase {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.game == nil {
		return pokerrpc.GamePhase_WAITING
	}
	return t.game.GetPhase()
}

// HandleTimeouts auto-checks or auto-folds the current player when their timebank expires.
// It uses only a read snapshot to decide, and performs exactly ONE mutating call.
// IMPORTANT: do not call this from GetGameState or other read-only RPCs.
// Call only from the dedicated timeout loop goroutine.
func (t *Table) HandleTimeouts() {
	// Fast exits with no locks
	if !t.isGameActive() || t.config.TimeBank == 0 {
		return
	}
	g := t.game
	if g == nil {
		return
	}

	// Only act during betting streets (read-only)
	switch g.GetPhase() {
	case pokerrpc.GamePhase_PRE_FLOP, pokerrpc.GamePhase_FLOP, pokerrpc.GamePhase_TURN, pokerrpc.GamePhase_RIVER:
	default:
		return
	}

	// Get current player directly from game
	cp := g.GetCurrentPlayerObject()
	if cp == nil {
		return
	}

	// Respect monotonic time: compare against a deadline derived from lastAction.
	// Protect reads with player lock to avoid data races with actions.
	cp.mu.RLock()
	la := cp.lastAction
	curBet := cp.currentBet
	pid := cp.id
	cp.mu.RUnlock()
	deadline := la.Add(t.config.TimeBank)
	if time.Now().Before(deadline) {
		return
	}

	playerID := pid
	need := g.GetCurrentBet() - curBet

	// Decide from snapshot WITHOUT holding any locks, then perform exactly one mutating call.
	// Use game-level handlers to avoid a second turn check race at the table layer.
	// Betting round completion is handled by Game FSM sending events to Table.
	if need <= 0 {
		_ = g.HandlePlayerCheck(playerID)
	} else {
		_ = g.HandlePlayerFold(playerID)
	}

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
	if !t.isGameActive() || t.game == nil {
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

	// Mutate and maybe-advance in one critical section.
	if err := t.game.HandlePlayerFold(userID); err != nil {
		t.mu.Unlock()
		return err
	}
	t.lastAction = time.Now()

	t.mu.Unlock()

	// Betting round completion is now handled by Game FSM sending events to Table
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
	if !t.isGameActive() || t.game == nil {
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
		// Fallback to direct handler if FSM not initialized
		if err := t.game.HandlePlayerCall(userID); err != nil {
			t.mu.Unlock()
			return err
		}
	} else {
		reply = make(chan error, 1)
		t.game.sm.Send(evHandleCallReq{id: userID, reply: reply})
		shouldWaitForReply = true
	}

	t.lastAction = time.Now()
	t.mu.Unlock()

	// Wait for the Game FSM to apply the call (if using FSM path)
	// Check this before unlocking to avoid race with endGame() setting t.game=nil
	if shouldWaitForReply {
		if err := <-reply; err != nil {
			return err
		}
	}

	// Betting round completion is now handled by Game FSM sending events to Table
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

	// Validate that it's this player's turn to act
	if t.isGameActive() && t.game != nil {
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

		// Delegate to Game layer - this handles all the checking logic (locks internally)
		if err := t.game.HandlePlayerCheck(userID); err != nil {
			t.mu.Unlock()
			return err
		}

		t.log.Debugf("HandleCheck: user %s checked; actionsInRound=%d currentBet=%d", userID, t.game.GetActionsInRound(), t.game.GetCurrentBet())
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

	t.users[user.ID] = user
	t.lastAction = time.Now()

	// Trigger state machine update to check if we should transition to PLAYERS_READY
	t.sm.Send(evUsersChanged{})

	return nil
}

// AddNewUser creates and adds a new user to the table in one operation
func (t *Table) AddNewUser(id, name string, dcrAccountBalance int64, seat int) (*User, error) {
	user := NewUser(id, name, dcrAccountBalance, seat)
	err := t.AddUser(user)
	if err != nil {
		return nil, err
	}
	return user, nil
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

// SetPlayerReady sets the ready status for a player
func (t *Table) SetPlayerReady(userID string, ready bool) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	user := t.users[userID]
	if user == nil {
		return fmt.Errorf("user not found at table")
	}

	user.IsReady = ready

	// Trigger state machine update to check if we should transition to PLAYERS_READY
	t.sm.Send(evUsersChanged{})

	return nil
}

// TableStateSnapshot represents a point-in-time snapshot of table state for safe concurrent access
type TableStateSnapshot struct {
	Config      TableConfig
	Users       []*User
	GameStarted bool
	GamePhase   pokerrpc.GamePhase
	Game        *GameStateSnapshot // Nested game state snapshot if game is active
}

// GetStateSnapshot returns an atomic snapshot of the table state for safe concurrent access
func (t *Table) GetStateSnapshot() TableStateSnapshot {
	// Grab table data while holding lock
	t.mu.RLock()

	// Create a deep copy of users to avoid race conditions
	usersCopy := make([]*User, 0, len(t.users))
	for _, user := range t.users {
		userCopy := &User{
			ID:                user.ID,
			Name:              user.Name,
			DCRAccountBalance: user.DCRAccountBalance,
			TableSeat:         user.TableSeat,
			IsReady:           user.IsReady,
			JoinedAt:          user.JoinedAt,
		}
		usersCopy = append(usersCopy, userCopy)
	}

	// Grab references we need without holding lock during expensive operations
	config := t.config
	game := t.game
	gamePhase := t.getGamePhase()
	t.mu.RUnlock()

	// Sort by TableSeat to ensure consistent ordering
	sort.Slice(usersCopy, func(i, j int) bool {
		return usersCopy[i].TableSeat < usersCopy[j].TableSeat
	})

	// Get game state snapshot WITHOUT holding table lock to avoid nested lock deadlock
	// (game.GetStateSnapshot may need to acquire player locks)
	var gameSnapshot *GameStateSnapshot
	if game != nil {
		snapshot := game.GetStateSnapshot()
		gameSnapshot = &snapshot
	}

	return TableStateSnapshot{
		Config:      config,
		Users:       usersCopy,
		GameStarted: game != nil,
		GamePhase:   gamePhase,
		Game:        gameSnapshot,
	}
}

// getGamePhase returns the current phase
func (t *Table) getGamePhase() pokerrpc.GamePhase {
	mustHeld(&t.mu)
	if t.game == nil {
		return pokerrpc.GamePhase_WAITING
	}
	return t.game.GetPhase()
}

// SetUserDCRAccountBalance safely updates the DCRAccountBalance of a user seated at the table.
// It acquires the table lock to synchronize concurrent access so that readers (e.g. state snapshots)
// don't race with writers like JoinTable.
func (t *Table) SetUserDCRAccountBalance(userID string, newBalance int64) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	u, ok := t.users[userID]
	if !ok {
		return fmt.Errorf("user not found at table")
	}

	u.DCRAccountBalance = newBalance
	return nil
}

// XX We need to properly fix this restore for clients. and properly restore game state from sm
func (t *Table) RestoreGame(tableID string) (*Game, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	tblCfg := t.config
	// Build new game for currently seated users
	gameLog := t.log
	if tblCfg.GameLog != nil {
		gameLog = tblCfg.GameLog
	}
	gCfg := GameConfig{
		NumPlayers:     len(t.users),
		StartingChips:  tblCfg.StartingChips,
		SmallBlind:     tblCfg.SmallBlind,
		BigBlind:       tblCfg.BigBlind,
		TimeBank:       tblCfg.TimeBank,
		AutoStartDelay: tblCfg.AutoStartDelay,
		Log:            gameLog,
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

	// Attach game to table and mark table as active
	t.game = game
	t.sm.Send(evStartGameReq{})

	return game, nil
}
