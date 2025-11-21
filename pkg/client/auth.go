package client

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/companyzero/bisonrelay/zkidentity"
	"github.com/decred/dcrd/chaincfg/chainhash"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
)

// UserIdentityData stores the persistent user identity (seed key)
type UserIdentityData struct {
	SeedHex string `json:"seed_hex,omitempty"` // User seed key (persistent)
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

// userIdentityFilePath returns the path to the user identity file
func (pc *PokerClient) userIdentityFilePath() string {
	if pc.DataDir == "" {
		return ""
	}
	return filepath.Join(pc.DataDir, "user_identity.json")
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
	Token    string
	UserID   string
	Nickname string
}

// Login logs in a user with a nickname
// If the user is not registered, it will automatically register them first
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

	return &LoginResponse{
		Token:    resp.Token,
		UserID:   resp.UserId,
		Nickname: resp.Nickname,
	}, nil
}
