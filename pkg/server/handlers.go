package server

import (
	"context"
	"fmt"
	"time"

	"github.com/vctt94/pokerbisonrelay/pkg/poker"
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
	// Finalize table removal: close table, remove from registry, delete from DB
	nh.server.finalizeTableRemoval(event.TableID)
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
	// Extract game ended payload with winner/settlement info
	pl, ok := event.Payload.(GameEndedPayload)
	if !ok {
		nh.server.log.Warnf("GAME_ENDED without GameEndedPayload; sending basic notification (table=%s)", event.TableID)
		notification := &pokerrpc.Notification{
			Type:    pokerrpc.NotificationType_GAME_ENDED,
			TableId: event.TableID,
			Message: "Game ended",
		}
		nh.server.notifyPlayers(event.PlayerIDs, notification)
		return
	}

	nh.server.log.Infof("Game ended - winner: %s, seat: %d, matchID: %s, pot: %d",
		pl.WinnerID, pl.WinnerSeat, pl.MatchID, pl.TotalPot)

	// Send personalized notifications to each player
	for _, playerID := range event.PlayerIDs {
		isWinner := playerID == pl.WinnerID
		var message string
		if isWinner {
			message = fmt.Sprintf("Congratulations! You won the game with %d chips!", pl.TotalPot)
		} else {
			message = fmt.Sprintf("Game over. %s won with %d chips.", pl.WinnerID, pl.TotalPot)
		}

		notification := &pokerrpc.Notification{
			Type:       pokerrpc.NotificationType_GAME_ENDED,
			TableId:    event.TableID,
			Message:    message,
			WinnerId:   pl.WinnerID,
			WinnerSeat: pl.WinnerSeat,
			MatchId:    pl.MatchID,
			Amount:     pl.TotalPot,
			IsWinner:   isWinner,
		}
		nh.server.sendNotificationToPlayer(playerID, notification)
	}

	// Attempt to finalize and broadcast Schnorr settlement if matchID and winner are valid.
	if pl.WinnerID != "" && pl.MatchID != "" && pl.WinnerSeat >= 0 {
		go nh.server.trySettlementBroadcast(event.TableID, pl.MatchID, pl.WinnerSeat, pl.WinnerID, event.PlayerIDs)
	}

	// After the match is finished, remove the table so subsequent RPCs
	// treat it as gone. Use a short grace period so clients/tests can
	// query final state (e.g., GetLastWinners) before the table closes.
	go func(tableID string) {
		time.Sleep(1 * time.Second)
		nh.server.publishTableRemovedEvent(tableID)
	}(event.TableID)

}

