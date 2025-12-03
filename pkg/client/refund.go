package client

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/decred/dcrd/chaincfg/chainhash"
	"github.com/decred/dcrd/chaincfg/v3"
	"github.com/decred/dcrd/crypto/blake256"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrutil/v4"
	"github.com/decred/dcrd/txscript/v4"
	"github.com/decred/dcrd/txscript/v4/stdaddr"
	"github.com/decred/dcrd/wire"
)

// signSchnorrV0 deterministically signs the provided 32-byte message digest using
// EC-Schnorr-DCRv0 with RFC6979 and enforcing an even-Y nonce point. Returns
// the 65-byte signature: r_x (32B) || s (32B) || 0x01 (SigHashAll).
func signSchnorrV0(privHex, mHex string) ([]byte, error) {
	xb, err := hex.DecodeString(strings.TrimSpace(privHex))
	if err != nil {
		return nil, fmt.Errorf("bad privkey hex: %w", err)
	}
	if len(xb) != 32 {
		return nil, fmt.Errorf("privkey must be 32 bytes")
	}
	mb, err := hex.DecodeString(strings.TrimSpace(mHex))
	if err != nil || len(mb) != 32 {
		return nil, fmt.Errorf("bad m (need 32B hex)")
	}

	var x secp256k1.ModNScalar
	if overflow := x.SetByteSlice(xb); overflow || x.IsZero() {
		return nil, fmt.Errorf("invalid private key scalar")
	}

	extra := blake256.Sum256(schnorrV0ExtraTag[:])
	var version []byte
	for iter := uint32(0); ; iter++ {
		k := secp256k1.NonceRFC6979(xb, mb, extra[:], version, iter)
		if k == nil || k.IsZero() {
			continue
		}
		kb := k.Bytes()
		R := secp256k1.PrivKeyFromBytes(kb[:]).PubKey()
		cp := R.SerializeCompressed()
		if len(cp) != 33 || cp[0] != 0x02 { // enforce even-Y
			continue
		}
		var r32 [32]byte
		copy(r32[:], cp[1:33])
		h := blake256.Sum256(append(r32[:], mb...))
		var e secp256k1.ModNScalar
		if overflow := e.SetByteSlice(h[:]); overflow { // e >= n -> retry
			continue
		}
		var ex, negex, s secp256k1.ModNScalar
		ex.Set(&e)
		ex.Mul(&x)
		negex.NegateVal(&ex)
		s.Set(k)
		s.Add(&negex)

		sBytes := s.Bytes()
		sig := make([]byte, 0, 65)
		sig = append(sig, r32[:]...)
		sig = append(sig, sBytes[:]...)
		sig = append(sig, byte(txscript.SigHashAll))
		return sig, nil
	}
}

