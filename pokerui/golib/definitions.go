package golib

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/companyzero/bisonrelay/clientrpc/types"
	"github.com/companyzero/bisonrelay/lockfile"
	"github.com/companyzero/bisonrelay/rates"
	"github.com/companyzero/bisonrelay/zkidentity"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/slog"
	"github.com/vctt94/pokerbisonrelay/pkg/client"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
)

type initClient struct {
	ServerAddr string `json:"server_addr"`

	GRPCCertPath   string `json:"grpc_cert_path"`
	PayoutAddress  string `json:"payout_address"`
	DBRoot         string `json:"dbroot"`
	DataDir        string `json:"datadir"`
	DownloadsDir   string `json:"downloads_dir"`
	LogFile        string `json:"log_file"`
	DebugLevel     string `json:"debug_level"`
	WantsLogNtfns  bool   `json:"wants_log_ntfns"`
	LogPings       bool   `json:"log_pings"`
	PingIntervalMs int64  `json:"ping_interval_ms"`

	// New fields for RPC configuration
	RPCWebsocketURL   string `json:"rpc_websocket_url"`
	RPCCertPath       string `json:"rpc_cert_path"`
	RPCCLientCertPath string `json:"rpc_client_cert_path"`
	RPCCLientKeyPath  string `json:"rpc_client_key_path"`
	RPCUser           string `json:"rpc_user"`
	RPCPass           string `json:"rpc_pass"`
}

// handleEscrowNotification inspects notification messages for escrow funding
// updates and persists them locally for history/refund flows.
func handleEscrowNotification(cctx *clientCtx, n *pokerrpc.Notification) {
	if cctx == nil || cctx.c == nil || n == nil {
		return
	}
	if strings.TrimSpace(n.Message) == "" {
		return
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(n.Message), &payload); err != nil {
		return
	}
	typ, _ := payload["type"].(string)
	if strings.ToLower(strings.TrimSpace(typ)) != "escrow_funding" {
		return
	}
	// Escrow funding updates are broadcast to the whole table so the UI can
	// highlight other players. Only persist updates that target the local
	// player to avoid polluting our cached escrow state with someone else's.
	targetPID := strings.TrimSpace(n.GetPlayerId())
	if targetPID == "" {
		if pidRaw, ok := payload["player_id"]; ok {
			targetPID = strings.TrimSpace(fmt.Sprint(pidRaw))
		} else if pidRaw, ok := payload["playerId"]; ok {
			targetPID = strings.TrimSpace(fmt.Sprint(pidRaw))
		}
	}
	if targetPID != "" && targetPID != cctx.ID.String() {
		return
	}
	escrowID, _ := payload["escrow_id"].(string)
	if strings.TrimSpace(escrowID) == "" {
		return
	}
	info := &client.EscrowInfo{
		EscrowID: escrowID,
		Status:   "funding",
	}
	if txid, ok := payload["funding_txid"].(string); ok {
		info.FundingTxid = txid
	}
	if vout, ok := payload["funding_vout"].(float64); ok {
		info.FundingVout = uint32(vout)
	}
	if amt, ok := payload["amount_atoms"].(float64); ok {
		info.FundedAmount = uint64(amt)
	}
	if csv, ok := payload["csv_blocks"].(float64); ok {
		info.CSVBlocks = uint32(csv)
	}
	if height, ok := payload["confirmed_height"].(float64); ok && height > 0 {
		info.ConfirmedHeight = uint32(height)
		info.Status = "funded"
	}

	if err := cctx.c.CacheEscrowInfo(info); err != nil && cctx.log != nil {
		cctx.log.Warnf("escrow cache update failed for %s: %v", escrowID, err)
	}
}

