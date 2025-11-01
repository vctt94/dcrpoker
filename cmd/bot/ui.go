package main

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"runtime"
	"sync"
	"time"

	"github.com/vctt94/pokerbisonrelay/pkg/poker"
	"github.com/vctt94/pokerbisonrelay/pkg/server"
)

//go:embed templates/*
var templatesFS embed.FS

var (
	statusTmpl     *template.Template
	statusTmplOnce sync.Once
)

// TableInfo summarizes a single table for metrics/reporting.
type TableInfo struct {
	ID          string
	Users       int
	OnlineUsers int
	ReadyUsers  int
	AllReady    bool
	GameActive  bool
}

// StatsSnapshot aggregates server-wide counts + per-table info for metrics.
type StatsSnapshot struct {
	Tables        int
	Users         int
	OnlineUsers   int
	ReadyUsers    int
	DistinctUsers int
	GamesActive   int
	TablesInfo    []TableInfo
}

// MetricValue represents a single metric with its value and description
type MetricValue struct {
	Name        string
	Description string
	Value       interface{}
	Type        string // "counter", "gauge", "histogram"
}

// MetricsData groups all metrics for user-friendly display
type MetricsData struct {
	Counters   []MetricValue
	Gauges     []MetricValue
	Histograms []HistogramData
}

// HistogramData represents a histogram metric
type HistogramData struct {
	Name        string
	Description string
	Buckets     []BucketData
	Sum         float64
	Count       int64
	Average     float64
}

// BucketData represents a single histogram bucket
type BucketData struct {
	UpperBound string
	Count      int64
}

type statusPageData struct {
	GeneratedAt       time.Time
	QueueDepth        int
	QueueCapacity     int
	EventDrops        int64
	GoRoutines        int
	Summary           StatsSnapshot
	PrometheusMetrics string
	Metrics           MetricsData
}

// Stats returns a snapshot of server-wide stats and per-table info.
func Stats(tbls []*poker.Table) StatsSnapshot {
	snap := StatsSnapshot{Tables: len(tbls)}
	userSeen := make(map[string]struct{})
	snap.TablesInfo = make([]TableInfo, 0, len(tbls))
	for _, t := range tbls {
		users := t.GetUsers()
		ti := TableInfo{ID: t.GetConfig().ID, Users: len(users)}
		for _, u := range users {
			if u.IsReady {
				ti.ReadyUsers++
			}
			if !u.IsDisconnected {
				ti.OnlineUsers++
				snap.OnlineUsers++
			}
			if _, ok := userSeen[u.ID]; !ok {
				userSeen[u.ID] = struct{}{}
				snap.Users++
				snap.DistinctUsers++
			}
		}
		ti.AllReady = t.AreAllPlayersReady()
		ti.GameActive = t.IsGameStarted()
		if ti.GameActive {
			snap.GamesActive++
		}
		snap.ReadyUsers += ti.ReadyUsers
		snap.TablesInfo = append(snap.TablesInfo, ti)
	}
	return snap
}

func parseTemplatesOnce() {
	statusTmplOnce.Do(func() {
		t, err := template.ParseFS(templatesFS, "templates/status.tmpl")
		if err != nil {
			// Fallback to a minimal template if parsing fails
			statusTmpl = template.Must(template.New("status").Parse(`<html><body><h1>Poker Status</h1>{{printf "%+v" .}}</body></html>`))
			return
		}
		statusTmpl = t
	})
}

