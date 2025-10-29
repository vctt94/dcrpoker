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
	"sync"
	"time"

	"github.com/vctt94/pokerbisonrelay/pkg/poker"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	"github.com/vctt94/pokerbisonrelay/pkg/server/internal/db"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

// loadTableFromDatabase restores a table config and currently seated players.
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

	// 3) Create in-memory table and wire event forwarding
	table := poker.NewTable(cfg)
	// Create a channel for table events and start a goroutine to process them
	tableEventChan := make(chan poker.TableEvent, 100)
	table.SetEventChannel(tableEventChan)
	go s.processTableEvents(tableEventChan)

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

	s.tables.Store(tableID, table)
	return table, nil
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
	s.log.Debugf("applyGameSnapshot: table=%s phase=%v br=%d curBet=%d comm=%d cur=%s",
		table.GetConfig().ID, gs.Phase, gs.BetRound, gs.CurrentBet, len(gs.CommunityCards), gs.CurrentPlayer)
	// 1) Ensure a game instance is attached to the table and players are set
	g, err := table.RestoreGame(table.GetConfig().ID)
	if err != nil {
		return fmt.Errorf("attach game: %w", err)
	}
	users := table.GetUsers()
	g.SetPlayers(users)

	// Wire game→table event channel so restored games can emit updates
	table.WireGameEvents(g)

	// 2) Restore hole cards and community cards (if any)
	if gs.Players != nil {
		_ = g.RestoreHoleCards(gs.Players)
	}
	if len(gs.CommunityCards) > 0 {
		g.SetCommunityCards(gs.CommunityCards)
	}

	// 2b) Restore deck state so future deals don't reintroduce already-dealt cards
	if ds := gs.DeckState; ds != nil && len(ds.RemainingCards) > 0 {
		if err := g.RestoreDeckState(ds); err != nil {
			s.log.Warnf("restore deck state failed: %v", err)
		}
	}

	// 3) Apply snapshot-declared phase exactly; do not derive from board size.
	phase := gs.Phase
	g.SetGameState(gs.Dealer, gs.Round, gs.CurrentBet, gs.Pot, phase)

	// 4) Restore per-player derived fields (balance, currentBet, positions, turn)
	// Use the available Player.Unmarshal helper to set role flags consistently.
	// Restore/override per-player derived fields from snapshot.
	if gs.Players != nil {
		// Build a quick index id -> player
		byID := map[string]*poker.Player{}
		for _, p := range g.GetPlayers() {
			if p != nil {
				byID[p.ID()] = p
			}
		}
		for _, ps := range gs.Players {
			p := byID[ps.ID]
			if p == nil {
				continue
			}
			p.SetStartingBalance(ps.StartingBalance)
			p.SetBalance(ps.Balance)
			p.SetCurrentBet(ps.CurrentBet)

			// Set role/turn flags via Unmarshal mirror
			mirror := &pokerrpc.Player{
				Id:              ps.ID,
				Balance:         ps.Balance,
				CurrentBet:      ps.CurrentBet,
				IsDealer:        ps.IsDealer,
				IsSmallBlind:    ps.IsSmallBlind,
				IsBigBlind:      ps.IsBigBlind,
				IsTurn:          ps.IsTurn,
				HandDescription: ps.HandDescription,
			}
			p.Unmarshal(mirror)
		}
	}

	// Ensure all seated players are in-hand (ACTIVE/ALL_IN) after restore
	for _, p := range g.GetPlayers() {
		if p == nil {
			continue
		}
		st := p.GetCurrentStateString()
		if st != "FOLDED" && st != "ALL_IN" {
			_ = p.HandleStartHand()
		}
	}

	// 5) Rebuild pots and derive current player/phase invariants from players
	g.RestoreGameState(table.GetConfig().ID)

	// 6) If snapshot had a known current player ID and we're not at SHOWDOWN,
	// prefer it; at SHOWDOWN there is no current player to act.
	if phase != pokerrpc.GamePhase_SHOWDOWN && gs.CurrentPlayer != "" {
		g.SetCurrentPlayerByID(gs.CurrentPlayer)
	}

	// 7) Restore pot contributions by deriving each player's total chips put
	// into the hand so far. PlayerSnapshot.StartingBalance reflects the
	// balance AFTER blinds were posted (it is captured on evStartHand), so
	// we must add the forced blind amounts back in to reconstruct full-hand
	// contributions. Without this, an SB/BB who did not invest further on
	// later streets would appear to have contributed 0, causing incorrect
	// winner eligibility and wrong payouts after a restore.
	if gs.Players != nil {
		contrib := make(map[string]int64, len(gs.Players))
		var sum int64
		// Table blind configuration
		tblCfg := table.GetConfig()
		for _, ps := range gs.Players {
			// Derive delta invested since hand start (post-blinds snapshot)
			if ps.StartingBalance > 0 && ps.Balance >= 0 {
				c := ps.StartingBalance - ps.Balance
				if c < 0 {
					c = 0
				}
				// Add forced blinds for this hand to reconstruct the full
				// contribution. PlayerSnapshot flags encode SB/BB roles.
				if ps.IsSmallBlind {
					c += tblCfg.SmallBlind
				}
				if ps.IsBigBlind {
					c += tblCfg.BigBlind
				}
				contrib[ps.ID] = c
				sum += c
			}
		}
		if sum > 0 && phase != pokerrpc.GamePhase_SHOWDOWN {
			g.RestorePotsFromContributions(contrib)
		}
	}

	// Start the Game FSM in a passive restored state so it processes events
	// (advance, timebank, etc.) without mutating restored fields on entry.
	g.StartFromRestoredSnapshot(context.Background())

	return nil
}

// buildGameStateForPlayer removed; GameUpdate is now built from stable snapshots

// buildGameState creates a GameUpdate for the requesting player
func (s *Server) buildGameState(tableID, requestingPlayerID string, snap *db.Snapshot) (*pokerrpc.GameUpdate, error) {
	// Fetch table pointer without coarse-grained server locking.
	table, ok := s.getTable(tableID)
	if !ok {
		return nil, status.Error(codes.NotFound, "table not found")
	}

	game := table.GetGame()
	// If the table exists but no game is attached (e.g., server restarted),
	// try to restore the game from the latest snapshot before registering the
	// stream so the initial snapshot reflects the live game.
	if game == nil {
		users := table.GetUsers()
		// Attempt fast-restore from persisted snapshot, but only when seats >= 2
		if len(users) >= 2 {
			var persisted struct {
				Game *poker.GameStateSnapshot `json:"Game"`
			}
			if err := json.Unmarshal(snap.Payload, &persisted); err == nil && persisted.Game != nil {
				gs := persisted.Game
				err = s.applyGameSnapshot(table, gs)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	// Build GameUpdate from a fresh, stable table snapshot
	ts, err := s.collectTableSnapshot(tableID)
	if err != nil {
		return nil, err
	}
	gsh := NewGameStateHandler(s)
	return gsh.buildGameUpdateFromSnapshot(ts, requestingPlayerID), nil
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
