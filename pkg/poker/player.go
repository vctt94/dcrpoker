package poker

import (
	"context"
	"fmt"
	"time"

	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	"github.com/vctt94/pokerbisonrelay/pkg/statemachine"
)

type Player struct {
	mu RWLock
	// identity
	id, name string

	// table-level
	tableSeat      int
	isReady        bool
	isDisconnected bool
	lastAction     time.Time

	// Durable attributes (persist across hands)
	balance         int64
	startingBalance int64

	// Per-hand state flags (set by FSM, cleared after settlement)
	// Hole cards are NOT stored here - they live in Game.currentHand.hole
	hasFolded  bool  // Set when player folds
	isAllIn    bool  // Set when player goes all-in
	currentBet int64 // Current bet in this round, reset at hand start

	// Per-hand role flags (reset each hand)
	isDealer     bool
	isSmallBlind bool
	isBigBlind   bool
	isTurn       bool

	// NEW: Separated state machines
	tablePresence     *statemachine.Machine[Player]
	handParticipation *statemachine.Machine[Player]

	// showdown info
	handValue       *HandValue
	handDescription string
}

func NewPlayer(id, name string, balance int64) *Player {
	p := &Player{
		mu:              RWLock{},
		id:              id,
		name:            name,
		balance:         balance,
		startingBalance: balance,
		tableSeat:       -1,
		lastAction:      time.Now(),
	}

	// Initialize new separated state machines
	p.tablePresence = statemachine.New(p, stateTableSeated, 32)
	p.tablePresence.Start(context.Background())

	// Hand participation FSM starts as nil (no hand active)

	return p
}

func (p *Player) Close() {
	// Grab references to state machines while holding lock
	p.mu.Lock()
	tablePresence := p.tablePresence
	handParticipation := p.handParticipation
	p.mu.Unlock()

	// Stop state machines without holding player mutex to avoid deadlock
	// (state machines may need to acquire p.mu during shutdown)
	if tablePresence != nil {
		tablePresence.Stop()
	}
	if handParticipation != nil {
		handParticipation.Stop()
	}
}

// StartHandParticipation starts the hand participation FSM
// We use it for testing purposes.
// This should not be called by the game, because it will make FSM fail when going to deal state
func (p *Player) StartHandParticipation() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.handParticipation != nil {
		return fmt.Errorf("hand participation already active")
	}

	p.handParticipation = statemachine.New(p, stateHandActive, 32)
	p.handParticipation.Start(context.Background())
	return nil
}

// HandleStartHand starts hand participation and determines initial state
func (p *Player) HandleStartHand() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.handParticipation != nil {
		return fmt.Errorf("hand participation already active")
	}

	p.startingBalance = p.balance
	isAllIn := (p.balance == 0 && p.currentBet > 0)

	// If player is already all-in from posting blinds, go directly to ALL_IN state
	if isAllIn {
		p.handParticipation = statemachine.New(p, stateHandAllIn, 32)
	} else {
		p.handParticipation = statemachine.New(p, stateHandActive, 32)
	}

	p.handParticipation.Start(context.Background())
	return nil
}

// HandlePostBlind posts a blind and returns the amount actually posted
func (p *Player) HandlePostBlind(amount int64) int64 {
	p.mu.RLock()
	tp := p.tablePresence
	p.mu.RUnlock()

	if tp == nil {
		return 0
	}

	reply := make(chan int64, 1)
	tp.Send(evDeductBlind{Amt: amount, Reply: reply})
	return <-reply
}

// -------------------------- State functions --------------------------

