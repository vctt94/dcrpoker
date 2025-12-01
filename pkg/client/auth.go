package client

import (
	"context"
	"crypto/hmac"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/companyzero/bisonrelay/zkidentity"
	"github.com/decred/dcrd/chaincfg/chainhash"
	"github.com/decred/dcrd/crypto/blake256"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
)

// UserIdentityData stores the persistent user identity (seed key)
type UserIdentityData struct {
	SeedHex string `json:"seed_hex,omitempty"` // User seed key (persistent)
}

// SessionData stores persisted session info to avoid re-logins across restarts.
type SessionData struct {
	Token         string `json:"token"`
	UserID        string `json:"user_id"`
	Nickname      string `json:"nickname"`
	PayoutAddress string `json:"payout_address"`
}

type LoginCode struct {
	Code        string
	TTL         time.Duration
	AddressHint string
}

// sessionKeyState tracks deterministic session key indices.
type sessionKeyState struct {
	NextIndex uint64 `json:"next_index"`
}

// GetOrCreateSeedKey generates or loads the seed key
func (pc *PokerClient) GetOrCreateSeedKey() (string, error) {
	// Try to load existing seed
	if seedHex, err := pc.loadSeedKey(); err == nil && seedHex != "" {
		return seedHex, nil
	}

	// Generate new seed
	seedPriv, err := secp256k1.GeneratePrivateKey()
	if err != nil {
		return "", fmt.Errorf("generate seed: %w", err)
	}
	seedHex := hex.EncodeToString(seedPriv.Serialize())

	// Save to disk
	if err := pc.saveSeedKey(seedHex); err != nil {
		return "", fmt.Errorf("save seed: %w", err)
	}

	return seedHex, nil
}

// DeriveUserID derives a user ID from the seed key
func (pc *PokerClient) DeriveUserID(seedHex string) (string, error) {
	seedBytes, err := hex.DecodeString(seedHex)
	if err != nil {
		return "", fmt.Errorf("decode seed: %w", err)
	}
	if len(seedBytes) != 32 {
		return "", fmt.Errorf("invalid seed length: %d", len(seedBytes))
	}

	// Double hash (double BLAKE-256)
	userIDBytes := chainhash.HashB(seedBytes)

	var userID zkidentity.ShortID
	userID.FromBytes(userIDBytes[:])

	return userID.String(), nil
}

// GetUserID gets the user ID (loads seed if needed)
func (pc *PokerClient) GetUserID() (string, error) {
	seedHex, err := pc.GetOrCreateSeedKey()
	if err != nil {
		return "", err
	}
	return pc.DeriveUserID(seedHex)
}

// RequestLoginCode asks the server for a nonce that must be signed by the
// wallet to complete login.
func (pc *PokerClient) RequestLoginCode(ctx context.Context) (*LoginCode, error) {
	userID, err := pc.GetUserID()
	if err != nil {
		return nil, err
	}

	authClient := pokerrpc.NewAuthServiceClient(pc.conn)
	resp, err := authClient.RequestLoginCode(ctx, &pokerrpc.RequestLoginCodeRequest{UserId: userID})
	if err != nil {
		return nil, err
	}

	return &LoginCode{
		Code:        resp.GetCode(),
		TTL:         time.Duration(resp.GetTtlSec()) * time.Second,
		AddressHint: resp.GetAddressHint(),
	}, nil
}

// userIdentityFilePath returns the path to the user identity file
func (pc *PokerClient) userIdentityFilePath() string {
	if pc.DataDir == "" {
		return ""
	}
	return filepath.Join(pc.DataDir, "user_identity.json")
}

// sessionFilePath returns the path to the persisted session file.
func (pc *PokerClient) sessionFilePath() string {
	if pc.DataDir == "" {
		return ""
	}
	return filepath.Join(pc.DataDir, "session.json")
}

// saveSeedKey saves the seed key to disk
func (pc *PokerClient) saveSeedKey(seedHex string) error {
	path := pc.userIdentityFilePath()
	if path == "" {
		return fmt.Errorf("no data directory configured")
	}

	// Load existing data if file exists
	var data UserIdentityData
	if existingData, err := os.ReadFile(path); err == nil {
		json.Unmarshal(existingData, &data)
	}

	// Update seed key
	data.SeedHex = seedHex

	// Save
	blob, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	return os.WriteFile(path, blob, 0600) // Read/write for owner only
}