func (nh *NotificationHandler) handlePlayerReady(event *GameEvent) {
	var (
		playerID string
		ready    = true
	)

	switch pl := event.Payload.(type) {
	case PlayerReadyPayload:
		playerID = pl.PlayerID
	case PlayerMarkedReadyPayload:
		playerID = pl.PlayerID
		ready = pl.Ready
	default:
		nh.server.log.Warnf("PLAYER_READY without PlayerReadyPayload; skipping (table=%s)", event.TableID)
		return
	}
	notification := &pokerrpc.Notification{
		Type:     pokerrpc.NotificationType_PLAYER_READY,
		PlayerId: playerID,
		TableId:  event.TableID,
		Ready:    ready,
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
	// Skip game state updates for GAME_ENDED events - game is over,
	// clients should have already received the final state from SHOWDOWN_RESULT.
	// This prevents unnecessary GetLastWinners calls after game ends.
	if event.Type == pokerrpc.NotificationType_GAME_ENDED {
		return
	}

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
	gameStates := make(map[string]*pokerrpc.GameUpdate)
	for _, playerSnapshot := range snapshot.Players {
		if playerSnapshot == nil {
			continue
		}
		// Build per-player updates from the provided table snapshot.
		gameUpdate, err := gsh.buildGameUpdateFromTableSnapshot(snapshot, playerSnapshot.ID)
		if err != nil {
			gsh.server.log.Errorf("Error building game update for player %s: %v", playerSnapshot.ID, err)
			continue
		}
		if gameUpdate != nil {
			gameStates[playerSnapshot.ID] = gameUpdate
		}
	}
	return gameStates
}

func (gsh *GameStateHandler) buildGameUpdateFromSnapshot(tableSnapshot *TableSnapshot, requestingPlayerID string) (*pokerrpc.GameUpdate, error) {
	if tableSnapshot == nil {
		return nil, nil
	}
	return gsh.buildGameUpdateFromTableSnapshot(tableSnapshot, requestingPlayerID)
}

// buildGameUpdateFromTableSnapshot builds a GameUpdate for a single requesting
// player from an already-collected TableSnapshot.
func (gsh *GameStateHandler) buildGameUpdateFromTableSnapshot(tableSnapshot *TableSnapshot, requestingPlayerID string) (*pokerrpc.GameUpdate, error) {
	// Early return if no game snapshot - return basic table info without game data
	if tableSnapshot.GameSnapshot == nil {
		// Build players list from snapshot data
		var players []*pokerrpc.Player
		for _, ps := range tableSnapshot.Players {
			player := &pokerrpc.Player{
				Id:              ps.ID,
				Name:            ps.Name,
				IsReady:         ps.IsReady,
				IsDisconnected:  ps.IsDisconnected,
				EscrowId:        ps.EscrowID,
				EscrowReady:     ps.EscrowReady,
				PresignComplete: ps.PresignComplete,
				TableSeat:       int32(ps.TableSeat),
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
			SmallBlind:      tableSnapshot.Config.SmallBlind,
			BigBlind:        tableSnapshot.Config.BigBlind,
		}, nil
	}

	// Build players list from snapshot data
	var players []*pokerrpc.Player
	for _, ps := range tableSnapshot.Players {
		player := &pokerrpc.Player{
			Id:             ps.ID,
			Name:           ps.Name,
			Balance:        ps.Balance,
			IsReady:        ps.IsReady,
			Folded:         ps.HasFolded,
			IsAllIn:        ps.IsAllIn,
			CurrentBet:     ps.HasBet,
			IsDealer:       ps.IsDealer,
			IsSmallBlind:   ps.IsSmallBlind,
			IsBigBlind:     ps.IsBigBlind,
			IsDisconnected: ps.IsDisconnected,
			// Use FSM-derived snapshot value. UIs should prefer
			// GameUpdate.CurrentPlayer for highlighting.
			IsTurn:          ps.IsTurn,
			EscrowId:        ps.EscrowID,
			EscrowReady:     ps.EscrowReady,
			PresignComplete: ps.PresignComplete,
			TableSeat:       int32(ps.TableSeat),
		}

		if ps.ID == requestingPlayerID {
			// Show own cards during all active game phases
			if len(ps.Hand) > 0 {
				player.Hand = make([]*pokerrpc.Card, len(ps.Hand))
				for i, card := range ps.Hand {
					player.Hand[i] = &pokerrpc.Card{
						Suit:  card.GetSuit(),
						Value: card.GetValue(),
					}
				}
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
		SmallBlind:         tableSnapshot.Config.SmallBlind,
		BigBlind:           tableSnapshot.Config.BigBlind,
	}, nil
}

// toRPCPlayerState maps saved state strings to the protobuf enum.
func toRPCPlayerState(state string) pokerrpc.PlayerState {
	switch state {
	case poker.AT_TABLE_STATE:
		return pokerrpc.PlayerState_PLAYER_STATE_AT_TABLE
	case poker.IN_GAME_STATE:
		return pokerrpc.PlayerState_PLAYER_STATE_IN_GAME
	case poker.ALL_IN_STATE:
		return pokerrpc.PlayerState_PLAYER_STATE_ALL_IN
	case poker.FOLDED_STATE:
		return pokerrpc.PlayerState_PLAYER_STATE_FOLDED
	case poker.LEFT_TABLE_STATE:
		return pokerrpc.PlayerState_PLAYER_STATE_LEFT
	default:
		return pokerrpc.PlayerState_PLAYER_STATE_AT_TABLE
	}
}

// trySettlementBroadcast attempts to finalize and broadcast the Schnorr settlement.
// It runs asynchronously and notifies players of the result.
// For non-Schnorr tables (no escrows), this silently returns without action.
func (s *Server) trySettlementBroadcast(tableID, matchID string, winnerSeat int32, winnerID string, playerIDs []string) {
	// Check if this match has any escrows bound (i.e., is a Schnorr-enabled table).
	// If not, silently skip settlement - this is a normal non-escrow game.
	escrows, err := s.readyMatchEscrows(matchID)
	if err != nil || len(escrows) == 0 {
		s.log.Debugf("No escrows for match %s; skipping settlement broadcast (non-Schnorr table)", matchID)
		return
	}

	// Verify presigning was completed before attempting settlement.
	complete, completedSeats, totalSeats := s.IsPresigningComplete(matchID)
	if !complete {
		s.log.Warnf("Settlement skipped for match %s: presigning incomplete (%d/%d seats)", matchID, completedSeats, totalSeats)
		s.sendNotificationToPlayer(winnerID, &pokerrpc.Notification{
			Type:    pokerrpc.NotificationType_MESSAGE,
			TableId: tableID,
			Message: fmt.Sprintf("Settlement cannot proceed: presigning incomplete (%d/%d players). Game should not have started without presigning.", completedSeats, totalSeats),
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Finalize and broadcast settlement for the winning table seat. Seat-to-branch
	// mapping is handled inside GetFinalizeBundle so we pass the table seat here.
	txid, err := s.FinalizeAndBroadcastSettlement(ctx, matchID, winnerSeat)
	if err != nil {
		// Settlement failed - presigs incomplete, dcrd not connected, etc.
		s.log.Warnf("Settlement broadcast failed for match %s: %v", matchID, err)

		// Notify winner they may need to finalize manually
		s.sendNotificationToPlayer(winnerID, &pokerrpc.Notification{
			Type:    pokerrpc.NotificationType_MESSAGE,
			TableId: tableID,
			Message: fmt.Sprintf("Settlement broadcast failed: %v. You may call GetFinalizeBundle to complete manually.", err),
		})
		return
	}

	// Notify all players of the successful settlement
	for _, playerID := range playerIDs {
		var message string
		if playerID == winnerID {
			message = fmt.Sprintf("🎉 Settlement broadcasted! Your winnings are on the way. txid=%s", txid)
		} else {
			message = fmt.Sprintf("Settlement broadcasted. txid=%s", txid)
		}

		s.sendNotificationToPlayer(playerID, &pokerrpc.Notification{
			Type:    pokerrpc.NotificationType_SETTLEMENT_BROADCAST,
			TableId: tableID,
			Message: message,
		})
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
