# Nickname-Based Authentication System

## Overview

This document describes a simple authentication system where users authenticate using their cryptographic user ID (derived from a seed key), while providing a nickname for UI display purposes. The system automatically manages cryptographic seed keys behind the scenes. This design provides a user-friendly experience while establishing the infrastructure needed for future Schnorr settlement features.

## Design Philosophy

**Start Simple, Build Smart:**
- **User Experience**: Users simply enter a nickname to login (no wallet signatures, no complex setup)
- **Cryptographic Foundation**: Behind the scenes, the system generates and manages a persistent seed private key
- **Future-Ready**: The seed key infrastructure is already in place for Schnorr settlement, multi-address support, and advanced features

## Core Components

### 1. User Nickname
- **User-Facing**: Simple text string (e.g., "alice", "bob123")
- **Purpose**: Human-readable identifier for UI display only (not used for authentication)
- **Constraints**: 
  - Minimum length: 3 characters
  - Maximum length: 32 characters
  - Allowed: alphanumeric, underscore, hyphen
  - **Not unique**: Multiple users can have the same nickname
  - Case-sensitive (optional: normalize to lowercase)

### 2. Seed Private Key
- **Hidden from User**: Automatically generated and managed
- **Purpose**: Cryptographic foundation for user identity
- **Generation**: 32-byte secp256k1 private key
- **Storage**: Client-side, persisted to disk in `SessionKeyData` structure (reuses existing `settlement_session_key.json` infrastructure)
- **Lifetime**: Persistent across sessions (not ephemeral, unlike session keys)

### 3. User ID
- **Derived from Seed**: `chainhash.HashB(seed_bytes)`
- **Purpose**: Stable, cryptographic user identifier
- **Format**: `zkidentity.ShortID` (33 bytes)
- **Properties**: Deterministic, non-reversible, stable

## Authentication Flow

### First-Time User (Registration)

1. **User enters nickname** (e.g., "alice")
2. **Client checks if seed exists**:
   ```go
   seedExists := checkSeedKeyExists()
   ```
3. **If no seed exists, generate one**:
   ```go
   seedPriv, err := secp256k1.GeneratePrivateKey()
   seedPrivHex := hex.EncodeToString(seedPriv.Serialize())
   saveSeedKey(seedPrivHex)  // Persist to disk
   ```
4. **Derive user ID from seed**:
   ```go
   seedBytes := seedPriv.Serialize()  // 32 bytes
   userIDBytes := chainhash.HashB(seedBytes)  // Double BLAKE-256
   var userID zkidentity.ShortID
   userID.FromBytes(userIDBytes[:])
   ```
5. **Send registration request**:
   ```
   Client → Server: Register({
     nickname: "alice",
     user_id: userID.String()
   })
   ```
6. **Server validates and stores**:
   - Validates nickname format
   - Stores user with `user_id` as primary key and `nickname` for display
   - Returns success

### Returning User (Login)

1. **User enters nickname** (e.g., "alice")
2. **Client loads seed key**:
   ```go
   seedPrivHex := loadSeedKey()  // From disk
   ```
3. **Derive user ID from seed**:
   ```go
   seedBytes, _ := hex.DecodeString(seedPrivHex)
   userIDBytes := chainhash.HashB(seedBytes)
   var userID zkidentity.ShortID
   userID.FromBytes(userIDBytes[:])
   ```
4. **Send login request**:
   ```
   Client → Server: Login({
     nickname: "alice",
     user_id: userID.String()
   })
   ```
5. **Server validates**:
   - Authenticates by `user_id` only (looks up user in database by user_id)
   - Stores/updates nickname for UI display (nickname is not used for authentication)
   - Creates session token
   - Returns token and user info

## Implementation Details

### Client-Side

#### Seed Key Management

```go
// Generate or load seed key
func (pc *PongClient) GetOrCreateSeedKey() (string, error) {
    // Try to load existing seed from SessionKeyData
    if seedHex, err := pc.loadSeedKey(); err == nil && seedHex != "" {
        return seedHex, nil
    }
    
    // Generate new seed
    seedPriv, err := secp256k1.GeneratePrivateKey()
    if err != nil {
        return "", fmt.Errorf("generate seed: %w", err)
    }
    seedHex := hex.EncodeToString(seedPriv.Serialize())
    
    // Save to disk (updates SessionKeyData with SeedHex field)
    if err := pc.saveSeedKey(seedHex); err != nil {
        return "", fmt.Errorf("save seed: %w", err)
    }
    
    return seedHex, nil
}

// Derive user ID from seed
func (pc *PongClient) DeriveUserID(seedHex string) (string, error) {
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

// Get user ID (loads seed if needed)
func (pc *PongClient) GetUserID() (string, error) {
    seedHex, err := pc.GetOrCreateSeedKey()
    if err != nil {
        return "", err
    }
    return pc.DeriveUserID(seedHex)
}
```

