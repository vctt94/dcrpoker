package server

import (
	"context"
	"fmt"

	"github.com/vctt94/pokerbisonrelay/pkg/poker"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func (s *Server) StartGameStream(req *pokerrpc.StartGameStreamRequest, stream pokerrpc.PokerService_StartGameStreamServer) error {
	tableID, playerID := req.TableId, req.PlayerId

	// Get or create the table bucket (single step).
	bAny, _ := s.gameStreams.LoadOrStore(tableID, &bucket{})
	b := bAny.(*bucket)

	// Register player stream. If a stream already exists for this player,
	// replace it with the newest one without incrementing the count. This
	// ensures the most recent attachment (e.g., Flutter UI) receives updates
	// and avoids starving newer clients when multiple components attach.
	if _, loaded := b.streams.Load(playerID); loaded {
		b.streams.Store(playerID, stream)
	} else {
		b.streams.Store(playerID, stream)
		b.count.Add(1)
	}

	// Unregister on exit only if this goroutine still owns the stored stream.
	// This prevents a replaced (older) stream from deleting the newer mapping.
	defer func() {
		if v, present := b.streams.Load(playerID); present && v == stream {
			b.streams.Delete(playerID)
			if b.count.Add(-1) == 0 {
				// Remove this bucket iff it's still the same one we used.
				s.gameStreams.CompareAndDelete(tableID, b)
			}
		}
	}()

	// Send initial game state.
	gs, err := s.buildGameState(tableID, playerID)
	if err != nil {
		return err
	}
	if err := stream.Send(gs); err != nil {
		return err
	}

	<-stream.Context().Done()
	return nil
}

func (s *Server) MakeBet(ctx context.Context, req *pokerrpc.MakeBetRequest) (*pokerrpc.MakeBetResponse, error) {
	table, ok := s.getTable(req.TableId)
	if !ok {
		return nil, status.Error(codes.NotFound, "table not found")
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
	if evt, err := s.buildGameEvent(
		pokerrpc.NotificationType_BET_MADE,
		req.TableId,
		BetMadePayload{
			PlayerID: req.PlayerId,
			Amount:   acceptedAmount,
		},
	); err == nil {
		s.eventProcessor.PublishEvent(evt)
	} else {
		s.log.Errorf("Failed to build BET_MADE event: %v", err)
	}

	// If this action took the player all-in, emit a dedicated ALL_IN event.
	if game := table.GetGame(); game != nil {
		for _, p := range game.GetPlayers() {
			if p.ID() == req.PlayerId {
				if p.GetCurrentStateString() == "ALL_IN" || (p.Balance() == 0 && p.CurrentBet() > 0) {
					contributed := prevBalance - p.Balance()
					if contributed < 0 {
						contributed = 0
					}
					if evt, err := s.buildGameEvent(
						pokerrpc.NotificationType_PLAYER_ALL_IN,
						req.TableId,
						PlayerAllInPayload{PlayerID: req.PlayerId, Amount: contributed},
					); err == nil {
						s.eventProcessor.PublishEvent(evt)
					} else {
						s.log.Errorf("Failed to build PLAYER_ALL_IN event: %v", err)
					}
				}
				break
			}
		}
	}

	// DCR account balance is independent of chip bets; this just returns the wallet balance.
	balance, err := s.db.GetPlayerBalance(ctx, req.PlayerId)
	if err != nil {
		return nil, err
	}

	return &pokerrpc.MakeBetResponse{
		Success:    true,
		Message:    "Bet placed successfully",
		NewBalance: balance,
	}, nil
}

func (s *Server) FoldBet(ctx context.Context, req *pokerrpc.FoldBetRequest) (*pokerrpc.FoldBetResponse, error) {
	table, ok := s.getTable(req.TableId)
	if !ok {
		return nil, status.Error(codes.NotFound, "table not found")
	}
	if !table.IsGameStarted() {
		return nil, status.Error(codes.FailedPrecondition, "game not started")
	}

	if err := table.HandleFold(req.PlayerId); err != nil {
		// Invalid-at-this-time actions are a client precondition issue, not server-internal.
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	// Publish typed PLAYER_FOLDED event
	if evt, err := s.buildGameEvent(
		pokerrpc.NotificationType_PLAYER_FOLDED,
		req.TableId,
		PlayerFoldedPayload{PlayerID: req.PlayerId},
	); err == nil {
		s.eventProcessor.PublishEvent(evt)
	} else {
		s.log.Errorf("Failed to build PLAYER_FOLDED event: %v", err)
	}

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
	if evt, err := s.buildGameEvent(
		pokerrpc.NotificationType_CALL_MADE,
		req.TableId,
		CallMadePayload{
			PlayerID: req.PlayerId,
			Amount:   delta,
		},
	); err == nil {
		s.eventProcessor.PublishEvent(evt)
	} else {
		s.log.Errorf("Failed to build CALL_MADE event: %v", err)
	}

	// If the call put the player all-in, emit a PLAYER_ALL_IN event with the contributed amount.
	if game := table.GetGame(); game != nil {
		for _, p := range game.GetPlayers() {
			if p.ID() == req.PlayerId {
				if p.GetCurrentStateString() == "ALL_IN" || (newBalance == 0 && p.CurrentBet() > 0) {
					if evt, err := s.buildGameEvent(
						pokerrpc.NotificationType_PLAYER_ALL_IN,
						req.TableId,
						PlayerAllInPayload{PlayerID: req.PlayerId, Amount: delta},
					); err == nil {
						s.eventProcessor.PublishEvent(evt)
					} else {
						s.log.Errorf("Failed to build PLAYER_ALL_IN event: %v", err)
					}
				}
				break
			}
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
	if !table.IsGameStarted() {
		return nil, status.Error(codes.FailedPrecondition, "game not started")
	}

	if err := table.HandleCheck(req.PlayerId); err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	// Publish typed CHECK_MADE event
	if evt, err := s.buildGameEvent(
		pokerrpc.NotificationType_CHECK_MADE,
		req.TableId,
		CheckMadePayload{PlayerID: req.PlayerId},
	); err == nil {
		s.eventProcessor.PublishEvent(evt)
	} else {
		s.log.Errorf("Failed to build CHECK_MADE event: %v", err)
	}

	return &pokerrpc.CheckBetResponse{Success: true, Message: "Check successful"}, nil
}

// buildPlayerForUpdate creates a Player proto message with appropriate card visibility
func (s *Server) buildPlayerForUpdate(p *poker.Player, requestingPlayerID string, game *poker.Game) *pokerrpc.Player {
	stateStr := p.GetCurrentStateString()
	grpcPlayer := p.Marshal() // snapshot with turn/dealer/blinds flags
	player := &pokerrpc.Player{
		Id:      p.ID(),
		Balance: p.Balance(),
		IsReady: p.IsReady(),
		Folded:  stateStr == "FOLDED",
		// Surface all-in so UIs can render an explicit badge without inference.
		IsAllIn:      stateStr == "ALL_IN",
		CurrentBet:   p.CurrentBet(),
		PlayerState:  p.GetTablePresenceState(),
		IsDealer:     grpcPlayer.IsDealer,
		IsSmallBlind: grpcPlayer.IsSmallBlind,
		IsBigBlind:   grpcPlayer.IsBigBlind,
		IsTurn:       grpcPlayer.IsTurn,
	}

	// Heads-up sanity: dealer must also be SB.
	if game != nil && len(game.GetPlayers()) == 2 && grpcPlayer.IsDealer && !grpcPlayer.IsSmallBlind {
		s.log.Warnf("INCONSISTENT STATE: Player %s is dealer but not SB in heads-up! phase=%v", p.ID(), game.GetPhase())
	}

	// No game -> nothing else to surface.
	if game == nil {
		return player
	}

	hand := game.GetCurrentHand()
	if hand == nil {
		// Still return base player info; cards come only from an active hand.
		return player
	}

	// Decide visibility once, then fill if any cards are visible.
	var cards []poker.Card
	isShowdown := game.GetPhase() == pokerrpc.GamePhase_SHOWDOWN
	isSelf := p.ID() == requestingPlayerID

	switch {
	case isSelf:
		// Always show own cards as soon as they exist.
		cards = hand.GetPlayerCards(p.ID(), requestingPlayerID)
		if len(cards) > 0 {
			s.log.Debugf("DEBUG: Showing %d cards for player %s (own cards, phase=%v, state=%s)",
				len(cards), p.ID(), game.GetPhase(), stateStr)
		}
	case isShowdown:
		// Show others' cards only at showdown (visibility enforced by GetPlayerCards).
		cards = hand.GetPlayerCards(p.ID(), requestingPlayerID)
	}

	if n := len(cards); n > 0 {
		player.Hand = make([]*pokerrpc.Card, n)
		for i, c := range cards {
			player.Hand[i] = &pokerrpc.Card{Suit: c.GetSuit(), Value: c.GetValue()}
		}
	}

	// Hand description is surfaced only at showdown.
	if isShowdown && p.HandDescription() != "" {
		player.HandDescription = p.HandDescription()
	}

	return player
}

// buildPlayers creates a slice of Player proto messages with appropriate card visibility
func (s *Server) buildPlayers(tablePlayers []*poker.Player, game *poker.Game, requestingPlayerID string) []*pokerrpc.Player {
	players := make([]*pokerrpc.Player, 0, len(tablePlayers))
	for _, p := range tablePlayers {
		player := s.buildPlayerForUpdate(p, requestingPlayerID, game)
		players = append(players, player)
	}
	return players
}

// buildGameStateForPlayer creates a GameUpdate with all the necessary data for a specific player
func (s *Server) buildGameStateForPlayer(table *poker.Table, game *poker.Game, requestingPlayerID string) *pokerrpc.GameUpdate {
	// Build players list from users and game players
	var players []*pokerrpc.Player
	if game != nil {
		players = s.buildPlayers(game.GetPlayers(), game, requestingPlayerID)
	} else {
		// If no game, build from table users
		users := table.GetUsers()
		players = make([]*pokerrpc.Player, 0, len(users))
		for _, user := range users {
			players = append(players, &pokerrpc.Player{
				Id:      user.ID,
				Balance: 0, // No poker chips when no game - Balance field should be poker chips, not DCR
				IsReady: user.IsReady,

				Hand:        make([]*pokerrpc.Card, 0), // Empty hand when no game
				PlayerState: pokerrpc.PlayerState_PLAYER_STATE_AT_TABLE,
			})
		}
	}

	// Build community cards slice
	communityCards := make([]*pokerrpc.Card, 0)
	var pot int64 = 0
	if game != nil {
		pot = game.GetPot()
		for _, c := range game.GetCommunityCards() {
			communityCards = append(communityCards, &pokerrpc.Card{
				Suit:  c.GetSuit(),
				Value: c.GetValue(),
			})
		}
	}

	var currentPlayerID string
	if table.IsGameStarted() && game != nil {
		// Only expose current player when action is valid (not during setup or showdown)
		phase := game.GetPhase()
		if phase != pokerrpc.GamePhase_NEW_HAND_DEALING && phase != pokerrpc.GamePhase_SHOWDOWN {
			currentPlayerID = table.GetCurrentPlayerID()
		}
	}

	// Note: Do not override per-player IsTurn here; the Player FSM is the
	// single authority for that flag. UIs should rely on CurrentPlayer for
	// highlighting to avoid transient races between EndTurn/StartTurn events.

	// Authoritative timebank fields
	var tbSec int32
	var deadlineMs int64
	cfg := table.GetConfig()
	if cfg.TimeBank > 0 {
		tbSec = int32(cfg.TimeBank.Seconds())
		// Compute deadline for the current player if applicable, using a snapshot
		if currentPlayerID != "" {
			snap := game.GetStateSnapshot()
			for _, ps := range snap.Players {
				if ps.ID == currentPlayerID {
					dl := ps.LastAction.Add(cfg.TimeBank)
					deadlineMs = dl.UnixMilli()
					break
				}
			}
		}
	}

	return &pokerrpc.GameUpdate{
		TableId:            table.GetConfig().ID,
		Phase:              table.GetGamePhase(),
		PhaseName:          table.GetGamePhase().String(),
		Players:            players,
		CommunityCards:     communityCards,
		Pot:                pot,
		CurrentBet:         table.GetCurrentBet(),
		CurrentPlayer:      currentPlayerID,
		GameStarted:        table.IsGameStarted(),
		PlayersRequired:    int32(table.GetMinPlayers()),
		PlayersJoined:      int32(len(table.GetUsers())),
		TimeBankSeconds:    tbSec,
		TurnDeadlineUnixMs: deadlineMs,
	}
}

func (s *Server) GetGameState(ctx context.Context, req *pokerrpc.GetGameStateRequest) (*pokerrpc.GetGameStateResponse, error) {
	// Acquire server lock only to fetch table pointer, then release before
	// calling into table methods to avoid lock coupling (Server → Table).
	table, ok := s.getTable(req.TableId)
	if !ok {
		return nil, status.Error(codes.NotFound, "table not found")
	}

	// Extract requesting player ID from context metadata
	requestingPlayerID := ""
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if playerIDs := md.Get("player-id"); len(playerIDs) > 0 {
			requestingPlayerID = playerIDs[0]
		}
	}

	game := table.GetGame()

	return &pokerrpc.GetGameStateResponse{
		GameState: s.buildGameStateForPlayer(table, game, requestingPlayerID),
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

	// Broadcast card visibility notification to all players at the table
	s.broadcastNotificationToTable(req.TableId, &pokerrpc.Notification{
		Type:     pokerrpc.NotificationType_CARDS_SHOWN,
		PlayerId: req.PlayerId,
		TableId:  req.TableId,
		Message:  fmt.Sprintf("%s is showing their cards", req.PlayerId),
	})

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

	// Broadcast card visibility notification to all players at the table
	s.broadcastNotificationToTable(req.TableId, &pokerrpc.Notification{
		Type:     pokerrpc.NotificationType_CARDS_HIDDEN,
		PlayerId: req.PlayerId,
		TableId:  req.TableId,
		Message:  fmt.Sprintf("%s is hiding their cards", req.PlayerId),
	})

	return &pokerrpc.HideCardsResponse{
		Success: true,
		Message: "Cards hidden from other players",
	}, nil
}
