package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
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
	timeBank := time.Duration(req.TimeBankSeconds) * time.Second
	if timeBank == 0 {
		timeBank = 30 * time.Second
	}
	startingChips := req.StartingChips
	if startingChips == 0 {
		startingChips = 1000
	}

	tblLog := s.logBackend.Logger("TABLE")
	gameLog := s.logBackend.Logger("GAME")

	// Set defaults for auto-start and auto-advance
	autoStartDelay := time.Duration(req.AutoStartMs) * time.Millisecond
	if autoStartDelay == 0 {
		autoStartDelay = 2 * time.Second // Default 2 seconds for auto-start
	}
	autoAdvanceDelay := time.Duration(req.AutoAdvanceMs) * time.Millisecond
	if autoAdvanceDelay == 0 {
		autoAdvanceDelay = 2 * time.Second // Default 2 second for auto-advance
	}

	cfg := poker.TableConfig{
		ID:               newTableID(),
		Log:              tblLog,
		GameLog:          gameLog,
		HostID:           req.PlayerId,
		BuyIn:            req.BuyIn,
		MinPlayers:       int(req.MinPlayers),
		MaxPlayers:       int(req.MaxPlayers),
		SmallBlind:       req.SmallBlind,
		BigBlind:         req.BigBlind,
		StartingChips:    startingChips,
		TimeBank:         timeBank,
		AutoStartDelay:   autoStartDelay,
		AutoAdvanceDelay: autoAdvanceDelay,
	}

	// Create table
	table := poker.NewTable(cfg)

	// persist table config before publishing events or accepting joins
	if err := s.db.UpsertTable(ctx, &cfg); err != nil {
		s.log.Errorf("UpsertTable failed: %v", err)
		return nil, status.Error(codes.Internal, "failed to persist table")
	}

	// Create a channel for table events and start a goroutine to process them
	tableEventChan := make(chan poker.TableEvent, 100) // Buffered channel
	table.SetEventChannel(tableEventChan)

	// Start a goroutine to process table events
	go s.processTableEvents(tableEventChan)

	// Seat creator
	if _, err := table.AddNewUser(req.PlayerId, &poker.AddUserOptions{
		DisplayName: s.displayNameFor(req.PlayerId),
	}); err != nil {
		return nil, err
	}
	// Persist the creator's seat so restarts can restore both participants
	if err := s.db.SeatPlayer(ctx, cfg.ID, req.PlayerId, 0); err != nil {
		s.log.Errorf("SeatPlayer (host) failed (table=%s player=%s seat=%d): %v", cfg.ID, req.PlayerId, 0, err)
	}

	// Register table using concurrent registry
	s.tables.Store(cfg.ID, table)

	// Publish initial PLAYER_JOINED event for the host
	if err := s.publishPlayerJoined(cfg.ID, req.PlayerId); err != nil {
		s.log.Errorf("Failed to publish host PLAYER_JOINED event: %v", err)
	}

	// Publish TABLE_CREATED event so all connected clients can promptly refresh
	// their lobby/waiting rooms view.
	evt, err := s.buildGameEvent(
		pokerrpc.NotificationType_TABLE_CREATED,
		cfg.ID,
		nil,
	)
	if err != nil {
		s.log.Errorf("Failed to build TABLE_CREATED event: %v", err)
		return &pokerrpc.CreateTableResponse{TableId: cfg.ID}, nil
	}
	s.eventProcessor.PublishEvent(evt)

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

	return &pokerrpc.JoinTableResponse{
		Success: true,
		Message: "Successfully joined table",
	}, nil
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

	config := table.GetConfig()
	isHost := req.PlayerId == config.HostID

	// If a game is in progress, keep the seat (disconnect)
	if table.IsGameStarted() {
		user.SendDisconnect()

		if snap, err := s.collectTableSnapshot(req.TableId); err == nil {
			s.publishTableSnapshotEvent(req.TableId, snap)
		}
		s.saveTableStateAsync(req.TableId, "player disconnected")

		return &pokerrpc.LeaveTableResponse{
			Success: true,
			Message: fmt.Sprintf("You have been disconnected but your seat is reserved because you have chips remaining."),
		}, nil
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

	// If the host leaves, transfer host if possible, else close the table
	if isHost {
		remaining := table.GetUsers()

		// Transfer to first non-host user if available
		if len(remaining) > 0 {
			var newHostID string
			for _, u := range remaining {
				if u.ID != req.PlayerId {
					newHostID = u.ID
					break
				}
			}
			if newHostID != "" {
				if err := s.transferTableHost(req.TableId, newHostID); err != nil {
					return &pokerrpc.LeaveTableResponse{Success: false, Message: err.Error()}, nil
				}
				s.saveTableStateAsync(req.TableId, "host transferred")
				if snap, err := s.collectTableSnapshot(req.TableId); err == nil {
					s.publishTableSnapshotEvent(req.TableId, snap)
				}
				return &pokerrpc.LeaveTableResponse{
					Success: true,
					Message: fmt.Sprintf("Successfully left table. Host transferred to %s", newHostID),
				}, nil
			}
		}

		// No other players: publish removal through the event pipeline.
		ack := s.publishTableRemovedEvent(req.TableId)

		// Wait briefly for cleanup so callers/tests see a consistent state.
		select {
		case <-ack:
		case <-time.After(2 * time.Second):
			s.log.Warnf("timeout waiting for table %s removal", req.TableId)
		}

		return &pokerrpc.LeaveTableResponse{
			Success: true,
			Message: "Host left - table closed (no other players)",
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

// transferTableHost transfers host ownership to a new user
func (s *Server) transferTableHost(tableID, newHostID string) error {
	table, ok := s.getTable(tableID)
	if !ok {
		return fmt.Errorf("table not found")
	}

	// Use the table's SetHost method to transfer ownership
	err := table.SetHost(newHostID)
	if err != nil {
		return fmt.Errorf("failed to transfer host: %v", err)
	}

	s.log.Infof("Host transferred to %s for table %s", newHostID, tableID)

	return nil
}

func (s *Server) GetTables(ctx context.Context, req *pokerrpc.GetTablesRequest) (*pokerrpc.GetTablesResponse, error) {
	// Snapshot current tables from concurrent registry
	tableRefs := s.getAllTables()

	// Build response using regular table methods (no server lock held)
	tables := make([]*pokerrpc.Table, 0, len(tableRefs))
	for _, table := range tableRefs {
		config := table.GetConfig()
		users := table.GetUsers()
		game := table.GetGame()

		protoTable := &pokerrpc.Table{
			Id:              config.ID,
			HostId:          config.HostID,
			SmallBlind:      config.SmallBlind,
			BigBlind:        config.BigBlind,
			MaxPlayers:      int32(table.GetMaxPlayers()),
			MinPlayers:      int32(table.GetMinPlayers()),
			CurrentPlayers:  int32(len(users)),
			BuyIn:           config.BuyIn,
			GameStarted:     game != nil,
			AllPlayersReady: table.AreAllPlayersReady(),
		}
		tables = append(tables, protoTable)
	}

	return &pokerrpc.GetTablesResponse{Tables: tables}, nil
}

func (s *Server) GetPlayerCurrentTable(ctx context.Context, req *pokerrpc.GetPlayerCurrentTableRequest) (*pokerrpc.GetPlayerCurrentTableResponse, error) {
	// Get table references with server lock
	tableRefs := s.getAllTables()

	// Search through tables using regular methods (no server lock held)
	for _, table := range tableRefs {
		if table.GetUser(req.PlayerId) != nil {
			config := table.GetConfig()
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
		s.eventProcessor.PublishEvent(ev)
	}
}

// removeTableFromRegistry prunes all runtime references for a table that has
// already been closed. This is triggered when a TABLE_REMOVED event is
// received, mirroring the explicit handling of TABLE_CREATED.
func (s *Server) removeTableFromRegistry(tableID string) {
	s.tables.Delete(tableID)
	s.saveMutexes.Delete(tableID)
	s.broadcastMutexes.Delete(tableID)
	s.gameStreams.Delete(tableID)
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
			s.publishTableRemovedEvent(tableID)
		case <-s.getTableRemovalAck(tableID):
			// Already removed elsewhere; skip.
		}
	}()
	return s.getTableRemovalAck(tableID)
}

// publishTableRemovedEvent enqueues a TABLE_REMOVED event so the event pipeline
// can notify clients and persist the final snapshot before cleanup.
func (s *Server) publishTableRemovedEvent(tableID string) <-chan struct{} {
	ack := s.getTableRemovalAck(tableID)

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

	if tbl, ok := s.getTable(tableID); ok && tbl != nil {
		// Close is idempotent; safe if table already shut down.
		tbl.Close()
	}

	// Remove from registry - this makes getTable() return false for this table
	s.removeTableFromRegistry(tableID)

	// Delete from database
	if err := s.db.DeleteTable(context.Background(), tableID); err != nil {
		s.log.Errorf("Failed to delete table %s in DB after TABLE_REMOVED: %v", tableID, err)
	}

	// Notify waiters (tests/handlers) that removal is complete.
	s.signalTableRemovalDone(tableID)
}
