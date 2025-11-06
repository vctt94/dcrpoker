package server

import (
	"fmt"
	"time"

	"github.com/vctt94/pokerbisonrelay/pkg/poker"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
)

// collectPlayerSnapshot collects an immutable snapshot of player state
func (s *Server) collectPlayerSnapshot(user *poker.User, game *poker.Game) *PlayerSnapshot {
	snapshot := &PlayerSnapshot{
		ID:                user.ID,
		TableSeat:         user.TableSeat,
		Balance:           0, // Default for users not in game
		Hand:              make([]poker.Card, 0),
		DCRAccountBalance: user.DCRAccountBalance,
		IsReady:           user.IsReady,
		IsDisconnected:    user.IsDisconnected,
		HasFolded:         false,
		IsAllIn:           false,
		IsDealer:          false,
		IsSmallBlind:      false,
		IsBigBlind:        false,
		IsTurn:            false,
		GameState:         "AT_TABLE",
		HandDescription:   "",
		HasBet:            0,
		StartingBalance:   0,
		LastAction:        time.Time{},
	}

	// If game exists and player is in it, get game-specific data
	if game != nil {
		for _, player := range game.GetPlayers() {
			grpcPlayer := player.Marshal()
			if grpcPlayer.Id == user.ID {
				snapshot.Balance = grpcPlayer.Balance
				snapshot.HasFolded = grpcPlayer.Folded
				snapshot.IsAllIn = grpcPlayer.IsAllIn
				snapshot.IsDealer = grpcPlayer.IsDealer
				snapshot.IsSmallBlind = grpcPlayer.IsSmallBlind
				snapshot.IsBigBlind = grpcPlayer.IsBigBlind
				snapshot.IsTurn = grpcPlayer.IsTurn
				snapshot.GameState = player.GetCurrentStateString()
				snapshot.HandDescription = grpcPlayer.HandDescription
				snapshot.HasBet = grpcPlayer.CurrentBet
				snapshot.StartingBalance = player.StartingBalance()

				// Get hand cards from Game.currentHand
				// Player can always see their own cards
				if currentHand := game.GetCurrentHand(); currentHand != nil {
					cards := currentHand.GetPlayerCards(user.ID, user.ID)
					if len(cards) > 0 {
						snapshot.Hand = make([]poker.Card, len(cards))
						copy(snapshot.Hand, cards)
					}
				}
				break
			}
		}
	}

	return snapshot
}

// collectGameSnapshotStable builds a GameSnapshot from a stable GameStateSnapshot,
// without falling back to live getters. All required fields must be present.
func (s *Server) collectGameSnapshot(gs *poker.GameStateSnapshot) *GameSnapshot {
	if gs == nil {
		return nil
	}
	snapshot := &GameSnapshot{
		Phase:      gs.Phase,
		Pot:        gs.Pot,
		CurrentBet: gs.CurrentBet,
		Dealer:     gs.Dealer,
		Round:      gs.Round,
		BetRound:   gs.BetRound,
		Winners:    make([]string, 0),
	}
	// Current player from snapshot if available
	snapshot.CurrentPlayer = gs.CurrentPlayer
	// Deep copy community cards from snapshot
	if len(gs.CommunityCards) > 0 {
		snapshot.CommunityCards = make([]poker.Card, len(gs.CommunityCards))
		copy(snapshot.CommunityCards, gs.CommunityCards)
	}

	// Get winners if available
	if len(gs.Winners) > 0 {
		snapshot.Winners = make([]string, len(gs.Winners))
		for i, winner := range gs.Winners {
			snapshot.Winners[i] = winner.ID
		}
	}

	return snapshot
}

// collectTableSnapshot collects a complete immutable snapshot of table state
// Follows lock hierarchy: Table.mu → Game.mu → Player.mu
func (s *Server) collectTableSnapshot(tableID string) *TableSnapshot {
	t, ok := s.getTable(tableID)
	if !ok || t == nil {
		return nil
	}

	// Snapshot table data under table lock, then release before calling game methods
	// This follows the policy: release lock before calling into other objects
	users := t.GetUsers()
	config := t.GetConfig()
	gameStarted := t.IsGameStarted()
	allPlayersReady := t.CheckAllPlayersReady()
	game := t.GetGame() // Get reference, don't hold lock while using it

	// Release table lock before accessing game/player methods (lock hierarchy)
	// Collect player snapshots (may access game/player locks)
	players := make([]*PlayerSnapshot, 0, len(users))
	for _, user := range users {
		playerSnapshot := s.collectPlayerSnapshot(user, game)
		players = append(players, playerSnapshot)
	}

	// Collect game snapshot if game exists (acquires Game.mu internally)
	var gameSnapshot *GameSnapshot
	if game != nil {
		// Get stable game state snapshot to avoid races
		gs := game.GetStateSnapshot()
		gameSnapshot = s.collectGameSnapshot(&gs)
	}

	// Compute table state (using snapshot values, no locks needed)
	tableState := TableState{
		GameStarted:     gameStarted,
		AllPlayersReady: allPlayersReady,
		PlayerCount:     len(users),
	}

	return &TableSnapshot{
		ID:           tableID,
		Players:      players,
		GameSnapshot: gameSnapshot,
		Config:       config,
		State:        tableState,
		Timestamp:    time.Now(),
	}
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
		default:
			s.log.Warnf("Unknown payload type %T for event %s on table %s", payload, eventType, tableID)
			serverPayload = nil
		}
	}

	// Use the reusable table snapshot collection method
	tableSnapshot := s.collectTableSnapshot(tableID)

	return &GameEvent{
		Type:          eventType,
		TableID:       tableID,
		PlayerIDs:     playerIDs,
		Timestamp:     time.Now(),
		TableSnapshot: tableSnapshot,
		Payload:       serverPayload,
	}, nil
}
