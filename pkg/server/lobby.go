package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/vctt94/pokerbisonrelay/pkg/poker"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// newTableID generates a random 16-byte hex table ID.
// This serves as both the table identifier and the matchID for Schnorr settlement.
func newTableID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		// Fallback to timestamp if crypto/rand fails (should never happen)
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf[:])
}

func (s *Server) CreateTable(ctx context.Context, req *pokerrpc.CreateTableRequest) (*pokerrpc.CreateTableResponse, error) {
	// Prevent joining a different table while still seated in an active game.
	if activeTableID, found := s.findActiveTableForPlayer(req.PlayerId); found {
		return nil, status.Errorf(codes.FailedPrecondition, "player already in active game at table %s", activeTableID)
	}
	s.log.Debugf("Creating table with buy-in %d", req.BuyIn)

	// Config
	tblLog := s.logBackend.Logger("TABLE")
	gameLog := s.logBackend.Logger("GAME")

	cfg := poker.TableConfig{
		ID:                    newTableID(),
		Name:                  strings.TrimSpace(req.GetName()),
		Log:                   tblLog,
		GameLog:               gameLog,
		Source:                managedTableSourceUser,
		BuyIn:                 req.BuyIn,
		MinPlayers:            int(req.MinPlayers),
		MaxPlayers:            int(req.MaxPlayers),
		SmallBlind:            req.SmallBlind,
		BigBlind:              req.BigBlind,
		StartingChips:         req.StartingChips,
		TimeBank:              time.Duration(req.TimeBankSeconds) * time.Second,
		AutoStartDelay:        time.Duration(req.AutoStartMs) * time.Millisecond,
		AutoAdvanceDelay:      time.Duration(req.AutoAdvanceMs) * time.Millisecond,
		BlindIncreaseInterval: time.Duration(req.BlindIncreaseIntervalSec) * time.Second,
	}
	if _, err := s.createTable(ctx, cfg, req.PlayerId); err != nil {
		s.log.Errorf("CreateTable failed: %v", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pokerrpc.CreateTableResponse{TableId: cfg.ID}, nil
}

func (s *Server) JoinTable(ctx context.Context, req *pokerrpc.JoinTableRequest) (*pokerrpc.JoinTableResponse, error) {
	table, ok := s.getTable(req.TableId)
	if !ok {
		return &pokerrpc.JoinTableResponse{Success: false, Message: "Table not found"}, nil
	}

	// Prevent joining a different table while still seated in an active game.
	if activeTableID, found := s.findActiveTableForPlayer(req.PlayerId); found && activeTableID != req.TableId {
		return &pokerrpc.JoinTableResponse{
			Success: false,
			Message: fmt.Sprintf("player already in active game at table %s", activeTableID),
		}, nil
	}

	s.log.Debugf("Joining table %s", req.TableId)
	config := table.GetConfig()

	// Reconnect: already seated in-memory.
	if existingUser := table.GetUser(req.PlayerId); existingUser != nil {
		// Reconnect: player is already seated in-memory. Publish a fresh
		// PLAYER_JOINED event.
		if err := s.publishPlayerJoined(req.TableId, req.PlayerId); err != nil {
			s.log.Errorf("Failed to publish reconnect PLAYER_JOINED event: %v", err)
			return nil, status.Error(codes.Internal, "failed to publish reconnect event")
		}
		return &pokerrpc.JoinTableResponse{
			Success: true,
			Message: "Reconnected to table.",
		}, nil
	}

	// Block new seats once a game has started to prevent eliminated players
	// (or new entrants) from rejoining mid-match.
	if table.IsGameStarted() {
		return &pokerrpc.JoinTableResponse{
			Success: false,
			Message: "Game already started; joining not allowed",
		}, nil
	}

	// Pick next free seat.
	occupied := make(map[int]bool)
	for _, u := range table.GetUsers() {
		occupied[u.TableSeat] = true
	}
	seat := -1
	for i := 0; i < config.MaxPlayers; i++ {
		if !occupied[i] {
			seat = i
			break
		}
	}
	if seat == -1 {
		return &pokerrpc.JoinTableResponse{Success: false, Message: "Table is full"}, nil
	}

	// Persist seat first (DB enforces UNIQUE(table_id, seat)).
	if err := s.db.SeatPlayer(ctx, req.TableId, req.PlayerId, seat); err != nil {
		s.log.Errorf("SeatPlayer failed (table=%s player=%s seat=%d): %v", req.TableId, req.PlayerId, seat, err)
		return &pokerrpc.JoinTableResponse{Success: false, Message: "Seat not available, try again"}, nil
	}
	rollbackSeat := func() {
		if err := s.db.UnseatPlayer(ctx, req.TableId, req.PlayerId); err != nil {
			s.log.Errorf("UnseatPlayer rollback failed (table=%s player=%s): %v", req.TableId, req.PlayerId, err)
		}
	}

	// Seat player in-memory
	if _, err := table.AddNewUser(req.PlayerId, &poker.AddUserOptions{
		DisplayName: s.displayNameFor(req.PlayerId),
		Seat:        seat,
	}); err != nil {
		rollbackSeat()
		return &pokerrpc.JoinTableResponse{Success: false, Message: err.Error()}, nil
	}

	// Publish join notification.
	if err := s.publishPlayerJoined(req.TableId, req.PlayerId); err != nil {
		s.log.Errorf("Failed to publish PLAYER_JOINED event: %v", err)
		return nil, err
	}
	s.removeTableWatcher(req.TableId, req.PlayerId)

	return &pokerrpc.JoinTableResponse{
		Success: true,
		Message: "Successfully joined table",
	}, nil
}

func (s *Server) WatchTable(ctx context.Context, req *pokerrpc.WatchTableRequest) (*pokerrpc.WatchTableResponse, error) {
	table, ok := s.getTable(req.TableId)
	if !ok {
		return &pokerrpc.WatchTableResponse{Success: false, Message: "Table not found"}, nil
	}

	if table.GetUser(req.PlayerId) != nil {
		return &pokerrpc.WatchTableResponse{
			Success: true,
			Message: "Already seated at table",
		}, nil
	}

	if !s.addTableWatcher(req.TableId, req.PlayerId) {
		return &pokerrpc.WatchTableResponse{
			Success: true,
			Message: "Already watching table",
		}, nil
	}

	return &pokerrpc.WatchTableResponse{
		Success: true,
		Message: "Watching table",
	}, nil
}

func (s *Server) createTable(ctx context.Context, cfg poker.TableConfig, initialPlayerID string) (*poker.Table, error) {
	normalizeTableConfig(&cfg)
	if strings.TrimSpace(cfg.ID) == "" {
		cfg.ID = newTableID()
	}
	if strings.TrimSpace(cfg.Name) == "" {
		cfg.Name = fmt.Sprintf("Table %s", cfg.ID[:8])
	}

	table := poker.NewTable(cfg)
	tableEventChan := make(chan poker.TableEvent, 100)
	table.SetEventChannel(tableEventChan)
	go s.processTableEvents(tableEventChan)

	if err := s.db.UpsertTable(ctx, &cfg); err != nil {
		table.Close()
		return nil, fmt.Errorf("persist table %s: %w", cfg.ID, err)
	}

	s.tables.Store(cfg.ID, table)

	if initialPlayerID != "" {
		if _, err := table.AddNewUser(initialPlayerID, &poker.AddUserOptions{
			DisplayName: s.displayNameFor(initialPlayerID),
		}); err != nil {
			s.cleanupCreatedTable(ctx, cfg.ID, table)
			return nil, err
		}
		if err := s.db.SeatPlayer(ctx, cfg.ID, initialPlayerID, 0); err != nil {
			s.cleanupCreatedTable(ctx, cfg.ID, table)
			return nil, fmt.Errorf("seat initial player: %w", err)
		}
		if err := s.publishPlayerJoined(cfg.ID, initialPlayerID); err != nil {
			s.cleanupCreatedTable(ctx, cfg.ID, table)
			return nil, fmt.Errorf("publish initial player join: %w", err)
		}
	}

	evt, err := s.buildGameEvent(
		pokerrpc.NotificationType_TABLE_CREATED,
		cfg.ID,
		nil,
	)
	if err != nil {
		s.cleanupCreatedTable(ctx, cfg.ID, table)
		return nil, fmt.Errorf("build table created event: %w", err)
	}
	s.eventProcessor.PublishEvent(evt)

	return table, nil
}

func (s *Server) cleanupCreatedTable(ctx context.Context, tableID string, table *poker.Table) {
	s.removeTableFromRegistry(tableID)
	if table != nil {
		table.Close()
	}
	if s.db != nil {
		_ = s.db.DeleteTable(ctx, tableID)
	}
}

// publishPlayerJoined builds and publishes a PLAYER_JOINED event for the given
// table and player, using the standard event pipeline so that snapshots and
// lobby/game state remain consistent everywhere.
func (s *Server) publishPlayerJoined(tableID, playerID string) error {
	evt, err := s.buildGameEvent(
		pokerrpc.NotificationType_PLAYER_JOINED,
		tableID,
		PlayerJoinedPayload{PlayerID: playerID},
	)
	if err != nil {
		return err
	}
	s.eventProcessor.PublishEvent(evt)
	return nil
}

// clearEscrowBindingForSeat removes any referee escrow binding for the given
// table/match and seat. It also clears the bound table/seat metadata from the
// escrow session itself, while keeping the escrow session available for
// refunds or reuse.
func (s *Server) clearEscrowBindingForSeat(tableID string, seat uint32) {
	if s.referee == nil {
		return
	}

	matchID := tableID

	// Capture the currently bound escrow (if any) for metadata cleanup.
	var escrowID string
	s.referee.mu.RLock()
	if seats := s.referee.matchEscrows[matchID]; seats != nil {
		escrowID = seats[seat]
	}
	s.referee.mu.RUnlock()

	// Drive escrow reset through the table/user FSM.
	if table, ok := s.getTable(tableID); ok && table != nil {
		if u := table.GetUserAtSeat(int(seat)); u != nil {
			_ = u.SendEscrowReset()
		}
		// Rebuild matchEscrows from the authoritative table snapshot.
		s.rebuildMatchEscrowsFromTable(matchID, table)
	}

	// Clear escrow session metadata.
	if escrowID != "" {
		s.referee.mu.RLock()
		es := s.referee.escrows[escrowID]
		s.referee.mu.RUnlock()
		if es != nil {
			es.mu.Lock()
			if es.TableID == tableID && es.SeatIndex == seat {
				es.TableID = ""
				es.SessionID = ""
				es.SeatIndex = 0
			}
			es.mu.Unlock()
		}
	}
}

func (s *Server) resetTablePreparationState(tableID string, table *poker.Table) {
	if table == nil {
		return
	}

	for _, u := range table.GetUsers() {
		if err := table.ResetPlayerPresign(u.ID); err != nil {
			s.log.Warnf("failed to reset presign for player %s at table %s: %v", u.ID, tableID, err)
		}
	}
	s.clearMatchPreparationState(tableID)
}

func (s *Server) reserveSeatOnLeave(tableID string, user *poker.User, reason string) (*pokerrpc.LeaveTableResponse, error) {
	user.SendDisconnect()

	if snap, err := s.collectTableSnapshot(tableID); err == nil {
		s.publishTableSnapshotEvent(tableID, snap)
	}
	s.saveTableStateAsync(tableID, reason)

	return &pokerrpc.LeaveTableResponse{
		Success: true,
		Message: "You have been disconnected but your seat is reserved while the match remains active.",
	}, nil
}

func (s *Server) LeaveTable(ctx context.Context, req *pokerrpc.LeaveTableRequest) (*pokerrpc.LeaveTableResponse, error) {
	table, ok := s.getTable(req.TableId)
	if !ok {
		return &pokerrpc.LeaveTableResponse{Success: false, Message: "Table not found"}, nil
	}

	// Get user's current state
	user := table.GetUser(req.PlayerId)
	if user == nil {
		return &pokerrpc.LeaveTableResponse{Success: false, Message: "Player not at table"}, nil
	}

	if table.IsGameStarted() {
		return s.reserveSeatOnLeave(req.TableId, user, "player disconnected during active game")
	}

	if s.hasPendingSettlement(req.TableId) {
		return s.reserveSeatOnLeave(req.TableId, user, "player disconnected during pending settlement")
	}

	// No active game: fully leave table and clear any escrow bindings for this seat.
	// This allows the player to re-bind a different escrow if they rejoin later.
	userSnap := user.GetSnapshot()
	if userSnap.TableSeat >= 0 {
		s.clearEscrowBindingForSeat(req.TableId, uint32(userSnap.TableSeat))
	}

	// Otherwise, remove completely from the table runtime first
	if err := table.RemoveUser(req.PlayerId); err != nil {
		return &pokerrpc.LeaveTableResponse{Success: false, Message: err.Error()}, nil
	}

	// Unseat in the DB (normalized schema)
	if err := s.db.UnseatPlayer(ctx, req.TableId, req.PlayerId); err != nil {
		s.log.Errorf("Failed to unseat player in DB: %v", err)
		// We continue; in-memory removal already happened.
	}

	// Publish PLAYER_LEFT so lobby/waiting room lists refresh immediately.
	evt, err := s.buildGameEvent(
		pokerrpc.NotificationType_PLAYER_LEFT,
		req.TableId,
		PlayerLeftPayload{PlayerID: req.PlayerId},
	)
	if err != nil {
		return &pokerrpc.LeaveTableResponse{Success: false, Message: err.Error()}, nil
	}
	s.eventProcessor.PublishEvent(evt)

	s.resetTablePreparationState(req.TableId, table)

	if s.maybeRemoveTable(req.TableId) {
		return &pokerrpc.LeaveTableResponse{
			Success: true,
			Message: "Successfully left table. Table closed.",
		}, nil
	}

	// Save updated snapshot (optional fast-restore)
	s.saveTableStateAsync(req.TableId, "player left")
	if snap, err := s.collectTableSnapshot(req.TableId); err == nil {
		s.publishTableSnapshotEvent(req.TableId, snap)
	}

	return &pokerrpc.LeaveTableResponse{
		Success: true,
		Message: "Successfully left table",
	}, nil
}

func (s *Server) UnwatchTable(ctx context.Context, req *pokerrpc.UnwatchTableRequest) (*pokerrpc.UnwatchTableResponse, error) {
	if req.TableId == "" {
		return &pokerrpc.UnwatchTableResponse{Success: false, Message: "Table ID is required"}, nil
	}

	if s.removeTableWatcher(req.TableId, req.PlayerId) {
		return &pokerrpc.UnwatchTableResponse{
			Success: true,
			Message: "Stopped watching table",
		}, nil
	}

	return &pokerrpc.UnwatchTableResponse{
		Success: true,
		Message: "Was not watching table",
	}, nil
}

func (s *Server) GetTables(ctx context.Context, req *pokerrpc.GetTablesRequest) (*pokerrpc.GetTablesResponse, error) {
	// Snapshot current tables from concurrent registry
	tableRefs := s.getAllTables()

	// Build response using collected snapshots so initial lobby fetches carry
	// the same roster data as notification-driven table updates.
	tables := make([]*pokerrpc.Table, 0, len(tableRefs))
	for _, table := range tableRefs {
		config := table.GetConfig()
		snap, err := s.collectTableSnapshot(config.ID)
		if err == nil {
			tables = append(tables, tableSnapshotToProtoTable(snap))
			continue
		}

		s.log.Warnf("GetTables: failed to collect snapshot for %s: %v", config.ID, err)

		users := table.GetUsers()

		protoTable := &pokerrpc.Table{
			Id:                       config.ID,
			Name:                     config.Name,
			SmallBlind:               config.SmallBlind,
			BigBlind:                 config.BigBlind,
			MaxPlayers:               int32(table.GetMaxPlayers()),
			MinPlayers:               int32(table.GetMinPlayers()),
			CurrentPlayers:           int32(len(users)),
			BuyIn:                    config.BuyIn,
			GameStarted:              table.IsGameStarted(),
			AllPlayersReady:          table.AreAllPlayersReady(),
			Phase:                    pokerrpc.GamePhase_WAITING,
			BlindIncreaseIntervalSec: int32(config.BlindIncreaseInterval.Seconds()),
		}
		for _, user := range users {
			if user == nil {
				continue
			}
			userSnap := user.GetSnapshot()
			protoTable.Players = append(protoTable.Players, &pokerrpc.Player{
				Id:              userSnap.ID,
				Name:            userSnap.Name,
				IsReady:         userSnap.IsReady,
				IsDisconnected:  userSnap.IsDisconnected,
				EscrowId:        userSnap.EscrowID,
				EscrowReady:     userSnap.EscrowReady,
				PresignComplete: userSnap.PresignComplete,
				TableSeat:       int32(userSnap.TableSeat),
			})
		}
		tables = append(tables, protoTable)
	}

	return &pokerrpc.GetTablesResponse{Tables: tables}, nil
}

func (s *Server) GetPlayerCurrentTable(ctx context.Context, req *pokerrpc.GetPlayerCurrentTableRequest) (*pokerrpc.GetPlayerCurrentTableResponse, error) {
	// Get table references with server lock
	tableRefs := s.getAllTables()

	// Search through tables using regular methods (no server lock held).
	// Prefer seated tables, but fall back to spectator/watcher tables so
	// reconnect logic can restore watcher streams as well.
	for _, table := range tableRefs {
		if table.GetUser(req.PlayerId) != nil {
			config := table.GetConfig()
			return &pokerrpc.GetPlayerCurrentTableResponse{
				TableId: config.ID,
			}, nil
		}
	}
	for _, table := range tableRefs {
		config := table.GetConfig()
		if s.isTableWatcher(config.ID, req.PlayerId) {
			return &pokerrpc.GetPlayerCurrentTableResponse{
				TableId: config.ID,
			}, nil
		}
	}

	// Player is not in any table
	return &pokerrpc.GetPlayerCurrentTableResponse{TableId: ""}, nil
}

// findActiveTableForPlayer returns the table ID where the player is seated in an active game.
func (s *Server) findActiveTableForPlayer(playerID string) (string, bool) {
	tableRefs := s.getAllTables()
	for _, table := range tableRefs {
		if table.GetUser(playerID) == nil {
			continue
		}
		if table.IsGameStarted() || table.GetGame() != nil {
			return table.GetConfig().ID, true
		}
	}
	return "", false
}

func (s *Server) SetPlayerReady(ctx context.Context, req *pokerrpc.SetPlayerReadyRequest) (*pokerrpc.SetPlayerReadyResponse, error) {
	// Get table reference
	table, ok := s.getTable(req.TableId)

	if !ok {
		return nil, status.Error(codes.NotFound, "table not found")
	}

	// Enforce escrow funding when the player has bound an escrow to this table.
	user := table.GetUser(req.PlayerId)
	if user == nil {
		return nil, status.Error(codes.NotFound, "player not at table")
	}
	cfg := table.GetConfig()

	// Get user snapshot to safely read fields without races
	userSnap := user.GetSnapshot()

	if cfg.BuyIn > 0 && userSnap.EscrowID == "" {
		return nil, status.Error(codes.FailedPrecondition, "escrow required for this table")
	}
	if userSnap.EscrowID != "" {
		s.referee.mu.RLock()
		es := s.referee.escrows[userSnap.EscrowID]
		s.referee.mu.RUnlock()
		if es == nil {
			return nil, status.Errorf(codes.FailedPrecondition, "escrow %s not found for player", userSnap.EscrowID)
		}
		if cfg.BuyIn > 0 && es.AmountAtoms != uint64(cfg.BuyIn) {
			return nil, status.Errorf(codes.FailedPrecondition, "escrow amount %d must equal table buy-in %d", es.AmountAtoms, cfg.BuyIn)
		}
		if _, err := ensureBoundFunding(es); err != nil {
			return nil, status.Errorf(codes.FailedPrecondition, "escrow not funded: %v", err)
		}
		// Record binding into table model to keep FSM readiness in sync.
		s.referee.mu.Lock()
		if es.TableID == "" {
			es.TableID = req.TableId
		}
		if es.SeatIndex == 0 && userSnap.TableSeat >= 0 {
			es.SeatIndex = uint32(userSnap.TableSeat)
		}
		s.referee.mu.Unlock()
	}

	// Inform FSM so snapshots reflect readiness via player marshal.
	user.SendReady()
	table.SendPlayerReady(req.PlayerId, true)

	if snap, err := s.collectTableSnapshot(req.TableId); err == nil {
		s.publishTableSnapshotEvent(req.TableId, snap)
	}

	allReady := table.CheckAllPlayersReady()
	gameStarted := table.IsGameStarted()

	// Publish typed PLAYER_READY event
	event, err := s.buildGameEvent(
		pokerrpc.NotificationType_PLAYER_READY,
		req.TableId,
		PlayerReadyPayload{PlayerID: req.PlayerId},
	)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to build PLAYER_READY event: %v", err))
	}
	s.eventProcessor.PublishEvent(event)

	// If all players are ready and the game hasn't started yet, either:
	//   - for escrow-backed tables (BuyIn > 0), ensure presigning is complete; or
	//   - for non-escrow tables (BuyIn == 0), start immediately.
	if allReady && !gameStarted {
		matchID := req.TableId
		if cfg.BuyIn > 0 {
			complete, completedSeats, totalSeats := s.IsPresigningComplete(matchID)
			if !complete {
				s.log.Infof("All players ready but presigning not complete for table %s: %d/%d seats", req.TableId, completedSeats, totalSeats)
				// Notify players that presigning is required
				s.broadcastNotificationToTable(req.TableId, &pokerrpc.Notification{
					Type:    pokerrpc.NotificationType_PRESIGN_PENDING,
					TableId: req.TableId,
					Message: fmt.Sprintf("Waiting for presigning: %d/%d players complete. Please complete settlement presign.", completedSeats, totalSeats),
				})
				return &pokerrpc.SetPlayerReadyResponse{
					Success:         true,
					Message:         fmt.Sprintf("Player is ready. Waiting for presigning: %d/%d complete.", completedSeats, totalSeats),
					AllPlayersReady: allReady,
				}, nil
			}
		}

		// At this point either no buy-in is required, or presigning is complete.
		s.maybeStartGameAfterPresign(table, req.TableId, matchID, req.PlayerId)
	}

	return &pokerrpc.SetPlayerReadyResponse{
		Success:         true,
		Message:         "Player is ready",
		AllPlayersReady: allReady,
	}, nil
}

// maybeStartGameAfterPresign checks if the game should start after presigning completes
// and starts it if all conditions are met. This is called from both SetPlayerReady
// and when presigning completes in the referee.
func (s *Server) maybeStartGameAfterPresign(table *poker.Table, tableID, matchID, playerID string) {
	allReady := table.CheckAllPlayersReady()
	gameStarted := table.IsGameStarted()
	if !allReady || gameStarted {
		return
	}

	cfg := table.GetConfig()
	if cfg.BuyIn > 0 {
		complete, completedSeats, totalSeats := s.IsPresigningComplete(matchID)
		if !complete {
			s.log.Debugf("Presigning not yet complete for table %s: %d/%d seats", tableID, completedSeats, totalSeats)
			return
		}
		s.log.Infof("Presigning complete for table %s, starting game", tableID)
	}

	// Start game asynchronously to avoid blocking
	go func() {
		if errStart := table.StartGame(); errStart != nil {
			s.log.Errorf("Failed to start game for table %s: %v", tableID, errStart)
			return
		}

		// Publish typed GAME_STARTED event *after* the game has been
		// successfully created so that the emitted snapshot reflects the brand-new
		// game state (dealer, blinds, current player, etc.). Without this, the first
		// game update received by the clients would still be in the pre-start state
		// which prevents the UI from progressing to the actual hand.
		gameStartedEvent, errGS := s.buildGameEvent(
			pokerrpc.NotificationType_GAME_STARTED,
			tableID,
			GameStartedPayload{PlayerIDs: []string{playerID}},
		)
		if errGS != nil {
			s.log.Errorf("Failed to build GAME_STARTED event: %v", errGS)
			return
		}
		s.eventProcessor.PublishEvent(gameStartedEvent)
	}()
}

func (s *Server) SetPlayerUnready(ctx context.Context, req *pokerrpc.SetPlayerUnreadyRequest) (*pokerrpc.SetPlayerUnreadyResponse, error) {
	// Get table reference
	table, ok := s.getTable(req.TableId)

	if !ok {
		return nil, status.Error(codes.NotFound, "table not found")
	}

	if err := table.SendPlayerReady(req.PlayerId, false); err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	user := table.GetUser(req.PlayerId)
	if user == nil {
		return nil, status.Error(codes.NotFound, "player not at table")
	}
	user.SendUnready()
	table.SendPlayerReady(req.PlayerId, false)
	if snap, err := s.collectTableSnapshot(req.TableId); err == nil {
		s.publishTableSnapshotEvent(req.TableId, snap)
	}

	// Publish typed PLAYER_READY event (with ready=false)
	event, err := s.buildGameEvent(
		pokerrpc.NotificationType_PLAYER_READY,
		req.TableId,
		PlayerMarkedReadyPayload{PlayerID: req.PlayerId, Ready: false},
	)
	if err != nil {
		s.log.Errorf("Failed to build PLAYER_READY event: %v", err)
		return nil, err
	}
	s.eventProcessor.PublishEvent(event)

	return &pokerrpc.SetPlayerUnreadyResponse{
		Success: true,
		Message: "Player is unready",
	}, nil
}

// StartNotificationStream handles notification streaming
func (s *Server) StartNotificationStream(req *pokerrpc.StartNotificationStreamRequest, stream pokerrpc.LobbyService_StartNotificationStreamServer) error {
	playerID := req.PlayerId
	if playerID == "" {
		return status.Error(codes.InvalidArgument, "player ID is required")
	}

	// Create notification stream
	notifStream := &NotificationStream{
		playerID: playerID,
		stream:   stream,
		done:     make(chan struct{}),
	}

	// Register the stream in concurrent registry
	s.notificationStreams.Store(playerID, notifStream)

	// Remove stream when done
	defer func() {
		s.notificationStreams.Delete(playerID)
		close(notifStream.done)
	}()

	// Send an initial notification to ensure the stream is established
	initialNotification := &pokerrpc.Notification{
		Type:     pokerrpc.NotificationType_NOTIFICATION_STREAM_CONNECTED,
		Message:  "notification stream connected",
		PlayerId: playerID,
	}
	if err := stream.Send(initialNotification); err != nil {
		return err
	}

	// Keep the stream open and wait for context cancellation
	ctx := stream.Context()
	<-ctx.Done()
	return nil
}

// processTableEvents processes events from poker tables and forwards them to the event processor
func (s *Server) processTableEvents(eventChan <-chan poker.TableEvent) {
	for event := range eventChan {
		s.log.Debugf("Processing table event: %s for table %s", event.Type, event.TableID)

		// Convert poker table event to server game event
		ev, err := s.buildGameEvent(GameEventType(event.Type), event.TableID, event.Payload)
		if err != nil {
			s.log.Errorf("failed to build %s event: %v", event.Type, err)
			continue
		}

		// Publish to notification/state handlers
		s.eventProcessor.PublishEvent(ev)

	}
}

// removeTableFromRegistry prunes all runtime references for a table that has
// already been closed. This is triggered when a TABLE_REMOVED event is
// received, mirroring the explicit handling of TABLE_CREATED.
func (s *Server) removeTableFromRegistry(tableID string) {
	s.removeAllWatchersForTable(tableID)
	s.tables.Delete(tableID)
	s.saveMutexes.Delete(tableID)
	s.broadcastMutexes.Delete(tableID)
	s.gameStreams.Delete(tableID)
}

func (s *Server) canRemoveTable(tableID string) bool {
	table, ok := s.getTable(tableID)
	if !ok || table == nil {
		return false
	}
	if table.IsGameStarted() {
		return false
	}
	if len(table.GetUsers()) != 0 {
		return false
	}
	if s.hasPendingSettlement(tableID) {
		return false
	}
	return !s.tableHasRefereeState(tableID)
}

func (s *Server) maybeRemoveTable(tableID string) bool {
	if !s.canRemoveTable(tableID) {
		return false
	}

	ack := s.publishTableRemovedEvent(tableID)
	select {
	case <-ack:
	case <-time.After(2 * time.Second):
		s.log.Warnf("timeout waiting for table %s removal", tableID)
	}

	_, ok := s.getTable(tableID)
	return !ok
}

// getTableRemovalAck returns a completion channel for a given table removal.
// The channel is created once per tableID and closed when finalization ends.
func (s *Server) getTableRemovalAck(tableID string) chan struct{} {
	ch, _ := s.tableRemovalAcks.LoadOrStore(tableID, make(chan struct{}))
	return ch.(chan struct{})
}

// signalTableRemovalDone closes and cleans up the ack channel for a tableID.
func (s *Server) signalTableRemovalDone(tableID string) {
	if ch, ok := s.tableRemovalAcks.Load(tableID); ok {
		c := ch.(chan struct{})
		select {
		case <-c:
		default:
			close(c)
		}
		s.tableRemovalAcks.Delete(tableID)
	}
}

// scheduleTableRemoval publishes TABLE_REMOVED after a short grace period to
// allow late reads of final state (e.g., GetLastWinners) without sprinkling
// arbitrary sleeps. Returns the removal ack channel for callers to wait on.
func (s *Server) scheduleTableRemoval(tableID string) <-chan struct{} {
	const grace = 1 * time.Second
	timer := time.NewTimer(grace)
	go func() {
		defer timer.Stop()
		select {
		case <-timer.C:
			select {
			case <-s.stopChan:
				return
			default:
			}
			s.publishTableRemovedEvent(tableID)
		case <-s.stopChan:
			return
		case <-s.getTableRemovalAck(tableID):
			// Already removed elsewhere; skip.
		}
	}()
	return s.getTableRemovalAck(tableID)
}

func (s *Server) schedulePostGameTableCleanup(tableID string) {
	table, ok := s.getTable(tableID)
	if !ok || table == nil {
		return
	}

	source := table.GetConfig().Source
	switch source {
	case managedTableSourceDefault:
		s.log.Debugf("Scheduling managed table %s for removal after game end; default table manager will replace it", tableID)
		s.scheduleTableRemoval(tableID)
	case managedTableSourceUser, "":
		s.log.Debugf("Scheduling user table %s for removal after game end", tableID)
		s.scheduleTableRemoval(tableID)
	default:
		s.log.Warnf("Scheduling table %s with unknown source %q for removal after game end", tableID, source)
		s.scheduleTableRemoval(tableID)
	}
}

// publishTableRemovedEvent enqueues a TABLE_REMOVED event so the event pipeline
// can notify clients and persist the final snapshot before cleanup.
func (s *Server) publishTableRemovedEvent(tableID string) <-chan struct{} {
	ack := s.getTableRemovalAck(tableID)
	select {
	case <-s.stopChan:
		s.signalTableRemovalDone(tableID)
		return ack
	default:
	}

	evt, err := s.buildGameEvent(
		pokerrpc.NotificationType_TABLE_REMOVED,
		tableID,
		nil,
	)
	if err != nil {
		s.log.Errorf("Failed to build TABLE_REMOVED event: %v", err)
		return ack
	}
	s.eventProcessor.PublishEvent(evt)
	s.log.Debugf("Published TABLE_REMOVED event for table %s", tableID)
	return ack
}

// finalizeTableRemoval performs the irreversible cleanup after TABLE_REMOVED
// has been published. The server owns shutdown to avoid self-deadlocks in the
// table FSM.
func (s *Server) finalizeTableRemoval(tableID string) {
	// Block on any in-flight snapshot save for this table so we don't delete
	// the DB row while a GAME_ENDED persistence is writing.
	v, _ := s.saveMutexes.LoadOrStore(tableID, &sync.Mutex{})
	saveMutex, _ := v.(*sync.Mutex)
	saveMutex.Lock()
	defer saveMutex.Unlock()

	var removedCfg poker.TableConfig
	if tbl, ok := s.getTable(tableID); ok && tbl != nil {
		removedCfg = tbl.GetConfig()
		// Close is idempotent; safe if table already shut down.
		tbl.Close()
	}

	// Remove from registry - this makes getTable() return false for this table
	s.removeTableFromRegistry(tableID)

	// Delete from database
	if err := s.db.DeleteTable(context.Background(), tableID); err != nil {
		s.log.Errorf("Failed to delete table %s in DB after TABLE_REMOVED: %v", tableID, err)
	}
	if s.defaultTableMgr != nil && removedCfg.Source == managedTableSourceDefault {
		s.defaultTableMgr.notifyManagedTableRemoved(removedCfg.ID, defaultTableConfigProfileKey(removedCfg))
	}

	// Notify waiters (tests/handlers) that removal is complete.
	s.signalTableRemovalDone(tableID)
}