func stateAtTable(p *Player, in <-chan any) PlayerStateFn {
	p.mu.Lock()
	p.isTurn = false
	p.mu.Unlock()

	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			timer.Reset(30 * time.Second)

		case ev, ok := <-in:
			if !ok {
				return nil
			}
			switch e := ev.(type) {
			case evReady:
				p.mu.Lock()
				p.isReady = true
				p.mu.Unlock()

			case evDeductBlind:
				// Deduct blind while still AT_TABLE (before hand starts)
				p.mu.Lock()
				amount := e.Amt
				if amount > p.balance {
					amount = p.balance
				}
				if amount > 0 {
					p.balance -= amount
					p.currentBet += amount
					p.lastAction = time.Now()
				}
				p.mu.Unlock()
				// Reply with actual amount deducted
				if e.Reply != nil {
					e.Reply <- amount
				}
			// Stay in AT_TABLE; evStartHand will handle transition to IN_GAME or ALL_IN

			case evStartHand:
				p.mu.Lock()
				p.startingBalance = p.balance
				isAllIn := (p.balance == 0 && p.currentBet > 0)
				p.mu.Unlock()
				// If player is already all-in from posting blinds, go directly to ALL_IN state
				if isAllIn {
					return stateAllIn
				}
				return stateInGame

			case evStartTurn:
				// Ack start-turn requests while AT_TABLE to avoid blocking callers
				if e.Reply != nil {
					e.Reply <- nil
				}
				// Stay in AT_TABLE; players are not in-hand yet

			case evEndTurn:
				// Ack end-turn requests while AT_TABLE to avoid blocking callers
				if e.Reply != nil {
					e.Reply <- nil
				}
				// Stay in AT_TABLE

			case evCallDelta: // allow call while FSM is still AT_TABLE
				p.mu.Lock()
				can := e.Amt > 0 && e.Amt <= p.balance
				var nextState PlayerState
				var err error
				if can {
					p.balance -= e.Amt
					p.currentBet += e.Amt
					p.lastAction = time.Now()
					// Atomically update stateID alongside balance/currentBet
					if p.balance == 0 && p.currentBet > 0 {
						return stateAllIn
					} else {
						return stateInGame
					}
				} else {
					err = fmt.Errorf("invalid call amount or insufficient balance")
				}
				p.mu.Unlock()

				// Send reply if channel provided
				if e.Reply != nil {
					e.Reply <- err
				}

				if can {
					// Return appropriate state function (which will also set stateID on entry)
					if nextState == psAllIn {
						return stateAllIn
					}
					return stateInGame
				}
			// invalid or ignored, stay at table

			case evBalanceNotification:
				// Async notification from Game FSM - balance already changed
				// Can be used for UI updates, logging, etc.
				// No action needed here

			case evDisconnect:
				p.mu.Lock()
				p.isDisconnected = true
				p.mu.Unlock()
			case evLeave:
				return stateLeft
			default:
				// ignore others
			}
		}
	}
}