// handlePresignNotification triggers auto-presign when server requests it.
func handlePresignNotification(cctx *clientCtx, n *pokerrpc.Notification) {
	if cctx == nil || cctx.c == nil || n == nil {
		return
	}
	if n.Type != pokerrpc.NotificationType_PRESIGN_PENDING {
		return
	}
	tableID := strings.TrimSpace(n.TableId)
	if tableID == "" {
		return
	}
	// Check if we're already processing presign for this table
	cctx.presignMu.Lock()
	if cctx.presignActive == nil {
		cctx.presignActive = make(map[string]bool)
	}
	if cctx.presignActive[tableID] {
		cctx.presignMu.Unlock()
		return
	}
	cctx.presignActive[tableID] = true
	cctx.presignMu.Unlock()
	// Trigger auto-presign in a goroutine to avoid blocking notification loop
	go func() {
		defer func() {
			cctx.presignMu.Lock()
			delete(cctx.presignActive, tableID)
			cctx.presignMu.Unlock()
		}()
		triggerAutoPresign(cctx, tableID)
	}()
}

// triggerAutoPresign attempts to start presigning for a table when escrow is ready.
func triggerAutoPresign(cctx *clientCtx, tableID string) {
	if cctx == nil || cctx.c == nil {
		return
	}
	if cctx.Token == "" {
		if cctx.log != nil {
			cctx.log.Debugf("auto-presign: no session token")
		}
		return
	}
	// Get game state to find player's escrow_id and seat_index
	ctx, cancel := context.WithTimeout(cctx.ctx, 10*time.Second)
	defer cancel()
	gameState, err := cctx.c.PokerService.GetGameState(ctx, &pokerrpc.GetGameStateRequest{
		TableId:  tableID,
		PlayerId: cctx.ID.String(),
	})
	if err != nil {
		if cctx.log != nil {
			cctx.log.Debugf("auto-presign: get game state failed: %v", err)
		}
		return
	}
	gu := gameState.GetGameState()
	if gu == nil {
		return
	}
	// Find this player's info
	var playerEscrowID string
	var seatIndex int32 = -1
	for _, p := range gu.Players {
		if p.Id == cctx.ID.String() {
			playerEscrowID = p.EscrowId
			seatIndex = p.TableSeat
			break
		}
	}
	if playerEscrowID == "" || seatIndex < 0 {
		if cctx.log != nil {
			cctx.log.Debugf("auto-presign: no escrow_id or seat_index for table %s", tableID)
		}
		return
	}
	// Get escrow info to check if it's ready and get key_index
	escrowInfo, err := cctx.c.GetEscrowById(playerEscrowID)
	if err != nil {
		if cctx.log != nil {
			cctx.log.Debugf("auto-presign: get escrow info failed: %v", err)
		}
		return
	}
	// Check escrow status
	status, _ := escrowInfo["status"].(string)
	if status != "funded" {
		if cctx.log != nil {
			cctx.log.Debugf("auto-presign: escrow %s not ready (status=%s)", playerEscrowID, status)
		}
		return
	}
	// Get key_index to derive comp_priv
	keyIndex, ok := escrowInfo["key_index"].(float64)
	if !ok || keyIndex <= 0 {
		if cctx.log != nil {
			cctx.log.Debugf("auto-presign: escrow missing key_index")
		}
		return
	}
	// Derive session private key
	compPrivHex, _, err := cctx.c.DeriveSessionKeyAt(uint64(keyIndex))
	if err != nil {
		if cctx.log != nil {
			cctx.log.Debugf("auto-presign: derive session key failed: %v", err)
		}
		return
	}
	// Decode to get comp_pub
	compPriv, err := hex.DecodeString(compPrivHex)
	if err != nil || len(compPriv) == 0 {
		return
	}
	compKey := secp256k1.PrivKeyFromBytes(compPriv)
	compPub := compKey.PubKey().SerializeCompressed()
	// For poker tables, match_id is the table_id
	matchID := tableID
	// Session ID: use escrow_id as session_id
	sessionID := playerEscrowID
	// Start presign
	ref := cctx.c.Referee(cctx.Token)
	if err := ref.StartPresign(ctx, matchID, tableID, sessionID, uint32(seatIndex), playerEscrowID, compPub, compPrivHex); err != nil {
		if cctx.log != nil {
			cctx.log.Debugf("auto-presign: start presign failed: %v", err)
		}
		return
	}
	if cctx.log != nil {
		cctx.log.Infof("auto-presign started for table %s escrow %s", tableID, playerEscrowID)
	}
}

