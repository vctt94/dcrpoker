package server

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/companyzero/bisonrelay/zkidentity"
	"github.com/decred/dcrd/chaincfg/chainhash"
	"github.com/decred/dcrd/chaincfg/v3"
	"github.com/decred/dcrd/crypto/blake256"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"github.com/decred/dcrd/dcrutil/v4"
	"github.com/decred/dcrd/txscript/v4"
	"github.com/decred/dcrd/txscript/v4/stdaddr"
	"github.com/decred/dcrd/wire"
	"github.com/vctt94/pokerbisonrelay/pkg/chainwatcher"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const refereeNotReadyMsg = "schnorr referee not yet wired"

// DefaultSettlementFeeAtoms is the fixed fee applied to Schnorr SNG/WTA
// settlement drafts. Keep in sync with docs DEFAULT_FEE.
const DefaultSettlementFeeAtoms uint64 = 10_000

// MaxEscrowAtoms caps per-player escrow to avoid oversized pots.
const MaxEscrowAtoms uint64 = 100_000_000 // 1 DCR

// schnorrMatchKey keys escrow/presign state by table + session.
type schnorrMatchKey struct {
	TableID   string
	SessionID string
}

type presignInput struct {
	InputID         string
	OwnerUID        zkidentity.ShortID
	SeatIndex       uint32
	AmountAtoms     uint64
	RedeemScriptHex string
	SighashHex      string
	AdaptorPointHex string
	InputIndex      uint32
}

type branchDraft struct {
	DraftHex  string
	Inputs    []presignInput
	GammaHex  string
	WinnerUID zkidentity.ShortID
}

func (k schnorrMatchKey) String() string {
	return fmt.Sprintf("%s|%s", k.TableID, k.SessionID)
}

// refereePreSignCtx stores presign artifacts for a specific input/branch.
type refereePreSignCtx struct {
	InputID            string
	AmountAtoms        uint64
	RedeemScriptHex    string
	DraftHex           string
	SighashHex         string
	TCompressed        []byte
	RPrimeCompressed   []byte
	SPrime32           []byte
	InputIndex         uint32
	Branch             int32
	SeatIndex          uint32
	OwnerUID           zkidentity.ShortID
	WinnerCandidateUID zkidentity.ShortID
}

// refereeEscrowSession tracks one player's escrow for a match.
type refereeEscrowSession struct {
	EscrowID   string
	OwnerUID   zkidentity.ShortID
	Token      string
	TableID    string
	SessionID  string
	SeatIndex  uint32
	CompPubkey []byte
	// Derived from authenticated session wallet (P2PKH).
	PayoutAddr      string
	AmountAtoms     uint64
	CSVBlocks       uint32
	RedeemScriptHex string
	PkScriptHex     string

	mu sync.RWMutex

	WatcherUnsub  func()
	LatestFunding chainwatcher.DepositUpdate

	// Bound input once ensureBoundFunding succeeds.
	BoundUTXO *chainwatcher.EscrowUTXO
}

// cloneUTXO returns a shallow copy to avoid races on shared pointers.
func cloneUTXO(u *chainwatcher.EscrowUTXO) *chainwatcher.EscrowUTXO {
	if u == nil {
		return nil
	}
	clone := *u
	return &clone
}

type escrowSnapshot struct {
	EscrowID        string
	OwnerUID        zkidentity.ShortID
	SeatIndex       uint32
	PayoutAddr      string
	AmountAtoms     uint64
	RedeemScriptHex string
	PkScriptHex     string
	BoundUTXO       *chainwatcher.EscrowUTXO
}

func (es *refereeEscrowSession) snapshot() escrowSnapshot {
	es.mu.RLock()
	defer es.mu.RUnlock()
	return escrowSnapshot{
		EscrowID:        es.EscrowID,
		OwnerUID:        es.OwnerUID,
		SeatIndex:       es.SeatIndex,
		PayoutAddr:      es.PayoutAddr,
		AmountAtoms:     es.AmountAtoms,
		RedeemScriptHex: es.RedeemScriptHex,
		PkScriptHex:     es.PkScriptHex,
		BoundUTXO:       cloneUTXO(es.BoundUTXO),
	}
}

// schnorrRefereeState is the in-memory view of Schnorr SNG escrow/presign state.
type schnorrRefereeState struct {
	network       string
	adaptorSecret string
	feeAtoms      uint64

	mu              sync.RWMutex
	escrows         map[string]*refereeEscrowSession                   // escrowID -> session
	matchEscrows    map[string]map[uint32]string                       // matchKey -> seat -> escrowID
	presigns        map[string]map[int32]map[string]*refereePreSignCtx // matchKey -> branch -> inputID -> ctx
	branchGamma     map[string]map[int32]string                        // matchKey -> branch -> gammaHex
	presignComplete map[string]map[uint32]bool                         // matchKey -> seat -> completed
}

func newSchnorrRefereeState(cfg ServerConfig) *schnorrRefereeState {
	return &schnorrRefereeState{
		network:         cfg.Network,
		adaptorSecret:   cfg.AdaptorSecret,
		feeAtoms:        DefaultSettlementFeeAtoms,
		escrows:         make(map[string]*refereeEscrowSession),
		matchEscrows:    make(map[string]map[uint32]string),
		presigns:        make(map[string]map[int32]map[string]*refereePreSignCtx),
		branchGamma:     make(map[string]map[int32]string),
		presignComplete: make(map[string]map[uint32]bool),
	}
}

// buildWTADrafts builds one draft per possible winner (seat) for N escrows (2-6).
func (s *Server) buildWTADrafts(matchID string, escrows []*refereeEscrowSession) ([]branchDraft, error) {
	n := len(escrows)
	if n < 2 || n > 6 {
		return nil, fmt.Errorf("expected 2-6 escrows, got %d", n)
	}

	var snaps []escrowSnapshot
	for _, es := range escrows {
		snap := es.snapshot()
		if snap.BoundUTXO == nil {
			return nil, fmt.Errorf("escrow %s missing bound utxo", snap.EscrowID)
		}
		snaps = append(snaps, snap)
	}

	// Sort inputs deterministically (pkScript+txid:vout) to have stable indexes.
	sort.Slice(snaps, func(i, j int) bool {
		a := snaps[i].PkScriptHex
		b := snaps[j].PkScriptHex
		if a == b {
			if snaps[i].BoundUTXO.Txid == snaps[j].BoundUTXO.Txid {
				return snaps[i].BoundUTXO.Vout < snaps[j].BoundUTXO.Vout
			}
			return snaps[i].BoundUTXO.Txid < snaps[j].BoundUTXO.Txid
		}
		return a < b
	})

	// Build per-branch drafts.
	var drafts []branchDraft
	for winner := 0; winner < n; winner++ {
		gammaHex, TCompHex := deriveAdaptorForBranch(s.referee.adaptorSecret, matchID, int32(winner))
		tx := wire.NewMsgTx()
		// Schnorr via OP_CHECKSIGALTVERIFY requires tx version >= 3 (consensus).
		tx.Version = 3
		var total uint64
		for _, es := range snaps {
			utxo := es.BoundUTXO
			hash, err := chainhashFromStr(utxo.Txid)
			if err != nil {
				return nil, err
			}
			out := wire.NewOutPoint(hash, utxo.Vout, wire.TxTreeRegular)
			amt := dcrutil.Amount(utxo.Value)
			txIn := wire.NewTxIn(out, int64(amt), nil)
			tx.AddTxIn(txIn)
			total += utxo.Value
		}
		if total <= s.referee.feeAtoms {
			return nil, fmt.Errorf("total inputs %d <= fee %d", total, s.referee.feeAtoms)
		}

		// Single output to winner payout addr.
		winnerAddr := snaps[winner].PayoutAddr
		addr, err := stdaddr.DecodeAddress(winnerAddr, s.chainParams)
		if err != nil {
			return nil, fmt.Errorf("decode payout addr: %v", err)
		}
		_, payScript := addr.PaymentScript()
		out := wire.NewTxOut(int64(total-s.referee.feeAtoms), payScript)
		tx.AddTxOut(out)

		// Compute sighashes per input.
		var inputs []presignInput
		for idx, es := range snaps {
			redeem, err := hex.DecodeString(es.RedeemScriptHex)
			if err != nil {
				return nil, fmt.Errorf("redeem decode: %v", err)
			}
			sighash, err := txscript.CalcSignatureHash(redeem, txscript.SigHashAll, tx, idx, nil)
			if err != nil {
				return nil, fmt.Errorf("sighash: %v", err)
			}
			inputs = append(inputs, presignInput{
				InputID:         fmt.Sprintf("%s:%d", es.BoundUTXO.Txid, es.BoundUTXO.Vout),
				OwnerUID:        es.OwnerUID,
				SeatIndex:       es.SeatIndex,
				AmountAtoms:     es.BoundUTXO.Value,
				RedeemScriptHex: es.RedeemScriptHex,
				SighashHex:      hex.EncodeToString(sighash),
				AdaptorPointHex: TCompHex,
				InputIndex:      uint32(idx),
			})
		}

		var buf bytes.Buffer
		if err := tx.Serialize(&buf); err != nil {
			return nil, fmt.Errorf("serialize tx: %v", err)
		}
		drafts = append(drafts, branchDraft{
			DraftHex:  hex.EncodeToString(buf.Bytes()),
			Inputs:    inputs,
			GammaHex:  gammaHex,
			WinnerUID: snaps[winner].OwnerUID,
		})
	}

	return drafts, nil
}

