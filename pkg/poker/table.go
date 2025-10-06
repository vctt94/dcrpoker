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

// marker (optional)
type tableEvent interface{ isTableEvent() }

// fired when users join/leave or toggle ready; state may move to/from PLAYERS_READY
type evUsersChanged struct{}

func (evUsersChanged) isTableEvent() {}

// request to enter GAME_ACTIVE (StartGame / startNewHand)
type evStartGameReq struct{}

func (evStartGameReq) isTableEvent() {}

// force game ended → WAITING_FOR_PLAYERS (endGame / game nil)
type evGameEnded struct{}

func (evGameEnded) isTableEvent() {}

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
	mu         sync.RWMutex
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
}

// NewTable creates a new poker table
func NewTable(cfg TableConfig) *Table {
	t := &Table{
		log:          cfg.Log,
		config:       cfg,
		users:        make(map[string]*User),
		createdAt:    time.Now(),
		lastAction:   time.Now(),
		eventManager: &TableEventManager{},
		timeoutChan:  make(chan struct{}, 1),
		timeoutStop:  make(chan struct{}),
	}

	// Initialize state machine with first state function
	t.sm = statemachine.New(t, tableStateWaitingForPlayers, 32)
	t.sm.Start(context.Background())

	// Start timeout goroutine
	go t.timeoutLoop()

	return t
}

