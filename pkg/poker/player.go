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

func (p *Player) ReadyUp()      { p.sm.Send(evReady{}) }
func (p *Player) StartHand()    { p.sm.Send(evStartHand{}) }
func (p *Player) YourTurn()     { p.sm.Send(evYourTurn{}) }
func (p *Player) Bet(amt int64) { p.sm.Send(evBet{Amt: amt}) }
func (p *Player) Call()         { p.sm.Send(evCall{}) }
func (p *Player) Fold()         { p.sm.Send(evFold{}) }
func (p *Player) EndHand()      { p.sm.Send(evEndHand{}) }
func (p *Player) Disconnect()   { p.sm.Send(evDisconnect{}) }
func (p *Player) LeaveTable()   { p.sm.Send(evLeave{}) }

func (p *Player) StateID() playerState { return playerState(p.stateID.Load()) }
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
	p.isDealer = false
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

			case evStartHand:
				p.mu.Lock()
				p.startingBalance = p.balance
				p.mu.Unlock()
				return stateInGame

			case evCallDelta: // allow call while FSM is still AT_TABLE
				p.mu.Lock()
				can := e.Amt > 0 && e.Amt <= p.balance
				if can {
					p.balance -= e.Amt
					p.currentBet += e.Amt
					p.lastAction = time.Now()
				}
				p.mu.Unlock()
				if can {
					// Move into the hand; if stack is now zero, go straight to ALL_IN.
					p.mu.RLock()
					zero := p.balance == 0 && p.currentBet > 0
					p.mu.RUnlock()
					if zero {
						return stateAllIn
					}
					return stateInGame
				}
				// invalid or ignored, stay at table

			case evReeval:
				// re-check derived all-in from external mutation
				p.mu.RLock()
				zero := p.balance == 0 && p.currentBet > 0
				p.mu.RUnlock()
				if zero {
					return stateAllIn
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
		case evYourTurn:
			p.mu.Lock()
			p.isTurn = true
			p.mu.Unlock()

		case evBet:
			p.mu.Lock()
			can := p.isTurn && e.Amt > 0 && e.Amt <= p.balance
			if can {
				p.balance -= e.Amt
				p.currentBet += e.Amt
				p.lastAction = time.Now()
				p.isTurn = false
			}
			zero := (p.balance == 0)
			p.mu.Unlock()
			if can && zero {
				return stateAllIn
			}

		case evCall:
			// Amount computed by table; state only observes results.
			p.mu.Lock()
			p.lastAction = time.Now()
			p.isTurn = false
			zero := (p.balance == 0 && p.currentBet > 0)
			p.mu.Unlock()
			if zero {
				return stateAllIn
			}

		case evFold:
			return stateFolded

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
			if can {
				p.balance -= e.Amt
				p.currentBet += e.Amt
				p.lastAction = time.Now()
			}
			zero := p.balance == 0 && p.currentBet > 0
			p.mu.Unlock()
			if can {
				if zero {
					return stateAllIn
				}
				return stateInGame
			}

		case evReeval:
			// ← NEW: re-check derived condition after external chip mutation
			p.mu.RLock()
			zero := p.balance == 0 && p.currentBet > 0
			p.mu.RUnlock()
			if zero {
				return stateAllIn
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
		switch ev.(type) {
		case evFold:
			// ignored by policy
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
		switch ev.(type) {
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

	// Ensure we return to AT_TABLE first, then start the new hand.
	p.sm.Send(evEndHand{})   // FOLDED/ALL_IN/etc. -> AT_TABLE
	p.sm.Send(evStartHand{}) // AT_TABLE -> IN_GAME
	return nil
}

// GetCurrentStateString returns a stable string based on the fast state snapshot.
func (p *Player) GetCurrentStateString() string {
	switch playerState(p.stateID.Load()) {
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

	return &pokerrpc.Player{
		Id:              p.id,
		Name:            p.name,
		Balance:         p.balance,
		Hand:            grpcHand,
		CurrentBet:      p.currentBet,
		Folded:          stateStr == "FOLDED",
		IsTurn:          p.isTurn,
		IsAllIn:         stateStr == "ALL_IN",
		IsDealer:        p.isDealer,
		IsReady:         p.isReady,
		HandDescription: p.handDescription,
		PlayerState:     p.ProtoState(), // renamed to ProtoState below
	}
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

// Marker interface for player events (optional, for readability).

type evCallDelta struct{ Amt int64 }

type evReady struct{}

type evStartHand struct{}

type evYourTurn struct{}

type evBet struct{ Amt int64 }

type evCall struct{}

type evFold struct{}

type evAllIn struct{} // not strictly needed; all-in is derived from stack, but provided if you want it

type evReeval struct{}
type evEndHand struct{}

type evDisconnect struct{}

type evLeave struct{}

type PlayerStateFn = statemachine.StateFn[Player]

type playerState int32

const (
	psUninitialized playerState = iota
	psAtTable
	psInGame
	psAllIn
	psFolded
	psLeft
)
