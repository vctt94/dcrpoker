package server

import "github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"

// Each event carries exactly one payload implementing this interface.
type EventPayload interface {
	Kind() pokerrpc.NotificationType
}

// ---------- Generic/game-wide payloads ----------

// ShowdownPayload wraps pokerrpc.Showdown to implement EventPayload interface
type ShowdownPayload struct {
	*pokerrpc.Showdown
}

func (ShowdownPayload) Kind() pokerrpc.NotificationType {
	return pokerrpc.NotificationType_SHOWDOWN_RESULT
}

type GameStartedPayload struct {
	PlayerIDs []string // optional; handlers don't require, but useful
}

func (GameStartedPayload) Kind() pokerrpc.NotificationType {
	return pokerrpc.NotificationType_GAME_STARTED
}

type NewHandStartedPayload struct {
	HandID    uint64 // optional
	DealerPos int    // optional
}

func (NewHandStartedPayload) Kind() pokerrpc.NotificationType {
	return pokerrpc.NotificationType_NEW_HAND_STARTED
}

// ---------- Action payloads ----------

type BetMadePayload struct {
	PlayerID string
	Amount   int64
}

func (BetMadePayload) Kind() pokerrpc.NotificationType { return pokerrpc.NotificationType_BET_MADE }

type CallMadePayload struct {
	PlayerID string
	Amount   int64 // amount that was called to (or put in)
}

func (CallMadePayload) Kind() pokerrpc.NotificationType { return pokerrpc.NotificationType_CALL_MADE }

type CheckMadePayload struct {
	PlayerID string
}

func (CheckMadePayload) Kind() pokerrpc.NotificationType { return pokerrpc.NotificationType_CHECK_MADE }

type PlayerFoldedPayload struct {
	PlayerID string
}

func (PlayerFoldedPayload) Kind() pokerrpc.NotificationType {
	return pokerrpc.NotificationType_PLAYER_FOLDED
}

type PlayerReadyPayload struct {
	PlayerID string // see note below; usually string
}

func (PlayerReadyPayload) Kind() pokerrpc.NotificationType {
	return pokerrpc.NotificationType_PLAYER_READY
}

// If "ready" is binary+who, prefer this simpler one instead of boolOrString:
type PlayerMarkedReadyPayload struct {
	PlayerID string
	Ready    bool
}

func (PlayerMarkedReadyPayload) Kind() pokerrpc.NotificationType {
	return pokerrpc.NotificationType_PLAYER_READY
}

type PlayerJoinedPayload struct {
	PlayerID string
}

func (PlayerJoinedPayload) Kind() pokerrpc.NotificationType {
	return pokerrpc.NotificationType_PLAYER_JOINED
}

type PlayerLeftPayload struct {
	PlayerID string
}

func (PlayerLeftPayload) Kind() pokerrpc.NotificationType {
	return pokerrpc.NotificationType_PLAYER_LEFT
}

type PlayerLostPayload struct {
	PlayerID string
}

func (PlayerLostPayload) Kind() pokerrpc.NotificationType {
	return pokerrpc.NotificationType_PLAYER_LOST
}

// AutoShowCardsPayload announces that a player's hole cards were auto-revealed.
type AutoShowCardsPayload struct {
	PlayerID string
	Cards    []*pokerrpc.Card
}

func (AutoShowCardsPayload) Kind() pokerrpc.NotificationType {
	return pokerrpc.NotificationType_CARDS_SHOWN
}

// PlayerAllInPayload announces that a player has gone all-in and the amount
// they just committed in the action that caused it.
type PlayerAllInPayload struct {
	PlayerID string
	Amount   int64
}

func (PlayerAllInPayload) Kind() pokerrpc.NotificationType {
	return pokerrpc.NotificationType_PLAYER_ALL_IN
}

// GameEndedPayload carries information about the game winner and settlement.
type GameEndedPayload struct {
	WinnerID   string // Player ID of the game winner
	WinnerSeat int32  // Seat index of the winner (for settlement branch)
	MatchID    string // table_id|session_id for referee settlement
	TotalPot   int64  // Total pot won (in chips)
	Entrants   int    // Number of players who started the tournament
}

func (GameEndedPayload) Kind() pokerrpc.NotificationType {
	return pokerrpc.NotificationType_GAME_ENDED
}

// MessagePayload carries a generic string message for table-wide notifications.
type MessagePayload struct {
	Message string
}

func (MessagePayload) Kind() pokerrpc.NotificationType {
	return pokerrpc.NotificationType_MESSAGE
}
