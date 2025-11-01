package server

import (
	"math"
	"strconv"
	"sync/atomic"
	"time"
)

// Metrics aggregates lightweight counters and histograms for the server.
type Metrics struct {
	// Dropped events observed at the server layer (e.g., event processor drops).
	droppedEvents atomic.Int64

	// DB busy/locked errors observed by the instrumented DB wrapper.
	dbBusy atomic.Int64

	// DB operation latency histogram (coarse) in seconds.
	// We store counts per bucket and track sum and count.
	buckets     []float64      // upper bounds in seconds; last is +Inf
	bucketCount []atomic.Int64 // counters per bucket
	opCount     atomic.Int64   // total observations
	sumNanos    atomic.Int64   // sum of observed durations in nanoseconds
}

var globalMetrics = newDefaultMetrics()

func newDefaultMetrics() *Metrics {
	b := []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, math.Inf(1)}
	m := &Metrics{
		buckets:     b,
		bucketCount: make([]atomic.Int64, len(b)),
	}
	return m
}

// GetMetrics returns the global metrics instance.
func GetMetrics() *Metrics { return globalMetrics }

// IncEventDrop increments the server-level event drop counter.
func (m *Metrics) IncEventDrop() { m.droppedEvents.Add(1) }

// EventDropsTotal returns current server-level event drops total.
func (m *Metrics) EventDropsTotal() int64 { return m.droppedEvents.Load() }

// IncDBBusy increments the DB busy/locked counter.
func (m *Metrics) IncDBBusy() { m.dbBusy.Add(1) }

// DBBusyTotal returns current DB busy/locked total.
func (m *Metrics) DBBusyTotal() int64 { return m.dbBusy.Load() }

// ObserveDBOp records a DB operation latency.
func (m *Metrics) ObserveDBOp(d time.Duration) {
	secs := float64(d) / float64(time.Second)
	// find bucket index
	idx := 0
	for i, ub := range m.buckets {
		if secs <= ub {
			idx = i
			break
		}
	}
	m.bucketCount[idx].Add(1)
	m.opCount.Add(1)
	m.sumNanos.Add(d.Nanoseconds())
}

// DBOpHistogram returns a snapshot suitable for Prometheus exposition.
type HistogramSnapshot struct {
	Buckets    []Bucket
	Count      int64
	SumSeconds float64
}

type Bucket struct {
	LE    string
	Count int64
}

func (m *Metrics) DBOpHistogram() HistogramSnapshot {
	out := HistogramSnapshot{Buckets: make([]Bucket, len(m.buckets))}
	for i, ub := range m.buckets {
		le := "+Inf"
		if !math.IsInf(ub, 1) {
			le = strconv.FormatFloat(ub, 'f', 3, 64)
		}
		out.Buckets[i] = Bucket{LE: le, Count: m.bucketCount[i].Load()}
	}
	out.Count = m.opCount.Load()
	out.SumSeconds = float64(m.sumNanos.Load()) / float64(time.Second)
	return out
}
