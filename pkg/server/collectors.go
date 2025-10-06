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
    // Take a stable game snapshot early to avoid racing with live player mutations
    var gameStateSnap *poker.GameStateSnapshot
    if game != nil {
        snap := game.GetStateSnapshot()
        gameStateSnap = &snap
    }

	// Collect player snapshots
	playerSnapshots := make([]*PlayerSnapshot, 0, len(users))
    for _, user := range users {
        var snapshot *PlayerSnapshot
        if gameStateSnap != nil {
            snapshot = s.collectPlayerSnapshotFromGameSnapshot(user, gameStateSnap)
        } else {
            snapshot = s.collectPlayerSnapshot(user, nil)
        }
        playerSnapshots = append(playerSnapshots, snapshot)
    }

	// Collect game snapshot if game exists
	var gameSnapshot *GameSnapshot
    if game != nil {
        // If we captured a stable game snapshot above, prefer to build the
        // server GameSnapshot from the live game object, which uses getters
        // under internal locks. This is safe and keeps behavior unchanged.
        gameSnapshot = s.collectGameSnapshot(game)
        // Mirror authoritative winners from table's cached lastShowdown, if any
        if ls := table.GetLastShowdown(); ls != nil && gameSnapshot != nil {
            if len(ls.Winners) > 0 {
                gameSnapshot.Winners = make([]string, len(ls.Winners))
                copy(gameSnapshot.Winners, ls.Winners)
            } else {
                gameSnapshot.Winners = nil
            }
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
        if player == nil {
            continue
        }
        grpcPlayer := player.Marshal()
        if grpcPlayer.Id != user.ID {
            continue
        }
        snapshot.Balance = grpcPlayer.Balance
        snapshot.HasFolded = grpcPlayer.Folded
        snapshot.IsAllIn = grpcPlayer.IsAllIn
        snapshot.IsDealer = grpcPlayer.IsDealer
        snapshot.IsTurn = grpcPlayer.IsTurn
        snapshot.GameState = player.GetCurrentStateString()
        snapshot.HandDescription = grpcPlayer.HandDescription
        snapshot.HasBet = grpcPlayer.CurrentBet
        snapshot.StartingBalance = player.StartingBalance()

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

    return snapshot
}

// collectGameSnapshot collects an immutable snapshot of game state
func (s *Server) collectGameSnapshot(game *poker.Game) *GameSnapshot {
    snapshot := &GameSnapshot{
        Phase:      game.GetPhase(),
        Pot:        game.GetPot(),
        CurrentBet: game.GetCurrentBet(),
        Dealer:     game.GetDealer(),
        Round:      game.GetRound(),
        BetRound:   game.GetBetRound(),
        Winners:    make([]string, 0),
    }

    // Derive current player from a stable game snapshot to avoid races
    gs := game.GetStateSnapshot()
    for _, pl := range gs.Players {
        if pl == nil {
            continue
        }
        grpcPlayer := pl.Marshal()
        if grpcPlayer.IsTurn {
            snapshot.CurrentPlayer = grpcPlayer.Id
            break
        }
    }

    // Deep copy community cards to ensure immutability
    communityCards := game.GetCommunityCards()
    if len(communityCards) > 0 {
        snapshot.CommunityCards = make([]poker.Card, len(communityCards))
        copy(snapshot.CommunityCards, communityCards)
    }

    // Get winners if available
    winners := game.GetWinners()
    if len(winners) > 0 {
        snapshot.Winners = make([]string, len(winners))
        copy(snapshot.Winners, winners)
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
