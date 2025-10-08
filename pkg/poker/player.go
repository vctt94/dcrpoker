package poker

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	"github.com/vctt94/pokerbisonrelay/pkg/statemachine"
)

type Player struct {
	mu sync.RWMutex
	// identity
	id, name string

	// table-level
	tableSeat      int
	isReady        bool
	isDisconnected bool
	lastAction     time.Time

	// hand-level (reset each hand)
	balance         int64
	startingBalance int64
	hand            []Card
	currentBet      int64
	isDealer        bool
	isSmallBlind    bool
	isBigBlind      bool
	isTurn          bool

	// Pike state machine
	sm *statemachine.Machine[Player]

	// fast snapshot for cheap reads
	stateID atomic.Int32

	// showdown info
	handValue       *HandValue
	handDescription string
}

func NewPlayer(id, name string, balance int64) *Player {
	p := &Player{
		mu:              sync.RWMutex{},
		id:              id,
		name:            name,
		balance:         balance,
		startingBalance: balance,
		tableSeat:       -1,
		hand:            make([]Card, 0, 2),
		lastAction:      time.Now(),
	}
	p.stateID.Store(int32(psAtTable))
	p.sm = statemachine.New(p, stateAtTable, 32)
	p.sm.Start(context.Background())
	return p
}

func (p *Player) Close() { p.sm.Stop() }

// ------------ Player public API (thread-safe; only sends events) ------------

func (p *Player) StateID() PlayerState { return PlayerState(p.stateID.Load()) }
func (p *Player) ProtoState() pokerrpc.PlayerState {
	switch p.StateID() {
	case psAtTable:
		return pokerrpc.PlayerState_PLAYER_STATE_AT_TABLE
	case psInGame:
		return pokerrpc.PlayerState_PLAYER_STATE_IN_GAME
	case psAllIn:
		return pokerrpc.PlayerState_PLAYER_STATE_ALL_IN
	case psFolded:
		return pokerrpc.PlayerState_PLAYER_STATE_FOLDED
	case psLeft:
		return pokerrpc.PlayerState_PLAYER_STATE_LEFT
	default:
		return pokerrpc.PlayerState_PLAYER_STATE_UNINITIALIZED
	}
}

// -------------------------- State functions --------------------------

