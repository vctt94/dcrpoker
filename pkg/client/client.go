package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/decred/slog"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/companyzero/bisonrelay/clientrpc/types"
	"github.com/companyzero/bisonrelay/zkidentity"
	"github.com/vctt94/bisonbotkit/botclient"
	"github.com/vctt94/bisonbotkit/logging"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	pokerutils "github.com/vctt94/pokerbisonrelay/pkg/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Message types for UI communication
type GameUpdateMsg *pokerrpc.GameUpdate

// PokerClient represents a poker client with notification handling
type PokerClient struct {
	sync.RWMutex
	ID           zkidentity.ShortID
	DataDir      string
	BRClient     *botclient.BotClient
	LobbyService pokerrpc.LobbyServiceClient
	PokerService pokerrpc.PokerServiceClient
	conn         *grpc.ClientConn
	IsReady      bool
	BetAmt       int64 // bet amount in atoms
	tableID      string
	cfg          *AppConfig
	ntfns        *NotificationManager
	log          slog.Logger
	logBackend   *logging.LogBackend
	notifier     pokerrpc.LobbyService_StartNotificationStreamClient

	// helper channels for pokerctl
	UpdatesCh       chan tea.Msg
	ErrorsCh        chan error
	NotificationsCh chan *pokerrpc.Notification

	// Game streaming
	gameStream   pokerrpc.PokerService_StartGameStreamClient
	gameStreamMu sync.Mutex

	// For reconnection handling
	ctx          context.Context
	cancelFunc   context.CancelFunc
	reconnecting bool
	reconnectMu  sync.Mutex
}

// NewPokerClient creates a new poker client with notification support
func NewPokerClient(ctx context.Context, cfg *AppConfig) (*PokerClient, error) {
	// Validate that notifications are properly initialized
	if cfg.Notifications == nil {
		// initialize notification manager with NewNotificationManager
		return nil, fmt.Errorf("notification manager cannot be nil - client startup aborted")
	}

	// Create the base client
	client, err := newClient(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create base client: %v", err)
	}

	ctx, cancel := context.WithCancel(ctx)

	pc := &PokerClient{
		ID:           client.ID,
		DataDir:      client.DataDir,
		BRClient:     client.BRClient,
		LobbyService: client.LobbyService,
		PokerService: client.PokerService,
		conn:         client.conn,
		cfg:          cfg,
		ntfns:        cfg.Notifications,
		log:          client.log,
		logBackend:   client.logBackend,
		UpdatesCh:    make(chan tea.Msg, 100),
		ErrorsCh:     make(chan error, 10),
		ctx:          ctx,
		cancelFunc:   cancel,
	}

	// Final validation that client is properly initialized
	if err := pc.validate(); err != nil {
		return nil, fmt.Errorf("client validation failed: %v", err)
	}

	return pc, nil
}

// newBaseClient creates a basic client without notification support (internal use)
func newClient(ctx context.Context, cfg *AppConfig) (*PokerClient, error) {
	if cfg == nil {
		return nil, fmt.Errorf("cfg is nil")
	}
	// Ensure datadir exists
	if err := pokerutils.EnsureDataDirExists(cfg.DataDir); err != nil {
		return nil, fmt.Errorf("failed to create datadir: %v", err)
	}

	// Convert to BisonRelay config (unless offline)
	var brClient *botclient.BotClient
	var log slog.Logger
	var logBackend *logging.LogBackend
	var clientID zkidentity.ShortID
	if cfg.Offline {
		if cfg.PlayerID == "" {
			return nil, fmt.Errorf("clientID is required when running offline")
		}
		// generate a random client ID

		// Minimal logging backend when offline
		lb, _ := logging.NewLogBackend(logging.LogConfig{DebugLevel: "info"})
		log = lb.Logger("PokerClient")
		logBackend = lb
	} else {
		// connect to BisonRelay
		clientConfig := cfg.ToBisonRelayConfig()

		// Initialize BisonRelay client
		bc, err := botclient.NewClient(clientConfig)
		if err != nil {
			fmt.Printf("Failed to create bot client: %v\n", err)
			os.Exit(1)
		}
		if bc == nil {
			return nil, fmt.Errorf("bot client is nil")
		}
		brClient = bc
		// Start the RPC client in a goroutine if brClient was created successfully
		go brClient.RunRPC(ctx)
		// Get the client ID
		var publicIdentity types.PublicIdentity
		err = brClient.Chat.UserPublicIdentity(ctx, &types.PublicIdentityReq{}, &publicIdentity)
		if err != nil {
			return nil, fmt.Errorf("Failed to get user public identity: %v", err)
		}

		clientID.FromBytes(publicIdentity.Identity)

		log = brClient.LogBackend.Logger("PokerClient")
		logBackend = brClient.LogBackend
	}

	client := &PokerClient{
		ID:         clientID,
		DataDir:    cfg.DataDir,
		BRClient:   brClient,
		log:        log,
		logBackend: logBackend,
		cfg:        cfg,
	}

	log.Debugf("Using client ID: %s", client.ID)

	// Connect to the poker server
	if err := client.connectToPokerServer(ctx, cfg.GRPCHost); err != nil {
		return nil, fmt.Errorf("failed to connect to poker server: %v", err)
	}

	// Initialize account
	if err := client.initializeAccount(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize account: %v", err)
	}

	return client, nil
}

