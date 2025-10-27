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
	snap, err := s.db.GetSnapshot(ctx, tableID)
	if err != nil || snap == nil || len(snap.Payload) == 0 {
		// No snapshot available, start fresh if players are ready
		if table.CheckAllPlayersReady() {
			// Start game asynchronously to avoid blocking table restoration
			go func() {
				if err := table.StartGame(); err != nil {
					s.log.Errorf("auto-start game for table %s: %v", tableID, err)
				}
			}()
		}
		return table, nil
	}

	// Unmarshal only the game sub-structure; ignore other fields like Config.Log
	var persisted struct {
		Game *poker.GameStateSnapshot `json:"Game"`
	}
	if err := json.Unmarshal(snap.Payload, &persisted); err != nil {
		s.log.Errorf("unmarshal snapshot for table %s: %v", tableID, err)
		return table, nil
	}

	if persisted.Game == nil {
		// No game data in snapshot, start fresh if players are ready
		if table.CheckAllPlayersReady() {
			go func() {
				if err := table.StartGame(); err != nil {
					s.log.Errorf("auto-start game for table %s: %v", tableID, err)
				}
			}()
		}
		return table, nil
	}

	if err := s.applyGameSnapshot(table, persisted.Game); err != nil {
		s.log.Errorf("apply snapshot for table %s: %v", tableID, err)
		return table, nil
	}

	s.log.Infof("Restored game from snapshot for table %s", tableID)

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

// buildPlayerForUpdate creates a Player proto message with appropriate card visibility
func (s *Server) buildPlayerForUpdate(p *poker.Player, requestingPlayerID string, game *poker.Game) *pokerrpc.Player {
	stateStr := p.GetCurrentStateString()
	grpcPlayer := p.Marshal() // snapshot with turn/dealer/blinds flags
	// XXX repeating here, can use grpcplayer directly
	player := &pokerrpc.Player{
		Id:      p.ID(),
		Balance: p.Balance(),
		IsReady: p.IsReady(),
		Folded:  stateStr == "FOLDED",
		// Surface all-in so UIs can render an explicit badge without inference.
		IsAllIn:      stateStr == "ALL_IN",
		CurrentBet:   p.CurrentBet(),
		PlayerState:  p.GetTablePresenceState(),
		IsDealer:     grpcPlayer.IsDealer,
		IsSmallBlind: grpcPlayer.IsSmallBlind,
		IsBigBlind:   grpcPlayer.IsBigBlind,
		IsTurn:       grpcPlayer.IsTurn,
	}

	// Heads-up sanity: dealer must also be SB.
	if game != nil && len(game.GetPlayers()) == 2 && grpcPlayer.IsDealer && !grpcPlayer.IsSmallBlind {
		s.log.Warnf("INCONSISTENT STATE: Player %s is dealer but not SB in heads-up! phase=%v", p.ID(), game.GetPhase())
	}

	// No game -> nothing else to surface.
	if game == nil {
		return player
	}

	hand := game.GetCurrentHand()
	if hand == nil {
		// Still return base player info; cards come only from an active hand.
		return player
	}

	// Decide visibility once, then fill if any cards are visible.
	var cards []poker.Card
	isShowdown := game.GetPhase() == pokerrpc.GamePhase_SHOWDOWN
	isSelf := p.ID() == requestingPlayerID

	switch {
	case isSelf:
		// Always show own cards as soon as they exist.
		cards = hand.GetPlayerCards(p.ID(), requestingPlayerID)
		if len(cards) > 0 {
			s.log.Debugf("DEBUG: Showing %d cards for player %s (own cards, phase=%v, state=%s)",
				len(cards), p.ID(), game.GetPhase(), stateStr)
		}
	case isShowdown:
		// Show others' cards only at showdown (visibility enforced by GetPlayerCards).
		cards = hand.GetPlayerCards(p.ID(), requestingPlayerID)
	}

	if n := len(cards); n > 0 {
		player.Hand = make([]*pokerrpc.Card, n)
		for i, c := range cards {
			player.Hand[i] = &pokerrpc.Card{Suit: c.GetSuit(), Value: c.GetValue()}
		}
	}

	// Hand description is surfaced only at showdown.
	if isShowdown && p.HandDescription() != "" {
		player.HandDescription = p.HandDescription()
	}

	return player
}

