package server

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	"github.com/vctt94/pokerbisonrelay/pkg/server/internal/db"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// buildGameState creates a GameUpdate for the requesting player
func (s *Server) buildGameState(tableID, requestingPlayerID string) (*pokerrpc.GameUpdate, error) {
	// Fetch table pointer without coarse-grained server locking.
	table, ok := s.getTable(tableID)
	if !ok {
		return nil, status.Error(codes.NotFound, "table not found")
	}

	game := table.GetGame()

	return s.buildGameStateForPlayer(table, game, requestingPlayerID), nil
}

// saveTableState persists a fast-restore snapshot (opaque JSON blob) to the DB.
// Canonical state is history (hands/actions); this is only a cache to speed up
// warm starts and reconnects.
func (s *Server) saveTableState(tableID string) error {
	table, ok := s.getTable(tableID)
	if !ok {
		return fmt.Errorf("table not found")
	}

	// Take an atomic snapshot from the runtime (table implements this).
	// This should contain everything you want for quick hydration:
	// config, users (seats/ready), and the game's own snapshot if you include it.
	tableSnapshot := table.GetStateSnapshot()

	// Marshal to JSON (opaque payload for db.table_snapshots).
	payload, err := json.Marshal(tableSnapshot)
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}

	// Upsert snapshot in the DB.
	ctx := context.Background()
	err = s.db.UpsertSnapshot(ctx, db.Snapshot{
		TableID:    tableID,
		SnapshotAt: time.Now(),
		Payload:    payload,
	})
	if err != nil {
		return fmt.Errorf("upsert snapshot: %w", err)
	}

	return nil
}

// saveTableStateAsync saves table state asynchronously to avoid blocking game operations
func (s *Server) saveTableStateAsync(tableID string, reason string) {
	// Get or create a mutex for this table using concurrent map
	v, _ := s.saveMutexes.LoadOrStore(tableID, &sync.Mutex{})
	saveMutex, _ := v.(*sync.Mutex)

	// Track this goroutine
	s.saveWg.Add(1)

	go func() {
		defer s.saveWg.Done()
		saveMutex.Lock()
		defer saveMutex.Unlock()

		if err := s.saveTableState(tableID); err != nil {
			s.log.Errorf("Failed to save table state for %s (%s): %v", tableID, reason, err)
		}
	}()
}