// BuildCSVRefundTx constructs a raw transaction hex that spends a single escrow
// deposit UTXO via the CSV timeout path back to a destination address.
//
// Parameters:
//   - privHex:      32-byte hex of the depositor's private key (controls A_c)
//   - utxoTxid:     funding txid
//   - utxoVout:     funding vout index
//   - utxoValue:    amount in atoms
//   - redeemHex:    redeem script hex for the deposit P2SH
//   - destAddr:     Decred address to receive refund
//   - feeAtoms:     fee to subtract in atoms (e.g., 20000)
//   - csvBlocks:    relative timelock blocks to satisfy (sequence >= csvBlocks)
//
// It returns the serialized transaction hex. The caller is responsible for
// broadcasting it once CSV has matured on-chain.
func BuildCSVRefundTx(privHex, utxoTxid string, utxoVout uint32, utxoValue uint64, redeemHex, destAddr string, feeAtoms uint64, csvBlocks uint32) (string, error) {
	// Decode destination address across known networks and get payment script.
	paramsList := []*chaincfg.Params{
		chaincfg.TestNet3Params(),
		chaincfg.MainNetParams(),
		chaincfg.SimNetParams(),
		chaincfg.RegNetParams(),
	}
	var pkScript []byte
	var lastErr error
	for _, p := range paramsList {
		addr, err := stdaddr.DecodeAddress(strings.TrimSpace(destAddr), p)
		if err != nil {
			lastErr = err
			continue
		}
		_, pk := addr.PaymentScript()
		pkScript = pk
		lastErr = nil
		break
	}
	if pkScript == nil {
		return "", fmt.Errorf("bad dest address: %v", lastErr)
	}

	if feeAtoms == 0 {
		feeAtoms = 20000
	}
	payout := int64(utxoValue) - int64(feeAtoms)
	if payout <= 0 {
		return "", fmt.Errorf("fee %d exceeds input %d", feeAtoms, utxoValue)
	}

	// Build tx: version 3, one input, one output.
	var h chainhash.Hash
	if err := chainhash.Decode(&h, strings.TrimSpace(utxoTxid)); err != nil {
		return "", fmt.Errorf("bad txid: %w", err)
	}
	tx := wire.NewMsgTx()
	tx.Version = 3
	tx.AddTxIn(&wire.TxIn{PreviousOutPoint: wire.OutPoint{Hash: h, Index: utxoVout}, ValueIn: int64(utxoValue)})
	// Set sequence to satisfy CSV.
	tx.TxIn[0].Sequence = csvBlocks
	tx.AddTxOut(&wire.TxOut{Value: payout, PkScript: pkScript})

	// Compute sighash against redeem script.
	redeem, err := hex.DecodeString(strings.TrimSpace(redeemHex))
	if err != nil {
		return "", fmt.Errorf("bad redeem hex: %w", err)
	}
	mBytes, err := txscript.CalcSignatureHash(redeem, txscript.SigHashAll, tx, 0, nil)
	if err != nil || len(mBytes) != 32 {
		return "", fmt.Errorf("sighash failed: %v", err)
	}
	mHex := hex.EncodeToString(mBytes)

	// Sign Schnorr v0 and build CSV branch scriptSig.
	sig65, err := signSchnorrV0(privHex, mHex)
	if err != nil {
		return "", err
	}
	sb := txscript.NewScriptBuilder()
	sb.AddData(sig65)
	sb.AddOp(txscript.OP_0) // select CSV branch (ELSE)
	sb.AddData(redeem)
	sigScript, err := sb.Script()
	if err != nil {
		return "", fmt.Errorf("build scriptSig: %w", err)
	}
	tx.TxIn[0].SignatureScript = sigScript

	// Optional local VM verification (P2SH wrapper):
	sh := dcrutil.Hash160(redeem)
	p2sh, err := txscript.NewScriptBuilder().AddOp(txscript.OP_HASH160).AddData(sh).AddOp(txscript.OP_EQUAL).Script()
	if err != nil {
		return "", fmt.Errorf("build pkScript: %w", err)
	}
	vm, err := txscript.NewEngine(p2sh, tx, 0, 0, 0, nil)
	if err != nil {
		return "", fmt.Errorf("engine init: %w", err)
	}
	if err := vm.Execute(); err != nil {
		return "", fmt.Errorf("local VM verify failed: %w", err)
	}

	var out bytes.Buffer
	_ = tx.Serialize(&out)
	return hex.EncodeToString(out.Bytes()), nil
}

// RefundTxResult is the JSON-serializable result of a refund build attempt.
type RefundTxResult struct {
	RefundTxHex string `json:"refund_tx_hex"`
	UTXOTxid    string `json:"utxo_txid"`
	UTXOVout    uint32 `json:"utxo_vout"`
	UTXOValue   uint64 `json:"utxo_value"`
	RedeemHex   string `json:"redeem_hex"`
	CSVBlocks   uint32 `json:"csv_blocks"`
	CanRefund   bool   `json:"can_refund"`
	Reason      string `json:"reason,omitempty"`
}

