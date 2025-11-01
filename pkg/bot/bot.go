package bot

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/companyzero/bisonrelay/clientrpc/types"
	"github.com/companyzero/bisonrelay/zkidentity"
	"github.com/decred/dcrd/dcrutil/v4"
	kit "github.com/vctt94/bisonbotkit"
	"github.com/vctt94/bisonbotkit/logging"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	"github.com/vctt94/pokerbisonrelay/pkg/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

const STARTING_CHIPS = 1000

// State holds the state of the poker bot
type State struct {
	db server.Database
	mu sync.RWMutex
}

// NewState creates a new bot state with the given database
func NewState(db server.Database) *State {
	return &State{
		db: db,
	}
}

// SetupGRPCServer sets up and returns a configured GRPC server with TLS
func SetupGRPCServer(datadir, certFile, keyFile, serverAddress string, db server.Database, logBackend *logging.LogBackend) (*grpc.Server, net.Listener, *server.Server, error) {
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
		return nil, nil, nil, fmt.Errorf("certificate file not found: %s", grpcCertFile)
	}
	if _, err := os.Stat(grpcKeyFile); os.IsNotExist(err) {
		return nil, nil, nil, fmt.Errorf("key file not found: %s", grpcKeyFile)
	}

	// Load TLS credentials
	creds, err := credentials.NewServerTLSFromFile(grpcCertFile, grpcKeyFile)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load TLS credentials: %v", err)
	}

	// Create gRPC server with TLS credentials and production-avg keepalives
	grpcServer := grpc.NewServer(
		grpc.Creds(creds),
		grpc.MaxConcurrentStreams(1000),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:                  1 * time.Minute,
			Timeout:               20 * time.Second,
			MaxConnectionIdle:     5 * time.Minute,
			MaxConnectionAge:      2 * time.Hour,
			MaxConnectionAgeGrace: 5 * time.Minute,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             30 * time.Second,
			PermitWithoutStream: true,
		}),
	)

	// Create listener
	grpcLis, err := net.Listen("tcp", serverAddress)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to listen for gRPC poker server: %v", err)
	}

	// Initialize and register the poker server
	pokerServer := server.NewServer(db, logBackend)
	pokerrpc.RegisterLobbyServiceServer(grpcServer, pokerServer)
	pokerrpc.RegisterPokerServiceServer(grpcServer, pokerServer)

	return grpcServer, grpcLis, pokerServer, nil
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

	case "help":
		s.handleHelp(ctx, bot, pm)

	default:
		bot.SendPM(ctx, pm.Nick, "Unknown command. Type 'help' for available commands.")
	}
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
