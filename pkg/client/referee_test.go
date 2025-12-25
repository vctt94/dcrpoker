package client

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/decred/dcrd/chaincfg/chainhash"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/txscript/v4"
	"github.com/decred/dcrd/wire"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
)

func buildTestNeedPreSigs(t *testing.T) *pokerrpc.NeedPreSigs {
	t.Helper()

	tx := wire.NewMsgTx()
	tx.Version = 3

	var h chainhash.Hash
	copy(h[:], bytes.Repeat([]byte{0x01}, chainhash.HashSize))
	outpoint := wire.NewOutPoint(&h, 0, wire.TxTreeRegular)
	tx.AddTxIn(wire.NewTxIn(outpoint, 0, nil))

	script, err := txscript.NewScriptBuilder().AddOp(txscript.OP_TRUE).Script()
	if err != nil {
		t.Fatalf("build script: %v", err)
	}
	tx.AddTxOut(wire.NewTxOut(1000, script))

	sighash, err := txscript.CalcSignatureHash(script, txscript.SigHashAll, tx, 0, nil)
	if err != nil {
		t.Fatalf("calc sighash: %v", err)
	}

	var buf bytes.Buffer
	if err := tx.Serialize(&buf); err != nil {
		t.Fatalf("serialize tx: %v", err)
	}

	priv := secp256k1.PrivKeyFromBytes([]byte{0x02})
	adaptorHex := hex.EncodeToString(priv.PubKey().SerializeCompressed())

	return &pokerrpc.NeedPreSigs{
		MatchId:    "m1",
		Branch:     0,
		DraftTxHex: hex.EncodeToString(buf.Bytes()),
		Inputs: []*pokerrpc.NeedPreSigsInput{
			{
				InputId:         h.String() + ":0",
				RedeemScriptHex: hex.EncodeToString(script),
				SighashHex:      hex.EncodeToString(sighash),
				AdaptorPointHex: adaptorHex,
				InputIndex:      0,
				AmountAtoms:     1000,
			},
		},
	}
}

func TestValidateNeedPreSigsOK(t *testing.T) {
	need := buildTestNeedPreSigs(t)
	if err := validateNeedPreSigs(need); err != nil {
		t.Fatalf("expected valid need presigs, got %v", err)
	}
}

func TestValidateNeedPreSigsRejectsTamper(t *testing.T) {
	need := buildTestNeedPreSigs(t)

	need.Inputs[0].SighashHex = "00" + need.Inputs[0].SighashHex[2:]
	if err := validateNeedPreSigs(need); err == nil {
		t.Fatalf("expected mismatch error after sighash tamper")
	}

	need = buildTestNeedPreSigs(t)
	need.Inputs[0].InputId = "00" + need.Inputs[0].InputId[2:]
	if err := validateNeedPreSigs(need); err == nil {
		t.Fatalf("expected input mismatch after txid tamper")
	}
}

func TestBuildVerifyOkUsesValidation(t *testing.T) {
	need := buildTestNeedPreSigs(t)
	if _, err := BuildVerifyOk("02", need); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Tamper sighash to trigger same validation failure.
	need.Inputs[0].SighashHex = "00" + need.Inputs[0].SighashHex[2:]
	if _, err := BuildVerifyOk("02", need); err == nil {
		t.Fatalf("expected validation error on tampered sighash")
	}
}

func TestVerifyPreSigRejectsAlteredPresig(t *testing.T) {
	need := buildTestNeedPreSigs(t)

	x := bytes.Repeat([]byte{0x03}, 32)
	xPrivHex := hex.EncodeToString(x)
	compPub := secp256k1.PrivKeyFromBytes(x).PubKey().SerializeCompressed()

	presigs, err := BuildVerifyOk(xPrivHex, need)
	if err != nil {
		t.Fatalf("build presigs: %v", err)
	}
	if len(presigs) != 1 {
		t.Fatalf("unexpected presig count %d", len(presigs))
	}

	// Happy path verify.
	if err := VerifyPreSig(need, compPub, presigs[0]); err != nil {
		t.Fatalf("verify presig failed: %v", err)
	}

	// Tamper s' to simulate server altering stored presig.
	presigs[0].SPrimeHex = "00" + presigs[0].SPrimeHex[2:]
	if err := VerifyPreSig(need, compPub, presigs[0]); err == nil {
		t.Fatalf("expected verification failure on altered presig")
	}
}
