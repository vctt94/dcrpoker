package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
)

// Renderer handles all rendering of UI screens and game elements
type Renderer struct {
	ui *PokerUI
}

// RenderMainMenu renders the main menu screen
func (r *Renderer) RenderMainMenu() string {
	var s string
	s += TitleStyle.Render("Poker Client - Main Menu") + "\n"
	s += fmt.Sprintf("Client ID: %s\n", r.ui.clientID)

	// Display cached balance in DCR
	if r.ui.balance > 0 {
		s += fmt.Sprintf("Account Balance: %.8f DCR\n", float64(r.ui.balance)/1e8)
	} else {
		s += "Account Balance: (loading...)\n"
	}

	// Show current table info if player is at a table
	currentTableID := r.ui.pc.GetCurrentTableID()
	if currentTableID != "" {
		s += fmt.Sprintf("Current Table: %s (Phase: %s)\n", currentTableID, r.ui.gamePhase.String())
	}
	s += "\n"

	options := r.ui.getMainMenuOptions()
	for i, option := range options {
		if i == r.ui.selectedItem {
			s += FocusedStyle.Render(fmt.Sprintf("> %s", option)) + "\n"
		} else {
			s += BlurredStyle.Render(fmt.Sprintf("  %s", option)) + "\n"
		}
	}

	return s
}

// RenderTableList renders the table list screen with compact styling to show more tables
func (r *Renderer) RenderTableList() string {
	var s string
	s += TitleStyle.Render("Available Tables") + "\n"

	if len(r.ui.tables) == 0 {
		s += BlurredStyle.Render("No tables available.") + "\n"
	} else {
		// Add header for better organization
		s += TitleStyle.Render("SELECT A TABLE TO JOIN") + "\n"

		for i, table := range r.ui.tables {
			isSelected := i == r.ui.selectedTable

			// Create more compact table info
			tableID := table.Id
			if len(tableID) > 20 {
				tableID = tableID[:17] + "..."
			}

			// Add status indicators for better information
			var status string
			if table.GameStarted {
				status = "PLAYING"
			} else if table.AllPlayersReady {
				status = "STARTING"
			} else if table.CurrentPlayers >= table.MinPlayers {
				status = "WAITING"
			} else {
				status = "OPEN"
			}

			// Compact single-line format with enhanced information
			tableInfo := fmt.Sprintf("%s | %s | Players: %d/%d | Blinds: %d/%d",
				status,
				tableID,
				table.CurrentPlayers,
				table.MaxPlayers,
				table.SmallBlind,
				table.BigBlind)

			// Add selection indicator and styling
			if isSelected {
				s += FocusedStyle.Render("> "+tableInfo) + "\n"
			} else {
				s += BlurredStyle.Render("  "+tableInfo) + "\n"
			}
		}

		// Add pagination info if there are many tables
		if len(r.ui.tables) > 10 {
			s += "\n" + BlurredStyle.Render(fmt.Sprintf("Showing %d tables (use ↑↓ to scroll)", len(r.ui.tables))) + "\n"
		}
	}

	s += "\n" + HelpStyle.Render("Press Enter to join selected table, 'r' to refresh, or 'q' to go back")
	return s
}

// RenderCreateTable renders the create table form screen
func (r *Renderer) RenderCreateTable() string {
	var s string
	s += TitleStyle.Render("Create New Table") + "\n"

	fields := []struct {
		label string
		value string
	}{
		{"Small Blind", r.ui.smallBlind},
		{"Big Blind", r.ui.bigBlind},
		{"Required Players", r.ui.requiredPlayers},
		{"Buy In", r.ui.buyIn},
		{"Min Balance", r.ui.minBalance},
		{"Starting Chips", r.ui.startingChips},
		{"Auto Advance (ms)", r.ui.autoAdvanceMs},
	}

	for i, field := range fields {
		style := BlurredStyle
		if i == r.ui.selectedFormField {
			style = FocusedStyle
		}
		s += style.Render(fmt.Sprintf("%s %s: %s",
			func() string {
				if i == r.ui.selectedFormField {
					return ">"
				}
				return " "
			}(),
			field.label,
			field.value,
		)) + "\n"
	}
	s += "\n" + HelpStyle.Render("Use arrow keys to navigate, type to edit, Enter to create table")
	return s
}

