package poker

import (
	"sync/atomic"
)

// eventMetrics holds simple in-memory counters for table event flow.
// These are intentionally lightweight and concurrency-safe via atomics.
var eventMetrics struct {
	published atomic.Int64
	dropped   atomic.Int64
}

// IncrementEventPublished increments the published counter.
func IncrementEventPublished() { eventMetrics.published.Add(1) }

// IncrementEventDropped increments the dropped counter.
func IncrementEventDropped() { eventMetrics.dropped.Add(1) }

// ResetEventMetrics resets all counters. Intended for tests only.
func ResetEventMetrics() {
	eventMetrics.published.Store(0)
	eventMetrics.dropped.Store(0)
}

// EventMetricsSnapshot returns a copy of current counters.
type EventMetricsSnapshot struct {
	Published int64
	Dropped   int64
}

// GetEventMetricsSnapshot atomically reads the counters for external inspection.
func GetEventMetricsSnapshot() EventMetricsSnapshot {
	return EventMetricsSnapshot{
		Published: eventMetrics.published.Load(),
		Dropped:   eventMetrics.dropped.Load(),
	}
}
