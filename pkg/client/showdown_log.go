package client

import (
	"fmt"
	"strings"

	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
)

func buildShowdownLogLines(ntfn *pokerrpc.Notification, fallback *pokerrpc.GameUpdate, selfID string) []string {
	if ntfn == nil {
		return nil
	}
	if isWatchingTable(fallback, selfID) {
		return nil
	}

	showdown := ntfn.GetShowdown()
	if showdown == nil {
		board := "-"
		if fallback != nil {
			board = formatCardList(fallback.GetCommunityCards())
		}
		if winners := formatWinnerSummaries(ntfn.GetWinners(), playerNamesFromGameUpdate(fallback)); winners != "" {
			return []string{
				fmt.Sprintf("showdown table=%s board=%s winners=%s", fallbackLabel(ntfn.GetTableId(), "-"), board, winners),
			}
		}
		return nil
	}

	playerNames := playerNamesFromGameUpdate(fallback)
	if playerNames == nil {
		playerNames = make(map[string]string, len(showdown.GetPlayers()))
	}
	for _, player := range showdown.GetPlayers() {
		if player == nil || player.GetPlayerId() == "" {
			continue
		}
		playerNames[player.GetPlayerId()] = displayNameOrID(
			firstNonEmpty(player.GetName(), playerNames[player.GetPlayerId()]),
			player.GetPlayerId(),
		)
	}

	board := showdown.GetBoard()
	if len(board) == 0 && fallback != nil {
		board = fallback.GetCommunityCards()
	}

	lines := []string{
		fmt.Sprintf(
			"showdown table=%s hand=%s round=%d pot=%d board=%s",
			fallbackLabel(ntfn.GetTableId(), "-"),
			fallbackLabel(showdown.GetHandId(), "-"),
			showdown.GetRound(),
			showdown.GetPot(),
			formatCardList(board),
		),
	}

	if winners := formatWinnerSummaries(showdown.GetWinners(), playerNames); winners != "" {
		lines = append(lines, "showdown winners="+winners)
	} else if winners := formatWinnerSummaries(ntfn.GetWinners(), playerNames); winners != "" {
		lines = append(lines, "showdown winners="+winners)
	}

	if hands := formatShowdownPlayers(showdown.GetPlayers(), fallback, selfID); hands != "" {
		lines = append(lines, "showdown hands="+hands)
	}

	return lines
}

func formatWinnerSummaries(winners []*pokerrpc.Winner, playerNames map[string]string) string {
	if len(winners) == 0 {
		return ""
	}

	parts := make([]string, 0, len(winners))
	for _, winner := range winners {
		if winner == nil || winner.GetPlayerId() == "" {
			continue
		}

		label := lookupPlayerName(playerNames, winner.GetPlayerId())
		rank := handRankLabel(winner.GetHandRank())
		if rank == "" {
			parts = append(parts, fmt.Sprintf("%s(+%d)", label, winner.GetWinnings()))
			continue
		}
		parts = append(parts, fmt.Sprintf("%s(+%d, %s)", label, winner.GetWinnings(), rank))
	}

	return strings.Join(parts, ", ")
}

func formatShowdownPlayers(players []*pokerrpc.ShowdownPlayer, fallback *pokerrpc.GameUpdate, selfID string) string {
	if len(players) == 0 {
		return ""
	}

	fallbackPlayers := playersByID(fallback)
	parts := make([]string, 0, len(players))
	for _, player := range players {
		if player == nil || player.GetPlayerId() == "" {
			continue
		}

		fallbackPlayer := fallbackPlayers[player.GetPlayerId()]
		fallbackName := ""
		if fallbackPlayer != nil {
			fallbackName = fallbackPlayer.GetName()
		}
		label := displayNameOrID(firstNonEmpty(player.GetName(), fallbackName), player.GetPlayerId())
		if player.GetPlayerId() == selfID {
			label = "*" + label
		}

		holeCards := player.GetHoleCards()
		if len(holeCards) == 0 && fallbackPlayer != nil {
			holeCards = fallbackPlayer.GetHand()
		}

		part := fmt.Sprintf("%s[%s", label, formatCardList(holeCards))
		if state := playerStateLabel(player.GetFinalState()); state != "" {
			part += "; " + state
		}
		part += "]"
		parts = append(parts, part)
	}

	return strings.Join(parts, ", ")
}