// RenderJoinTable renders the join table screen
func (r *Renderer) RenderJoinTable() string {
	var s string
	s += TitleStyle.Render("Join Table") + "\n"
	s += FocusedStyle.Render(fmt.Sprintf("Table ID: %s", r.ui.tableIdInput)) + "\n"
	s += HelpStyle.Render("Enter table ID and press Enter to join")
	return s
}

// RenderGameLobby renders the game lobby screen
func (r *Renderer) RenderGameLobby() string {
	var s string
	s += TitleStyle.Render(fmt.Sprintf("Game Lobby - Table %s", r.ui.pc.GetCurrentTableID())) + "\n"

	// Show client ID
	s += fmt.Sprintf("Client ID: %s\n", r.ui.clientID)

	// Display cached balance in DCR
	if r.ui.balance > 0 {
		s += fmt.Sprintf("Account Balance: %.8f DCR\n", float64(r.ui.balance)/1e8)
	} else {
		s += "Account Balance: (loading...)\n"
	}

	// Show table information if we have game update data
	if len(r.ui.players) > 0 {
		s += "Table Status:\n"
		s += fmt.Sprintf("Players: %d/%d (required to start)\n", r.ui.playersJoined, r.ui.playersRequired)
		s += fmt.Sprintf("Game Phase: %s\n", r.ui.gamePhase.String())
		if r.ui.pot > 0 {
			s += fmt.Sprintf("Pot: %d chips\n", r.ui.pot)
		}
		s += "\n"

		s += "Players at table:\n"
		for _, player := range r.ui.players {
			readyStatus := ""
			if player.IsReady {
				readyStatus = " ✅"
			} else {
				readyStatus = " ⏳"
			}

			currentPlayerIndicator := ""
			if player.Id == r.ui.clientID {
				currentPlayerIndicator = " (You)"
			}

			s += fmt.Sprintf("  %s: %d chips%s%s\n",
				player.Id, player.Balance, readyStatus, currentPlayerIndicator)
		}
		s += "\n"
	} else {
		s += "Loading table information...\n"
	}

	options := r.ui.getGameLobbyOptions()
	for i, option := range options {
		if i == r.ui.selectedItem {
			s += FocusedStyle.Render(fmt.Sprintf("> %s", option)) + "\n"
		} else {
			s += BlurredStyle.Render(fmt.Sprintf("  %s", option)) + "\n"
		}
	}

	return s
}

// RenderBetInput renders the bet input screen
func (r *Renderer) RenderBetInput() string {
	var s string
	s += TitleStyle.Render("Enter Bet Amount") + "\n"

	// Show client ID
	s += fmt.Sprintf("Client ID: %s\n", r.ui.clientID)

	bigBlind := r.ui.GetCurrentTableBigBlind()
	currentBet := r.ui.currentBet

	// Find the current player's bet amount
	var playerCurrentBet int64 = 0
	for _, player := range r.ui.players {
		if player.Id == r.ui.clientID {
			playerCurrentBet = player.CurrentBet
			break
		}
	}

	if currentBet == 0 {
		s += "No current bet\n"
		s += fmt.Sprintf("Minimum bet (big blind): %d\n", bigBlind)
	} else {
		s += fmt.Sprintf("Current bet to call: %d\n", currentBet)
		if playerCurrentBet < currentBet {
			// Player has a bet to call
			callAmount := currentBet - playerCurrentBet
			s += fmt.Sprintf("Amount to call: %d (you have %d bet, need %d more)\n", callAmount, playerCurrentBet, callAmount)
		}
		s += fmt.Sprintf("Minimum to call: %d\n", currentBet)
		s += fmt.Sprintf("Minimum to raise: %d\n", currentBet+bigBlind)
	}

	s += FocusedStyle.Render(fmt.Sprintf("Bet Amount: %s", r.ui.betAmount)) + "\n"
	s += HelpStyle.Render("Type amount and press Enter to bet, or 'q' to cancel")
	return s
}