// timeoutLoop runs a periodic timeout check every 200ms
func (t *Table) timeoutLoop() {
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

// StopTimeout stops the timeout goroutine
func (t *Table) StopTimeout() {
	close(t.timeoutStop)
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
	for {
		switch (<-in).(type) {
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
}

// PLAYERS_READY
func tableStatePlayersReady(t *Table, in <-chan any) TableStateFn {
	for {
		fmt.Println("tableStatePlayersReady")
		switch (<-in).(type) {
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
}

// GAME_ACTIVE
func tableStateGameActive(t *Table, in <-chan any) TableStateFn {
	for {
		switch (<-in).(type) {
		case evGameEnded:
			return tableStateWaitingForPlayers
		default:
		}
	}
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

// caller MUST hold t.mu
func (t *Table) allPlayersReadyLocked() bool {
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
	// Drop any stale game explicitly.
	t.game = nil

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

	// 7) Perform full, deterministic hand setup while holding the table lock
	//    to avoid races (deals, posts blinds, sets current player).
	if err := t.setupNewHand(active); err != nil {
		return fmt.Errorf("failed to setup new hand: %w", err)
	}

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

// handleShowdown delegates showdown logic to the game and handles notifications
func (t *Table) handleShowdown() error {
	if t.game == nil {
		return fmt.Errorf("game is nil")
	}

	currentRound := t.game.GetRound()

	// Idempotency guard: already resolved for this hand
	if t.lastShowdown != nil && t.resolvedRound == currentRound {
		t.log.Debugf("handleShowdown: idempotency guard triggered, round=%d, resolvedRound=%d", currentRound, t.resolvedRound)
		return nil
	}

	// Delegate showdown logic to the game and cache authoritative result
	result, err := t.game.handleShowdown()
	if err != nil {
		t.log.Errorf("failed to handle showdown: %v", err)
		return err
	}
	// Persist result for retrieval after phase advances
	t.lastShowdown = result
	t.resolvedRound = currentRound
	t.log.Debugf("handleShowdown: cached result with %d winners, totalPot=%d", len(result.WinnerInfo), result.TotalPot)

	tableID := t.config.ID
	amount := t.lastShowdown.TotalPot

	t.PublishEvent(pokerrpc.NotificationType_SHOWDOWN_RESULT, tableID, &pokerrpc.Showdown{
		Winners: t.lastShowdown.WinnerInfo,
		Pot:     amount,
	})

	// Remove busted players (0 chips) and count remaining players
	playersToRemove := make([]string, 0)
	snap := t.game.GetStateSnapshot()
	// Build quick lookup of balances by player ID
	balances := make(map[string]int64, len(snap.Players))
	for _, p := range snap.Players {
		if p != nil {
			balances[p.id] = p.balance
		}
	}

	t.mu.Lock()
	for _, u := range t.users {
		if balances[u.ID] == 0 {
			playersToRemove = append(playersToRemove, u.ID)
		}
	}

	// Check if the game should end BEFORE removing players
	// This ensures all players (including losing ones) get notified
	if t.shouldGameEnd() {
		t.log.Infof("Game should end, calling endGame()")
		t.endGame()
		t.mu.Unlock()
		return nil
	}

	// Remove busted players AFTER game ended notification
	for _, userID := range playersToRemove {
		t.log.Infof("Removing busted player %s (0 chips)", userID)
		t.removeUserWithoutLock(userID)
		t.log.Infof("removed busted player %s (0 chips)", userID)
	}

	// Reset round-local counters and update timestamp
	t.game.ResetActionsInRound()
	t.lastAction = time.Now()

	if t.config.AutoStartDelay == 0 {
		t.log.Debugf("Auto-start delay is 0, skipping auto-start")
	}

	// Schedule auto-start of the next hand strictly after showdown resolution
	if t.config.AutoStartDelay > 0 {
		t.log.Debugf("Scheduling auto-start for new hand with delay %v", t.config.AutoStartDelay)
		// Provide callbacks if not already set (check with game lock)
		if !t.game.HasAutoStartCallbacks() {
			t.game.SetAutoStartCallbacks(&AutoStartCallbacks{
				MinPlayers: func() int {
					remainingPlayers := len(t.users)
					if remainingPlayers >= 2 {
						return 2 // Allow heads-up play
					}
					return t.config.MinPlayers
				},
				StartNewHand:     func() error { return t.startNewHand() },
				OnNewHandStarted: nil,
			})
		}
		t.game.ScheduleAutoStart()
	}
	t.mu.Unlock()

	// Trigger state machine update after removing players (outside the lock)
	if len(playersToRemove) > 0 {
		t.sm.Send(evUsersChanged{})
	}

	return nil
}

// shouldGameEnd checks various conditions to determine if the game should end
func (t *Table) shouldGameEnd() bool {
	// Check if we have enough players to continue
	remainingPlayers := len(t.users)
	minRequired := t.config.MinPlayers
	if remainingPlayers >= 2 && remainingPlayers < t.config.MinPlayers {
		minRequired = 2 // Allow heads-up play
	}

	if remainingPlayers < minRequired {
		t.log.Infof("shouldGameEnd: Not enough players remaining (%d < %d)", remainingPlayers, minRequired)
		return true
	}

	// Check if any remaining players have sufficient chips to play
	playersWithChips := 0
	for _, u := range t.users {
		// Find player's current chip balance
		var playerBalance int64 = 0
		for _, player := range t.game.players {
			if player.id == u.ID {
				playerBalance = player.balance
				break
			}
		}

		if playerBalance > 0 {
			playersWithChips++
		}
	}

	if playersWithChips < 2 {
		t.log.Infof("shouldGameEnd: Not enough players with sufficient chips (%d < 2)", playersWithChips)
		return true
	}

	// Add more game ending conditions here as needed
	// For example:
	// - Tournament time limit reached
	// - Maximum hands played
	// - All players but one eliminated
	// - etc.

	return false
}

// endGame ends the current game and transitions to WAITING_FOR_PLAYERS state
func (t *Table) endGame() {
	t.log.Infof("Ending game - not enough players remaining")

	// Clear the game
	t.game = nil

	// Reset all players to not ready
	for _, u := range t.users {
		u.IsReady = false
	}

	// Transition back to WAITING_FOR_PLAYERS state
	t.sm.Send(evGameEnded{})

	// Publish game ended event
	t.PublishEvent(pokerrpc.NotificationType_GAME_ENDED, t.config.ID, map[string]interface{}{
		"reason": "Not enough players remaining",
	})

	t.log.Infof("Game ended, table back to WAITING_FOR_PLAYERS state")
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

	// (Optional) Log balances safely using a snapshot
	snap := g.GetStateSnapshot()
	byID := map[string]int64{}
	for _, p := range snap.Players {
		if p != nil {
			byID[p.id] = p.balance
		}
	}
	for _, u := range activeUsers {
		bal := byID[u.ID]
		if bal >= t.config.BigBlind {
			t.log.Debugf("User %s eligible for new hand: pokerBalance=%d >= bigBlind=%d", u.ID, bal, t.config.BigBlind)
		} else {
			t.log.Debugf("User %s will play all-in: pokerBalance=%d < bigBlind=%d", u.ID, bal, t.config.BigBlind)
		}
	}

	// Set up notification to broadcast NEW_HAND_STARTED when PRE_FLOP is reached.
	// Create the notification channel BEFORE triggering FSM transitions.
	preFlopCh := g.SetupPreFlopNotification()
	defer g.ClearPreFlopNotification()

	// Let the Game do all mutations under g.mu (reuse players, reset, send evStartHand).
	// This triggers the FSM: evStartHand → statePreDeal → stateDeal → stateBlinds → statePreFlop
	if err := g.ResetForNewHandFromUsers(activeUsers); err != nil {
		return fmt.Errorf("failed to setup new hand: %w", err)
	}

	// Deal hole cards for the new hand immediately so the next broadcast contains hands.
	if err := t.dealCardsToPlayers(activeUsers); err != nil {
		return fmt.Errorf("failed to deal initial cards for new hand: %w", err)
	}

	// Update table state / bookkeeping
	t.mu.Lock()
	t.lastShowdown = nil
	t.resolvedRound = -1
	t.sm.Send(evStartGameReq{})
	t.lastAction = time.Now()
	t.mu.Unlock()

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

// setupNewHand handles card dealing for the initial hand.
// All other setup (dealer advancement, blind posting, current player) is handled by the FSM.
func (t *Table) setupNewHand(activePlayers []*User) error {
	if t.game == nil {
		return fmt.Errorf("game not initialized")
	}
	if t.log == nil {
		t.log = slog.NewBackend(nil).Logger("TESTING")
	}

	t.log.Debugf("setupNewHand: Dealing cards to %d players", len(activePlayers))

	// Only deal cards here - the FSM (statePreDeal → stateBlinds) will handle:
	// - Dealer advancement
	// - Blind posting
	// - Current player initialization
	err := t.dealCardsToPlayers(activePlayers)
	if err != nil {
		return fmt.Errorf("failed to deal cards: %v", err)
	}

	t.log.Debugf("setupNewHand: Cards dealt, FSM will handle rest of setup")
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

	// Check if this action completes the betting round (outside table lock)
	t.MaybeCompleteBettingRound()
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
	// (Assumes lastAction preserves monotonic clock; if you copy it into the snapshot, keep the monotonic part.)
	deadline := cp.lastAction.Add(t.config.TimeBank)
	if time.Now().Before(deadline) {
		return
	}

	playerID := cp.id
	need := g.GetCurrentBet() - cp.currentBet

	// Decide from snapshot WITHOUT holding any locks, then perform exactly one mutating call.
	// Use table-level wrappers (Check/Fold/MakeBet) if they enforce your global lock order.
	if need <= 0 {
		// Auto-check (no chips required)
		_ = g.HandlePlayerCheck(playerID)
	} else {
		// Auto-fold when the player cannot check
		_ = g.HandlePlayerFold(playerID)
	}

	// Do NOT call g.MaybeCompleteBettingRound() here:
	// HandlePlayerCheck/Fold should already trigger state progression.
	// Calling it again risks re-entrancy and lock-order inversions.
}

// MaybeCompleteBettingRound delegates to Game layer for phase advancement logic
func (t *Table) MaybeCompleteBettingRound() error {
	if !t.isGameActive() || t.game == nil {
		return nil
	}

	// Compute actionable counts from a safe snapshot to avoid data races.
	snapshot := t.game.GetStateSnapshot()
	alivePlayers := 0
	activePlayers := 0
	for _, p := range snapshot.Players {
		if p == nil {
			continue
		}
		if p.GetCurrentStateString() != "FOLDED" {
			alivePlayers++
			if p.GetCurrentStateString() != "ALL_IN" {
				activePlayers++
			}
		}
	}

	var err error

	// Handle showdown if only 1 player remains or betting is effectively closed
	// (all remaining players are all-in or only one can act)
	shouldShowdown := alivePlayers <= 1 || activePlayers <= 1
	if shouldShowdown {
		// Step through missing streets synchronously to ensure showdown is completed
		// before returning from this method
		startPhase := t.game.GetPhase()
		tableID := t.config.ID
		ap := activePlayers
		al := alivePlayers

		// Refund any uncalled portion before we reset current bets by dealing
		// additional streets. This avoids creating invalid side pots that only
		// the all-in player is eligible to win.
		if localErr := t.game.RefundUncalledBets(); localErr != nil {
			t.log.Errorf("table.maybeAdvancePhase: failed to refund uncalled bets: %v", localErr)
		}

		// Only step through streets if we're not already in SHOWDOWN phase
		if startPhase != pokerrpc.GamePhase_SHOWDOWN {
			step := func(do func(), note string) {
				do()
				t.log.Debugf("table.maybeAdvancePhase: broadcast %s", note)
				t.PublishEvent(pokerrpc.NotificationType_NEW_ROUND, tableID, nil)
			}
			switch startPhase {
			case pokerrpc.GamePhase_PRE_FLOP:
				step(t.game.StateFlop, "FLOP")
				step(t.game.StateTurn, "TURN")
				step(t.game.StateRiver, "RIVER")
			case pokerrpc.GamePhase_FLOP:
				step(t.game.StateTurn, "TURN")
				step(t.game.StateRiver, "RIVER")
			case pokerrpc.GamePhase_TURN:
				step(t.game.StateRiver, "RIVER")
			}
		}
		t.log.Debugf("table.maybeAdvancePhase: betting closed (alive=%d active=%d), proceeding to SHOWDOWN with broadcasts", al, ap)
		// Proceed to showdown (phase will be set by game logic inside handleShowdown).
		err = t.handleShowdown()
		return err
	}

	// Otherwise, delegate to Game layer for normal progression
	phaseBefore := t.game.GetPhase()
	t.log.Debugf("table.maybeAdvancePhase: delegating (phase=%v actionsInRound=%d currentBet=%d)", phaseBefore, t.game.GetActionsInRound(), t.game.GetCurrentBet())
	t.game.maybeCompleteBettingRound()
	phaseAfter := t.game.GetPhase()

	// If phase changed, publish NEW_ROUND event to notify players
	if phaseBefore != phaseAfter && phaseAfter != pokerrpc.GamePhase_SHOWDOWN {
		t.log.Debugf("table.maybeAdvancePhase: phase changed from %v to %v, broadcasting NEW_ROUND", phaseBefore, phaseAfter)
		t.PublishEvent(pokerrpc.NotificationType_NEW_ROUND, t.config.ID, nil)
	}

	// Handle showdown if we reached that phase
	// Note: showdown is already handled in the goroutine above for fast-forward cases
	// For normal progression, we need to handle showdown here
	// REMOVED: Duplicate showdown call - showdown is handled in the goroutine above
	if phaseAfter == pokerrpc.GamePhase_SHOWDOWN {
		t.log.Debugf("table.maybeAdvancePhase: entering SHOWDOWN, handling showdown")
		err = t.handleShowdown()
		if err != nil {
			t.log.Errorf("table.maybeAdvancePhase: failed to handle showdown: %v", err)
			return err
		}
	}

	return err
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

// currentPlayerID returns the current player ID without acquiring locks (private helper)
func (t *Table) currentPlayerID() string {
	if t.game == nil {
		return ""
	}
	p := t.game.GetCurrentPlayerObject()
	if p == nil {
		return ""
	}
	return p.id
}

// advanceToNextPlayer delegates to Game layer
func (t *Table) advanceToNextPlayer(now time.Time) {
	if t.game == nil {
		return
	}
	t.game.AdvanceToNextPlayer(now)
}

// initializeCurrentPlayer delegates to Game layer
func (t *Table) initializeCurrentPlayer() {
	if t.game == nil {
		return
	}
	t.game.InitializeCurrentPlayer()
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
	curr := t.currentPlayerID()
	t.log.Debugf("HandleFold: userID=%s, currentPlayerID=%s, currentPlayer=%d, gamePhase=%v",
		userID, curr, t.game.GetCurrentPlayer(), t.game.GetPhase())
	if curr != userID {
		t.mu.Unlock()
		return fmt.Errorf("not your turn to act")
	}

	// Mutate and maybe-advance in one critical section.
	if err := t.game.handlePlayerFold(userID); err != nil {
		t.mu.Unlock()
		return err
	}
	t.lastAction = time.Now()

	t.mu.Unlock()

	// Check if this action completes the betting round (outside table lock)
	t.MaybeCompleteBettingRound()
	return nil
}

// HandleCall handles call actions by delegating to the Game layer
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

	curr := t.currentPlayerID()
	t.log.Debugf("HandleCall: userID=%s, currentPlayerID=%s, currentPlayer=%d, gamePhase=%v",
		userID, curr, t.game.GetCurrentPlayer(), t.game.GetPhase())
	if curr != userID {
		t.mu.Unlock()
		return fmt.Errorf("not your turn to act")
	}

	// Call the *internal* game logic while we still hold the table lock,
	// just like HandleFold does.
	if err := t.game.handlePlayerCall(userID); err != nil {
		t.mu.Unlock()
		return err
	}

	t.lastAction = time.Now()
	t.mu.Unlock()

	// Outside the table lock: may end the betting round / advance
	t.MaybeCompleteBettingRound()
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

	// Check if this action completes the betting round (outside table lock)
	err := t.MaybeCompleteBettingRound()
	if err != nil {
		t.log.Errorf("HandleCheck: failed to complete betting round: %v", err)
		return err
	}
	return nil
}

// postBlindsFromGame calls the game state machine logic to post blinds
func (t *Table) postBlindsFromGame() error {
	if t.game == nil {
		return fmt.Errorf("game not started")
	}

	numPlayers := len(t.game.players)
	if numPlayers < 2 {
		return fmt.Errorf("not enough players for blinds")
	}

	// Calculate blind positions
	smallBlindPos := (t.game.dealer + 1) % numPlayers
	bigBlindPos := (t.game.dealer + 2) % numPlayers

	// For heads-up (2 players), dealer posts small blind
	if numPlayers == 2 {
		smallBlindPos = t.game.dealer
		bigBlindPos = (t.game.dealer + 1) % numPlayers
	}

	t.log.Debugf("postBlindsFromGame: numPlayers=%d, dealer=%d, smallBlindPos=%d, bigBlindPos=%d",
		numPlayers, t.game.dealer, smallBlindPos, bigBlindPos)

	// Reset all player bets before posting blinds
	for _, player := range t.game.players {
		if player != nil {
			player.currentBet = 0
		}
	}

	// Post small blind
	if t.game.players[smallBlindPos] != nil {
		smallBlindAmount := t.game.config.SmallBlind
		player := t.game.players[smallBlindPos]

		// Handle all-in logic for small blind
		if smallBlindAmount > player.balance {
			// Player cannot cover small blind - treat as all-in of remaining balance
			smallBlindAmount = player.balance
			t.log.Debugf("Player %s all-in for small blind: posting %d (had %d)", player.id, smallBlindAmount, player.balance)
		}

		// Apply changes first, then set state accordingly
		player.balance -= smallBlindAmount
		player.currentBet = smallBlindAmount

		t.game.potManager.addBet(smallBlindPos, smallBlindAmount, t.game.players)

		// Send small blind notification
		// DISABLED: Notification callbacks cause deadlocks - server handles notifications directly
		// go t.eventManager.NotifyBlindPosted(t.config.ID, t.game.players[smallBlindPos].ID, smallBlindAmount, true)
	}

	// Post big blind
	if t.game.players[bigBlindPos] != nil {
		bigBlindAmount := t.game.config.BigBlind
		player := t.game.players[bigBlindPos]

		// Handle all-in logic for big blind
		if bigBlindAmount > player.balance {
			// Player cannot cover big blind - treat as all-in of remaining balance
			bigBlindAmount = player.balance
			t.log.Debugf("Player %s all-in for big blind: posting %d (had %d)", player.id, bigBlindAmount, player.balance)
		}

		// Apply changes first, then set state accordingly
		player.balance -= bigBlindAmount
		player.currentBet = bigBlindAmount

		t.game.potManager.addBet(bigBlindPos, bigBlindAmount, t.game.players)
		t.game.currentBet = bigBlindAmount // Set current bet to big blind amount

		// Send big blind notification
	}

	return nil
}

// dealCardsToPlayers deals cards to active players using the unified player state
func (t *Table) dealCardsToPlayers(activePlayers []*User) error {
	if t.game == nil || t.game.deck == nil {
		return fmt.Errorf("game or deck not initialized")
	}

	// Deal 2 cards to each active player
	for i := 0; i < 2; i++ {
		for _, u := range activePlayers {
			card, ok := t.game.deck.Draw()
			if !ok {
				return fmt.Errorf("failed to deal card to user %s: deck is empty", u.ID)
			}

			// Also sync the card to the corresponding game player
			found := false
			for _, player := range t.game.players {
				if player.id == u.ID {
					player.hand = append(player.hand, card)
					found = true
					break
				}
			}

			if !found {
				t.log.Debugf("DEBUG: Could not find game player for user %s when dealing cards", u.ID)
			} else {

			}
		}
	}
	return nil
}

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

// removeUserWithoutLock removes a user from the table without acquiring the lock
// This is used internally when the caller already holds the table lock
func (t *Table) removeUserWithoutLock(userID string) error {
	if _, exists := t.users[userID]; !exists {
		return fmt.Errorf("user not at table")
	}

	delete(t.users, userID)
	t.lastAction = time.Now()
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
	t.mu.RLock()
	defer t.mu.RUnlock()

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

	// Sort by TableSeat to ensure consistent ordering
	sort.Slice(usersCopy, func(i, j int) bool {
		return usersCopy[i].TableSeat < usersCopy[j].TableSeat
	})

	// Get game state snapshot if game is active
	var gameSnapshot *GameStateSnapshot
	if t.game != nil {
		snapshot := t.game.GetStateSnapshot()
		gameSnapshot = &snapshot
	}

	return TableStateSnapshot{
		Config:      t.config,
		Users:       usersCopy,
		GameStarted: t.game != nil,
		GamePhase:   t.getGamePhase(),
		Game:        gameSnapshot,
	}
}

// getGamePhase returns the current phase without acquiring locks (private helper)
func (t *Table) getGamePhase() pokerrpc.GamePhase {
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
