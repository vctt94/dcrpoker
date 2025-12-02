package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/companyzero/bisonrelay/zkidentity"
	"github.com/decred/dcrd/chaincfg/v3"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"github.com/decred/dcrd/txscript/v4/stdaddr"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
)

// authState manages authentication state
type authState struct {
	mu sync.RWMutex
	db Database

	// In-memory cache for fast lookups (loaded from DB on startup)
	// User ID → Nickname mapping (each user has one nickname)
	uidToNickname map[zkidentity.ShortID]string

	// User ID → payout address (P2PKH) verified at login.
	uidToPayoutAddr map[zkidentity.ShortID]string

	// Active sessions: token → session info (ephemeral, not persisted)
	sessions map[string]sessionInfo

	// Pending login codes: nonce → metadata
	nonces map[string]loginNonce

	// User ID → metadata (for future use, loaded from DB)
	userMetadata map[zkidentity.ShortID]UserMetadata
}

// sessionInfo contains session information
type sessionInfo struct {
	userID     zkidentity.ShortID
	nickname   string
	payoutAddr string
	created    time.Time
}

type loginNonce struct {
	userID  *zkidentity.ShortID
	expires time.Time
}

func (s *Server) sessionForToken(tok string) (sessionInfo, bool) {
	if s.auth == nil {
		return sessionInfo{}, false
	}
	s.auth.mu.RLock()
	defer s.auth.mu.RUnlock()
	sess, ok := s.auth.sessions[tok]
	return sess, ok
}

func (s *Server) payoutForToken(tok string) (zkidentity.ShortID, string, bool) {
	sess, ok := s.sessionForToken(tok)
	if !ok {
		return zkidentity.ShortID{}, "", false
	}
	return sess.userID, sess.payoutAddr, true
}

// authSessionCount returns the number of active auth sessions (for debug logs).
func (s *Server) authSessionCount() int {
	if s.auth == nil {
		return 0
	}
	s.auth.mu.RLock()
	defer s.auth.mu.RUnlock()
	return len(s.auth.sessions)
}

// TestSeedSession seeds an auth session for tests without wallet login.
// It also creates the auth user in the database so foreign key constraints are satisfied.
func (s *Server) TestSeedSession(token string, uid zkidentity.ShortID, payoutAddr, nickname string) {
	if s.auth == nil {
		s.auth = newAuthState(s.db)
	}

	// Create auth user in database (required for foreign key constraints)
	ctx := context.Background()
	if err := s.db.UpsertAuthUser(ctx, nickname, uid.String()); err != nil {
		// Log but don't fail - this is a test helper
		if s.log != nil {
			s.log.Errorf("TestSeedSession: failed to create auth user %s: %v", uid.String(), err)
		}
	}

	s.auth.mu.Lock()
	s.auth.sessions[token] = sessionInfo{
		userID:     uid,
		nickname:   nickname,
		payoutAddr: payoutAddr,
		created:    time.Now(),
	}
	s.auth.uidToPayoutAddr[uid] = payoutAddr
	s.auth.uidToNickname[uid] = nickname
	s.auth.mu.Unlock()
}

// UserMetadata contains user metadata
type UserMetadata struct {
	Nickname  string
	Created   time.Time
	LastLogin time.Time
}

// newAuthState creates a new auth state
func newAuthState(db Database) *authState {
	return &authState{
		db:              db,
		uidToNickname:   make(map[zkidentity.ShortID]string),
		uidToPayoutAddr: make(map[zkidentity.ShortID]string),
		sessions:        make(map[string]sessionInfo),
		nonces:          make(map[string]loginNonce),
		userMetadata:    make(map[zkidentity.ShortID]UserMetadata),
	}
}

func (a *authState) purgeExpiredNoncesLocked(now time.Time) {
	for code, meta := range a.nonces {
		if now.After(meta.expires) {
			delete(a.nonces, code)
		}
	}
}

func (a *authState) storeNonce(code string, userID *zkidentity.ShortID, ttl time.Duration) {
	now := time.Now()
	a.mu.Lock()
	a.purgeExpiredNoncesLocked(now)
	a.nonces[code] = loginNonce{userID: userID, expires: now.Add(ttl)}
	a.mu.Unlock()
}