// RenderActiveGame creates an enhanced poker table visualization
func (r *Renderer) RenderActiveGame() string {
	var s string

	// Game title with icon
	s += TitleStyle.Render(fmt.Sprintf("🎰 Table %s", r.ui.pc.GetCurrentTableID())) + "\n"

	// Show client ID
	s += fmt.Sprintf("Client ID: %s\n", r.ui.clientID)

	// COMMUNITY CARDS
	s += r.renderCommunityCardsSection() + "\n"

	// YOUR CARDS
	s += r.renderYourCardsAndGameInfo() + "\n"

	// SHOWDOWN RESULTS (if in showdown phase)
	if r.ui.gamePhase == pokerrpc.GamePhase_SHOWDOWN {
		s += r.renderShowdownResults() + "\n"
	}

	// Player information
	s += r.renderPlayersCompact() + "\n"

	// Action buttons for current player or waiting/leave options
	actionButtons := r.renderActionButtons()
	if actionButtons != "" {
		s += actionButtons
	} else {
		// Show appropriate options when not player's turn
		if r.ui.gamePhase == pokerrpc.GamePhase_SHOWDOWN {
			s += HelpStyle.Render("🏆 SHOWDOWN") + " | "
		} else if r.ui.currentPlayerID == "" {
			s += HelpStyle.Render("⏳ STARTING") + " | "
		} else if !isPlayerTurn(r.ui.currentPlayerID, r.ui.clientID) {
			// Find the current player's name for display
			currentPlayerName := r.ui.currentPlayerID
			if len(currentPlayerName) > 8 {
				currentPlayerName = currentPlayerName[:8] + "..."
			}
			s += HelpStyle.Render(fmt.Sprintf("⏰ Waiting for %s", currentPlayerName)) + " | "
		}

		// Always show leave table option when not actively playing
		s += BlurredStyle.Render("🚪 Leave Table")
	}

	s += "\n" + HelpStyle.Render("Arrow keys to navigate, Enter to select, 'q' to go back")
	return s
}

// renderCommunityCardsSection creates a clear, prominent display of community cards with game info in the header
func (r *Renderer) renderCommunityCardsSection() string {
	var s string

	// Phase indicator - cleaner with minimal icons
	var phaseText string
	switch r.ui.gamePhase {
	case pokerrpc.GamePhase_WAITING:
		phaseText = "⏳ WAITING FOR PLAYERS"
	case pokerrpc.GamePhase_NEW_HAND_DEALING:
		phaseText = "🃏 DEALING NEW HAND"
	case pokerrpc.GamePhase_PRE_FLOP:
		phaseText = "🎯 PRE-FLOP"
	case pokerrpc.GamePhase_FLOP:
		phaseText = "🎲 FLOP"
	case pokerrpc.GamePhase_TURN:
		phaseText = "♠️ TURN"
	case pokerrpc.GamePhase_RIVER:
		phaseText = "♥️ RIVER"
	case pokerrpc.GamePhase_SHOWDOWN:
		phaseText = "🏆 SHOWDOWN"
	default:
		phaseText = "❓ UNKNOWN"
	}

	// Create header with game info
	headerSection := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Bold(true).
		Background(lipgloss.Color("22")).
		Padding(0, 2).
		MarginTop(1).
		MarginBottom(1).
		Render("🃏 COMMUNITY CARDS")

	// Game info section with phase and pot info
	var gameInfo string
	gameInfo += lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Bold(true).
		Render(phaseText)

	// Add pot and bet info
	potDisplay := fmt.Sprintf("💰 Pot: %d", r.ui.pot)
	if r.ui.currentBet > 0 {
		potDisplay += fmt.Sprintf(" | Current Bet: %d", r.ui.currentBet)
	}

	gameInfo += " | " + lipgloss.NewStyle().
		Foreground(lipgloss.Color("140")).
		Render(potDisplay)

	gameInfoSection := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Bold(true).
		MarginLeft(3).
		Render(gameInfo)

	// Join header and game info horizontally
	s += lipgloss.JoinHorizontal(lipgloss.Center, headerSection, gameInfoSection) + "\n"

	// Cards display
	var cardElements []string

	switch r.ui.gamePhase {
	case pokerrpc.GamePhase_WAITING, pokerrpc.GamePhase_NEW_HAND_DEALING, pokerrpc.GamePhase_PRE_FLOP:
		// Show placeholders
		for i := 0; i < 5; i++ {
			cardElements = append(cardElements, CardStyle.Render("  "))
		}
	case pokerrpc.GamePhase_SHOWDOWN:
		// During showdown, show all dealt community cards without placeholders
		for _, card := range r.ui.communityCards {
			cardDisplay := r.formatCard(card)
			var styledCard string
			if isRedSuit(card.Suit) {
				styledCard = RedCardStyle.Render(cardDisplay)
			} else {
				styledCard = CardStyle.Render(cardDisplay)
			}
			cardElements = append(cardElements, styledCard)
		}
	default:
		// Add dealt community cards
		for _, card := range r.ui.communityCards {
			cardDisplay := r.formatCard(card)
			var styledCard string
			if isRedSuit(card.Suit) {
				styledCard = RedCardStyle.Render(cardDisplay)
			} else {
				styledCard = CardStyle.Render(cardDisplay)
			}
			cardElements = append(cardElements, styledCard)
		}

		// Add placeholders for remaining cards
		for i := len(r.ui.communityCards); i < 5; i++ {
			cardElements = append(cardElements, CardStyle.Render("  "))
		}
	}

	// Display cards centered
	cardsDisplay := strings.Join(cardElements, " ")
	s += lipgloss.NewStyle().
		Align(lipgloss.Center).
		Render(cardsDisplay)

	return s
}

