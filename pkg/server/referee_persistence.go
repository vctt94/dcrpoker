package server

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/companyzero/bisonrelay/zkidentity"
	"github.com/vctt94/pokerbisonrelay/pkg/chainwatcher"
	"github.com/vctt94/pokerbisonrelay/pkg/server/internal/db"
)

type persistedEscrowSession struct {
	EscrowID           string                     `json:"escrow_id"`
	OwnerUID           string                     `json:"owner_uid"`
	Token              string                     `json:"token,omitempty"`
	TableID            string                     `json:"table_id,omitempty"`
	SessionID          string                     `json:"session_id,omitempty"`
	SeatIndex          uint32                     `json:"seat_index"`
	CompPubkeyHex      string                     `json:"comp_pubkey_hex,omitempty"`
	PayoutAddr         string                     `json:"payout_addr,omitempty"`
	AmountAtoms        uint64                     `json:"amount_atoms"`
	CSVBlocks          uint32                     `json:"csv_blocks"`
	RedeemScriptHex    string                     `json:"redeem_script_hex,omitempty"`
	PkScriptHex        string                     `json:"pk_script_hex,omitempty"`
	LatestFunding      chainwatcher.DepositUpdate `json:"latest_funding"`
	FundingState       string                     `json:"funding_state,omitempty"`
	FundingStateReason string                     `json:"funding_state_reason,omitempty"`
	HadFunding         bool                       `json:"had_funding"`
	BoundUTXO          *chainwatcher.EscrowUTXO   `json:"bound_utxo,omitempty"`
}

type persistedPreSignCtx struct {
	InputID             string `json:"input_id"`
	AmountAtoms         uint64 `json:"amount_atoms"`
	RedeemScriptHex     string `json:"redeem_script_hex"`
	DraftHex            string `json:"draft_hex"`
	SighashHex          string `json:"sighash_hex"`
	TCompressedHex      string `json:"t_compressed_hex"`
	RPrimeCompressedHex string `json:"r_prime_compressed_hex"`
	SPrimeHex           string `json:"s_prime_hex"`
	InputIndex          uint32 `json:"input_index"`
	Branch              int32  `json:"branch"`
	SeatIndex           uint32 `json:"seat_index"`
	OwnerUID            string `json:"owner_uid"`
	WinnerCandidateUID  string `json:"winner_candidate_uid"`
}

type pendingSettlementState struct {
	TableID    string
	WinnerID   string
	WinnerSeat int32
}

func marshalEscrowSession(es *refereeEscrowSession) (db.RefereeEscrow, error) {
	es.mu.RLock()
	payload := persistedEscrowSession{
		EscrowID:           es.EscrowID,
		OwnerUID:           es.OwnerUID.String(),
		Token:              es.Token,
		TableID:            es.TableID,
		SessionID:          es.SessionID,
		SeatIndex:          es.SeatIndex,
		CompPubkeyHex:      hex.EncodeToString(es.CompPubkey),
		PayoutAddr:         es.PayoutAddr,
		AmountAtoms:        es.AmountAtoms,
		CSVBlocks:          es.CSVBlocks,
		RedeemScriptHex:    es.RedeemScriptHex,
		PkScriptHex:        es.PkScriptHex,
		LatestFunding:      cloneDepositUpdate(es.LatestFunding),
		FundingState:       es.fundingState,
		FundingStateReason: es.fundingStateReason,
		HadFunding:         es.HadFunding,
		BoundUTXO:          cloneUTXO(es.BoundUTXO),
	}
	es.mu.RUnlock()

	raw, err := json.Marshal(payload)
	if err != nil {
		return db.RefereeEscrow{}, err
	}
	return db.RefereeEscrow{
		EscrowID:  payload.EscrowID,
		UpdatedAt: time.Now(),
		Payload:   raw,
	}, nil
}

