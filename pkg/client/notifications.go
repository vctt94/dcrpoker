package client

import (
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/companyzero/bisonrelay/zkidentity"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
)

// Following are the notification types. Add new types at the bottom of this
// list, then add a notifyX() to NotificationManager and initialize a new
// container in NewNotificationManager().

const onTableCreatedNtfnType = "onTableCreated"

// OnTableCreatedNtfn is the handler for table creation notifications.
type OnTableCreatedNtfn func(*pokerrpc.Table, time.Time)

func (_ OnTableCreatedNtfn) typ() string { return onTableCreatedNtfnType }

const onPlayerJoinedNtfnType = "onPlayerJoined"

// OnPlayerJoinedNtfn is the handler for player joined notifications.
type OnPlayerJoinedNtfn func(*pokerrpc.Table, string, time.Time)

func (_ OnPlayerJoinedNtfn) typ() string { return onPlayerJoinedNtfnType }

const onPlayerLeftNtfnType = "onPlayerLeft"

// OnPlayerLeftNtfn is the handler for player left notifications.
type OnPlayerLeftNtfn func(*pokerrpc.Table, string, time.Time)

func (_ OnPlayerLeftNtfn) typ() string { return onPlayerLeftNtfnType }

const onGameStartedNtfnType = "onGameStarted"

// OnGameStartedNtfn is the handler for game started notifications.
type OnGameStartedNtfn func(string, time.Time)

func (_ OnGameStartedNtfn) typ() string { return onGameStartedNtfnType }

const onGameEndedNtfnType = "onGameEnded"

// OnGameEndedNtfn is the handler for game ended notifications.
type OnGameEndedNtfn func(string, string, time.Time)

func (_ OnGameEndedNtfn) typ() string { return onGameEndedNtfnType }

const onBetMadeNtfnType = "onBetMade"

// OnBetMadeNtfn is the handler for bet made notifications.
type OnBetMadeNtfn func(string, int64, time.Time)

func (_ OnBetMadeNtfn) typ() string { return onBetMadeNtfnType }

const onPlayerFoldedNtfnType = "onPlayerFolded"

// OnPlayerFoldedNtfn is the handler for player folded notifications.
type OnPlayerFoldedNtfn func(string, time.Time)

func (_ OnPlayerFoldedNtfn) typ() string { return onPlayerFoldedNtfnType }

const onPlayerReadyNtfnType = "onPlayerReady"

// OnPlayerReadyNtfn is the handler for player ready notifications.
type OnPlayerReadyNtfn func(string, bool, time.Time)

func (_ OnPlayerReadyNtfn) typ() string { return onPlayerReadyNtfnType }

const onBalanceUpdatedNtfnType = "onBalanceUpdated"

// OnBalanceUpdatedNtfn is the handler for balance updated notifications.
type OnBalanceUpdatedNtfn func(string, int64, time.Time)

func (_ OnBalanceUpdatedNtfn) typ() string { return onBalanceUpdatedNtfnType }

const onTipReceivedNtfnType = "onTipReceived"

// OnTipReceivedNtfn is the handler for tip received notifications.
type OnTipReceivedNtfn func(string, string, int64, string, time.Time)

func (_ OnTipReceivedNtfn) typ() string { return onTipReceivedNtfnType }

const onShowdownResultNtfnType = "onShowdownResult"

// OnShowdownResultNtfn is the handler for showdown result notifications.
type OnShowdownResultNtfn func(string, []*pokerrpc.Winner, time.Time)

func (_ OnShowdownResultNtfn) typ() string { return onShowdownResultNtfnType }

const onPlayerCalledNtfnType = "onPlayerCalled"

// OnPlayerCalledNtfn is the handler for player called notifications.
type OnPlayerCalledNtfn func(string, int64, time.Time)

func (_ OnPlayerCalledNtfn) typ() string { return onPlayerCalledNtfnType }

const onPlayerRaisedNtfnType = "onPlayerRaised"

// OnPlayerRaisedNtfn is the handler for player raised notifications.
type OnPlayerRaisedNtfn func(string, int64, time.Time)

func (_ OnPlayerRaisedNtfn) typ() string { return onPlayerRaisedNtfnType }