// renderYourCardsAndGameInfo creates a compact display of player cards
func (r *Renderer) renderYourCardsAndGameInfo() string {
	var s string

	// Your cards header - clean and prominent with subtle icon
	s += lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Bold(true).
		Background(lipgloss.Color("17")).
		Padding(0, 2).
		MarginTop(1).
		MarginBottom(1).
		Render("🂠 YOUR HAND") + "\n"

	// During WAITING we don't expect cards; render header only.
	if r.ui.gamePhase == pokerrpc.GamePhase_WAITING {
		return s
	}

	// Require player list outside WAITING; otherwise surface an error.
	if len(r.ui.players) == 0 {
		r.ui.err = fmt.Errorf("players not available for rendering hand (phase=%s)", r.ui.gamePhase.String())
		return s
	}

	// Find player's cards
	var playerCards []*pokerrpc.Card
	for _, player := range r.ui.players {
		if player.Id == r.ui.clientID {
			playerCards = player.Hand
			break
		}
	}

	// If no cards are available during an active phase, raise an error and do not render placeholders.
	if len(playerCards) == 0 {
		r.ui.err = fmt.Errorf("no hole cards available (phase=%s)", r.ui.gamePhase.String())
		return s
	}

	// Render actual cards
	var cardElements []string
	for _, card := range playerCards {
		cardDisplay := r.formatCard(card)
		var styledCard string
		if isRedSuit(card.Suit) {
			styledCard = RedCardStyle.Render(cardDisplay)
		} else {
			styledCard = CardStyle.Render(cardDisplay)
		}
		cardElements = append(cardElements, styledCard)
	}

	cardsDisplay := strings.Join(cardElements, " ")
	s += lipgloss.NewStyle().
		Align(lipgloss.Center).
		Render(cardsDisplay)

	return s
}