func unmarshalEscrowSession(row db.RefereeEscrow) (*refereeEscrowSession, error) {
	var payload persistedEscrowSession
	if err := json.Unmarshal(row.Payload, &payload); err != nil {
		return nil, err
	}

	var ownerUID zkidentity.ShortID
	if err := ownerUID.FromString(strings.TrimSpace(payload.OwnerUID)); err != nil {
		return nil, err
	}

	compPubkey, err := hex.DecodeString(strings.TrimSpace(payload.CompPubkeyHex))
	if err != nil {
		return nil, err
	}

	return &refereeEscrowSession{
		EscrowID:           strings.TrimSpace(payload.EscrowID),
		OwnerUID:           ownerUID,
		Token:              strings.TrimSpace(payload.Token),
		TableID:            strings.TrimSpace(payload.TableID),
		SessionID:          strings.TrimSpace(payload.SessionID),
		SeatIndex:          payload.SeatIndex,
		CompPubkey:         compPubkey,
		PayoutAddr:         strings.TrimSpace(payload.PayoutAddr),
		AmountAtoms:        payload.AmountAtoms,
		CSVBlocks:          payload.CSVBlocks,
		RedeemScriptHex:    strings.TrimSpace(payload.RedeemScriptHex),
		PkScriptHex:        strings.TrimSpace(payload.PkScriptHex),
		LatestFunding:      cloneDepositUpdate(payload.LatestFunding),
		fundingState:       strings.TrimSpace(payload.FundingState),
		fundingStateReason: strings.TrimSpace(payload.FundingStateReason),
		HadFunding:         payload.HadFunding,
		BoundUTXO:          cloneUTXO(payload.BoundUTXO),
	}, nil
}

func marshalPresignContext(matchID string, ctx *refereePreSignCtx) (db.RefereePresign, error) {
	payload := persistedPreSignCtx{
		InputID:             ctx.InputID,
		AmountAtoms:         ctx.AmountAtoms,
		RedeemScriptHex:     ctx.RedeemScriptHex,
		DraftHex:            ctx.DraftHex,
		SighashHex:          ctx.SighashHex,
		TCompressedHex:      hex.EncodeToString(ctx.TCompressed),
		RPrimeCompressedHex: hex.EncodeToString(ctx.RPrimeCompressed),
		SPrimeHex:           hex.EncodeToString(ctx.SPrime32),
		InputIndex:          ctx.InputIndex,
		Branch:              ctx.Branch,
		SeatIndex:           ctx.SeatIndex,
		OwnerUID:            ctx.OwnerUID.String(),
		WinnerCandidateUID:  ctx.WinnerCandidateUID.String(),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return db.RefereePresign{}, err
	}
	return db.RefereePresign{
		MatchID:   matchID,
		Branch:    ctx.Branch,
		InputID:   ctx.InputID,
		UpdatedAt: time.Now(),
		Payload:   raw,
	}, nil
}

func unmarshalPresignContext(row db.RefereePresign) (*refereePreSignCtx, error) {
	var payload persistedPreSignCtx
	if err := json.Unmarshal(row.Payload, &payload); err != nil {
		return nil, err
	}

	var ownerUID zkidentity.ShortID
	if err := ownerUID.FromString(strings.TrimSpace(payload.OwnerUID)); err != nil {
		return nil, err
	}
	var winnerUID zkidentity.ShortID
	if err := winnerUID.FromString(strings.TrimSpace(payload.WinnerCandidateUID)); err != nil {
		return nil, err
	}
	tCompressed, err := hex.DecodeString(strings.TrimSpace(payload.TCompressedHex))
	if err != nil {
		return nil, err
	}
	rPrime, err := hex.DecodeString(strings.TrimSpace(payload.RPrimeCompressedHex))
	if err != nil {
		return nil, err
	}
	sPrime, err := hex.DecodeString(strings.TrimSpace(payload.SPrimeHex))
	if err != nil {
		return nil, err
	}

	return &refereePreSignCtx{
		InputID:            strings.TrimSpace(payload.InputID),
		AmountAtoms:        payload.AmountAtoms,
		RedeemScriptHex:    strings.TrimSpace(payload.RedeemScriptHex),
		DraftHex:           strings.TrimSpace(payload.DraftHex),
		SighashHex:         strings.TrimSpace(payload.SighashHex),
		TCompressed:        tCompressed,
		RPrimeCompressed:   rPrime,
		SPrime32:           sPrime,
		InputIndex:         payload.InputIndex,
		Branch:             payload.Branch,
		SeatIndex:          payload.SeatIndex,
		OwnerUID:           ownerUID,
		WinnerCandidateUID: winnerUID,
	}, nil
}