func chainhashFromStr(s string) (*chainhash.Hash, error) {
	var h chainhash.Hash
	if err := chainhash.Decode(&h, s); err != nil {
		return nil, err
	}
	return &h, nil
}

// deriveAdaptorForBranch derives a branch adaptor point (gamma not returned to clients).
func deriveAdaptorForBranch(secret string, matchID string, branch int32) (gammaHex string, TCompHex string) {
	// For now, reuse helper from pong derivation style.
	return deriveAdaptorGamma(matchID, branch, secret)
}

// OpenEscrow creates an escrow session for a Schnorr SNG table.
func (s *Server) OpenEscrow(ctx context.Context, req *pokerrpc.OpenEscrowRequest) (*pokerrpc.OpenEscrowResponse, error) {
	token := strings.TrimSpace(req.GetToken())
	if token == "" {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if vals := md.Get("token"); len(vals) > 0 {
				token = strings.TrimSpace(vals[0])
			}
		}
	}
	if req == nil || token == "" {
		s.log.Debugf("OpenEscrow denied: empty token")
		return nil, status.Error(codes.Unauthenticated, "token required")
	}
	ownerUID, payoutAddr, ok := s.payoutForToken(token)
	if !ok || payoutAddr == "" {
		s.log.Debugf("OpenEscrow denied: invalid token len=%d sessions=%d", len(token), s.authSessionCount())
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "invalid or expired session")
		}
		return nil, status.Error(codes.FailedPrecondition, "payout address not set; please sign address first")
	}

	if req.AmountAtoms == 0 {
		return nil, status.Error(codes.InvalidArgument, "amount_atoms required")
	}
	if req.AmountAtoms > MaxEscrowAtoms {
		return nil, status.Errorf(codes.InvalidArgument, "amount_atoms exceeds max %d", MaxEscrowAtoms)
	}
	if req.CsvBlocks == 0 {
		return nil, status.Error(codes.InvalidArgument, "csv_blocks required")
	}
	if len(req.CompPubkey) != 33 {
		return nil, status.Error(codes.InvalidArgument, "comp_pubkey must be 33 bytes")
	}
	if s.chainParams == nil {
		return nil, status.Error(codes.FailedPrecondition, "chain params not configured")
	}

	// Parse and canonicalize compressed pubkey.
	if len(req.CompPubkey) != 33 {
		return nil, status.Error(codes.InvalidArgument, "comp_pubkey (33 bytes) required")
	}
	pub, err := secp256k1.ParsePubKey(req.CompPubkey)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid comp_pubkey: %v", err)
	}
	comp := pub.SerializeCompressed()

	// Build redeem + deposit address.
	redeem, err := buildPerDepositorRedeemScript(comp, req.CsvBlocks)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "build redeem: %v", err)
	}
	pkScriptHex, depositAddr, err := pkScriptAndAddrFromRedeem(redeem, s.chainParams)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "derive deposit address: %v", err)
	}

	// Ensure payout address matches network.
	if _, err := stdaddr.DecodeAddress(payoutAddr, s.chainParams); err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "payout address network mismatch: %v", err)
	}

	es := &refereeEscrowSession{
		EscrowID:        newEscrowID(),
		OwnerUID:        ownerUID,
		Token:           strings.TrimSpace(req.Token),
		CompPubkey:      append([]byte(nil), comp...),
		PayoutAddr:      payoutAddr,
		AmountAtoms:     req.AmountAtoms,
		CSVBlocks:       req.CsvBlocks,
		RedeemScriptHex: hex.EncodeToString(redeem),
		PkScriptHex:     pkScriptHex,
	}

	// Register in referee state.
	s.referee.mu.Lock()
	s.referee.escrows[es.EscrowID] = es
	s.referee.mu.Unlock()

	// Subscribe to funding updates if watcher available.
	if s.chainWatcher != nil {
		updates, unsub := s.chainWatcher.Subscribe(pkScriptHex)
		es.WatcherUnsub = unsub
		go s.trackEscrowFunding(es, updates)
	}
	// Bind immediately if already funded.
	es.mu.RLock()
	lf := es.LatestFunding
	es.mu.RUnlock()
	if lf.UTXOCount == 1 && len(lf.UTXOs) == 1 {
		_ = enforceSingleFunding(es)
	}

	return &pokerrpc.OpenEscrowResponse{
		EscrowId:              es.EscrowID,
		DepositAddr:           depositAddr,
		RedeemScriptHex:       es.RedeemScriptHex,
		PkScriptHex:           es.PkScriptHex,
		MatchId:               "",
		RequiredConfirmations: 1,
	}, nil
}