func stateInGame(p *Player, in <-chan any) PlayerStateFn {

	for ev := range in {
		// Derived ALL-IN: evaluated each loop iteration (under read lock)
		p.mu.RLock()
		zeroAllIn := (p.balance == 0 && p.currentBet > 0)
		p.mu.RUnlock()
		if zeroAllIn {
			return stateAllIn
		}

		switch e := ev.(type) {
		case evStartTurn:
			p.mu.Lock()
			p.isTurn = true
			p.lastAction = time.Now()
			p.mu.Unlock()
			// Send reply if channel provided
			if e.Reply != nil {
				e.Reply <- nil
			}

		case evEndTurn:
			p.mu.Lock()
			p.isTurn = false
			p.mu.Unlock()
			// Send reply if channel provided
			if e.Reply != nil {
				e.Reply <- nil
			}

		case evBet:
			p.mu.Lock()
			can := p.isTurn && e.Amt > 0 && e.Amt <= p.balance
			shouldTransition := false
			var err error
			if can {
				p.balance -= e.Amt
				p.currentBet += e.Amt
				p.lastAction = time.Now()
				// Note: isTurn is cleared by evEndTurn, not here
				// Atomically update stateID if going all-in
				if p.balance == 0 && p.currentBet > 0 {
					shouldTransition = true
				}
			} else {
				err = fmt.Errorf("invalid bet: not your turn, invalid amount, or insufficient balance")
			}
			p.mu.Unlock()

			// Send reply if channel provided
			if e.Reply != nil {
				e.Reply <- err
			}

			if shouldTransition {
				return stateAllIn
			}

		case evCall:
			// Amount computed by table; state only observes results.
			p.mu.Lock()
			p.lastAction = time.Now()
			// Note: isTurn is cleared by evEndTurn, not here
			zero := (p.balance == 0 && p.currentBet > 0)
			p.mu.Unlock()
			if zero {
				return stateAllIn
			}

		case evFoldReq:
			// Fold and acknowledge to caller
			p.mu.Lock()
			p.lastAction = time.Now()
			// Note: isTurn is cleared by evEndTurn, not here
			p.mu.Unlock()
			if e.Reply != nil {
				e.Reply <- nil
			}
			return stateFolded

		case evBalanceNotification:
			// Async notification from Game FSM - balance already changed
			// Can be used for UI updates, logging, etc.
			// No action needed here

		case evEndHand:
			p.mu.Lock()
			p.currentBet = 0
			p.isTurn = false
			p.mu.Unlock()
			return stateAtTable

		case evDisconnect:
			p.mu.Lock()
			p.isDisconnected = true
			p.mu.Unlock()

		case evLeave:
			return stateLeft

		case evCallDelta:
			p.mu.Lock()
			can := e.Amt > 0 && e.Amt <= p.balance
			var nextState PlayerState
			var err error
			if can {
				p.balance -= e.Amt
				p.currentBet += e.Amt
				p.lastAction = time.Now()
				if p.balance == 0 && p.currentBet > 0 {
					return stateAllIn
				} else {
					return stateInGame
					// stateID already IN_GAME, no update needed
				}
			} else {
				err = fmt.Errorf("invalid call amount or insufficient balance")
			}
			p.mu.Unlock()

			// Send reply if channel provided
			if e.Reply != nil {
				e.Reply <- err
			}

			if can {
				if nextState == psAllIn {
					return stateAllIn
				}
				// State remains IN_GAME
				return stateInGame
			}
		}
	}
	return nil
}

func stateAllIn(p *Player, in <-chan any) PlayerStateFn {
	p.mu.Lock()
	p.isTurn = false
	p.mu.Unlock()

	for ev := range in {
		switch e := ev.(type) {
		case evStartTurn:
			// All-in players cannot act; acknowledge to avoid blocking senders
			if e.Reply != nil {
				e.Reply <- nil
			}
		case evEndTurn:
			// All-in players can still receive EndTurn events (e.g., during phase transitions)
			p.mu.Lock()
			p.isTurn = false
			p.mu.Unlock()
			// Send reply if channel provided
			if e.Reply != nil {
				e.Reply <- nil
			}

		case evBalanceNotification:
			// Async notification from Game FSM - balance already changed
			// Can be used for UI updates, logging, etc.
			// No action needed here
		case evFoldReq:
			// Cannot fold while all-in; acknowledge with error if requested.
			if e.Reply != nil {
				e.Reply <- fmt.Errorf("cannot fold while all-in")
			}
		case evEndHand:
			p.mu.Lock()
			p.currentBet = 0
			p.mu.Unlock()
			return stateAtTable
		case evLeave:
			return stateLeft
		}
	}
	return nil
}

func stateFolded(p *Player, in <-chan any) PlayerStateFn {
	p.mu.Lock()
	p.isTurn = false
	p.mu.Unlock()

	for ev := range in {
		switch e := ev.(type) {
		case evStartTurn:
			// Folded players cannot act; acknowledge to avoid blocking senders
			if e.Reply != nil {
				e.Reply <- nil
			}
		case evEndTurn:
			// Folded players can still receive EndTurn events (e.g., during phase transitions)
			p.mu.Lock()
			p.isTurn = false
			p.mu.Unlock()
			// Send reply if channel provided
			if e.Reply != nil {
				e.Reply <- nil
			}

		case evBalanceNotification:
			// Async notification from Game FSM - balance already changed
			// Can be used for UI updates, logging, etc.
			// No action needed here
		case evEndHand:
			p.mu.Lock()
			p.currentBet = 0
			p.mu.Unlock()
			return stateAtTable
		case evLeave:
			return stateLeft
		}
	}
	return nil
}

