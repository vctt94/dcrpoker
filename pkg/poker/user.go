package poker

import (
	"fmt"
	"sync"
	"time"

	"github.com/vctt94/pokerbisonrelay/pkg/statemachine"
)

type UserMachineStateFn = statemachine.StateFn[User]

// User represents someone seated at the table (not necessarily playing)
type User struct {
	mu                sync.RWMutex
	sm                *statemachine.Machine[User] // state machine
	ID                string
	Name              string
	DCRAccountBalance int64 // DCR account balance (in atoms)
	table             *Table
	TableSeat         int  // Seat position at the table
	IsReady           bool // Ready to start/continue games
	JoinedAt          time.Time
	IsDisconnected    bool // Whether the user is disconnected
	EscrowID          string
	EscrowReady       bool // Whether escrow funding is valid/bound
	PresignComplete   bool // Whether settlement presigning is complete
}

// fired when users join/leave or toggle ready; state may move to/from PLAYERS_READY
type evUsersChanged struct{}

// ready/unready requests routed through the table FSM
type evSetUserReady struct {
	userID string
	ready  bool
	reply  chan<- error
}

// evSetUserEscrow sets escrow binding for a user via the table FSM
type evSetUserEscrow struct {
	userID   string
	escrowID string
	ready    bool
	reply    chan<- error
}

// evSetUserPresignComplete marks presigning complete for a user via the table FSM
type evSetUserPresignComplete struct {
	userID string
	reply  chan<- error
}

// AddUserOptions allows callers to attach optional metadata to a user.
type AddUserOptions struct {
	DisplayName       string
	DCRAccountBalance int64
	EscrowID          string
	EscrowReady       bool
}

// NewUser creates a new user with optional metadata.
// If opts is nil, defaults are used (DisplayName = id, zero balance, no escrow).
func NewUser(id string, table *Table, opts *AddUserOptions) *User {
	cfg := AddUserOptions{
		DisplayName: id,
	}
	if opts != nil {
		if opts.DisplayName != "" {
			cfg.DisplayName = opts.DisplayName
		}
		cfg.DCRAccountBalance = opts.DCRAccountBalance
		cfg.EscrowID = opts.EscrowID
		cfg.EscrowReady = opts.EscrowReady
	}

	user := &User{
		ID:                id,
		Name:              cfg.DisplayName,
		DCRAccountBalance: cfg.DCRAccountBalance,
		table:             table,
		TableSeat:         -1, // -1 indicates unseated
		IsReady:           false,
		JoinedAt:          time.Now(),
		EscrowID:          cfg.EscrowID,
		EscrowReady:       cfg.EscrowReady,
	}
	user.sm = statemachine.New(user, stateTableSeated, 32)

	return user
}

func stateTableSeated(u *User, in <-chan any) UserMachineStateFn {

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
				u.mu.Lock()
				u.IsReady = true
				table := u.table
				u.mu.Unlock()
				if table != nil {
					err := table.SendPlayerReady(u.ID, true)
					if err != nil {
						// xxx
					}
				}
			case evEscrowBound:
				u.mu.Lock()
				u.EscrowID = e.EscrowID
				u.EscrowReady = e.Ready
				u.mu.Unlock()

			case evPresignComplete:
				u.mu.Lock()
				u.PresignComplete = true
				u.mu.Unlock()

			case evBalanceNotification:
				// Async notification from Game FSM - balance already changed
				// Can be used for UI updates, logging, etc.
				// No action needed here

			case evDisconnect:
				u.mu.Lock()
				u.IsDisconnected = true
				u.mu.Unlock()
			case evReconnect:
				u.mu.Lock()
				u.IsDisconnected = false
				u.mu.Unlock()

			case evLeave:
				return stateTableLeft

			default:
				u.mu.RLock()
				tm := u.sm
				u.mu.RUnlock()
				if tm != nil {
					tm.Send(ev)
				}
			}
		}
	}
}

func stateTableLeft(u *User, _ <-chan any) UserMachineStateFn {
	return nil // terminal
}

// SendEscrowBound sends an escrow bound event through the FSM
func (u *User) SendEscrowBound(escrowID string, ready bool) error {
	u.mu.RLock()
	sm := u.sm
	table := u.table
	u.mu.RUnlock()
	if sm == nil {
		return fmt.Errorf("state machine is nil")
	}
	if table == nil {
		return fmt.Errorf("table is nil")
	}
	err := table.SetPlayerEscrow(u.ID, escrowID, ready)
	if err != nil {
		return err
	}
	sm.Send(evEscrowBound{EscrowID: escrowID, Ready: ready})
	return nil
}

// SendPresignComplete sends a presign complete event through the FSM
func (u *User) SendPresignComplete() {
	u.mu.RLock()
	tp := u.sm
	u.mu.RUnlock()
	if tp == nil {
		return
	}
	tp.Send(evPresignComplete{})
}

func (u *User) Close() {
	// Grab reference to state machine while holding lock
	u.mu.Lock()
	tableMachine := u.sm
	u.mu.Unlock()

	// Stop state machine
	if tableMachine != nil {
		tableMachine.Stop()
	}
}

func (u *User) SendReconnection() {
	u.mu.RLock()
	tm := u.sm
	u.mu.RUnlock()

	if tm == nil {
		return
	}

	tm.Send(evReconnect{})
}

func (u *User) SendDisconnect() {
	u.mu.RLock()
	tm := u.sm
	u.mu.RUnlock()

	if tm == nil {
		return
	}

	tm.Send(evDisconnect{})
}

// HandlePostBlind posts a blind and returns the amount actually posted
func (u *User) HandlePostBlind(amount int64) int64 {
	u.mu.RLock()
	tm := u.sm
	u.mu.RUnlock()

	if tm == nil {
		return 0
	}

	reply := make(chan int64, 1)
	tm.Send(evDeductBlind{Amt: amount, Reply: reply})
	return <-reply
}

// NotifyBalanceChange sends an async notification to the Player FSM about a balance change.
// This is called by the Game FSM after directly modifying player.balance.
// The notification is non-blocking - it's for UI/logging purposes only.
func (u *User) NotifyBalanceChange(newBalance, delta int64, reason string) {
	u.mu.RLock()
	tm := u.sm
	u.mu.RUnlock()

	if tm != nil {
		// Non-blocking send using the statemachine's Send method
		tm.Send(evBalanceNotification{
			NewBalance: newBalance,
			Delta:      delta,
			Reason:     reason,
		})
	}
}

// Ready/unready helpers for table presence FSM.
func (u *User) SendReady() error {
	u.mu.RLock()
	sm := u.sm
	table := u.table
	u.mu.RUnlock()
	if sm == nil {
		return fmt.Errorf("state machine is nil")
	}
	if table == nil {
		return fmt.Errorf("table is nil")
	}
	err := table.SendPlayerReady(u.ID, true)
	if err != nil {
		return err
	}
	sm.Send(evReady{})
	return nil
}

func (u *User) SendUnready() error {
	u.mu.RLock()
	sm := u.sm
	table := u.table
	u.mu.RUnlock()
	if sm == nil {
		return fmt.Errorf("state machine is nil")
	}
	if table == nil {
		return fmt.Errorf("table is nil")
	}
	err := table.SendPlayerReady(u.ID, false)
	if err != nil {
		// xxx
		return err
	}
	sm.Send(evUnready{})
	return nil
}