// ConsumeNonce checks if a nonce is valid and returns the nonce metadata.
func (a *authState) ConsumeNonce(code string) (loginNonce, bool) {
	now := time.Now()
	a.mu.Lock()
	a.purgeExpiredNoncesLocked(now)
	meta, ok := a.nonces[code]
	if !ok || now.After(meta.expires) {
		a.mu.Unlock()
		return loginNonce{}, false
	}
	delete(a.nonces, code)
	a.mu.Unlock()
	return meta, true
}

func (a *authState) payoutAddrHint(userID zkidentity.ShortID) string {
	a.mu.RLock()
	addr := a.uidToPayoutAddr[userID]
	a.mu.RUnlock()
	return addr
}

// displayNameFor returns the cached nickname for a user ID, if known.
func (s *Server) displayNameFor(userID string) string {
	if s == nil || s.auth == nil {
		return strings.TrimSpace(userID)
	}
	var uid zkidentity.ShortID
	if err := uid.FromString(strings.TrimSpace(userID)); err != nil {
		return strings.TrimSpace(userID)
	}
	s.auth.mu.RLock()
	nick := s.auth.uidToNickname[uid]
	s.auth.mu.RUnlock()
	if strings.TrimSpace(nick) == "" {
		return strings.TrimSpace(userID)
	}
	return nick
}