// renderPlayersAroundTable creates a visual representation of players around a poker table
func (r *Renderer) renderPlayersAroundTable() string {
	if len(r.ui.players) == 0 {
		return ""
	}

	var result string
	result += TitleStyle.Render("👥 Players at Table 👥")

	// Arrange players in a visual table layout
	for i, player := range r.ui.players {
		playerInfo := r.formatPlayerInfo(player)

		// Choose style based on player state
		var style lipgloss.Style
		if player.Id == r.ui.clientID {
			style = YourPlayerStyle
		} else if player.Id == r.ui.currentPlayerID {
			style = CurrentPlayerStyle
		} else if player.Folded {
			style = FoldedPlayerStyle
		} else {
			style = PlayerBoxStyle
		}

		// Add position indicator
		position := fmt.Sprintf("Seat %d", i+1)
		fullPlayerInfo := fmt.Sprintf("%s\n%s", position, playerInfo)

		result += style.Render(fullPlayerInfo) + "\n"
	}

	return result
}

// formatPlayerInfo creates a clean formatted string for player information
func (r *Renderer) formatPlayerInfo(player *pokerrpc.Player) string {
	var info []string

	// Player ID (truncated for display)
	playerName := player.Id
	if len(playerName) > 12 {
		playerName = playerName[:12] + "..."
	}
	info = append(info, playerName)

	// Game chips balance
	info = append(info, fmt.Sprintf("Chips: %d", player.Balance))

	// Current bet
	if player.CurrentBet > 0 {
		// Clarify that this is the total committed this round (absolute),
		// not the delta of the last action.
		info = append(info, fmt.Sprintf("Bet (total this street): %d", player.CurrentBet))
	}

	// Status indicators - clean and clear
	var status []string

	// Check if player is you
	if player.Id == r.ui.clientID {
		status = append(status, "YOU")
	}

	// Check if it's their turn (and they haven't folded)
	if player.Id == r.ui.currentPlayerID && !player.Folded {
		status = append(status, "ACTING")
	}

	// Only show folded if player has actually folded
	if player.Folded {
		status = append(status, "FOLDED")
	} else {
		// Player is active in the game
		if r.ui.gamePhase != pokerrpc.GamePhase_WAITING && r.ui.gamePhase != pokerrpc.GamePhase_NEW_HAND_DEALING {
			status = append(status, "ACTIVE")
		} else {
			// In waiting or dealing phase, show ready status
			if player.IsReady {
				status = append(status, "READY")
			} else {
				status = append(status, "NOT READY")
			}
		}
	}

	if len(status) > 0 {
		info = append(info, fmt.Sprintf("[%s]", strings.Join(status, ", ")))
	}

	return strings.Join(info, " | ")
}

// renderActionButtons creates clear action buttons with better UX
func (r *Renderer) renderActionButtons() string {
	// During showdown, show card visibility options for all players
	if r.ui.gamePhase == pokerrpc.GamePhase_SHOWDOWN {
		var result string

		// Card visibility toggle section
		result += lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true).
			MarginTop(1).
			MarginBottom(1).
			Render("🎴 CARD VISIBILITY OPTIONS") + "\n"

		options := r.ui.getActiveGameOptions()
		for i, option := range options {
			var buttonText string
			switch option {
			case "Show My Cards":
				buttonText = "👁️ Show My Cards"
			case "Hide My Cards":
				buttonText = "🙈 Hide My Cards"
			case "Leave Table":
				buttonText = "🚪 Leave Table"
			default:
				buttonText = option
			}

			if i == r.ui.selectedItem {
				result += FocusedStyle.
					Render(fmt.Sprintf("> %s", buttonText)) + "\n"
			} else {
				result += BlurredStyle.
					Render(fmt.Sprintf("  %s", buttonText)) + "\n"
			}
		}
		return result
	}

	// Check if it's player's turn and they're in an active game phase
	if !isPlayerTurn(r.ui.currentPlayerID, r.ui.clientID) {
		return ""
	}

	// Only show action buttons during betting phases
	switch r.ui.gamePhase {
	case pokerrpc.GamePhase_PRE_FLOP, pokerrpc.GamePhase_FLOP, pokerrpc.GamePhase_TURN, pokerrpc.GamePhase_RIVER:
		// These are valid phases for player actions
	case pokerrpc.GamePhase_SHOWDOWN:
		// Show limited actions during showdown (card visibility, leave table)
	default:
		// In other phases (WAITING, NEW_HAND_DEALING), don't show action buttons
		return ""
	}

	var result string

	// Very clear turn indicator with icon
	result += YourTurnStyle.
		MarginTop(1).
		MarginBottom(1).
		Render("⚡ YOUR TURN - CHOOSE ACTION") + "\n"

	// Clean action buttons with minimal icons
	options := r.ui.getActiveGameOptions()
	for i, option := range options {
		var buttonText string

		switch option {
		case "Check":
			buttonText = "✅ Check"
		case "Call":
			buttonText = "📞 Call"
		case "Bet":
			buttonText = "💸 Bet/Raise"
		case "Fold":
			buttonText = "❌ Fold"
		case "Show My Cards":
			buttonText = "👁️ Show My Cards"
		case "Hide My Cards":
			buttonText = "🙈 Hide My Cards"
		case "Leave Table":
			buttonText = "🚪 Leave Table"
		default:
			buttonText = option
		}

		if i == r.ui.selectedItem {
			result += FocusedStyle.
				Render(fmt.Sprintf("> %s", buttonText)) + "\n"
		} else {
			result += BlurredStyle.
				Render(fmt.Sprintf("  %s", buttonText)) + "\n"
		}
	}

	return result
}

