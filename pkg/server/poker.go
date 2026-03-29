package server

import (
	"context"
	"fmt"
	"time"

	"github.com/vctt94/pokerbisonrelay/pkg/poker"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) StartGameStream(req *pokerrpc.StartGameStreamRequest, stream pokerrpc.PokerService_StartGameStreamServer) error {
	// Track this stream handler to ensure it completes before Server.Stop() waits on saveWg
	s.streamHandlersWg.Add(1)
	defer s.streamHandlersWg.Done()

	tableID, playerID := req.TableId, req.PlayerId

	// Ensure the table exists and the player is currently seated before wiring the stream.
	table, ok := s.getTable(tableID)
	if !ok {
		return status.Error(codes.NotFound, "table not found")
	}
	user := table.GetUser(playerID)
	isSeated := user != nil
	isSpectator := s.isTableWatcher(tableID, playerID)
	if !isSeated && !isSpectator {
		return status.Error(codes.NotFound, "player not at table (join or watch first)")
	}

	// Get or create the table bucket (single step).
	bAny, _ := s.gameStreams.LoadOrStore(tableID, &bucket{})
	b := bAny.(*bucket)

	// Register player stream. If a stream already exists for this player,
	// replace it with the newest one without incrementing the count. This
	// ensures the most recent attachment (e.g., Flutter UI) receives updates
	// and avoids starving newer clients when multiple components attach.
	if _, loaded := b.streams.Load(playerID); !loaded {
		b.count.Add(1)
	}
	b.streams.Store(playerID, stream)

	// Unregister on exit only if this goroutine still owns the stored stream.
	// This prevents a replaced (older) stream from deleting the newer mapping.
	defer func() {
		if isSeated {
			// Inform table peers this player's game stream disconnected.
			s.broadcastNotificationToTable(tableID, &pokerrpc.Notification{
				Type:     pokerrpc.NotificationType_GAME_STREAM_DISCONNECTED,
				Message:  "game stream disconnected",
				TableId:  tableID,
				PlayerId: playerID,
			})
		}

		if v, present := b.streams.Load(playerID); present && v == stream {
			b.streams.Delete(playerID)
			if b.count.Add(-1) == 0 {
				// Remove this bucket iff it's still the same one we used.
				s.gameStreams.CompareAndDelete(tableID, b)
			}

			if isSeated {
				// Handle player disconnect when stream closes (safe to call - checks conditions internally)
				s.handlePlayerDisconnect(tableID, playerID)
			}
		}
	}()

	if isSeated {
		user.SendReconnection()

		// Inform table peers this player's game stream is connected (reconnected).
		s.broadcastNotificationToTable(tableID, &pokerrpc.Notification{
			Type:     pokerrpc.NotificationType_GAME_STREAM_CONNECTED,
			Message:  "game stream connected",
			TableId:  tableID,
			PlayerId: playerID,
		})
	}

	// Build a fresh snapshot to broadcast the reconnection and seed this stream.
	gsh := NewGameStateHandler(s)
	tableSnapshot, err := s.collectTableSnapshot(tableID)
	if err != nil {
		return err
	}

	if isSeated {
		// Broadcast connection state to other players via event pipeline
		s.publishTableSnapshotEvent(tableID, tableSnapshot)
	}

	upd, err := gsh.buildGameUpdateFromSnapshot(tableSnapshot, playerID)
	if err != nil {
		return err
	}
	if err := stream.Send(upd); err != nil {
		return err
	}

	// Wait for stream context to be done (client disconnected)
	<-stream.Context().Done()
	// XXX mark player is disconnected?
	return nil
}

// handlePlayerDisconnect handles player disconnection when their game stream closes
func (s *Server) handlePlayerDisconnect(tableID, playerID string) {
	table, ok := s.getTable(tableID)
	if !ok {
		// Table doesn't exist, nothing to do
		return
	}

	// Broadcast updated game state so remaining players see the disconnect flag.
	if tableSnapshot, err := s.collectTableSnapshot(tableID); err == nil && tableSnapshot != nil {
		s.publishTableSnapshotEvent(tableID, tableSnapshot)
	}

	// Send disconnect event
	user := table.GetUser(playerID)
	if user == nil {
		return
	}
	user.SendDisconnect()
	// Save table state asynchronously
	s.saveTableStateAsync(tableID, "player disconnected")
}