// connectToPokerServer establishes gRPC connection to the poker server
func (pc *PokerClient) connectToPokerServer(ctx context.Context, grpcHost string) error {
	var dialOpts []grpc.DialOption

	if pc.cfg == nil {
		return fmt.Errorf("cfg is nil")
	}

	// Check if GRPCHost and GRPCPort are properly configured
	if pc.cfg.GRPCHost == "" {
		return fmt.Errorf("GRPCHost is not configured")
	}
	if pc.cfg.GRPCPort == "" {
		return fmt.Errorf("GRPCPort is not configured")
	}

	// TLS or insecure
	if pc.cfg.Insecure {
		// Note: WithInsecure is deprecated but fine for tests; we keep local usage to integration paths
		dialOpts = append(dialOpts, grpc.WithInsecure())
	} else {
		// Use TLS
		grpcServerCertPath := pc.cfg.GRPCServerCert
		if grpcServerCertPath == "" {
			return fmt.Errorf("GRPCServerCert not configured and insecure mode disabled")
		}

		// Require that the server certificate exists; do not auto-create.
		if _, err := os.Stat(grpcServerCertPath); os.IsNotExist(err) {
			return fmt.Errorf("server certificate not found at %s", grpcServerCertPath)
		}

		// Load the server certificate
		pemServerCA, err := os.ReadFile(grpcServerCertPath)
		if err != nil {
			return fmt.Errorf("failed to read server certificate: %v", err)
		}

		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(pemServerCA) {
			return fmt.Errorf("failed to add server certificate to pool")
		}

		// Use GRPCHost for TLS ServerName strictly; do not fallback.
		serverName := pc.cfg.GRPCHost
		if serverName == "" {
			return fmt.Errorf("GRPCHost not configured for TLS ServerName")
		}

		// Create the TLS credentials with ServerName
		tlsConfig := &tls.Config{
			RootCAs:    certPool,
			ServerName: serverName,
		}

		creds := credentials.NewTLS(tlsConfig)
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(creds))
	}

	// Construct server address from GRPCHost and GRPCPort
	serverAddr := fmt.Sprintf("%s:%s", pc.cfg.GRPCHost, pc.cfg.GRPCPort)

	// Create the client connection
	conn, err := grpc.Dial(serverAddr, dialOpts...)
	if err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}

	pc.conn = conn
	pc.LobbyService = pokerrpc.NewLobbyServiceClient(conn)
	pc.PokerService = pokerrpc.NewPokerServiceClient(conn)

	return nil
}

// initializeAccount ensures the client has an account with the server
func (pc *PokerClient) initializeAccount(ctx context.Context) error {
	// Make sure we have an account
	balanceResp, err := pc.LobbyService.GetBalance(ctx, &pokerrpc.GetBalanceRequest{
		PlayerId: pc.ID.String(),
	})
	if err != nil {
		// Initialize account with deposit
		updateResp, err := pc.LobbyService.UpdateBalance(ctx, &pokerrpc.UpdateBalanceRequest{
			PlayerId:    pc.ID.String(),
			Amount:      1000,
			Description: "Initial deposit",
		})
		if err != nil {
			return fmt.Errorf("could not initialize balance: %v", err)
		}
		pc.log.Debugf("Initialized DCR account balance: %d", updateResp.NewBalance)
		return nil
	}

	pc.log.Debugf("Current DCR account balance: %d", balanceResp.Balance)
	return nil
}

