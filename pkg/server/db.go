// Package server wires the normalized DB into the table/game runtime.
// It restores only durable facts (table config + active seats). Live, per-hand
// state is created by the poker engine on demand (or via a snapshot fast-path
// you can add later).
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/vctt94/pokerbisonrelay/pkg/poker"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	"github.com/vctt94/pokerbisonrelay/pkg/server/internal/db"
)

// Database is the minimal surface the server needs from the storage layer.
type Database interface {
	// ---- Players / wallet ----
	GetPlayerBalance(ctx context.Context, playerID string) (int64, error)
	UpdatePlayerBalance(ctx context.Context, playerID string, amount int64, transactionType, description string) error
	UpsertSnapshot(ctx context.Context, s db.Snapshot) error
	GetSnapshot(ctx context.Context, tableID string) (*db.Snapshot, error)
	// ---- Tables (configuration) ----
	UpsertTable(ctx context.Context, t *poker.TableConfig) error
	GetTable(ctx context.Context, id string) (*db.Table, error)
	DeleteTable(ctx context.Context, id string) error
	ListTableIDs(ctx context.Context) ([]string, error)

	// ---- Participants ----
	ActiveParticipants(ctx context.Context, tableID string) ([]db.Participant, error)
	SeatPlayer(ctx context.Context, tableID, playerID string, seat int) error
	UnseatPlayer(ctx context.Context, tableID, playerID string) error

	// ---- Close ----
	Close() error
}

// Transaction kept for compatibility if referenced elsewhere.
type Transaction struct {
	ID          int64
	PlayerID    string
	Amount      int64
	Type        string
	Description string
	CreatedAt   string
}

// NewDatabase ensures the DB directory exists and opens/initializes SQLite.
func NewDatabase(dbPath string) (Database, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}
	return db.NewDB(dbPath)
}

// loadTableFromDatabase restores a table config + currently seated players.
// It intentionally does NOT resurrect an in-flight hand; the game engine
// will start a new hand when appropriate (or you can plug in snapshot restore).
func (s *Server) loadTableFromDatabase(tableID string) (*poker.Table, error) {
	ctx := context.Background()

	// 1) Load table config
	tcfg, err := s.db.GetTable(ctx, tableID)
	if err != nil {
		return nil, fmt.Errorf("failed to load table config: %w", err)
	}

	// 2) Build poker.TableConfig (DB stores ms; convert to Duration)
	tblLog := s.logBackend.Logger("TABLE")
	gameLog := s.logBackend.Logger("GAME")

	timeBankDur := time.Duration(tcfg.TimebankMS) * time.Millisecond
	autoStartDur := time.Duration(tcfg.AutoStartMS) * time.Millisecond
	autoAdvanceDur := time.Duration(tcfg.AutoAdvanceMS) * time.Millisecond

	cfg := poker.TableConfig{
		ID:               tcfg.ID,
		Log:              tblLog,
		GameLog:          gameLog,
		HostID:           tcfg.HostID,
		BuyIn:            tcfg.BuyIn,
		MinPlayers:       tcfg.MinPlayers,
		MaxPlayers:       tcfg.MaxPlayers,
		SmallBlind:       tcfg.SmallBlind,
		BigBlind:         tcfg.BigBlind,
		MinBalance:       tcfg.MinBalance,
		StartingChips:    tcfg.StartingChips,
		TimeBank:         timeBankDur,
		AutoStartDelay:   autoStartDur,
		AutoAdvanceDelay: autoAdvanceDur,
	}

	// 3) Create in-memory table
	table := poker.NewTable(cfg)

	// 4) Load active participants and seat deterministically by seat number
	parts, err := s.db.ActiveParticipants(ctx, tableID)
	if err != nil {
		return nil, fmt.Errorf("failed to load participants: %w", err)
	}
	sort.Slice(parts, func(i, j int) bool { return parts[i].Seat < parts[j].Seat })

	for _, p := range parts {
		// Use durable wallet balance (not table chips)
		dcrBalance, err := s.db.GetPlayerBalance(ctx, p.PlayerID)
		if err != nil {
			s.log.Errorf("GetPlayerBalance(%s): %v", p.PlayerID, err)
			dcrBalance = 0
		}

		// Your User factory signature: (id, name, dcrBalance, seat)
		user := poker.NewUser(p.PlayerID, p.PlayerID, dcrBalance, p.Seat)

		// Seat the user in the table model
		if _, err := table.AddNewUser(user.ID, user.ID, user.DCRAccountBalance, user.TableSeat); err != nil {
			s.log.Errorf("AddNewUser(%s): %v", user.ID, err)
			continue
		}

		// Apply lobby flag via table API (fires state update in FSM)
		if err := table.SetPlayerReady(user.ID, p.Ready); err != nil {
			s.log.Errorf("SetPlayerReady(%s): %v", user.ID, err)
		}
	}

	// 5) Try fast-restore snapshot; otherwise, start fresh when appropriate.
	if snap, err := s.db.GetSnapshot(ctx, tableID); err == nil && snap != nil && len(snap.Payload) > 0 {
		// Unmarshal only the game sub-structure; ignore other fields like Config.Log
		var persisted struct {
			Game *poker.GameStateSnapshot `json:"Game"`
		}
		if err := json.Unmarshal(snap.Payload, &persisted); err != nil {
			s.log.Errorf("unmarshal snapshot for table %s: %v", tableID, err)
		} else if persisted.Game != nil {
			if err := s.applyGameSnapshot(table, persisted.Game); err != nil {
				s.log.Errorf("apply snapshot for table %s: %v", tableID, err)
			} else {
				s.log.Infof("Restored game from snapshot for table %s", tableID)
			}
		}
	} else {
		if table.CheckAllPlayersReady() {
			// Start game asynchronously to avoid blocking table restoration
			go func() {
				if err := table.StartGame(); err != nil {
					s.log.Errorf("auto-start game for table %s: %v", tableID, err)
				}
			}()
		}
	}

	// Register the runtime table
	s.tables.Store(tableID, table)
	return table, nil
}