func formatCardList(cards []*pokerrpc.Card) string {
	if len(cards) == 0 {
		return "-"
	}

	parts := make([]string, 0, len(cards))
	for _, card := range cards {
		parts = append(parts, formatCard(card))
	}
	return strings.Join(parts, " ")
}

func formatCard(card *pokerrpc.Card) string {
	if card == nil {
		return "??"
	}

	value := strings.ToUpper(strings.TrimSpace(card.GetValue()))
	switch value {
	case "":
		value = "?"
	case "10":
		value = "T"
	}

	suit := strings.ToLower(strings.TrimSpace(card.GetSuit()))
	switch suit {
	case "spades", "spade", "s", "♠":
		suit = "s"
	case "hearts", "heart", "h", "♥":
		suit = "h"
	case "diamonds", "diamond", "d", "♦":
		suit = "d"
	case "clubs", "club", "c", "♣":
		suit = "c"
	case "":
		suit = "?"
	default:
		suit = strings.TrimSpace(card.GetSuit())
		if suit == "" {
			suit = "?"
		}
	}

	return value + suit
}

func handRankLabel(rank pokerrpc.HandRank) string {
	switch rank {
	case pokerrpc.HandRank_HIGH_CARD:
		return "High Card"
	case pokerrpc.HandRank_PAIR:
		return "Pair"
	case pokerrpc.HandRank_TWO_PAIR:
		return "Two Pair"
	case pokerrpc.HandRank_THREE_OF_A_KIND:
		return "Trips"
	case pokerrpc.HandRank_STRAIGHT:
		return "Straight"
	case pokerrpc.HandRank_FLUSH:
		return "Flush"
	case pokerrpc.HandRank_FULL_HOUSE:
		return "Full House"
	case pokerrpc.HandRank_FOUR_OF_A_KIND:
		return "Quads"
	case pokerrpc.HandRank_STRAIGHT_FLUSH:
		return "Straight Flush"
	case pokerrpc.HandRank_ROYAL_FLUSH:
		return "Royal Flush"
	default:
		return ""
	}
}

func playerStateLabel(state pokerrpc.PlayerState) string {
	switch state {
	case pokerrpc.PlayerState_PLAYER_STATE_FOLDED:
		return "folded"
	case pokerrpc.PlayerState_PLAYER_STATE_ALL_IN:
		return "all-in"
	case pokerrpc.PlayerState_PLAYER_STATE_IN_GAME:
		return "in-game"
	case pokerrpc.PlayerState_PLAYER_STATE_AT_TABLE:
		return "at-table"
	default:
		return ""
	}
}

func lookupPlayerName(playerNames map[string]string, playerID string) string {
	if len(playerNames) > 0 {
		if name := strings.TrimSpace(playerNames[playerID]); name != "" {
			return name
		}
	}
	return displayNameOrID("", playerID)
}

func displayNameOrID(name, playerID string) string {
	trimmedName := strings.TrimSpace(name)
	if trimmedName != "" {
		return trimmedName
	}
	return fallbackLabel(playerID, "-")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func playerNamesFromGameUpdate(update *pokerrpc.GameUpdate) map[string]string {
	players := playersByID(update)
	if len(players) == 0 {
		return nil
	}

	names := make(map[string]string, len(players))
	for playerID, player := range players {
		names[playerID] = displayNameOrID(player.GetName(), playerID)
	}
	return names
}

func playersByID(update *pokerrpc.GameUpdate) map[string]*pokerrpc.Player {
	if update == nil || len(update.GetPlayers()) == 0 {
		return nil
	}

	players := make(map[string]*pokerrpc.Player, len(update.GetPlayers()))
	for _, player := range update.GetPlayers() {
		if player == nil || player.GetId() == "" {
			continue
		}
		players[player.GetId()] = player
	}
	return players
}

func isWatchingTable(update *pokerrpc.GameUpdate, selfID string) bool {
	if update == nil || strings.TrimSpace(selfID) == "" {
		return false
	}

	for _, player := range update.GetPlayers() {
		if player != nil && player.GetId() == selfID {
			return false
		}
	}

	return len(update.GetPlayers()) > 0
}

func fallbackLabel(value, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed != "" {
		return trimmed
	}
	return fallback
}