type createDefaultConfigArgs struct {
	DataDir         string `json:"datadir"`
	ServerAddr      string `json:"server_addr"`
	GRPCCertPath    string `json:"grpc_cert_path"`
	DebugLevel      string `json:"debug_level"`
	BrRpcUrl        string `json:"br_rpc_url"`
	BrClientCert    string `json:"br_client_cert"`
	BrClientRpcCert string `json:"br_client_rpc_cert"`
	BrClientRpcKey  string `json:"br_client_rpc_key"`
	RpcUser         string `json:"rpc_user"`
	RpcPass         string `json:"rpc_pass"`
}

// JSON payloads from Flutter
type joinWaitingRoom struct {
	RoomID   string `json:"room_id"`
	EscrowId string `json:"escrow_id"` // optional
}

type createWaitingRoom struct {
	ClientID string `json:"client_id"`
	BetAmt   int64  `json:"bet_amt"`
	EscrowId string `json:"escrow_id"` // optional
}

type openEscrowReq struct {
	BetAtoms   int64  `json:"bet_atoms"`
	CSVBlocks  int64  `json:"csv_blocks"`
	CompPubkey string `json:"comp_pubkey"` // hex-encoded 33-byte session pubkey
	KeyIndex   int64  `json:"key_index"`
}

type preSignReq struct {
	MatchID   string `json:"match_id"`
	TableID   string `json:"table_id"`
	SessionID string `json:"session_id"`
	SeatIndex int    `json:"seat_index"`
	EscrowID  string `json:"escrow_id"`
	CompPriv  string `json:"comp_priv"` // hex session priv
}

type joinPokerTable struct {
	TableID string `json:"table_id"`
}

type createPokerTable struct {
	SmallBlind      int64 `json:"small_blind"`
	BigBlind        int64 `json:"big_blind"`
	MaxPlayers      int32 `json:"max_players"`
	MinPlayers      int32 `json:"min_players"`
	MinBalance      int64 `json:"min_balance"`
	BuyIn           int64 `json:"buy_in"`
	StartingChips   int64 `json:"starting_chips"`
	TimeBankSeconds int32 `json:"time_bank_seconds"`
	AutoStartMs     int32 `json:"auto_start_ms"`
	AutoAdvanceMs   int32 `json:"auto_advance_ms"`
}

type makeBet struct {
	Amount int64 `json:"amount"`
}

type registerReq struct {
	Nickname string `json:"nickname"`
}

type loginReq struct {
	Nickname string `json:"nickname"`
}

type loginResp struct {
	Token    string `json:"token"`
	UserID   string `json:"user_id"`
	Nickname string `json:"nickname"`
	Address  string `json:"address"`
}

type setPayoutAddressReq struct {
	Address   string `json:"address"`
	Signature string `json:"signature"`
	Code      string `json:"code"`
}

type escrowStatusReq struct {
	EscrowID string `json:"escrow_id"`
}

type evaluateHand struct {
	Cards []struct {
		Suit  int32 `json:"suit"`
		Value int32 `json:"value"`
	} `json:"cards"`
}

// JSON returned to Flutter (shape must match Dart LocalWaitingRoom/LocalPlayer)
type player struct {
	UID    string `json:"uid"`
	Nick   string `json:"nick"`
	BetAmt int64  `json:"bet_amt"`
	Ready  bool   `json:"ready"`
}

type waitingRoom struct {
	ID      string    `json:"id"`
	HostID  string    `json:"host_id"`
	BetAmt  int64     `json:"bet_amt"`
	Players []*player `json:"players"`
}

// pokerTable represents a poker table DTO for Flutter
// All fields are explicitly set to avoid JSON type ambiguity
type pokerTable struct {
	ID              string `json:"id"`
	HostID          string `json:"host_id"`
	SmallBlind      int64  `json:"small_blind"`
	BigBlind        int64  `json:"big_blind"`
	MaxPlayers      int32  `json:"max_players"`
	MinPlayers      int32  `json:"min_players"`
	CurrentPlayers  int32  `json:"current_players"`
	MinBalance      int64  `json:"min_balance"`
	BuyIn           int64  `json:"buy_in"`
	GameStarted     bool   `json:"game_started"`
	AllPlayersReady bool   `json:"all_players_ready"`
}

