package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
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
		MinBalance:       req.MinBalance,
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
		// Do not emit PLAYER_JOINED again for reconnections to avoid client
		// feedback loops. The reconnecting client will immediately attach a
		// game stream and receive the initial snapshot.
		return &pokerrpc.JoinTableResponse{
			Success: true,
			Message: "Reconnected to table.",
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
	}); err != nil {
		rollbackSeat()
		return &pokerrpc.JoinTableResponse{Success: false, Message: err.Error()}, nil
	}

	// Publish join notification.
	evt, err := s.buildGameEvent(
		pokerrpc.NotificationType_PLAYER_JOINED,
		req.TableId,
		PlayerJoinedPayload{PlayerID: req.PlayerId},
	)
	if err != nil {
		s.log.Errorf("Failed to build PLAYER_JOINED event: %v", err)
		return nil, err
	}
	s.eventProcessor.PublishEvent(evt)

	return &pokerrpc.JoinTableResponse{
		Success: true,
		Message: "Successfully joined table",
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

	// Otherwise, remove completely from the table runtime first
	if err := table.RemoveUser(req.PlayerId); err != nil {
		return &pokerrpc.LeaveTableResponse{Success: false, Message: err.Error()}, nil
	}

	// Unseat in the DB (normalized schema)
	if err := s.db.UnseatPlayer(ctx, req.TableId, req.PlayerId); err != nil {
		s.log.Errorf("Failed to unseat player in DB: %v", err)
		// We continue; in-memory removal already happened.
	}

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

		// No other players: close table (runtime + DB)
		if table != nil {
			table.Close() // Properly clean up all goroutines to prevent leaks
		}
		s.tables.Delete(req.TableId)
		s.saveMutexes.Delete(req.TableId)

		// Remove table from DB; cascades will clean participants/hands/snapshot
		if err := s.db.DeleteTable(ctx, req.TableId); err != nil {
			s.log.Errorf("Failed to delete table in DB: %v", err)
		}

		// Notify clients
		evt, err := s.buildGameEvent(
			pokerrpc.NotificationType_TABLE_REMOVED,
			req.TableId,
			nil,
		)
		if err != nil {
			s.log.Errorf("Failed to build TABLE_REMOVED event: %v", err)
			return nil, err
		}
		s.eventProcessor.PublishEvent(evt)

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
			MinBalance:      config.MinBalance,
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

func (s *Server) GetBalance(ctx context.Context, req *pokerrpc.GetBalanceRequest) (*pokerrpc.GetBalanceResponse, error) {
	balance, err := s.db.GetPlayerBalance(ctx, req.PlayerId)
	if err != nil {
		// If you want precise classification, expose a sentinel from db package.
		// For now, map any error to NotFound if it mentions "player not found".
		if strings.Contains(strings.ToLower(err.Error()), "player not found") {
			return nil, status.Error(codes.NotFound, "player not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pokerrpc.GetBalanceResponse{Balance: balance}, nil
}

func (s *Server) UpdateBalance(ctx context.Context, req *pokerrpc.UpdateBalanceRequest) (*pokerrpc.UpdateBalanceResponse, error) {
	// typ then description per new signature
	if err := s.db.UpdatePlayerBalance(ctx, req.PlayerId, req.Amount, "balance_update", req.Description); err != nil {
		return nil, err
	}

	balance, err := s.db.GetPlayerBalance(ctx, req.PlayerId)
	if err != nil {
		return nil, err
	}

	return &pokerrpc.UpdateBalanceResponse{
		NewBalance: balance,
		Message:    "Balance updated successfully",
	}, nil
}

func (s *Server) ProcessTip(ctx context.Context, req *pokerrpc.ProcessTipRequest) (*pokerrpc.ProcessTipResponse, error) {
	// debit sender, credit recipient
	if err := s.db.UpdatePlayerBalance(ctx, req.FromPlayerId, -req.Amount, "tip_sent", req.Message); err != nil {
		return nil, err
	}
	if err := s.db.UpdatePlayerBalance(ctx, req.ToPlayerId, req.Amount, "tip_received", req.Message); err != nil {
		return nil, err
	}

	balance, err := s.db.GetPlayerBalance(ctx, req.ToPlayerId)
	if err != nil {
		return nil, err
	}

	return &pokerrpc.ProcessTipResponse{
		Success:    true,
		Message:    "Tip processed successfully",
		NewBalance: balance,
	}, nil
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
	if cfg.BuyIn > 0 && user.EscrowID == "" {
		return nil, status.Error(codes.FailedPrecondition, "escrow required for this table")
	}
	if user.EscrowID != "" {
		s.referee.mu.RLock()
		es := s.referee.escrows[user.EscrowID]
		s.referee.mu.RUnlock()
		if es == nil {
			return nil, status.Errorf(codes.FailedPrecondition, "escrow %s not found for player", user.EscrowID)
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
		if es.SeatIndex == 0 && user.TableSeat >= 0 {
			es.SeatIndex = uint32(user.TableSeat)
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

	// If all players are ready and the game hasn't started yet
	// notify players that presigning is required
	if allReady && !gameStarted {
		// For escrow-backed tables, verify presigning is complete before starting
		if cfg.BuyIn > 0 {
			// The matchID for poker tables is the tableID
			matchID := req.TableId
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
