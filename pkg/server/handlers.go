package server

import (
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
)

// EventHandler defines the interface for handling events
type EventHandler interface {
	HandleEvent(event *GameEvent)
}

// NotificationHandler handles broadcasting notifications for events
type NotificationHandler struct {
	server *Server
}

// NewNotificationHandler creates a new notification handler
func NewNotificationHandler(server *Server) *NotificationHandler {
	return &NotificationHandler{server: server}
}

// HandleEvent processes an event and broadcasts appropriate notifications
func (nh *NotificationHandler) HandleEvent(event *GameEvent) {
	switch event.Type {
	case pokerrpc.NotificationType_TABLE_CREATED:
		nh.handleTableCreated(event)
	case pokerrpc.NotificationType_TABLE_REMOVED:
		nh.handleTableRemoved(event)
	case pokerrpc.NotificationType_BET_MADE:
		nh.handleBetMade(event)
	case pokerrpc.NotificationType_PLAYER_FOLDED:
		nh.handlePlayerFolded(event)
	case pokerrpc.NotificationType_CALL_MADE:
		nh.handleCallMade(event)
	case pokerrpc.NotificationType_CHECK_MADE:
		nh.handleCheckMade(event)
	case pokerrpc.NotificationType_GAME_STARTED:
		nh.handleGameStarted(event)
	case pokerrpc.NotificationType_GAME_ENDED:
		nh.handleGameEnded(event)
	case pokerrpc.NotificationType_PLAYER_READY:
		nh.handlePlayerReady(event)
	case pokerrpc.NotificationType_PLAYER_JOINED:
		nh.handlePlayerJoined(event)
	case pokerrpc.NotificationType_PLAYER_LEFT:
		nh.handlePlayerLeft(event)
	case pokerrpc.NotificationType_NEW_HAND_STARTED:
		nh.handleNewHandStarted(event)
	case pokerrpc.NotificationType_SHOWDOWN_RESULT:
		nh.handleShowdownResult(event)
	case pokerrpc.NotificationType_PLAYER_ALL_IN:
		nh.handlePlayerAllIn(event)
	}
}

func (nh *NotificationHandler) handleTableCreated(event *GameEvent) {
	// Inform all connected clients that a new table was created so they can
	// refresh their lobby/waiting room lists.
	notification := &pokerrpc.Notification{
		Type:    pokerrpc.NotificationType_TABLE_CREATED,
		TableId: event.TableID,
	}
	nh.server.broadcastNotificationToAll(notification)
}

func (nh *NotificationHandler) handleTableRemoved(event *GameEvent) {
	// Inform all connected clients that a table was removed so they can
	// remove it from their lobby/waiting room lists.
	notification := &pokerrpc.Notification{
		Type:    pokerrpc.NotificationType_TABLE_REMOVED,
		TableId: event.TableID,
	}
	nh.server.broadcastNotificationToAll(notification)
}

func (nh *NotificationHandler) handleBetMade(event *GameEvent) {
	pl, ok := event.Payload.(BetMadePayload)
	if !ok {
		nh.server.log.Warnf("BET_MADE without BetMadePayload; skipping (table=%s)", event.TableID)
		return
	}
	notification := &pokerrpc.Notification{
		Type:     pokerrpc.NotificationType_BET_MADE,
		PlayerId: pl.PlayerID,
		TableId:  event.TableID,
		Amount:   pl.Amount,
	}
	nh.server.notifyPlayers(event.PlayerIDs, notification)
}

func (nh *NotificationHandler) handlePlayerFolded(event *GameEvent) {
	pl, ok := event.Payload.(PlayerFoldedPayload)
	if !ok {
		nh.server.log.Warnf("PLAYER_FOLDED without PlayerFoldedPayload; skipping (table=%s)", event.TableID)
		return
	}
	notification := &pokerrpc.Notification{
		Type:     pokerrpc.NotificationType_PLAYER_FOLDED,
		PlayerId: pl.PlayerID,
		TableId:  event.TableID,
	}
	nh.server.notifyPlayers(event.PlayerIDs, notification)
}