// BindEscrow attaches an existing escrow to a table/seat and checks funding.
func (s *Server) BindEscrow(ctx context.Context, req *pokerrpc.BindEscrowRequest) (*pokerrpc.BindEscrowResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	token := strings.TrimSpace(req.GetToken())
	if token == "" {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if vals := md.Get("token"); len(vals) > 0 {
				token = strings.TrimSpace(vals[0])
			}
		}
	}
	if token == "" {
		return nil, status.Error(codes.Unauthenticated, "token required")
	}
	uid, payoutAddr, ok := s.payoutForToken(token)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "invalid or expired session")
	}
	if payoutAddr == "" {
		return nil, status.Error(codes.FailedPrecondition, "payout address not set")
	}

	outpoint := strings.TrimSpace(req.GetOutpoint())
	if outpoint == "" {
		return nil, status.Error(codes.InvalidArgument, "outpoint required")
	}
	tableID := strings.TrimSpace(req.GetTableId())
	sessionID := strings.TrimSpace(req.GetSessionId())
	seat := req.GetSeatIndex()

	if tableID == "" {
		return nil, status.Error(codes.InvalidArgument, "table_id required")
	}

	// Auto-detect seat from caller's position at the table.
	// We always auto-detect to ensure the caller binds to their own seat.
	// The seat_index parameter is ignored for security.
	callerID := uid.String()
	table, ok := s.getTable(tableID)
	if !ok || table == nil {
		return nil, status.Error(codes.NotFound, "table not found")
	}
	// Find the caller at the table and get their seat
	callerUser := table.GetUser(callerID)
	if callerUser == nil {
		return nil, status.Error(codes.FailedPrecondition, "you are not seated at this table")
	}
	seat = uint32(callerUser.TableSeat)

	parts := strings.Split(outpoint, ":")
	if len(parts) != 2 {
		return nil, status.Error(codes.InvalidArgument, "outpoint must be txid:vout")
	}
	txid := parts[0]
	vout64, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "outpoint vout invalid")
	}
	vout := uint32(vout64)

	var (
		es       *refereeEscrowSession
		escrowID string
	)
	s.referee.mu.RLock()
	for _, cand := range s.referee.escrows {
		if cand == nil {
			continue
		}
		cand.mu.RLock()
		bu := cloneUTXO(cand.BoundUTXO)
		var lf chainwatcher.DepositUpdate
		if len(cand.LatestFunding.UTXOs) == 1 {
			lf = cand.LatestFunding
		}
		cand.mu.RUnlock()
		if bu != nil && bu.Txid == txid && bu.Vout == vout {
			es = cand
			break
		}
		if len(lf.UTXOs) == 1 {
			u := lf.UTXOs[0]
			if u.Txid == txid && u.Vout == vout {
				es = cand
				break
			}
		}
	}
	s.referee.mu.RUnlock()
	if es != nil {
		if es.OwnerUID != uid {
			return nil, status.Error(codes.PermissionDenied, "escrow not owned by caller")
		}
		escrowID = es.EscrowID
	}
	if es == nil {
		if req.RedeemScriptHex == "" || req.CsvBlocks == 0 {
			return nil, status.Error(codes.NotFound, "escrow not found for outpoint")
		}
		if s.chainWatcher == nil || s.chainParams == nil {
			return nil, status.Error(codes.FailedPrecondition, "chain watcher not configured")
		}
		redeem, err := hex.DecodeString(req.RedeemScriptHex)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid redeem_script_hex")
		}
		pkHex, _, err := pkScriptAndAddrFromRedeem(redeem, s.chainParams)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "derive pkScript: %v", err)
		}
		u, err := s.chainWatcher.LookupUTXO(outpoint, pkHex)
		if err != nil {
			return nil, status.Errorf(codes.NotFound, "utxo not found on chain: %v", err)
		}
		var compPub []byte
		if len(redeem) >= 35 && redeem[0] == txscript.OP_IF && redeem[1] == txscript.OP_DATA_33 {
			compPub = redeem[2 : 2+33]
		}
		if len(compPub) != 33 {
			return nil, status.Error(codes.InvalidArgument, "redeem script format not recognized")
		}
		es = &refereeEscrowSession{
			EscrowID:        newEscrowID(),
			OwnerUID:        uid,
			Token:           token,
			TableID:         tableID,
			SessionID:       sessionID,
			SeatIndex:       seat,
			CompPubkey:      compPub,
			PayoutAddr:      payoutAddr,
			AmountAtoms:     u.Value,
			CSVBlocks:       req.CsvBlocks,
			RedeemScriptHex: req.RedeemScriptHex,
			PkScriptHex:     pkHex,
			BoundUTXO:       u,
			LatestFunding: chainwatcher.DepositUpdate{
				PkScriptHex: pkHex,
				Confs:       u.Confs,
				UTXOCount:   1,
				OK:          true,
				At:          time.Now(),
				UTXOs:       []*chainwatcher.EscrowUTXO{u},
			},
		}
		escrowID = es.EscrowID
		s.referee.mu.Lock()
		s.referee.escrows[escrowID] = es
		s.referee.mu.Unlock()
	}
	matchID := strings.TrimSpace(req.GetMatchId())
	if matchID == "" && tableID != "" {
		// For WTA poker, tableID alone is the matchID (no sessionID suffix needed)
		matchID = tableID
	}
	if matchID == "" {
		return nil, status.Error(codes.InvalidArgument, "match_id or table_id required")
	}

	// Enforce table buy-in amount match if table exists.
	requiredAmount := uint64(0)
	if table, ok := s.getTable(tableID); ok && table != nil {
		cfg := table.GetConfig()
		requiredAmount = uint64(cfg.BuyIn)
		if cfg.BuyIn > 0 && es.AmountAtoms != uint64(cfg.BuyIn) {
			return nil, status.Errorf(codes.FailedPrecondition, "escrow amount %d must equal table buy-in %d", es.AmountAtoms, cfg.BuyIn)
		}
	}

	// If escrow lacks funding info, attempt on-chain lookup using provided redeem/csv.
	es.mu.RLock()
	hasBound := es.BoundUTXO != nil
	es.mu.RUnlock()
	if !hasBound {
		if req.RedeemScriptHex == "" || req.CsvBlocks == 0 {
			return nil, status.Error(codes.FailedPrecondition, "escrow not funded and redeem/csv not provided")
		}
		if s.chainWatcher == nil {
			return nil, status.Error(codes.FailedPrecondition, "chain watcher not configured")
		}
		// Build pkScript from redeem.
		redeem, err := hex.DecodeString(req.RedeemScriptHex)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid redeem_script_hex")
		}
		pkHex, _, err := pkScriptAndAddrFromRedeem(redeem, s.chainParams)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "derive pkScript: %v", err)
		}
		// Query watcher directly.
		u, err := s.chainWatcher.LookupUTXO(outpoint, pkHex)
		if err != nil {
			return nil, status.Errorf(codes.NotFound, "utxo not found on chain: %v", err)
		}
		es.mu.Lock()
		es.RedeemScriptHex = req.RedeemScriptHex
		es.CSVBlocks = req.CsvBlocks
		es.PkScriptHex = pkHex
		es.BoundUTXO = cloneUTXO(u)
		es.LatestFunding = chainwatcher.DepositUpdate{
			PkScriptHex: pkHex,
			Confs:       u.Confs,
			UTXOCount:   1,
			OK:          true,
			At:          time.Now(),
			UTXOs:       []*chainwatcher.EscrowUTXO{u},
		}
		es.AmountAtoms = u.Value
		es.mu.Unlock()
	}

	// Ensure single utxo funding bound.
	if _, err := ensureBoundFunding(es); err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "escrow not funded: %v", err)
	}

	// Bind state.
	es.mu.Lock()
	es.TableID = tableID
	es.SessionID = sessionID
	es.SeatIndex = seat
	ready := es.BoundUTXO != nil && es.BoundUTXO.Value == es.AmountAtoms
	es.mu.Unlock()
	s.referee.mu.Lock()
	if s.referee.matchEscrows[matchID] == nil {
		s.referee.matchEscrows[matchID] = make(map[uint32]string)
	}
	if existing := s.referee.matchEscrows[matchID][seat]; existing != "" && existing != escrowID {
		s.referee.mu.Unlock()
		return nil, status.Errorf(codes.AlreadyExists, "seat %d already bound", seat)
	}
	s.referee.matchEscrows[matchID][seat] = escrowID
	s.referee.mu.Unlock()

	// Notify the user's state machine about the escrow binding (similar to SendReady pattern).
	if err := callerUser.SendEscrowBound(escrowID, ready); err != nil {
		s.log.Warnf("Failed to send escrow bound event: %v", err)
		// Don't fail the bind if user event fails - escrow is still bound in referee state
	}

	return &pokerrpc.BindEscrowResponse{
		MatchId:             matchID,
		TableId:             tableID,
		SessionId:           sessionID,
		SeatIndex:           seat,
		EscrowId:            escrowID,
		EscrowReady:         ready,
		AmountAtoms:         es.AmountAtoms,
		RequiredAmountAtoms: requiredAmount,
		Outpoint:            outpoint,
	}, nil
}

// PublishSessionKey records the client's session pubkey used during presign.
func (s *Server) PublishSessionKey(ctx context.Context, req *pokerrpc.PublishSessionKeyRequest) (*pokerrpc.PublishSessionKeyResponse, error) {
	return nil, status.Error(codes.Unimplemented, refereeNotReadyMsg)
}