// reconnect attempts to reconnect to the server and restart the notification stream
func (pc *PokerClient) reconnect() error {
	pc.reconnectMu.Lock()
	defer pc.reconnectMu.Unlock()

	if pc.reconnecting {
		return nil // Already reconnecting
	}

	pc.reconnecting = true
	defer func() { pc.reconnecting = false }()

	pc.log.Info("attempting to reconnect...")

	// Close existing connection
	if pc.conn != nil {
		pc.conn.Close()
	}

	// Create new context for reconnection
	ctx, cancel := context.WithCancel(pc.ctx)
	pc.cancelFunc = cancel

	client, err := newClient(ctx, pc.cfg)
	if err != nil {
		return fmt.Errorf("failed to reconnect client: %v", err)
	}

	// Update client fields
	pc.LobbyService = client.LobbyService
	pc.PokerService = client.PokerService
	pc.conn = client.conn

	// Restart notification stream
	if err := pc.StartNotificationStream(ctx); err != nil {
		return fmt.Errorf("failed to restart notification stream: %v", err)
	}

	pc.log.Info("successfully reconnected")
	return nil
}

// GetCurrentTableID returns the current table ID
func (pc *PokerClient) GetCurrentTableID() string {
	pc.RLock()
	defer pc.RUnlock()
	return pc.tableID
}

// SetCurrentTableID sets the current table ID without making any RPC calls.
// This is useful for stateless CLI invocations that need to target a table by ID.
func (pc *PokerClient) SetCurrentTableID(tableID string) {
	pc.Lock()
	pc.tableID = tableID
	pc.Unlock()
}

// Close closes the poker client and its connections
func (pc *PokerClient) Close() error {
	if pc.cancelFunc != nil {
		pc.cancelFunc()
	}

	// Stop game stream if active
	pc.stopGameStream()

	if pc.conn != nil {
		return pc.conn.Close()
	}
	return nil
}

// stopGameStream stops the current game stream
func (pc *PokerClient) stopGameStream() {
	pc.gameStreamMu.Lock()
	defer pc.gameStreamMu.Unlock()

	if pc.gameStream != nil {
		pc.gameStream.CloseSend()
		pc.gameStream = nil
		pc.log.Info("Stopped game stream")
	}
}

// handleGameStreamUpdates processes incoming game updates from the stream
func (pc *PokerClient) handleGameStreamUpdates(ctx context.Context) {
	defer func() {
		pc.gameStreamMu.Lock()
		pc.gameStream = nil
		pc.gameStreamMu.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			pc.gameStreamMu.Lock()
			stream := pc.gameStream
			pc.gameStreamMu.Unlock()

			if stream == nil {
				return
			}

			update, err := stream.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) || strings.Contains(err.Error(), "transport is closing") ||
					strings.Contains(err.Error(), "connection is being forcefully terminated") {
					pc.log.Info("Game stream closed")
					return
				}

				pc.log.Errorf("Game stream error: %v", err)
				pc.ErrorsCh <- fmt.Errorf("game stream error: %v", err)
				return
			}

			// Convert to UI message type and send to updates channel
			select {
			case pc.UpdatesCh <- GameUpdateMsg(update):
			case <-ctx.Done():
				return
			default:
				// Channel is full, drop the update
				pc.log.Warn("Updates channel full, dropping game update")
			}
		}
	}
}

// Validate checks if the PokerClient is properly initialized and ready to use
func (pc *PokerClient) validate() error {
	if pc == nil {
		return fmt.Errorf("poker client is nil")
	}
	if pc.log == nil {
		return fmt.Errorf("logger is not initialized")
	}
	if pc.logBackend == nil {
		return fmt.Errorf("log backend is not initialized")
	}
	if pc.ntfns == nil {
		return fmt.Errorf("notification manager is not initialized")
	}
	if pc.LobbyService == nil {
		return fmt.Errorf("lobby service is not initialized")
	}
	if pc.PokerService == nil {
		return fmt.Errorf("poker service is not initialized")
	}
	if pc.ID.String() == "" {
		return fmt.Errorf("client ID is not set")
	}
	if pc.UpdatesCh == nil {
		return fmt.Errorf("updates channel is not initialized")
	}
	if pc.ErrorsCh == nil {
		return fmt.Errorf("errors channel is not initialized")
	}
	return nil
}
