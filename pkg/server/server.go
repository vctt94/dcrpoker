package server

import (
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/decred/slog"
	"github.com/vctt94/bisonbotkit/logging"
	"github.com/vctt94/pokerbisonrelay/pkg/poker"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

// NotificationStream represents a client's notification stream
type NotificationStream struct {
	playerID string
	stream   pokerrpc.LobbyService_StartNotificationStreamServer
	done     chan struct{}
}

type ServerConfig struct {
	Datadir string
	// Additional specific fields
	GRPCHost     string
	GRPCPort     string
	GRPCCertPath string
	GRPCKeyPath  string

	metricsAddr string

	// dcrd connectivity (optional)
	DcrdHost string
	DcrdCert string
	DcrdUser string
	DcrdPass string

	// Schnorr adaptor secret (32-byte hex string)
	AdaptorSecret string

	// Network specifies the Decred network: "mainnet" or "testnet" (defaults to "testnet")
	Network string

	LogBackend *logging.LogBackend

	DB *Database
}

// bucket manages game stream connections for a specific poker table.
// It serves as a container for all active player streams connected to a table,
// allowing efficient broadcasting of game state updates to all players at that table.
// The bucket is automatically created when the first player connects to a table
// and is removed when the last player disconnects.
type bucket struct {
	streams sync.Map     // playerID -> pokerrpc.PokerService_StartGameStreamServer
	count   atomic.Int32 // active players in this table
}

// Server implements both PokerService and LobbyService
type Server struct {
	grpcServer *grpc.Server
	grpcLis    net.Listener
	pokerrpc.UnimplementedPokerServiceServer
	pokerrpc.UnimplementedLobbyServiceServer
	log        slog.Logger
	logBackend *logging.LogBackend
	db         Database
	// Concurrent registry of tables to avoid coarse-grained server locking.
	tables sync.Map // key: string (tableID) -> value: *poker.Table

	// Notification streaming
	notificationStreams sync.Map // key: playerID string -> *NotificationStream

	// Game streaming
	// Maps tableID to bucket containing all active player streams for that table
	// Each bucket manages streams for players connected to a specific table
	gameStreams sync.Map // key: tableID string -> value: *bucket

	// Table state saving synchronization
	// key: tableID string -> *sync.Mutex (serialize saves per table)
	saveMutexes sync.Map

	// Broadcast serialization per table (notifications + game state streams)
	// key: tableID string -> *sync.Mutex
	broadcastMutexes sync.Map

	// Notification send serialization per player
	// key: playerID string -> *sync.Mutex
	notifSendMutexes sync.Map

	// WaitGroup to ensure all async save goroutines complete before Shutdown
	saveWg sync.WaitGroup

	// WaitGroup to ensure all active stream handlers complete before Shutdown
	streamHandlersWg sync.WaitGroup

	// Event-driven architecture components
	eventProcessor *EventProcessor
}

// NewServer creates a new poker server
func NewServer(cfg ServerConfig) (*Server, error) {
	// Initialize database
	db, err := NewDatabase(filepath.Join(cfg.Datadir, "poker.db"))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize and start the gRPC poker server
	grpcServer, grpcLis, err := SetupGRPCServer(cfg.Datadir, cfg.GRPCCertPath, cfg.GRPCKeyPath, cfg.GRPCHost+":"+cfg.GRPCPort, db, cfg.LogBackend)
	if err != nil {
		return nil, fmt.Errorf("failed to setup gRPC server: %v", err)
	}

	// Start gRPC server
	errCh := make(chan error, 1)
	go func() {
		cfg.LogBackend.Logger("SERVER").Infof("Starting gRPC poker server on %s", cfg.GRPCHost+":"+cfg.GRPCPort)
		errCh <- grpcServer.Serve(grpcLis)
	}()
	defer grpcServer.Stop() // Ensure gRPC server is stopped on exit
	pokerServer := &Server{
		grpcServer: grpcServer,
		grpcLis:    grpcLis,
		log:        cfg.LogBackend.Logger("SERVER"),
		logBackend: cfg.LogBackend,
		db:         db,
	}

	// Initialize and register the poker server
	pokerrpc.RegisterLobbyServiceServer(grpcServer, pokerServer)
	pokerrpc.RegisterPokerServiceServer(grpcServer, pokerServer)
	// Initialize event processor for deadlock-free architecture
	pokerServer.eventProcessor = NewEventProcessor(pokerServer, 1000, 3) // queue size: 1000, workers: 3
	pokerServer.eventProcessor.Start()

	// Load persisted tables on startup AFTER the event processor is fully
	// initialized.
	if err := pokerServer.loadAllTables(); err != nil {
		pokerServer.log.Errorf("Failed to load persisted tables: %v", err)
	}

	return pokerServer, nil
}

// Load config function
func LoadServerConfig(datadir, filename string) (*ServerConfig, error) {
	cfg, err := loadServerConf(datadir, filename)
	if err != nil {
		return nil, fmt.Errorf("failed to load server config: %v", err)
	}
	logBackend, err := logging.NewLogBackend(logging.LogConfig{
		LogFile:        filepath.Join(datadir, "log", "server.log"),
		DebugLevel:     cfg.Debug,
		MaxLogFiles:    cfg.MaxLogFiles,
		MaxBufferLines: cfg.MaxBufferLines,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create log backend: %v", err)
	}

	// Create the combined config
	serverCfg := &ServerConfig{
		Datadir:       cfg.Datadir,
		GRPCHost:      cfg.GRPCHost,
		GRPCPort:      cfg.GRPCPort,
		GRPCCertPath:  cfg.GRPCCertPath,
		GRPCKeyPath:   cfg.GRPCKeyPath,
		DcrdHost:      cfg.DcrdHost,
		DcrdCert:      cfg.DcrdCert,
		DcrdUser:      cfg.DcrdUser,
		DcrdPass:      cfg.DcrdPass,
		AdaptorSecret: cfg.AdaptorSecret,
		Network:       cfg.Network,
		LogBackend:    logBackend,
	}

	// Validate adaptor secret: must be present and 32 bytes of hex (64 chars)
	if cfg.AdaptorSecret == "" {
		return nil, fmt.Errorf("missing adaptorsecret in server config")
	}
	sb, err := hex.DecodeString(cfg.AdaptorSecret)
	if err != nil || len(sb) != 32 {
		return nil, fmt.Errorf("invalid adaptorsecret: expected 64 hex chars (32 bytes)")
	}

	return serverCfg, nil
}

// SetupGRPCServer sets up and returns a configured GRPC server with TLS
func SetupGRPCServer(datadir, certFile, keyFile, serverAddress string, db Database, logBackend *logging.LogBackend) (*grpc.Server, net.Listener, error) {
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
		return nil, nil, fmt.Errorf("failed to listen for gRPC poker server: %v", err)
	}

	return grpcServer, grpcLis, nil
}

// NewTestServer creates a Server for testing with the provided database and log backend.
// This is a test helper that constructs a Server without gRPC setup.
// It can be used by e2e tests that need to set up their own gRPC server.
func NewTestServer(db Database, logBackend *logging.LogBackend) (*Server, error) {
	pokerServer := &Server{
		log:        logBackend.Logger("SERVER"),
		logBackend: logBackend,
		db:         db,
	}

	// Initialize event processor for deadlock-free architecture
	pokerServer.eventProcessor = NewEventProcessor(pokerServer, 1000, 3) // queue size: 1000, workers: 3
	pokerServer.eventProcessor.Start()

	// Load persisted tables on startup AFTER the event processor is fully
	// initialized.
	if err := pokerServer.loadAllTables(); err != nil {
		pokerServer.log.Errorf("Failed to load persisted tables: %v", err)
	}

	return pokerServer, nil
}

// Stop gracefully stops the server
func (s *Server) Stop() {
	// First, wait for all active stream handlers to finish so that no new
	// disconnect handlers can run concurrently with shutdown.
	s.streamHandlersWg.Wait()

	// Stop the event processor so workers stop reading from the queue and
	// publishing new events while tables/games are being closed.
	if s.eventProcessor != nil {
		s.eventProcessor.Stop()
	}

	// Close all tables properly to prevent goroutine leaks. This cascades
	// into Game and Player shutdown, including state machine Stop() calls.
	tables := s.getAllTables()
	for _, table := range tables {
		table.Close()
	}

	// Wait for any in-flight asynchronous saves to complete before returning.
	s.saveWg.Wait()
	if s.grpcServer != nil {
		s.grpcServer.Stop()
	}
	if s.grpcLis != nil {
		s.grpcLis.Close()
	}
}

// getTable retrieves a table by ID from the registry.
func (s *Server) getTable(tableID string) (*poker.Table, bool) {
	if v, ok := s.tables.Load(tableID); ok {
		if t, ok2 := v.(*poker.Table); ok2 && t != nil {
			return t, true
		}
	}
	return nil, false
}

// GetAllTables returns all tables from the server registry.
func (s *Server) GetAllTables() []*poker.Table {
	tableRefs := make([]*poker.Table, 0)
	s.tables.Range(func(_, value any) bool {
		if t, ok := value.(*poker.Table); ok && t != nil {
			tableRefs = append(tableRefs, t)
		}
		return true
	})
	return tableRefs
}

func (s *Server) getAllTables() []*poker.Table {
	return s.GetAllTables()
}

// GetAllInGameUsers returns a map of tableID -> set of playerIDs that have active game streams.
// This provides the authoritative source of in-game users based on runtime state.
func (s *Server) GetAllInGameUsers() map[string]map[string]bool {
	result := make(map[string]map[string]bool)
	s.gameStreams.Range(func(tableIDAny, bucketAny any) bool {
		tableID := tableIDAny.(string)
		b := bucketAny.(*bucket)
		if b == nil {
			return true
		}
		result[tableID] = make(map[string]bool)
		b.streams.Range(func(playerIDAny, streamAny any) bool {
			playerID := playerIDAny.(string)
			result[tableID][playerID] = true
			return true
		})
		return true
	})
	return result
}

// GetAllOnlineUsers returns a set of all playerIDs that have active notification streams.
// This provides the authoritative source of online users (regardless of table membership).
func (s *Server) GetAllOnlineUsers() map[string]bool {
	result := make(map[string]bool)
	s.notificationStreams.Range(func(playerIDAny, streamAny any) bool {
		playerID := playerIDAny.(string)
		result[playerID] = true
		return true
	})
	return result
}

// GetInLobbyAndInGameUsers returns sets of playerIDs categorized by their status:
// - inLobby: Users with game streams but no active game (game not started)
// - inGame: Users with game streams in active games (game started)
func (s *Server) GetInLobbyAndInGameUsers() (inLobby map[string]bool, inGame map[string]bool) {
	inLobby = make(map[string]bool)
	inGame = make(map[string]bool)

	inGameUsers := s.GetAllInGameUsers()
	tables := s.GetAllTables()

	// Build index of tableID -> table for quick lookup
	tableMap := make(map[string]*poker.Table)
	for _, t := range tables {
		tableMap[t.GetConfig().ID] = t
	}

	// Categorize users based on whether their table has an active game
	for tableID, playerIDs := range inGameUsers {
		table := tableMap[tableID]
		if table == nil {
			continue
		}

		// Check if game has actually started (game object only exists after PRE_FLOP is reached)
		gameActive := table.IsGameStarted()
		for playerID := range playerIDs {
			if gameActive {
				inGame[playerID] = true
			} else {
				inLobby[playerID] = true
			}
		}
	}

	return inLobby, inGame
}

// EventQueueDepth returns the current depth of the server event queue.
func (s *Server) EventQueueDepth() int {
	if s.eventProcessor == nil || s.eventProcessor.queue == nil {
		return 0
	}
	return len(s.eventProcessor.queue)
}

// EventQueueCapacity returns the capacity of the server event queue buffer.
func (s *Server) EventQueueCapacity() int {
	if s.eventProcessor == nil || s.eventProcessor.queue == nil {
		return 0
	}
	return cap(s.eventProcessor.queue)
}