func (s *Server) publishTableSnapshotEvent(tableID string, snapshot *TableSnapshot) {
	if snapshot == nil || s.eventProcessor == nil {
		return
	}

	s.eventProcessor.PublishEvent(&GameEvent{
		Type:          pokerrpc.NotificationType_GAME_STATE_UPDATED,
		TableID:       tableID,
		PlayerIDs:     s.tableAudience(tableID, snapshot.playerIDs()),
		Timestamp:     time.Now(),
		TableSnapshot: snapshot,
	})
}

func (s *Server) requireSeatedPlayer(table *poker.Table, playerID string) error {
	if table.GetUser(playerID) == nil {
		return status.Error(codes.FailedPrecondition, "player not at table")
	}
	return nil
}

func (s *Server) MakeBet(ctx context.Context, req *pokerrpc.MakeBetRequest) (*pokerrpc.MakeBetResponse, error) {
	table, ok := s.getTable(req.TableId)
	if !ok {
		return nil, status.Error(codes.NotFound, "table not found")
	}
	if err := s.requireSeatedPlayer(table, req.PlayerId); err != nil {
		return nil, err
	}

	if !table.IsGameStarted() {
		return nil, status.Error(codes.FailedPrecondition, "game not started")
	}

	// Snapshot previous balance to compute contributed amount on all-in
	var prevBalance int64
	if game := table.GetGame(); game != nil {
		for _, p := range game.GetPlayers() {
			if p.ID() == req.PlayerId {
				prevBalance = p.Balance()
				break
			}
		}
	}

	if err := table.MakeBet(req.PlayerId, req.Amount); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Determine the actual absolute bet the server accepted for this player
	// (it may be lower than the requested amount due to stack limits/all-in).
	var acceptedAmount int64 = req.Amount
	if game := table.GetGame(); game != nil {
		for _, p := range game.GetPlayers() {
			if p.ID() == req.PlayerId {
				acceptedAmount = p.CurrentBet()
				break
			}
		}
	}

	// Publish BET_MADE event with the accepted amount
	evt, err := s.buildGameEvent(
		pokerrpc.NotificationType_BET_MADE,
		req.TableId,
		BetMadePayload{
			PlayerID: req.PlayerId,
			Amount:   acceptedAmount,
		},
	)
	if err != nil {
		s.log.Errorf("Failed to build BET_MADE event: %v", err)
		return nil, err
	}
	s.eventProcessor.PublishEvent(evt)

	// If this action took the player all-in, emit a dedicated ALL_IN event.
	if game := table.GetGame(); game != nil {
		for _, p := range game.GetPlayers() {
			if p.ID() == req.PlayerId {
				if p.GetCurrentStateString() == poker.ALL_IN_STATE || (p.Balance() == 0 && p.CurrentBet() > 0) {
					contributed := prevBalance - p.Balance()
					if contributed < 0 {
						contributed = 0
					}
					evt, err := s.buildGameEvent(
						pokerrpc.NotificationType_PLAYER_ALL_IN,
						req.TableId,
						PlayerAllInPayload{PlayerID: req.PlayerId, Amount: contributed},
					)
					if err != nil {
						s.log.Errorf("Failed to build PLAYER_ALL_IN event: %v", err)
						return nil, err
					}
					s.eventProcessor.PublishEvent(evt)
				}
				break
			}
		}
	}

	return &pokerrpc.MakeBetResponse{
		Success: true,
		Message: "Bet placed successfully",
	}, nil
}

