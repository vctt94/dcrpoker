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
	"time"

	"github.com/decred/slog"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/companyzero/bisonrelay/zkidentity"
	"github.com/vctt94/bisonbotkit/config"
	"github.com/vctt94/bisonbotkit/logging"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	pokerutils "github.com/vctt94/pokerbisonrelay/pkg/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// LoadClientConfig loads the client config and creates ClientConfig with LogBackend
func LoadClientConfig(datadir, filename string) (*ClientConfig, error) {
	cfg, err := LoadClientConf(datadir, filename)
	if err != nil {
		return nil, fmt.Errorf("failed to load client config: %v", err)
	}
	logBackend, err := logging.NewLogBackend(logging.LogConfig{
		LogFile:        cfg.LogFile,
		DebugLevel:     cfg.Debug,
		MaxLogFiles:    cfg.MaxLogFiles,
		MaxBufferLines: cfg.MaxBufferLines,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create log backend: %v", err)
	}

	// Create the combined config
	clientCfg := &ClientConfig{
		Datadir:        cfg.Datadir,
		GRPCHost:       cfg.GRPCHost,
		GRPCPort:       cfg.GRPCPort,
		GRPCCertPath:   cfg.GRPCCertPath,
		PayoutAddress:  cfg.PayoutAddress,
		LogFile:        cfg.LogFile,
		Debug:          cfg.Debug,
		MaxLogFiles:    cfg.MaxLogFiles,
		MaxBufferLines: cfg.MaxBufferLines,
		LogBackend:     logBackend,
		Notifications:  NewNotificationManager(),
		ConfFileName:   filename,
	}

	return clientCfg, nil
}

// ClientConfig is the client configuration with LogBackend and Notifications
type ClientConfig struct {
	Datadir        string
	GRPCHost       string
	GRPCPort       string
	GRPCCertPath   string
	PayoutAddress  string
	LogFile        string
	Debug          string
	MaxLogFiles    int
	MaxBufferLines int
	LogBackend     *logging.LogBackend
	Notifications  *NotificationManager
	PlayerID       string
	ConfFileName   string
}

// PokerClient represents a poker client with notification handling
type PokerClient struct {
	sync.RWMutex
	ID           zkidentity.ShortID
	DataDir      string
	LobbyService pokerrpc.LobbyServiceClient
	PokerService pokerrpc.PokerServiceClient
	conn         *grpc.ClientConn
	IsReady      bool
	BetAmt       int64 // bet amount in atoms
	tableID      string
	cfg          *ClientConfig
	ntfns        *NotificationManager
	log          slog.Logger
	logBackend   *logging.LogBackend
	notifier     pokerrpc.LobbyService_StartNotificationStreamClient

	// helper channels for pokerctl
	UpdatesCh       chan tea.Msg
	ErrorsCh        chan error
	NotificationsCh chan *pokerrpc.Notification

	// Game streaming
	gameStream       pokerrpc.PokerService_StartGameStreamClient
	gameStreamMu     sync.Mutex
	gameStreamCtx    context.Context
	gameStreamCancel context.CancelFunc
	gameStreamTable  string

	// For reconnection handling
	ctx          context.Context
	cancelFunc   context.CancelFunc
	reconnecting bool
	reconnectMu  sync.Mutex

	// Connection state tracking
	isConnected         bool
	gameStreamConnected bool
	lastConnectTime     time.Time
	lastDisconnectTime  time.Time

	// Loop coordination
	ntfnLoopMu      sync.Mutex
	ntfnLoopRunning bool
}

// NewPokerClient creates a new poker client with notification support
func NewPokerClient(ctx context.Context, cfg *ClientConfig) (*PokerClient, error) {
	if err := ensureNotifications(cfg); err != nil {
		return nil, err
	}

	baseClient, err := newClient(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create base client: %v", err)
	}

	return wrapPokerClient(ctx, cfg, baseClient)
}

// NewPokerClientWithDialOptions creates a new poker client using the provided
// dial target and options. This is intended for tests that use in-memory gRPC
// servers.
func NewPokerClientWithDialOptions(ctx context.Context, cfg *ClientConfig, dialTarget string, dialOpts ...grpc.DialOption) (*PokerClient, error) {
	if err := ensureNotifications(cfg); err != nil {
		return nil, err
	}

	baseClient, err := newClientWithDialOptions(ctx, cfg, dialTarget, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create base client: %v", err)
	}

	return wrapPokerClient(ctx, cfg, baseClient)
}

func ensureNotifications(cfg *ClientConfig) error {
	if cfg.Notifications == nil {
		return fmt.Errorf("notification manager cannot be nil - client startup aborted")
	}
	return nil
}

func wrapPokerClient(ctx context.Context, cfg *ClientConfig, base *PokerClient) (*PokerClient, error) {
	ctx, cancel := context.WithCancel(ctx)

	pc := &PokerClient{
		ID:           base.ID,
		DataDir:      base.DataDir,
		LobbyService: base.LobbyService,
		PokerService: base.PokerService,
		conn:         base.conn,
		cfg:          cfg,
		ntfns:        cfg.Notifications,
		log:          base.log,
		logBackend:   base.logBackend,
		UpdatesCh:    make(chan tea.Msg, 100),
		ErrorsCh:     make(chan error, 10),
		ctx:          ctx,
		cancelFunc:   cancel,
	}

	if err := pc.validate(); err != nil {
		return nil, fmt.Errorf("client validation failed: %v", err)
	}

	return pc, nil
}

// newBaseClient creates a basic client without notification support (internal use)
func newClient(ctx context.Context, cfg *ClientConfig) (*PokerClient, error) {
	client, err := buildBaseClient(cfg)
	if err != nil {
		return nil, err
	}

	serverAddr := fmt.Sprintf("%s:%s", cfg.GRPCHost, cfg.GRPCPort)
	dialOpts, err := client.defaultDialOptions()
	if err != nil {
		return nil, fmt.Errorf("failed to get default dial options: %v", err)
	}
	if err := client.dialWithOptions(serverAddr, dialOpts...); err != nil {
		return nil, fmt.Errorf("failed to dial with options: %v", err)
	}

	if err := client.initializeAccount(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize account: %v", err)
	}

	return client, nil
}

func newClientWithDialOptions(ctx context.Context, cfg *ClientConfig, dialTarget string, dialOpts ...grpc.DialOption) (*PokerClient, error) {
	client, err := buildBaseClient(cfg)
	if err != nil {
		return nil, err
	}

	if dialTarget == "" {
		return nil, fmt.Errorf("dial target is not configured")
	}

	if err := client.dialWithOptions(dialTarget, dialOpts...); err != nil {
		return nil, fmt.Errorf("failed to dial with options: %v", err)
	}

	if err := client.initializeAccount(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize account: %v", err)
	}

	return client, nil
}

func buildBaseClient(cfg *ClientConfig) (*PokerClient, error) {
	if cfg == nil {
		return nil, fmt.Errorf("cfg is nil")
	}
	// Ensure datadir exists
	if err := pokerutils.EnsureDataDirExists(cfg.Datadir); err != nil {
		return nil, fmt.Errorf("failed to create datadir: %v", err)
	}

	// Convert to BisonRelay config (unless offline)
	var log slog.Logger
	var logBackend *logging.LogBackend
	var clientID zkidentity.ShortID

	clientConfig := &config.ClientConfig{
		DataDir:        cfg.Datadir,
		LogFile:        cfg.LogFile,
		Debug:          cfg.Debug,
		MaxLogFiles:    cfg.MaxLogFiles,
		MaxBufferLines: cfg.MaxBufferLines,
		ExtraConfig:    make(map[string]string),
	}
	// Set grpchost and grpcport in ExtraConfig for backward compatibility
	if cfg.GRPCHost != "" {
		clientConfig.SetString("grpchost", cfg.GRPCHost)
	}
	if cfg.GRPCPort != "" {
		clientConfig.SetString("grpcport", cfg.GRPCPort)
	}

	// Load or create persistent user ID from seed key
	userIDStr, err := getUserIDFromDir(cfg.Datadir)
	if err != nil {
		return nil, fmt.Errorf("failed to get user ID: %v", err)
	}
	if err := clientID.FromString(userIDStr); err != nil {
		return nil, fmt.Errorf("failed to parse user ID: %v", err)
	}

	// Use config's log backend for now
	log = cfg.LogBackend.Logger("PokerClient")
	logBackend = cfg.LogBackend

	client := &PokerClient{
		ID:         clientID,
		DataDir:    cfg.Datadir,
		log:        log,
		logBackend: logBackend,
		cfg:        cfg,
	}

	log.Debugf("Using client ID: %s", client.ID)

	return client, nil
}

func (pc *PokerClient) defaultDialOptions() ([]grpc.DialOption, error) {
	var dialOpts []grpc.DialOption

	// Check if GRPCHost and GRPCPort are properly configured
	if pc.cfg.GRPCHost == "" {
		return nil, fmt.Errorf("GRPCHost is not configured")
	}
	if pc.cfg.GRPCPort == "" {
		return nil, fmt.Errorf("GRPCPort is not configured")
	}

	grpcServerCertPath := pc.cfg.GRPCCertPath
	if grpcServerCertPath == "" {
		// Tests and local dev may run the server without TLS; allow insecure in that case.
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
		return dialOpts, nil
	}

	// Require that the server certificate exists; do not auto-create.
	if _, err := os.Stat(grpcServerCertPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("server certificate not found at %s", grpcServerCertPath)
	}

	// Load the server certificate
	pemServerCA, err := os.ReadFile(grpcServerCertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read server certificate: %v", err)
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(pemServerCA) {
		return nil, fmt.Errorf("failed to add server certificate to pool")
	}

	// Use GRPCHost for TLS ServerName strictly; do not fallback.
	serverName := pc.cfg.GRPCHost
	if serverName == "" {
		return nil, fmt.Errorf("GRPCHost not configured for TLS ServerName")
	}

	// Create the TLS credentials with ServerName
	tlsConfig := &tls.Config{
		RootCAs:    certPool,
		ServerName: serverName,
	}

	creds := credentials.NewTLS(tlsConfig)
	dialOpts = append(dialOpts, grpc.WithTransportCredentials(creds))

	return dialOpts, nil
}

func (pc *PokerClient) dialWithOptions(dialTarget string, dialOpts ...grpc.DialOption) error {
	// Copy so we do not mutate provided dial options.
	dialOpts = append([]grpc.DialOption{}, dialOpts...)

	// Enable client-side keepalives so we detect idle/half-open connections
	// after the host sleeps and can restart the streams automatically.
	dialOpts = append(dialOpts, grpc.WithKeepaliveParams(keepalive.ClientParameters{
		Time:                10 * time.Second,
		Timeout:             7 * time.Second,
		PermitWithoutStream: true,
	}))

	// Construct server address
	serverAddr := dialTarget
	if serverAddr == "" {
		serverAddr = fmt.Sprintf("%s:%s", pc.cfg.GRPCHost, pc.cfg.GRPCPort)
	}
	if serverAddr == "" {
		return fmt.Errorf("server address not configured")
	}

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

// PayoutAddress returns the configured payout address (if any).
func (pc *PokerClient) PayoutAddress() string {
	pc.RLock()
	defer pc.RUnlock()
	return pc.cfg.PayoutAddress
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

	if pc.gameStreamCancel != nil {
		pc.gameStreamCancel()
		pc.gameStreamCancel = nil
	}

	if pc.gameStream != nil {
		pc.gameStream.CloseSend()
		pc.gameStream = nil
	}

	pc.gameStreamCtx = nil
	pc.gameStreamTable = ""
	pc.setGameStreamConnectionState(false, nil)
	pc.log.Info("Stopped game stream")
}

// consumeGameStream processes incoming game updates from a stream until error or cancellation.
func (pc *PokerClient) consumeGameStream(ctx context.Context, stream pokerrpc.PokerService_StartGameStreamClient, tableID string) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		if current := pc.GetCurrentTableID(); current != "" && tableID != "" && current != tableID {
			return fmt.Errorf("game stream table changed from %s to %s", tableID, current)
		}

		update, err := stream.Recv()
		if err != nil {
			if isTransportClosing(err) {
				pc.log.Info("Game stream closed")
				return err
			}

			pc.log.Errorf("Game stream error: %v", err)
			pc.enqueueError(fmt.Errorf("game stream error: %v", err))
			return err
		}

		if update == nil {
			continue
		}

		pc.enqueueUpdate(update)
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

// enqueueUpdate sends a message to the updates channel without blocking.
func (pc *PokerClient) enqueueUpdate(msg tea.Msg) {
	select {
	case pc.UpdatesCh <- msg:
	case <-pc.ctx.Done():
	default:
		pc.log.Warn("Updates channel full, dropping update")
	}
}

// enqueueError sends an error to the error channel without blocking.
func (pc *PokerClient) enqueueError(err error) {
	select {
	case pc.ErrorsCh <- err:
	case <-pc.ctx.Done():
	default:
		pc.log.Warnf("Errors channel full, dropping error: %v", err)
	}
}

// isTransportClosing detects transient transport errors from gRPC streams.
func isTransportClosing(err error) bool {
	if err == nil {
		return false
	}

	return errors.Is(err, io.EOF) ||
		strings.Contains(err.Error(), "transport is closing") ||
		strings.Contains(err.Error(), "connection is being forcefully terminated")
}

// setConnectionState updates connection flags and emits a synthetic notification for UI layers.
func (pc *PokerClient) setConnectionState(connected bool, reason error) {
	pc.Lock()
	if pc.isConnected == connected {
		pc.Unlock()
		return
	}

	pc.isConnected = connected
	now := time.Now()
	if connected {
		pc.lastConnectTime = now
	} else {
		pc.lastDisconnectTime = now
	}
	pc.Unlock()

	var msg string
	if connected {
		msg = "connection restored"
		pc.log.Infof("notification stream connected at %s", now.Format(time.RFC3339))
	} else {
		msg = "connection lost"
		if reason != nil {
			msg = fmt.Sprintf("connection lost: %v", reason)
		}
		pc.log.Warnf("notification stream disconnected: %v", reason)
	}

	ntype := pokerrpc.NotificationType_NOTIFICATION_STREAM_CONNECTED
	if !connected {
		ntype = pokerrpc.NotificationType_NOTIFICATION_STREAM_DISCONNECTED
	}
	pc.enqueueUpdate(&pokerrpc.Notification{Type: ntype, Message: msg})
}

func (pc *PokerClient) setGameStreamConnectionState(connected bool, reason error) {
	pc.Lock()
	if pc.gameStreamConnected == connected {
		pc.Unlock()
		return
	}

	pc.gameStreamConnected = connected
	pc.Unlock()

	var msg string
	if connected {
		msg = "game stream restored"
		pc.log.Infof("game stream connected")
	} else {
		msg = "game stream disconnected"
		if reason != nil {
			msg = fmt.Sprintf("game stream disconnected: %v", reason)
		}
		pc.log.Warnf("game stream disconnected: %v", reason)
	}

	ntype := pokerrpc.NotificationType_GAME_STREAM_CONNECTED
	if !connected {
		ntype = pokerrpc.NotificationType_GAME_STREAM_DISCONNECTED
	}
	pc.enqueueUpdate(&pokerrpc.Notification{Type: ntype, Message: msg})
}

func capBackoff(current, max time.Duration) time.Duration {
	if current >= max {
		return max
	}

	next := current * 2
	if next > max {
		return max
	}
	return next
}

func waitWithBackoff(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