func stateLeft(p *Player, _ <-chan any) PlayerStateFn {
	return nil // terminal
}

// ===== NEW TABLE PRESENCE STATE FUNCTIONS =====

func stateTableSeated(p *Player, in <-chan any) TablePresenceStateFn {
	p.mu.Lock()
	p.isTurn = false
	p.mu.Unlock()

	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			timer.Reset(30 * time.Second)

		case ev, ok := <-in:
			if !ok {
				return nil
			}
			switch e := ev.(type) {
			case evReady:
				p.mu.Lock()
				p.isReady = true
				p.mu.Unlock()

			case evDeductBlind:
				// Deduct blind while still at table (before hand participation starts)
				p.mu.Lock()
				amount := e.Amt
				if amount > p.balance {
					amount = p.balance
				}
				if amount > 0 {
					p.balance -= amount
					p.currentBet += amount
					p.lastAction = time.Now()
				}
				p.mu.Unlock()
				// Reply with actual amount deducted
				if e.Reply != nil {
					e.Reply <- amount
				}
				// Stay in table seated state; hand participation will start separately

			case evBalanceNotification:
				// Async notification from Game FSM - balance already changed
				// Can be used for UI updates, logging, etc.
				// No action needed here

			case evDisconnect:
				p.mu.Lock()
				p.isDisconnected = true
				p.mu.Unlock()

			case evLeave:
				return stateTableLeft

			default:
				// Forward hand participation events to hand FSM if it exists
				p.mu.RLock()
				hp := p.handParticipation
				p.mu.RUnlock()
				if hp != nil {
					hp.Send(ev)
				}
			}
		}
	}
}

func stateTableLeft(p *Player, _ <-chan any) TablePresenceStateFn {
	return nil // terminal
}

// ===== NEW HAND PARTICIPATION STATE FUNCTIONS =====

func stateHandActive(p *Player, in <-chan any) HandParticipationStateFn {
	for ev := range in {
		// Derived ALL-IN: evaluated each loop iteration (under read lock)
		p.mu.RLock()
		zeroAllIn := (p.balance == 0 && p.currentBet > 0)
		p.mu.RUnlock()
		if zeroAllIn {
			return stateHandAllIn
		}

		switch e := ev.(type) {
		case evStartTurn:
			p.mu.Lock()
			p.isTurn = true
			p.lastAction = time.Now()
			p.mu.Unlock()
			// Send reply if channel provided
			if e.Reply != nil {
				e.Reply <- nil
			}

		case evEndTurn:
			p.mu.Lock()
			p.isTurn = false
			p.mu.Unlock()
			// Send reply if channel provided
			if e.Reply != nil {
				e.Reply <- nil
			}

		case evBet:
			p.mu.Lock()
			can := p.isTurn && e.Amt > 0 && e.Amt <= p.balance
			shouldTransition := false
			var err error
			if can {
				p.balance -= e.Amt
				p.currentBet += e.Amt
				p.lastAction = time.Now()
				// Note: isTurn is cleared by evEndTurn, not here
				// Atomically update stateID if going all-in
				if p.balance == 0 && p.currentBet > 0 {
					p.isAllIn = true // Set flag
					shouldTransition = true
				}
			} else {
				err = fmt.Errorf("invalid bet: not your turn, invalid amount, or insufficient balance")
			}
			p.mu.Unlock()

			// Send reply if channel provided
			if e.Reply != nil {
				e.Reply <- err
			}

			if shouldTransition {
				return stateHandAllIn
			}

		case evCall:
			// Amount computed by table; state only observes results.
			p.mu.Lock()
			p.lastAction = time.Now()
			// Note: isTurn is cleared by evEndTurn, not here
			zero := (p.balance == 0 && p.currentBet > 0)
			if zero {
				p.isAllIn = true // Set flag
			}
			p.mu.Unlock()
			if zero {
				return stateHandAllIn
			}

		case evFoldReq:
			// Fold and acknowledge to caller
			p.mu.Lock()
			p.lastAction = time.Now()
			p.hasFolded = true // Set flag
			// Note: isTurn is cleared by evEndTurn, not here
			p.mu.Unlock()
			if e.Reply != nil {
				e.Reply <- nil
			}
			return stateHandFolded

		case evEndHand:
			p.mu.Lock()
			p.currentBet = 0
			p.isTurn = false
			p.mu.Unlock()
			return nil // Hand participation ends, FSM stops

		case evCallDelta:
			p.mu.Lock()
			can := e.Amt > 0 && e.Amt <= p.balance
			var nextState HandParticipationState
			var err error
			if can {
				p.balance -= e.Amt
				p.currentBet += e.Amt
				p.lastAction = time.Now()
				if p.balance == 0 && p.currentBet > 0 {
					p.isAllIn = true // Set flag
					nextState = hpAllIn
				} else {
					nextState = hpActive
				}
			} else {
				err = fmt.Errorf("invalid call amount or insufficient balance")
			}
			p.mu.Unlock()

			// Send reply if channel provided
			if e.Reply != nil {
				e.Reply <- err
			}

			if can {
				if nextState == hpAllIn {
					return stateHandAllIn
				}
				// State remains ACTIVE
				return stateHandActive
			}
		}
	}
	return nil
}

