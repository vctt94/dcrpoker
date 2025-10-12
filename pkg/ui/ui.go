package ui

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vctt94/pokerbisonrelay/pkg/client"
	"github.com/vctt94/pokerbisonrelay/pkg/poker"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
)

// stateFn represents a state function that processes input and returns the next state
type stateFn func(*PokerUI, tea.Msg) (stateFn, tea.Cmd)

// PokerUI contains all the state for our poker client UI
type PokerUI struct {
	// Common fields
	ctx          context.Context
	clientID     string
	pc           *client.PokerClient
	currentState stateFn
	currentView  string // Add this to track current view type
	err          error
	selectedItem int
	tables       []*pokerrpc.Table

	// Temporary message
	message string

	// Selected table in table list
	selectedTable int

	// Create table form inputs (just strings for simplicity)
	smallBlind        string
	bigBlind          string
	requiredPlayers   string
	buyIn             string
	minBalance        string
	startingChips     string
	autoAdvanceMs     string
	selectedFormField int // Track which form field is selected

	// For join table
	tableIdInput string

	// Game state tracking
	gamePhase       pokerrpc.GamePhase
	pot             int64
	currentBet      int64
	players         []*pokerrpc.Player
	communityCards  []*pokerrpc.Card
	currentPlayerID string
	playersRequired int32
	playersJoined   int32

	// Showdown results
	winners []*pokerrpc.Winner

	// Table configuration tracking
	currentTableBigBlind int64

	// For betting input
	betAmount string

	// Card visibility toggle
	showMyCards bool

	// Track card visibility for all players
	playersShowingCards map[string]bool

	// Component handlers
	dispatcher *CommandDispatcher
	renderer   *Renderer

	// current account balance
	balance int64
}

// NewPokerUI creates a new poker UI model
func NewPokerUI(ctx context.Context, client *client.PokerClient) *PokerUI {
	ui := &PokerUI{
		ctx:      ctx,
		clientID: client.ID.String(),
		pc:       client,

		smallBlind:          "10",
		bigBlind:            "20",
		requiredPlayers:     "2",
		buyIn:               "100",
		minBalance:          "100",
		startingChips:       "1000",
		autoAdvanceMs:       "1000", // Default 1 second delay between streets when all-in
		currentView:         "mainMenu",
		showMyCards:         false, // Default to not showing cards
		playersShowingCards: make(map[string]bool),
	}

	// Create component handlers
	ui.dispatcher = &CommandDispatcher{
		ctx:      ctx,
		clientID: client.ID.String(),
		pc:       client,
	}

	ui.renderer = &Renderer{ui: ui}

	// Set initial state
	ui.currentState = ui.stateMainMenu

	return ui
}

func (m *PokerUI) Init() tea.Cmd {
	return m.dispatcher.getBalanceCmd()
}

// GetCurrentTableBigBlind returns the big blind value for the current table
func (m *PokerUI) GetCurrentTableBigBlind() int64 {
	// Fallback: try to find current table in tables list
	currentTableID := m.pc.GetCurrentTableID()
	if currentTableID != "" {
		for _, table := range m.tables {
			if table.Id == currentTableID {
				m.currentTableBigBlind = table.BigBlind
				return table.BigBlind
			}
		}
	}

	// Default fallback (should not happen in normal operation)
	return 20
}

func (m *PokerUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle global messages first
	switch msg := msg.(type) {
	case tablesMsg:
		m.tables = []*pokerrpc.Table(msg)
		m.selectedTable = 0
		m.err = nil
		m.currentState = m.stateTableList
		m.currentView = "tableList"
		return m, nil

	case notificationMsg:
		notif := (*pokerrpc.Notification)(msg)
		m.message = notif.Message
		m.err = nil
		if notif.Type == pokerrpc.NotificationType_BALANCE_UPDATED {
			m.balance = notif.NewBalance
		}
		cmd := m.handleNotification(notif)
		return m, cmd

	case *pokerrpc.Notification:
		notif := msg
		m.message = notif.Message
		m.err = nil
		if notif.Type == pokerrpc.NotificationType_BALANCE_UPDATED {
			m.balance = notif.NewBalance
		}
		cmd := m.handleNotification(notif)
		return m, cmd

	case client.GameUpdateMsg:
		gameUpdate := (*pokerrpc.GameUpdate)(msg)
		m.updateGameState(gameUpdate)
		return m, nil

	case errorMsg:
		m.err = error(msg)
		m.message = ""
		return m, nil
	}

	// Delegate to current state function
	nextState, cmd := m.currentState(m, msg)
	m.currentState = nextState
	return m, cmd
}