func (s *Server) persistEscrowSession(es *refereeEscrowSession) {
	if s == nil || s.db == nil || es == nil {
		return
	}
	row, err := marshalEscrowSession(es)
	if err != nil {
		s.log.Warnf("failed to marshal escrow session %s: %v", es.EscrowID, err)
		return
	}
	if err := s.db.UpsertRefereeEscrow(context.Background(), row); err != nil {
		s.log.Warnf("failed to persist escrow session %s: %v", es.EscrowID, err)
	}
}

func (s *Server) persistBranchGamma(matchID string, branch int32, gammaHex string) {
	if s == nil || s.db == nil || strings.TrimSpace(matchID) == "" || strings.TrimSpace(gammaHex) == "" {
		return
	}
	err := s.db.UpsertRefereeBranchGamma(context.Background(), db.RefereeBranchGamma{
		MatchID:   matchID,
		Branch:    branch,
		GammaHex:  gammaHex,
		UpdatedAt: time.Now(),
	})
	if err != nil {
		s.log.Warnf("failed to persist branch gamma for %s/%d: %v", matchID, branch, err)
	}
}

func (s *Server) persistPresignContext(matchID string, ctx *refereePreSignCtx) {
	if s == nil || s.db == nil || ctx == nil || strings.TrimSpace(matchID) == "" {
		return
	}
	row, err := marshalPresignContext(matchID, ctx)
	if err != nil {
		s.log.Warnf("failed to marshal presign %s for %s: %v", ctx.InputID, matchID, err)
		return
	}
	if err := s.db.UpsertRefereePresign(context.Background(), row); err != nil {
		s.log.Warnf("failed to persist presign %s for %s: %v", ctx.InputID, matchID, err)
	}
}

func (s *Server) persistPendingSettlement(matchID string, state pendingSettlementState) {
	if s == nil || s.db == nil || strings.TrimSpace(matchID) == "" {
		return
	}
	if strings.TrimSpace(state.TableID) == "" || strings.TrimSpace(state.WinnerID) == "" {
		return
	}
	err := s.db.UpsertPendingSettlement(context.Background(), db.PendingSettlement{
		MatchID:    matchID,
		TableID:    state.TableID,
		WinnerID:   state.WinnerID,
		WinnerSeat: state.WinnerSeat,
		UpdatedAt:  time.Now(),
	})
	if err != nil {
		s.log.Warnf("failed to persist pending settlement %s: %v", matchID, err)
	}
}

func (s *Server) deletePersistedMatchArtifacts(matchID string) {
	if s == nil || s.db == nil || strings.TrimSpace(matchID) == "" {
		return
	}
	ctx := context.Background()
	if err := s.db.DeleteSettlementEscrows(ctx, matchID); err != nil {
		s.log.Warnf("failed to delete persisted settlement escrows for %s: %v", matchID, err)
	}
	if err := s.db.DeleteRefereePresigns(ctx, matchID); err != nil {
		s.log.Warnf("failed to delete persisted presigns for %s: %v", matchID, err)
	}
	if err := s.db.DeleteRefereeBranchGammas(ctx, matchID); err != nil {
		s.log.Warnf("failed to delete persisted branch gammas for %s: %v", matchID, err)
	}
	if err := s.db.DeletePendingSettlement(ctx, matchID); err != nil {
		s.log.Warnf("failed to delete pending settlement for %s: %v", matchID, err)
	}
}