const onPlayerCheckedNtfnType = "onPlayerChecked"

// OnPlayerCheckedNtfn is the handler for player checked notifications.
type OnPlayerCheckedNtfn func(string, time.Time)

func (_ OnPlayerCheckedNtfn) typ() string { return onPlayerCheckedNtfnType }

// UINotificationsConfig is the configuration for how UI notifications are
// emitted.
type UINotificationsConfig struct {
	// GameStarted flag whether to emit notification after game starts.
	GameStarted bool

	TableCreated bool

	// MaxLength is the max length of messages emitted.
	MaxLength int

	// MentionRegexp is the regexp to detect mentions.
	MentionRegexp *regexp.Regexp

	// EmitInterval is the interval to wait for additional messages before
	// emitting a notification. Multiple messages received within this
	// interval will only generate a single UI notification.
	EmitInterval time.Duration

	// CancelEmissionChannel may be set to a Context.Done() channel to
	// cancel emission of notifications.
	CancelEmissionChannel <-chan struct{}
}

func (cfg *UINotificationsConfig) clip(msg string) string {
	if len(msg) < cfg.MaxLength {
		return msg
	}
	return msg[:cfg.MaxLength]
}

// UINotificationType is the type of notification.
type UINotificationType string

const (
	UINtfnGameStarted  UINotificationType = "gamestarted"
	UINtfnTableCreated UINotificationType = "tablecreated"
	UINtfnPlayerJoined UINotificationType = "playerjoined"
	UINtfnBetMade      UINotificationType = "betmade"
	UINtfnTipReceived  UINotificationType = "tipreceived"
	UINtfnMultiple     UINotificationType = "multiple"
)

// UINotification is a notification that should be shown as an UI alert.
type UINotification struct {
	// Type of notification.
	Type UINotificationType `json:"type"`

	// Text of the notification.
	Text string `json:"text"`

	// Count will be greater than one when multiple notifications were
	// batched.
	Count int `json:"count"`

	// From is the original sender or table of the notification.
	From zkidentity.ShortID `json:"from"`

	// FromNick is the nick of the sender.
	FromNick string `json:"from_nick"`

	// Timestamp is the unix timestamp in seconds of the first message.
	Timestamp int64 `json:"timestamp"`
}

// fromSame returns true if the notification is from the same ID.
func (n *UINotification) fromSame(id *zkidentity.ShortID) bool {
	if id == nil || n.From.IsEmpty() {
		return false
	}

	return *id == n.From
}

const onUINtfnType = "uintfn"

// OnUINotification is called when a notification should be shown by the UI to
// the user. This should usually take the form of an alert dialog about a
// received message.
type OnUINotification func(ntfn UINotification)

func (_ OnUINotification) typ() string { return onUINtfnType }

// The following is used only in tests.

const onTestNtfnType = "testNtfnType"

type onTestNtfn func()

func (_ onTestNtfn) typ() string { return onTestNtfnType }

// Following is the generic notification code.

type NotificationRegistration struct {
	unreg func() bool
}

func (reg NotificationRegistration) Unregister() bool {
	return reg.unreg()
}

type NotificationHandler interface {
	typ() string
}

type handler[T any] struct {
	handler T
	async   bool
}

type handlersFor[T any] struct {
	mtx      sync.Mutex
	next     uint
	handlers map[uint]handler[T]
}

func (hn *handlersFor[T]) register(h T, async bool) NotificationRegistration {
	var id uint

	hn.mtx.Lock()
	id, hn.next = hn.next, hn.next+1
	if hn.handlers == nil {
		hn.handlers = make(map[uint]handler[T])
	}
	hn.handlers[id] = handler[T]{handler: h, async: async}
	registered := true
	hn.mtx.Unlock()

	return NotificationRegistration{
		unreg: func() bool {
			hn.mtx.Lock()
			res := registered
			if registered {
				delete(hn.handlers, id)
				registered = false
			}
			hn.mtx.Unlock()
			return res
		},
	}
}

func (hn *handlersFor[T]) visit(f func(T)) {
	hn.mtx.Lock()
	for _, h := range hn.handlers {
		if h.async {
			go f(h.handler)
		} else {
			f(h.handler)
		}
	}
	hn.mtx.Unlock()
}

