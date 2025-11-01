package poker

import (
    "testing"

    "github.com/decred/slog"
    "github.com/stretchr/testify/require"
    "github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
)

// TestEventChannel_Critical_NoSilentDrop ensures table event drops are counted and not silent
func TestEventChannel_Critical_NoSilentDrop(t *testing.T) {
    ResetEventMetrics()

    cfg := TableConfig{
        ID:      "tbl-evt",
        Log:     slog.Disabled,
        GameLog: slog.Disabled,
        // Minimal config sufficient for table construction
        MinPlayers:       2,
        MaxPlayers:       2,
        AutoAdvanceDelay: 1, // non-zero to avoid zero values in some paths
    }
    tbl := NewTable(cfg)
    defer tbl.Close()

    ch := make(chan TableEvent, 1)
    tbl.SetEventChannel(ch)

    // Publish first event → should be enqueued
    tbl.PublishEvent(pokerrpc.NotificationType_CHECK_MADE, cfg.ID, nil)
    require.Equal(t, 1, len(ch))

    // Publish second while buffer is full → dropped, not blocked
    tbl.PublishEvent(pokerrpc.NotificationType_PLAYER_FOLDED, cfg.ID, nil)
    require.Equal(t, 1, len(ch))

    snap := GetEventMetricsSnapshot()
    require.EqualValues(t, 1, snap.Published, "published count")
    require.EqualValues(t, 1, snap.Dropped, "dropped count")
}

// TestEventMetrics_Counters validates counters increment on publish/drop
func TestEventMetrics_Counters(t *testing.T) {
    ResetEventMetrics()

    cfg := TableConfig{
        ID:      "tbl-evt-metrics",
        Log:     slog.Disabled,
        GameLog: slog.Disabled,
        MinPlayers: 2,
        MaxPlayers: 2,
        AutoAdvanceDelay: 1,
    }
    tbl := NewTable(cfg)
    defer tbl.Close()

    ch := make(chan TableEvent, 2)
    tbl.SetEventChannel(ch)

    // Two publishes should both succeed
    tbl.PublishEvent(pokerrpc.NotificationType_BET_MADE, cfg.ID, nil)
    tbl.PublishEvent(pokerrpc.NotificationType_CALL_MADE, cfg.ID, nil)
    require.Equal(t, 2, len(ch))

    // One more publish when full should drop
    tbl.PublishEvent(pokerrpc.NotificationType_CHECK_MADE, cfg.ID, nil)

    snap := GetEventMetricsSnapshot()
    require.EqualValues(t, 2, snap.Published)
    require.EqualValues(t, 1, snap.Dropped)
}