// loadSeedKey loads the seed key from disk
func (pc *PokerClient) loadSeedKey() (string, error) {
	path := pc.userIdentityFilePath()
	if path == "" {
		return "", fmt.Errorf("no data directory configured")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // No seed exists yet
		}
		return "", err
	}

	var keyData UserIdentityData
	if err := json.Unmarshal(data, &keyData); err != nil {
		return "", err
	}

	if keyData.SeedHex == "" {
		return "", nil // No seed in file
	}

	// Validate seed format
	seedBytes, err := hex.DecodeString(keyData.SeedHex)
	if err != nil || len(seedBytes) != 32 {
		return "", fmt.Errorf("invalid seed format")
	}

	return keyData.SeedHex, nil
}

// getOrCreateSeedKeyFromDir loads or creates a seed key from the data directory
// This is a standalone function that doesn't require a PokerClient instance
func getOrCreateSeedKeyFromDir(dataDir string) (string, error) {
	path := filepath.Join(dataDir, "user_identity.json")
	if path == "" {
		return "", fmt.Errorf("no data directory configured")
	}

	// Try to load existing seed
	data, err := os.ReadFile(path)
	if err == nil {
		var keyData UserIdentityData
		if err := json.Unmarshal(data, &keyData); err == nil && keyData.SeedHex != "" {
			// Validate seed format
			seedBytes, err := hex.DecodeString(keyData.SeedHex)
			if err == nil && len(seedBytes) == 32 {
				return keyData.SeedHex, nil
			}
		}
	}

	// Generate new seed
	seedPriv, err := secp256k1.GeneratePrivateKey()
	if err != nil {
		return "", fmt.Errorf("generate seed: %w", err)
	}
	seedHex := hex.EncodeToString(seedPriv.Serialize())

	// Save to disk
	var keyData UserIdentityData
	keyData.SeedHex = seedHex
	blob, err := json.MarshalIndent(keyData, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal seed: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return "", fmt.Errorf("create data dir: %w", err)
	}

	if err := os.WriteFile(path, blob, 0600); err != nil {
		return "", fmt.Errorf("save seed: %w", err)
	}

	return seedHex, nil
}

// deriveUserIDFromSeed derives a user ID from a seed key
// This is a standalone function that doesn't require a PokerClient instance
func deriveUserIDFromSeed(seedHex string) (string, error) {
	seedBytes, err := hex.DecodeString(seedHex)
	if err != nil {
		return "", fmt.Errorf("decode seed: %w", err)
	}
	if len(seedBytes) != 32 {
		return "", fmt.Errorf("invalid seed length: %d", len(seedBytes))
	}

	// Double hash (double BLAKE-256)
	userIDBytes := chainhash.HashB(seedBytes)

	var userID zkidentity.ShortID
	userID.FromBytes(userIDBytes[:])

	return userID.String(), nil
}

// getUserIDFromDir loads or creates a user ID from the data directory
// This is a standalone function that doesn't require a PokerClient instance
func getUserIDFromDir(dataDir string) (string, error) {
	seedHex, err := getOrCreateSeedKeyFromDir(dataDir)
	if err != nil {
		return "", err
	}
	return deriveUserIDFromSeed(seedHex)
}

// validateNickname validates a nickname
func validateNickname(nickname string) error {
	nickname = strings.TrimSpace(nickname)

	if len(nickname) < 3 {
		return fmt.Errorf("nickname too short (minimum 3 characters)")
	}
	if len(nickname) > 32 {
		return fmt.Errorf("nickname too long (maximum 32 characters)")
	}

	// Allow: alphanumeric, underscore, hyphen
	validPattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validPattern.MatchString(nickname) {
		return fmt.Errorf("nickname contains invalid characters (only letters, numbers, underscore, and hyphen allowed)")
	}

	return nil
}

// Register registers a new user with a nickname
func (pc *PokerClient) Register(ctx context.Context, nickname string) error {
	// Validate nickname
	if err := validateNickname(nickname); err != nil {
		return err
	}

	// Get or create seed
	seedHex, err := pc.GetOrCreateSeedKey()
	if err != nil {
		return err
	}

	// Derive user ID
	userID, err := pc.DeriveUserID(seedHex)
	if err != nil {
		return err
	}

	// Send registration request
	req := &pokerrpc.RegisterRequest{
		Nickname: nickname,
		UserId:   userID,
	}

	authClient := pokerrpc.NewAuthServiceClient(pc.conn)
	resp, err := authClient.Register(ctx, req)
	if err != nil {
		return err
	}

	if !resp.Ok {
		return fmt.Errorf("registration failed: %s", resp.Error)
	}

	return nil
}

// LoginResponse contains login response data
type LoginResponse struct {
	Token         string
	UserID        string
	Nickname      string
	PayoutAddress string
}

