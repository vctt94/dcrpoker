package bot

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/companyzero/bisonrelay/clientrpc/types"
	"github.com/companyzero/bisonrelay/zkidentity"
	"github.com/decred/dcrd/dcrutil/v4"
	kit "github.com/vctt94/bisonbotkit"
	"github.com/vctt94/bisonbotkit/logging"
	"github.com/vctt94/pokerbisonrelay/pkg/poker"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	"github.com/vctt94/pokerbisonrelay/pkg/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const STARTING_CHIPS = 1000

// State holds the state of the poker bot
type State struct {
	db     server.Database
	tables map[string]*poker.Table
	mu     sync.RWMutex
}

// NewState creates a new bot state with the given database
func NewState(db server.Database) *State {
	return &State{
		db:     db,
		tables: make(map[string]*poker.Table),
	}
}

// SetupGRPCServer sets up and returns a configured GRPC server with TLS
func SetupGRPCServer(datadir, certFile, keyFile, serverAddress string, db server.Database, logBackend *logging.LogBackend) (*grpc.Server, net.Listener, error) {
	// Determine certificate and key file paths
	grpcCertFile := certFile
	grpcKeyFile := keyFile

	// If paths are still empty, use defaults
	if grpcCertFile == "" {
		grpcCertFile = filepath.Join(datadir, "server.cert")
	}
	if grpcKeyFile == "" {
		grpcKeyFile = filepath.Join(datadir, "server.key")
	}

	// Check if certificate files exist
	if _, err := os.Stat(grpcCertFile); os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("certificate file not found: %s", grpcCertFile)
	}
	if _, err := os.Stat(grpcKeyFile); os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("key file not found: %s", grpcKeyFile)
	}

	// Load TLS credentials
	creds, err := credentials.NewServerTLSFromFile(grpcCertFile, grpcKeyFile)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load TLS credentials: %v", err)
	}

	// Create gRPC server with TLS credentials
	grpcServer := grpc.NewServer(grpc.Creds(creds))

	// Create listener
	grpcLis, err := net.Listen("tcp", serverAddress)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to listen for gRPC poker server: %v", err)
	}

	// Initialize and register the poker server
	pokerServer := server.NewServer(db, logBackend)
	pokerrpc.RegisterLobbyServiceServer(grpcServer, pokerServer)
	pokerrpc.RegisterPokerServiceServer(grpcServer, pokerServer)

	return grpcServer, grpcLis, nil
}

// HandlePM handles incoming PM commands.
func (s *State) HandlePM(ctx context.Context, bot *kit.Bot, pm *types.ReceivedPM) {
	tokens := strings.Fields(pm.Msg.Message)
	if len(tokens) == 0 {
		return
	}

	cmd := strings.ToLower(tokens[0])
	var uid zkidentity.ShortID
	uid.FromBytes(pm.Uid)
	playerID := uid.String()

	switch cmd {
	case "balance":
		balance, err := s.db.GetPlayerBalance(ctx, playerID)
		if err != nil {
			bot.SendPM(ctx, pm.Nick, "Error checking balance: "+err.Error())
			return
		}
		bot.SendPM(ctx, pm.Nick, fmt.Sprintf("Your current balance is: %.8f DCR",
			dcrutil.Amount(balance).ToCoin()))

	case "create":
		s.handleCreateTable(ctx, bot, pm, tokens, playerID)

	case "join":
		s.handleJoinTable(ctx, bot, pm, tokens, playerID)

	case "tables":
		s.handleListTables(ctx, bot, pm)

	case "help":
		s.handleHelp(ctx, bot, pm)

	default:
		bot.SendPM(ctx, pm.Nick, "Unknown command. Type 'help' for available commands.")
	}
}