// formatPrometheusMetrics generates Prometheus text format metrics for the given server state.
func formatPrometheusMetrics(srv *server.Server, sstats StatsSnapshot) string {
	var buf bytes.Buffer
	// Human-friendly summary (comments)
	fmt.Fprintf(&buf, "# Poker Server Summary\n")
	fmt.Fprintf(&buf, "# tables=%d users=%d distinct_users=%d online_users=%d ready_users=%d active_games=%d\n",
		sstats.Tables, sstats.Users, sstats.DistinctUsers, sstats.OnlineUsers, sstats.ReadyUsers, sstats.GamesActive)

	// Base counters/gauges
	snap := poker.GetEventMetricsSnapshot()
	evtDrops := snap.Dropped + server.GetMetrics().EventDropsTotal()
	fmt.Fprintf(&buf, "# HELP poker_event_drops_total Total events dropped (table + server)\n")
	fmt.Fprintf(&buf, "# TYPE poker_event_drops_total counter\n")
	fmt.Fprintf(&buf, "poker_event_drops_total %d\n", evtDrops)

	fmt.Fprintf(&buf, "# HELP poker_event_queue_depth Current depth of server event queue\n")
	fmt.Fprintf(&buf, "# TYPE poker_event_queue_depth gauge\n")
	fmt.Fprintf(&buf, "poker_event_queue_depth %d\n", srv.EventQueueDepth())

	fmt.Fprintf(&buf, "# HELP poker_event_queue_capacity Capacity of server event queue\n")
	fmt.Fprintf(&buf, "# TYPE poker_event_queue_capacity gauge\n")
	fmt.Fprintf(&buf, "poker_event_queue_capacity %d\n", srv.EventQueueCapacity())

	fmt.Fprintf(&buf, "# HELP poker_db_busy_total Number of DB busy/locked errors observed\n")
	fmt.Fprintf(&buf, "# TYPE poker_db_busy_total counter\n")
	fmt.Fprintf(&buf, "poker_db_busy_total %d\n", server.GetMetrics().DBBusyTotal())

	fmt.Fprintf(&buf, "# HELP go_goroutines Number of goroutines\n")
	fmt.Fprintf(&buf, "# TYPE go_goroutines gauge\n")
	fmt.Fprintf(&buf, "go_goroutines %d\n", runtime.NumGoroutine())

	// Server-wide gauges
	fmt.Fprintf(&buf, "# HELP poker_tables_total Number of tables (loaded/active)\n")
	fmt.Fprintf(&buf, "# TYPE poker_tables_total gauge\n")
	fmt.Fprintf(&buf, "poker_tables_total %d\n", sstats.Tables)

	fmt.Fprintf(&buf, "# HELP poker_users_total Total unique users across all tables\n")
	fmt.Fprintf(&buf, "# TYPE poker_users_total gauge\n")
	fmt.Fprintf(&buf, "poker_users_total %d\n", sstats.Users)

	fmt.Fprintf(&buf, "# HELP poker_distinct_users_total Distinct users across all tables\n")
	fmt.Fprintf(&buf, "# TYPE poker_distinct_users_total gauge\n")
	fmt.Fprintf(&buf, "poker_distinct_users_total %d\n", sstats.DistinctUsers)

	fmt.Fprintf(&buf, "# HELP poker_online_users_total Online users across all tables\n")
	fmt.Fprintf(&buf, "# TYPE poker_online_users_total gauge\n")
	fmt.Fprintf(&buf, "poker_online_users_total %d\n", sstats.OnlineUsers)

	fmt.Fprintf(&buf, "# HELP poker_ready_users_total Users marked ready across all tables\n")
	fmt.Fprintf(&buf, "# TYPE poker_ready_users_total gauge\n")
	fmt.Fprintf(&buf, "poker_ready_users_total %d\n", sstats.ReadyUsers)

	fmt.Fprintf(&buf, "# HELP poker_games_active_total Number of tables with active game\n")
	fmt.Fprintf(&buf, "# TYPE poker_games_active_total gauge\n")
	fmt.Fprintf(&buf, "poker_games_active_total %d\n", sstats.GamesActive)

	// Per-table gauges
	fmt.Fprintf(&buf, "# HELP poker_table_users Users per table\n")
	fmt.Fprintf(&buf, "# TYPE poker_table_users gauge\n")
	fmt.Fprintf(&buf, "# HELP poker_table_online_users Online users per table\n")
	fmt.Fprintf(&buf, "# TYPE poker_table_online_users gauge\n")
	fmt.Fprintf(&buf, "# HELP poker_table_active Whether table has active game (1/0)\n")
	fmt.Fprintf(&buf, "# TYPE poker_table_active gauge\n")
	fmt.Fprintf(&buf, "# HELP poker_table_all_ready Whether all players are ready (1/0)\n")
	fmt.Fprintf(&buf, "# TYPE poker_table_all_ready gauge\n")
	for _, ti := range sstats.TablesInfo {
		fmt.Fprintf(&buf, "poker_table_users{table=\"%s\"} %d\n", ti.ID, ti.Users)
		fmt.Fprintf(&buf, "poker_table_online_users{table=\"%s\"} %d\n", ti.ID, ti.OnlineUsers)
		if ti.GameActive {
			fmt.Fprintf(&buf, "poker_table_active{table=\"%s\"} 1\n", ti.ID)
		} else {
			fmt.Fprintf(&buf, "poker_table_active{table=\"%s\"} 0\n", ti.ID)
		}
		if ti.AllReady {
			fmt.Fprintf(&buf, "poker_table_all_ready{table=\"%s\"} 1\n", ti.ID)
		} else {
			fmt.Fprintf(&buf, "poker_table_all_ready{table=\"%s\"} 0\n", ti.ID)
		}
	}

	// Histogram exposition
	bks := server.GetMetrics().DBOpHistogram()
	cum := int64(0)
	for _, b := range bks.Buckets {
		cum += b.Count
		fmt.Fprintf(&buf, "poker_db_op_seconds_bucket{le=\"%s\"} %d\n", b.LE, cum)
	}
	fmt.Fprintf(&buf, "poker_db_op_seconds_sum %.6f\n", bks.SumSeconds)
	fmt.Fprintf(&buf, "poker_db_op_seconds_count %d\n", bks.Count)

	return buf.String()
}