func (hn *handlersFor[T]) Register(v interface{}, async bool) NotificationRegistration {
	if h, ok := v.(T); !ok {
		panic("wrong type")
	} else {
		return hn.register(h, async)
	}
}

func (hn *handlersFor[T]) AnyRegistered() bool {
	hn.mtx.Lock()
	res := len(hn.handlers) > 0
	hn.mtx.Unlock()
	return res
}

type handlersRegistry interface {
	Register(v interface{}, async bool) NotificationRegistration
	AnyRegistered() bool
}

type NotificationManager struct {
	handlers map[string]handlersRegistry

	uiMtx      sync.Mutex
	uiConfig   UINotificationsConfig
	uiNextNtfn UINotification
	uiTimer    *time.Timer
}

// UpdateUIConfig updates the config used to generate UI notifications about
// game events, table creation, etc.
func (nmgr *NotificationManager) UpdateUIConfig(cfg UINotificationsConfig) {
	nmgr.uiMtx.Lock()
	nmgr.uiConfig = cfg
	nmgr.uiMtx.Unlock()
}

func (nmgr *NotificationManager) register(handler NotificationHandler, async bool) NotificationRegistration {
	handlers := nmgr.handlers[handler.typ()]
	if handlers == nil {
		panic(fmt.Sprintf("forgot to init the handler type %T "+
			"in NewNotificationManager", handler))
	}

	return handlers.Register(handler, async)
}

// Register registers a callback notification function that is called
// asynchronously to the event (i.e. in a separate goroutine).
func (nmgr *NotificationManager) Register(handler NotificationHandler) NotificationRegistration {
	return nmgr.register(handler, true)
}

// RegisterSync registers a callback notification function that is called
// synchronously to the event. This callback SHOULD return as soon as possible,
// otherwise the client might hang.
//
// Synchronous callbacks are mostly intended for tests and when external
// callers need to ensure proper order of multiple sequential events. In
// general it is preferable to use callbacks registered with the Register call,
// to ensure the client will not deadlock or hang.
func (nmgr *NotificationManager) RegisterSync(handler NotificationHandler) NotificationRegistration {
	return nmgr.register(handler, false)
}

// AnyRegistered returns true if there are any handlers registered for the given
// handler type.
func (nmgr *NotificationManager) AnyRegistered(handler NotificationHandler) bool {
	return nmgr.handlers[handler.typ()].AnyRegistered()
}

func (nmgr *NotificationManager) waitAndEmitUINtfn(c <-chan time.Time, cancel <-chan struct{}) {
	select {
	case <-c:
	case <-cancel:
		return
	}

	nmgr.uiMtx.Lock()
	n := nmgr.uiNextNtfn
	nmgr.uiNextNtfn = UINotification{}
	nmgr.uiMtx.Unlock()

	nmgr.handlers[onUINtfnType].(*handlersFor[OnUINotification]).
		visit(func(h OnUINotification) { h(n) })
}

func (nmgr *NotificationManager) addUINtfn(from zkidentity.ShortID, typ UINotificationType, msg string, ts time.Time) {
	nmgr.uiMtx.Lock()

	n := &nmgr.uiNextNtfn
	cfg := &nmgr.uiConfig

	switch {
	case typ == UINtfnTableCreated && !cfg.TableCreated:
		// Ignore
		nmgr.uiMtx.Unlock()
		return

	case typ == UINtfnTableCreated && n.Type == UINtfnTableCreated:
		// First table creation.
		n.Type = typ
		n.Count = 1
		n.From = from
		n.Timestamp = ts.Unix()
		n.Text = fmt.Sprintf("Table created by %s: %s", from,
			cfg.clip(msg))

	case typ == UINtfnGameStarted && cfg.GameStarted:
		// Game started notification.
		n.Type = typ
		n.Count = 1
		n.From = from
		n.Timestamp = ts.Unix()
		n.Text = cfg.clip(msg)

	default:
		// Multiple types.
		n.Type = UINtfnMultiple
		n.FromNick = "multiple"
		n.Count += 1
		n.Text = fmt.Sprintf("%d notifications received", n.Count)
	}

	// The first notification starts the timer to emit the actual UI
	// notification. Other notifications will get batched.
	if n.Count == 1 {
		nmgr.uiTimer.Reset(cfg.EmitInterval)
		c, cancel := nmgr.uiTimer.C, cfg.CancelEmissionChannel
		go nmgr.waitAndEmitUINtfn(c, cancel)
	}

	nmgr.uiMtx.Unlock()
}