// SettlementStream coordinates presign exchanges for all branches.
func (s *Server) SettlementStream(stream pokerrpc.PokerReferee_SettlementStreamServer) error {
	first, err := stream.Recv()
	if err != nil || first == nil || first.GetHello() == nil {
		return status.Error(codes.Unauthenticated, "hello with token required")
	}
	hello := first.GetHello()
	token := strings.TrimSpace(hello.Token)
	if token == "" {
		if md, ok := metadata.FromIncomingContext(stream.Context()); ok {
			if vals := md.Get("token"); len(vals) > 0 {
				token = strings.TrimSpace(vals[0])
			}
		}
	}
	if token == "" {
		return status.Error(codes.Unauthenticated, "token required")
	}
	uid, _, ok := s.payoutForToken(token)
	if !ok {
		return status.Error(codes.Unauthenticated, "invalid or expired session")
	}

	matchID := strings.TrimSpace(hello.MatchId)
	escrowID := strings.TrimSpace(hello.EscrowId)
	if escrowID == "" {
		return status.Error(codes.InvalidArgument, "escrow_id required")
	}

	// Resolve escrow and ensure it belongs to caller.
	s.referee.mu.RLock()
	es, ok := s.referee.escrows[escrowID]
	s.referee.mu.RUnlock()
	if !ok || es == nil {
		return status.Error(codes.NotFound, "escrow not found")
	}
	if es.OwnerUID != uid {
		return status.Error(codes.PermissionDenied, "escrow not owned by caller")
	}
	if _, err := ensureBoundFunding(es); err != nil {
		return status.Errorf(codes.FailedPrecondition, "escrow not funded: %v", err)
	}

	// Bind escrow to match/seat if provided.
	tableID := strings.TrimSpace(hello.TableId)
	if tableID == "" {
		return status.Error(codes.InvalidArgument, "table_id required")
	}

	sessionID := strings.TrimSpace(hello.SessionId)
	seat := hello.SeatIndex
	// For WTA poker, tableID alone is the matchID (sessionID is optional/ignored)
	if matchID == "" {
		matchID = tableID
	}

	s.referee.mu.Lock()
	if s.referee.matchEscrows[matchID] == nil {
		s.referee.matchEscrows[matchID] = make(map[uint32]string)
	}
	if existing := s.referee.matchEscrows[matchID][seat]; existing != "" && existing != escrowID {
		s.referee.mu.Unlock()
		return status.Errorf(codes.AlreadyExists, "seat %d already bound", seat)
	}
	s.referee.matchEscrows[matchID][seat] = escrowID
	s.referee.mu.Unlock()

	es.mu.Lock()
	es.TableID = tableID
	es.SessionID = sessionID
	es.SeatIndex = seat
	es.mu.Unlock()

	// Enforce escrow amount matches table buy-in and record funding state into table model.
	table, ok := s.getTable(tableID)
	if !ok || table == nil {
		return status.Error(codes.NotFound, "table not found for escrow binding")
	}
	cfg := table.GetConfig()
	if cfg.BuyIn > 0 && es.AmountAtoms != uint64(cfg.BuyIn) {
		return status.Errorf(codes.FailedPrecondition, "escrow amount %d must equal table buy-in %d", es.AmountAtoms, cfg.BuyIn)
	}
	es.mu.RLock()
	ready := es.BoundUTXO != nil && es.BoundUTXO.Value == es.AmountAtoms
	es.mu.RUnlock()
	if ready {
		user := table.GetUser(uid.String())
		if user != nil {
			if err := user.SendEscrowBound(escrowID, ready); err != nil {
				s.log.Warnf("Failed to send escrow bound event: %v", err)
			}
		}
	}
	// Enforce presign only when table is "full": all escrows for the match
	// are funded and bound to a single UTXO (2-6 players).
	allEscrows, err := s.readyMatchEscrows(matchID)
	if err != nil {
		return status.Errorf(codes.FailedPrecondition, "match not ready for presign: %v", err)
	}

	// Build drafts per possible winner and request presigs.
	branchDrafts, err := s.buildWTADrafts(matchID, allEscrows)
	if err != nil {
		return status.Errorf(codes.Internal, "build drafts: %v", err)
	}

	// Cache gamma per branch.
	s.referee.mu.Lock()
	if s.referee.branchGamma[matchID] == nil {
		s.referee.branchGamma[matchID] = make(map[int32]string)
	}
	for idx, draft := range branchDrafts {
		s.referee.branchGamma[matchID][int32(idx)] = draft.GammaHex
	}
	s.referee.mu.Unlock()

	draftMap := make(map[int32]branchDraft)
	for idx, d := range branchDrafts {
		draftMap[int32(idx)] = d
	}

	// Stream NeedPreSigs for each branch to this caller (their own inputs only).
	for branchIdx, draft := range branchDrafts {
		need := &pokerrpc.NeedPreSigs{
			MatchId:    matchID,
			Branch:     int32(branchIdx),
			DraftTxHex: draft.DraftHex,
		}
		for _, in := range draft.Inputs {
			if in.OwnerUID != uid {
				continue
			}
			need.Inputs = append(need.Inputs, &pokerrpc.NeedPreSigsInput{
				InputId:         in.InputID,
				RedeemScriptHex: in.RedeemScriptHex,
				SighashHex:      in.SighashHex,
				AdaptorPointHex: in.AdaptorPointHex,
				InputIndex:      in.InputIndex,
				AmountAtoms:     in.AmountAtoms,
			})
		}
		if len(need.Inputs) == 0 {
			continue
		}
		if err := stream.Send(&pokerrpc.SettlementStreamMessage{Msg: &pokerrpc.SettlementStreamMessage_NeedPreSigs{NeedPreSigs: need}}); err != nil {
			return err
		}
	}

	// Receive presigs, verify, and store.
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			// Mark this seat's presigning as complete in referee state.
			s.referee.mu.Lock()
			if s.referee.presignComplete[matchID] == nil {
				s.referee.presignComplete[matchID] = make(map[uint32]bool)
			}
			s.referee.presignComplete[matchID][seat] = true
			s.referee.mu.Unlock()

			// Also update the table FSM so the lobby knows presigning is done.
			if tableID != "" {
				if table, ok := s.getTable(tableID); ok && table != nil {
					playerID := uid.String()
					if err := table.SetPlayerPresignComplete(playerID); err != nil {
						s.log.Warnf("Failed to set presign complete on table %s for player %s: %v", tableID, playerID, err)
					} else {
						s.log.Debugf("Set presign complete on table %s for player %s", tableID, playerID)
						// Check if we should start the game now that presigning is complete
						s.maybeStartGameAfterPresign(table, tableID, matchID, playerID)
					}
				}
			}

			s.log.Infof("Presigning complete for match %s seat %d", matchID, seat)
			return nil
		}
		if err != nil {
			return err
		}
		ps := msg.GetProvidePreSigs()
		if ps == nil {
			continue
		}
		draft, ok := draftMap[ps.Branch]
		if !ok {
			return status.Errorf(codes.InvalidArgument, "unknown branch %d", ps.Branch)
		}
		if err := s.storePresigs(matchID, ps.Branch, uid, draft, ps.Presigs); err != nil {
			return status.Errorf(codes.InvalidArgument, "presig verify: %v", err)
		}
		// Ack success for this branch.
		if err := stream.Send(&pokerrpc.SettlementStreamMessage{Msg: &pokerrpc.SettlementStreamMessage_VerifyOk{
			VerifyOk: &pokerrpc.VerifyPreSigsOk{MatchId: matchID, Branch: ps.Branch},
		}}); err != nil {
			return err
		}
	}
}

