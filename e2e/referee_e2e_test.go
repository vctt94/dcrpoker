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

	// Prepare two players with balances.
	players := []string{"alice", "bob"}
	buyIn := settlementTestBuyIn()
	for _, p := range players {
		env.SetBalance(ctx, p, int64(buyIn)*2)
	}

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

	// Winner (alice) fetches finalize bundle for seat 0.
	bundle, err := refAlice.GetFinalizeBundle(ctx, matchID, 0)
	require.NoError(t, err)
	assertFinalizeBundle(t, bundle, matchID, 0, []string{"TsRnk22spGQJTpKFcRBc281rmfNFpywh337", "TsgsQwSZTkbXPGdFBg5z3wthjkQs1EeKcJ5"}, amount, 2)
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
	for _, p := range players {
		env.SetBalance(ctx, p, int64(buyIn)*2)
	}
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

	// Winner seat 3 fetches finalize bundle.
	winnerSeat := int32(3)
	bundle, err := seats[winnerSeat].ref.GetFinalizeBundle(ctx, matchID, winnerSeat)
	require.NoError(t, err)
	assertFinalizeBundle(t, bundle, matchID, winnerSeat, payouts, amount, len(seats))
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
