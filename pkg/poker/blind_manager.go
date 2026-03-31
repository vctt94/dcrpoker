package poker

import (
	"context"
	"fmt"
	"time"

	"github.com/decred/slog"

	"github.com/vctt94/pokerbisonrelay/pkg/statemachine"
)

// BlindLevel represents a single blind level with small and big blind amounts.
type BlindLevel struct {
	SmallBlind int64
	BigBlind   int64
}

// DefaultBlindSchedule is the standard blind level progression.
var DefaultBlindSchedule = []BlindLevel{
	{10, 20},
	{15, 30},
	{20, 40},
	{25, 50},
	{30, 60},
	{40, 80},
	{50, 100},
	{60, 120},
	{75, 150},
	{100, 200},
}

// BlindState represents the FSM state for the blind manager.
type BlindState int

const (
	BlindStateDisabled BlindState = iota // Blind increases are off
	BlindStateWaiting                    // FSM created, waiting for first hand
	BlindStateActive                     // At a level, timer scheduled for next
	BlindStatePending                    // Timer fired; increase waiting for hand boundary
	BlindStateMaxLevel                   // Reached final level; no more increases
)

func (s BlindState) String() string {
	switch s {
	case BlindStateDisabled:
		return "DISABLED"
	case BlindStateWaiting:
		return "WAITING"
	case BlindStateActive:
		return "ACTIVE"
	case BlindStatePending:
		return "PENDING"
	case BlindStateMaxLevel:
		return "MAX_LEVEL"
	default:
		return "UNKNOWN"
	}
}

// BlindInfo is the immutable snapshot returned by the BlindManager FSM.
type BlindInfo struct {
	Level              BlindLevel // Current active level
	Index              int        // Current level index
	State              BlindState
	NextIncreaseUnixMs int64 // 0 when disabled or at max
}

// BlindApplyResult is the reply from evBlindApply.
type BlindApplyResult struct {
	Level   BlindLevel
	Index   int
	Changed bool   // true if level advanced
	Message string // human-readable notification, empty if unchanged
}

// ── Events sent to the BlindManager FSM ────────────────────────────

type evBlindStart struct {
	startTime time.Time // zero = use time.Now()
}

type evBlindTimerFired struct{}

type evBlindGetLevel struct {
	reply chan<- BlindLevel
}

type evBlindGetInfo struct {
	reply chan<- BlindInfo
}

// evBlindApply is sent by the Game FSM at hand boundaries to apply any
// pending blind increase. The level is only mutated here.
type evBlindApply struct {
	reply chan<- BlindApplyResult
}

// ── BlindManager FSM ───────────────────────────────────────────────

type BlindManagerStateFn = statemachine.StateFn[BlindManager]

// BlindManager is an FSM that tracks blind level progression.
//
// The timer fires when the next level is due but does NOT mutate the
// current level. Instead the FSM enters BlindStatePending and sends a
// GameEventBlindsPending notification so the Table can warn players.
// The actual level mutation only happens when the Game FSM sends
// evBlindApply at the start of the next hand (statePreDeal).
//
// State transitions:
//
//	Disabled  (interval == 0 or ≤1 level — terminal)
//	Waiting   → evBlindStart   → Active
//	Active    → timer fires    → Pending  (notify, no mutation)
//	Pending   → evBlindApply   → Active   (mutation!) or MaxLevel
//	Active    → evBlindApply   → Active   (no-op, not due yet)
//	MaxLevel  (terminal, only handles queries)
type BlindManager struct {
	schedule     []BlindLevel
	interval     time.Duration
	currentLevel int
	startTime    time.Time
	log          slog.Logger

	sm *statemachine.Machine[BlindManager]

	// gameEventChan sends pending/increased notifications to the Game/Table.
	gameEventChan chan<- GameEvent
}

// NewBlindManager creates a BlindManager FSM. If interval is 0 or the
// schedule has ≤1 level, the manager starts in BlindStateDisabled.
func NewBlindManager(schedule []BlindLevel, interval time.Duration, log slog.Logger) *BlindManager {
	if len(schedule) == 0 {
		schedule = []BlindLevel{{0, 0}}
	}

	bm := &BlindManager{
		schedule: schedule,
		interval: interval,
		log:      log,
	}

	initial := BlindManagerStateFn(blindWaiting)
	if interval <= 0 || len(schedule) <= 1 {
		initial = blindDisabled
	}
	bm.sm = statemachine.New(bm, initial, 16)
	return bm
}