func (nh *NotificationHandler) handleCallMade(event *GameEvent) {
	pl, ok := event.Payload.(CallMadePayload)
	if !ok {
		nh.server.log.Warnf("CALL_MADE without CallMadePayload; skipping (table=%s)", event.TableID)
		return
	}
	notification := &pokerrpc.Notification{
		Type:     pokerrpc.NotificationType_CALL_MADE,
		PlayerId: pl.PlayerID,
		TableId:  event.TableID,
		Amount:   pl.Amount, // e.g., amount called; adjust field name if different
	}
	nh.server.notifyPlayers(event.PlayerIDs, notification)
}

func (nh *NotificationHandler) handleCheckMade(event *GameEvent) {
	pl, ok := event.Payload.(CheckMadePayload)
	if !ok {
		nh.server.log.Warnf("CHECK_MADE without CheckMadePayload; skipping (table=%s)", event.TableID)
		return
	}
	notification := &pokerrpc.Notification{
		Type:     pokerrpc.NotificationType_CHECK_MADE,
		PlayerId: pl.PlayerID,
		TableId:  event.TableID,
	}
	nh.server.notifyPlayers(event.PlayerIDs, notification)
}

func (nh *NotificationHandler) handleGameStarted(event *GameEvent) {
	// payload optional; we only need table id here
	notification := &pokerrpc.Notification{
		Type:    pokerrpc.NotificationType_GAME_STARTED,
		TableId: event.TableID,
		Started: true,
	}
	nh.server.log.Debugf("Sending GAME_STARTED notification to %d players: %v", len(event.PlayerIDs), event.PlayerIDs)
	nh.server.notifyPlayers(event.PlayerIDs, notification)
}

func (nh *NotificationHandler) handleGameEnded(event *GameEvent) {
	// If you have a typed payload (e.g., winner summary), assert it here
	notification := &pokerrpc.Notification{
		Type:    pokerrpc.NotificationType_GAME_ENDED,
		TableId: event.TableID,
	}
	nh.server.notifyPlayers(event.PlayerIDs, notification)
}

func (nh *NotificationHandler) handlePlayerReady(event *GameEvent) {
	pl, ok := event.Payload.(PlayerReadyPayload)
	if !ok {
		nh.server.log.Warnf("PLAYER_READY without PlayerReadyPayload; skipping (table=%s)", event.TableID)
		return
	}
	notification := &pokerrpc.Notification{
		Type:     pokerrpc.NotificationType_PLAYER_READY,
		PlayerId: pl.PlayerID,
		TableId:  event.TableID,
	}
	nh.server.notifyPlayers(event.PlayerIDs, notification)
}

func (nh *NotificationHandler) handlePlayerJoined(event *GameEvent) {
	pl, ok := event.Payload.(PlayerJoinedPayload)
	if !ok {
		nh.server.log.Warnf("PLAYER_JOINED without PlayerJoinedPayload; skipping (table=%s)", event.TableID)
		return
	}
	notification := &pokerrpc.Notification{
		Type:     pokerrpc.NotificationType_PLAYER_JOINED,
		PlayerId: pl.PlayerID,
		TableId:  event.TableID,
	}
	// Broadcast to all so lobby lists update on every client.
	nh.server.broadcastNotificationToAll(notification)
}

func (nh *NotificationHandler) handlePlayerLeft(event *GameEvent) {
	pl, ok := event.Payload.(PlayerLeftPayload)
	if !ok {
		nh.server.log.Warnf("PLAYER_LEFT without PlayerLeftPayload; skipping (table=%s)", event.TableID)
		return
	}
	notification := &pokerrpc.Notification{
		Type:     pokerrpc.NotificationType_PLAYER_LEFT,
		PlayerId: pl.PlayerID,
		TableId:  event.TableID,
	}
	// Broadcast to all so lobby lists update on every client.
	nh.server.broadcastNotificationToAll(notification)
}

func (nh *NotificationHandler) handleNewHandStarted(event *GameEvent) {
	// If you define a payload (e.g., handID, dealerPos), assert/use it here
	notification := &pokerrpc.Notification{
		Type:    pokerrpc.NotificationType_NEW_HAND_STARTED,
		TableId: event.TableID,
	}
	nh.server.notifyPlayers(event.PlayerIDs, notification)
}