// loadAuthStateFromDB loads all registered users from the database into memory
func (a *authState) loadAuthStateFromDB(ctx context.Context) error {
	users, err := a.db.ListAllAuthUsers(ctx)
	if err != nil {
		return fmt.Errorf("failed to load auth users: %w", err)
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	for _, u := range users {
		var userID zkidentity.ShortID
		if err := userID.FromString(u.UserID); err != nil {
			continue // Skip invalid user IDs
		}

		a.uidToNickname[userID] = u.Nickname

		if u.PayoutAddress.Valid {
			addr := strings.TrimSpace(u.PayoutAddress.String)
			if addr != "" {
				a.uidToPayoutAddr[userID] = addr
			}
		}

		// Initialize metadata
		meta := UserMetadata{
			Nickname:  u.Nickname,
			Created:   u.CreatedAt,
			LastLogin: time.Time{},
		}
		if u.LastLogin.Valid {
			meta.LastLogin = u.LastLogin.Time
		}
		a.userMetadata[userID] = meta
	}

	return nil
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

// Register handles user registration
func (s *Server) Register(ctx context.Context, req *pokerrpc.RegisterRequest) (*pokerrpc.RegisterResponse, error) {
	// Validate nickname
	nickname := strings.TrimSpace(req.Nickname)
	if err := validateNickname(nickname); err != nil {
		return &pokerrpc.RegisterResponse{
			Ok:    false,
			Error: err.Error(),
		}, nil
	}

	// Parse user ID
	var userID zkidentity.ShortID
	if err := userID.FromString(req.UserId); err != nil {
		return &pokerrpc.RegisterResponse{
			Ok:    false,
			Error: "invalid user ID format",
		}, nil
	}

	// Initialize auth state if needed
	if s.auth == nil {
		s.auth = newAuthState(s.db)
	}

	// Persist to database (nickname is just for UI display, no uniqueness check)
	if err := s.db.UpsertAuthUser(ctx, nickname, userID.String()); err != nil {
		return &pokerrpc.RegisterResponse{
			Ok:    false,
			Error: fmt.Sprintf("failed to save user: %v", err),
		}, nil
	}

	// Update in-memory cache
	s.auth.mu.Lock()
	s.auth.uidToNickname[userID] = nickname

	// Initialize metadata
	if _, ok := s.auth.userMetadata[userID]; !ok {
		s.auth.userMetadata[userID] = UserMetadata{
			Nickname:  nickname,
			Created:   time.Now(),
			LastLogin: time.Now(),
		}
	}
	s.auth.mu.Unlock()

	return &pokerrpc.RegisterResponse{
		Ok: true,
	}, nil
}

// RequestLoginCode issues a short-lived nonce that must be signed by the
// caller's wallet to complete login.
func (s *Server) RequestLoginCode(ctx context.Context, req *pokerrpc.RequestLoginCodeRequest) (*pokerrpc.RequestLoginCodeResponse, error) {
	if s.auth == nil {
		s.auth = newAuthState(s.db)
	}

	var uidPtr *zkidentity.ShortID
	addrHint := ""
	if trimmed := strings.TrimSpace(req.GetUserId()); trimmed != "" {
		var uid zkidentity.ShortID
		if err := uid.FromString(trimmed); err != nil {
			return nil, fmt.Errorf("invalid user id: %w", err)
		}
		uidCopy := uid
		uidPtr = &uidCopy
		addrHint = s.auth.payoutAddrHint(uid)
	}

	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return nil, fmt.Errorf("generate login code: %w", err)
	}
	code := hex.EncodeToString(b[:])
	ttl := 5 * time.Minute

	s.auth.storeNonce(code, uidPtr, ttl)

	return &pokerrpc.RequestLoginCodeResponse{
		Code:        code,
		TtlSec:      int64(ttl.Seconds()),
		AddressHint: addrHint,
	}, nil
}

// Login handles user login
func (s *Server) Login(ctx context.Context, req *pokerrpc.LoginRequest) (*pokerrpc.LoginResponse, error) {
	// Validate nickname format (but don't check uniqueness)
	nickname := strings.TrimSpace(req.Nickname)
	if err := validateNickname(nickname); err != nil {
		return &pokerrpc.LoginResponse{
			Ok:    false,
			Error: err.Error(),
		}, nil
	}

	// Parse user ID
	var userID zkidentity.ShortID
	if err := userID.FromString(req.UserId); err != nil {
		return &pokerrpc.LoginResponse{
			Ok:    false,
			Error: "invalid user ID format",
		}, nil
	}

	// Initialize auth state if needed
	if s.auth == nil {
		s.auth = newAuthState(s.db)
	}

	// Authenticate by user ID only - check if user exists in database
	authUser, err := s.db.GetAuthUserByUserID(ctx, userID.String())
	if err != nil {
		return &pokerrpc.LoginResponse{
			Ok:    false,
			Error: "user not found - please register first",
		}, nil
	}

	// Store/update nickname in database (nickname is just for UI display)
	if err := s.db.UpsertAuthUser(ctx, nickname, userID.String()); err != nil {
		return &pokerrpc.LoginResponse{
			Ok:    false,
			Error: fmt.Sprintf("failed to update nickname: %v", err),
		}, nil
	}

	params := s.chainParams
	if params == nil {
		params = chaincfg.TestNet3Params()
	}

	// If the client supplied an address + signature, verify it and persist as payout.
	addrStr := strings.TrimSpace(req.Address)
	sigB64 := strings.TrimSpace(req.Signature)
	if addrStr != "" || sigB64 != "" {
		if addrStr == "" || sigB64 == "" {
			return &pokerrpc.LoginResponse{Ok: false, Error: "address and signature are required together"}, nil
		}
		code := strings.TrimSpace(req.Code)
		if code == "" {
			return &pokerrpc.LoginResponse{Ok: false, Error: "login code is required when providing signature"}, nil
		}
		nonceMeta, ok := s.auth.ConsumeNonce(code)
		if !ok {
			return &pokerrpc.LoginResponse{Ok: false, Error: "invalid or expired code"}, nil
		}
		if nonceMeta.userID != nil && *nonceMeta.userID != userID {
			return &pokerrpc.LoginResponse{Ok: false, Error: "code does not match user id"}, nil
		}

		addr, err := stdaddr.DecodeAddress(addrStr, params)
		if err != nil {
			return &pokerrpc.LoginResponse{
				Ok:    false,
				Error: fmt.Sprintf("invalid address: %v", err),
			}, nil
		}

		// Verify signature proves control of the provided address.
		sig, err := base64.StdEncoding.DecodeString(sigB64)
		if err != nil {
			return &pokerrpc.LoginResponse{
				Ok:    false,
				Error: fmt.Sprintf("invalid signature encoding: %v", err),
			}, nil
		}

		digest, err := signedMessageDigest(code)
		if err != nil {
			return &pokerrpc.LoginResponse{
				Ok:    false,
				Error: fmt.Sprintf("failed to build signed message payload: %v", err),
			}, nil
		}

		pub, _, err := ecdsa.RecoverCompact(sig, digest)
		if err != nil {
			return &pokerrpc.LoginResponse{
				Ok:    false,
				Error: fmt.Sprintf("failed to recover pubkey: %v", err),
			}, nil
		}
		got, err := stdaddr.NewAddressPubKeyHashEcdsaSecp256k1V0(stdaddr.Hash160(pub.SerializeCompressed()), params)
		if err != nil {
			return &pokerrpc.LoginResponse{
				Ok:    false,
				Error: fmt.Sprintf("failed to compute address: %v", err),
			}, nil
		}
		if got.String() != addr.String() {
			return &pokerrpc.LoginResponse{
				Ok:    false,
				Error: "address does not match recovered signature",
			}, nil
		}

		s.auth.mu.Lock()
		if existing, ok := s.auth.uidToPayoutAddr[userID]; ok && existing != addrStr {
			s.auth.mu.Unlock()
			return &pokerrpc.LoginResponse{
				Ok:    false,
				Error: "payout address mismatch for this user id",
			}, nil
		}
		s.auth.uidToPayoutAddr[userID] = addrStr
		s.auth.mu.Unlock()

		// Persist payout address to database for long-term storage.
		if err := s.db.UpdateAuthUserPayoutAddress(ctx, userID.String(), addrStr); err != nil {
			s.log.Warnf("Failed to update payout address: %v", err)
		}
	}

	// Update last login in database
	if err := s.db.UpdateAuthUserLastLogin(ctx, userID.String()); err != nil {
		// Log but don't fail login
		s.log.Warnf("Failed to update last login: %v", err)
	}

	// Update in-memory cache
	s.auth.mu.Lock()
	s.auth.uidToNickname[userID] = nickname
	if meta, ok := s.auth.userMetadata[userID]; ok {
		meta.Nickname = nickname
		meta.LastLogin = time.Now()
		s.auth.userMetadata[userID] = meta
	} else {
		s.auth.userMetadata[userID] = UserMetadata{
			Nickname:  nickname,
			Created:   authUser.CreatedAt,
			LastLogin: time.Now(),
		}
	}
	s.auth.mu.Unlock()

	// Create session token
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		return &pokerrpc.LoginResponse{
			Ok:    false,
			Error: "failed to generate session token",
		}, nil
	}
	token := fmt.Sprintf("sess_%d_%x", time.Now().Unix(), tokenBytes)

	// Derive current payout address hint (may have been loaded from DB or just set).
	payoutAddr := s.auth.payoutAddrHint(userID)

	s.auth.mu.Lock()
	s.auth.sessions[token] = sessionInfo{
		userID:     userID,
		nickname:   nickname,
		payoutAddr: payoutAddr,
		created:    time.Now(),
	}
	s.auth.mu.Unlock()

	return &pokerrpc.LoginResponse{
		Ok:            true,
		Token:         token,
		UserId:        userID.String(),
		Nickname:      nickname,
		PayoutAddress: payoutAddr,
	}, nil
}