// loadEscrowInfoForRefund attempts to load EscrowInfo for the given escrow ID
// from the history_session directory. It supports both cache-style files
// (escrow_<id>.json) and archived session files (<match>_<id>.json) that wrap
// EscrowInfo under an "escrow_info" key.
func (pc *PokerClient) loadEscrowInfoForRefund(escrowID string) (*EscrowInfo, error) {
	escrowID = strings.TrimSpace(escrowID)
	if escrowID == "" {
		return nil, fmt.Errorf("escrow_id required")
	}
	dir := pc.historyDir()
	if dir == "" {
		return nil, fmt.Errorf("no data directory configured")
	}

	// First try the direct cache path (escrow_<id>.json).
	candidate := filepath.Join(dir, fmt.Sprintf("escrow_%s.json", sanitizeForFile(escrowID)))
	if b, err := os.ReadFile(candidate); err == nil {
		var info EscrowInfo
		if err := json.Unmarshal(b, &info); err == nil && strings.TrimSpace(info.EscrowID) == escrowID {
			return &info, nil
		}
	}

	// Fallback: scan all files and look for a matching escrow_info wrapper or
	// top-level EscrowInfo entry.
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		// Try as SessionArchive first.
		var arch SessionArchive
		if err := json.Unmarshal(b, &arch); err == nil && arch.Escrow != nil {
			if strings.TrimSpace(arch.Escrow.EscrowID) == escrowID {
				return arch.Escrow, nil
			}
		}
		// Fallback to bare EscrowInfo.
		var info EscrowInfo
		if err := json.Unmarshal(b, &info); err == nil {
			if strings.TrimSpace(info.EscrowID) == escrowID {
				return &info, nil
			}
		}
	}
	return nil, fmt.Errorf("escrow %s not found in history", escrowID)
}

// RefundEscrow builds a CSV refund transaction for a historic escrow using
// only local data (seed-derived session key + cached escrow info).
func (pc *PokerClient) RefundEscrow(escrowID, destAddr string, feeAtoms uint64, csvBlocks uint32, utxoValue uint64) (*RefundTxResult, error) {
	if strings.TrimSpace(destAddr) == "" {
		return &RefundTxResult{
			CanRefund: false,
			Reason:    "destination address required",
		}, nil
	}

	info, err := pc.loadEscrowInfoForRefund(escrowID)
	if err != nil {
		return &RefundTxResult{
			CanRefund: false,
			Reason:    fmt.Sprintf("load escrow info: %v", err),
		}, nil
	}
	if strings.TrimSpace(info.FundingTxid) == "" {
		return &RefundTxResult{
			CanRefund: false,
			Reason:    "historic escrow missing funding txid",
		}, nil
	}
	if info.RedeemScriptHex == "" {
		return &RefundTxResult{
			CanRefund: false,
			Reason:    "historic escrow missing redeem script",
		}, nil
	}

	// Determine CSV blocks.
	if csvBlocks == 0 {
		csvBlocks = info.CSVBlocks
	}
	if csvBlocks == 0 {
		return &RefundTxResult{
			CanRefund: false,
			Reason:    "csv_blocks not set for escrow",
		}, nil
	}

	// Determine UTXO value (funded amount).
	value := info.FundedAmount
	if utxoValue > 0 {
		value = utxoValue
	}
	if value == 0 {
		return &RefundTxResult{
			CanRefund: false,
			Reason:    "funded amount not known for escrow",
		}, nil
	}

	// Derive session private key for this escrow from the stored key index.
	privHex, _, err := pc.DeriveSessionKeyAt(uint64(info.KeyIndex))
	if err != nil {
		return &RefundTxResult{
			CanRefund: false,
			Reason:    fmt.Sprintf("derive session key for escrow: %v", err),
		}, nil
	}

	txHex, err := BuildCSVRefundTx(
		privHex,
		info.FundingTxid,
		info.FundingVout,
		value,
		info.RedeemScriptHex,
		destAddr,
		feeAtoms,
		csvBlocks,
	)
	if err != nil {
		return &RefundTxResult{
			CanRefund: false,
			Reason:    fmt.Sprintf("build refund tx: %v", err),
		}, nil
	}

	return &RefundTxResult{
		RefundTxHex: txHex,
		UTXOTxid:    info.FundingTxid,
		UTXOVout:    info.FundingVout,
		UTXOValue:   value,
		RedeemHex:   info.RedeemScriptHex,
		CSVBlocks:   csvBlocks,
		CanRefund:   true,
	}, nil
}