// tableFromProto converts a protobuf Table to a clean DTO with all fields explicitly set
func tableFromProto(t *pokerrpc.Table) *pokerTable {
	if t == nil {
		return nil
	}
	return &pokerTable{
		ID:              t.Id,
		HostID:          t.HostId,
		SmallBlind:      t.SmallBlind,
		BigBlind:        t.BigBlind,
		MaxPlayers:      t.MaxPlayers,
		MinPlayers:      t.MinPlayers,
		CurrentPlayers:  t.CurrentPlayers,
		MinBalance:      t.MinBalance,
		BuyIn:           t.BuyIn,
		GameStarted:     t.GameStarted,
		AllPlayersReady: t.AllPlayersReady,
	}
}

// cardDTO represents a card for JSON marshaling
type cardDTO struct {
	Suit  string `json:"suit"`
	Value string `json:"value"`
}

// playerDTO represents a player for JSON marshaling
type playerDTO struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	Balance         int64      `json:"balance"`
	Hand            []*cardDTO `json:"hand"`
	CurrentBet      int64      `json:"currentBet"`
	Folded          bool       `json:"folded"`
	IsTurn          bool       `json:"isTurn"`
	IsAllIn         bool       `json:"isAllIn"`
	IsDealer        bool       `json:"isDealer"`
	IsReady         bool       `json:"isReady"`
	Disconnected    bool       `json:"disconnected"`
	HandDescription string     `json:"handDescription"`
	PlayerState     int32      `json:"playerState"` // enum as int
	IsSmallBlind    bool       `json:"isSmallBlind"`
	IsBigBlind      bool       `json:"isBigBlind"`
	EscrowID        string     `json:"escrowId"`
	EscrowReady     bool       `json:"escrowReady"`
	TableSeat       int32      `json:"tableSeat"`
	PresignComplete bool       `json:"presignComplete"`
}

// gameUpdateDTO represents a game update for JSON marshaling
type gameUpdateDTO struct {
	TableId            string       `json:"tableId"`
	Phase              int32        `json:"phase"` // enum as int
	Players            []*playerDTO `json:"players"`
	CommunityCards     []*cardDTO   `json:"communityCards"`
	Pot                int64        `json:"pot"`
	CurrentBet         int64        `json:"currentBet"`
	CurrentPlayer      string       `json:"currentPlayer"`
	MinRaise           int64        `json:"minRaise"`
	MaxRaise           int64        `json:"maxRaise"`
	GameStarted        bool         `json:"gameStarted"`
	PlayersRequired    int32        `json:"playersRequired"`
	PlayersJoined      int32        `json:"playersJoined"`
	PhaseName          string       `json:"phaseName"`
	TimeBankSeconds    int32        `json:"timeBankSeconds"`
	TurnDeadlineUnixMs int64        `json:"turnDeadlineUnixMs"`
}

func cardToDTO(c *pokerrpc.Card) *cardDTO {
	if c == nil {
		return nil
	}
	return &cardDTO{
		Suit:  c.Suit,
		Value: c.Value,
	}
}

func playerToDTO(p *pokerrpc.Player) *playerDTO {
	if p == nil {
		return nil
	}
	hand := make([]*cardDTO, 0, len(p.Hand))
	for _, c := range p.Hand {
		hand = append(hand, cardToDTO(c))
	}
	return &playerDTO{
		ID:              p.Id,
		Name:            p.Name,
		Balance:         p.Balance,
		Hand:            hand,
		CurrentBet:      p.CurrentBet,
		Folded:          p.Folded,
		IsTurn:          p.IsTurn,
		IsAllIn:         p.IsAllIn,
		IsDealer:        p.IsDealer,
		IsReady:         p.IsReady,
		Disconnected:    p.IsDisconnected,
		HandDescription: p.HandDescription,
		IsSmallBlind:    p.IsSmallBlind,
		IsBigBlind:      p.IsBigBlind,
		EscrowID:        p.EscrowId,
		EscrowReady:     p.EscrowReady,
		TableSeat:       p.TableSeat,
		PresignComplete: p.PresignComplete,
	}
}