func (s *Server) FoldBet(ctx context.Context, req *pokerrpc.FoldBetRequest) (*pokerrpc.FoldBetResponse, error) {
	table, ok := s.getTable(req.TableId)
	if !ok {
		return nil, status.Error(codes.NotFound, "table not found")
	}
	if err := s.requireSeatedPlayer(table, req.PlayerId); err != nil {
		return nil, err
	}
	if !table.IsGameStarted() {
		return nil, status.Error(codes.FailedPrecondition, "game not started")
	}

	if err := table.HandleFold(req.PlayerId); err != nil {
		// Invalid-at-this-time actions are a client precondition issue, not server-internal.
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	// Publish typed PLAYER_FOLDED event
	evt, err := s.buildGameEvent(
		pokerrpc.NotificationType_PLAYER_FOLDED,
		req.TableId,
		PlayerFoldedPayload{PlayerID: req.PlayerId},
	)
	if err != nil {
		s.log.Errorf("Failed to build PLAYER_FOLDED event: %v", err)
		return nil, err
	}
	s.eventProcessor.PublishEvent(evt)

	return &pokerrpc.FoldBetResponse{
		Success: true,
		Message: "Folded successfully",
	}, nil
}

// Call implements the Call RPC method
func (s *Server) CallBet(ctx context.Context, req *pokerrpc.CallBetRequest) (*pokerrpc.CallBetResponse, error) {
	table, ok := s.getTable(req.TableId)
	if !ok {
		return nil, status.Error(codes.NotFound, "table not found")
	}
	if err := s.requireSeatedPlayer(table, req.PlayerId); err != nil {
		return nil, err
	}
	if !table.IsGameStarted() {
		return nil, status.Error(codes.FailedPrecondition, "game not started")
	}

	// Snapshot player's previous bet to compute actual delta contributed.
	var prevBet int64
	if game := table.GetGame(); game != nil {
		for _, p := range game.GetPlayers() {
			if p.ID() == req.PlayerId {
				prevBet = p.CurrentBet()
				break
			}
		}
	}

	if err := table.HandleCall(req.PlayerId); err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	// After handling the call, recompute player's bet and derive the actual delta
	// (important for short-stack all-ins where full call isn't possible).
	var newBet int64 = prevBet
	var newBalance int64
	if game := table.GetGame(); game != nil {
		for _, p := range game.GetPlayers() {
			if p.ID() == req.PlayerId {
				newBet = p.CurrentBet()
				newBalance = p.Balance()
				break
			}
		}
	}
	delta := newBet - prevBet
	if delta < 0 {
		delta = 0 // safety
	}

	// Publish typed CALL_MADE event
	evt, err := s.buildGameEvent(
		pokerrpc.NotificationType_CALL_MADE,
		req.TableId,
		CallMadePayload{
			PlayerID: req.PlayerId,
			Amount:   delta,
		},
	)
	if err != nil {
		s.log.Errorf("Failed to build CALL_MADE event: %v", err)
		return nil, err
	}
	s.eventProcessor.PublishEvent(evt)

	// If the call put the player all-in, emit a PLAYER_ALL_IN event with the contributed amount.
	if game := table.GetGame(); game != nil {
		for _, p := range game.GetPlayers() {
			if p.ID() != req.PlayerId {
				continue
			}
			if p.GetCurrentStateString() == poker.ALL_IN_STATE || (newBalance == 0 && p.CurrentBet() > 0) {
				evt, err := s.buildGameEvent(
					pokerrpc.NotificationType_PLAYER_ALL_IN,
					req.TableId,
					PlayerAllInPayload{PlayerID: req.PlayerId, Amount: delta},
				)
				if err != nil {
					s.log.Errorf("Failed to build PLAYER_ALL_IN event: %v", err)
					return nil, err
				}
				s.eventProcessor.PublishEvent(evt)
			}
			break
		}
	}

	return &pokerrpc.CallBetResponse{Success: true, Message: "Call successful"}, nil
}

// Check implements the Check RPC method
func (s *Server) CheckBet(ctx context.Context, req *pokerrpc.CheckBetRequest) (*pokerrpc.CheckBetResponse, error) {
	table, ok := s.getTable(req.TableId)
	if !ok {
		return nil, status.Error(codes.NotFound, "table not found")
	}
	if err := s.requireSeatedPlayer(table, req.PlayerId); err != nil {
		return nil, err
	}
	if !table.IsGameStarted() {
		return nil, status.Error(codes.FailedPrecondition, "game not started")
	}

	if err := table.HandleCheck(req.PlayerId); err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	// Publish typed CHECK_MADE event
	evt, err := s.buildGameEvent(
		pokerrpc.NotificationType_CHECK_MADE,
		req.TableId,
		CheckMadePayload{PlayerID: req.PlayerId},
	)
	if err != nil {
		s.log.Errorf("Failed to build CHECK_MADE event: %v", err)
		return nil, err
	}
	s.eventProcessor.PublishEvent(evt)

	return &pokerrpc.CheckBetResponse{Success: true, Message: "Check successful"}, nil
}

func (s *Server) GetGameState(ctx context.Context, req *pokerrpc.GetGameStateRequest) (*pokerrpc.GetGameStateResponse, error) {
	// Verify table exists
	_, ok := s.getTable(req.TableId)
	if !ok {
		return nil, status.Error(codes.NotFound, "table not found")
	}

	// Build game state using GameStateHandler (same logic as StartGameStream).
	// If the caller is authenticated (token present and valid), use the
	// session user ID as the requesting player to reveal their own hole cards.
	// Otherwise, fall back to req.PlayerId (or empty for a table-level view).
	gsh := NewGameStateHandler(s)
	tableSnapshot, err := s.collectTableSnapshot(req.TableId)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to collect table snapshot: %v", err))
	}

	requestingPlayerID := req.PlayerId
	if token := extractTokenFromMetadata(ctx); token != "" {
		if sess, ok := s.sessionForToken(token); ok {
			requestingPlayerID = sess.userID.String()
		}
	}

	gameUpdate, err := gsh.buildGameUpdateFromSnapshot(tableSnapshot, requestingPlayerID)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to build game state: %v", err))
	}

	return &pokerrpc.GetGameStateResponse{
		GameState: gameUpdate,
	}, nil
}