// Following are the notifyX() calls (one for each type of notification).

func (nmgr *NotificationManager) notifyTest() {
	nmgr.handlers[onTestNtfnType].(*handlersFor[onTestNtfn]).
		visit(func(h onTestNtfn) { h() })
}

func (nmgr *NotificationManager) notifyTableCreated(table *pokerrpc.Table, ts time.Time) {
	nmgr.handlers[onTableCreatedNtfnType].(*handlersFor[OnTableCreatedNtfn]).
		visit(func(h OnTableCreatedNtfn) { h(table, ts) })

	var id zkidentity.ShortID
	nmgr.addUINtfn(id, UINtfnTableCreated, fmt.Sprintf("Table %s created", table.Id), ts)
}

func (nmgr *NotificationManager) notifyPlayerJoined(table *pokerrpc.Table, playerID string, ts time.Time) {
	nmgr.handlers[onPlayerJoinedNtfnType].(*handlersFor[OnPlayerJoinedNtfn]).
		visit(func(h OnPlayerJoinedNtfn) { h(table, playerID, ts) })

	var id zkidentity.ShortID
	id.FromString(playerID)
	nmgr.addUINtfn(id, UINtfnPlayerJoined, fmt.Sprintf("Player joined table %s", table.Id), ts)
}

func (nmgr *NotificationManager) notifyPlayerLeft(table *pokerrpc.Table, playerID string, ts time.Time) {
	nmgr.handlers[onPlayerLeftNtfnType].(*handlersFor[OnPlayerLeftNtfn]).
		visit(func(h OnPlayerLeftNtfn) { h(table, playerID, ts) })
}

func (nmgr *NotificationManager) notifyGameStarted(gameID string, ts time.Time) {
	nmgr.handlers[onGameStartedNtfnType].(*handlersFor[OnGameStartedNtfn]).
		visit(func(h OnGameStartedNtfn) { h(gameID, ts) })

	var id zkidentity.ShortID
	nmgr.addUINtfn(id, UINtfnGameStarted, fmt.Sprintf("Game %s started", gameID), ts)
}

func (nmgr *NotificationManager) notifyGameEnded(gameID string, msg string, ts time.Time) {
	nmgr.handlers[onGameEndedNtfnType].(*handlersFor[OnGameEndedNtfn]).
		visit(func(h OnGameEndedNtfn) { h(gameID, msg, ts) })
}

func (nmgr *NotificationManager) notifyBetMade(playerID string, amount int64, ts time.Time) {
	nmgr.handlers[onBetMadeNtfnType].(*handlersFor[OnBetMadeNtfn]).
		visit(func(h OnBetMadeNtfn) { h(playerID, amount, ts) })

	var id zkidentity.ShortID
	id.FromString(playerID)
	nmgr.addUINtfn(id, UINtfnBetMade, fmt.Sprintf("Player bet %d", amount), ts)
}

func (nmgr *NotificationManager) notifyPlayerFolded(playerID string, ts time.Time) {
	nmgr.handlers[onPlayerFoldedNtfnType].(*handlersFor[OnPlayerFoldedNtfn]).
		visit(func(h OnPlayerFoldedNtfn) { h(playerID, ts) })
}

func (nmgr *NotificationManager) notifyPlayerReady(playerID string, ready bool, ts time.Time) {
	nmgr.handlers[onPlayerReadyNtfnType].(*handlersFor[OnPlayerReadyNtfn]).
		visit(func(h OnPlayerReadyNtfn) { h(playerID, ready, ts) })
}

