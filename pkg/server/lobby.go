package server

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/vctt94/pokerbisonrelay/pkg/poker"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) CreateTable(ctx context.Context, req *pokerrpc.CreateTableRequest) (*pokerrpc.CreateTableResponse, error) {
	// Get creator's DCR balance
	creatorBalance, err := s.db.GetPlayerBalance(ctx, req.PlayerId)
	if err != nil {
		return nil, err
	}

	s.log.Debugf("Creating table with buy-in %d", req.BuyIn)
	if creatorBalance < req.BuyIn {
		return nil, fmt.Errorf("insufficient DCR balance for buy-in: need %d, have %d", req.BuyIn, creatorBalance)
	}

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

	// Validate AutoAdvanceMs - must be set and > 0
	autoAdvanceDelay := time.Duration(req.AutoAdvanceMs) * time.Millisecond
	if autoAdvanceDelay == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "auto_advance_ms must be set to a positive value (e.g., 1000 for 1 second)")
	}

	cfg := poker.TableConfig{
		ID:               fmt.Sprintf("table_%d", time.Now().UnixNano()),
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
		AutoStartDelay:   time.Duration(req.AutoStartMs) * time.Millisecond,
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
	if _, err := table.AddNewUser(req.PlayerId, req.PlayerId, creatorBalance, 0); err != nil {
		return nil, err
	}

	// Deduct buy-in
	if err := s.db.UpdatePlayerBalance(ctx, req.PlayerId, -req.BuyIn, "table buy-in", "created table"); err != nil {
		return nil, err
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

	s.log.Debugf("Joining table %s", req.TableId)
	config := table.GetConfig()

	// Reconnect: already seated in-memory.
	if existingUser := table.GetUser(req.PlayerId); existingUser != nil {
		// Do not emit PLAYER_JOINED again for reconnections to avoid client
		// feedback loops. The reconnecting client will immediately attach a
		// game stream and receive the initial snapshot.
		return &pokerrpc.JoinTableResponse{
			Success:    true,
			Message:    fmt.Sprintf("Reconnected. You have %d DCR balance.", existingUser.DCRAccountBalance),
			NewBalance: existingUser.DCRAccountBalance,
		}, nil
	}

	// Verify wallet balance.
	dcrBalance, err := s.db.GetPlayerBalance(ctx, req.PlayerId)
	if err != nil {
		return nil, err
	}
	if dcrBalance < config.BuyIn {
		return &pokerrpc.JoinTableResponse{Success: false, Message: "Insufficient DCR balance for buy-in"}, nil
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

	// Add to in-memory table.
	newUser, err := table.AddNewUser(req.PlayerId, req.PlayerId, dcrBalance, seat)
	if err != nil {
		rollbackSeat()
		return &pokerrpc.JoinTableResponse{Success: false, Message: err.Error()}, nil
	}

	// Deduct buy-in from wallet.
	if err := s.db.UpdatePlayerBalance(ctx, req.PlayerId, -config.BuyIn, "table_buy_in", "joined table"); err != nil {
		table.RemoveUser(req.PlayerId)
		rollbackSeat()
		return nil, err
	}
	_ = table.SetUserDCRAccountBalance(req.PlayerId, dcrBalance-config.BuyIn)

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
		Success:    true,
		Message:    "Successfully joined table",
		NewBalance: newUser.DCRAccountBalance,
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

	// Check if player has chips in an active game
	var playerChips int64
	if table.IsGameStarted() && table.GetGame() != nil {
		game := table.GetGame()
		for _, p := range game.GetPlayers() {
			if p.ID() == req.PlayerId {
				playerChips = p.Balance()
				break
			}
		}
	}

	// If a hand is in progress AND player still has chips, keep the seat (disconnect)
	if table.IsGameStarted() && playerChips > 0 {
		user.IsDisconnected = true
		// Optional: if you want to persist lobby readiness, add SetReady to Database interface and call it here.
		s.saveTableStateAsync(req.TableId, "player disconnected")

		return &pokerrpc.LeaveTableResponse{
			Success: true,
			Message: fmt.Sprintf("You have been disconnected but your seat is reserved. You have %d chips remaining.", playerChips),
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

	// Refund buy-in if no hand has started
	refundAmount := int64(0)
	if !table.IsGameStarted() {
		refundAmount = config.BuyIn
		if err := s.db.UpdatePlayerBalance(ctx, req.PlayerId, refundAmount, "table_refund", "left table"); err != nil {
			return nil, err
		}
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

	// Use table method to set player ready - table handles its own locking
	// Following lock hierarchy: Server → Table (no server lock held during table operation)
	err := table.SetPlayerReady(req.PlayerId, true)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
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
	// If all players are ready and the game hasn't started yet, start the game
	if allReady && !gameStarted {
		// Start game asynchronously to avoid blocking this RPC handler
		// (StartGame now waits for FSM to reach PRE_FLOP before returning)
		go func() {
			if errStart := table.StartGame(); errStart != nil {
				s.log.Errorf("Failed to start game for table %s: %v", req.TableId, errStart)
				return
			}

			// Publish typed GAME_STARTED event *after* the game has been
			// successfully created so that the emitted snapshot reflects the brand-new
			// game state (dealer, blinds, current player, etc.). Without this, the first
			// game update received by the clients would still be in the pre-start state
			// which prevents the UI from progressing to the actual hand.
			gameStartedEvent, errGS := s.buildGameEvent(
				pokerrpc.NotificationType_GAME_STARTED,
				req.TableId,
				GameStartedPayload{PlayerIDs: []string{req.PlayerId}},
			)
			if errGS != nil {
				s.log.Errorf("Failed to build GAME_STARTED event: %v", errGS)
				return
			}
			s.eventProcessor.PublishEvent(gameStartedEvent)
		}()
	}

	return &pokerrpc.SetPlayerReadyResponse{
		Success:         true,
		Message:         "Player is ready",
		AllPlayersReady: allReady,
	}, nil
}

func (s *Server) SetPlayerUnready(ctx context.Context, req *pokerrpc.SetPlayerUnreadyRequest) (*pokerrpc.SetPlayerUnreadyResponse, error) {
	// Get table reference
	table, ok := s.getTable(req.TableId)

	if !ok {
		return nil, status.Error(codes.NotFound, "table not found")
	}

	// Use table method to set player unready - table handles its own locking
	// Following lock hierarchy: Server → Table (no server lock held during table operation)
	err := table.SetPlayerReady(req.PlayerId, false)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
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
		Type:     pokerrpc.NotificationType_UNKNOWN,
		Message:  "Connected to notification stream",
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