func (bm *BlindManager) Start(ctx context.Context) { bm.sm.Start(ctx) }

func (bm *BlindManager) Stop() {
	bm.sm.Stop()
}

// SetGameEventChannel wires the notification channel. Must be called before Start.
func (bm *BlindManager) SetGameEventChannel(ch chan<- GameEvent) {
	bm.gameEventChan = ch
}

// ── Public query helpers (send event → block for reply) ────────────

// GetLevel queries the current blind level synchronously.
func (bm *BlindManager) GetLevel() BlindLevel {
	reply := make(chan BlindLevel, 1)
	bm.sm.Send(evBlindGetLevel{reply: reply})
	return <-reply
}

// GetInfo queries full blind state info synchronously.
func (bm *BlindManager) GetInfo() BlindInfo {
	reply := make(chan BlindInfo, 1)
	bm.sm.Send(evBlindGetInfo{reply: reply})
	return <-reply
}

// SendStart tells the FSM to begin tracking time.
func (bm *BlindManager) SendStart(startTime time.Time) {
	bm.sm.Send(evBlindStart{startTime: startTime})
}

// Apply asks the FSM to apply any pending blind increase. This is the
// ONLY way the current level advances. Called by Game at hand boundaries.
func (bm *BlindManager) Apply() BlindApplyResult {
	reply := make(chan BlindApplyResult, 1)
	bm.sm.Send(evBlindApply{reply: reply})
	return <-reply
}

// Schedule returns a copy of the blind schedule.
func (bm *BlindManager) Schedule() []BlindLevel {
	out := make([]BlindLevel, len(bm.schedule))
	copy(out, bm.schedule)
	return out
}

// TotalLevels returns the number of levels in the schedule.
func (bm *BlindManager) TotalLevels() int {
	return len(bm.schedule)
}

// ── FSM state functions ────────────────────────────────────────────

// blindDisabled is the terminal state when blind increases are off.
func blindDisabled(bm *BlindManager, in <-chan any) BlindManagerStateFn {
	for ev := range in {
		switch e := ev.(type) {
		case evBlindGetLevel:
			e.reply <- bm.schedule[0]
		case evBlindGetInfo:
			e.reply <- BlindInfo{
				Level: bm.schedule[0],
				State: BlindStateDisabled,
			}
		case evBlindApply:
			e.reply <- BlindApplyResult{Level: bm.schedule[0]}
		}
	}
	return nil
}

// blindWaiting waits for the first hand to start before activating timers.
func blindWaiting(bm *BlindManager, in <-chan any) BlindManagerStateFn {
	for ev := range in {
		switch e := ev.(type) {
		case evBlindStart:
			if e.startTime.IsZero() {
				bm.startTime = time.Now()
			} else {
				bm.startTime = e.startTime
			}
			bm.log.Infof("BlindManager started: level 0 (%d/%d), interval %v, %d levels",
				bm.schedule[0].SmallBlind, bm.schedule[0].BigBlind,
				bm.interval, len(bm.schedule))
			return blindActive

		case evBlindGetLevel:
			e.reply <- bm.schedule[0]
		case evBlindGetInfo:
			e.reply <- BlindInfo{
				Level: bm.schedule[0],
				State: BlindStateWaiting,
			}
		case evBlindApply:
			e.reply <- BlindApplyResult{Level: bm.schedule[0]}
		}
	}
	return nil
}

// blindActive is the steady-state: a timer is scheduled for the next
// level transition but the current level is NOT mutated until the Game
// explicitly sends evBlindApply.
func blindActive(bm *BlindManager, in <-chan any) BlindManagerStateFn {
	nextLevelTime := bm.startTime.Add(time.Duration(bm.currentLevel+1) * bm.interval)
	remaining := time.Until(nextLevelTime)
	if remaining <= 0 {
		// Already past the next level boundary — go directly to pending.
		return blindPending
	}

	timer := time.AfterFunc(remaining, func() {
		bm.sm.TrySend(evBlindTimerFired{})
	})
	defer func() {
		timer.Stop()
	}()

	for ev := range in {
		switch e := ev.(type) {
		case evBlindTimerFired:
			return blindPending

		case evBlindApply:
			// Not due yet — reply with current level, unchanged.
			e.reply <- BlindApplyResult{
				Level: bm.schedule[bm.currentLevel],
				Index: bm.currentLevel,
			}

		case evBlindGetLevel:
			e.reply <- bm.schedule[bm.currentLevel]
		case evBlindGetInfo:
			e.reply <- bm.buildInfo(BlindStateActive)
		}
	}
	return nil
}