// formatCard creates a clean visual representation of a playing card
func (r *Renderer) formatCard(card *pokerrpc.Card) string {
	if card == nil {
		return "  "
	}

	// Convert suit to symbol
	suitSymbol := getSuitSymbol(card.Suit)

	// Format value
	value := card.Value
	if value == "T" {
		value = "10"
	}

	return fmt.Sprintf("%s%s", value, suitSymbol)
}

// getSuitSymbol returns the appropriate symbol for a suit
func getSuitSymbol(suit string) string {
	// The suit is already the Unicode symbol, so just return it
	switch suit {
	case "♠":
		return "♠"
	case "♥":
		return "♥"
	case "♦":
		return "♦"
	case "♣":
		return "♣"
	default:
		return "?"
	}
}

// isRedSuit determines if a suit should be displayed in red
func isRedSuit(suit string) bool {
	return suit == "♥" || suit == "♦"
}

// renderPlayersCompact creates a clear representation of players with better turn indication
func (r *Renderer) renderPlayersCompact() string {
	if len(r.ui.players) == 0 {
		return HelpStyle.Render("No players")
	}

	var result strings.Builder
	result.WriteString(lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Bold(true).
		MarginTop(1).
		MarginBottom(1).
		Render("👥 PLAYERS") + "\n")

	var playerLines []string
	for _, player := range r.ui.players {
		// Player name (truncated reasonably)
		playerName := player.Id
		if len(playerName) > 12 {
			playerName = playerName[:12] + "..."
		}

		// Determine player status and styling
		var line string
		var style lipgloss.Style

		if player.Id == r.ui.clientID {
			// This is YOU - make it very clear
			style = lipgloss.NewStyle().
				Foreground(lipgloss.Color("39")).
				Bold(true).
				Background(lipgloss.Color("17"))
			line = fmt.Sprintf(" 🔵 YOU (%s) - Chips: %d", playerName, player.Balance)
		} else if player.Id == r.ui.currentPlayerID && !player.Folded {
			// Current player's turn - highlight clearly
			style = lipgloss.NewStyle().
				Foreground(lipgloss.Color("46")).
				Bold(true).
				Background(lipgloss.Color("22"))
			line = fmt.Sprintf(" ⏰ ACTING: %s - Chips: %d", playerName, player.Balance)
		} else if player.Folded {
			// Folded player - muted
			style = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241"))
			line = fmt.Sprintf(" ❌ %s - FOLDED", playerName)
		} else {
			// Regular active player
			style = lipgloss.NewStyle().
				Foreground(lipgloss.Color("255"))
			line = fmt.Sprintf(" %s - Chips: %d", playerName, player.Balance)
		}

		// Add current bet if they have one
		if player.CurrentBet > 0 && !player.Folded {
			line += fmt.Sprintf(" (Bet: %d)", player.CurrentBet)
		}

		playerLines = append(playerLines, style.Render(line))
	}

	result.WriteString(strings.Join(playerLines, "\n"))
	return result.String()
}

