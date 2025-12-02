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
)

// Database is the minimal surface the server needs from the storage layer.
type Database interface {
	// ---- Players / wallet ----
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

	// ---- Auth ----
	UpsertAuthUser(ctx context.Context, nickname, userID string) error
	GetAuthUserByNickname(ctx context.Context, nickname string) (*db.AuthUser, error)
	GetAuthUserByUserID(ctx context.Context, userID string) (*db.AuthUser, error)
	UpdateAuthUserLastLogin(ctx context.Context, userID string) error
	UpdateAuthUserPayoutAddress(ctx context.Context, userID, payoutAddress string) error
	ListAllAuthUsers(ctx context.Context) ([]db.AuthUser, error)

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
	inner, err := db.NewDB(dbPath)
	if err != nil {
		return nil, err
	}
	// Wrap with metrics instrumentation.
	return &metricsDB{inner: inner}, nil
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
		user := poker.NewUser(p.PlayerID, table, &poker.AddUserOptions{
			DisplayName: s.displayNameFor(p.PlayerID),
		})

		// Set the seat from the database record
		user.SetTableSeat(p.Seat)

		// Add the user to the table model
		if err := table.AddUser(user); err != nil {
			s.log.Errorf("AddUser(%s): %v", user.ID, err)
		}
	}

	// Attempt to hydrate an in-progress game from the latest fast-restore snapshot
	if snap, err := s.db.GetSnapshot(ctx, tableID); err == nil && snap != nil && len(snap.Payload) > 0 {
		var persisted struct {
			Game *poker.GameStateSnapshot `json:"Game"`
		}
		if uerr := json.Unmarshal(snap.Payload, &persisted); uerr != nil {
			s.log.Errorf("failed to unmarshal snapshot for table %s: %v", tableID, uerr)
		} else if persisted.Game != nil {
			if err := s.applyGameSnapshot(table, persisted.Game); err != nil {
				s.log.Errorf("applyGameSnapshot(%s) failed: %v", tableID, err)
			}
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

	// Skip restore if phase is WAITING (no active game to restore)
	if gs.Phase == pokerrpc.GamePhase_WAITING {
		s.log.Debugf("applyGameSnapshot: skipping restore for table=%s (phase=WAITING, no active game)",
			table.GetConfig().ID)
		return nil
	}

	s.log.Debugf("applyGameSnapshot: table=%s phase=%v br=%d curBet=%d comm=%d cur=%s",
		table.GetConfig().ID, gs.Phase, gs.BetRound, gs.CurrentBet, len(gs.CommunityCards), gs.CurrentPlayer)
	// 1) Ensure a game instance is attached to the table and players are set
	// RestoreGame already calls SetPlayers internally, so we don't need to call it again
	g, err := table.RestoreGame(table.GetConfig().ID)
	if err != nil {
		return fmt.Errorf("attach game: %w", err)
	}

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
		if st != poker.FOLDED_STATE && st != poker.ALL_IN_STATE {
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
