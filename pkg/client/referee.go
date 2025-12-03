package client

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/decred/dcrd/crypto/blake256"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/slog"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	"google.golang.org/grpc/metadata"
)

var schnorrV0ExtraTag = func() [32]byte {
	const tagHex = "0b75f97b60e8a5762876c004829ee9b926fa6f0d2eeaec3a4fd1446a768331cb"
	b, _ := hex.DecodeString(tagHex)
	var out [32]byte
	copy(out[:], b)
	return out
}()

// RefereeClient wraps PokerReferee RPCs with presign helpers.
type RefereeClient struct {
	rc    pokerrpc.PokerRefereeClient
	log   slog.Logger
	token string
}

// NewRefereeClient constructs a referee client using an existing gRPC conn.
func NewRefereeClient(conn pokerrpc.PokerRefereeClient, log slog.Logger, token string) *RefereeClient {
	return &RefereeClient{rc: conn, log: log, token: token}
}

// Referee returns a RefereeClient bound to this PokerClient's connection/token.
func (pc *PokerClient) Referee(token string) *RefereeClient {
	return NewRefereeClient(pokerrpc.NewPokerRefereeClient(pc.conn), pc.log, token)
}

// SetPayoutAddress verifies a signed code and binds the payout address to the current session/user.
func (c *RefereeClient) SetPayoutAddress(ctx context.Context, address, signature, code string) (string, error) {
	if c.token != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "token", c.token)
	}
	req := &pokerrpc.SetPayoutAddressRequest{
		Token:     c.token,
		Address:   address,
		Signature: signature,
		Code:      code,
	}
	resp, err := c.rc.SetPayoutAddress(ctx, req)
	if err != nil {
		return "", err
	}
	if !resp.Ok {
		return "", fmt.Errorf("set payout address failed: %s", resp.Error)
	}
	return resp.Address, nil
}

func (c *RefereeClient) GetEscrowStatus(ctx context.Context, escrowID string) (*pokerrpc.GetEscrowStatusResponse, error) {
	if c.token != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "token", c.token)
	}
	return c.rc.GetEscrowStatus(ctx, &pokerrpc.GetEscrowStatusRequest{
		EscrowId: escrowID,
	})
}

// OpenEscrow opens a Schnorr escrow for a table/session/seat using the caller's token.
func (c *RefereeClient) OpenEscrow(ctx context.Context, amountAtoms uint64, csvBlocks uint32, compPubkey []byte) (*pokerrpc.OpenEscrowResponse, error) {
	if c.token != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "token", c.token)
	}
	req := &pokerrpc.OpenEscrowRequest{
		AmountAtoms: amountAtoms,
		CsvBlocks:   csvBlocks,
		CompPubkey:  compPubkey,
	}
	return c.rc.OpenEscrow(ctx, req)
}

// BindEscrow binds escrow funding (txid:vout) to a table/session seat.
func (c *RefereeClient) BindEscrow(ctx context.Context, tableID, sessionID, matchID string, seatIndex uint32, outpoint string, redeemScriptHex string, csvBlocks uint32) (*pokerrpc.BindEscrowResponse, error) {
	if c.token != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "token", c.token)
	}
	req := &pokerrpc.BindEscrowRequest{
		TableId:         tableID,
		SessionId:       sessionID,
		MatchId:         matchID,
		SeatIndex:       seatIndex,
		Outpoint:        outpoint,
		RedeemScriptHex: redeemScriptHex,
		CsvBlocks:       csvBlocks,
	}
	return c.rc.BindEscrow(ctx, req)
}

