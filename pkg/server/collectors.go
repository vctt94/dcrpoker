package server

import (
	"fmt"
	"time"

	"github.com/vctt94/pokerbisonrelay/pkg/poker"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
)

// collectTableSnapshot collects a complete immutable snapshot of table state
func (s *Server) collectTableSnapshot(tableID string) (*TableSnapshot, error) {
	// Fetch table pointer from concurrent registry without coarse locks.
	table, ok := s.getTable(tableID)
	if !ok {
		return nil, fmt.Errorf("table not found: %s", tableID)
	}

	config := table.GetConfig()
	users := table.GetUsers()
	game := table.GetGame()

	// Early return if no game exists
	if game == nil {
		// Collect player snapshots without game data
		playerSnapshots := make([]*PlayerSnapshot, 0, len(users))
		for _, user := range users {
			snapshot := s.collectPlayerSnapshot(user, nil)
			playerSnapshots = append(playerSnapshots, snapshot)
		}

		// Collect table state
		tableState := TableState{
			GameStarted:     table.IsGameStarted(),
			AllPlayersReady: table.AreAllPlayersReady(),
			PlayerCount:     len(users),
		}

		return &TableSnapshot{
			ID:           tableID,
			Players:      playerSnapshots,
			GameSnapshot: nil,
			Config:       config,
			State:        tableState,
			Timestamp:    time.Now(),
		}, nil
	}

	// Take a stable game snapshot early to avoid racing with live player mutations
	snap := game.GetStateSnapshot()

	// Collect player snapshots
	playerSnapshots := make([]*PlayerSnapshot, 0, len(users))
	for _, user := range users {
		snapshot := s.collectPlayerSnapshotFromGameSnapshot(user, &snap)
		playerSnapshots = append(playerSnapshots, snapshot)
	}

	// Collect game snapshot
	gameSnapshot := s.collectGameSnapshot(&snap)
	// Mirror authoritative winners from table's cached lastShowdown, if any
	if ls := table.GetLastShowdown(); ls != nil && gameSnapshot != nil {
		if len(ls.Winners) > 0 {
			gameSnapshot.Winners = make([]string, len(ls.Winners))
			copy(gameSnapshot.Winners, ls.Winners)
		} else {
			gameSnapshot.Winners = nil
		}
	}

	// Collect table state
	tableState := TableState{
		GameStarted:     table.IsGameStarted(),
		AllPlayersReady: table.AreAllPlayersReady(),
		PlayerCount:     len(users),
	}

	return &TableSnapshot{
		ID:           tableID,
		Players:      playerSnapshots,
		GameSnapshot: gameSnapshot,
		Config:       config,
		State:        tableState,
		Timestamp:    time.Now(),
	}, nil
}

// collectPlayerSnapshot collects an immutable snapshot of player state
func (s *Server) collectPlayerSnapshot(user *poker.User, game *poker.Game) *PlayerSnapshot {
	snapshot := &PlayerSnapshot{
		ID:                user.ID,
		TableSeat:         user.TableSeat,
		Balance:           0, // Default for users not in game
		Hand:              make([]poker.Card, 0),
		DCRAccountBalance: user.DCRAccountBalance,
		IsReady:           user.IsReady,
		IsDisconnected:    false,
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

				// Deep copy hand cards to ensure immutability
				if len(grpcPlayer.Hand) > 0 {
					snapshot.Hand = make([]poker.Card, len(grpcPlayer.Hand))
					for i, grpcCard := range grpcPlayer.Hand {
						snapshot.Hand[i] = poker.NewCardFromSuitValue(
							poker.Suit(grpcCard.Suit),
							poker.Value(grpcCard.Value),
						)
					}
				}
				break
			}
		}
	}

	return snapshot
}

// collectPlayerSnapshotFromGameSnapshot builds a PlayerSnapshot using a stable
// copy of the game state to avoid races with live player mutations.
func (s *Server) collectPlayerSnapshotFromGameSnapshot(user *poker.User, gs *poker.GameStateSnapshot) *PlayerSnapshot {
	snapshot := &PlayerSnapshot{
		ID:                user.ID,
		TableSeat:         user.TableSeat,
		Balance:           0,
		Hand:              make([]poker.Card, 0),
		DCRAccountBalance: user.DCRAccountBalance,
		IsReady:           user.IsReady,
		IsDisconnected:    false,
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
	}

	if gs == nil || gs.Players == nil {
		return snapshot
	}

	for _, player := range gs.Players {
		if player.ID != user.ID {
			continue
		}
		snapshot.Balance = player.Balance
		snapshot.HasFolded = player.StateString == "FOLDED"
		snapshot.IsAllIn = player.StateString == "ALL_IN"
		snapshot.IsDealer = player.IsDealer
		snapshot.IsSmallBlind = player.IsSmallBlind
		snapshot.IsBigBlind = player.IsBigBlind
		snapshot.IsTurn = player.IsTurn
		snapshot.GameState = player.StateString
		snapshot.HandDescription = player.HandDescription
		snapshot.HasBet = player.CurrentBet
		snapshot.StartingBalance = player.StartingBalance

		if len(player.Hand) > 0 {
			snapshot.Hand = make([]poker.Card, len(player.Hand))
			copy(snapshot.Hand, player.Hand)
		}
		break
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

	tableSnapshot, err := s.collectTableSnapshot(tableID)
	if err != nil {
		return nil, err
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

	return &GameEvent{
		Type:          eventType,
		TableID:       tableID,
		PlayerIDs:     playerIDs,
		Timestamp:     time.Now(),
		TableSnapshot: tableSnapshot,
		Payload:       serverPayload,
	}, nil
}
