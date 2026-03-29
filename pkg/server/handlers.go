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
	case pokerrpc.NotificationType_PLAYER_LOST:
		nh.handlePlayerLost(event)
	case pokerrpc.NotificationType_NEW_HAND_STARTED:
		nh.handleNewHandStarted(event)
	case pokerrpc.NotificationType_SHOWDOWN_RESULT:
		nh.handleShowdownResult(event)
	case pokerrpc.NotificationType_PLAYER_ALL_IN:
		nh.handlePlayerAllIn(event)
	case pokerrpc.NotificationType_CARDS_SHOWN:
		nh.handleCardsShown(event)
	}
}

func (nh *NotificationHandler) handleTableCreated(event *GameEvent) {
	table := nh.tableFromSnapshot(event)
	// Inform all connected clients that a new table was created so they can
	// refresh their lobby/waiting room lists.
	notification := &pokerrpc.Notification{
		Type:    pokerrpc.NotificationType_TABLE_CREATED,
		TableId: event.TableID,
		Table:   table,
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

	winnerLabel := pl.WinnerID
	if table, ok := nh.server.getTable(event.TableID); ok && table != nil {
		if winner := table.GetUser(pl.WinnerID); winner != nil && winner.Name != "" {
			winnerLabel = winner.Name
		}
	}

	// Send personalized notifications to each player
	for _, playerID := range event.PlayerIDs {
		isWinner := playerID == pl.WinnerID
		var message string
		if isWinner {
			message = fmt.Sprintf("Congratulations! You won the game with %d chips!", pl.TotalPot)
		} else {
			message = fmt.Sprintf("Game over. %s won with %d chips.", winnerLabel, pl.TotalPot)
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
		if nh.server.matchHasEscrows(pl.MatchID) {
			nh.server.markPendingSettlement(pl.MatchID)
		}
		go nh.server.trySettlementBroadcast(event.TableID, pl.MatchID, pl.WinnerSeat, pl.WinnerID, event.PlayerIDs)
	}

	nh.server.schedulePostGameTableCleanup(event.TableID)
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

	// Include an up-to-date lobby snapshot so UIs can immediately reflect
	table := nh.tableFromSnapshot(event)

	notification := &pokerrpc.Notification{
		Type:     pokerrpc.NotificationType_PLAYER_READY,
		PlayerId: playerID,
		TableId:  event.TableID,
		Ready:    ready,
		Table:    table,
	}
	nh.server.notifyPlayers(event.PlayerIDs, notification)
}

func (nh *NotificationHandler) handlePlayerJoined(event *GameEvent) {
	pl, ok := event.Payload.(PlayerJoinedPayload)
	if !ok {
		nh.server.log.Warnf("PLAYER_JOINED without PlayerJoinedPayload; skipping (table=%s)", event.TableID)
		return
	}
	table := nh.tableFromSnapshot(event)
	notification := &pokerrpc.Notification{
		Type:     pokerrpc.NotificationType_PLAYER_JOINED,
		PlayerId: pl.PlayerID,
		TableId:  event.TableID,
		Table:    table,
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
	table := nh.tableFromSnapshot(event)
	notification := &pokerrpc.Notification{
		Type:     pokerrpc.NotificationType_PLAYER_LEFT,
		PlayerId: pl.PlayerID,
		TableId:  event.TableID,
		Table:    table,
	}
	// Broadcast to all so lobby lists update on every client.
	nh.server.broadcastNotificationToAll(notification)
}

func (nh *NotificationHandler) handlePlayerLost(event *GameEvent) {
	pl, ok := event.Payload.(PlayerLostPayload)
	if !ok {
		nh.server.log.Warnf("PLAYER_LOST without PlayerLostPayload; skipping (table=%s)", event.TableID)
		return
	}
	notification := &pokerrpc.Notification{
		Type:     pokerrpc.NotificationType_PLAYER_LOST,
		PlayerId: pl.PlayerID,
		TableId:  event.TableID,
		Message:  "You lost all your chips and have been removed from the table.",
	}
	// Send specifically to the player who lost, and also broadcast to others
	nh.server.sendNotificationToPlayer(pl.PlayerID, notification)
	// Also broadcast to all so lobby lists update
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

func (nh *NotificationHandler) handleCardsShown(event *GameEvent) {
	pl, ok := event.Payload.(AutoShowCardsPayload)
	if !ok {
		nh.server.log.Warnf("CARDS_SHOWN without AutoShowCardsPayload; skipping (table=%s)", event.TableID)
		return
	}

	msg := "Cards revealed"
	if pl.PlayerID != "" {
		msg = fmt.Sprintf("%s's cards were revealed", pl.PlayerID)
	}

	notification := &pokerrpc.Notification{
		Type:     pokerrpc.NotificationType_CARDS_SHOWN,
		TableId:  event.TableID,
		PlayerId: pl.PlayerID,
		Cards:    pl.Cards,
		Message:  msg,
	}
	nh.server.notifyPlayers(event.PlayerIDs, notification)
}

// tableFromSnapshot converts the event snapshot (or a fresh snapshot) into a
// proto Table for lobby updates.
func (nh *NotificationHandler) tableFromSnapshot(event *GameEvent) *pokerrpc.Table {
	snap := event.TableSnapshot
	if snap == nil {
		// TABLE_CREATED/TABLE_REMOVED events don't collect snapshots by design,
		// so fetch one lazily for lobby notifications.
		var err error
		snap, err = nh.server.collectTableSnapshot(event.TableID)
		if err != nil {
			nh.server.log.Warnf("failed to collect table snapshot for %s: %v", event.TableID, err)
			return nil
		}
	}
	return tableSnapshotToProtoTable(snap)
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
	gameStates := gsh.buildGameStatesFromSnapshot(event.TableSnapshot, event.PlayerIDs)
	if len(gameStates) > 0 {
		gsh.server.sendGameStateUpdates(event.TableID, gameStates)
	}
}

func (gsh *GameStateHandler) buildGameStatesFromSnapshot(snapshot *TableSnapshot, recipientIDs []string) map[string]*pokerrpc.GameUpdate {
	if snapshot == nil {
		return nil
	}
	if len(recipientIDs) == 0 {
		recipientIDs = snapshot.playerIDs()
	}
	gameStates := make(map[string]*pokerrpc.GameUpdate)
	for _, playerID := range recipientIDs {
		if playerID == "" {
			continue
		}
		// Build per-recipient updates from the provided table snapshot.
		gameUpdate, err := gsh.buildGameUpdateFromTableSnapshot(snapshot, playerID)
		if err != nil {
			gsh.server.log.Errorf("Error building game update for player %s: %v", playerID, err)
			continue
		}
		if gameUpdate != nil {
			gameStates[playerID] = gameUpdate
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

	// Build players list from the authoritative game snapshot when available.
	// If the game snapshot lacks per-player data (e.g., early phases in tests),
	// fall back to the table-level snapshots so we still emit complete rosters.
	userSnapshots := make(map[string]*PlayerSnapshot, len(tableSnapshot.Players))
	for _, us := range tableSnapshot.Players {
		if us != nil && us.ID != "" {
			userSnapshots[us.ID] = us
		}
	}

	gPlayers := tableSnapshot.GameSnapshot.Players
	if len(gPlayers) == 0 {
		gPlayers = make([]poker.PlayerSnapshot, 0, len(tableSnapshot.Players))
		for _, us := range tableSnapshot.Players {
			if us == nil || us.ID == "" {
				continue
			}
			gPlayers = append(gPlayers, poker.PlayerSnapshot{
				ID:              us.ID,
				Name:            us.Name,
				TableSeat:       us.TableSeat,
				IsReady:         us.IsReady,
				IsDisconnected:  us.IsDisconnected,
				LastAction:      us.LastAction,
				Balance:         us.Balance,
				StartingBalance: us.StartingBalance,
				Folded:          us.HasFolded,
				IsAllIn:         us.IsAllIn,
				Hand:            us.Hand,
				CurrentBet:      us.HasBet,
				IsDealer:        us.IsDealer,
				IsSmallBlind:    us.IsSmallBlind,
				IsBigBlind:      us.IsBigBlind,
				IsTurn:          us.IsTurn,
				StateString:     us.GameState,
				HandDescription: us.HandDescription,
				CardsRevealed:   us.CardsRevealed,
			})
		}
	}

	var players []*pokerrpc.Player
	for _, gps := range gPlayers {
		if gps.ID == "" {
			continue
		}
		// If the table roster no longer contains this player (e.g., after PLAYER_LOST),
		// skip it to keep the game state aligned with the current table membership.
		userSnap, ok := userSnapshots[gps.ID]
		if !ok {
			continue
		}

		player := &pokerrpc.Player{
			Id:             gps.ID,
			Name:           gps.Name,
			Balance:        gps.Balance,
			IsReady:        gps.IsReady,
			Folded:         gps.Folded,
			IsAllIn:        gps.IsAllIn,
			CurrentBet:     gps.CurrentBet,
			IsDealer:       gps.IsDealer,
			IsSmallBlind:   gps.IsSmallBlind,
			IsBigBlind:     gps.IsBigBlind,
			IsDisconnected: gps.IsDisconnected,
			IsTurn:         gps.IsTurn,
			TableSeat:      int32(gps.TableSeat),
			CardsRevealed:  gps.CardsRevealed,
		}

		// Prefer table-level metadata when present.
		if userSnap != nil {
			if userSnap.Name != "" {
				player.Name = userSnap.Name
			}
			player.IsReady = player.IsReady || userSnap.IsReady
			player.IsDisconnected = player.IsDisconnected || userSnap.IsDisconnected
			player.TableSeat = int32(userSnap.TableSeat)
			player.EscrowId = userSnap.EscrowID
			player.EscrowReady = userSnap.EscrowReady
			player.PresignComplete = userSnap.PresignComplete
		}

		// Determine visibility for hand info
		showdownReady := tableSnapshot.GameSnapshot.Phase == pokerrpc.GamePhase_SHOWDOWN || tableSnapshot.LastShowdown != nil
		autoRevealVisible := false
		if !showdownReady {
			active := 0
			for _, ps := range gPlayers {
				if ps.ID == "" || ps.Folded {
					continue
				}
				if !ps.IsAllIn {
					active++
				}
			}
			// During auto-advance (all players all-in or only one actionable), revealed cards
			// should be visible even before showdown completes.
			autoRevealVisible = active <= 1
		}
		// Reveal opponent cards only at showdown (or when replaying the last showdown).
		showCards := gps.ID == requestingPlayerID || (gps.CardsRevealed && (showdownReady || autoRevealVisible))
		if showCards && len(gps.Hand) > 0 {
			player.Hand = make([]*pokerrpc.Card, len(gps.Hand))
			for i, card := range gps.Hand {
				player.Hand[i] = &pokerrpc.Card{
					Suit:  card.GetSuit(),
					Value: card.GetValue(),
				}
			}
		}
		if showCards {
			player.HandDescription = gps.HandDescription
		} else {
			player.HandDescription = ""
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
			curID := tableSnapshot.GameSnapshot.CurrentPlayer
			if curID != "" {
				for _, ps := range gPlayers {
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
		PlayersJoined:      int32(len(players)),
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

// tableSnapshotToProtoTable converts a TableSnapshot into a pokerrpc.Table for lobby notifications.
func tableSnapshotToProtoTable(snapshot *TableSnapshot) *pokerrpc.Table {
	if snapshot == nil {
		return nil
	}
	table := &pokerrpc.Table{
		Id:              snapshot.ID,
		Name:            snapshot.Config.Name,
		SmallBlind:      snapshot.Config.SmallBlind,
		BigBlind:        snapshot.Config.BigBlind,
		MaxPlayers:      int32(snapshot.Config.MaxPlayers),
		MinPlayers:      int32(snapshot.Config.MinPlayers),
		CurrentPlayers:  int32(snapshot.State.PlayerCount),
		BuyIn:           snapshot.Config.BuyIn,
		GameStarted:     snapshot.State.GameStarted,
		AllPlayersReady: snapshot.State.AllPlayersReady,
		Phase:           pokerrpc.GamePhase_WAITING,
	}

	if snapshot.GameSnapshot != nil {
		table.Phase = snapshot.GameSnapshot.Phase
	}

	for _, ps := range snapshot.Players {
		if ps == nil {
			continue
		}
		player := &pokerrpc.Player{
			Id:              ps.ID,
			Name:            ps.Name,
			Balance:         ps.Balance,
			CurrentBet:      ps.HasBet,
			Folded:          ps.HasFolded,
			IsTurn:          ps.IsTurn,
			IsAllIn:         ps.IsAllIn,
			IsDealer:        ps.IsDealer,
			IsReady:         ps.IsReady,
			HandDescription: ps.HandDescription,
			PlayerState:     toRPCPlayerState(ps.GameState),
			IsSmallBlind:    ps.IsSmallBlind,
			IsBigBlind:      ps.IsBigBlind,
			IsDisconnected:  ps.IsDisconnected,
			EscrowId:        ps.EscrowID,
			EscrowReady:     ps.EscrowReady,
			TableSeat:       int32(ps.TableSeat),
			PresignComplete: ps.PresignComplete,
		}
		table.Players = append(table.Players, player)
	}

	return table
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
	// TABLE_REMOVED is handled synchronously in the event worker to avoid races.
	if event.Type == pokerrpc.NotificationType_TABLE_REMOVED {
		return
	}
	ph.server.saveTableStateAsync(event.TableID, string(event.Type))
}
