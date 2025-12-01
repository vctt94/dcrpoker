package server

import (
	"fmt"
	"time"

	"github.com/vctt94/pokerbisonrelay/pkg/poker"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
)

// collectPlayerSnapshot collects an immutable snapshot of player state
func (s *Server) collectPlayerSnapshot(user *poker.User, gameSnapshot *poker.GameStateSnapshot) *PlayerSnapshot {
	snapshot := &PlayerSnapshot{
		ID:              user.ID,
		Name:            user.Name,
		TableSeat:       user.TableSeat,
		Balance:         0, // Default for users not in game
		Hand:            make([]poker.Card, 0),
		IsReady:         user.IsReady,
		EscrowID:        user.EscrowID,
		EscrowReady:     user.EscrowReady,
		PresignComplete: user.PresignComplete,
		IsDisconnected:  user.IsDisconnected,
		HasFolded:       false,
		IsAllIn:         false,
		IsDealer:        false,
		IsSmallBlind:    false,
		IsBigBlind:      false,
		IsTurn:          false,
		GameState:       poker.AT_TABLE_STATE,
		HandDescription: "",
		HasBet:          0,
		StartingBalance: 0,
		LastAction:      time.Time{},
	}

	// If game exists and player is in it, get game-specific data
	if gameSnapshot != nil && len(gameSnapshot.Players) > 0 {
		for _, ps := range gameSnapshot.Players {
			if ps.ID != user.ID {
				continue
			}
			snapshot.Balance = ps.Balance
			snapshot.HasFolded = ps.Folded
			snapshot.IsAllIn = ps.IsAllIn
			snapshot.IsDealer = ps.IsDealer
			snapshot.IsSmallBlind = ps.IsSmallBlind
			snapshot.IsBigBlind = ps.IsBigBlind
			snapshot.IsTurn = ps.IsTurn
			snapshot.GameState = ps.StateString
			snapshot.IsDisconnected = ps.IsDisconnected
			snapshot.HandDescription = ps.HandDescription
			snapshot.HasBet = ps.CurrentBet
			snapshot.StartingBalance = ps.StartingBalance
			snapshot.LastAction = ps.LastAction

			if len(ps.Hand) > 0 {
				snapshot.Hand = make([]poker.Card, len(ps.Hand))
				copy(snapshot.Hand, ps.Hand)
			}

			break
		}
	}

	return snapshot
}

// collectTableSnapshot collects a complete immutable snapshot of table state
// Follows lock hierarchy: Table.mu → Game.mu → Player.mu
func (s *Server) collectTableSnapshot(tableID string) (*TableSnapshot, error) {
	t, ok := s.getTable(tableID)
	if !ok || t == nil {
		return nil, fmt.Errorf("table not found: %s", tableID)
	}

	// Snapshot table data under table lock, then release before calling game methods
	// This follows the policy: release lock before calling into other objects
	// Use GetStateSnapshot() to safely copy User fields (IsReady, IsDisconnected) under lock
	tableSnapshot := t.GetStateSnapshot()

	config := tableSnapshot.Config
	gameStarted := t.IsGameStarted()
	allPlayersReady := t.CheckAllPlayersReady()
	game := t.GetGame() // Get reference

	// Collect game snapshot if game exists (acquires Game.mu internally)
	var gameSnapshot *poker.GameStateSnapshot
	if game != nil {
		// Get stable game state snapshot to avoid races
		gs := game.GetStateSnapshot()
		gameSnapshot = &gs
	}
	// Release table lock before accessing game/player methods (lock hierarchy)
	// Collect player snapshots (may access game/player locks)
	// Use the safe User copies from the snapshot to avoid race conditions
	players := make([]*PlayerSnapshot, 0, len(tableSnapshot.Users))
	for _, safeUser := range tableSnapshot.Users {
		playerSnapshot := s.collectPlayerSnapshot(&safeUser, gameSnapshot)
		players = append(players, playerSnapshot)
	}

	// Compute table state (using snapshot values, no locks needed)
	tableState := TableState{
		GameStarted:     gameStarted,
		AllPlayersReady: allPlayersReady,
		PlayerCount:     len(tableSnapshot.Users),
	}

	return &TableSnapshot{
		ID:           tableID,
		Players:      players,
		GameSnapshot: gameSnapshot,
		Config:       config,
		State:        tableState,
		Timestamp:    time.Now(),
	}, nil
}

func (s *Server) buildGameEvent(
	eventType pokerrpc.NotificationType,
	tableID string,
	payload interface{},
) (*GameEvent, error) {
	// Avoid collecting snapshots or taking locks for lightweight global events
	// to prevent deadlocks when called under the server write lock.
	if eventType == pokerrpc.NotificationType_TABLE_CREATED || eventType == pokerrpc.NotificationType_TABLE_REMOVED {
		return &GameEvent{
			Type:          eventType,
			TableID:       tableID,
			PlayerIDs:     nil,
			Timestamp:     time.Now(),
			TableSnapshot: nil,
			Payload:       nil,
		}, nil
	}

	t, ok := s.getTable(tableID)
	if !ok || t == nil {
		return nil, fmt.Errorf("table not found: %s", tableID)
	}

	users := t.GetUsers()
	playerIDs := make([]string, 0, len(users))
	for _, u := range users {
		playerIDs = append(playerIDs, u.ID)
	}

	// Convert poker package payloads to server payloads
	var serverPayload EventPayload
	if payload != nil {
		switch p := payload.(type) {
		case *pokerrpc.Showdown:
			serverPayload = ShowdownPayload{Showdown: p}
		case EventPayload:
			// Already a server payload
			serverPayload = p
		case poker.ActionPayload:
			// Convert poker.ActionPayload to appropriate server payload based on event type
			switch eventType {
			case pokerrpc.NotificationType_PLAYER_FOLDED:
				serverPayload = PlayerFoldedPayload{PlayerID: p.PlayerID}
			case pokerrpc.NotificationType_CHECK_MADE:
				serverPayload = CheckMadePayload{PlayerID: p.PlayerID}
			default:
				s.log.Warnf("ActionPayload for unsupported event type %s on table %s", eventType, tableID)
				serverPayload = nil
			}
		case poker.GameEndedPayload:
			// Convert poker.GameEndedPayload to server GameEndedPayload
			serverPayload = GameEndedPayload{
				WinnerID:   p.WinnerID,
				WinnerSeat: p.WinnerSeat,
				MatchID:    p.MatchID,
				TotalPot:   p.TotalPot,
			}
		default:
			s.log.Warnf("Unknown payload type %T for event %s on table %s", payload, eventType, tableID)
			serverPayload = nil
		}
	}

	// Use the reusable table snapshot collection method
	tableSnapshot, err := s.collectTableSnapshot(tableID)
	if err != nil {
		return nil, err
	}

	return &GameEvent{
		Type:          eventType,
		TableID:       tableID,
		PlayerIDs:     playerIDs,
		Timestamp:     time.Now(),
		TableSnapshot: tableSnapshot,
		Payload:       serverPayload,
	}, nil
}