func (s *State) handleCreateTable(ctx context.Context, bot *kit.Bot, pm *types.ReceivedPM, tokens []string, playerID string) {
	if len(tokens) < 2 {
		bot.SendPM(ctx, pm.Nick, "Usage: create <buy-in amount in DCR> [starting-chips]")
		return
	}

	// Parse buy-in amount
	buyInFloat, err := strconv.ParseFloat(tokens[1], 64)
	if err != nil {
		bot.SendPM(ctx, pm.Nick, "Invalid buy-in amount. Please enter a valid number.")
		return
	}

	buyIn, err := dcrutil.NewAmount(buyInFloat)
	if err != nil {
		bot.SendPM(ctx, pm.Nick, "Invalid DCR amount. Please enter a valid number.")
		return
	}

	// Parse starting chips (optional, default to 1000)
	startingChips := int64(STARTING_CHIPS)
	if len(tokens) >= 3 {
		parsed, err := strconv.ParseInt(tokens[2], 10, 64)
		if err != nil {
			bot.SendPM(ctx, pm.Nick, "Invalid starting chips amount. Please enter a valid number.")
			return
		}
		if parsed <= 0 {
			bot.SendPM(ctx, pm.Nick, "Starting chips must be greater than 0.")
			return
		}
		startingChips = parsed
	}

	// Check player balance
	balance, err := s.db.GetPlayerBalance(ctx, playerID)
	if err != nil {
		bot.SendPM(ctx, pm.Nick, "Error checking balance: "+err.Error())
		return
	}

	if balance < int64(buyIn) {
		bot.SendPM(ctx, pm.Nick, "Insufficient balance for buy-in.")
		return
	}

	// Create new table
	tableID := fmt.Sprintf("table-%d", time.Now().Unix())
	table := poker.NewTable(poker.TableConfig{
		ID:            tableID,
		HostID:        playerID,
		BuyIn:         int64(buyIn), // DCR buy-in amount (in atoms)
		MinPlayers:    2,
		MaxPlayers:    6,
		SmallBlind:    10,            // Fixed chip amount for small blind
		BigBlind:      20,            // Fixed chip amount for big blind
		StartingChips: startingChips, // Poker chips given to each player
		TimeBank:      6 * time.Second,
	})

	// Add creator as user to table
	_, err = table.AddNewUser(playerID, pm.Nick, balance, 0)
	if err != nil {
		bot.SendPM(ctx, pm.Nick, "Error creating table: "+err.Error())
		return
	}

	// Deduct DCR buy-in from creator's account balance
	err = s.db.UpdatePlayerBalance(ctx, playerID, -int64(buyIn), "table buy-in", "created table")
	if err != nil {
		bot.SendPM(ctx, pm.Nick, "Error deducting buy-in: "+err.Error())
		return
	}

	// Add table to state
	s.mu.Lock()
	s.tables[tableID] = table
	s.mu.Unlock()

	bot.SendPM(ctx, pm.Nick, fmt.Sprintf("Table %s created with buy-in of %.8f DCR and %d starting chips. Use 'join %s' to join.",
		tableID, buyIn.ToCoin(), startingChips, tableID))
}

func (s *State) handleJoinTable(ctx context.Context, bot *kit.Bot, pm *types.ReceivedPM, tokens []string, playerID string) {
	if len(tokens) < 2 {
		bot.SendPM(ctx, pm.Nick, "Usage: join <table-id>")
		return
	}

	tableID := tokens[1]
	s.mu.RLock()
	table, exists := s.tables[tableID]
	s.mu.RUnlock()

	if !exists {
		bot.SendPM(ctx, pm.Nick, "Table not found.")
		return
	}

	// Check player DCR balance
	dcrBalance, err := s.db.GetPlayerBalance(ctx, playerID)
	if err != nil {
		bot.SendPM(ctx, pm.Nick, "Error checking balance: "+err.Error())
		return
	}

	config := table.GetConfig()

	// Check if player has enough DCR for the buy-in
	if dcrBalance < config.BuyIn {
		bot.SendPM(ctx, pm.Nick, "Insufficient DCR balance for buy-in.")
		return
	}

	// Find next available seat
	users := table.GetUsers()
	occupiedSeats := make(map[int]bool)
	for _, user := range users {
		occupiedSeats[user.TableSeat] = true
	}

	nextSeat := 0
	for i := 0; i < config.MaxPlayers; i++ {
		if !occupiedSeats[i] {
			nextSeat = i
			break
		}
	}

	// Add user to table
	_, err = table.AddNewUser(playerID, pm.Nick, dcrBalance, nextSeat)
	if err != nil {
		bot.SendPM(ctx, pm.Nick, "Error joining table: "+err.Error())
		return
	}

	// Deduct DCR buy-in from player's account balance
	err = s.db.UpdatePlayerBalance(ctx, playerID, -config.BuyIn, "table buy-in", "joined table")
	if err != nil {
		// If balance update fails, remove player from table
		table.RemoveUser(playerID)
		bot.SendPM(ctx, pm.Nick, "Error deducting buy-in: "+err.Error())
		return
	}

	// Notify all users at the table
	users = table.GetUsers()
	for _, u := range users {
		if u.ID != playerID {
			bot.SendPM(ctx, u.ID, fmt.Sprintf("%s has joined the table.", pm.Nick))
		}
	}

	bot.SendPM(ctx, pm.Nick, fmt.Sprintf("Joined table %s. Current status:\n%s",
		tableID, table.GetStatus()))

	// Check if we can start the game
	if len(users) >= table.GetMinPlayers() {
		err = table.StartGame()
		if err != nil {
			bot.SendPM(ctx, pm.Nick, "Error starting game: "+err.Error())
			return
		}

		// Notify all users
		for _, u := range users {
			bot.SendPM(ctx, u.ID, "Game started!")
		}
	}
}

func (s *State) handleListTables(ctx context.Context, bot *kit.Bot, pm *types.ReceivedPM) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.tables) == 0 {
		bot.SendPM(ctx, pm.Nick, "No active tables.")
		return
	}

	msg := "Active tables:\n"
	for id, table := range s.tables {
		msg += fmt.Sprintf("%s: %d/%d players\n", id, len(table.GetUsers()), table.GetMaxPlayers())
	}
	bot.SendPM(ctx, pm.Nick, msg)
}

func (s *State) handleHelp(ctx context.Context, bot *kit.Bot, pm *types.ReceivedPM) {
	helpMsg := `Available commands:
- balance: Check your current balance
- create <amount> [starting-chips]: Create a new poker table with specified buy-in and optional starting chips (default: 1000)
- join <table-id>: Join an existing poker table
- tables: List all active tables
- help: Show this help message`
	bot.SendPM(ctx, pm.Nick, helpMsg)
}