// blindPending is entered when the timer fires. The next level is due
// but not yet applied. The FSM notifies the game channel and waits for
// evBlindApply before mutating currentLevel.
func blindPending(bm *BlindManager, in <-chan any) BlindManagerStateFn {
	// Compute what the target level would be (not applied yet).
	targetLevel := bm.computeTarget()
	if targetLevel <= bm.currentLevel {
		targetLevel = bm.currentLevel + 1
	}
	if targetLevel >= len(bm.schedule) {
		targetLevel = len(bm.schedule) - 1
	}

	nextLevel := bm.schedule[targetLevel]

	// Notify the game/table that an increase is pending.
	bm.notifyPending(nextLevel)

	bm.log.Infof("Blind increase pending: level %d → %d (%d/%d), waiting for hand boundary",
		bm.currentLevel, targetLevel, nextLevel.SmallBlind, nextLevel.BigBlind)

	for ev := range in {
		switch e := ev.(type) {
		case evBlindApply:
			// ── THIS IS THE ONLY PLACE currentLevel IS MUTATED ──
			prev := bm.currentLevel
			bm.advanceToTarget()
			level := bm.schedule[bm.currentLevel]

			result := BlindApplyResult{
				Level:   level,
				Index:   bm.currentLevel,
				Changed: bm.currentLevel > prev,
			}
			if result.Changed {
				result.Message = fmt.Sprintf("Blinds increased to %d/%d",
					level.SmallBlind, level.BigBlind)
				bm.log.Infof("Blinds applied: level %d → %d (%d/%d)",
					prev, bm.currentLevel, level.SmallBlind, level.BigBlind)
			}
			e.reply <- result

			if bm.currentLevel >= len(bm.schedule)-1 {
				return blindMaxLevel
			}
			return blindActive

		case evBlindTimerFired:
			// Another interval passed while still pending — recompute target.
			targetLevel = bm.computeTarget()
			if targetLevel >= len(bm.schedule) {
				targetLevel = len(bm.schedule) - 1
			}

		case evBlindGetLevel:
			// Return CURRENT level (not the pending one).
			e.reply <- bm.schedule[bm.currentLevel]
		case evBlindGetInfo:
			info := bm.buildInfo(BlindStatePending)
			e.reply <- info
		}
	}
	return nil
}

// blindMaxLevel is the terminal state when the highest level is reached.
func blindMaxLevel(bm *BlindManager, in <-chan any) BlindManagerStateFn {
	for ev := range in {
		switch e := ev.(type) {
		case evBlindGetLevel:
			e.reply <- bm.schedule[bm.currentLevel]
		case evBlindGetInfo:
			e.reply <- bm.buildInfo(BlindStateMaxLevel)
		case evBlindApply:
			e.reply <- BlindApplyResult{
				Level: bm.schedule[bm.currentLevel],
				Index: bm.currentLevel,
			}
		}
	}
	return nil
}

// ── Internal helpers (called only from inside state functions) ─────

// computeTarget returns the target level index based on elapsed time
// WITHOUT mutating currentLevel.
func (bm *BlindManager) computeTarget() int {
	if bm.startTime.IsZero() {
		return 0
	}
	elapsed := time.Since(bm.startTime)
	target := int(elapsed / bm.interval)
	if target >= len(bm.schedule) {
		target = len(bm.schedule) - 1
	}
	return target
}

// advanceToTarget sets currentLevel to the target based on elapsed time.
// ONLY called from evBlindApply handler inside state functions.
func (bm *BlindManager) advanceToTarget() {
	target := bm.computeTarget()
	if target > bm.currentLevel {
		bm.currentLevel = target
	}
}

func (bm *BlindManager) notifyPending(nextLevel BlindLevel) {
	if bm.gameEventChan != nil {
		select {
		case bm.gameEventChan <- GameEvent{
			Type:      GameEventBlindsPending,
			NextBlind: &nextLevel,
		}:
		default:
		}
	}
}

func (bm *BlindManager) buildInfo(state BlindState) BlindInfo {
	info := BlindInfo{
		Level: bm.schedule[bm.currentLevel],
		Index: bm.currentLevel,
		State: state,
	}
	if state == BlindStateActive || state == BlindStatePending {
		nextTime := bm.startTime.Add(time.Duration(bm.currentLevel+1) * bm.interval)
		info.NextIncreaseUnixMs = nextTime.UnixMilli()
	}
	return info
}
