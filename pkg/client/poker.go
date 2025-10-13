package client

import (
	"context"
	"fmt"

	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
)

// ShowCards notifies other players that this player is showing their cards
func (pc *PokerClient) ShowCards(ctx context.Context) error {
	tableID := pc.GetCurrentTableID()

	if tableID == "" {
		return fmt.Errorf("not currently in a table")
	}

	resp, err := pc.PokerService.ShowCards(ctx, &pokerrpc.ShowCardsRequest{
		PlayerId: pc.ID.String(),
		TableId:  tableID,
	})
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("failed to show cards: %s", resp.Message)
	}

	return nil
}

// HideCards notifies other players that this player is hiding their cards
func (pc *PokerClient) HideCards(ctx context.Context) error {
	tableID := pc.GetCurrentTableID()

	if tableID == "" {
		return fmt.Errorf("not currently in a table")
	}

	resp, err := pc.PokerService.HideCards(ctx, &pokerrpc.HideCardsRequest{
		PlayerId: pc.ID.String(),
		TableId:  tableID,
	})
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("failed to hide cards: %s", resp.Message)
	}

	return nil
}

// Fold folds the current hand
func (pc *PokerClient) Fold(ctx context.Context) error {
	currentTableID := pc.GetCurrentTableID()
	if currentTableID == "" {
		return fmt.Errorf("not at any table")
	}

	_, err := pc.PokerService.FoldBet(ctx, &pokerrpc.FoldBetRequest{
		PlayerId: pc.ID.String(),
		TableId:  currentTableID,
	})
	return err
}

// Check checks (bet 0 when no one has bet)
func (pc *PokerClient) Check(ctx context.Context) error {
	currentTableID := pc.GetCurrentTableID()
	if currentTableID == "" {
		return fmt.Errorf("not at any table")
	}

	_, err := pc.PokerService.CheckBet(ctx, &pokerrpc.CheckBetRequest{
		PlayerId: pc.ID.String(),
		TableId:  currentTableID,
	})
	return err
}

// Call calls the current bet (matches the current bet amount)
func (pc *PokerClient) Call(ctx context.Context, currentBet int64) error {
	currentTableID := pc.GetCurrentTableID()
	if currentTableID == "" {
		return fmt.Errorf("not at any table")
	}

	// Use dedicated Call RPC to avoid race with fetching current bet separately
	_, err := pc.PokerService.CallBet(ctx, &pokerrpc.CallBetRequest{
		PlayerId: pc.ID.String(),
		TableId:  currentTableID,
	})
	return err
}

// Raise raises the bet to the specified amount
func (pc *PokerClient) Raise(ctx context.Context, amount int64) error {
	currentTableID := pc.GetCurrentTableID()
	if currentTableID == "" {
		return fmt.Errorf("not at any table")
	}

	_, err := pc.PokerService.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: pc.ID.String(),
		TableId:  currentTableID,
		Amount:   amount,
	})
	return err
}

// Bet makes a bet of the specified amount
func (pc *PokerClient) Bet(ctx context.Context, amount int64) error {
	currentTableID := pc.GetCurrentTableID()
	if currentTableID == "" {
		return fmt.Errorf("not at any table")
	}

	_, err := pc.PokerService.MakeBet(ctx, &pokerrpc.MakeBetRequest{
		PlayerId: pc.ID.String(),
		TableId:  currentTableID,
		Amount:   amount,
	})
	return err
}