// trackEscrowFunding records funding updates into the escrow session.
func (s *Server) trackEscrowFunding(es *refereeEscrowSession, updates <-chan chainwatcher.DepositUpdate) {
	for u := range updates {
		es.mu.Lock()
		prevConfs := es.LatestFunding.Confs
		prevUTXOCount := es.LatestFunding.UTXOCount
		es.LatestFunding = u
		s.log.Debugf("escrow funding update id=%s confs=%d utxos=%d prevConfs=%d", es.EscrowID, u.Confs, u.UTXOCount, prevConfs)
		// Auto-bind when exactly one UTXO and at least 1 conf.
		if u.UTXOCount == 1 && u.Confs >= 1 && len(u.UTXOs) == 1 {
			if u.UTXOs[0].Value == es.AmountAtoms {
				es.BoundUTXO = cloneUTXO(u.UTXOs[0])
			} else {
				es.BoundUTXO = nil
			}
		} else if u.UTXOCount != 1 {
			es.BoundUTXO = nil
		}
		es.mu.Unlock()

		if u.UTXOCount > 1 && u.UTXOCount != prevUTXOCount {
			payload := map[string]interface{}{
				"type":       "escrow_funding",
				"escrow_id":  es.EscrowID,
				"utxo_count": u.UTXOCount,
				"error":      fmt.Sprintf("expected a single funding UTXO, found %d", u.UTXOCount),
			}
			if msg, err := json.Marshal(payload); err == nil {
				s.notifyPlayer(es.OwnerUID.String(), &pokerrpc.Notification{
					Type:     pokerrpc.NotificationType_ESCROW_FUNDING,
					Message:  string(msg),
					PlayerId: es.OwnerUID.String(),
				})
			} else {
				s.log.Warnf("trackEscrowFunding: marshal multi-utxo payload: %v", err)
			}
		}

		// Emit funding status updates to the escrow owner so the UI can refresh
		// cached escrow metadata without polling. Notify once on first sighting
		// (mempool) and once on first confirmation to reduce noise.
		shouldNotify := (prevUTXOCount == 0 && u.UTXOCount >= 1) || (u.Confs == 1 && prevConfs < 1)
		if shouldNotify {
			amountAtoms := es.AmountAtoms
			if len(u.UTXOs) >= 1 {
				amountAtoms = u.UTXOs[0].Value
			}
			payload := map[string]interface{}{
				"type":                   "escrow_funding",
				"escrow_id":              es.EscrowID,
				"confs":                  u.Confs,
				"utxo_count":             u.UTXOCount,
				"csv_blocks":             es.CSVBlocks,
				"required_confirmations": uint32(1),
				"mature_for_csv":         u.Confs >= es.CSVBlocks,
				"amount_atoms":           amountAtoms,
			}
			if len(u.UTXOs) >= 1 {
				payload["funding_txid"] = u.UTXOs[0].Txid
				payload["funding_vout"] = u.UTXOs[0].Vout
				if u.Confs > 0 {
					payload["confirmed_height"] = u.Confs // height unavailable; best-effort marker
				}
			}
			msg, err := json.Marshal(payload)
			if err != nil {
				s.log.Warnf("trackEscrowFunding: marshal payload: %v", err)
				continue
			}
			s.notifyPlayer(es.OwnerUID.String(), &pokerrpc.Notification{
				Type:     pokerrpc.NotificationType_ESCROW_FUNDING,
				Message:  string(msg),
				PlayerId: es.OwnerUID.String(),
			})
			s.log.Debugf("escrow funding notify id=%s confs=%d utxos=%d", es.EscrowID, u.Confs, u.UTXOCount)

		}

	}
}

// GetFinalizeBundle returns the winning branch draft and presign artifacts.
// The WinnerSeat parameter is interpreted as a table seat index and mapped to the
// corresponding branch index (branches are indexed by sorted UTXO order, not seat order).
func (s *Server) GetFinalizeBundle(ctx context.Context, req *pokerrpc.GetFinalizeBundleRequest) (*pokerrpc.GetFinalizeBundleResponse, error) {
	if req == nil || strings.TrimSpace(req.MatchId) == "" {
		return nil, status.Error(codes.InvalidArgument, "match_id required")
	}
	matchID := strings.TrimSpace(req.MatchId)
	tableSeat := req.WinnerSeat
	if tableSeat < 0 {
		return nil, status.Error(codes.InvalidArgument, "winner_seat required")
	}

	// Ensure match is fully bound/funded
	escrows, err := s.readyMatchEscrows(matchID)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "match not ready: %v", err)
	}

	// Map table seat to branch index (branches are indexed by sorted UTXO order, not seat order)
	branch, err := s.seatToBranchIndex(matchID, tableSeat)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to map table seat %d to branch: %v", tableSeat, err)
	}

	if int(branch) >= len(escrows) {
		return nil, status.Errorf(codes.InvalidArgument, "branch %d (from seat %d) out of range (n=%d)", branch, tableSeat, len(escrows))
	}

	// Fetch presigns + gamma for requested branch.
	s.referee.mu.RLock()
	pres := s.referee.presigns[matchID][branch]
	gammaHex := s.referee.branchGamma[matchID][branch]
	s.referee.mu.RUnlock()
	if gammaHex == "" {
		return nil, status.Errorf(codes.FailedPrecondition, "gamma not cached for match %s branch %d", matchID, branch)
	}
	if len(pres) == 0 {
		return nil, status.Errorf(codes.FailedPrecondition, "no presigs stored for match %s branch %d", matchID, branch)
	}

	// Collect inputs sorted by input index for determinism.
	var inputs []*refereePreSignCtx
	for _, ctx := range pres {
		inputs = append(inputs, ctx)
	}
	sort.Slice(inputs, func(i, j int) bool { return inputs[i].InputIndex < inputs[j].InputIndex })

	// Ensure we have one presig per escrow/input.
	if len(inputs) != len(escrows) {
		return nil, status.Errorf(codes.FailedPrecondition, "presigs incomplete for branch %d: have %d, want %d", branch, len(inputs), len(escrows))
	}

	resp := &pokerrpc.GetFinalizeBundleResponse{
		MatchId:    matchID,
		Branch:     branch,
		DraftTxHex: inputs[0].DraftHex,
		GammaHex:   gammaHex,
	}
	for _, in := range inputs {
		resp.Inputs = append(resp.Inputs, &pokerrpc.FinalizeInput{
			InputId:          in.InputID,
			RPrimeCompactHex: hex.EncodeToString(in.RPrimeCompressed),
			SPrimeHex:        hex.EncodeToString(in.SPrime32),
			InputIndex:       in.InputIndex,
			RedeemScriptHex:  in.RedeemScriptHex,
		})
	}
	return resp, nil
}