func gameUpdateToDTO(gu *pokerrpc.GameUpdate) *gameUpdateDTO {
	if gu == nil {
		return nil
	}
	players := make([]*playerDTO, 0, len(gu.Players))
	for _, p := range gu.Players {
		players = append(players, playerToDTO(p))
	}
	communityCards := make([]*cardDTO, 0, len(gu.CommunityCards))
	for _, c := range gu.CommunityCards {
		communityCards = append(communityCards, cardToDTO(c))
	}
	return &gameUpdateDTO{
		TableId:            gu.TableId,
		Phase:              int32(gu.Phase),
		Players:            players,
		CommunityCards:     communityCards,
		Pot:                gu.Pot,
		CurrentBet:         gu.CurrentBet,
		CurrentPlayer:      gu.CurrentPlayer,
		MinRaise:           gu.MinRaise,
		MaxRaise:           gu.MaxRaise,
		GameStarted:        gu.GameStarted,
		PlayersRequired:    gu.PlayersRequired,
		PlayersJoined:      gu.PlayersJoined,
		PhaseName:          gu.PhaseName,
		TimeBankSeconds:    gu.TimeBankSeconds,
		TurnDeadlineUnixMs: gu.TurnDeadlineUnixMs,
	}
}

// localInfo represents local client information
type localInfo struct {
	// Full hex-encoded client ID used by the server.
	ID   string `json:"id"`
	Nick string `json:"nick"`
}

// runState represents the current run state
type runState struct {
	ClientRunning bool `json:"client_running"`
}

// escrowState represents escrow information
type escrowState struct {
	EscrowId       string `json:"escrow_id"`
	DepositAddress string `json:"deposit_address"`
	PkScriptHex    string `json:"pk_script_hex"`
}

// clientCtx represents a client context
type clientCtx struct {
	ID     zkidentity.ShortID
	Nick   string
	c      *client.PokerClient
	ctx    context.Context
	chat   types.ChatServiceClient
	cancel func()
	runMtx sync.Mutex
	runErr error

	log          slog.Logger
	certConfChan chan bool

	httpClient *http.Client
	rates      *rates.Rates

	// expirationDays are the expiration days provided by the server when
	// connected
	expirationDays uint64

	Token string

	serverState atomic.Value

	presignMu     sync.Mutex
	presignActive map[string]bool
}

// Global variables
var (
	cmtx sync.Mutex
	cs   map[uint32]*clientCtx
	lfs  map[string]*lockfile.LockFile = map[string]*lockfile.LockFile{}

	// The following are debug vars.
	sigUrgCount       atomic.Uint64
	isServerConnected atomic.Bool

	// Global escrow state for demo purposes
	es *escrowState
)

// parseJoinWRPayload parses the join waiting room payload
func parseJoinWRPayload(payload []byte) (roomID, escrowID string, err error) {
	var req joinWaitingRoom
	if err := json.Unmarshal(payload, &req); err != nil {
		return "", "", fmt.Errorf("unmarshal join WR payload: %w", err)
	}
	return req.RoomID, req.EscrowId, nil
}