func stateHandAllIn(p *Player, in <-chan any) HandParticipationStateFn {
	p.mu.Lock()
	p.isTurn = false
	p.mu.Unlock()

	for ev := range in {
		switch e := ev.(type) {
		case evStartTurn:
			// All-in players cannot act; acknowledge to avoid blocking senders
			if e.Reply != nil {
				e.Reply <- nil
			}
		case evEndTurn:
			// All-in players can still receive EndTurn events (e.g., during phase transitions)
			p.mu.Lock()
			p.isTurn = false
			p.mu.Unlock()
			// Send reply if channel provided
			if e.Reply != nil {
				e.Reply <- nil
			}
		case evFoldReq:
			// Cannot fold while all-in; acknowledge with error if requested.
			if e.Reply != nil {
				e.Reply <- fmt.Errorf("cannot fold while all-in")
			}
		case evCallDelta:
			// Cannot call while all-in; acknowledge with error if requested.
			if e.Reply != nil {
				e.Reply <- fmt.Errorf("cannot call while all-in")
			}
		case evEndHand:
			p.mu.Lock()
			p.currentBet = 0
			p.mu.Unlock()
			return nil // Hand participation ends, FSM stops
		}
	}
	return nil
}

func stateHandFolded(p *Player, in <-chan any) HandParticipationStateFn {
	p.mu.Lock()
	p.isTurn = false
	p.mu.Unlock()

	for ev := range in {
		switch e := ev.(type) {
		case evStartTurn:
			// Folded players cannot act; acknowledge to avoid blocking senders
			if e.Reply != nil {
				e.Reply <- nil
			}
		case evEndTurn:
			// Folded players can still receive EndTurn events (e.g., during phase transitions)
			p.mu.Lock()
			p.isTurn = false
			p.mu.Unlock()
			// Send reply if channel provided
			if e.Reply != nil {
				e.Reply <- nil
			}
		case evCallDelta:
			// Cannot call while folded; acknowledge with error if requested.
			if e.Reply != nil {
				e.Reply <- fmt.Errorf("cannot call while folded")
			}
		case evEndHand:
			p.mu.Lock()
			p.currentBet = 0
			p.mu.Unlock()
			return nil // Hand participation ends, FSM stops
		}
	}
	return nil
}