// State functions

func (m *PokerUI) stateMainMenu(ui *PokerUI, msg tea.Msg) (stateFn, tea.Cmd) {
	m.currentView = "mainMenu"
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selectedItem > 0 {
				m.selectedItem--
			}
		case "down", "j":
			options := m.getMainMenuOptions()
			if m.selectedItem < len(options)-1 {
				m.selectedItem++
			}
		case "enter", " ":
			options := m.getMainMenuOptions()
			if m.selectedItem < len(options) {
				return m.handleMainMenuSelection(options[m.selectedItem])
			}
		case "q", "ctrl+c":
			return m.stateMainMenu, tea.Quit
		}
	}
	return m.stateMainMenu, nil
}

func (m *PokerUI) stateTableList(ui *PokerUI, msg tea.Msg) (stateFn, tea.Cmd) {
	m.currentView = "tableList"
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selectedTable > 0 {
				m.selectedTable--
			}
		case "down", "j":
			if m.selectedTable < len(m.tables)-1 {
				m.selectedTable++
			}
		case "enter", " ":
			if len(m.tables) > 0 && m.selectedTable < len(m.tables) {
				table := m.tables[m.selectedTable]
				return m.stateTableList, m.dispatcher.joinTableCmd(table.Id)
			}
		case "r":
			return m.stateTableList, m.dispatcher.getTablesCmd()
		case "q":
			m.selectedItem = 0
			m.currentView = "mainMenu"
			return m.stateMainMenu, nil
		case "ctrl+c":
			return m.stateTableList, tea.Quit
		}
	}
	return m.stateTableList, nil
}

func (m *PokerUI) stateCreateTable(ui *PokerUI, msg tea.Msg) (stateFn, tea.Cmd) {
	m.currentView = "createTable"
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selectedFormField > 0 {
				m.selectedFormField--
			}
		case "down", "j":
			if m.selectedFormField < 6 { // 7 fields total (0-6)
				m.selectedFormField++
			}
		case "enter", " ":
			// Parse form values and create table config
			smallBlind, _ := strconv.ParseInt(m.smallBlind, 10, 64)
			bigBlind, _ := strconv.ParseInt(m.bigBlind, 10, 64)
			requiredPlayers, _ := strconv.ParseInt(m.requiredPlayers, 10, 32)
			buyIn, _ := strconv.ParseInt(m.buyIn, 10, 64)
			minBalance, _ := strconv.ParseInt(m.minBalance, 10, 64)
			startingChips, _ := strconv.ParseInt(m.startingChips, 10, 64)
			autoAdvanceMs, _ := strconv.ParseInt(m.autoAdvanceMs, 10, 64)

			config := poker.TableConfig{
				SmallBlind:       smallBlind,
				BigBlind:         bigBlind,
				MinPlayers:       int(requiredPlayers),
				MaxPlayers:       int(requiredPlayers), // Using same value for min and max for now
				BuyIn:            buyIn,
				MinBalance:       minBalance,
				StartingChips:    startingChips,
				AutoStartDelay:   3 * time.Second,                                 // Auto-start new hands after 3 seconds
				AutoAdvanceDelay: time.Duration(autoAdvanceMs) * time.Millisecond, // Auto-advance delay when all-in
			}
			return m.stateCreateTable, m.dispatcher.createTableCmd(config)
		case "q":
			m.selectedItem = 0
			m.currentView = "mainMenu"
			return m.stateMainMenu, nil
		case "ctrl+c":
			return m.stateCreateTable, tea.Quit
		default:
			// Handle text input for the selected field
			m.updateFormField(msg.String())
		}
	}
	return m.stateCreateTable, nil
}