// buildPlayers creates a slice of Player proto messages with appropriate card visibility
func (s *Server) buildPlayers(tablePlayers []*poker.Player, game *poker.Game, requestingPlayerID string) []*pokerrpc.Player {
	players := make([]*pokerrpc.Player, 0, len(tablePlayers))
	for _, p := range tablePlayers {
		player := s.buildPlayerForUpdate(p, requestingPlayerID, game)
		players = append(players, player)
	}
	return players
}

// buildGameStateForPlayer creates a GameUpdate with all the necessary data for a specific player
func (s *Server) buildGameStateForPlayer(table *poker.Table, game *poker.Game, requestingPlayerID string) *pokerrpc.GameUpdate {
	// Build players list from users and game players
	var players []*pokerrpc.Player
	if game != nil {
		players = s.buildPlayers(game.GetPlayers(), game, requestingPlayerID)
	} else {
		// If no game, build from table users
		users := table.GetUsers()
		players = make([]*pokerrpc.Player, 0, len(users))
		for _, user := range users {
			players = append(players, &pokerrpc.Player{
				Id:      user.ID,
				Balance: 0, // No poker chips when no game - Balance field should be poker chips, not DCR
				IsReady: user.IsReady,

				Hand:        make([]*pokerrpc.Card, 0), // Empty hand when no game
				PlayerState: pokerrpc.PlayerState_PLAYER_STATE_AT_TABLE,
			})
		}
	}

	// Build community cards slice
	communityCards := make([]*pokerrpc.Card, 0)
	var pot int64 = 0
	if game != nil {
		pot = game.GetPot()
		for _, c := range game.GetCommunityCards() {
			communityCards = append(communityCards, &pokerrpc.Card{
				Suit:  c.GetSuit(),
				Value: c.GetValue(),
			})
		}
	}

	var currentPlayerID string
	if table.IsGameStarted() && game != nil {
		// Only expose current player when action is valid (not during setup or showdown)
		phase := game.GetPhase()
		if phase != pokerrpc.GamePhase_NEW_HAND_DEALING && phase != pokerrpc.GamePhase_SHOWDOWN {
			currentPlayerID = table.GetCurrentPlayerID()
		}
	}

	// Note: Do not override per-player IsTurn here; the Player FSM is the
	// single authority for that flag. UIs should rely on CurrentPlayer for
	// highlighting to avoid transient races between EndTurn/StartTurn events.

	// Authoritative timebank fields
	var tbSec int32
	var deadlineMs int64
	cfg := table.GetConfig()
	if cfg.TimeBank > 0 {
		tbSec = int32(cfg.TimeBank.Seconds())
		// Compute deadline for the current player if applicable, using a snapshot
		if currentPlayerID != "" {
			snap := game.GetStateSnapshot()
			for _, ps := range snap.Players {
				if ps.ID == currentPlayerID {
					dl := ps.LastAction.Add(cfg.TimeBank)
					deadlineMs = dl.UnixMilli()
					break
				}
			}
		}
	}

	return &pokerrpc.GameUpdate{
		TableId:            table.GetConfig().ID,
		Phase:              table.GetGamePhase(),
		PhaseName:          table.GetGamePhase().String(),
		Players:            players,
		CommunityCards:     communityCards,
		Pot:                pot,
		CurrentBet:         table.GetCurrentBet(),
		CurrentPlayer:      currentPlayerID,
		GameStarted:        table.IsGameStarted(),
		PlayersRequired:    int32(table.GetMinPlayers()),
		PlayersJoined:      int32(len(table.GetUsers())),
		TimeBankSeconds:    tbSec,
		TurnDeadlineUnixMs: deadlineMs,
	}
}

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