#### Registration

```go
func (pc *PongClient) Register(nickname string) error {
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
    req := &pong.RegisterRequest{
        Nickname: nickname,
        UserId:   userID,
    }
    
    resp, err := pc.authClient.Register(ctx, req)
    if err != nil {
        return err
    }
    
    if !resp.Ok {
        return fmt.Errorf("registration failed: %s", resp.Error)
    }
    
    return nil
}
```

#### Login

```go
func (pc *PongClient) Login(nickname string) (*LoginResponse, error) {
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
    req := &pong.LoginRequest{
        Nickname: nickname,
        UserId:   userID,
    }
    
    resp, err := pc.authClient.Login(ctx, req)
    if err != nil {
        return nil, err
    }
    
    if !resp.Ok {
        return nil, fmt.Errorf("login failed: %s", resp.Error)
    }
    
    // Store session
    pc.setSession(resp.Token, resp.UserId, nickname)
    
    return &LoginResponse{
        Token:    resp.Token,
        UserId:   resp.UserId,
        Nickname: nickname,
    }, nil
}
```

#### Seed Key Storage

We extend the existing `SessionKeyData` structure to include the seed key. This reuses the existing infrastructure for key storage and persistence.

```go
// Extended SessionKeyData (add SeedHex field)
type SessionKeyData struct {
    Priv                  string      `json:"priv"`                    // Session private key (ephemeral)
    Pub                   string      `json:"pub"`                     // Session public key (ephemeral)
    SeedHex               string      `json:"seed_hex,omitempty"`       // User seed key (persistent) - NEW
    EscrowInfo            *EscrowInfo `json:"escrow_info,omitempty"`
    WalletAddress         string      `json:"wallet_address,omitempty"`
    PayoutAddressOrPubkey string      `json:"payout_address_or_pubkey,omitempty"`
}

// Reuse existing session key file path
// File: settlement_session_key.json (or user_identity.json for clarity)
func (pc *PongClient) userIdentityFilePath() string {
    if pc.appCfg == nil || pc.appCfg.DataDir == "" {
        return ""
    }
    // Option 1: Reuse existing file
    return filepath.Join(pc.appCfg.DataDir, "settlement_session_key.json")
    // Option 2: Separate file for clarity
    // return filepath.Join(pc.appCfg.DataDir, "user_identity.json")
}

func (pc *PongClient) saveSeedKey(seedHex string) error {
    path := pc.userIdentityFilePath()
    if path == "" {
        return fmt.Errorf("no data directory configured")
    }
    
    // Load existing data if file exists
    var data SessionKeyData
    if existingData, err := os.ReadFile(path); err == nil {
        json.Unmarshal(existingData, &data)
    }
    
    // Update seed key
    data.SeedHex = seedHex
    
    // Save (reuse existing saveSettlementSessionKey logic)
    blob, err := json.MarshalIndent(data, "", "  ")
    if err != nil {
        return err
    }
    
    if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
        return err
    }
    
    return os.WriteFile(path, blob, 0600)  // Read/write for owner only
}

func (pc *PongClient) loadSeedKey() (string, error) {
    path := pc.userIdentityFilePath()
    if path == "" {
        return "", fmt.Errorf("no data directory configured")
    }
    
    data, err := os.ReadFile(path)
    if err != nil {
        if os.IsNotExist(err) {
            return "", nil  // No seed exists yet
        }
        return "", err
    }
    
    var keyData SessionKeyData
    if err := json.Unmarshal(data, &keyData); err != nil {
        return "", err
    }
    
    if keyData.SeedHex == "" {
        return "", nil  // No seed in file
    }
    
    // Validate seed format
    seedBytes, err := hex.DecodeString(keyData.SeedHex)
    if err != nil || len(seedBytes) != 32 {
        return "", fmt.Errorf("invalid seed format")
    }
    
    return keyData.SeedHex, nil
}
```

**Benefits of reusing SessionKeyData:**
- ✅ Reuses existing file infrastructure
- ✅ Same security (0600 permissions)
- ✅ Same storage location
- ✅ Can coexist with session keys
- ✅ Minimal code changes

### Server-Side