func (m *PokerUI) stateJoinTable(ui *PokerUI, msg tea.Msg) (stateFn, tea.Cmd) {
	m.currentView = "joinTable"
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", " ":
			return m.stateJoinTable, m.dispatcher.joinTableCmd(m.tableIdInput)
		case "q":
			m.selectedItem = 0
			m.tableIdInput = ""
			m.currentView = "mainMenu"
			return m.stateMainMenu, nil
		case "ctrl+c":
			return m.stateJoinTable, tea.Quit
		case "backspace":
			if len(m.tableIdInput) > 0 {
				m.tableIdInput = m.tableIdInput[:len(m.tableIdInput)-1]
			}
		default:
			if len(msg.String()) == 1 {
				m.tableIdInput += msg.String()
			}
		}
	}
	return m.stateJoinTable, nil
}

func (m *PokerUI) stateGameLobby(ui *PokerUI, msg tea.Msg) (stateFn, tea.Cmd) {
	m.currentView = "gameLobby"
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selectedItem > 0 {
				m.selectedItem--
			}
		case "down", "j":
			options := m.getGameLobbyOptions()
			if m.selectedItem < len(options)-1 {
				m.selectedItem++
			}
		case "enter", " ":
			options := m.getGameLobbyOptions()
			if m.selectedItem < len(options) {
				return m.handleGameLobbySelection(options[m.selectedItem])
			}
		case "q":
			return m.stateGameLobby, m.dispatcher.leaveTableCmd()
		case "ctrl+c":
			return m.stateGameLobby, tea.Quit
		}
	}
	return m.stateGameLobby, nil
}

func (m *PokerUI) stateActiveGame(ui *PokerUI, msg tea.Msg) (stateFn, tea.Cmd) {
	m.currentView = "activeGame"
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selectedItem > 0 {
				m.selectedItem--
			}
		case "down", "j":
			options := m.getActiveGameOptions()
			if m.selectedItem < len(options)-1 {
				m.selectedItem++
			}
		case "enter", " ":
			options := m.getActiveGameOptions()
			if m.selectedItem < len(options) {
				return m.handleActiveGameSelection(options[m.selectedItem])
			}
		case "q":
			return m.stateActiveGame, m.dispatcher.leaveTableCmd()
		case "ctrl+c":
			return m.stateActiveGame, tea.Quit
		}
	}
	return m.stateActiveGame, nil
}

func (m *PokerUI) stateBetInput(ui *PokerUI, msg tea.Msg) (stateFn, tea.Cmd) {
	m.currentView = "betInput"
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", " ":
			// Parse bet amount and call betCmd
			amount, err := strconv.ParseInt(m.betAmount, 10, 64)
			if err != nil {
				return m.stateBetInput, nil // Stay in bet input if invalid amount
			}
			return m.stateActiveGame, m.dispatcher.raiseCmd(amount)
		case "q":
			m.betAmount = ""
			m.currentView = "activeGame"
			return m.stateActiveGame, nil
		case "ctrl+c":
			return m.stateBetInput, tea.Quit
		case "backspace":
			if len(m.betAmount) > 0 {
				m.betAmount = m.betAmount[:len(m.betAmount)-1]
			}
		default:
			if len(msg.String()) == 1 && msg.String() >= "0" && msg.String() <= "9" {
				m.betAmount += msg.String()
			}
		}
	}
	return m.stateBetInput, nil
}

// Helper functions for menu options

func (m *PokerUI) getMainMenuOptions() []string {
	options := []string{
		"List Tables",
		"Create Table",
		"Join Table",
		"Check Balance",
		"Quit",
	}
	currentTableID := m.pc.GetCurrentTableID()
	if currentTableID != "" {
		options = append([]string{"Return to Table"}, options...)
	}
	return options
}

func (m *PokerUI) getGameLobbyOptions() []string {
	return []string{
		"Set Ready",
		"Set Unready",
		"Leave Table",
		"Check Balance",
		"Quit",
	}
}