// ResetForNewHand prepares the player for a new hand.
// This clears per-hand flags that persisted through showdown.
func (p *Player) ResetForNewHand(startingChips int64) error {
	p.mu.RLock()
	tp := p.tablePresence
	p.mu.RUnlock()

	if tp == nil {
		return fmt.Errorf("player state machine not initialized")
	}

	// Set stack for the new hand (table-sourced).
	// Clear per-hand flags (these persisted through showdown, now reset for new hand)
	p.mu.Lock()
	p.balance = startingChips
	p.startingBalance = startingChips
	p.lastAction = time.Now()
	// Clear per-hand state flags that persisted through showdown
	p.hasFolded = false
	p.isAllIn = false
	p.currentBet = 0
	p.handDescription = ""
	// Stop and clear the hand participation state machine if active
	if p.handParticipation != nil {
		p.handParticipation.Stop()
		p.handParticipation = nil
	}
	p.mu.Unlock()

	// Player remains seated at table (table presence unchanged)
	// Hand participation will start when Game FSM sends evStartHand
	return nil
}

// GetPlayerStateString converts a playerState to its string representation
func GetPlayerStateString(state PlayerState) string {
	switch state {
	case psAtTable:
		return "AT_TABLE"
	case psInGame:
		return "IN_GAME"
	case psAllIn:
		return "ALL_IN"
	case psFolded:
		return "FOLDED"
	case psLeft:
		return "LEFT"
	default:
		return "UNINITIALIZED"
	}
}

// Marshal converts the Player to gRPC Player for external access.
// Uses the fast state snapshot to derive Folded/AllIn booleans.
// Note: This does NOT include hole cards - cards must be fetched separately from
// Game.currentHand.GetPlayerCards() with proper visibility rules applied by the caller.
func (p *Player) Marshal() *pokerrpc.Player {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Use new state information for more accurate representation
	handStateStr := p.getCurrentStateString()

	proto := &pokerrpc.Player{
		Id:              p.id,
		Name:            p.name,
		Balance:         p.balance,
		Hand:            nil, // Cards stored in Game.currentHand, not in Player
		CurrentBet:      p.currentBet,
		Folded:          handStateStr == "FOLDED",
		IsTurn:          p.isTurn,
		IsAllIn:         handStateStr == "ALL_IN",
		IsDealer:        p.isDealer,
		IsSmallBlind:    p.isSmallBlind,
		IsBigBlind:      p.isBigBlind,
		IsReady:         p.isReady,
		HandDescription: p.handDescription,
		PlayerState:     p.getTablePresenceState(),
	}

	return proto
}

// Unmarshal updates the Player from a gRPC mirror.
//
// WARNING: This mutates fields while the Pike machine may be running. In production,
// prefer sending events instead of bulk overriding. This is typically used on
// client-side mirrors (no local machine running). Keep it if you need that.
func (p *Player) Unmarshal(grpcPlayer *pokerrpc.Player) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.id = grpcPlayer.Id
	p.name = grpcPlayer.Name
	p.balance = grpcPlayer.Balance
	p.currentBet = grpcPlayer.CurrentBet
	p.isTurn = grpcPlayer.IsTurn
	p.isDealer = grpcPlayer.IsDealer
	p.isSmallBlind = grpcPlayer.IsSmallBlind
	p.isBigBlind = grpcPlayer.IsBigBlind
	p.isReady = grpcPlayer.IsReady
	p.handDescription = grpcPlayer.HandDescription
	// Cards are not stored in Player - they're in Game.currentHand.hole
}

func (p *Player) HandDescription() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.handDescription
}

func (p *Player) ID() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.id
}

// Name returns the player's display name
func (p *Player) Name() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.name
}

func (p *Player) Balance() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.balance
}

func (p *Player) IsReady() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.isReady
}

// GetTableSeat returns the table seat (for external access)
func (p *Player) TableSeat() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.tableSeat
}

