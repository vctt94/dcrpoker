package e2e

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/companyzero/bisonrelay/zkidentity"
	"github.com/decred/dcrd/chaincfg/chainhash"
	"github.com/decred/dcrd/chaincfg/v3"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/txscript/v4/stdaddr"
	"github.com/decred/dcrd/wire"
	"github.com/stretchr/testify/require"
	testenv "github.com/vctt94/pokerbisonrelay/e2e/internal/testenv"
	"github.com/vctt94/pokerbisonrelay/pkg/client"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	"github.com/vctt94/pokerbisonrelay/pkg/server"
)

func settlementTestBuyIn() uint64 {
	// Ensure at least two seats cover the fixed settlement fee with some buffer.
	return server.DefaultSettlementFeeAtoms/2 + 1_000
}

// TestRefereePresignFlow exercises the client referee helper through the UI stubs:
// - two players login, open escrow with session key, bind to match/table/seat via SettlementHello,
// - presign completes for both branches.
func TestRefereePresignFlow(t *testing.T) {
	t.Parallel()
	env := testenv.New(t)
	defer env.Close()

	ctx := context.Background()

	buyIn := settlementTestBuyIn()

	// Create table (2 players).
	tableID := env.CreateTableWithBuyIn(ctx, "alice", 2, 2, int64(buyIn))

	// Seed auth sessions with tokens and payout addresses (bypass signed login for test).
	seedSession := func(tok, uid, payout string) {
		var uidShort zkidentity.ShortID
		_ = uidShort.FromString(uid)
		env.PokerSrv.TestSeedSession(tok, uidShort, payout, uid)
	}
	seedSession("alice-token", "alice", "TsRnk22spGQJTpKFcRBc281rmfNFpywh337")
	seedSession("bob-token", "bob", "TsgsQwSZTkbXPGdFBg5z3wthjkQs1EeKcJ5")

	// Create PokerClients on the same conn.
	logBackend := testenv.NewLogBackend()
	pcAlice, err := client.NewPokerClientWithDialOptions(ctx, &client.ClientConfig{
		Datadir:       t.TempDir(),
		LogBackend:    logBackend,
		Notifications: client.NewNotificationManager(),
	}, env.DialTarget(), env.DialOptions()...)
	require.NoError(t, err)
	pcBob, err := client.NewPokerClientWithDialOptions(ctx, &client.ClientConfig{
		Datadir:       t.TempDir(),
		LogBackend:    logBackend,
		Notifications: client.NewNotificationManager(),
	}, env.DialTarget(), env.DialOptions()...)
	require.NoError(t, err)

	// Stub login tokens: use existing ResumeSession which returns nil token; set manually.
	pcAliceToken := "alice-token"
	pcBobToken := "bob-token"

	// Generate session keys for escrow.
	alicePriv, _ := secp256k1.GeneratePrivateKey()
	alicePub := alicePriv.PubKey().SerializeCompressed()
	bobPriv, _ := secp256k1.GeneratePrivateKey()
	bobPub := bobPriv.PubKey().SerializeCompressed()

	// Open escrows (no table binding yet).
	refAlice := pcAlice.Referee(pcAliceToken)
	refBob := pcBob.Referee(pcBobToken)
	amount := buyIn // Must match table buy-in for referee binding
	escrowA, err := refAlice.OpenEscrow(ctx, amount, 64, alicePub)
	require.NoError(t, err)
	escrowB, err := refBob.OpenEscrow(ctx, amount, 64, bobPub)
	require.NoError(t, err)
	require.NotEmpty(t, escrowA.EscrowId)
	require.NotEmpty(t, escrowB.EscrowId)

	// Manually mark escrows as funded/bound (chainwatcher not exercised in test).
	env.PokerSrv.TestBindEscrowFunding(escrowA.EscrowId, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 0, amount)
	env.PokerSrv.TestBindEscrowFunding(escrowB.EscrowId, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 0, amount)

	// Bind and presign for a match.
	sessionID := "sess1"
	matchID := tableID + "|" + sessionID

	// Run presign concurrently for both players.
	errCh := make(chan error, 2)
	runPresign := func(ref *client.RefereeClient, seat uint32, escrowID string, pub []byte, privHex string) {
		const retries = 10
		var err error
		for i := 0; i < retries; i++ {
			err = ref.StartPresign(ctx, matchID, tableID, sessionID, seat, escrowID, pub, privHex)
			if err == nil {
				errCh <- nil
				return
			}
			if strings.Contains(err.Error(), "match seats not filled") {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			errCh <- err
			return
		}
		errCh <- fmt.Errorf("presign retries exhausted: %w", err)
	}
	go runPresign(refAlice, 0, escrowA.EscrowId, alicePub, hex.EncodeToString(alicePriv.Serialize()))
	go runPresign(refBob, 1, escrowB.EscrowId, bobPub, hex.EncodeToString(bobPriv.Serialize()))

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("presign timed out")
	}
	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("presign timed out (second)")
	}

	expectedBranch, err := env.PokerSrv.BranchIndexForSeat(matchID, 0)
	require.NoError(t, err)

	// Winner (alice) fetches finalize bundle for seat 0.
	bundle, err := refAlice.GetFinalizeBundle(ctx, matchID, 0)
	require.NoError(t, err)
	require.Equal(t, expectedBranch, bundle.Branch)
	assertFinalizeBundle(t, bundle, matchID, expectedBranch, []string{"TsRnk22spGQJTpKFcRBc281rmfNFpywh337", "TsgsQwSZTkbXPGdFBg5z3wthjkQs1EeKcJ5"}, amount, 2)
}