func (m *PokerUI) getActiveGameOptions() []string {
	// During showdown, show card visibility toggle and leave table option
	if m.gamePhase == pokerrpc.GamePhase_SHOWDOWN {
		cardToggleText := "Hide My Cards"
		if !m.showMyCards {
			cardToggleText = "Show My Cards"
		}
		return []string{
			cardToggleText,
			"Leave Table",
		}
	}

	if !isPlayerTurn(m.currentPlayerID, m.clientID) {
		return []string{"Leave Table"}
	}

	// Find the current player's bet amount
	var playerCurrentBet int64 = 0
	for _, player := range m.players {
		if player.Id == m.clientID {
			playerCurrentBet = player.CurrentBet
			break
		}
	}

	// Determine if player can check or needs to call
	if playerCurrentBet < m.currentBet {
		// Player has a bet to call - show Call instead of Check
		return []string{
			"Call",
			"Bet", // This will be raise since there's a bet to call
			"Fold",
			"Leave Table",
		}
	} else {
		// Player can check (no bet to call)
		return []string{
			"Check",
			"Bet",
			"Fold",
			"Leave Table",
		}
	}
}

// Selection handlers

func (m *PokerUI) handleMainMenuSelection(option string) (stateFn, tea.Cmd) {
	switch option {
	case "List Tables":
		return m.stateMainMenu, m.dispatcher.getTablesCmd()
	case "Create Table":
		m.selectedFormField = 0
		m.currentView = "createTable"
		return m.stateCreateTable, nil
	case "Join Table":
		m.tableIdInput = ""
		m.currentView = "joinTable"
		return m.stateJoinTable, nil
	case "Check Balance":
		return m.stateMainMenu, m.dispatcher.getBalanceCmd()
	case "Return to Table":
		currentTableID := m.pc.GetCurrentTableID()
		if currentTableID != "" {
			m.currentView = "gameLobby"
			return m.stateGameLobby, nil
		}
		return m.stateMainMenu, nil
	case "Quit":
		return m.stateMainMenu, tea.Quit
	}
	return m.stateMainMenu, nil
}

func (m *PokerUI) handleGameLobbySelection(option string) (stateFn, tea.Cmd) {
	switch option {
	case "Set Ready":
		return m.stateGameLobby, m.dispatcher.setPlayerReadyCmd()
	case "Set Unready":
		return m.stateGameLobby, m.dispatcher.setPlayerUnreadyCmd()
	case "Leave Table":
		return m.stateGameLobby, m.dispatcher.leaveTableCmd()
	case "Check Balance":
		return m.stateGameLobby, m.dispatcher.getBalanceCmd()
	case "Quit":
		return m.stateGameLobby, tea.Quit
	}
	return m.stateGameLobby, nil
}

func (m *PokerUI) handleActiveGameSelection(option string) (stateFn, tea.Cmd) {
	switch option {
	case "Check":
		return m.stateActiveGame, m.dispatcher.checkCmd()
	case "Call":
		return m.stateActiveGame, m.dispatcher.callCmd()
	case "Bet":
		m.betAmount = ""
		m.currentView = "betInput"
		return m.stateBetInput, nil
	case "Fold":
		return m.stateActiveGame, m.dispatcher.foldCmd()
	case "Show My Cards":
		// Toggle card visibility and send notification
		m.showMyCards = true
		return m.stateActiveGame, m.dispatcher.showCardsCmd()
	case "Hide My Cards":
		// Toggle card visibility and send notification
		m.showMyCards = false
		return m.stateActiveGame, m.dispatcher.hideCardsCmd()
	case "Leave Table":
		return m.stateActiveGame, m.dispatcher.leaveTableCmd()
	}
	return m.stateActiveGame, nil
}

// Helper functions