// restoreGameState currently just starts a fresh Game using the table runtime.
// If you later persist snapshots, you can hydrate here.
func (s *Server) restoreGameState(table *poker.Table, tcfg *db.Table, _ []db.Participant) (*poker.Game, error) {
	users := table.GetUsers()
	sort.Slice(users, func(i, j int) bool { return users[i].TableSeat < users[j].TableSeat })

	game, err := table.RestoreGame(tcfg.ID)
	if err != nil {
		return nil, fmt.Errorf("restore/start game: %w", err)
	}
	s.log.Infof("Started new game for table %s with %d players", tcfg.ID, len(users))
	return game, nil
}

// loadAllTables loads all persisted tables on startup.
func (s *Server) loadAllTables() error {
	ctx := context.Background()
	s.log.Infof("Loading persisted tables from database...")

	ids, err := s.db.ListTableIDs(ctx)
	if err != nil {
		return fmt.Errorf("list table IDs: %w", err)
	}
	if len(ids) == 0 {
		s.log.Infof("No persisted tables found")
		return nil
	}

	loaded := 0
	for _, id := range ids {
		t, err := s.loadTableFromDatabase(id)
		if err != nil {
			s.log.Errorf("load table %s: %v", id, err)
			continue
		}
		s.tables.Store(id, t)
		loaded++
		s.log.Infof("Loaded table %s", id)
	}

	s.log.Infof("Successfully loaded %d of %d persisted tables", loaded, len(ids))
	return nil
}

// applyPokerSnapshot hydrates the table/game from a saved poker.TableStateSnapshot.
func (s *Server) applyGameSnapshot(table *poker.Table, gs *poker.GameStateSnapshot) error {
	if gs == nil {
		return fmt.Errorf("invalid snapshot")
	}
	// Ensure a game instance is attached to the table and players are set
	g, err := table.RestoreGame(table.GetConfig().ID)
	if err != nil {
		return fmt.Errorf("attach game: %w", err)
	}
	users := table.GetUsers()
	g.SetPlayers(users)

	// Restore community cards if any
	if len(gs.CommunityCards) > 0 {
		g.SetCommunityCards(gs.CommunityCards)
	}

	// Derive phase from community cards count
	phase := pokerrpc.GamePhase_PRE_FLOP
	switch n := len(gs.CommunityCards); n {
	case 0:
		phase = pokerrpc.GamePhase_PRE_FLOP
	case 3:
		phase = pokerrpc.GamePhase_FLOP
	case 4:
		phase = pokerrpc.GamePhase_TURN
	case 5:
		phase = pokerrpc.GamePhase_RIVER
	}

	// Set coarse game state: dealer, counters, current bet, phase
	g.SetGameState(gs.Dealer, gs.Round, gs.BetRound, gs.CurrentBet, gs.Pot, phase)

	// If snapshot had a known current player ID, prefer it to avoid
	// re-deriving actor mid-street, which can be ambiguous without
	// per-player bet deltas.
	if gs.CurrentPlayer != "" {
		g.SetCurrentPlayerByID(gs.CurrentPlayer)
	}

	return nil
}