// Logout handles user logout
func (s *Server) Logout(ctx context.Context, req *pokerrpc.LogoutRequest) (*pokerrpc.LogoutResponse, error) {
	if s.auth == nil {
		return &pokerrpc.LogoutResponse{Ok: true}, nil
	}

	s.auth.mu.Lock()
	defer s.auth.mu.Unlock()

	delete(s.auth.sessions, req.Token)

	return &pokerrpc.LogoutResponse{Ok: true}, nil
}

// GetUserInfo retrieves user information
func (s *Server) GetUserInfo(ctx context.Context, req *pokerrpc.GetUserInfoRequest) (*pokerrpc.GetUserInfoResponse, error) {
	if s.auth == nil {
		return &pokerrpc.GetUserInfoResponse{}, nil
	}

	s.auth.mu.RLock()
	defer s.auth.mu.RUnlock()

	session, exists := s.auth.sessions[req.Token]
	if !exists {
		return &pokerrpc.GetUserInfoResponse{}, nil
	}

	meta, ok := s.auth.userMetadata[session.userID]
	if !ok {
		return &pokerrpc.GetUserInfoResponse{
			UserId:        session.userID.String(),
			Nickname:      session.nickname,
			PayoutAddress: session.payoutAddr,
		}, nil
	}

	return &pokerrpc.GetUserInfoResponse{
		UserId:        session.userID.String(),
		Nickname:      meta.Nickname,
		Created:       meta.Created.Unix(),
		LastLogin:     meta.LastLogin.Unix(),
		PayoutAddress: session.payoutAddr,
	}, nil
}
