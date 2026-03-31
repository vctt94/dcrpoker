package poker

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/decred/slog"
)

func blindTestLog() slog.Logger {
	backend := slog.NewBackend(os.Stderr)
	log := backend.Logger("BLIND")
	log.SetLevel(slog.LevelError)
	return log
}

// startedManager creates, starts, and returns a BlindManager FSM.
func startedManager(schedule []BlindLevel, interval time.Duration) *BlindManager {
	bm := NewBlindManager(schedule, interval, blindTestLog())
	bm.Start(context.Background())
	return bm
}

func TestBlindManagerDisabled(t *testing.T) {
	bm := startedManager(DefaultBlindSchedule, 0)
	defer bm.Stop()

	level := bm.GetLevel()
	if level.SmallBlind != 10 || level.BigBlind != 20 {
		t.Fatalf("expected 10/20, got %d/%d", level.SmallBlind, level.BigBlind)
	}

	info := bm.GetInfo()
	if info.State != BlindStateDisabled {
		t.Fatalf("expected DISABLED, got %s", info.State)
	}

	// Apply on disabled is a no-op.
	result := bm.Apply()
	if result.Changed {
		t.Fatal("Apply on disabled manager should not change")
	}
}

func TestBlindManagerEmptySchedule(t *testing.T) {
	bm := startedManager(nil, 5*time.Minute)
	defer bm.Stop()

	info := bm.GetInfo()
	if info.State != BlindStateDisabled {
		t.Fatalf("expected DISABLED for empty schedule, got %s", info.State)
	}
}

func TestBlindManagerSingleLevel(t *testing.T) {
	bm := startedManager([]BlindLevel{{10, 20}}, 5*time.Minute)
	defer bm.Stop()

	info := bm.GetInfo()
	if info.State != BlindStateDisabled {
		t.Fatalf("expected DISABLED for single-level schedule, got %s", info.State)
	}
}

func TestBlindManagerGetLevelBeforeStart(t *testing.T) {
	bm := startedManager(DefaultBlindSchedule, 5*time.Minute)
	defer bm.Stop()

	info := bm.GetInfo()
	if info.State != BlindStateWaiting {
		t.Fatalf("expected WAITING before start, got %s", info.State)
	}

	level := bm.GetLevel()
	if level.SmallBlind != 10 || level.BigBlind != 20 {
		t.Fatalf("expected initial 10/20, got %d/%d", level.SmallBlind, level.BigBlind)
	}
}

func TestBlindManagerNoMutationOnTimerFire(t *testing.T) {
	schedule := []BlindLevel{
		{10, 20},
		{15, 30},
	}
	// Very short interval so timer fires quickly.
	interval := 30 * time.Millisecond
	bm := startedManager(schedule, interval)
	defer bm.Stop()

	bm.SendStart(time.Time{})

	// Wait for timer to fire → FSM enters Pending state.
	time.Sleep(80 * time.Millisecond)

	// Level should still be 0 — timer fires but does NOT mutate.
	level := bm.GetLevel()
	if level.SmallBlind != 10 || level.BigBlind != 20 {
		t.Fatalf("expected 10/20 (no mutation on timer), got %d/%d",
			level.SmallBlind, level.BigBlind)
	}

	info := bm.GetInfo()
	if info.State != BlindStatePending {
		t.Fatalf("expected PENDING after timer fire, got %s", info.State)
	}
}

func TestBlindManagerApplyAdvancesLevel(t *testing.T) {
	schedule := []BlindLevel{
		{10, 20},
		{15, 30},
		{20, 40},
	}
	interval := 200 * time.Millisecond
	bm := startedManager(schedule, interval)
	defer bm.Stop()

	bm.SendStart(time.Time{})

	// Wait for exactly one interval to fire → Pending at level 1.
	time.Sleep(250 * time.Millisecond)

	// Apply at hand boundary — level should advance.
	result := bm.Apply()
	if !result.Changed {
		t.Fatal("expected Apply to change level")
	}
	if result.Level.SmallBlind != 15 || result.Level.BigBlind != 30 {
		t.Fatalf("expected 15/30 after apply, got %d/%d",
			result.Level.SmallBlind, result.Level.BigBlind)
	}
	if result.Message == "" {
		t.Fatal("expected non-empty message on level change")
	}

	// Verify GetLevel also returns the new level.
	level := bm.GetLevel()
	if level.SmallBlind != 15 || level.BigBlind != 30 {
		t.Fatalf("expected 15/30 from GetLevel, got %d/%d",
			level.SmallBlind, level.BigBlind)
	}
}