#### Database Schema

```sql
-- Authentication: user_id is primary key, nickname is just for UI display (not unique)
CREATE TABLE IF NOT EXISTS auth_users (
    user_id     TEXT PRIMARY KEY,
    nickname    TEXT NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_login  TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_auth_users_user_id ON auth_users(user_id);
```

**Key Points:**
- `user_id` is the PRIMARY KEY (unique, used for authentication)
- `nickname` is a regular column (not unique, multiple users can share nicknames)
- Authentication is performed by `user_id` only
- `nickname` is stored for UI display purposes

#### Data Structures

```go
type authState struct {
    mu sync.RWMutex
    
    // User ID → Nickname mapping (each user has one nickname for display)
    uidToNickname map[zkidentity.ShortID]string
    
    // Active sessions: token → session info
    sessions map[string]sessionInfo
    
    // User ID → metadata (for future use)
    userMetadata map[zkidentity.ShortID]UserMetadata
}

type sessionInfo struct {
    userID   zkidentity.ShortID
    nickname string
    created  time.Time
}

type UserMetadata struct {
    Nickname    string
    Created     time.Time
    LastLogin   time.Time
    // Future: addresses, preferences, etc.
}
```

#### Registration Handler

```go
func (s *Server) Register(ctx context.Context, req *pong.RegisterRequest) (*pong.RegisterResponse, error) {
    // Validate nickname format (but don't check uniqueness)
    nickname := strings.TrimSpace(req.Nickname)
    if err := validateNickname(nickname); err != nil {
        return &pong.RegisterResponse{
            Ok:    false,
            Error: err.Error(),
        }, nil
    }
    
    // Parse user ID
    var userID zkidentity.ShortID
    if err := userID.FromString(req.UserId); err != nil {
        return &pong.RegisterResponse{
            Ok:    false,
            Error: "invalid user ID format",
        }, nil
    }
    
    // Persist to database (user_id is primary key, nickname is just for display)
    if err := s.db.UpsertAuthUser(ctx, nickname, userID.String()); err != nil {
        return &pong.RegisterResponse{
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
    
    return &pong.RegisterResponse{
        Ok: true,
    }, nil
}
```

#### Login Handler

```go
func (s *Server) Login(ctx context.Context, req *pong.LoginRequest) (*pong.LoginResponse, error) {
    // Validate nickname format (but don't check uniqueness - it's just for UI)
    nickname := strings.TrimSpace(req.Nickname)
    if err := validateNickname(nickname); err != nil {
        return &pong.LoginResponse{
            Ok:    false,
            Error: err.Error(),
        }, nil
    }
    
    // Parse user ID
    var userID zkidentity.ShortID
    if err := userID.FromString(req.UserId); err != nil {
        return &pong.LoginResponse{
            Ok:    false,
            Error: "invalid user ID format",
        }, nil
    }
    
    // Authenticate by user ID only - check if user exists in database
    authUser, err := s.db.GetAuthUserByUserID(ctx, userID.String())
    if err != nil {
        return &pong.LoginResponse{
            Ok:    false,
            Error: "user not found - please register first",
        }, nil
    }
    
    // Store/update nickname in database (nickname is just for UI display)
    if err := s.db.UpsertAuthUser(ctx, nickname, userID.String()); err != nil {
        return &pong.LoginResponse{
            Ok:    false,
            Error: fmt.Sprintf("failed to update nickname: %v", err),
        }, nil
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
        return &pong.LoginResponse{
            Ok:    false,
            Error: "failed to generate session token",
        }, nil
    }
    token := fmt.Sprintf("sess_%d_%x", time.Now().Unix(), tokenBytes)
    s.auth.sessions[token] = sessionInfo{
        userID:   userID,
        nickname: nickname,
        created:  time.Now(),
    }
    
    return &pong.LoginResponse{
        Ok:       true,
        Token:    token,
        UserId:   userID.String(),
        Nickname: nickname,
    }, nil
}
```

#### Nickname Validation

```go
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
```

## Protocol Definition

### gRPC Service