func stateAtTable(p *Player, in <-chan any) PlayerStateFn {
	p.mu.Lock()
	p.isTurn = false
	p.mu.Unlock()
	p.stateID.Store(int32(psAtTable))

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

			case evPostBlind:
				// Post blind while still AT_TABLE (before hand starts)
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
				// Reply with actual amount posted
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
						nextState = psAllIn
						p.stateID.Store(int32(psAllIn))
					} else {
						nextState = psInGame
						p.stateID.Store(int32(psInGame))
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

			case evCredit:
				var err error
				if e.Amt > 0 {
					p.mu.Lock()
					p.balance += e.Amt
					p.mu.Unlock()
				} else if e.Amt < 0 {
					err = fmt.Errorf("credit amount must be positive")
				}
				// Send reply if channel provided
				if e.Reply != nil {
					e.Reply <- err
				}

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
	p.stateID.Store(int32(psInGame))

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
					p.stateID.Store(int32(psAllIn))
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

		case evFold:
			p.mu.Lock()
			p.lastAction = time.Now()
			// Note: isTurn is cleared by evEndTurn, not here
			p.mu.Unlock()
			return stateFolded

		case evFoldReq:
			// Same as evFold, but acknowledge to caller.
			p.mu.Lock()
			p.lastAction = time.Now()
			// Note: isTurn is cleared by evEndTurn, not here
			p.mu.Unlock()
			if e.Reply != nil {
				e.Reply <- nil
			}
			return stateFolded

		case evCredit:
			var err error
			if e.Amt > 0 {
				p.mu.Lock()
				p.balance += e.Amt
				p.mu.Unlock()
			} else if e.Amt < 0 {
				err = fmt.Errorf("credit amount must be positive")
			}
			// Send reply if channel provided
			if e.Reply != nil {
				e.Reply <- err
			}

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
					nextState = psAllIn
					p.stateID.Store(int32(psAllIn))
				} else {
					nextState = psInGame
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
	p.stateID.Store(int32(psAllIn))
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
		case evCredit:
			// All-in players can still receive winnings
			var err error
			if e.Amt > 0 {
				p.mu.Lock()
				p.balance += e.Amt
				p.mu.Unlock()
			} else if e.Amt < 0 {
				err = fmt.Errorf("credit amount must be positive")
			}
			// Send reply if channel provided
			if e.Reply != nil {
				e.Reply <- err
			}
		case evFold:
			// ignored by policy
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
	p.stateID.Store(int32(psFolded))
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
		case evCredit:
			// Folded players can still receive refunds (e.g., uncalled bets)
			var err error
			if e.Amt > 0 {
				p.mu.Lock()
				p.balance += e.Amt
				p.mu.Unlock()
			} else if e.Amt < 0 {
				err = fmt.Errorf("credit amount must be positive")
			}
			// Send reply if channel provided
			if e.Reply != nil {
				e.Reply <- err
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

func stateLeft(p *Player, _ <-chan any) PlayerStateFn {
	p.stateID.Store(int32(psLeft))
	return nil // terminal
}

// ResetForNewHand prepares the player for a new hand.
func (p *Player) ResetForNewHand(startingChips int64) error {
	if p.sm == nil {
		return fmt.Errorf("player state machine not initialized")
	}

	// Set stack for the new hand (table-sourced).
	p.mu.Lock()
	p.balance = startingChips
	p.startingBalance = startingChips
	p.lastAction = time.Now()

	// Clear previous hand's cards and state
	p.hand = nil
	p.currentBet = 0
	p.handDescription = ""
	p.isTurn = false
	p.mu.Unlock()

	// Reset player back to AT_TABLE state for the new hand.
	// The Game FSM's statePreDeal will post blinds (which requires AT_TABLE state)
	// and then send evStartHand to transition players to IN_GAME.
	p.sm.Send(evEndHand{}) // FOLDED/ALL_IN/IN_GAME/etc. -> AT_TABLE
	return nil
}

// GetCurrentStateString returns a stable string based on the fast state snapshot.
func (p *Player) GetCurrentStateString() string {
	return GetPlayerStateString(PlayerState(p.stateID.Load()))
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

// TryFold attempts to fold the player (no-op if all-in).
// Returns true if a fold request was accepted, false if disallowed.
func (p *Player) TryFold() (bool, error) {
	// All-in players cannot fold; keep fast check (also enforced in state fn).
	p.mu.RLock()
	zeroAllIn := p.balance == 0 && p.currentBet > 0
	p.mu.RUnlock()
	if zeroAllIn {
		return false, fmt.Errorf("player is all-in")
	}
	if p.sm == nil {
		return false, fmt.Errorf("player state machine not initialized")
	}
	p.sm.Send(evFold{})
	return true, nil
}

// credit adds chips to the player's balance atomically and notifies FSM to re-evaluate.
// Centralizes balance mutations for pot settlement and refunds.
func (p *Player) credit(amount int64) error {
	if amount <= 0 {
		return nil
	}
	// Send credit event to FSM and wait for it to process
	if p.sm != nil {
		reply := make(chan error, 1)
		p.sm.Send(evCredit{Amt: amount, Reply: reply})
		return <-reply
	}
	return fmt.Errorf("player state machine not initialized")
}

// Marshal converts the Player to gRPC Player for external access.
// Uses the fast state snapshot to derive Folded/AllIn booleans.
func (p *Player) Marshal() *pokerrpc.Player {
	p.mu.RLock()
	defer p.mu.RUnlock()
	grpcHand := make([]*pokerrpc.Card, len(p.hand))
	for i, card := range p.hand {
		grpcHand[i] = &pokerrpc.Card{
			Suit:  string(card.suit),
			Value: string(card.value),
		}
	}

	stateStr := p.GetCurrentStateString()

	proto := &pokerrpc.Player{
		Id:              p.id,
		Name:            p.name,
		Balance:         p.balance,
		Hand:            grpcHand,
		CurrentBet:      p.currentBet,
		Folded:          stateStr == "FOLDED",
		IsTurn:          p.isTurn,
		IsAllIn:         stateStr == "ALL_IN",
		IsDealer:        p.isDealer,
		IsSmallBlind:    p.isSmallBlind,
		IsBigBlind:      p.isBigBlind,
		IsReady:         p.isReady,
		HandDescription: p.handDescription,
		PlayerState:     p.ProtoState(), // renamed to ProtoState below
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

	p.hand = make([]Card, len(grpcPlayer.Hand))
	for i, grpcCard := range grpcPlayer.Hand {
		p.hand[i] = Card{
			suit:  Suit(grpcCard.Suit),
			value: Value(grpcCard.Value),
		}
	}
	p.mu.Unlock()
}

// RestoreState forcefully sets the player's state machine to the provided state.
// This is intended for snapshots/tests/recovery. We rebuild the Pike
// machine with the desired initial state to keep ownership semantics correct.
func (p *Player) RestoreState(state string) error {
	var initial statemachine.StateFn[Player]

	switch state {
	case "AT_TABLE":
		initial = stateAtTable
		p.stateID.Store(int32(psAtTable))
	case "IN_GAME":
		initial = stateInGame
		p.stateID.Store(int32(psInGame))
	case "ALL_IN":
		initial = stateAllIn
		p.stateID.Store(int32(psAllIn))
	case "FOLDED":
		initial = stateFolded
		p.stateID.Store(int32(psFolded))
	case "LEFT":
		initial = stateLeft
		p.stateID.Store(int32(psLeft))
	default:
		return fmt.Errorf("unknown player state: %s", state)
	}

	// Tear down the old machine (if any) and start anew at the requested state.
	if p.sm != nil {
		p.sm.Stop()
	}
	p.sm = statemachine.New(p, initial, 32)
	p.sm.Start(context.Background())
	return nil
}

func (p *Player) Hand() []Card {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]Card, len(p.hand))
	copy(out, p.hand)
	return out
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

// SetHand sets the player's hand (for testing only)
func (p *Player) SetHand(hand []Card) {
	p.mu.Lock()
	p.hand = hand
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

func (p *Player) StartTurn() {
	if p.sm != nil {
		reply := make(chan error, 1)
		p.sm.Send(evStartTurn{Reply: reply})
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
	if p.sm != nil {
		reply := make(chan error, 1)
		p.sm.Send(evEndTurn{Reply: reply})
		<-reply // Wait for FSM to process
	}
}

// Marker interface for player events (optional, for readability).

type evCallDelta struct {
	Amt   int64
	Reply chan<- error // Optional reply channel for synchronous confirmation
}

type evPostBlind struct {
	Amt   int64
	Reply chan<- int64 // Reply with the amount actually posted (for synchronous confirmation)
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
type evCredit struct {
	Amt   int64
	Reply chan<- error // Optional reply channel for synchronous confirmation
}

type evFold struct{}

// evFoldReq requests a synchronous fold with an acknowledgment once the
// player's state has transitioned to FOLDED. This allows callers to avoid
// racing against reads that depend on the folded state.
type evFoldReq struct{ Reply chan<- error }

type evAllIn struct{} // not strictly needed; all-in is derived from stack, but provided if you want it

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