// convertGRPCCardToInternal converts a gRPC Card to internal Card format
func convertGRPCCardToInternal(grpcCard *pokerrpc.Card) (poker.Card, error) {
	if grpcCard == nil {
		return poker.Card{}, fmt.Errorf("card is nil")
	}

	// Convert suit string to internal Suit type
	var suit poker.Suit
	switch grpcCard.Suit {
	case "♠", "s", "S", "spades", "Spades":
		suit = poker.Spades
	case "♥", "h", "H", "hearts", "Hearts":
		suit = poker.Hearts
	case "♦", "d", "D", "diamonds", "Diamonds":
		suit = poker.Diamonds
	case "♣", "c", "C", "clubs", "Clubs":
		suit = poker.Clubs
	default:
		return poker.Card{}, fmt.Errorf("invalid suit: %s", grpcCard.Suit)
	}

	// Convert value string to internal Value type
	var value poker.Value
	switch grpcCard.Value {
	case "A", "a", "ace", "Ace":
		value = poker.Ace
	case "K", "k", "king", "King":
		value = poker.King
	case "Q", "q", "queen", "Queen":
		value = poker.Queen
	case "J", "j", "jack", "Jack":
		value = poker.Jack
	case "10", "T", "t", "ten", "Ten":
		value = poker.Ten
	case "9", "nine", "Nine":
		value = poker.Nine
	case "8", "eight", "Eight":
		value = poker.Eight
	case "7", "seven", "Seven":
		value = poker.Seven
	case "6", "six", "Six":
		value = poker.Six
	case "5", "five", "Five":
		value = poker.Five
	case "4", "four", "Four":
		value = poker.Four
	case "3", "three", "Three":
		value = poker.Three
	case "2", "two", "Two":
		value = poker.Two
	default:
		return poker.Card{}, fmt.Errorf("invalid value: %s", grpcCard.Value)
	}

	// Create the card using a helper function since fields are unexported
	return poker.NewCardFromSuitValue(suit, value), nil
}