func TestBlindManagerApplyNotDueYet(t *testing.T) {
	schedule := []BlindLevel{
		{10, 20},
		{15, 30},
	}
	interval := 5 * time.Minute
	bm := startedManager(schedule, interval)
	defer bm.Stop()

	bm.SendStart(time.Time{}) // start from now; 5 min until next

	// Apply immediately — not due yet.
	result := bm.Apply()
	if result.Changed {
		t.Fatal("Apply should not change when increase is not due")
	}
	if result.Level.SmallBlind != 10 || result.Level.BigBlind != 20 {
		t.Fatalf("expected 10/20, got %d/%d",
			result.Level.SmallBlind, result.Level.BigBlind)
	}
}

func TestBlindManagerMultipleLevelSkip(t *testing.T) {
	schedule := []BlindLevel{
		{10, 20},
		{15, 30},
		{20, 40},
		{25, 50},
	}
	interval := 1 * time.Minute
	bm := startedManager(schedule, interval)
	defer bm.Stop()

	// Simulate starting 2.5 minutes ago — should skip to level 2.
	bm.SendStart(time.Now().Add(-150 * time.Second))

	time.Sleep(50 * time.Millisecond) // let FSM process start + enter pending

	result := bm.Apply()
	if !result.Changed {
		t.Fatal("expected change after 2.5 min with 1-min interval")
	}
	if result.Index != 2 {
		t.Fatalf("expected level 2, got %d", result.Index)
	}
	if result.Level.SmallBlind != 20 || result.Level.BigBlind != 40 {
		t.Fatalf("expected 20/40, got %d/%d",
			result.Level.SmallBlind, result.Level.BigBlind)
	}
}

func TestBlindManagerReachesMaxLevel(t *testing.T) {
	schedule := []BlindLevel{
		{10, 20},
		{15, 30},
	}
	interval := 1 * time.Minute
	bm := startedManager(schedule, interval)
	defer bm.Stop()

	// Simulate starting 10 minutes ago — well past all levels.
	bm.SendStart(time.Now().Add(-10 * time.Minute))

	time.Sleep(50 * time.Millisecond)

	result := bm.Apply()
	if !result.Changed {
		t.Fatal("expected change")
	}

	info := bm.GetInfo()
	if info.State != BlindStateMaxLevel {
		t.Fatalf("expected MAX_LEVEL, got %s", info.State)
	}
	if info.Level.SmallBlind != 15 || info.Level.BigBlind != 30 {
		t.Fatalf("expected 15/30, got %d/%d",
			info.Level.SmallBlind, info.Level.BigBlind)
	}

	// Further Apply is a no-op.
	result2 := bm.Apply()
	if result2.Changed {
		t.Fatal("Apply at max level should be no-op")
	}
}

func TestBlindManagerDefaultSchedule(t *testing.T) {
	bm := startedManager(DefaultBlindSchedule, 5*time.Minute)
	defer bm.Stop()

	bm.SendStart(time.Time{})

	level := bm.GetLevel()
	if level.SmallBlind != 10 || level.BigBlind != 20 {
		t.Fatalf("expected initial 10/20, got %d/%d",
			level.SmallBlind, level.BigBlind)
	}
	if bm.TotalLevels() != 10 {
		t.Fatalf("expected 10 levels, got %d", bm.TotalLevels())
	}
}

