package server

import (
	"fmt"
	"sync"

	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
)

// broadcastNotification sends a notification to a specific player
func (s *Server) sendNotificationToPlayer(playerID string, notification *pokerrpc.Notification) {
	// Serialize per-player sends to avoid concurrent Send on the same stream
	muAny, _ := s.notifSendMutexes.LoadOrStore(playerID, &sync.Mutex{})
	mu := muAny.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()

	v, exists := s.notificationStreams.Load(playerID)
	if !exists {
		return // Player doesn't have an active notification stream
	}
	notifStream, _ := v.(*NotificationStream)

	select {
	case <-notifStream.done:
		return // Stream is closed
	default:
		// Send notification, ignore errors as client might have disconnected
		_ = notifStream.stream.Send(notification)
	}
}

// broadcastNotificationToAll sends a notification to all connected players
// that currently have an active notification stream.
func (s *Server) broadcastNotificationToAll(notification *pokerrpc.Notification) {
	s.notificationStreams.Range(func(key, value any) bool {
		notifStream, ok := value.(*NotificationStream)
		if !ok || notifStream == nil {
			return true
		}
		// Lock per player before sending
		playerID := notifStream.playerID
		muAny, _ := s.notifSendMutexes.LoadOrStore(playerID, &sync.Mutex{})
		mu := muAny.(*sync.Mutex)
		mu.Lock()
		{
			select {
			case <-notifStream.done:
				// Skip closed streams
			default:
				_ = notifStream.stream.Send(notification)
			}
		}
		mu.Unlock()
		return true
	})
}

// broadcastNotificationToTable sends a notification to all players at a table
func (s *Server) broadcastNotificationToTable(tableID string, notification *pokerrpc.Notification) {
	table, exists := s.getTable(tableID)
	if !exists {
		return
	}

	audience := make([]string, 0, len(table.GetUsers()))
	users := table.GetUsers()
	for _, user := range users {
		audience = append(audience, user.ID)
	}
	for _, playerID := range s.tableAudience(tableID, audience) {
		s.sendNotificationToPlayer(playerID, notification)
	}
}

// notifyPlayers sends a notification to specific players
// This version doesn't acquire the server mutex, requiring player IDs to be passed as parameters
func (s *Server) notifyPlayers(playerIDs []string, notification *pokerrpc.Notification) {
	for _, playerID := range playerIDs {
		s.notifyPlayer(playerID, notification)
	}
}

// NotificationSender interface implementation

// SendAllPlayersReady sends ALL_PLAYERS_READY notification to all players at the table
func (s *Server) SendAllPlayersReady(tableID string) {
	notification := &pokerrpc.Notification{
		Type:    pokerrpc.NotificationType_ALL_PLAYERS_READY,
		Message: "All players are ready! Game starting soon...",
		TableId: tableID,
	}

	s.broadcastNotificationToTable(tableID, notification)
}

// SendGameStarted sends GAME_STARTED notification to all players at the table
func (s *Server) SendGameStarted(tableID string) {
	notification := &pokerrpc.Notification{
		Type:    pokerrpc.NotificationType_GAME_STARTED,
		Message: "Game started!",
		TableId: tableID,
		Started: true,
	}

	s.broadcastNotificationToTable(tableID, notification)
}

// SendNewHandStarted sends NEW_HAND_STARTED notification to all players at the table
func (s *Server) SendNewHandStarted(tableID string) {
	notification := &pokerrpc.Notification{
		Type:    pokerrpc.NotificationType_NEW_HAND_STARTED,
		Message: "New hand started!",
		TableId: tableID,
	}

	s.broadcastNotificationToTable(tableID, notification)
}

// SendPlayerReady sends PLAYER_READY notification to all players at the table
func (s *Server) SendPlayerReady(tableID, playerID string, ready bool) {
	var notificationType pokerrpc.NotificationType
	var message string

	if ready {
		notificationType = pokerrpc.NotificationType_PLAYER_READY
		message = fmt.Sprintf("%s is now ready", playerID)
	} else {
		notificationType = pokerrpc.NotificationType_PLAYER_UNREADY
		message = fmt.Sprintf("%s is no longer ready", playerID)
	}

	notification := &pokerrpc.Notification{
		Type:     notificationType,
		Message:  message,
		TableId:  tableID,
		PlayerId: playerID,
		Ready:    ready,
	}

	s.broadcastNotificationToTable(tableID, notification)
}

// SendBlindPosted sends blind posted notification to all players at the table
func (s *Server) SendBlindPosted(tableID, playerID string, amount int64, isSmallBlind bool) {
	var notificationType pokerrpc.NotificationType
	var message string

	if isSmallBlind {
		notificationType = pokerrpc.NotificationType_SMALL_BLIND_POSTED
		message = fmt.Sprintf("Small blind posted: %d chips", amount)
	} else {
		notificationType = pokerrpc.NotificationType_BIG_BLIND_POSTED
		message = fmt.Sprintf("Big blind posted: %d chips", amount)
	}

	notification := &pokerrpc.Notification{
		Type:     notificationType,
		Message:  message,
		TableId:  tableID,
		PlayerId: playerID,
		Amount:   amount,
	}

	s.broadcastNotificationToTable(tableID, notification)
}

// SendShowdownResult sends SHOWDOWN_RESULT notification to all players at the table
func (s *Server) SendShowdownResult(tableID string, winners []*pokerrpc.Winner, pot int64) {
	notification := &pokerrpc.Notification{
		Type:    pokerrpc.NotificationType_SHOWDOWN_RESULT,
		Message: fmt.Sprintf("Showdown complete! Pot: %d chips", pot),
		TableId: tableID,
		Winners: winners,
		Amount:  pot,
	}

	s.broadcastNotificationToTable(tableID, notification)
}

// notifyPlayer sends a notification to a specific player
// This version only uses the notification mutex, not the main server mutex
func (s *Server) notifyPlayer(playerID string, notification *pokerrpc.Notification) {
	s.sendNotificationToPlayer(playerID, notification)
}

// sendGameStateUpdates broadcasts pre-built updates to all players at tableID.
// Assumes s.gameStreams is sync.Map{tableID -> *bucket} and bucket.streams is sync.Map{playerID -> stream}.
func (s *Server) sendGameStateUpdates(tableID string, byPlayer map[string]*pokerrpc.GameUpdate) {
	// Serialize per-table game state sends to avoid concurrent Send on same stream
	muVal, _ := s.broadcastMutexes.LoadOrStore(tableID, &sync.Mutex{})
	mu := muVal.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()

	bAny, ok := s.gameStreams.Load(tableID)
	if !ok {
		return
	}
	b, _ := bAny.(*bucket)
	if b == nil {
		return
	}

	n := int(b.count.Load())
	s.log.Debugf("sendGameStateUpdates: broadcasting to %d players on table %s", n, tableID)

	b.streams.Range(func(k, v any) bool {
		playerID, ok1 := k.(string)
		stream, ok2 := v.(pokerrpc.PokerService_StartGameStreamServer)
		if ok1 && ok2 {
			if upd, ok := byPlayer[playerID]; ok && upd != nil {
				_ = stream.Send(upd)
			}
		}
		return true
	})
}