// GetStartingBalance returns the starting balance (for external access)
func (p *Player) StartingBalance() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.startingBalance
}

// GetIsDisconnected returns the disconnected state (for external access)
func (p *Player) IsDisconnected() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.isDisconnected
}

// GetCurrentBet returns the current bet (for external access)
func (p *Player) CurrentBet() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.currentBet
}

// SetTableSeat sets the table seat (for external access)
func (p *Player) SetTableSeat(seat int) {
	p.mu.Lock()
	p.tableSeat = seat
	p.mu.Unlock()
}

// SetStartingBalance sets the starting balance (for external access)
func (p *Player) SetStartingBalance(balance int64) {
	p.mu.Lock()
	p.startingBalance = balance
	p.mu.Unlock()
}

// SetBalance sets the player's balance (for testing only)
func (p *Player) SetBalance(balance int64) {
	p.mu.Lock()
	p.balance = balance
	p.mu.Unlock()
}

// SetCurrentBet sets the player's current bet (for testing only)
func (p *Player) SetCurrentBet(bet int64) {
	p.mu.Lock()
	p.currentBet = bet
	p.mu.Unlock()
}

// NotifyBalanceChange sends an async notification to the Player FSM about a balance change.
// This is called by the Game FSM after directly modifying player.balance.
// The notification is non-blocking - it's for UI/logging purposes only.
func (p *Player) NotifyBalanceChange(newBalance, delta int64, reason string) {
	p.mu.RLock()
	tp := p.tablePresence
	p.mu.RUnlock()

	if tp != nil {
		// Non-blocking send using the statemachine's Send method
		tp.Send(evBalanceNotification{
			NewBalance: newBalance,
			Delta:      delta,
			Reason:     reason,
		})
	}
}

func (p *Player) StartTurn() {
	// Send to hand participation FSM if active, otherwise to table presence FSM
	p.mu.RLock()
	hp := p.handParticipation
	tp := p.tablePresence
	p.mu.RUnlock()

	if hp != nil {
		reply := make(chan error, 1)
		hp.Send(evStartTurn{Reply: reply})
		<-reply // Wait for FSM to process
	} else if tp != nil {
		reply := make(chan error, 1)
		tp.Send(evStartTurn{Reply: reply})
		<-reply // Wait for FSM to process
	}
}

// For debugging
func (p *Player) IsTurn() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.isTurn
}

func (p *Player) EndTurn() {
	// Send to hand participation FSM if active, otherwise to table presence FSM
	p.mu.RLock()
	hp := p.handParticipation
	tp := p.tablePresence
	p.mu.RUnlock()

	if hp != nil {
		reply := make(chan error, 1)
		hp.Send(evEndTurn{Reply: reply})
		<-reply // Wait for FSM to process
	} else if tp != nil {
		reply := make(chan error, 1)
		tp.Send(evEndTurn{Reply: reply})
		<-reply // Wait for FSM to process
	}
}

// Marker interface for player events (optional, for readability).

type evCallDelta struct {
	Amt   int64
	Reply chan<- error // Optional reply channel for synchronous confirmation
}

type evDeductBlind struct {
	Amt   int64
	Reply chan<- int64 // Reply with the amount actually deducted (for synchronous confirmation)
}

type evReady struct{}

type evStartHand struct{}

type evStartTurn struct {
	Reply chan<- error // Optional reply channel for synchronous confirmation
}

type evEndTurn struct {
	Reply chan<- error // Optional reply channel for synchronous confirmation
}

type evBet struct {
	Amt   int64
	Reply chan<- error // Optional reply channel for synchronous confirmation
}

type evCall struct{}

// evBalanceNotification is an async notification sent to Player FSM when
// the Game FSM directly modifies player.balance (e.g., during pot distribution).
// This is for UI/logging purposes only - the balance change has already occurred.
type evBalanceNotification struct {
	NewBalance int64
	Delta      int64
	Reason     string // "pot_win", "refund", etc.
}

