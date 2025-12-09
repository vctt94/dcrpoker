package golib

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"path/filepath"
	"strings"
	"testing"

	"github.com/companyzero/bisonrelay/zkidentity"
	"github.com/stretchr/testify/require"
	"github.com/vctt94/bisonbotkit/logging"
	"github.com/vctt94/pokerbisonrelay/pkg/client"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	"github.com/vctt94/pokerbisonrelay/pkg/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

// This test reproduces the cache-spam scenario: invalid escrow funding
// updates (wrong amount) for multiple escrows that all reference the same
// funding outpoint are cached individually, leading to duplicate dropdown
// values in the Flutter bind dialog.
func TestHandleEscrowNotificationCachesInvalidFundingUpdates(t *testing.T) {
	t.Helper()

	// Use a deterministic player ID for the notification payload.
	var pid zkidentity.ShortID
	if err := pid.FromString(strings.Repeat("01", 32)); err != nil {
		t.Fatalf("short id: %v", err)
	}

	tmp := t.TempDir()
	pc := &client.PokerClient{DataDir: tmp}
	cctx := &clientCtx{ID: pid, c: pc}

	makeNotification := func(escrowID string) *pokerrpc.Notification {
		payload := map[string]interface{}{
			"type":          "escrow_funding",
			"player_id":     pid.String(),
			"escrow_id":     escrowID,
			"funding_txid":  "ea729de5f1f0e185359c1f43b258bf06a7a1ff646f64451081713cbb0600a527",
			"funding_vout":  0,
			"amount_atoms":  float64(10_000_000), // underfunded vs expected
			"csv_blocks":    float64(64),
			"funding_state": "ESCROW_STATE_INVALID",
			"error":         "expected funding 100000000 but found 10000000",
			"utxo_count":    1,
		}
		b, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal payload: %v", err)
		}
		return &pokerrpc.Notification{
			Type:     pokerrpc.NotificationType_ESCROW_FUNDING,
			Message:  string(b),
			PlayerId: pid.String(),
		}
	}

	// Two escrows, same funding outpoint, both invalid.
	handleEscrowNotification(cctx, makeNotification("escrow-one"))
	handleEscrowNotification(cctx, makeNotification("escrow-two"))

	// Refund construction needs the cached funding outpoint per escrow.
	escrowOne, err := cctx.c.GetEscrowById("escrow-one")
	require.NoError(t, err)
	escrowTwo, err := cctx.c.GetEscrowById("escrow-two")
	require.NoError(t, err)

	require.Equal(t, "funding_error", escrowOne["status"])
	require.Equal(t, "funding_error", escrowTwo["status"])
	require.Equal(t, "ea729de5f1f0e185359c1f43b258bf06a7a1ff646f64451081713cbb0600a527", escrowOne["funding_txid"])
	require.Equal(t, "ea729de5f1f0e185359c1f43b258bf06a7a1ff646f64451081713cbb0600a527", escrowTwo["funding_txid"])
	require.EqualValues(t, 0, escrowOne["funding_vout"])
	require.EqualValues(t, 0, escrowTwo["funding_vout"])
	require.EqualValues(t, 10_000_000, escrowOne["funded_amount"])
	require.EqualValues(t, 10_000_000, escrowTwo["funded_amount"])

}