func (s *Server) loadPersistedRefereeEscrows() error {
	if s == nil || s.referee == nil || s.db == nil {
		return nil
	}
	rows, err := s.db.ListRefereeEscrows(context.Background())
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}

	loaded := 0
	for _, row := range rows {
		es, err := unmarshalEscrowSession(row)
		if err != nil {
			s.log.Warnf("failed to restore escrow session %s: %v", row.EscrowID, err)
			continue
		}
		if es == nil || es.EscrowID == "" {
			continue
		}
		s.referee.mu.Lock()
		s.referee.escrows[es.EscrowID] = es
		s.referee.mu.Unlock()
		if es.fundingState != escrowStateSpent {
			s.subscribeEscrow(es)
		}
		loaded++
	}
	if loaded > 0 {
		s.log.Infof("Loaded %d persisted referee escrow sessions", loaded)
	}
	return nil
}

func (s *Server) loadPersistedBranchGammas() error {
	if s == nil || s.referee == nil || s.db == nil {
		return nil
	}
	rows, err := s.db.ListRefereeBranchGammas(context.Background())
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}

	s.referee.mu.Lock()
	defer s.referee.mu.Unlock()
	loaded := 0
	for _, row := range rows {
		matchID := strings.TrimSpace(row.MatchID)
		gammaHex := strings.TrimSpace(row.GammaHex)
		if matchID == "" || gammaHex == "" {
			continue
		}
		if s.referee.branchGamma[matchID] == nil {
			s.referee.branchGamma[matchID] = make(map[int32]string)
		}
		s.referee.branchGamma[matchID][row.Branch] = gammaHex
		loaded++
	}
	if loaded > 0 {
		s.log.Infof("Loaded %d persisted branch gammas", loaded)
	}
	return nil
}

func (s *Server) loadPersistedPresigns() error {
	if s == nil || s.referee == nil || s.db == nil {
		return nil
	}
	rows, err := s.db.ListRefereePresigns(context.Background())
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}

	loaded := 0
	for _, row := range rows {
		ctx, err := unmarshalPresignContext(row)
		if err != nil {
			s.log.Warnf("failed to restore presign %s for %s/%d: %v", row.InputID, row.MatchID, row.Branch, err)
			continue
		}
		if ctx == nil || ctx.InputID == "" {
			continue
		}
		s.referee.mu.Lock()
		if s.referee.presigns[row.MatchID] == nil {
			s.referee.presigns[row.MatchID] = make(map[int32]map[string]*refereePreSignCtx)
		}
		if s.referee.presigns[row.MatchID][row.Branch] == nil {
			s.referee.presigns[row.MatchID][row.Branch] = make(map[string]*refereePreSignCtx)
		}
		s.referee.presigns[row.MatchID][row.Branch][ctx.InputID] = ctx
		s.referee.mu.Unlock()
		loaded++
	}
	if loaded > 0 {
		s.log.Infof("Loaded %d persisted referee presigns", loaded)
	}
	return nil
}

func (s *Server) loadPendingSettlements() error {
	if s == nil || s.referee == nil || s.db == nil {
		return nil
	}
	rows, err := s.db.ListPendingSettlements(context.Background())
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}

	s.referee.mu.Lock()
	defer s.referee.mu.Unlock()
	loaded := 0
	for _, row := range rows {
		matchID := strings.TrimSpace(row.MatchID)
		tableID := strings.TrimSpace(row.TableID)
		winnerID := strings.TrimSpace(row.WinnerID)
		if matchID == "" || tableID == "" || winnerID == "" {
			continue
		}
		s.referee.pendingSettlements[matchID] = pendingSettlementState{
			TableID:    tableID,
			WinnerID:   winnerID,
			WinnerSeat: row.WinnerSeat,
		}
		loaded++
	}
	if loaded > 0 {
		s.log.Infof("Loaded %d pending settlements", loaded)
	}
	return nil
}