func (nh *NotificationHandler) handleShowdownResult(event *GameEvent) {
	sp, ok := event.Payload.(ShowdownPayload)
	if !ok {
		nh.server.log.Warnf("SHOWDOWN_RESULT without ShowdownPayload; skipping (table=%s)", event.TableID)
		return
	}
	notification := &pokerrpc.Notification{
		Type:     pokerrpc.NotificationType_SHOWDOWN_RESULT,
		TableId:  event.TableID,
		Showdown: sp.Showdown,
	}
	nh.server.notifyPlayers(event.PlayerIDs, notification)
}

func (nh *NotificationHandler) handlePlayerAllIn(event *GameEvent) {
	pl, ok := event.Payload.(PlayerAllInPayload)
	if !ok {
		nh.server.log.Warnf("PLAYER_ALL_IN without PlayerAllInPayload; skipping (table=%s)", event.TableID)
		return
	}
	notification := &pokerrpc.Notification{
		Type:     pokerrpc.NotificationType_PLAYER_ALL_IN,
		PlayerId: pl.PlayerID,
		TableId:  event.TableID,
		Amount:   pl.Amount,
	}
	nh.server.notifyPlayers(event.PlayerIDs, notification)
}

// ------------------------ Game State Handler ------------------------

type GameStateHandler struct {
	server *Server
}

func NewGameStateHandler(server *Server) *GameStateHandler {
	return &GameStateHandler{server: server}
}

func (gsh *GameStateHandler) HandleEvent(event *GameEvent) {
	// Build game states from the event snapshot
	gameStates := gsh.buildGameStatesFromSnapshot(event.TableSnapshot)
	if len(gameStates) > 0 {
		gsh.server.sendGameStateUpdates(event.TableID, gameStates)
	}
}

func (gsh *GameStateHandler) buildGameStatesFromSnapshot(snapshot *TableSnapshot) map[string]*pokerrpc.GameUpdate {
	if snapshot == nil {
		return nil
	}

	// Use tableID to collect fresh snapshot for each player
	tableID := snapshot.ID
	gameStates := make(map[string]*pokerrpc.GameUpdate)
	for _, playerSnapshot := range snapshot.Players {
		gameUpdate := gsh.buildGameUpdateFromSnapshot(tableID, playerSnapshot.ID)
		if gameUpdate != nil {
			gameStates[playerSnapshot.ID] = gameUpdate
		}
	}
	return gameStates
}