```protobuf
service AuthService {
  rpc Register (RegisterRequest) returns (RegisterResponse);
  rpc Login (LoginRequest) returns (LoginResponse);
  rpc Logout (LogoutRequest) returns (LogoutResponse);
  rpc GetUserInfo (GetUserInfoRequest) returns (GetUserInfoResponse);
}

message RegisterRequest {
  string nickname = 1;
  string user_id = 2;  // Derived from seed key
}

message RegisterResponse {
  bool ok = 1;
  string error = 2;  // Empty if ok == true
}

message LoginRequest {
  string nickname = 1;
  string user_id = 2;  // Derived from seed key
}

message LoginResponse {
  bool ok = 1;
  string error = 2;
  string token = 3;     // Session token
  string user_id = 4;
  string nickname = 5;
}

message LogoutRequest {
  string token = 1;
}

message LogoutResponse {
  bool ok = 1;
}

message GetUserInfoRequest {
  string token = 1;
}

message GetUserInfoResponse {
  string user_id = 1;
  string nickname = 2;
  int64 created = 3;
  int64 last_login = 4;
}
```

## User Experience Flow

### First Launch

1. User opens app
2. App shows login screen with nickname input
3. User enters nickname (e.g., "alice")
4. User clicks "Login" or "Register"
5. **Behind the scenes**:
   - App generates seed key (if first time)
   - App derives user ID from seed
   - App sends registration/login request
6. User is logged in and sees main screen

### Subsequent Launches

1. User opens app
2. App shows login screen
3. User enters nickname
4. User clicks "Login"
5. **Behind the scenes**:
   - App loads seed key from disk
   - App derives user ID from seed
   - App sends login request
6. User is logged in

### Key Points for Users

- **Simple**: Just enter a nickname
- **No Wallet Required**: No need to sign messages or manage addresses
- **Persistent**: Same seed key always logs into same account (user ID is stable)
- **Secure**: Cryptographic identity managed automatically
- **Nickname Freedom**: Any user can use any nickname - it's just for UI display

## Security Considerations

### Seed Key Protection

1. **Storage**: Seed key stored with permissions 0600 (owner read/write only)
2. **Location**: Client data directory (user-controlled)
3. **Backup**: Users should backup seed key file (losing it = losing account)
4. **Encryption**: Future enhancement: encrypt seed key with user password

### User ID Security

1. **Deterministic**: Same seed always produces same user ID
2. **Non-Reversible**: User ID doesn't reveal seed key (one-way hash)
3. **Collision Resistance**: Double hash provides strong collision resistance
4. **Stable**: User ID doesn't change (stable identity)

### Nickname Security

1. **Validation**: Server validates nickname format
2. **Not Unique**: Multiple users can have the same nickname (nickname is UI-only, not used for authentication)
3. **Case Sensitivity**: Decide on case handling (recommend: case-insensitive lookup, case-preserving storage)

### Session Security

1. **Token Generation**: Cryptographically random tokens
2. **Token Expiry**: Tokens expire after inactivity (e.g., 24 hours)
3. **Token Validation**: Server validates tokens on each request

## Future Enhancements

### Phase 2: Wallet Integration

When adding wallet-based features:

1. **Multiple Addresses**: User can link wallet addresses to account
2. **Address Authentication**: Prove ownership via signature
3. **Payout Addresses**: Use linked addresses for escrow payouts

**Implementation**: Seed key already exists, just add address linking:
```go
// User links address to account
func (pc *PongClient) LinkAddress(address string, signature string) error {
    userID, _ := pc.GetUserID()
    // Send address + signature + userID to server
    // Server verifies signature and links address → userID
}
```

### Phase 3: Schnorr Settlement

When adding Schnorr settlement:

1. **Session Keys**: Generate ephemeral keys per escrow (already implemented)
2. **Seed Key**: Use seed key for user identity (already implemented)
3. **Integration**: Link settlement sessions to user ID

**Implementation**: Infrastructure already in place:
- Seed key management: ✅
- User ID derivation: ✅
- Session key generation: ✅ (existing code)

### Phase 4: Advanced Features

1. **Password Protection**: Encrypt seed key with user password
2. **Multi-Device**: Sync seed key across devices (encrypted)
3. **Recovery**: Seed key recovery mechanism
4. **Privacy**: Users can use any nickname (not tied to identity), user ID remains stable

## Migration Path

### From Current System

If migrating from address-based auth:

1. **New Users**: Use nickname-based system
2. **Existing Users**: 
   - Option A: Generate seed, map old address to new user ID
   - Option B: Keep old system for existing, new for new users
   - Option C: Force migration (users choose nickname, generate seed)

### To Wallet-Based System

When adding wallet features:

1. **Backward Compatible**: Nickname login still works
2. **Add Address Linking**: Users can optionally link addresses
3. **Gradual Migration**: Users can migrate to address-based auth over time

## Benefits