func (m *PokerUI) updateFormField(input string) {
	if len(input) == 1 && input >= "0" && input <= "9" {
		switch m.selectedFormField {
		case 0:
			m.smallBlind += input
		case 1:
			m.bigBlind += input
		case 2:
			m.requiredPlayers += input
		case 3:
			m.buyIn += input
		case 4:
			m.minBalance += input
		case 5:
			m.startingChips += input
		case 6:
			m.autoAdvanceMs += input
		}
	} else if input == "backspace" {
		switch m.selectedFormField {
		case 0:
			if len(m.smallBlind) > 0 {
				m.smallBlind = m.smallBlind[:len(m.smallBlind)-1]
			}
		case 1:
			if len(m.bigBlind) > 0 {
				m.bigBlind = m.bigBlind[:len(m.bigBlind)-1]
			}
		case 2:
			if len(m.requiredPlayers) > 0 {
				m.requiredPlayers = m.requiredPlayers[:len(m.requiredPlayers)-1]
			}
		case 3:
			if len(m.buyIn) > 0 {
				m.buyIn = m.buyIn[:len(m.buyIn)-1]
			}
		case 4:
			if len(m.minBalance) > 0 {
				m.minBalance = m.minBalance[:len(m.minBalance)-1]
			}
		case 5:
			if len(m.startingChips) > 0 {
				m.startingChips = m.startingChips[:len(m.startingChips)-1]
			}
		case 6:
			if len(m.autoAdvanceMs) > 0 {
				m.autoAdvanceMs = m.autoAdvanceMs[:len(m.autoAdvanceMs)-1]
			}
		}
	}
}

func (m *PokerUI) updateGameState(gameUpdate *pokerrpc.GameUpdate) {
	// Update current player ID
	m.currentPlayerID = gameUpdate.CurrentPlayer

	// Handle state updates from the game
	m.gamePhase = gameUpdate.Phase
	m.players = gameUpdate.Players
	m.communityCards = gameUpdate.CommunityCards
	m.pot = gameUpdate.Pot
	m.currentBet = gameUpdate.CurrentBet

	// Count joined players
	m.playersJoined = int32(len(m.players))
	// Set required players from game update
	m.playersRequired = gameUpdate.PlayersRequired

	// Determine current UI state based on game phase
	switch gameUpdate.Phase {
	case pokerrpc.GamePhase_WAITING:
		m.currentState = m.stateGameLobby
		m.currentView = "gameLobby"
	case pokerrpc.GamePhase_NEW_HAND_DEALING:
		// During dealing phase, stay in active game view but show dealing status
		m.currentState = m.stateActiveGame
		m.currentView = "activeGame"
	default:
		m.currentState = m.stateActiveGame
		m.currentView = "activeGame"
	}

	m.err = nil
}