func (s *Server) EvaluateHand(ctx context.Context, req *pokerrpc.EvaluateHandRequest) (*pokerrpc.EvaluateHandResponse, error) {
	// Convert gRPC cards to internal Card format
	cards := make([]poker.Card, len(req.Cards))
	for i, grpcCard := range req.Cards {
		card, err := convertGRPCCardToInternal(grpcCard)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid card at index %d: %v", i, err))
		}
		cards[i] = card
	}

	// We need at least 5 cards to evaluate a hand
	if len(cards) < 5 {
		return nil, status.Error(codes.InvalidArgument, "need at least 5 cards to evaluate a hand")
	}

	// For hand evaluation, we'll treat the first 2 cards as hole cards
	// and the rest as community cards (this is a simplification)
	var holeCards, communityCards []poker.Card
	if len(cards) == 5 {
		// If exactly 5 cards, evaluate them all as community cards with empty hole cards
		holeCards = []poker.Card{}
		communityCards = cards
	} else if len(cards) >= 7 {
		// Standard Texas Hold'em: 2 hole + 5 community
		holeCards = cards[:2]
		communityCards = cards[2:7]
	} else {
		// 6 cards: 2 hole + 4 community (incomplete hand)
		holeCards = cards[:2]
		communityCards = cards[2:]
	}

	// Evaluate the hand
	handValue, err := poker.EvaluateHand(holeCards, communityCards)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to evaluate hand: %v", err)
	}

	// Convert best hand back to gRPC format
	bestHandGRPC := make([]*pokerrpc.Card, len(handValue.BestHand))
	for i, card := range handValue.BestHand {
		bestHandGRPC[i] = &pokerrpc.Card{
			Suit:  card.GetSuit(),
			Value: card.GetValue(),
		}
	}

	return &pokerrpc.EvaluateHandResponse{
		Rank:        handValue.HandRank,
		Description: handValue.HandDescription,
		BestHand:    bestHandGRPC,
	}, nil
}

func (s *Server) GetLastWinners(ctx context.Context, req *pokerrpc.GetLastWinnersRequest) (*pokerrpc.GetLastWinnersResponse, error) {
	table, ok := s.getTable(req.TableId)
	if !ok {
		return nil, status.Error(codes.NotFound, "table not found")
	}
	last := table.GetLastShowdown()
	if last == nil {
		return &pokerrpc.GetLastWinnersResponse{
			Winners: []*pokerrpc.Winner{},
		}, nil
	}

	winners := make([]*pokerrpc.Winner, 0, len(last.WinnerInfo))
	// If there is a single winner, surface the total pot (pre-refund snapshot)
	// as their Winnings in this response so clients/tests can display the
	// headline pot amount. Actual chip credits already reflect refunds.
	singleWinner := len(last.WinnerInfo) == 1

	// If game hasn't started or is nil, fall back to last showdown if available.
	if !table.IsGameStarted() || table.GetGame() == nil {
		s.log.Debugf("GetLastWinners: table %s returning cached showdown: winners=%d pot=%d", req.TableId, len(last.WinnerInfo), last.TotalPot)
		for _, wi := range last.WinnerInfo {
			amt := wi.Winnings
			if singleWinner {
				amt = last.TotalPot
			}
			winners = append(winners, &pokerrpc.Winner{PlayerId: wi.PlayerId, Winnings: amt, HandRank: wi.HandRank, BestHand: wi.BestHand})
		}
		return &pokerrpc.GetLastWinnersResponse{Winners: winners}, nil
	}

	game := table.GetGame()

	s.log.Debugf("GetLastWinners: table %s game phase=%v", req.TableId, game.GetPhase())
	for _, wi := range last.WinnerInfo {
		amt := wi.Winnings
		if singleWinner {
			amt = last.TotalPot
		}
		winners = append(winners, &pokerrpc.Winner{PlayerId: wi.PlayerId, Winnings: amt, HandRank: wi.HandRank, BestHand: wi.BestHand})
	}
	return &pokerrpc.GetLastWinnersResponse{Winners: winners}, nil

}