// StartPresign runs the SettlementStream presign flow for a match/escrow.
// xPrivHex is the session private scalar (hex) corresponding to compPubkey.
func (c *RefereeClient) StartPresign(ctx context.Context, matchID, tableID, sessionID string, seatIndex uint32, escrowID string, compPubkey []byte, xPrivHex string) error {
	if c.token != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "token", c.token)
	}
	stream, err := c.rc.SettlementStream(ctx)
	if err != nil {
		return err
	}

	hello := &pokerrpc.SettlementHello{
		MatchId:    matchID,
		TableId:    tableID,
		SessionId:  sessionID,
		SeatIndex:  seatIndex,
		EscrowId:   escrowID,
		CompPubkey: compPubkey,
		Token:      c.token,
	}
	if err := stream.Send(&pokerrpc.SettlementStreamMessage{Msg: &pokerrpc.SettlementStreamMessage_Hello{Hello: hello}}); err != nil {
		return err
	}

	// Track expected branches and which have been acknowledged by VerifyOk.
	// The server sends NeedPreSigs once per branch before any VerifyOk,
	// so we discover the full branch set from NeedPreSigs messages.
	branches := make(map[int32]bool) // branch -> acked

	for {
		msg, err := stream.Recv()
		if err != nil {
			return err
		}
		if need := msg.GetNeedPreSigs(); need != nil {
			if _, ok := branches[need.Branch]; !ok {
				branches[need.Branch] = false
			}
			pres, err := buildPresigs(xPrivHex, need)
			if err != nil {
				return err
			}
			resp := &pokerrpc.ProvidePreSigs{
				MatchId: need.MatchId,
				Branch:  need.Branch,
				Presigs: pres,
			}
			if err := stream.Send(&pokerrpc.SettlementStreamMessage{Msg: &pokerrpc.SettlementStreamMessage_ProvidePreSigs{ProvidePreSigs: resp}}); err != nil {
				return err
			}
			continue
		}
		if errMsg := msg.GetError(); errMsg != nil {
			return fmt.Errorf("referee error: %s", errMsg.Error)
		}
		if ok := msg.GetVerifyOk(); ok != nil {
			// Mark this branch as acknowledged.
			if _, exists := branches[ok.Branch]; !exists {
				branches[ok.Branch] = true
			} else {
				branches[ok.Branch] = true
			}

			// Check if all known branches have been acknowledged.
			allAcked := len(branches) > 0
			for _, acked := range branches {
				if !acked {
					allAcked = false
					break
				}
			}
			if allAcked {
				// Presign finished for all branches for this seat.
				// Close the send direction to signal EOF to the server.
				_ = stream.CloseSend()
				return nil
			}
		}
	}
}

// GetFinalizeBundle fetches the winning draft + presigs for a branch.
func (c *RefereeClient) GetFinalizeBundle(ctx context.Context, matchID string, winnerSeat int32) (*pokerrpc.GetFinalizeBundleResponse, error) {
	if c.token != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "token", c.token)
	}
	return c.rc.GetFinalizeBundle(ctx, &pokerrpc.GetFinalizeBundleRequest{
		MatchId:    matchID,
		WinnerSeat: winnerSeat,
	})
}

func buildPresigs(xPrivHex string, need *pokerrpc.NeedPreSigs) ([]*pokerrpc.PreSignature, error) {
	privB, err := hex.DecodeString(xPrivHex)
	if err != nil || len(privB) == 0 {
		return nil, fmt.Errorf("bad x priv")
	}
	var out []*pokerrpc.PreSignature
	for _, in := range need.Inputs {
		if len(in.SighashHex) != 64 {
			return nil, fmt.Errorf("bad sighash for %s", in.InputId)
		}
		if len(in.AdaptorPointHex) != 66 {
			return nil, fmt.Errorf("bad adaptor point for %s", in.InputId)
		}
		rComp, sPrime, err := computePreSig(privB, in.SighashHex, in.AdaptorPointHex)
		if err != nil {
			return nil, fmt.Errorf("compute presig %s: %w", in.InputId, err)
		}
		out = append(out, &pokerrpc.PreSignature{
			InputId:          in.InputId,
			RPrimeCompactHex: rComp,
			SPrimeHex:        sPrime,
		})
	}
	return out, nil
}

