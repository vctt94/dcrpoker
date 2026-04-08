package server

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/vctt94/pokerbisonrelay/pkg/poker"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	"github.com/vctt94/pokerbisonrelay/pkg/server/internal/db"
)

type matchCheckpointPayload struct {
	Version                 int                    `json:"version"`
	Round                   int                    `json:"round"`
	Dealer                  int                    `json:"dealer"`
	Players                 []poker.PlayerSnapshot `json:"players"`
	Blind                   poker.BlindSnapshot    `json:"blind"`
	SmallBlind              int64                  `json:"small_blind"`
	BigBlind                int64                  `json:"big_blind"`
	BlindLevel              int                    `json:"blind_level"`
	NextBlindIncreaseUnixMs int64                  `json:"next_blind_increase_unix_ms"`
	LastShowdown            *poker.ShowdownResult  `json:"last_showdown,omitempty"`
}

func (s *Server) shouldCheckpointTable(table *poker.Table) bool {
	if table == nil {
		return false
	}
	game := table.GetGame()
	if game == nil {
		return false
	}

	switch game.GetPhase() {
	case pokerrpc.GamePhase_SHOWDOWN, pokerrpc.GamePhase_NEW_HAND_DEALING:
		return true
	default:
		return false
	}
}

func (s *Server) buildMatchCheckpointPayload(table *poker.Table) (*matchCheckpointPayload, error) {
	if table == nil {
		return nil, fmt.Errorf("table is nil")
	}
	game := table.GetGame()
	if game == nil {
		return nil, fmt.Errorf("game not found")
	}

	snap := game.GetStateSnapshot()
	payload := &matchCheckpointPayload{
		Version:                 1,
		Round:                   snap.Round,
		Dealer:                  snap.Dealer,
		Players:                 snap.Players,
		Blind:                   game.GetBlindSnapshot(),
		SmallBlind:              snap.SmallBlind,
		BigBlind:                snap.BigBlind,
		BlindLevel:              snap.BlindLevel,
		NextBlindIncreaseUnixMs: snap.NextBlindIncreaseUnixMs,
		LastShowdown:            table.GetLastShowdown(),
	}
	return payload, nil
}

func (s *Server) saveMatchCheckpoint(tableID string) error {
	table, ok := s.getTable(tableID)
	if !ok {
		return fmt.Errorf("table not found")
	}
	if !s.shouldCheckpointTable(table) {
		return nil
	}

	payload, err := s.buildMatchCheckpointPayload(table)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal match checkpoint: %w", err)
	}

	return s.db.UpsertMatchCheckpoint(context.Background(), db.MatchCheckpoint{
		TableID:   tableID,
		UpdatedAt: time.Now(),
		Payload:   raw,
	})
}

func (s *Server) saveMatchCheckpointSync(tableID string, reason string) error {
	v, _ := s.saveMutexes.LoadOrStore(tableID, &sync.Mutex{})
	saveMutex, _ := v.(*sync.Mutex)

	saveMutex.Lock()
	defer saveMutex.Unlock()

	if err := s.saveMatchCheckpoint(tableID); err != nil {
		switch err.Error() {
		case "table not found":
			s.log.Debugf("Skipping match checkpoint save for removed table %s (%s)", tableID, reason)
			return nil
		default:
			return err
		}
	}
	return nil
}

func (s *Server) saveMatchCheckpointAsync(tableID string, reason string) {
	s.saveWg.Add(1)
	go func() {
		defer s.saveWg.Done()
		if err := s.saveMatchCheckpointSync(tableID, reason); err != nil {
			s.log.Errorf("Failed to save match checkpoint for %s (%s): %v", tableID, reason, err)
		}
	}()
}

func (s *Server) saveAllMatchCheckpointsSync(reason string) {
	for _, table := range s.getAllTables() {
		tableID := table.GetConfig().ID
		if err := s.saveMatchCheckpointSync(tableID, reason); err != nil {
			s.log.Errorf("Failed to save match checkpoint for %s (%s): %v", tableID, reason, err)
		}
	}
}

func (s *Server) loadMatchCheckpoint(tableID string, table *poker.Table) error {
	if table == nil {
		return fmt.Errorf("table is nil")
	}

	checkpoint, err := s.db.GetMatchCheckpoint(context.Background(), tableID)
	if err != nil {
		return err
	}
	if checkpoint == nil || len(checkpoint.Payload) == 0 {
		return nil
	}

	var payload matchCheckpointPayload
	if err := json.Unmarshal(checkpoint.Payload, &payload); err != nil {
		return fmt.Errorf("unmarshal match checkpoint: %w", err)
	}

	return table.RestorePausedMatch(poker.MatchStateSnapshot{
		Round:                   payload.Round,
		Dealer:                  payload.Dealer,
		Players:                 payload.Players,
		Blind:                   payload.Blind,
		SmallBlind:              payload.SmallBlind,
		BigBlind:                payload.BigBlind,
		BlindLevel:              payload.BlindLevel,
		NextBlindIncreaseUnixMs: payload.NextBlindIncreaseUnixMs,
		LastShowdown:            payload.LastShowdown,
	})
}