func (s *Server) ShowCards(ctx context.Context, req *pokerrpc.ShowCardsRequest) (*pokerrpc.ShowCardsResponse, error) {
	table, ok := s.getTable(req.TableId)

	if !ok {
		return nil, status.Error(codes.NotFound, "table not found")
	}

	// Verify player is at the table
	user := table.GetUser(req.PlayerId)
	if user == nil {
		return nil, status.Error(codes.FailedPrecondition, "player not at table")
	}

	game := table.GetGame()
	if game == nil {
		return nil, status.Error(codes.FailedPrecondition, "no active game")
	}

	cards, err := game.RevealPlayerCards(req.PlayerId)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	phase := game.GetPhase()
	// Treat showdown as active once results are available, even if phase flips late.
	showdownReady := phase == pokerrpc.GamePhase_SHOWDOWN || table.GetLastShowdown() != nil
	// Broadcast card visibility notification to all players at the table with revealed cards.
	// At showdown, include cards. During hand, just notify intent.
	if showdownReady && len(cards) > 0 {
		// At showdown, cards are actually revealed
		msg := fmt.Sprintf("%s is showing their cards", req.PlayerId)
		s.broadcastNotificationToTable(req.TableId, &pokerrpc.Notification{
			Type:     pokerrpc.NotificationType_CARDS_SHOWN,
			PlayerId: req.PlayerId,
			TableId:  req.TableId,
			Cards:    cards,
			Message:  msg,
		})
		// Publish game state update so all players receive the updated state with revealed cards
		if tableSnapshot, err := s.collectTableSnapshot(req.TableId); err == nil && tableSnapshot != nil {
			s.publishTableSnapshotEvent(req.TableId, tableSnapshot)
		}
		return &pokerrpc.ShowCardsResponse{
			Success: true,
			Message: "Cards shown to other players",
		}, nil
	}

	// During hand, just notify intent (no cards shown yet)
	msg := fmt.Sprintf("%s will show their cards at showdown", req.PlayerId)
	s.broadcastNotificationToTable(req.TableId, &pokerrpc.Notification{
		Type:     pokerrpc.NotificationType_CARDS_SHOWN,
		PlayerId: req.PlayerId,
		TableId:  req.TableId,
		Cards:    nil, // No cards during hand
		Message:  msg,
	})
	// Publish snapshot so the toggled reveal intent is reflected in the UI.
	if tableSnapshot, err := s.collectTableSnapshot(req.TableId); err == nil && tableSnapshot != nil {
		s.publishTableSnapshotEvent(req.TableId, tableSnapshot)
	}

	return &pokerrpc.ShowCardsResponse{
		Success: true,
		Message: "Cards shown to other players",
	}, nil
}

func (s *Server) HideCards(ctx context.Context, req *pokerrpc.HideCardsRequest) (*pokerrpc.HideCardsResponse, error) {
	table, ok := s.getTable(req.TableId)

	if !ok {
		return nil, status.Error(codes.NotFound, "table not found")
	}

	// Verify player is at the table
	user := table.GetUser(req.PlayerId)
	if user == nil {
		return nil, status.Error(codes.FailedPrecondition, "player not at table")
	}

	game := table.GetGame()
	if game == nil {
		return nil, status.Error(codes.FailedPrecondition, "no active game")
	}
	if err := game.HidePlayerCards(req.PlayerId); err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	phase := game.GetPhase()
	showdownReady := phase == pokerrpc.GamePhase_SHOWDOWN || table.GetLastShowdown() != nil
	// Broadcast card visibility notification to all players at the table
	if showdownReady {
		// At showdown, cards are actually hidden
		s.broadcastNotificationToTable(req.TableId, &pokerrpc.Notification{
			Type:     pokerrpc.NotificationType_CARDS_HIDDEN,
			PlayerId: req.PlayerId,
			TableId:  req.TableId,
			Message:  fmt.Sprintf("%s is hiding their cards", req.PlayerId),
		})
		// Publish snapshot so the toggled reveal intent is reflected in the UI.
		if tableSnapshot, err := s.collectTableSnapshot(req.TableId); err == nil && tableSnapshot != nil {
			s.publishTableSnapshotEvent(req.TableId, tableSnapshot)
		}
		return &pokerrpc.HideCardsResponse{
			Success: true,
			Message: "Cards hidden from other players",
		}, nil
	}

	// During hand, just notify intent change
	s.broadcastNotificationToTable(req.TableId, &pokerrpc.Notification{
		Type:     pokerrpc.NotificationType_CARDS_HIDDEN,
		PlayerId: req.PlayerId,
		TableId:  req.TableId,
		Message:  fmt.Sprintf("%s will not show their cards at showdown", req.PlayerId),
	})
	// Publish snapshot so the toggled reveal intent is reflected in the UI.
	if tableSnapshot, err := s.collectTableSnapshot(req.TableId); err == nil && tableSnapshot != nil {
		s.publishTableSnapshotEvent(req.TableId, tableSnapshot)
	}

	return &pokerrpc.HideCardsResponse{
		Success: true,
		Message: "Cards hidden from other players",
	}, nil
}
