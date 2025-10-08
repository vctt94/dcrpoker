package server

import (
	"sync"
	"sync/atomic"

	"github.com/decred/slog"
	"github.com/vctt94/bisonbotkit/logging"
	"github.com/vctt94/pokerbisonrelay/pkg/poker"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
)

// NotificationStream represents a client's notification stream
type NotificationStream struct {
	playerID string
	stream   pokerrpc.LobbyService_StartNotificationStreamServer
	done     chan struct{}
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

	// Event-driven architecture components
	eventProcessor *EventProcessor
}

// NewServer creates a new poker server
func NewServer(db Database, logBackend *logging.LogBackend) *Server {
	server := &Server{
		log:        logBackend.Logger("SERVER"),
		logBackend: logBackend,
		db:         db,
	}

	// Load persisted tables on startup
	err := server.loadAllTables()
	if err != nil {
		server.log.Errorf("Failed to load persisted tables: %v", err)
	}

	// Initialize event processor for deadlock-free architecture
	server.eventProcessor = NewEventProcessor(server, 1000, 3) // queue size: 1000, workers: 3
	server.eventProcessor.Start()
	return server
}

// Stop gracefully stops the server
func (s *Server) Stop() {
	if s.eventProcessor != nil {
		s.eventProcessor.Stop()
	}

	// Close all tables properly to prevent goroutine leaks
	tables := s.getAllTables()
	for _, table := range tables {
		table.Close()
	}

	// Wait for any in-flight asynchronous saves to complete before returning.
	s.saveWg.Wait()
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

func (s *Server) getAllTables() []*poker.Table {
	tableRefs := make([]*poker.Table, 0)
	s.tables.Range(func(_, value any) bool {
		if t, ok := value.(*poker.Table); ok && t != nil {
			tableRefs = append(tableRefs, t)
		}
		return true
	})
	return tableRefs
}