// Login logs in a user with a nickname. If the user is not registered, it will
// automatically register them first.
func (pc *PokerClient) Login(ctx context.Context, nickname string) (*LoginResponse, error) {
	// Validate nickname
	if err := validateNickname(nickname); err != nil {
		return nil, err
	}

	// Get user ID from seed
	userID, err := pc.GetUserID()
	if err != nil {
		return nil, err
	}

	// Send login request
	req := &pokerrpc.LoginRequest{
		Nickname: nickname,
		UserId:   userID,
	}

	authClient := pokerrpc.NewAuthServiceClient(pc.conn)
	resp, err := authClient.Login(ctx, req)
	if err != nil {
		return nil, err
	}

	if !resp.Ok {
		// Check if user needs to register first
		if strings.Contains(strings.ToLower(resp.Error), "please register first") {
			// Auto-register the user
			if err := pc.Register(ctx, nickname); err != nil {
				return nil, fmt.Errorf("auto-registration failed: %w", err)
			}
			// Retry login after registration
			resp, err = authClient.Login(ctx, req)
			if err != nil {
				return nil, err
			}
			if !resp.Ok {
				return nil, fmt.Errorf("login failed after registration: %s", resp.Error)
			}
		} else {
			return nil, fmt.Errorf("login failed: %s", resp.Error)
		}
	}

	// Persist session for future resumes. Do not fail login if persistence fails.
	session := &SessionData{
		Token:         resp.Token,
		UserID:        resp.UserId,
		Nickname:      resp.Nickname,
		PayoutAddress: resp.PayoutAddress,
	}
	if err := pc.SaveSession(session); err != nil {
		pc.log.Warnf("failed to persist session for %s: %v", nickname, err)
	}

	pc.PersistPayoutAddress(resp.PayoutAddress)

	return &LoginResponse{
		Token:         resp.Token,
		UserID:        resp.UserId,
		Nickname:      resp.Nickname,
		PayoutAddress: resp.PayoutAddress,
	}, nil
}

// SaveSession persists the session details to disk so we can resume without a new login.
func (pc *PokerClient) SaveSession(session *SessionData) error {
	if session == nil {
		return fmt.Errorf("session cannot be nil")
	}

	path := pc.sessionFilePath()
	if path == "" {
		return fmt.Errorf("no data directory configured")
	}

	blob, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	return os.WriteFile(path, blob, 0600)
}

func (pc *PokerClient) configFilePath() string {
	name := strings.TrimSpace(pc.cfg.ConfFileName)
	if name == "" {
		return ""
	}
	if filepath.IsAbs(name) {
		return name
	}
	return filepath.Join(pc.cfg.Datadir, name)
}

// persistPayoutAddress writes the verified payout address into the shared
// client config so subsequent runs can reuse it without prompting the user.
func (pc *PokerClient) PersistPayoutAddress(addr string) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return
	}
	pc.cfg.PayoutAddress = addr

	confPath := pc.configFilePath()
	if confPath == "" {
		return
	}
	pcConf := &PokerConf{
		Datadir:        pc.cfg.Datadir,
		GRPCHost:       pc.cfg.GRPCHost,
		GRPCPort:       pc.cfg.GRPCPort,
		GRPCCertPath:   pc.cfg.GRPCCertPath,
		PayoutAddress:  pc.cfg.PayoutAddress,
		LogFile:        pc.cfg.LogFile,
		Debug:          pc.cfg.Debug,
		MaxLogFiles:    pc.cfg.MaxLogFiles,
		MaxBufferLines: pc.cfg.MaxBufferLines,
	}
	if err := WriteClientConfigFile(pcConf, confPath); err != nil {
		pc.log.Warnf("failed to persist payout address to config: %v", err)
	}
}

func (pc *PokerClient) sessionKeysPath() string {
	if pc.DataDir == "" {
		return ""
	}
	return filepath.Join(pc.DataDir, "session_keys.json")
}

func (pc *PokerClient) loadSessionKeyState() (*sessionKeyState, error) {
	path := pc.sessionKeysPath()
	if path == "" {
		return nil, fmt.Errorf("no data directory configured")
	}
	state := &sessionKeyState{NextIndex: 0}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, state); err != nil {
		return nil, err
	}
	return state, nil
}

func (pc *PokerClient) saveSessionKeyState(state *sessionKeyState) error {
	path := pc.sessionKeysPath()
	if path == "" {
		return fmt.Errorf("no data directory configured")
	}
	blob, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	return os.WriteFile(path, blob, 0600)
}