// FinalizeAndBroadcastSettlement completes the Schnorr settlement for a match:
// - Retrieves the finalize bundle for the winning branch
// - Computes final signatures: s = s' + gamma (mod n) for each input
// - Attaches Schnorr signatures to the draft transaction
// - Broadcasts the finalized transaction via dcrd
// Returns the txid of the broadcast transaction.
func (s *Server) FinalizeAndBroadcastSettlement(ctx context.Context, matchID string, winnerSeat int32) (string, error) {
	if s.dcrd == nil {
		return "", fmt.Errorf("dcrd client not connected; cannot broadcast")
	}

	// Get the finalize bundle with gamma and presigs.
	bundle, err := s.GetFinalizeBundle(ctx, &pokerrpc.GetFinalizeBundleRequest{
		MatchId:    matchID,
		WinnerSeat: winnerSeat,
	})
	if err != nil {
		return "", fmt.Errorf("GetFinalizeBundle failed: %w", err)
	}

	// Decode draft transaction.
	draftBytes, err := hex.DecodeString(bundle.DraftTxHex)
	if err != nil {
		return "", fmt.Errorf("decode draft tx hex: %w", err)
	}
	tx, err := decodeMsgTx(draftBytes)
	if err != nil {
		return "", fmt.Errorf("deserialize draft tx: %w", err)
	}

	// Decode gamma scalar.
	gammaBytes, err := hex.DecodeString(bundle.GammaHex)
	if err != nil || len(gammaBytes) != 32 {
		return "", fmt.Errorf("invalid gamma hex")
	}
	var gamma secp256k1.ModNScalar
	gamma.SetBytes((*[32]byte)(gammaBytes))

	// Build signature scripts for each input.
	for _, fin := range bundle.Inputs {
		if int(fin.InputIndex) >= len(tx.TxIn) {
			return "", fmt.Errorf("input index %d out of range", fin.InputIndex)
		}

		// Decode R' (compressed point) and s' (scalar).
		rPrimeBytes, err := hex.DecodeString(fin.RPrimeCompactHex)
		if err != nil || len(rPrimeBytes) != 33 {
			return "", fmt.Errorf("invalid R' for input %s", fin.InputId)
		}
		sPrimeBytes, err := hex.DecodeString(fin.SPrimeHex)
		if err != nil || len(sPrimeBytes) != 32 {
			return "", fmt.Errorf("invalid s' for input %s", fin.InputId)
		}

		// Compute final s = s' + gamma (mod n).
		var sPrime secp256k1.ModNScalar
		sPrime.SetBytes((*[32]byte)(sPrimeBytes))
		var sFinal secp256k1.ModNScalar
		sFinal.Add2(&sPrime, &gamma)
		sBytes := sFinal.Bytes()

		// Build Schnorr signature: R' (32 bytes x-coord) || s (32 bytes) || SigHashAll (1 byte).
		// For Decred Schnorr, we need the 32-byte x-coordinate of R'.
		rPrimePub, err := secp256k1.ParsePubKey(rPrimeBytes)
		if err != nil {
			return "", fmt.Errorf("parse R' pubkey for input %s: %w", fin.InputId, err)
		}
		var rX secp256k1.FieldVal
		rX.SetByteSlice(rPrimePub.X().Bytes())
		rXBytes := rX.Bytes()

		// Build 65-byte signature: 32 bytes rX + 32 bytes s + 1 byte SigHashAll
		sig65 := make([]byte, 0, 65)
		sig65 = append(sig65, rXBytes[:]...)
		sig65 = append(sig65, sBytes[:]...)
		sig65 = append(sig65, byte(txscript.SigHashAll))

		// Decode redeem script for P2SH signing.
		redeemScript, err := hex.DecodeString(fin.RedeemScriptHex)
		if err != nil {
			return "", fmt.Errorf("decode redeem script for input %s: %w", fin.InputId, err)
		}

		// Build signature script: <sig65> <OP_1> <redeemScript> for P2SH.
		// The redeem script starts with OP_IF, which requires a non-zero value (OP_1)
		// on the stack to take the winner branch (immediate spend).
		// Order matches working reference implementation.
		sigScript, err := txscript.NewScriptBuilder().
			AddData(sig65).
			AddOp(txscript.OP_1).
			AddData(redeemScript).
			Script()
		if err != nil {
			return "", fmt.Errorf("build sig script for input %s: %w", fin.InputId, err)
		}

		tx.TxIn[fin.InputIndex].SignatureScript = sigScript
	}

	// Broadcast the finalized transaction.
	txHash, err := s.dcrd.SendRawTransaction(ctx, tx, true)
	if err != nil {
		return "", fmt.Errorf("SendRawTransaction failed: %w", err)
	}

	txid := txHash.String()
	s.log.Infof("Settlement broadcast successful: matchID=%s winnerSeat=%d branch=%d txid=%s",
		matchID, winnerSeat, bundle.Branch, txid)

	// TODO: Persist settlement outcome and clean up escrow/presign state.

	return txid, nil
}

// GetEscrowStatus returns funding/conf status for an escrow owned by caller.
func (s *Server) GetEscrowStatus(ctx context.Context, req *pokerrpc.GetEscrowStatusRequest) (*pokerrpc.GetEscrowStatusResponse, error) {
	token := strings.TrimSpace(req.GetToken())
	if token == "" {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if vals := md.Get("token"); len(vals) > 0 {
				token = strings.TrimSpace(vals[0])
			}
		}
	}
	if token == "" {
		return nil, status.Error(codes.Unauthenticated, "token required")
	}

	escrowID := strings.TrimSpace(req.GetEscrowId())
	if escrowID == "" {
		return nil, status.Error(codes.InvalidArgument, "escrow_id required")
	}

	s.referee.mu.RLock()
	es, ok := s.referee.escrows[escrowID]
	s.referee.mu.RUnlock()
	if !ok || es == nil {
		return nil, status.Error(codes.NotFound, "escrow not found")
	}
	if es.Token != "" && es.Token != token {
		s.log.Debugf("GetEscrowStatus deny token mismatch escrow=%s have=%d want=%d", escrowID, len(token), len(es.Token))
		return nil, status.Error(codes.PermissionDenied, "escrow not owned by caller")
	}

	resp := &pokerrpc.GetEscrowStatusResponse{
		EscrowId:              es.EscrowID,
		CsvBlocks:             es.CSVBlocks,
		RequiredConfirmations: 1,
		AmountAtoms:           es.AmountAtoms,
		PkScriptHex:           es.PkScriptHex,
	}

	es.mu.RLock()
	lf := es.LatestFunding
	resp.Confs = lf.Confs
	resp.UtxoCount = uint32(lf.UTXOCount)
	resp.Ok = lf.OK
	if !lf.At.IsZero() {
		resp.UpdatedAtUnix = lf.At.Unix()
	}
	if len(lf.UTXOs) == 1 {
		resp.FundingTxid = lf.UTXOs[0].Txid
		resp.FundingVout = lf.UTXOs[0].Vout
	}
	if resp.Confs >= es.CSVBlocks && es.CSVBlocks > 0 {
		resp.MatureForCsv = true
	}
	es.mu.RUnlock()

	return resp, nil
}

// TestBindEscrowFunding seeds funding info for an escrow (testing only).
func (s *Server) TestBindEscrowFunding(escrowID, txid string, vout uint32, amount uint64) {
	if s.referee == nil {
		return
	}
	s.referee.mu.Lock()
	es := s.referee.escrows[escrowID]
	s.referee.mu.Unlock()
	if es == nil {
		return
	}
	u := &chainwatcher.EscrowUTXO{Txid: txid, Vout: vout, Value: amount, PkScriptHex: es.PkScriptHex}
	es.mu.Lock()
	es.LatestFunding = chainwatcher.DepositUpdate{
		PkScriptHex: es.PkScriptHex,
		Confs:       1,
		UTXOCount:   1,
		OK:          true,
		UTXOs:       []*chainwatcher.EscrowUTXO{u},
	}
	es.BoundUTXO = cloneUTXO(u)
	es.mu.Unlock()
}

// TestBindEscrowToMatch binds an escrow to a match/seat and table user model (testing only).
// This sets up the escrow bindings without completing presigning.
func (s *Server) TestBindEscrowToMatch(escrowID, matchID, tableID, playerID string, seat uint32) {
	if s.referee == nil {
		return
	}
	s.referee.mu.Lock()
	es := s.referee.escrows[escrowID]
	if es == nil {
		s.referee.mu.Unlock()
		return
	}
	// Bind escrow to matchEscrows.
	if s.referee.matchEscrows[matchID] == nil {
		s.referee.matchEscrows[matchID] = make(map[uint32]string)
	}
	s.referee.matchEscrows[matchID][seat] = escrowID
	s.referee.mu.Unlock()

	// Bind to table user model.
	es.mu.Lock()
	es.TableID = tableID
	es.SeatIndex = seat
	es.mu.Unlock()

	// Update user's escrow binding in the table.
	table, ok := s.getTable(tableID)
	if !ok || table == nil {
		return
	}

	// Determine if escrow is ready (has bound UTXO with correct value).
	es.mu.RLock()
	ready := es.BoundUTXO != nil && es.BoundUTXO.Value == es.AmountAtoms
	es.mu.RUnlock()

	// Update the user's escrow binding via the table FSM.
	if err := table.SetPlayerEscrow(playerID, escrowID, ready); err != nil {
		// Log error but don't fail test helper
		return
	}
}

// readyMatchEscrows returns all escrows for a match, ensuring 2-6 entries and
// that each has a bound UTXO and session pubkey.
func (s *Server) readyMatchEscrows(matchID string) ([]*refereeEscrowSession, error) {
	s.referee.mu.RLock()
	seatsMap := s.referee.matchEscrows[matchID]
	seats := make(map[uint32]string, len(seatsMap))
	for k, v := range seatsMap {
		seats[k] = v
	}
	s.referee.mu.RUnlock()
	if len(seats) < 2 || len(seats) > 6 {
		return nil, fmt.Errorf("match seats not filled (have %d, need 2-6)", len(seats))
	}

	var seatIdxs []int
	for idx := range seats {
		seatIdxs = append(seatIdxs, int(idx))
	}
	sort.Ints(seatIdxs)

	var escrows []*refereeEscrowSession
	for _, si := range seatIdxs {
		eid := seats[uint32(si)]
		if eid == "" {
			return nil, fmt.Errorf("seat %d has no escrow", si)
		}
		s.referee.mu.RLock()
		es := s.referee.escrows[eid]
		s.referee.mu.RUnlock()
		if es == nil {
			return nil, fmt.Errorf("escrow %s missing", eid)
		}
		if _, err := ensureBoundFunding(es); err != nil {
			return nil, fmt.Errorf("escrow %s not bound to funding: %v", eid, err)
		}
		es.mu.RLock()
		compLen := len(es.CompPubkey)
		es.mu.RUnlock()
		if compLen != 33 {
			return nil, fmt.Errorf("escrow %s missing session pubkey", eid)
		}
		escrows = append(escrows, es)
	}
	return escrows, nil
}