func (nmgr *NotificationManager) notifyBalanceUpdated(playerID string, newBalance int64, ts time.Time) {
	nmgr.handlers[onBalanceUpdatedNtfnType].(*handlersFor[OnBalanceUpdatedNtfn]).
		visit(func(h OnBalanceUpdatedNtfn) { h(playerID, newBalance, ts) })
}

func (nmgr *NotificationManager) notifyTipReceived(fromPlayerID, toPlayerID string, amount int64, message string, ts time.Time) {
	nmgr.handlers[onTipReceivedNtfnType].(*handlersFor[OnTipReceivedNtfn]).
		visit(func(h OnTipReceivedNtfn) { h(fromPlayerID, toPlayerID, amount, message, ts) })

	var id zkidentity.ShortID
	id.FromString(fromPlayerID)
	nmgr.addUINtfn(id, UINtfnTipReceived, fmt.Sprintf("Tip received: %d from %s", amount, fromPlayerID), ts)
}

func (nmgr *NotificationManager) notifyShowdownResult(gameID string, winners []*pokerrpc.Winner, ts time.Time) {
	nmgr.handlers[onShowdownResultNtfnType].(*handlersFor[OnShowdownResultNtfn]).
		visit(func(h OnShowdownResultNtfn) { h(gameID, winners, ts) })
}

func (nmgr *NotificationManager) notifyPlayerCalled(playerID string, amount int64, ts time.Time) {
	nmgr.handlers[onPlayerCalledNtfnType].(*handlersFor[OnPlayerCalledNtfn]).
		visit(func(h OnPlayerCalledNtfn) { h(playerID, amount, ts) })
}

func (nmgr *NotificationManager) notifyPlayerRaised(playerID string, amount int64, ts time.Time) {
	nmgr.handlers[onPlayerRaisedNtfnType].(*handlersFor[OnPlayerRaisedNtfn]).
		visit(func(h OnPlayerRaisedNtfn) { h(playerID, amount, ts) })
}

func (nmgr *NotificationManager) notifyPlayerChecked(playerID string, ts time.Time) {
	nmgr.handlers[onPlayerCheckedNtfnType].(*handlersFor[OnPlayerCheckedNtfn]).
		visit(func(h OnPlayerCheckedNtfn) { h(playerID, ts) })
}

func NewNotificationManager() *NotificationManager {
	nmgr := &NotificationManager{
		uiConfig: UINotificationsConfig{
			MaxLength:    255,
			EmitInterval: 30 * time.Second,
		},
		uiTimer: time.NewTimer(time.Hour * 24),
		handlers: map[string]handlersRegistry{
			onTestNtfnType:           &handlersFor[onTestNtfn]{},
			onTableCreatedNtfnType:   &handlersFor[OnTableCreatedNtfn]{},
			onPlayerJoinedNtfnType:   &handlersFor[OnPlayerJoinedNtfn]{},
			onPlayerLeftNtfnType:     &handlersFor[OnPlayerLeftNtfn]{},
			onGameStartedNtfnType:    &handlersFor[OnGameStartedNtfn]{},
			onGameEndedNtfnType:      &handlersFor[OnGameEndedNtfn]{},
			onBetMadeNtfnType:        &handlersFor[OnBetMadeNtfn]{},
			onPlayerFoldedNtfnType:   &handlersFor[OnPlayerFoldedNtfn]{},
			onPlayerReadyNtfnType:    &handlersFor[OnPlayerReadyNtfn]{},
			onBalanceUpdatedNtfnType: &handlersFor[OnBalanceUpdatedNtfn]{},
			onTipReceivedNtfnType:    &handlersFor[OnTipReceivedNtfn]{},
			onShowdownResultNtfnType: &handlersFor[OnShowdownResultNtfn]{},
			onPlayerCalledNtfnType:   &handlersFor[OnPlayerCalledNtfn]{},
			onPlayerRaisedNtfnType:   &handlersFor[OnPlayerRaisedNtfn]{},
			onPlayerCheckedNtfnType:  &handlersFor[OnPlayerCheckedNtfn]{},

			onUINtfnType: &handlersFor[OnUINotification]{},
		},
	}
	if !nmgr.uiTimer.Stop() {
		<-nmgr.uiTimer.C
	}

	return nmgr
}