// renderShowdownResults displays the showdown results with players side by side
func (r *Renderer) renderShowdownResults() string {
	if r.ui.gamePhase != pokerrpc.GamePhase_SHOWDOWN {
		return ""
	}

	var s string

	// Showdown header
	s += lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Bold(true).
		Background(lipgloss.Color("202")).
		Padding(0, 2).
		Align(lipgloss.Center).
		Width(50).
		MarginTop(1).
		MarginBottom(1).
		Render("🏆 SHOWDOWN RESULTS 🏆") + "\n"

	// Get active players - show all players who made it to showdown (didn't fold)
	var activePlayers []*pokerrpc.Player
	for _, player := range r.ui.players {
		if !player.Folded {
			activePlayers = append(activePlayers, player)
		}
	}

	if len(activePlayers) == 0 {
		return s + "No active players\n"
	}

	// Create player boxes side by side
	var playerBoxes []string
	for _, player := range activePlayers {
		playerName := player.Id
		if len(playerName) > 12 {
			playerName = playerName[:12] + "..."
		}

		// Build player box content
		var boxContent strings.Builder

		// Player name with icon
		if player.Id == r.ui.clientID {
			boxContent.WriteString(lipgloss.NewStyle().
				Foreground(lipgloss.Color("39")).
				Bold(true).
				Render("🔵 YOU") + "\n")
		} else {
			boxContent.WriteString(lipgloss.NewStyle().
				Foreground(lipgloss.Color("255")).
				Bold(true).
				Render("👤 "+playerName) + "\n")
		}

		// Show player's hole cards (check visibility for each player)
		shouldShowCards := false
		if player.Id == r.ui.clientID {
			// For current client, use their showMyCards setting
			shouldShowCards = r.ui.showMyCards
		} else {
			// For other players, check if they have chosen to show their cards
			shouldShowCards = r.ui.playersShowingCards[player.Id]
		}

		if len(player.Hand) > 0 && shouldShowCards {
			var cardElements []string
			for _, card := range player.Hand {
				cardDisplay := r.formatCard(card)
				var styledCard string
				if isRedSuit(card.Suit) {
					styledCard = RedCardStyle.Render(cardDisplay)
				} else {
					styledCard = CardStyle.Render(cardDisplay)
				}
				cardElements = append(cardElements, styledCard)
			}
			cardsDisplay := strings.Join(cardElements, " ")
			boxContent.WriteString(cardsDisplay + "\n")
		} else if !shouldShowCards {
			// Show hidden cards for current player who chose to hide
			hiddenCards := CardStyle.Render("??") + " " + CardStyle.Render("??")
			boxContent.WriteString(hiddenCards + "\n")
		}

		// Show hand description (respect each player's card visibility setting)
		handDescription := "Evaluating..."
		if player.GetHandDescription() != "" {
			handDescription = player.GetHandDescription()
		}

		// Hide hand description if player has chosen to hide cards
		if player.Id == r.ui.clientID && !r.ui.showMyCards {
			handDescription = "Cards Hidden"
		} else if player.Id != r.ui.clientID && !r.ui.playersShowingCards[player.Id] {
			handDescription = "Cards Hidden"
		}

		boxContent.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Render(handDescription))

		// Style the player box
		var boxStyle lipgloss.Style
		if player.Id == r.ui.clientID {
			boxStyle = lipgloss.NewStyle().
				Border(lipgloss.DoubleBorder()).
				BorderForeground(lipgloss.Color("39")).
				Padding(0, 1).
				Margin(0, 1).
				Background(lipgloss.Color("17")).
				Width(14)
		} else {
			boxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("255")).
				Padding(0, 1).
				Margin(0, 1).
				Width(14)
		}

		playerBoxes = append(playerBoxes, boxStyle.Render(boxContent.String()))
	}

	// Join player boxes horizontally
	s += lipgloss.JoinHorizontal(lipgloss.Top, playerBoxes...) + "\n"

	// Show folded players section
	var foldedPlayers []*pokerrpc.Player
	for _, player := range r.ui.players {
		if player.Folded {
			foldedPlayers = append(foldedPlayers, player)
		}
	}

	if len(foldedPlayers) > 0 {
		s += lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Bold(true).
			Background(lipgloss.Color("52")).
			Padding(0, 1).
			MarginTop(1).
			MarginBottom(1).
			Render("❌ FOLDED PLAYERS") + "\n"

		var foldedLines []string
		for _, player := range foldedPlayers {
			playerName := player.Id
			if len(playerName) > 15 {
				playerName = playerName[:15] + "..."
			}

			var foldedStyle lipgloss.Style
			if player.Id == r.ui.clientID {
				foldedStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("241")).
					Bold(true)
			} else {
				foldedStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("241"))
			}

			foldedLine := fmt.Sprintf("❌ %s - FOLDED", playerName)
			foldedLines = append(foldedLines, foldedStyle.Render(foldedLine))
		}

		s += strings.Join(foldedLines, "  ") + "\n"
	}

	// Display winners section
	if len(r.ui.winners) > 0 {
		// Create the winners line with title and winner info on same line
		winnersLine := lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Bold(true).
			Background(lipgloss.Color("22")).
			Padding(0, 1).
			MarginTop(1).
			Render("🏆 WINNERS")

		// Add winner info directly to the same line
		for _, winner := range r.ui.winners {
			winnerName := winner.PlayerId
			if len(winnerName) > 15 {
				winnerName = winnerName[:15] + "..."
			}

			var winnerStyle lipgloss.Style
			if winner.PlayerId == r.ui.clientID {
				winnerStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("46")).
					Bold(true)
			} else {
				winnerStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("46")).
					Bold(true)
			}

			winnerInfo := fmt.Sprintf(" 🎉 %s - Won %d chips", winnerName, winner.Winnings)
			winnersLine += winnerStyle.Render(winnerInfo)
		}

		s += winnersLine + "\n"

		// Continue with the rest of winner details on separate lines
		for _, winner := range r.ui.winners {

			// Show winner's best 5-card hand
			if len(winner.BestHand) > 0 {
				var bestHandCards []string
				for _, card := range winner.BestHand {
					cardDisplay := r.formatCard(card)
					var styledCard string
					if isRedSuit(card.Suit) {
						styledCard = RedCardStyle.Render(cardDisplay)
					} else {
						styledCard = CardStyle.Render(cardDisplay)
					}
					bestHandCards = append(bestHandCards, styledCard)
				}
				bestHandDisplay := strings.Join(bestHandCards, " ")
				s += "   Best Hand: " + bestHandDisplay + "\n"
			}

			// Show hand rank
			rankText := winner.HandRank.String()
			s += "   Rank: " + lipgloss.NewStyle().
				Foreground(lipgloss.Color("214")).
				Bold(true).
				Render(rankText)

		}
	}

	return s
}

func (r *Renderer) renderGamePhase() string {
	switch r.ui.gamePhase {
	case pokerrpc.GamePhase_WAITING, pokerrpc.GamePhase_NEW_HAND_DEALING, pokerrpc.GamePhase_PRE_FLOP:
		return fmt.Sprintf("Players: %d/%d", int(r.ui.playersJoined), int(r.ui.playersRequired))
	case pokerrpc.GamePhase_FLOP, pokerrpc.GamePhase_TURN, pokerrpc.GamePhase_RIVER:
		return fmt.Sprintf("Pot: %d chips | Current Bet: %d chips", r.ui.pot, r.ui.currentBet)
	case pokerrpc.GamePhase_SHOWDOWN:
		if len(r.ui.winners) > 0 {
			return fmt.Sprintf("Pot: %d chips | Winners: %d", r.ui.pot, len(r.ui.winners))
		}
		return fmt.Sprintf("Pot: %d chips", r.ui.pot)
	default:
		return ""
	}
}