// TestRefereePresignFlowSixPlayers exercises presign/finalize with a full 6-max table.
func TestRefereePresignFlowSixPlayers(t *testing.T) {
	t.Parallel()
	env := testenv.New(t)
	defer env.Close()

	ctx := context.Background()

	players := []string{"p1", "p2", "p3", "p4", "p5", "p6"}
	payouts := []string{
		"TsnjFNHhZ17TKTLtSdXh9Z91TRHNsEp6N1d",
		"TsoxGYvsyhVooBMazDcntmjFq3ZpQCWMNCc",
		"Tsmy2RLwSbTsqmmSrmf6ma8Vsea8UAoZxUX",
		"TscMmNEjrniey3KukDh2ZDfVaVVVB6V6kYX",
		"TshxcBJTirEyYMZzL3ggP7jos8C16S64g2t",
		"TshjJ9kX7of5Jc1MihARYftaYqMp9dwnifW",
	}

	buyIn := settlementTestBuyIn()
	tableID := env.CreateTableWithBuyIn(ctx, "p1", 6, 6, int64(buyIn))
	sessionID := "sess6"
	matchID := tableID + "|" + sessionID

	seedSession := func(tok, uid, payout string) {
		var uidShort zkidentity.ShortID
		_ = uidShort.FromString(uid)
		env.PokerSrv.TestSeedSession(tok, uidShort, payout, uid)
	}
	for i, p := range players {
		seedSession(p+"-token", p, payouts[i])
	}

	logBackend := testenv.NewLogBackend()
	amount := buyIn // Must match table buy-in for referee binding
	type seatClient struct {
		ref      *client.RefereeClient
		pub      []byte
		privHex  string
		escrowID string
		seat     uint32
	}
	var seats []seatClient
	for i, p := range players {
		pc, err := client.NewPokerClientWithDialOptions(ctx, &client.ClientConfig{
			Datadir:       t.TempDir(),
			LogBackend:    logBackend,
			Notifications: client.NewNotificationManager(),
		}, env.DialTarget(), env.DialOptions()...)
		require.NoError(t, err)

		priv, _ := secp256k1.GeneratePrivateKey()
		pub := priv.PubKey().SerializeCompressed()
		token := p + "-token"
		ref := pc.Referee(token)
		esc, err := ref.OpenEscrow(ctx, amount, 64, pub)
		require.NoError(t, err)
		require.NotEmpty(t, esc.EscrowId)

		// Bind funding manually.
		env.PokerSrv.TestBindEscrowFunding(esc.EscrowId, fmt.Sprintf("%064x", i+1), 0, amount)

		seats = append(seats, seatClient{
			ref:      ref,
			pub:      pub,
			privHex:  hex.EncodeToString(priv.Serialize()),
			escrowID: esc.EscrowId,
			seat:     uint32(i),
		})
	}

	errCh := make(chan error, len(seats))
	runPresign := func(sc seatClient) {
		const retries = 20
		var err error
		for i := 0; i < retries; i++ {
			err = sc.ref.StartPresign(ctx, matchID, tableID, sessionID, sc.seat, sc.escrowID, sc.pub, sc.privHex)
			if err == nil {
				errCh <- nil
				return
			}
			if strings.Contains(err.Error(), "match seats not filled") {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			errCh <- err
			return
		}
		errCh <- fmt.Errorf("presign retries exhausted: %w", err)
	}
	for _, sc := range seats {
		go runPresign(sc)
	}
	for i := 0; i < len(seats); i++ {
		select {
		case err := <-errCh:
			require.NoError(t, err)
		case <-time.After(5 * time.Second):
			t.Fatalf("presign timed out (%d)", i)
		}
	}

	winnerSeat := int32(3)
	expectedBranch, err := env.PokerSrv.BranchIndexForSeat(matchID, winnerSeat)
	require.NoError(t, err)

	// Winner seat 3 fetches finalize bundle.
	bundle, err := seats[winnerSeat].ref.GetFinalizeBundle(ctx, matchID, winnerSeat)
	require.NoError(t, err)
	require.Equal(t, expectedBranch, bundle.Branch)
	assertFinalizeBundle(t, bundle, matchID, expectedBranch, payouts, amount, len(seats))
}

// assertFinalizeBundle verifies structural correctness of the finalize bundle.
func assertFinalizeBundle(t *testing.T, bundle *pokerrpc.GetFinalizeBundleResponse, matchID string, winnerSeat int32, payoutAddrs []string, perSeatAmt uint64, seats int) {
	t.Helper()

	require.Equal(t, matchID, bundle.MatchId)
	require.Equal(t, winnerSeat, bundle.Branch)
	require.NotEmpty(t, bundle.DraftTxHex)
	require.NotEmpty(t, bundle.GammaHex)
	require.Len(t, bundle.Inputs, seats)

	draftBytes, err := hex.DecodeString(bundle.DraftTxHex)
	require.NoError(t, err, "decode draft hex")
	var tx wire.MsgTx
	require.NoError(t, tx.Deserialize(bytes.NewReader(draftBytes)), "deserialize draft tx")
	require.Len(t, tx.TxIn, seats)
	require.Len(t, tx.TxOut, 1)

	scripts := make(map[string][]byte)
	for _, pa := range payoutAddrs {
		addr, err := stdaddr.DecodeAddress(pa, chaincfg.TestNet3Params())
		require.NoError(t, err)
		_, payScript := addr.PaymentScript()
		scripts[pa] = payScript
	}
	var matched bool
	for _, ps := range scripts {
		if bytes.Equal(ps, tx.TxOut[0].PkScript) {
			matched = true
			break
		}
	}
	require.True(t, matched, "tx output not paying any expected payout address")

	totalIn := perSeatAmt * uint64(seats)
	require.EqualValues(t, int64(totalIn-server.DefaultSettlementFeeAtoms), tx.TxOut[0].Value)

	inputByIdx := make(map[uint32]*pokerrpc.FinalizeInput, len(bundle.Inputs))
	for _, in := range bundle.Inputs {
		require.NotEmpty(t, in.InputId)
		require.NotEmpty(t, in.RPrimeCompactHex)
		require.NotEmpty(t, in.SPrimeHex)
		require.NotEmpty(t, in.RedeemScriptHex)
		inputByIdx[in.InputIndex] = in
	}
	require.Len(t, inputByIdx, seats)

	for i, txIn := range tx.TxIn {
		in, ok := inputByIdx[uint32(i)]
		require.True(t, ok, "missing input %d", i)
		require.Equal(t, txIn.PreviousOutPoint.String(), in.InputId)
	}

	require.EqualValues(t, perSeatAmt*uint64(seats), totalIn)
}

// TestGetFinalizeBundleForWinner tests that a winner can retrieve the finalize bundle
// with gamma after presign is complete for all branches.
// This verifies the settlement flow works correctly for different winner seats.
func TestGetFinalizeBundleForWinner(t *testing.T) {
	t.Parallel()
	env := testenv.New(t)
	defer env.Close()

	ctx := context.Background()

	buyIn := settlementTestBuyIn()

	// Create table (2 players).
	tableID := env.CreateTableWithBuyIn(ctx, "alice", 2, 2, int64(buyIn))

	// Seed auth sessions with tokens and payout addresses.
	seedSession := func(tok, uid, payout string) {
		var uidShort zkidentity.ShortID
		_ = uidShort.FromString(uid)
		env.PokerSrv.TestSeedSession(tok, uidShort, payout, uid)
	}
	alicePayout := "TsRnk22spGQJTpKFcRBc281rmfNFpywh337"
	bobPayout := "TsgsQwSZTkbXPGdFBg5z3wthjkQs1EeKcJ5"
	seedSession("alice-token", "alice", alicePayout)
	seedSession("bob-token", "bob", bobPayout)

	// Create PokerClients.
	logBackend := testenv.NewLogBackend()
	pcAlice, err := client.NewPokerClientWithDialOptions(ctx, &client.ClientConfig{
		Datadir:       t.TempDir(),
		LogBackend:    logBackend,
		Notifications: client.NewNotificationManager(),
	}, env.DialTarget(), env.DialOptions()...)
	require.NoError(t, err)
	pcBob, err := client.NewPokerClientWithDialOptions(ctx, &client.ClientConfig{
		Datadir:       t.TempDir(),
		LogBackend:    logBackend,
		Notifications: client.NewNotificationManager(),
	}, env.DialTarget(), env.DialOptions()...)
	require.NoError(t, err)

	pcAliceToken := "alice-token"
	pcBobToken := "bob-token"

	// Generate session keys for escrow.
	alicePriv, _ := secp256k1.GeneratePrivateKey()
	alicePub := alicePriv.PubKey().SerializeCompressed()
	bobPriv, _ := secp256k1.GeneratePrivateKey()
	bobPub := bobPriv.PubKey().SerializeCompressed()

	// Open escrows.
	refAlice := pcAlice.Referee(pcAliceToken)
	refBob := pcBob.Referee(pcBobToken)
	escrowA, err := refAlice.OpenEscrow(ctx, buyIn, 64, alicePub)
	require.NoError(t, err)
	escrowB, err := refBob.OpenEscrow(ctx, buyIn, 64, bobPub)
	require.NoError(t, err)
	require.NotEmpty(t, escrowA.EscrowId)
	require.NotEmpty(t, escrowB.EscrowId)

	// Manually mark escrows as funded/bound.
	env.PokerSrv.TestBindEscrowFunding(escrowA.EscrowId, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 0, buyIn)
	env.PokerSrv.TestBindEscrowFunding(escrowB.EscrowId, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 0, buyIn)

	// Bind and presign for a match.
	sessionID := "game-session"
	matchID := tableID + "|" + sessionID

	// Run presign concurrently for both players.
	errCh := make(chan error, 2)
	runPresign := func(ref *client.RefereeClient, seat uint32, escrowID string, pub []byte, privHex string) {
		const retries = 15
		var err error
		for i := 0; i < retries; i++ {
			err = ref.StartPresign(ctx, matchID, tableID, sessionID, seat, escrowID, pub, privHex)
			if err == nil {
				errCh <- nil
				return
			}
			if strings.Contains(err.Error(), "match seats not filled") {
				time.Sleep(20 * time.Millisecond)
				continue
			}
			errCh <- err
			return
		}
		errCh <- fmt.Errorf("presign retries exhausted: %w", err)
	}
	go runPresign(refAlice, 0, escrowA.EscrowId, alicePub, hex.EncodeToString(alicePriv.Serialize()))
	go runPresign(refBob, 1, escrowB.EscrowId, bobPub, hex.EncodeToString(bobPriv.Serialize()))

	for i := 0; i < 2; i++ {
		select {
		case err := <-errCh:
			require.NoError(t, err, "presign failed")
		case <-time.After(5 * time.Second):
			t.Fatal("presign timed out")
		}
	}
	t.Log("✓ Presign completed for both players")

	// Test: Alice wins (seat 0)
	t.Run("AliceWins", func(t *testing.T) {
		winnerSeat := int32(0)
		bundle, err := refAlice.GetFinalizeBundle(ctx, matchID, winnerSeat)
		require.NoError(t, err, "GetFinalizeBundle should succeed for winner seat 0")

		// Verify finalize bundle structure.
		require.Equal(t, matchID, bundle.MatchId)
		expectedBranch, err := env.PokerSrv.BranchIndexForSeat(matchID, winnerSeat)
		require.NoError(t, err, "BranchIndexForSeat should succeed")
		require.Equal(t, expectedBranch, bundle.Branch)
		require.NotEmpty(t, bundle.DraftTxHex, "DraftTxHex should be present")
		require.NotEmpty(t, bundle.GammaHex, "GammaHex (adaptor secret) should be present")
		require.Len(t, bundle.Inputs, 2, "Should have presigs for both inputs")

		// Verify gamma is 32 bytes hex (64 chars).
		gammaBytes, err := hex.DecodeString(bundle.GammaHex)
		require.NoError(t, err, "GammaHex should be valid hex")
		require.Len(t, gammaBytes, 32, "Gamma should be 32 bytes")

		// Verify each input has presig data.
		for i, in := range bundle.Inputs {
			require.NotEmpty(t, in.InputId, "Input %d should have InputId", i)
			require.NotEmpty(t, in.RPrimeCompactHex, "Input %d should have R'", i)
			require.NotEmpty(t, in.SPrimeHex, "Input %d should have s'", i)
			require.NotEmpty(t, in.RedeemScriptHex, "Input %d should have redeem script", i)
		}

		t.Logf("✓ Alice (seat 0) can retrieve finalize bundle with gamma: %s...", bundle.GammaHex[:16])
	})

	// Test: Bob wins (seat 1)
	t.Run("BobWins", func(t *testing.T) {
		winnerSeat := int32(1)
		bundle, err := refBob.GetFinalizeBundle(ctx, matchID, winnerSeat)
		require.NoError(t, err, "GetFinalizeBundle should succeed for winner seat 1")

		// Verify finalize bundle structure.
		require.Equal(t, matchID, bundle.MatchId)
		expectedBranch, err := env.PokerSrv.BranchIndexForSeat(matchID, winnerSeat)
		require.NoError(t, err, "BranchIndexForSeat should succeed")
		require.Equal(t, expectedBranch, bundle.Branch)
		require.NotEmpty(t, bundle.DraftTxHex, "DraftTxHex should be present")
		require.NotEmpty(t, bundle.GammaHex, "GammaHex (adaptor secret) should be present")
		require.Len(t, bundle.Inputs, 2, "Should have presigs for both inputs")

		// Verify gamma is 32 bytes hex (64 chars).
		gammaBytes, err := hex.DecodeString(bundle.GammaHex)
		require.NoError(t, err, "GammaHex should be valid hex")
		require.Len(t, gammaBytes, 32, "Gamma should be 32 bytes")

		// Verify each input has presig data.
		for i, in := range bundle.Inputs {
			require.NotEmpty(t, in.InputId, "Input %d should have InputId", i)
			require.NotEmpty(t, in.RPrimeCompactHex, "Input %d should have R'", i)
			require.NotEmpty(t, in.SPrimeHex, "Input %d should have s'", i)
			require.NotEmpty(t, in.RedeemScriptHex, "Input %d should have redeem script", i)
		}

		t.Logf("✓ Bob (seat 1) can retrieve finalize bundle with gamma: %s...", bundle.GammaHex[:16])
	})

	// Verify different gammas for different branches (important for security)
	t.Run("DifferentGammasPerBranch", func(t *testing.T) {
		bundleA, err := refAlice.GetFinalizeBundle(ctx, matchID, 0)
		require.NoError(t, err)
		bundleB, err := refBob.GetFinalizeBundle(ctx, matchID, 1)
		require.NoError(t, err)

		require.NotEqual(t, bundleA.GammaHex, bundleB.GammaHex,
			"Different branches should have different gamma values")
		require.NotEqual(t, bundleA.DraftTxHex, bundleB.DraftTxHex,
			"Different branches should have different draft transactions")

		t.Log("✓ Each branch has unique gamma and draft tx")
	})
}

// TestSettlementMatchIDFromTable verifies that the table correctly provides
// the matchID for settlement when a game ends.
//
// For WTA poker, the tableID itself is the matchID (a random 16-byte hex string).
// This simplifies the design - no sessionID tracking needed.
func TestSettlementMatchIDFromTable(t *testing.T) {
	t.Parallel()
	env := testenv.New(t)
	defer env.Close()

	ctx := context.Background()

	buyIn := settlementTestBuyIn()

	tableID := env.CreateTableWithBuyIn(ctx, "alice", 2, 2, int64(buyIn))

	// Verify tableID is now hex format (32 chars = 16 bytes)
	require.Len(t, tableID, 32, "tableID should be 32 hex chars (16 bytes)")
	t.Logf("Table created with hex ID: %s", tableID)

	seedSession := func(tok, uid, payout string) {
		var uidShort zkidentity.ShortID
		_ = uidShort.FromString(uid)
		env.PokerSrv.TestSeedSession(tok, uidShort, payout, uid)
	}
	seedSession("alice-token", "alice", "TsRnk22spGQJTpKFcRBc281rmfNFpywh337")
	seedSession("bob-token", "bob", "TsgsQwSZTkbXPGdFBg5z3wthjkQs1EeKcJ5")

	logBackend := testenv.NewLogBackend()
	pcAlice, err := client.NewPokerClientWithDialOptions(ctx, &client.ClientConfig{
		Datadir:       t.TempDir(),
		LogBackend:    logBackend,
		Notifications: client.NewNotificationManager(),
	}, env.DialTarget(), env.DialOptions()...)
	require.NoError(t, err)
	pcBob, err := client.NewPokerClientWithDialOptions(ctx, &client.ClientConfig{
		Datadir:       t.TempDir(),
		LogBackend:    logBackend,
		Notifications: client.NewNotificationManager(),
	}, env.DialTarget(), env.DialOptions()...)
	require.NoError(t, err)

	alicePriv, _ := secp256k1.GeneratePrivateKey()
	alicePub := alicePriv.PubKey().SerializeCompressed()
	bobPriv, _ := secp256k1.GeneratePrivateKey()
	bobPub := bobPriv.PubKey().SerializeCompressed()

	refAlice := pcAlice.Referee("alice-token")
	refBob := pcBob.Referee("bob-token")
	escrowA, err := refAlice.OpenEscrow(ctx, buyIn, 64, alicePub)
	require.NoError(t, err)
	escrowB, err := refBob.OpenEscrow(ctx, buyIn, 64, bobPub)
	require.NoError(t, err)

	env.PokerSrv.TestBindEscrowFunding(escrowA.EscrowId, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 0, buyIn)
	env.PokerSrv.TestBindEscrowFunding(escrowB.EscrowId, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 0, buyIn)

	// For WTA poker, matchID = tableID (no sessionID suffix needed)
	matchID := tableID

	errCh := make(chan error, 2)
	runPresign := func(ref *client.RefereeClient, seat uint32, escrowID string, pub []byte, privHex string) {
		const retries = 15
		var err error
		for i := 0; i < retries; i++ {
			// Use tableID as both matchID and tableID; sessionID can be empty
			err = ref.StartPresign(ctx, matchID, tableID, "", seat, escrowID, pub, privHex)
			if err == nil {
				errCh <- nil
				return
			}
			if strings.Contains(err.Error(), "match seats not filled") {
				time.Sleep(20 * time.Millisecond)
				continue
			}
			errCh <- err
			return
		}
		errCh <- fmt.Errorf("presign retries exhausted: %w", err)
	}
	go runPresign(refAlice, 0, escrowA.EscrowId, alicePub, hex.EncodeToString(alicePriv.Serialize()))
	go runPresign(refBob, 1, escrowB.EscrowId, bobPub, hex.EncodeToString(bobPriv.Serialize()))

	for i := 0; i < 2; i++ {
		select {
		case err := <-errCh:
			require.NoError(t, err, "presign failed")
		case <-time.After(5 * time.Second):
			t.Fatal("presign timed out")
		}
	}
	t.Log("✓ Presign completed for both players")

	// The table's GetSettlementMatchID() should return just tableID
	table, ok := env.PokerSrv.GetTable(tableID)
	require.True(t, ok, "Table should exist")

	tableMatchID := table.GetSettlementMatchID()
	require.Equal(t, tableID, tableMatchID,
		"Table's GetSettlementMatchID() should return the tableID")

	// Verify this matchID works for GetFinalizeBundle
	bundle, err := refAlice.GetFinalizeBundle(ctx, tableMatchID, 0)
	require.NoError(t, err, "GetFinalizeBundle should work with table's matchID")
	require.NotEmpty(t, bundle.GammaHex)

	t.Log("✓ Table correctly provides matchID for settlement")
}

// TestGameDoesNotStartWithoutPresign verifies that an escrow-backed table
// will not start the game until all players have completed presigning.
func TestGameDoesNotStartWithoutPresign(t *testing.T) {
	t.Parallel()
	env := testenv.New(t)
	defer env.Close()

	ctx := context.Background()

	// Prepare two players with balances.
	// We need to create ShortIDs from the player names so that uid.String() matches the table's player ID.
	// Since ShortID.String() returns hex, we need to use the hex representation as the player ID.
	// Create ShortIDs from "alice" and "bob" by hashing them to get valid ShortID bytes.
	aliceBytes := chainhash.HashB([]byte("alice"))
	bobBytes := chainhash.HashB([]byte("bob"))
	var aliceUID, bobUID zkidentity.ShortID
	aliceUID.FromBytes(aliceBytes[:])
	bobUID.FromBytes(bobBytes[:])
	alicePlayerID := aliceUID.String()
	bobPlayerID := bobUID.String()

	buyIn := settlementTestBuyIn()

	// Seed auth sessions with tokens and payout addresses.
	env.PokerSrv.TestSeedSession("alice-token", aliceUID, "TsRnk22spGQJTpKFcRBc281rmfNFpywh337", "alice")
	env.PokerSrv.TestSeedSession("bob-token", bobUID, "TsgsQwSZTkbXPGdFBg5z3wthjkQs1EeKcJ5", "bob")

	// Create table with buy-in (escrow required).
	// Use the ShortID string representation as the player ID to match BindEscrow's lookup.
	tableID := env.CreateTableWithBuyIn(ctx, alicePlayerID, 2, 2, int64(buyIn))

	// Create PokerClients.
	logBackend := testenv.NewLogBackend()
	pcAlice, err := client.NewPokerClientWithDialOptions(ctx, &client.ClientConfig{
		Datadir:       t.TempDir(),
		LogBackend:    logBackend,
		Notifications: client.NewNotificationManager(),
	}, env.DialTarget(), env.DialOptions()...)
	require.NoError(t, err)
	pcBob, err := client.NewPokerClientWithDialOptions(ctx, &client.ClientConfig{
		Datadir:       t.TempDir(),
		LogBackend:    logBackend,
		Notifications: client.NewNotificationManager(),
	}, env.DialTarget(), env.DialOptions()...)
	require.NoError(t, err)

	pcAliceToken := "alice-token"
	pcBobToken := "bob-token"

	// Generate session keys for escrow.
	alicePriv, _ := secp256k1.GeneratePrivateKey()
	alicePub := alicePriv.PubKey().SerializeCompressed()
	bobPriv, _ := secp256k1.GeneratePrivateKey()
	bobPub := bobPriv.PubKey().SerializeCompressed()

	// Open escrows.
	refAlice := pcAlice.Referee(pcAliceToken)
	refBob := pcBob.Referee(pcBobToken)
	escrowA, err := refAlice.OpenEscrow(ctx, buyIn, 64, alicePub)
	require.NoError(t, err)
	escrowB, err := refBob.OpenEscrow(ctx, buyIn, 64, bobPub)
	require.NoError(t, err)

	// Mark escrows as funded.
	txidA := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	txidB := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	env.PokerSrv.TestBindEscrowFunding(escrowA.EscrowId, txidA, 0, buyIn)
	env.PokerSrv.TestBindEscrowFunding(escrowB.EscrowId, txidB, 0, buyIn)

	_, err = env.LobbyClient.JoinTable(ctx, &pokerrpc.JoinTableRequest{
		PlayerId: bobPlayerID,
		TableId:  tableID,
	})
	require.NoError(t, err)

	// Bind escrows to the table/match using proper RPC calls (not test helpers).
	// For poker tables, matchID = tableID (no sessionID suffix needed).
	matchID := tableID
	outpointA := fmt.Sprintf("%s:0", txidA)
	outpointB := fmt.Sprintf("%s:0", txidB)

	// Bind Alice's escrow (seat will be auto-detected from her position at table).
	bindRespA, err := refAlice.BindEscrow(ctx, tableID, "", matchID, 0, outpointA, escrowA.RedeemScriptHex, 64)
	require.NoError(t, err, "BindEscrow for alice failed")
	require.Equal(t, escrowA.EscrowId, bindRespA.EscrowId)
	require.True(t, bindRespA.EscrowReady, "Alice's escrow should be ready after binding")

	// Bind Bob's escrow (seat will be auto-detected from his position at table).
	bindRespB, err := refBob.BindEscrow(ctx, tableID, "", matchID, 0, outpointB, escrowB.RedeemScriptHex, 64)
	require.NoError(t, err, "BindEscrow for bob failed")
	require.Equal(t, escrowB.EscrowId, bindRespB.EscrowId)
	require.True(t, bindRespB.EscrowReady, "Bob's escrow should be ready after binding")

	// Both players set ready.
	readyResp, err := env.LobbyClient.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{
		PlayerId: alicePlayerID,
		TableId:  tableID,
	})
	require.NoError(t, err)
	require.True(t, readyResp.Success)

	readyResp, err = env.LobbyClient.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{
		PlayerId: bobPlayerID,
		TableId:  tableID,
	})
	require.NoError(t, err)
	require.True(t, readyResp.Success)
	// The response should indicate all players are ready, but waiting for presigning.
	require.True(t, readyResp.AllPlayersReady)
	require.Contains(t, readyResp.Message, "Waiting for presigning")

	// Verify the game has NOT started (presigning incomplete).
	time.Sleep(100 * time.Millisecond) // Give any async start a chance
	gameState, err := env.PokerClient.GetGameState(ctx, &pokerrpc.GetGameStateRequest{
		TableId: tableID,
	})
	require.NoError(t, err)
	require.False(t, gameState.GameState.GameStarted, "Game should NOT start without presigning")
	t.Log("✓ Game correctly blocked from starting without presigning")

	// Now complete presigning for both players.
	errCh := make(chan error, 2)
	runPresign := func(ref *client.RefereeClient, seat uint32, escrowID string, pub []byte, privHex string) {
		const retries = 10
		var err error
		for i := 0; i < retries; i++ {
			err = ref.StartPresign(ctx, matchID, tableID, "sess1", seat, escrowID, pub, privHex)
			if err == nil {
				errCh <- nil
				return
			}
			if strings.Contains(err.Error(), "match seats not filled") {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			errCh <- err
			return
		}
		errCh <- fmt.Errorf("presign retries exhausted: %w", err)
	}
	go runPresign(refAlice, 0, escrowA.EscrowId, alicePub, hex.EncodeToString(alicePriv.Serialize()))
	go runPresign(refBob, 1, escrowB.EscrowId, bobPub, hex.EncodeToString(bobPriv.Serialize()))

	for i := 0; i < 2; i++ {
		select {
		case err := <-errCh:
			require.NoError(t, err, "presign failed")
		case <-time.After(5 * time.Second):
			t.Fatal("presign timed out")
		}
	}
	t.Log("✓ Presigning completed for both players")

	// Wait for server to mark presigning as complete.
	require.Eventually(t, func() bool {
		complete, _, _ := env.PokerSrv.IsPresigningComplete(matchID)
		return complete
	}, 2*time.Second, 10*time.Millisecond, "Server should mark presigning as complete")

	// Now trigger the ready check again (simulate re-setting ready or a background check).
	// In practice, the server should auto-start when presigning completes and all are ready.
	// For this test, we set one player ready again to trigger the check.
	readyResp, err = env.LobbyClient.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{
		PlayerId: alicePlayerID,
		TableId:  tableID,
	})
	require.NoError(t, err)

	// Wait for game to start.
	var gameStarted bool
	for i := 0; i < 20; i++ {
		time.Sleep(50 * time.Millisecond)
		gameState, err = env.PokerClient.GetGameState(ctx, &pokerrpc.GetGameStateRequest{
			TableId: tableID,
		})
		require.NoError(t, err)
		if gameState.GameState.GameStarted {
			gameStarted = true
			break
		}
	}
	require.True(t, gameStarted, "Game should start after presigning is complete")
	t.Log("✓ Game started after presigning completed")
}