// seatToBranchIndex maps a table seat index to the corresponding branch index.
// Branches are indexed by the sorted UTXO order (PkScriptHex, Txid, Vout), not seat order.
// This function replicates the same sorting logic used in buildWTADrafts to find the correct branch.
func (s *Server) seatToBranchIndex(matchID string, tableSeat int32) (int32, error) {
	if tableSeat < 0 {
		return -1, fmt.Errorf("invalid table seat: %d", tableSeat)
	}

	// Get all escrows for the match
	escrows, err := s.readyMatchEscrows(matchID)
	if err != nil {
		return -1, fmt.Errorf("get escrows: %w", err)
	}

	// Create snapshots and sort them the same way buildWTADrafts does
	var snaps []escrowSnapshot
	for _, es := range escrows {
		snap := es.snapshot()
		if snap.BoundUTXO == nil {
			return -1, fmt.Errorf("escrow %s missing bound utxo", snap.EscrowID)
		}
		snaps = append(snaps, snap)
	}

	// Sort inputs deterministically (pkScript+txid:vout) - same logic as buildWTADrafts
	sort.Slice(snaps, func(i, j int) bool {
		a := snaps[i].PkScriptHex
		b := snaps[j].PkScriptHex
		if a == b {
			if snaps[i].BoundUTXO.Txid == snaps[j].BoundUTXO.Txid {
				return snaps[i].BoundUTXO.Vout < snaps[j].BoundUTXO.Vout
			}
			return snaps[i].BoundUTXO.Txid < snaps[j].BoundUTXO.Txid
		}
		return a < b
	})

	// Find the index of the escrow whose SeatIndex matches the table seat
	for idx, snap := range snaps {
		if snap.SeatIndex == uint32(tableSeat) {
			return int32(idx), nil
		}
	}

	return -1, fmt.Errorf("table seat %d not found in match escrows", tableSeat)
}

// IsPresigningComplete checks if all seats bound to a match have completed presigning.
// Returns true if presigning is complete for all seats, false otherwise.
// Also returns the number of seats that have completed and total seats.
func (s *Server) IsPresigningComplete(matchID string) (complete bool, completedSeats, totalSeats int) {
	s.referee.mu.RLock()
	defer s.referee.mu.RUnlock()

	seatsMap := s.referee.matchEscrows[matchID]
	if len(seatsMap) < 2 {
		return false, 0, len(seatsMap)
	}

	totalSeats = len(seatsMap)
	presignMap := s.referee.presignComplete[matchID]
	if presignMap == nil {
		return false, 0, totalSeats
	}

	for seat := range seatsMap {
		if presignMap[seat] {
			completedSeats++
		}
	}

	return completedSeats == totalSeats, completedSeats, totalSeats
}

// storePresigs verifies and stores presign artifacts for a branch.
func (s *Server) storePresigs(matchID string, branch int32, uid zkidentity.ShortID, draft branchDraft, presigs []*pokerrpc.PreSignature) error {
	inputs := make(map[string]presignInput)
	ownerCount := 0
	for _, in := range draft.Inputs {
		inputs[in.InputID] = in
		if in.OwnerUID == uid {
			ownerCount++
		}
	}
	if ownerCount == 0 {
		return fmt.Errorf("no inputs for caller in branch %d", branch)
	}
	if len(presigs) != ownerCount {
		return fmt.Errorf("expected %d presigs, got %d", ownerCount, len(presigs))
	}

	s.referee.mu.Lock()
	if s.referee.presigns[matchID] == nil {
		s.referee.presigns[matchID] = make(map[int32]map[string]*refereePreSignCtx)
	}
	if s.referee.presigns[matchID][branch] == nil {
		s.referee.presigns[matchID][branch] = make(map[string]*refereePreSignCtx)
	}
	s.referee.mu.Unlock()

	for _, ps := range presigs {
		in, ok := inputs[ps.InputId]
		if !ok {
			return fmt.Errorf("unknown input %s", ps.InputId)
		}
		if in.OwnerUID != uid {
			return fmt.Errorf("input %s not owned by caller", ps.InputId)
		}
		rb, err := hex.DecodeString(ps.RPrimeCompactHex)
		if err != nil || len(rb) != 33 {
			return fmt.Errorf("invalid r' for %s", ps.InputId)
		}
		sb, err := hex.DecodeString(ps.SPrimeHex)
		if err != nil || len(sb) != 32 {
			return fmt.Errorf("invalid s' for %s", ps.InputId)
		}
		tb, err := hex.DecodeString(in.AdaptorPointHex)
		if err != nil || len(tb) != 33 {
			return fmt.Errorf("invalid adaptor for %s", ps.InputId)
		}

		ctx := &refereePreSignCtx{
			InputID:            in.InputID,
			AmountAtoms:        in.AmountAtoms,
			RedeemScriptHex:    in.RedeemScriptHex,
			DraftHex:           draft.DraftHex,
			SighashHex:         in.SighashHex,
			TCompressed:        tb,
			RPrimeCompressed:   rb,
			SPrime32:           sb,
			InputIndex:         in.InputIndex,
			Branch:             branch,
			SeatIndex:          in.SeatIndex,
			OwnerUID:           uid,
			WinnerCandidateUID: draft.WinnerUID,
		}

		s.referee.mu.Lock()
		s.referee.presigns[matchID][branch][in.InputID] = ctx
		s.referee.mu.Unlock()
	}

	return nil
}

// pkScriptAndAddrFromRedeem builds a P2SH pkScript (hex) and human-readable
// address from a raw redeem script for the configured network.
func pkScriptAndAddrFromRedeem(redeem []byte, params stdaddr.AddressParams) (string, string, error) {
	a, err := stdaddr.NewAddressScriptHash(0, redeem, params)
	if err != nil {
		return "", "", err
	}
	_, pk := a.PaymentScript()
	return hex.EncodeToString(pk), a.String(), nil
}

// buildPerDepositorRedeemScript mirrors the Pong helper: winner branch
// pays to the provided compressed pubkey, else CSV timeout with same key.
func buildPerDepositorRedeemScript(comp33 []byte, csvBlocks uint32) ([]byte, error) {
	if len(comp33) != 33 {
		return nil, fmt.Errorf("need 33-byte compressed pubkey")
	}
	b := txscript.NewScriptBuilder()
	b.AddOp(txscript.OP_IF).
		AddData(comp33).
		AddInt64(2). // schnorr-secp256k1
		AddOp(txscript.OP_CHECKSIGALTVERIFY).
		AddOp(txscript.OP_TRUE).
		AddOp(txscript.OP_ELSE).
		AddInt64(int64(csvBlocks)).
		AddOp(txscript.OP_CHECKSEQUENCEVERIFY).
		AddOp(txscript.OP_DROP).
		AddData(comp33).
		AddInt64(2). // schnorr-secp256k1
		AddOp(txscript.OP_CHECKSIGALTVERIFY).
		AddOp(txscript.OP_TRUE).
		AddOp(txscript.OP_ENDIF)
	return b.Script()
}

func newEscrowID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "escrow-unknown"
	}
	return "e" + hex.EncodeToString(b[:])
}