// handleInitClient initializes a new client with proper configuration
func handleInitClient(handle uint32, args initClient) (*localInfo, error) {
	cmtx.Lock()
	defer cmtx.Unlock()
	if cs == nil {
		cs = make(map[uint32]*clientCtx)
	}
	if cs[handle] != nil {
		return &localInfo{ID: cs[handle].ID.String(), Nick: cs[handle].Nick}, nil
	}

	// Ensure the data directory exists first
	if err := os.MkdirAll(args.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory %s: %v", args.DataDir, err)
	}

	// Ensure the logs subdirectory exists
	logsDir := filepath.Dir(args.LogFile)
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create logs directory %s: %v", logsDir, err)
	}

	// Load (or create) the poker client configuration. This uses the same
	// config file as the CLI client so settings are shared.
	cfg, err := client.LoadClientConfig(args.DataDir, appName+".conf")
	if err != nil {
		return nil, fmt.Errorf("failed to load client config: %v", err)
	}

	// If the UI provided a payout address, prefer that over the config file.
	if args.PayoutAddress != "" {
		cfg.PayoutAddress = args.PayoutAddress
	}

	// Initialize notification manager BEFORE creating the client
	cfg.Notifications = client.NewNotificationManager()

	// Create context
	ctx, cancel := context.WithCancel(context.Background())

	// Create poker client with configuration
	pc, err := client.NewPokerClient(ctx, cfg)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create poker client: %v", err)
	}

	// Start the notification stream to receive server notifications (TABLE_CREATED, etc.)
	if err := pc.StartNotificationStream(ctx); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start notification stream: %v", err)
	}

	// Without a BR client, derive a simple local identity from the poker client ID.
	clientID := pc.ID
	nick := clientID.String()

	cctx := &clientCtx{
		ID:     pc.ID,
		Nick:   nick,
		ctx:    ctx,
		c:      pc,
		cancel: cancel,
		log:    cfg.LogBackend.Logger(appName),
	}
	cs[handle] = cctx

	// Start a goroutine to handle client closure and errors
	go func() {
		// Wait for context to be cancelled or client to stop
		<-ctx.Done()

		// Clean up the client if it stops running
		cmtx.Lock()
		delete(cs, handle)
		cmtx.Unlock()

		// Notify the system that the client stopped
		notify(NTClientStopped, nil, ctx.Err())
	}()

	// Bridge poker notifications from the Go client into the generic
	// plugin notification channel so Flutter can react to game events
	// without opening its own gRPC streams.
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-pc.UpdatesCh:
				if !ok {
					return
				}

				switch v := msg.(type) {
				case *pokerrpc.Notification:
					// Update local escrow cache when funding updates arrive.
					handleEscrowNotification(cctx, v)
					// Trigger auto-presign when server requests it.
					handlePresignNotification(cctx, v)
					notify(NTPokerNotification, v, nil)
				case *pokerrpc.GameUpdate:
					// Convert GameUpdate to DTO and forward to Flutter
					dto := gameUpdateToDTO(v)
					notify(NTGameUpdate, dto, nil)
				default:
					// Ignore other message types (like tea.Msg wrapper types)
				}
			}
		}
	}()

	cctx.log.Infof("Poker client initialized with ID: %s", clientID.String())

	return &localInfo{ID: clientID.String(), Nick: nick}, nil
}