func (gsh *GameStateHandler) buildGameUpdateFromSnapshot(tableID string, requestingPlayerID string) *pokerrpc.GameUpdate {
	// Collect table snapshot internally to avoid passing incomplete snapshots
	tableSnapshot := gsh.server.collectTableSnapshot(tableID)
	if tableSnapshot == nil {
		return nil
	}

	// Early return if no game snapshot - return basic table info without game data
	if tableSnapshot.GameSnapshot == nil {
		// Build players list from snapshot data
		var players []*pokerrpc.Player
		for _, ps := range tableSnapshot.Players {
			player := &pokerrpc.Player{
				Id:          ps.ID,
				IsReady:     ps.IsReady,
				PlayerState: pokerrpc.PlayerState_PLAYER_STATE_AT_TABLE,
			}
			players = append(players, player)
		}

		return &pokerrpc.GameUpdate{
			TableId:         tableSnapshot.ID,
			Phase:           pokerrpc.GamePhase_WAITING,
			PhaseName:       pokerrpc.GamePhase_WAITING.String(),
			Players:         players,
			PlayersRequired: int32(tableSnapshot.Config.MinPlayers),
			PlayersJoined:   int32(tableSnapshot.State.PlayerCount),
		}
	}

	// Build players list from snapshot data
	var players []*pokerrpc.Player
	for _, ps := range tableSnapshot.Players {
		player := &pokerrpc.Player{
			Id:           ps.ID,
			Balance:      ps.Balance,
			IsReady:      ps.IsReady,
			Folded:       ps.HasFolded,
			IsAllIn:      ps.IsAllIn,
			CurrentBet:   ps.HasBet,
			PlayerState:  toRPCPlayerState(ps.GameState),
			IsDealer:     ps.IsDealer,
			IsSmallBlind: ps.IsSmallBlind,
			IsBigBlind:   ps.IsBigBlind,
			// Use FSM-derived snapshot value. UIs should prefer
			// GameUpdate.CurrentPlayer for highlighting.
			IsTurn: ps.IsTurn,
		}

		if ps.ID == requestingPlayerID {
			// Show own cards during all active game phases
			gamePhase := tableSnapshot.GameSnapshot.Phase

			if len(ps.Hand) > 0 {
				player.Hand = make([]*pokerrpc.Card, len(ps.Hand))
				for i, card := range ps.Hand {
					player.Hand[i] = &pokerrpc.Card{
						Suit:  card.GetSuit(),
						Value: card.GetValue(),
					}
				}
				gsh.server.log.Debugf("GameStateHandler: showing %d cards to player %s", len(player.Hand), ps.ID)
			} else {
				gsh.server.log.Debugf("GameStateHandler: NOT showing cards to player %s (phase=%v, handSize=%d)",
					ps.ID, gamePhase, len(ps.Hand))
			}
		} else if tableSnapshot.GameSnapshot.Phase == pokerrpc.GamePhase_SHOWDOWN {
			// Show other players' cards only during showdown
			player.Hand = make([]*pokerrpc.Card, len(ps.Hand))
			player.HandDescription = ps.HandDescription
			for i, card := range ps.Hand {
				player.Hand[i] = &pokerrpc.Card{
					Suit:  card.GetSuit(),
					Value: card.GetValue(),
				}
			}
		}

		players = append(players, player)
	}

	// Build community cards slice
	var communityCards []*pokerrpc.Card
	for _, card := range tableSnapshot.GameSnapshot.CommunityCards {
		communityCards = append(communityCards, &pokerrpc.Card{
			Suit:  card.GetSuit(),
			Value: card.GetValue(),
		})
	}

	// Compute authoritative timebank fields from snapshot
	var tbSec int32
	var deadlineMs int64
	if tableSnapshot != nil {
		if tb := tableSnapshot.Config.TimeBank; tb > 0 {
			tbSec = int32(tb.Seconds())
			// find current player snapshot and add tb to LastAction
			curID := tableSnapshot.GameSnapshot.CurrentPlayer
			if curID != "" {
				for _, ps := range tableSnapshot.Players {
					if ps.ID == curID {
						dl := ps.LastAction.Add(tableSnapshot.Config.TimeBank)
						deadlineMs = dl.UnixMilli()
						break
					}
				}
			}
		}
	}

	return &pokerrpc.GameUpdate{
		TableId:            tableSnapshot.ID,
		Phase:              tableSnapshot.GameSnapshot.Phase,
		PhaseName:          tableSnapshot.GameSnapshot.Phase.String(),
		Players:            players,
		CommunityCards:     communityCards,
		Pot:                tableSnapshot.GameSnapshot.Pot,
		CurrentBet:         tableSnapshot.GameSnapshot.CurrentBet,
		CurrentPlayer:      tableSnapshot.GameSnapshot.CurrentPlayer,
		GameStarted:        tableSnapshot.State.GameStarted,
		PlayersRequired:    int32(tableSnapshot.Config.MinPlayers),
		PlayersJoined:      int32(tableSnapshot.State.PlayerCount),
		TimeBankSeconds:    tbSec,
		TurnDeadlineUnixMs: deadlineMs,
	}
}

// toRPCPlayerState maps saved state strings to the protobuf enum.
func toRPCPlayerState(state string) pokerrpc.PlayerState {
	switch state {
	case "AT_TABLE":
		return pokerrpc.PlayerState_PLAYER_STATE_AT_TABLE
	case "IN_GAME":
		return pokerrpc.PlayerState_PLAYER_STATE_IN_GAME
	case "ALL_IN":
		return pokerrpc.PlayerState_PLAYER_STATE_ALL_IN
	case "FOLDED":
		return pokerrpc.PlayerState_PLAYER_STATE_FOLDED
	case "LEFT":
		return pokerrpc.PlayerState_PLAYER_STATE_LEFT
	default:
		return pokerrpc.PlayerState_PLAYER_STATE_AT_TABLE
	}
}

// ------------------------ Persistence Handler ------------------------

type PersistenceHandler struct {
	server *Server
}

func NewPersistenceHandler(server *Server) *PersistenceHandler {
	return &PersistenceHandler{server: server}
}

func (ph *PersistenceHandler) SaveTableStateAsync(event *GameEvent) {
	ph.server.saveTableStateAsync(event.TableID, string(event.Type))
}