// enforceSingleFunding ensures exactly one UTXO with expected amount is bound.
func enforceSingleFunding(es *refereeEscrowSession) error {
	if es == nil {
		return fmt.Errorf("nil escrow")
	}
	es.mu.Lock()
	defer es.mu.Unlock()
	if es.LatestFunding.UTXOCount != 1 || len(es.LatestFunding.UTXOs) != 1 {
		return fmt.Errorf("expected exactly one funding UTXO, have %d", es.LatestFunding.UTXOCount)
	}
	u := es.LatestFunding.UTXOs[0]
	if u.Value != es.AmountAtoms {
		return fmt.Errorf("funding amount mismatch: have %d want %d", u.Value, es.AmountAtoms)
	}
	es.BoundUTXO = cloneUTXO(u)
	return nil
}

// ensureBoundFunding asserts the escrow is funded with a single UTXO matching
// the expected amount and returns the bound UTXO.
func ensureBoundFunding(es *refereeEscrowSession) (*chainwatcher.EscrowUTXO, error) {
	if es == nil {
		return nil, fmt.Errorf("nil escrow")
	}
	es.mu.RLock()
	bound := cloneUTXO(es.BoundUTXO)
	amt := es.AmountAtoms
	es.mu.RUnlock()
	if bound != nil && bound.Value == amt {
		return bound, nil
	}
	if err := enforceSingleFunding(es); err != nil {
		return nil, err
	}
	es.mu.RLock()
	defer es.mu.RUnlock()
	return cloneUTXO(es.BoundUTXO), nil
}

func shortIDFromString(s string) (zkidentity.ShortID, error) {
	var out zkidentity.ShortID
	return out, out.FromString(s)
}

// zeroShortID is a helper zero value.
var zeroShortID zkidentity.ShortID

// deriveAdaptorGamma deterministically derives a branch adaptor secret and point.
func deriveAdaptorGamma(matchID string, branch int32, secret string) (gammaHex string, TCompHex string) {
	h := blake256.New()
	h.Write([]byte("Adaptor/PokerWTA/v1"))
	h.Write([]byte(matchID))
	h.Write([]byte{byte(branch)})
	h.Write([]byte(secret))
	sum := h.Sum(nil)

	var sc secp256k1.ModNScalar
	sc.SetByteSlice(sum)
	if sc.IsZero() {
		var one secp256k1.ModNScalar
		one.SetInt(1)
		sc.Add(&one)
	}
	gamma := sc.Bytes()
	priv := secp256k1.PrivKeyFromBytes(gamma[:])
	T := priv.PubKey()
	comp := T.SerializeCompressed()
	// normalize to even-Y
	if len(comp) == 33 && comp[0] == 0x03 {
		var neg secp256k1.ModNScalar
		neg.NegateVal(&sc)
		sc = neg
		gamma = sc.Bytes()
		priv = secp256k1.PrivKeyFromBytes(gamma[:])
		T = priv.PubKey()
		comp = T.SerializeCompressed()
	}
	return hex.EncodeToString(gamma[:]), hex.EncodeToString(comp)
}

// decodeMsgTx is a small helper for tests to parse raw tx bytes.
func decodeMsgTx(raw []byte) (*wire.MsgTx, error) {
	var tx wire.MsgTx
	if err := tx.Deserialize(bytes.NewReader(raw)); err != nil {
		return nil, err
	}
	return &tx, nil
}

func signedMessageDigest(msg string) ([]byte, error) {
	var buf bytes.Buffer
	const header = "Decred Signed Message:\n"
	if err := wire.WriteVarString(&buf, 0, header); err != nil {
		return nil, err
	}
	if err := wire.WriteVarString(&buf, 0, msg); err != nil {
		return nil, err
	}
	return chainhash.HashB(buf.Bytes()), nil
}

// SetPayoutAddress verifies a signed code and binds the payout address to the
// current session/user. Token is passed via metadata, not the request body.

// SetPayoutAddress verifies a signed nonce and persists the payout address for
// the authenticated user/session.
func (s *Server) SetPayoutAddress(ctx context.Context, req *pokerrpc.SetPayoutAddressRequest) (*pokerrpc.SetPayoutAddressResponse, error) {
	if s.auth == nil {
		s.auth = newAuthState(s.db)
	}

	token := strings.TrimSpace(req.GetToken())
	if token == "" {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if vals := md.Get("token"); len(vals) > 0 {
				token = strings.TrimSpace(vals[0])
			}
		}
	}
	if token == "" {
		return &pokerrpc.SetPayoutAddressResponse{Ok: false, Error: "token required"}, nil
	}

	addrStr := strings.TrimSpace(req.GetAddress())
	sigB64 := strings.TrimSpace(req.GetSignature())
	code := strings.TrimSpace(req.GetCode())
	if addrStr == "" || sigB64 == "" || code == "" {
		return &pokerrpc.SetPayoutAddressResponse{Ok: false, Error: "address, signature, and code are required"}, nil
	}

	sess, ok := s.sessionForToken(token)
	if !ok {
		return &pokerrpc.SetPayoutAddressResponse{Ok: false, Error: "invalid or expired session"}, nil
	}

	params := s.chainParams
	if params == nil {
		params = chaincfg.TestNet3Params()
	}

	nonceMeta, ok := s.auth.ConsumeNonce(code)
	if !ok {
		return &pokerrpc.SetPayoutAddressResponse{Ok: false, Error: "invalid or expired code"}, nil
	}
	if nonceMeta.userID != nil && *nonceMeta.userID != sess.userID {
		return &pokerrpc.SetPayoutAddressResponse{Ok: false, Error: "code does not match user id"}, nil
	}

	addr, err := stdaddr.DecodeAddress(addrStr, params)
	if err != nil {
		return &pokerrpc.SetPayoutAddressResponse{Ok: false, Error: fmt.Sprintf("invalid address: %v", err)}, nil
	}

	sig, err := base64.StdEncoding.DecodeString(sigB64)
	if err != nil {
		return &pokerrpc.SetPayoutAddressResponse{Ok: false, Error: fmt.Sprintf("invalid signature encoding: %v", err)}, nil
	}
	digest, err := signedMessageDigest(code)
	if err != nil {
		return &pokerrpc.SetPayoutAddressResponse{Ok: false, Error: fmt.Sprintf("failed to build signed message payload: %v", err)}, nil
	}
	pub, _, err := ecdsa.RecoverCompact(sig, digest)
	if err != nil {
		return &pokerrpc.SetPayoutAddressResponse{Ok: false, Error: fmt.Sprintf("failed to recover pubkey: %v", err)}, nil
	}
	got, err := stdaddr.NewAddressPubKeyHashEcdsaSecp256k1V0(stdaddr.Hash160(pub.SerializeCompressed()), params)
	if err != nil {
		return &pokerrpc.SetPayoutAddressResponse{Ok: false, Error: fmt.Sprintf("failed to compute address: %v", err)}, nil
	}
	if got.String() != addr.String() {
		return &pokerrpc.SetPayoutAddressResponse{Ok: false, Error: "address does not match recovered signature"}, nil
	}

	s.auth.mu.Lock()
	s.auth.uidToPayoutAddr[sess.userID] = addrStr
	if meta, ok := s.auth.sessions[token]; ok {
		meta.payoutAddr = addrStr
		s.auth.sessions[token] = meta
	}
	if !ok {
		s.auth.mu.Unlock()
		return &pokerrpc.SetPayoutAddressResponse{Ok: false, Error: "session not found"}, nil
	}
	s.auth.mu.Unlock()

	// Persist to DB for future reference.
	if err := s.db.UpsertAuthUser(ctx, sess.nickname, sess.userID.String()); err != nil {
		s.log.Warnf("failed to upsert auth user when setting payout: %v", err)
	}
	if err := s.db.UpdateAuthUserPayoutAddress(ctx, sess.userID.String(), addrStr); err != nil {
		s.log.Warnf("failed to update payout address when setting payout: %v", err)
	}
	if err := s.db.UpdateAuthUserLastLogin(ctx, sess.userID.String()); err != nil {
		s.log.Warnf("failed to update last login on payout set: %v", err)
	}

	return &pokerrpc.SetPayoutAddressResponse{
		Ok:      true,
		Address: addrStr,
	}, nil
}