1. **User-Friendly**: Simple nickname login (no crypto knowledge needed)
2. **Cryptographic Foundation**: Seed key infrastructure ready for advanced features
3. **Future-Proof**: Easy to add wallet integration, Schnorr settlement, etc.
4. **Secure**: Cryptographic identity with simple UX
5. **Scalable**: Can add features incrementally without breaking changes

## Implementation Checklist

### Client-Side
- [ ] Seed key generation (`GenerateSeedKey`)
- [ ] Seed key storage (`saveSeedKey`, `loadSeedKey`)
- [ ] User ID derivation (`DeriveUserID`)
- [ ] Registration flow (`Register`)
- [ ] Login flow (`Login`)
- [ ] Nickname validation
- [ ] Session management

### Server-Side
- [ ] Registration handler (`Register`)
- [ ] Login handler (`Login`)
- [ ] Nickname validation
- [ ] User ID authentication (by user_id only)
- [ ] Session token management
- [ ] Data structures (`uidToNickname` - nickname is UI-only)
- [ ] Database schema (user_id as primary key)

### Protocol
- [ ] gRPC service definition
- [ ] Protobuf messages
- [ ] Error handling

### Testing
- [ ] Seed key generation and persistence
- [ ] User ID derivation (deterministic)
- [ ] Registration flow
- [ ] Login flow
- [ ] Nickname validation
- [ ] Session management

## References

```
// GenerateNewSettlementSessionKey always creates a new session key and overwrites the cached one.
func (pc *PongClient) GenerateNewSettlementSessionKey() (string, string, error) {
	pc.Lock()
	p, err := secp256k1.GeneratePrivateKey()
	if err != nil {
		pc.Unlock()
		return "", "", err
	}
	pc.settlePrivHex = hex.EncodeToString(p.Serialize())
	pc.settlePubHex = hex.EncodeToString(p.PubKey().SerializeCompressed())
	pc.Unlock()
	if err := pc.saveSettlementSessionKey(); err != nil {
		return "", "", fmt.Errorf("save session key: %w", err)
	}
	return pc.settlePrivHex, pc.settlePubHex, nil
}
```

- Double hash: `chainhash.HashB()` (double BLAKE-256)
- User ID format: `zkidentity.ShortID` (33 bytes)
- Session key storage: `client/client.go:403` - 
```
// saveSettlementSessionKey writes the current session keypair to disk (0600) in JSON.
func (pc *PongClient) saveSettlementSessionKey() error {
	path := pc.sessionKeyFilePath()
	if strings.TrimSpace(path) == "" {
		return nil // no datadir configured; skip persistence in POC mode
	}
	pc.RLock()
	data := SessionKeyData{
		Priv:                  pc.settlePrivHex,
		Pub:                   pc.settlePubHex,
		WalletAddress:         pc.walletAddress,
		PayoutAddressOrPubkey: pc.payoutAddressOrPubkey,
	}
	if pc.activeEscrowInfo != nil {
		copyInfo := *pc.activeEscrowInfo
		data.EscrowInfo = &copyInfo
	}
	pc.RUnlock()
	blob, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	return os.WriteFile(path, blob, 0600)
}
```
- SessionKeyData structure:

```
// EscrowInfo represents the data we need to store about an escrow for potential refund
type EscrowInfo struct {
	EscrowID        string `json:"escrow_id"`
	DepositAddress  string `json:"deposit_address,omitempty"`
	FundingTxid     string `json:"funding_txid"`
	FundingVout     uint32 `json:"funding_vout"`
	FundedAmount    uint64 `json:"funded_amount"`
	RedeemScriptHex string `json:"redeem_script_hex"`
	PKScriptHex     string `json:"pk_script_hex"`
	CSVBlocks       uint32 `json:"csv_blocks"`
	// Status is a simple lifecycle marker for UI/UX such as
	// "paid" (settled by match) or "tx built" (refund tx constructed).
	Status          string `json:"status,omitempty"`
	ArchivedAt      int64  `json:"archived_at"`
	FundingVoutSet  bool   `json:"-"`
	FundedAmountSet bool   `json:"-"`
	CSVBlocksSet    bool   `json:"-"`
}
```

```
// SessionKeyData includes both the keypair and escrow info for archiving
type SessionKeyData struct {
	Priv                  string      `json:"priv"`
	Pub                   string      `json:"pub"`
	EscrowInfo            *EscrowInfo `json:"escrow_info,omitempty"`
	WalletAddress         string      `json:"wallet_address,omitempty"`
	PayoutAddressOrPubkey string      `json:"payout_address_or_pubkey,omitempty"`
}
```

- File path: use datadir