// createDefaultConfig creates a default configuration file when none exists
func createDefaultConfig(dataDir, serverAddr, grpcCertPath, debugLevel, brRpcUrl, brClientCert, brClientRpcCert, brClientRpcKey, rpcUser, rpcPass string) error {
	// Ensure the data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %v", err)
	}

	// Set default values
	if serverAddr == "" {
		serverAddr = "178.156.178.191:50050" // Default server
	}

	// Validate required BR config values before writing the file. These are
	// required for the client to successfully connect to BisonRelay; writing
	// an incomplete config leads to startup failure later.
	var missing []string
	if strings.TrimSpace(brClientCert) == "" {
		missing = append(missing, "brclientcert")
	}
	if strings.TrimSpace(brClientRpcCert) == "" {
		missing = append(missing, "brclientrpccert")
	}
	if strings.TrimSpace(brClientRpcKey) == "" {
		missing = append(missing, "brclientrpckey")
	}
	if len(missing) > 0 {
		return fmt.Errorf("cannot create config: missing required BR fields: %s", strings.Join(missing, ", "))
	}

	// Note: grpcHost and grpcPort are not needed for the INI format
	// The Flutter config loader will parse the serverAddr directly

	// Create the configuration file content in the correct INI format
	configPath := filepath.Join(dataDir, "pokerui.conf")
	content := fmt.Sprintf(`[default]
serveraddr=%s
datadir=%s
grpcservercert=%s
address=
brrpcurl=%s
brclientcert=%s
brclientrpccert=%s
brclientrpckey=%s
rpcuser=%s
rpcpass=%s

[clientrpc]
wantsLogNtfns=0

[log]
debuglevel=%s
maxlogfiles=5
maxbufferlines=1000
`,
		serverAddr, dataDir, grpcCertPath, brRpcUrl, brClientCert, brClientRpcCert, brClientRpcKey, rpcUser, rpcPass, debugLevel)

	// Write the configuration file
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	// Also create/update the CLI-compatible pokerclient.conf so the UI and
	// CLI share the same connection settings.
	host, port, ok := strings.Cut(serverAddr, ":")
	if !ok || host == "" || port == "" {
		return fmt.Errorf("invalid server address: %s", serverAddr)
	}
	pcConf := &client.PokerConf{
		Datadir:        dataDir,
		GRPCHost:       host,
		GRPCPort:       port,
		GRPCCertPath:   grpcCertPath,
		PayoutAddress:  "",
		LogFile:        filepath.Join(dataDir, "logs", appName+".log"),
		Debug:          debugLevel,
		MaxLogFiles:    5,
		MaxBufferLines: 1000,
	}
	if err := client.WriteClientConfigFile(pcConf, filepath.Join(dataDir, appName+".conf")); err != nil {
		return fmt.Errorf("failed to write %s.conf: %v", appName, err)
	}

	// Create default server certificate if it doesn't exist
	if _, err := os.Stat(grpcCertPath); os.IsNotExist(err) {
		if err := client.CreateDefaultServerCert(grpcCertPath); err != nil {
			return fmt.Errorf("failed to create default server certificate: %v", err)
		}
	}

	return nil
}

// handleCreateDefaultConfig handles the CTCreateDefaultConfig command
func handleCreateDefaultConfig(args createDefaultConfigArgs) (map[string]string, error) {
	if err := createDefaultConfig(args.DataDir, args.ServerAddr, args.GRPCCertPath, args.DebugLevel,
		args.BrRpcUrl, args.BrClientCert, args.BrClientRpcCert, args.BrClientRpcKey, args.RpcUser, args.RpcPass); err != nil {
		return nil, err
	}

	return map[string]string{
		"status":      "created",
		"config_path": filepath.Join(args.DataDir, "pokerui.conf"),
	}, nil
}

// handleCreateDefaultServerCert handles the CTCreateDefaultServerCert command
func handleCreateDefaultServerCert(certPath string) (map[string]string, error) {
	if err := client.CreateDefaultServerCert(certPath); err != nil {
		return nil, err
	}

	return map[string]string{
		"status":    "created",
		"cert_path": certPath,
	}, nil
}

// handleLoadConfig loads config from a provided path (either a file path to
// pokerui.conf or a datadir) and returns a flat map for Flutter.
func handleLoadConfig(pathOrDir string) (map[string]interface{}, error) {
	datadir := pathOrDir
	if datadir == "" {
		return nil, fmt.Errorf("empty path")
	}
	// If a file path was provided, use its directory as datadir.
	if strings.HasSuffix(strings.ToLower(datadir), ".conf") {
		datadir = filepath.Dir(datadir)
	}

	// Load (or create) the shared poker client config used by both CLI and UI.
	cfg, err := client.LoadClientConf(datadir, appName+".conf")
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	serverAddr := ""
	if cfg.GRPCHost != "" && cfg.GRPCPort != "" {
		serverAddr = fmt.Sprintf("%s:%s", cfg.GRPCHost, cfg.GRPCPort)
	}

	// Build a map compatible with Flutter Config expectations
	res := map[string]interface{}{
		"server_addr":    serverAddr,
		"grpc_cert_path": cfg.GRPCCertPath,

		"debug_level":     cfg.Debug,
		"wants_log_ntfns": false,
		"datadir":         cfg.Datadir,
		"payout_address":  cfg.PayoutAddress,
	}

	return res, nil
}