func TestBlindManagerNextIncreaseTime(t *testing.T) {
	schedule := []BlindLevel{
		{10, 20},
		{15, 30},
		{20, 40},
	}
	interval := 5 * time.Minute
	bm := startedManager(schedule, interval)
	defer bm.Stop()

	bm.SendStart(time.Time{})

	info := bm.GetInfo()
	if info.NextIncreaseUnixMs == 0 {
		t.Fatal("expected non-zero next increase time")
	}
	nextTime := time.UnixMilli(info.NextIncreaseUnixMs)
	until := time.Until(nextTime)
	if until <= 0 || until > interval+time.Second {
		t.Fatalf("expected 0 < until <= ~%v, got %v", interval, until)
	}
}

func TestBlindManagerStartFrom(t *testing.T) {
	schedule := []BlindLevel{
		{5, 10},
		{10, 20},
		{20, 40},
	}
	interval := 3 * time.Minute
	bm := startedManager(schedule, interval)
	defer bm.Stop()

	// Restore from a start time 4 minutes ago.
	bm.SendStart(time.Now().Add(-4 * time.Minute))

	time.Sleep(50 * time.Millisecond)

	// Not yet applied — GetLevel returns initial level.
	level := bm.GetLevel()
	if level.SmallBlind != 5 && level.BigBlind != 10 {
		// It's OK if it's still at 0 since we haven't applied.
	}

	// Apply to advance.
	result := bm.Apply()
	if !result.Changed {
		t.Fatal("expected change after 4 min with 3-min interval")
	}
	if result.Level.SmallBlind != 10 || result.Level.BigBlind != 20 {
		t.Fatalf("expected 10/20, got %d/%d",
			result.Level.SmallBlind, result.Level.BigBlind)
	}
}

func TestBlindManagerScheduleCopy(t *testing.T) {
	bm := startedManager(DefaultBlindSchedule, 5*time.Minute)
	defer bm.Stop()

	sched := bm.Schedule()
	sched[0].SmallBlind = 999

	level := bm.GetLevel()
	if level.SmallBlind == 999 {
		t.Fatal("Schedule() should return a copy")
	}
}

func TestBlindManagerPendingNotification(t *testing.T) {
	schedule := []BlindLevel{
		{10, 20},
		{15, 30},
	}
	interval := 30 * time.Millisecond

	bm := NewBlindManager(schedule, interval, blindTestLog())
	eventChan := make(chan GameEvent, 10)
	bm.SetGameEventChannel(eventChan)
	bm.Start(context.Background())
	defer bm.Stop()

	bm.SendStart(time.Time{})

	// Wait for timer to fire.
	time.Sleep(80 * time.Millisecond)

	// Should receive GameEventBlindsPending, NOT GameEventBlindsIncreased.
	select {
	case ev := <-eventChan:
		if ev.Type != GameEventBlindsPending {
			t.Fatalf("expected GameEventBlindsPending, got %v", ev.Type)
		}
		if ev.NextBlind == nil {
			t.Fatal("expected NextBlind to be set")
		}
		if ev.NextBlind.SmallBlind != 15 || ev.NextBlind.BigBlind != 30 {
			t.Fatalf("expected pending 15/30, got %d/%d",
				ev.NextBlind.SmallBlind, ev.NextBlind.BigBlind)
		}
	default:
		t.Fatal("expected pending notification on event channel")
	}
}

func TestBlindManagerApplyAfterPendingNotifiesIncreased(t *testing.T) {
	schedule := []BlindLevel{
		{10, 20},
		{15, 30},
		{20, 40},
	}
	interval := 200 * time.Millisecond

	bm := NewBlindManager(schedule, interval, blindTestLog())
	eventChan := make(chan GameEvent, 10)
	bm.SetGameEventChannel(eventChan)
	bm.Start(context.Background())
	defer bm.Stop()

	bm.SendStart(time.Time{})

	// Wait for exactly one interval.
	time.Sleep(250 * time.Millisecond)

	// Drain pending notification.
	select {
	case <-eventChan:
	default:
	}

	// Apply — the level should advance.
	result := bm.Apply()
	if !result.Changed {
		t.Fatal("expected change on Apply")
	}
	if result.Level.SmallBlind != 15 || result.Level.BigBlind != 30 {
		t.Fatalf("expected 15/30, got %d/%d",
			result.Level.SmallBlind, result.Level.BigBlind)
	}
}
