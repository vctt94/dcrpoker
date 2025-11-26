package server

import (
	"context"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/companyzero/bisonrelay/zkidentity"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/stretchr/testify/require"
	"github.com/vctt94/pokerbisonrelay/pkg/chainwatcher"
	"github.com/vctt94/pokerbisonrelay/pkg/poker"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	"github.com/vctt94/pokerbisonrelay/pkg/server/internal/db"
)

// mockDB satisfies Database for tests that don't exercise persistence paths.
type mockDB struct{}

func (m *mockDB) Close() error { return nil }
func (m *mockDB) GetPlayerBalance(context.Context, string) (int64, error) {
	return 0, nil
}
func (m *mockDB) UpdatePlayerBalance(context.Context, string, int64, string, string) error {
	return nil
}
func (m *mockDB) UpsertSnapshot(context.Context, db.Snapshot) error { return nil }
func (m *mockDB) GetSnapshot(context.Context, string) (*db.Snapshot, error) {
	return nil, fmt.Errorf("not found")
}
func (m *mockDB) UpsertTable(context.Context, *poker.TableConfig) error { return nil }
func (m *mockDB) GetTable(context.Context, string) (*db.Table, error) {
	return nil, fmt.Errorf("not found")
}
func (m *mockDB) DeleteTable(context.Context, string) error      { return nil }
func (m *mockDB) ListTableIDs(context.Context) ([]string, error) { return nil, nil }
func (m *mockDB) ActiveParticipants(context.Context, string) ([]db.Participant, error) {
	return nil, nil
}
func (m *mockDB) SeatPlayer(context.Context, string, string, int) error { return nil }
func (m *mockDB) UnseatPlayer(context.Context, string, string) error    { return nil }
func (m *mockDB) UpsertAuthUser(context.Context, string, string) error  { return nil }
func (m *mockDB) GetAuthUserByNickname(context.Context, string) (*db.AuthUser, error) {
	return nil, fmt.Errorf("not found")
}
func (m *mockDB) GetAuthUserByUserID(context.Context, string) (*db.AuthUser, error) {
	return nil, fmt.Errorf("not found")
}
func (m *mockDB) UpdateAuthUserLastLogin(context.Context, string) error { return nil }
func (m *mockDB) ListAllAuthUsers(context.Context) ([]db.AuthUser, error) {
	return nil, nil
}

// seedEscrow seeds an escrow session with funding + presign metadata for tests.
func seedEscrow(t *testing.T, s *Server, uidStr, token, txid string, seat uint32, amount uint64) (*refereePreSignCtx, string) {
	t.Helper()

	var uid zkidentity.ShortID
	require.NoError(t, uid.FromString(uidStr))
	// register payout + token
	s.TestSeedSession(token, uid, "TsRnk22spGQJTpKFcRBc281rmfNFpywh337", uidStr)

	priv, _ := secp256k1.GeneratePrivateKey()
	compPub := priv.PubKey().SerializeCompressed()
	openResp, err := s.OpenEscrow(context.Background(), &pokerrpc.OpenEscrowRequest{
		AmountAtoms: amount,
		CsvBlocks:   64,
		Token:       token,
		CompPubkey:  compPub,
	})
	require.NoError(t, err)
	require.NotEmpty(t, openResp.EscrowId)

	s.TestBindEscrowFunding(openResp.EscrowId, txid, 0, amount)

	psCtx := &refereePreSignCtx{
		InputID:            txid + ":0",
		AmountAtoms:        amount,
		RedeemScriptHex:    openResp.RedeemScriptHex,
		DraftHex:           "deadbeef",
		SighashHex:         "0102030405060708090a0b0c0d0e0f00102030405060708090a0b0c0d0e0f00",
		TCompressed:        []byte{0x02, 0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39, 0x40, 0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47, 0x48, 0x49, 0x4a, 0x4b},
		RPrimeCompressed:   []byte{0x02, 0x50, 0x51, 0x52, 0x53, 0x54, 0x55, 0x56, 0x57, 0x58, 0x59, 0x60, 0x61, 0x62, 0x63, 0x64, 0x65, 0x66, 0x67, 0x68, 0x69, 0x70, 0x71, 0x72, 0x73, 0x74, 0x75, 0x76, 0x77, 0x78, 0x79, 0x7a, 0x7b},
		SPrime32:           bytesFromHex(t, "6c6f63616c2d73702d7363616c61722d746573742d30313233"),
		InputIndex:         seat,
		Branch:             0,
		SeatIndex:          seat,
		OwnerUID:           uid,
		WinnerCandidateUID: uid,
	}
	return psCtx, openResp.EscrowId
}