// buildMetricsData creates structured metrics data for user-friendly display
func buildMetricsData(srv *server.Server, sstats StatsSnapshot) MetricsData {
	snap := poker.GetEventMetricsSnapshot()
	evtDrops := snap.Dropped + server.GetMetrics().EventDropsTotal()
	metrics := MetricsData{}

	// Counters
	metrics.Counters = []MetricValue{
		{
			Name:        "Event Drops (Total)",
			Description: "Total events dropped at table and server layers",
			Value:       evtDrops,
			Type:        "counter",
		},
		{
			Name:        "DB Busy Errors",
			Description: "Number of database busy/locked errors observed",
			Value:       server.GetMetrics().DBBusyTotal(),
			Type:        "counter",
		},
	}

	// Gauges - Server Operations (only metrics not shown in summary cards)
	metrics.Gauges = []MetricValue{
		{
			Name:        "Event Queue Depth",
			Description: "Current depth of server event queue",
			Value:       fmt.Sprintf("%d / %d", srv.EventQueueDepth(), srv.EventQueueCapacity()),
			Type:        "gauge",
		},
		{
			Name:        "Goroutines",
			Description: "Number of active goroutines",
			Value:       runtime.NumGoroutine(),
			Type:        "gauge",
		},
	}

	// Histograms
	bks := server.GetMetrics().DBOpHistogram()
	histBuckets := make([]BucketData, len(bks.Buckets))
	cum := int64(0)
	for i, b := range bks.Buckets {
		cum += b.Count
		histBuckets[i] = BucketData{
			UpperBound: b.LE,
			Count:      cum,
		}
	}

	avg := float64(0)
	if bks.Count > 0 {
		avg = bks.SumSeconds / float64(bks.Count)
	}
	metrics.Histograms = []HistogramData{
		{
			Name:        "Database Operation Latency",
			Description: "Distribution of database operation latencies in seconds",
			Buckets:     histBuckets,
			Sum:         bks.SumSeconds,
			Count:       bks.Count,
			Average:     avg,
		},
	}

	return metrics
}