func (pc *PokerClient) deriveSessionPriv(index uint64) (*secp256k1.PrivateKey, error) {
	seedHex, err := pc.GetOrCreateSeedKey()
	if err != nil {
		return nil, err
	}
	seed, err := hex.DecodeString(seedHex)
	if err != nil || len(seed) != 32 {
		return nil, fmt.Errorf("invalid seed")
	}

	h := hmac.New(blake256.New, seed)
	h.Write([]byte("poker/session-key/v1"))
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], index)
	h.Write(buf[:])
	keyBytes := h.Sum(nil)
	if len(keyBytes) != 32 {
		return nil, fmt.Errorf("bad key length")
	}
	return secp256k1.PrivKeyFromBytes(keyBytes), nil
}

// GenerateSessionKey derives a deterministic session key using the client's
// seed and advances the local counter. Returns hex-encoded priv/pub and index.
func (pc *PokerClient) GenerateSessionKey() (privHex, pubHex string, index uint64, err error) {
	state, err := pc.loadSessionKeyState()
	if err != nil {
		return "", "", 0, err
	}
	index = state.NextIndex
	priv, err := pc.deriveSessionPriv(index)
	if err != nil {
		return "", "", 0, err
	}
	state.NextIndex++
	if err := pc.saveSessionKeyState(state); err != nil {
		return "", "", 0, err
	}
	return hex.EncodeToString(priv.Serialize()), hex.EncodeToString(priv.PubKey().SerializeCompressed()), index, nil
}

// DeriveSessionKeyAt deterministically derives the session key for a specific
// index without mutating local state.
func (pc *PokerClient) DeriveSessionKeyAt(index uint64) (privHex, pubHex string, err error) {
	priv, err := pc.deriveSessionPriv(index)
	if err != nil {
		return "", "", err
	}
	return hex.EncodeToString(priv.Serialize()), hex.EncodeToString(priv.PubKey().SerializeCompressed()), nil
}

// LoadSession loads a persisted session from disk.
func (pc *PokerClient) LoadSession() (*SessionData, error) {
	path := pc.sessionFilePath()
	if path == "" {
		return nil, fmt.Errorf("no data directory configured")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var session SessionData
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}

	if session.Token == "" {
		return nil, nil
	}

	return &session, nil
}

// ClearSession removes any persisted session token.
func (pc *PokerClient) ClearSession() error {
	path := pc.sessionFilePath()
	if path == "" {
		return nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// ResumeSession verifies a persisted session against the server.
// Returns nil if no valid session is available.
func (pc *PokerClient) ResumeSession(ctx context.Context) (*SessionData, error) {
	session, err := pc.LoadSession()
	if err != nil || session == nil {
		return session, err
	}

	expectedUserID, err := pc.GetUserID()
	if err != nil {
		return nil, err
	}
	if session.UserID != "" && session.UserID != expectedUserID {
		if err := pc.ClearSession(); err != nil {
			pc.log.Warnf("failed clearing mismatched session: %v", err)
		}
		return nil, nil
	}

	authClient := pokerrpc.NewAuthServiceClient(pc.conn)
	resp, err := authClient.GetUserInfo(ctx, &pokerrpc.GetUserInfoRequest{
		Token: session.Token,
	})
	if err != nil {
		return nil, err
	}

	// Empty response indicates expired or unknown session.
	if resp.GetUserId() == "" {
		if err := pc.ClearSession(); err != nil {
			pc.log.Warnf("failed clearing expired session: %v", err)
		}
		return nil, nil
	}

	if resp.UserId != expectedUserID {
		if err := pc.ClearSession(); err != nil {
			pc.log.Warnf("failed clearing mismatched session: %v", err)
		}
		return nil, nil
	}

	// If both local config and server session report a payout address, but
	// they differ, treat the session as invalid so the user is forced to go
	// through the Sign Address flow again. The local config is the source of
	// truth; equality means the address is already verified and does not need
	// to be re-signed.
	localPayout := strings.TrimSpace(pc.PayoutAddress())
	serverPayout := strings.TrimSpace(resp.GetPayoutAddress())
	if localPayout != "" && serverPayout != "" && localPayout != serverPayout {
		pc.log.Warnf("payout address mismatch (local=%s, server=%s); clearing session to require re-signing",
			localPayout, serverPayout)
		if err := pc.ClearSession(); err != nil {
			pc.log.Warnf("failed clearing session on payout mismatch: %v", err)
		}
		return nil, nil
	}

	// Refresh nickname from server in case it changed.
	session.UserID = resp.UserId
	if resp.Nickname != "" {
		session.Nickname = resp.Nickname
	}

	if err := pc.SaveSession(session); err != nil {
		pc.log.Warnf("failed to persist session refresh: %v", err)
	}

	return session, nil
}