// computePreSig derives adaptor pre-signature for (x, m, T).
func computePreSig(xb []byte, mHex, TCompHex string) (rCompHex string, sPrimeHex string, err error) {
	mb, err := hex.DecodeString(mHex)
	if err != nil || len(mb) != 32 {
		return "", "", fmt.Errorf("bad m")
	}
	Tb, err := hex.DecodeString(TCompHex)
	if err != nil {
		return "", "", err
	}

	var x secp256k1.ModNScalar
	if overflow := x.SetByteSlice(xb); overflow || x.IsZero() {
		return "", "", fmt.Errorf("bad x scalar")
	}
	Tpub, err := secp256k1.ParsePubKey(Tb)
	if err != nil {
		return "", "", err
	}

	extra := blake256.Sum256(append(schnorrV0ExtraTag[:], Tb...))

	for iter := uint32(0); ; iter++ {
		k := secp256k1.NonceRFC6979(xb, mb, extra[:], nil, iter)
		if k == nil || k.IsZero() {
			continue
		}

		var R secp256k1.JacobianPoint
		secp256k1.ScalarBaseMultNonConst(k, &R)

		var tJac secp256k1.JacobianPoint
		Tpub.AsJacobian(&tJac)
		secp256k1.AddNonConst(&R, &tJac, &R)
		if R.Z.IsZero() {
			continue
		}
		R.ToAffine()
		Rpub := secp256k1.NewPublicKey(&R.X, &R.Y)
		rComp := Rpub.SerializeCompressed()
		if len(rComp) != 33 || rComp[0] != 0x02 {
			continue
		}

		eBytes := hashSchnorr(rComp[1:], mb)
		var e secp256k1.ModNScalar
		if overflow := e.SetByteSlice(eBytes[:]); overflow || e.IsZero() {
			continue
		}

		var ex secp256k1.ModNScalar
		ex.Set(&e)
		ex.Mul(&x) // ex = e * x

		var s secp256k1.ModNScalar
		s.Set(k)    // s = k
		ex.Negate() // -e*x
		s.Add(&ex)  // s' = k - e*x
		if s.IsZero() {
			continue
		}
		sb := s.Bytes()
		return hex.EncodeToString(rComp), hex.EncodeToString(sb[:]), nil
	}
}

// hashSchnorr computes e = H(rx || m) mod n.
func hashSchnorr(rx []byte, m []byte) [32]byte {
	h := blake256.New()
	h.Write(rx)
	h.Write(m)
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

// VerifyPreSig can be used by tests to validate stored presigs.
func VerifyPreSig(ctx *pokerrpc.NeedPreSigs, compPubkey []byte, ps *pokerrpc.PreSignature) error {
	if ctx == nil || ps == nil {
		return fmt.Errorf("nil args")
	}
	var target *pokerrpc.NeedPreSigsInput
	for _, in := range ctx.Inputs {
		if in.InputId == ps.InputId {
			target = in
			break
		}
	}
	if target == nil {
		return fmt.Errorf("input not found in ctx")
	}
	rb, err := hex.DecodeString(ps.RPrimeCompactHex)
	if err != nil || len(rb) != 33 || rb[0] != 0x02 {
		return fmt.Errorf("bad R'")
	}
	sb, err := hex.DecodeString(ps.SPrimeHex)
	if err != nil || len(sb) != 32 {
		return fmt.Errorf("bad s'")
	}
	tb, err := hex.DecodeString(target.AdaptorPointHex)
	if err != nil || len(tb) != 33 {
		return fmt.Errorf("bad T")
	}
	T, err := secp256k1.ParsePubKey(tb)
	if err != nil {
		return fmt.Errorf("parse T: %w", err)
	}
	R, err := secp256k1.ParsePubKey(rb)
	if err != nil {
		return fmt.Errorf("parse R': %w", err)
	}

	// Recompute e
	mBytes, err := hex.DecodeString(target.SighashHex)
	if err != nil || len(mBytes) != 32 {
		return fmt.Errorf("bad m")
	}
	e := hashSchnorr(R.X().Bytes(), mBytes)
	var es, s secp256k1.ModNScalar
	es.SetByteSlice(e[:])
	s.SetByteSlice(sb)

	// Check s'G + eX + T == R'
	X, err := secp256k1.ParsePubKey(compPubkey)
	if err != nil {
		return fmt.Errorf("parse comp pubkey: %w", err)
	}
	var sG secp256k1.JacobianPoint
	secp256k1.ScalarBaseMultNonConst(&s, &sG)

	var xJac secp256k1.JacobianPoint
	X.AsJacobian(&xJac)
	var exP secp256k1.JacobianPoint
	secp256k1.ScalarMultNonConst(&es, &xJac, &exP)

	secp256k1.AddNonConst(&sG, &exP, &sG)

	var tJac secp256k1.JacobianPoint
	T.AsJacobian(&tJac)
	secp256k1.AddNonConst(&sG, &tJac, &sG)

	sG.ToAffine()
	L := secp256k1.NewPublicKey(&sG.X, &sG.Y)

	if !L.IsEqual(R) {
		return fmt.Errorf("presig verification failed")
	}
	return nil
}