// evFoldReq requests a fold with an acknowledgment once the player's state
// has transitioned to FOLDED. This allows callers to avoid racing against
// reads that depend on the folded state.
type evFoldReq struct{ Reply chan<- error }

type evEndHand struct{}

type evDisconnect struct{}

type evLeave struct{}

type PlayerStateFn = statemachine.StateFn[Player]

type PlayerState int32

const (
	psUninitialized PlayerState = iota
	psAtTable
	psInGame
	psAllIn
	psFolded
	psLeft
)

// ===== NEW SEPARATED STATE MACHINES =====

// Table Presence States
type TablePresenceState int32

const (
	tpUninitialized TablePresenceState = iota
	tpSeated
	tpLeft
)

// Hand Participation States
type HandParticipationState int32

const (
	hpNone HandParticipationState = iota
	hpActive
	hpFolded
	hpAllIn
)

// GetCurrentStateString returns a string representation of the current player state
func (p *Player) GetCurrentStateString() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.getCurrentStateString()
}

// getCurrentStateStringLocked returns the current state string without acquiring locks
// (assumes caller already holds p.mu.RLock or p.mu.Lock)
func (p *Player) getCurrentStateString() string {
	mustHeld(&p.mu)
	// Check per-hand flags FIRST - these persist through showdown even if FSM stops
	// This is the single source of truth for fold/all-in status
	if p.hasFolded {
		return "FOLDED"
	}
	if p.isAllIn {
		return "ALL_IN"
	}

	// If hand participation is active, return hand participation state
	if p.handParticipation != nil {
		currentState := p.handParticipation.Current()
		if currentState == nil {
			return "TERMINATED"
		}

		// Use function pointer comparison to determine state
		switch fmt.Sprintf("%p", currentState) {
		case fmt.Sprintf("%p", stateHandActive):
			return "IN_GAME"
		case fmt.Sprintf("%p", stateHandFolded):
			return "FOLDED" // redundant now, but kept for compatibility
		case fmt.Sprintf("%p", stateHandAllIn):
			return "ALL_IN" // redundant now, but kept for compatibility
		default:
			return "UNKNOWN"
		}
	}

	// If no hand participation, return table presence state
	if p.tablePresence != nil {
		currentState := p.tablePresence.Current()
		if currentState == nil {
			return "TERMINATED"
		}

		// Use function pointer comparison to determine state
		switch fmt.Sprintf("%p", currentState) {
		case fmt.Sprintf("%p", stateTableSeated):
			return "AT_TABLE"
		case fmt.Sprintf("%p", stateTableLeft):
			return "LEFT"
		default:
			return "UNKNOWN"
		}
	}

	return "NONE"
}

// GetTablePresenceState returns the current table presence state as a pokerrpc.PlayerState enum
func (p *Player) GetTablePresenceState() pokerrpc.PlayerState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.getTablePresenceState()
}

// getTablePresenceStateLocked returns the table presence state
func (p *Player) getTablePresenceState() pokerrpc.PlayerState {
	mustHeld(&p.mu)
	if p.tablePresence == nil {
		return pokerrpc.PlayerState_PLAYER_STATE_UNINITIALIZED
	}

	currentState := p.tablePresence.Current()
	if currentState == nil {
		return pokerrpc.PlayerState_PLAYER_STATE_UNINITIALIZED
	}

	// Use function pointer comparison to determine state
	switch fmt.Sprintf("%p", currentState) {
	case fmt.Sprintf("%p", stateTableSeated):
		return pokerrpc.PlayerState_PLAYER_STATE_AT_TABLE
	case fmt.Sprintf("%p", stateTableLeft):
		return pokerrpc.PlayerState_PLAYER_STATE_LEFT
	default:
		return pokerrpc.PlayerState_PLAYER_STATE_UNINITIALIZED
	}
}

// State function types
type TablePresenceStateFn = statemachine.StateFn[Player]
type HandParticipationStateFn = statemachine.StateFn[Player]