// handleNotification processes notifications from the backend
func (m *PokerUI) handleNotification(notification *pokerrpc.Notification) tea.Cmd {
	switch notification.Type {
	case pokerrpc.NotificationType_BALANCE_UPDATED:
		m.message = fmt.Sprintf("Balance: %d", notification.NewBalance)
		m.balance = notification.NewBalance
		return nil

	case pokerrpc.NotificationType_PLAYER_JOINED:
		if notification.PlayerId == m.clientID {
			m.currentState = m.stateGameLobby
			m.currentView = "gameLobby"
			m.message = fmt.Sprintf("Joined table %s", notification.TableId)
			if notification.Table != nil {
				m.currentTableBigBlind = notification.Table.BigBlind
			}
			return nil
		}

	case pokerrpc.NotificationType_PLAYER_LEFT:
		if notification.PlayerId == m.clientID {
			m.resetToMainMenu()
			m.message = "Left table"
			return nil
		}

	case pokerrpc.NotificationType_TABLE_CREATED:
		if notification.PlayerId == m.clientID {
			m.currentState = m.stateGameLobby
			m.currentView = "gameLobby"
			m.message = fmt.Sprintf("Created table %s", notification.TableId)
			if notification.Table != nil {
				m.currentTableBigBlind = notification.Table.BigBlind
			}
			return nil
		}

	case pokerrpc.NotificationType_TABLE_REMOVED:
		currentTableID := m.pc.GetCurrentTableID()
		if notification.TableId == currentTableID {
			m.resetToMainMenu()
			m.message = "Table was closed"
			return nil
		}

	case pokerrpc.NotificationType_GAME_STARTED:
		m.currentState = m.stateActiveGame
		m.currentView = "activeGame"
		m.message = "Game started!"
		return nil

	case pokerrpc.NotificationType_GAME_ENDED:
		m.currentState = m.stateGameLobby
		m.currentView = "gameLobby"
		m.message = "Game ended"
		return m.dispatcher.getBalanceCmd()

	case pokerrpc.NotificationType_SHOWDOWN_RESULT:
		// Store showdown results for display
		m.winners = notification.Winners
		m.message = fmt.Sprintf("Showdown complete! Winners: %d players", len(notification.Winners))
		return nil

	case pokerrpc.NotificationType_NEW_HAND_STARTED:
		m.playersShowingCards = make(map[string]bool) // Reset card visibility tracking
		m.message = "New hand started!"
		// Stay in active game view - the game state update already arrived with new cards
		m.currentState = m.stateActiveGame
		m.currentView = "activeGame"
		return tea.ClearScreen

	case pokerrpc.NotificationType_CARDS_SHOWN:
		// Track that this player is showing their cards
		if notification.PlayerId != "" {
			m.playersShowingCards[notification.PlayerId] = true
		}

		if notification.PlayerId != m.clientID {
			m.message = fmt.Sprintf("%s is showing their cards", notification.PlayerId)
		} else {
			m.message = "Cards shown to other players"
		}
		return nil

	case pokerrpc.NotificationType_CARDS_HIDDEN:
		// Track that this player is hiding their cards
		if notification.PlayerId != "" {
			m.playersShowingCards[notification.PlayerId] = false
		}

		if notification.PlayerId != m.clientID {
			m.message = fmt.Sprintf("%s is hiding their cards", notification.PlayerId)
		} else {
			m.message = "Cards hidden from other players"
		}
		return nil

	default:
		m.message = notification.Message
		return nil
	}
	return nil
}

// View renders the current state of the UI
func (m *PokerUI) View() string {
	var s string

	// Show any temporary message
	if m.message != "" {
		s += TitleStyle.Render(m.message) + "\n\n"
	}

	// Show error if any
	if m.err != nil {
		s += ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n\n"
	}

	// Determine which view to render based on current view
	switch m.currentView {
	case "mainMenu":
		s += m.renderer.RenderMainMenu()
	case "tableList":
		s += m.renderer.RenderTableList()
	case "createTable":
		s += m.renderer.RenderCreateTable()
	case "joinTable":
		s += m.renderer.RenderJoinTable()
	case "gameLobby":
		s += m.renderer.RenderGameLobby()
	case "activeGame":
		s += m.renderer.RenderActiveGame()
	case "betInput":
		s += m.renderer.RenderBetInput()
	}

	s += "\n" + HelpStyle.Render("Press 'q' to go back/quit, Ctrl+C to force quit")
	return s
}

// Run starts the UI
func Run(ctx context.Context, client *client.PokerClient) {
	model := NewPokerUI(ctx, client)

	p := tea.NewProgram(model, tea.WithAltScreen())

	// Start a goroutine to listen for updates from the poker client
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-client.UpdatesCh:
				p.Send(msg)
			case err := <-client.ErrorsCh:
				p.Send(errorMsg(err))
			}
		}
	}()

	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running UI: %v", err)
	}
}

// resetToMainMenu resets the model to the main menu state, clearing all table and game data
func (m *PokerUI) resetToMainMenu() {
	m.currentState = m.stateMainMenu
	m.currentView = "mainMenu"
	m.message = ""
	m.err = nil
	m.resetGameState()
	m.selectedItem = 0
}

// resetGameState resets all game-related fields while keeping table connection
func (m *PokerUI) resetGameState() {
	m.players = nil
	m.communityCards = nil
	m.gamePhase = pokerrpc.GamePhase_WAITING
	m.pot = 0
	m.currentBet = 0
	m.currentPlayerID = ""
	m.playersRequired = 0
	m.playersJoined = 0
	m.winners = nil
	m.showMyCards = true                          // Reset to show cards by default for new games
	m.playersShowingCards = make(map[string]bool) // Reset card visibility tracking
}