func bytesFromHex(t *testing.T, h string) []byte {
	t.Helper()
	b, err := hex.DecodeString(h)
	require.NoError(t, err)
	return b
}

// newTestServerWithState creates a referee-ready server with preloaded maps.
func newTestServerWithState(t *testing.T) *Server {
	logBackend := createTestLogBackend()
	srv := &Server{
		log:           logBackend.Logger("SERVER"),
		logBackend:    logBackend,
		db:            &mockDB{},
		chainParams:   selectChainParams("testnet"),
		adaptorSecret: "",
	}
	srv.auth = newAuthState(srv.db)
	srv.referee = newSchnorrRefereeState(ServerConfig{})
	srv.referee.presigns = make(map[string]map[int32]map[string]*refereePreSignCtx)
	srv.referee.branchGamma = make(map[string]map[int32]string)
	srv.referee.matchEscrows = make(map[string]map[uint32]string)
	srv.referee.escrows = make(map[string]*refereeEscrowSession)
	return srv
}

// TestGetFinalizeBundleSuccess ensures a winning finalize bundle is returned when presigs are present.
func TestGetFinalizeBundleSuccess(t *testing.T) {
	const matchID = "table1|sess1"
	const amount = uint64(1_000_000)
	srv := newTestServerWithState(t)

	ctx1, esc1 := seedEscrow(t, srv, "0000000000000000000000000000000000000000000000000000000000000001", "tok1", "aaaa", 0, amount)
	ctx2, esc2 := seedEscrow(t, srv, "0000000000000000000000000000000000000000000000000000000000000002", "tok2", "bbbb", 1, amount)

	// bind match escrows
	srv.referee.matchEscrows[matchID] = map[uint32]string{0: esc1, 1: esc2}
	srv.referee.escrows[esc1].SeatIndex = 0
	srv.referee.escrows[esc2].SeatIndex = 1

	// seed presigs + gamma for branch 0
	srv.referee.presigns[matchID] = map[int32]map[string]*refereePreSignCtx{
		0: {ctx1.InputID: ctx1, ctx2.InputID: ctx2},
	}
	srv.referee.branchGamma[matchID] = map[int32]string{0: "cafebabe"}

	resp, err := srv.GetFinalizeBundle(context.Background(), &pokerrpc.GetFinalizeBundleRequest{
		MatchId:    matchID,
		WinnerSeat: 0,
	})
	require.NoError(t, err)
	require.Equal(t, matchID, resp.MatchId)
	require.Equal(t, int32(0), resp.Branch)
	require.Equal(t, "cafebabe", resp.GammaHex)
	require.Equal(t, ctx1.DraftHex, resp.DraftTxHex)
	require.Len(t, resp.Inputs, 2)
}

// TestGetFinalizeBundleSuccessThreePlayers covers a 3-player match with winner seat 2.
func TestGetFinalizeBundleSuccessThreePlayers(t *testing.T) {
	const matchID = "table1|sess3"
	const amount = uint64(1_000_000)
	srv := newTestServerWithState(t)

	ctx1, esc1 := seedEscrow(t, srv, "0000000000000000000000000000000000000000000000000000000000000001", "tok1", "aaaa", 0, amount)
	ctx2, esc2 := seedEscrow(t, srv, "0000000000000000000000000000000000000000000000000000000000000002", "tok2", "bbbb", 1, amount)
	ctx3, esc3 := seedEscrow(t, srv, "0000000000000000000000000000000000000000000000000000000000000003", "tok3", "cccc", 2, amount)

	srv.referee.matchEscrows[matchID] = map[uint32]string{0: esc1, 1: esc2, 2: esc3}

	srv.referee.presigns[matchID] = map[int32]map[string]*refereePreSignCtx{
		2: {ctx1.InputID: ctx1, ctx2.InputID: ctx2, ctx3.InputID: ctx3},
	}
	srv.referee.branchGamma[matchID] = map[int32]string{2: "cafebabe"}

	resp, err := srv.GetFinalizeBundle(context.Background(), &pokerrpc.GetFinalizeBundleRequest{
		MatchId:    matchID,
		WinnerSeat: 2,
	})
	require.NoError(t, err)
	require.Len(t, resp.Inputs, 3)
	require.Equal(t, int32(2), resp.Branch)
}

