package server

import (
	"testing"
	"time"

	"github.com/decred/slog"
)

func TestShutdownCoordinatorWaitsForHandFinished(t *testing.T) {
	coord := newShutdownCoordinator(slog.Disabled)
	defer coord.Stop()

	coord.BeginDrain([]string{"table-1"})
	coord.NotifyDrainStarted("table-1", true)

	done := make(chan struct{})
	go func() {
		coord.WaitForQuiescence()
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("drain completed before the active hand finished")
	case <-time.After(100 * time.Millisecond):
	}

	coord.NotifyHandFinished("table-1", "SHOWDOWN_RESULT")

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("drain did not complete after the active hand finished")
	}
}

func TestShutdownCoordinatorIgnoresLateDrainStartedAfterHandFinished(t *testing.T) {
	coord := newShutdownCoordinator(slog.Disabled)
	defer coord.Stop()

	coord.BeginDrain([]string{"table-1"})
	coord.NotifyHandFinished("table-1", "SHOWDOWN_RESULT")
	coord.NotifyDrainStarted("table-1", true)

	done := make(chan struct{})
	go func() {
		coord.WaitForQuiescence()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("late drain-started event reopened a quiesced table")
	}
}
