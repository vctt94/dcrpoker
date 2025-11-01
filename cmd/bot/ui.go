package main

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
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

type statusPageData struct {
	GeneratedAt   time.Time
	QueueDepth    int
	QueueCapacity int
	EventDrops    int64
	GoRoutines    int
	Summary       StatsSnapshot
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

func handleMetrics(srv *server.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get tables before calling Stats()
		tables := srv.GetAllTables()
		// Human-friendly summary (comments)
		sstats := Stats(tables)
		fmt.Fprintf(w, "# Poker Server Summary\n")
		fmt.Fprintf(w, "# tables=%d users=%d distinct_users=%d online_users=%d ready_users=%d active_games=%d\n",
			sstats.Tables, sstats.Users, sstats.DistinctUsers, sstats.OnlineUsers, sstats.ReadyUsers, sstats.GamesActive)

		// Base counters/gauges
		snap := poker.GetEventMetricsSnapshot()
		evtDrops := snap.Dropped + server.GetMetrics().EventDropsTotal()
		fmt.Fprintf(w, "# HELP poker_event_drops_total Total events dropped (table + server)\n")
		fmt.Fprintf(w, "# TYPE poker_event_drops_total counter\n")
		fmt.Fprintf(w, "poker_event_drops_total %d\n", evtDrops)

		fmt.Fprintf(w, "# HELP poker_event_queue_depth Current depth of server event queue\n")
		fmt.Fprintf(w, "# TYPE poker_event_queue_depth gauge\n")
		fmt.Fprintf(w, "poker_event_queue_depth %d\n", srv.EventQueueDepth())

		fmt.Fprintf(w, "# HELP poker_event_queue_capacity Capacity of server event queue\n")
		fmt.Fprintf(w, "# TYPE poker_event_queue_capacity gauge\n")
		fmt.Fprintf(w, "poker_event_queue_capacity %d\n", srv.EventQueueCapacity())

		fmt.Fprintf(w, "# HELP poker_db_busy_total Number of DB busy/locked errors observed\n")
		fmt.Fprintf(w, "# TYPE poker_db_busy_total counter\n")
		fmt.Fprintf(w, "poker_db_busy_total %d\n", server.GetMetrics().DBBusyTotal())

		fmt.Fprintf(w, "# HELP go_goroutines Number of goroutines\n")
		fmt.Fprintf(w, "# TYPE go_goroutines gauge\n")
		fmt.Fprintf(w, "go_goroutines %d\n", runtime.NumGoroutine())

		// Server-wide gauges
		fmt.Fprintf(w, "# HELP poker_tables_total Number of tables (loaded/active)\n")
		fmt.Fprintf(w, "# TYPE poker_tables_total gauge\n")
		fmt.Fprintf(w, "poker_tables_total %d\n", sstats.Tables)

		fmt.Fprintf(w, "# HELP poker_users_total Total unique users across all tables\n")
		fmt.Fprintf(w, "# TYPE poker_users_total gauge\n")
		fmt.Fprintf(w, "poker_users_total %d\n", sstats.Users)

		fmt.Fprintf(w, "# HELP poker_distinct_users_total Distinct users across all tables\n")
		fmt.Fprintf(w, "# TYPE poker_distinct_users_total gauge\n")
		fmt.Fprintf(w, "poker_distinct_users_total %d\n", sstats.DistinctUsers)

		fmt.Fprintf(w, "# HELP poker_online_users_total Online users across all tables\n")
		fmt.Fprintf(w, "# TYPE poker_online_users_total gauge\n")
		fmt.Fprintf(w, "poker_online_users_total %d\n", sstats.OnlineUsers)

		fmt.Fprintf(w, "# HELP poker_ready_users_total Users marked ready across all tables\n")
		fmt.Fprintf(w, "# TYPE poker_ready_users_total gauge\n")
		fmt.Fprintf(w, "poker_ready_users_total %d\n", sstats.ReadyUsers)

		fmt.Fprintf(w, "# HELP poker_games_active_total Number of tables with active game\n")
		fmt.Fprintf(w, "# TYPE poker_games_active_total gauge\n")
		fmt.Fprintf(w, "poker_games_active_total %d\n", sstats.GamesActive)

		// Per-table gauges
		fmt.Fprintf(w, "# HELP poker_table_users Users per table\n")
		fmt.Fprintf(w, "# TYPE poker_table_users gauge\n")
		fmt.Fprintf(w, "# HELP poker_table_online_users Online users per table\n")
		fmt.Fprintf(w, "# TYPE poker_table_online_users gauge\n")
		fmt.Fprintf(w, "# HELP poker_table_active Whether table has active game (1/0)\n")
		fmt.Fprintf(w, "# TYPE poker_table_active gauge\n")
		fmt.Fprintf(w, "# HELP poker_table_all_ready Whether all players are ready (1/0)\n")
		fmt.Fprintf(w, "# TYPE poker_table_all_ready gauge\n")
		for _, ti := range sstats.TablesInfo {
			fmt.Fprintf(w, "poker_table_users{table=\"%s\"} %d\n", ti.ID, ti.Users)
			fmt.Fprintf(w, "poker_table_online_users{table=\"%s\"} %d\n", ti.ID, ti.OnlineUsers)
			if ti.GameActive {
				fmt.Fprintf(w, "poker_table_active{table=\"%s\"} 1\n", ti.ID)
			} else {
				fmt.Fprintf(w, "poker_table_active{table=\"%s\"} 0\n", ti.ID)
			}
			if ti.AllReady {
				fmt.Fprintf(w, "poker_table_all_ready{table=\"%s\"} 1\n", ti.ID)
			} else {
				fmt.Fprintf(w, "poker_table_all_ready{table=\"%s\"} 0\n", ti.ID)
			}
		}

		// Histogram exposition
		bks := server.GetMetrics().DBOpHistogram()
		cum := int64(0)
		for _, b := range bks.Buckets {
			cum += b.Count
			fmt.Fprintf(w, "poker_db_op_seconds_bucket{le=\"%s\"} %d\n", b.LE, cum)
		}
		fmt.Fprintf(w, "poker_db_op_seconds_sum %.6f\n", bks.SumSeconds)
		fmt.Fprintf(w, "poker_db_op_seconds_count %d\n", bks.Count)
	}
}