// TestResumeSessionPayoutAddressSync tests that ResumeSession syncs the payout
// address from the server when it matches the local config, or returns empty
// when the server doesn't have one.
//
// Bug scenario:
// 1. User logs in with payout address in config file
// 2. Login saves config payout address to session file (pkg/client/auth.go:395)
// 3. User never calls SetPayoutAddress, so server's session has empty payout address
// 4. ResumeSession returns session.PayoutAddress from saved file (has address)
// 5. But server's session.payoutAddr is empty
// 6. Client shows hasAuthedPayoutAddress: true, but OpenEscrow fails
//
// Expected fix: ResumeSession should:
// - Get payout address from server (GetUserInfo)
// - Compare with local config payout address
// - If they match, update session.PayoutAddress to that value (verified, no signing needed)
// - If server has empty or doesn't match, set session.PayoutAddress to empty (not verified)
//
// This test should FAIL until the bug is fixed in pkg/client/auth.go:ResumeSession
func TestResumeSessionPayoutAddressSync(t *testing.T) {
	t.Helper()

	// Set up test server
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "poker.sqlite")
	database, err := server.NewDatabase(dbPath)
	require.NoError(t, err)
	defer database.Close()

	logBackend, err := logging.NewLogBackend(logging.LogConfig{
		LogFile:        "",
		DebugLevel:     "error",
		MaxLogFiles:    1,
		MaxBufferLines: 100,
	})
	if err != nil {
		logBackend = &logging.LogBackend{}
	}
	pokerSrv, err := server.NewTestServer(database, logBackend)
	require.NoError(t, err)
	defer pokerSrv.Stop()

	lis := bufconn.Listen(1024 * 1024)
	grpcSrv := grpc.NewServer()
	pokerrpc.RegisterLobbyServiceServer(grpcSrv, pokerSrv)
	pokerrpc.RegisterPokerServiceServer(grpcSrv, pokerSrv)
	pokerrpc.RegisterAuthServiceServer(grpcSrv, pokerSrv)
	pokerrpc.RegisterPokerRefereeServer(grpcSrv, pokerSrv)
	go func() { _ = grpcSrv.Serve(lis) }()
	defer grpcSrv.Stop()

	dialer := func(ctx context.Context, _ string) (net.Conn, error) { return lis.Dial() }
	dialTarget := "bufnet"
	dialOpts := []grpc.DialOption{
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	// Set up client
	clientDir := t.TempDir()
	var uid zkidentity.ShortID
	uidStr := "0000000000000000000000000000000000000000000000000000000000000003"
	require.NoError(t, uid.FromString(uidStr))

	localPayoutAddr := "TsRnk22spGQJTpKFcRBc281rmfNFpywh337"
	token := "test-token-empty"

	// Create client config with payout address
	cfg := &client.ClientConfig{
		Datadir:       clientDir,
		PayoutAddress: localPayoutAddr, // Client config has payout address
		LogBackend:    logBackend,
		Notifications: client.NewNotificationManager(),
	}

	pc, err := client.NewPokerClientWithDialOptions(context.Background(), cfg, dialTarget, dialOpts...)
	require.NoError(t, err)

	// Get the client's actual userID (it will be generated from seed)
	clientUserID, err := pc.GetUserID()
	require.NoError(t, err)

	// Parse client userID to ShortID for server
	var clientUID zkidentity.ShortID
	require.NoError(t, clientUID.FromString(clientUserID))

	// Create session on server WITHOUT payout address (simulating user who logged in
	// but never called SetPayoutAddress), using client's actual userID
	pokerSrv.TestSeedSession(token, clientUID, "", "testuser")

	// Simulate Login saving session file with payout address from config
	// (This is what happens in pkg/client/auth.go:395)
	session := &client.SessionData{
		Token:         token,
		UserID:        clientUserID, // Use client's actual userID
		Nickname:      "testuser",
		PayoutAddress: localPayoutAddr, // Saved from config during Login
	}
	err = pc.SaveSession(session)
	require.NoError(t, err)

	// Now call ResumeSession - it should sync from server
	// Expected: Since server has empty payout address, session.PayoutAddress should be empty
	// BUG: Currently it returns the saved file value (localPayoutAddr)
	resumed, err := pc.ResumeSession(context.Background())
	require.NoError(t, err)
	require.NotNil(t, resumed, "ResumeSession should return session")

	// BUG: Currently this fails because ResumeSession doesn't sync from server
	// Expected: session.PayoutAddress should be empty (server doesn't have it)
	// Actual: session.PayoutAddress is localPayoutAddr (from saved file)
	if resumed.PayoutAddress != "" {
		t.Fatalf("BUG: ResumeSession should return empty payout address when server has empty. "+
			"Got: %s, Expected: empty. "+
			"Fix in pkg/client/auth.go:ResumeSession to sync session.PayoutAddress from server's GetUserInfo response.",
			resumed.PayoutAddress)
	}

	// Scenario 2: Server has payout address that matches local config
	// Expected: ResumeSession should return the payout address (they match, verified)
	token2 := "test-token-match"
	pokerSrv.TestSeedSession(token2, clientUID, localPayoutAddr, "testuser2")

	session2 := &client.SessionData{
		Token:         token2,
		UserID:        clientUserID, // Use client's actual userID
		Nickname:      "testuser2",
		PayoutAddress: localPayoutAddr, // Saved from config
	}
	err = pc.SaveSession(session2)
	require.NoError(t, err)

	resumed2, err := pc.ResumeSession(context.Background())
	require.NoError(t, err)
	require.NotNil(t, resumed2)

	// Expected: When server and local match, use the server's value (verified)
	// This should work after the fix
	if resumed2.PayoutAddress != localPayoutAddr {
		t.Fatalf("When server and local config match, ResumeSession should return the payout address. "+
			"Got: %s, Expected: %s",
			resumed2.PayoutAddress, localPayoutAddr)
	}
}

// This test locks in the decision to start session key indices at 1 to avoid
// dropping key_index when persisting escrow metadata (omitempty skips zeros).
func TestSessionKeysStartAtOne(t *testing.T) {
	t.Helper()

	tmp := t.TempDir()
	pc := &client.PokerClient{DataDir: tmp}

	priv, pub, idx, err := pc.GenerateSessionKey()
	require.NoError(t, err)
	require.EqualValues(t, 1, idx, "first generated session key should use index 1")
	require.NotEmpty(t, priv)
	require.NotEmpty(t, pub)

	err = pc.CacheEscrowInfo(&client.EscrowInfo{
		EscrowID: "escrow-one",
		Status:   "opened",
		KeyIndex: uint32(idx), // should be retained in cache (non-zero)
	})
	require.NoError(t, err)

	info, err := pc.GetEscrowById("escrow-one")
	require.NoError(t, err)

	if v, ok := info["key_index"]; !ok || fmt.Sprint(v) != "1" {
		t.Fatalf("expected cached escrow to include key_index=1, got %v", v)
	}
}