// TestGetFinalizeBundleErrors covers common failure cases.
func TestGetFinalizeBundleErrors(t *testing.T) {
	const matchID = "table1|sess1"
	srv := newTestServerWithState(t)

	// Missing match
	_, err := srv.GetFinalizeBundle(context.Background(), &pokerrpc.GetFinalizeBundleRequest{MatchId: matchID, WinnerSeat: 0})
	require.Error(t, err)

	// Seed escrows but no presigs/gamma
	srv.referee.matchEscrows[matchID] = map[uint32]string{0: "esc1", 1: "esc2"}
	srv.referee.escrows["esc1"] = &refereeEscrowSession{EscrowID: "esc1", BoundUTXO: &chainwatcher.EscrowUTXO{Value: 1, Txid: "a", Vout: 0}, CompPubkey: []byte{1}}
	srv.referee.escrows["esc2"] = &refereeEscrowSession{EscrowID: "esc2", BoundUTXO: &chainwatcher.EscrowUTXO{Value: 1, Txid: "b", Vout: 0}, CompPubkey: []byte{1}}

	_, err = srv.GetFinalizeBundle(context.Background(), &pokerrpc.GetFinalizeBundleRequest{MatchId: matchID, WinnerSeat: 0})
	require.Error(t, err)

	// Presigs without gamma
	var uid zkidentity.ShortID
	_ = uid.FromString("0000000000000000000000000000000000000000000000000000000000000001")
	srv.referee.presigns[matchID] = map[int32]map[string]*refereePreSignCtx{0: {"a:0": {InputID: "a:0", DraftHex: "aa", RedeemScriptHex: "bb", TCompressed: []byte{1}, RPrimeCompressed: []byte{2}, SPrime32: []byte{3}, InputIndex: 0, AmountAtoms: 1, OwnerUID: uid, WinnerCandidateUID: uid}}}
	_, err = srv.GetFinalizeBundle(context.Background(), &pokerrpc.GetFinalizeBundleRequest{MatchId: matchID, WinnerSeat: 0})
	require.Error(t, err)
}

// mkEscrow constructs a refereeEscrowSession with the given bound UTXO + payout.
func mkEscrow(t *testing.T, uidStr, txid string, vout uint32, pkScriptHex, redeemHex, payout string, seat uint32) *refereeEscrowSession {
	t.Helper()
	var uid zkidentity.ShortID
	require.NoError(t, uid.FromString(uidStr))
	return &refereeEscrowSession{
		EscrowID:        txid,
		OwnerUID:        uid,
		SeatIndex:       seat,
		PayoutAddr:      payout,
		PkScriptHex:     pkScriptHex,
		RedeemScriptHex: redeemHex,
		BoundUTXO:       &chainwatcher.EscrowUTXO{Txid: txid, Vout: vout, Value: 1_000_000, PkScriptHex: pkScriptHex},
		CompPubkey:      []byte{1}, // non-empty
	}
}

func TestBuildWTADraftsDeterministicAndPayout(t *testing.T) {
	s := &Server{
		referee:     newSchnorrRefereeState(ServerConfig{AdaptorSecret: "deadbeef"}),
		chainParams: selectChainParams("testnet"),
	}

	// Two escrows with differing pkScripts and txids; payout goes to seat 1.
	e1 := mkEscrow(t, "0000000000000000000000000000000000000000000000000000000000000001", "aaaa", 0, "51", "51", "TsRnk22spGQJTpKFcRBc281rmfNFpywh337", 0)
	e2 := mkEscrow(t, "0000000000000000000000000000000000000000000000000000000000000002", "bbbb", 1, "52", "52", "TsgsQwSZTkbXPGdFBg5z3wthjkQs1EeKcJ5", 1)

	drafts, err := s.buildWTADrafts("m1", []*refereeEscrowSession{e2, e1}) // reverse order on purpose
	require.NoError(t, err)
	require.Len(t, drafts, 2)

	// Inputs should be sorted deterministically: by pkScript then txid/vout.
	require.Equal(t, "51", drafts[0].Inputs[0].RedeemScriptHex)
	require.Equal(t, "52", drafts[0].Inputs[1].RedeemScriptHex)
	require.Equal(t, uint32(0), drafts[0].Inputs[0].InputIndex)
	require.Equal(t, uint32(1), drafts[0].Inputs[1].InputIndex)

	// Winner seat 1 draft should pay seat 1's payout address.
	winner := drafts[1]
	require.Equal(t, e2.OwnerUID, winner.WinnerUID)
	raw, err := hex.DecodeString(winner.DraftHex)
	require.NoError(t, err)
	tx, err := decodeMsgTx(raw)
	require.NoError(t, err)
	require.Len(t, tx.TxOut, 1)
	// Sum(inputs)=2_000_000, fee=DefaultSettlementFeeAtoms -> output value must match.
	require.Equal(t, int64(2_000_000-DefaultSettlementFeeAtoms), tx.TxOut[0].Value)
}
