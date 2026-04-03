package golib

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/companyzero/bisonrelay/client/clientintf"
	"github.com/companyzero/bisonrelay/clientrpc/types"
	"github.com/companyzero/bisonrelay/lockfile"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/vctt94/pokerbisonrelay/pkg/client"
	"github.com/vctt94/pokerbisonrelay/pkg/poker"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	"google.golang.org/protobuf/encoding/protojson"
)

func handleHello(name string) (string, error) {
	if name == "*bug" {
		return "", fmt.Errorf("name '%s' is an error", name)
	}
	return "hello " + name, nil
}

func isClientRunning(handle uint32) bool {
	cmtx.Lock()
	var res bool
	if cs != nil {
		res = cs[handle] != nil
	}
	cmtx.Unlock()
	return res
}

func handleClientCmd(handle uint32, cc *clientCtx, cmd *cmd) (interface{}, error) {
	switch cmd.Type {

	case CTGetUserNick:
		if cc.chat == nil {
			return "", fmt.Errorf("chat RPC not available")
		}
		resp := &types.UserNickResponse{}
		hexUid := strings.Trim(string(cmd.Payload), `"`)
		if err := cc.chat.UserNick(cc.ctx, &types.UserNickRequest{HexUid: hexUid}, resp); err != nil {
			return nil, err
		}
		return resp.Nick, nil

	case CTGetWRPlayers:
		// Not exposed; keep stub for now for UI compatibility.
		return []*player{}, nil

	case CTGetWaitingRooms:
		// Stub implementation - return empty list for now
		cc.log.Infof("GetWaitingRooms called (stub implementation)")
		return []*waitingRoom{}, nil

	case CTJoinWaitingRoom:
		{
			roomID, escrowID, err := parseJoinWRPayload(cmd.Payload)
			if err != nil {
				return nil, fmt.Errorf("join payload: %w", err)
			}
			cc.log.Infof("JoinWaitingRoom called: roomID=%s, escrowID=%s (stub implementation)", roomID, escrowID)
			// Stub implementation - return dummy waiting room
			out := &waitingRoom{
				ID:     roomID,
				HostID: "stub-host",
				BetAmt: 1000,
			}
			return out, nil
		}

	case CTCreateWaitingRoom:
		{
			var req createWaitingRoom
			if err := decodeStrict(cmd.Payload, &req); err != nil {
				return nil, fmt.Errorf("create payload: %w", err)
			}
			cc.log.Infof("CreateWaitingRoom called: clientID=%s, betAmt=%d, escrowID=%s (stub implementation)", req.ClientID, req.BetAmt, req.EscrowId)
			// Stub implementation - return dummy waiting room
			out := &waitingRoom{
				ID:     "stub-room-" + req.ClientID,
				HostID: req.ClientID,
				BetAmt: req.BetAmt,
			}
			// Emit a background notification that a waiting room was created.
			// This allows Flutter UI to update waiting room lists reactively.
			notify(NTWRCreated, out, nil)
			return out, nil
		}

	case CTLeaveWaitingRoom:
		roomID := strings.Trim(string(cmd.Payload), `"`)
		if roomID == "" {
			return nil, fmt.Errorf("leave: empty room id")
		}
		cc.log.Infof("LeaveWaitingRoom called: roomID=%s (stub implementation)", roomID)
		return nil, nil

	case CTGenerateSessionKey:
		cmtx.Lock()
		var cc *clientCtx
		if cs != nil {
			cc = cs[handle]
		}
		cmtx.Unlock()
		if cc == nil || cc.c == nil {
			return nil, fmt.Errorf("client not initialized")
		}
		priv, pub, idx, err := cc.c.GenerateSessionKey()
		if err != nil {
			return nil, fmt.Errorf("failed to generate session key: %v", err)
		}
		return map[string]string{
			"priv":  priv,
			"pub":   pub,
			"index": fmt.Sprintf("%d", idx),
		}, nil

	case CTDeriveSessionKey:
		cmtx.Lock()
		var cc *clientCtx
		if cs != nil {
			cc = cs[handle]
		}
		cmtx.Unlock()
		if cc == nil || cc.c == nil {
			return nil, fmt.Errorf("client not initialized")
		}
		var req struct {
			Index uint64 `json:"index"`
		}
		if err := decodeStrict(cmd.Payload, &req); err != nil {
			return nil, fmt.Errorf("derive session key payload: %w", err)
		}
		priv, pub, err := cc.c.DeriveSessionKeyAt(req.Index)
		if err != nil {
			return nil, fmt.Errorf("failed to derive session key: %v", err)
		}
		return map[string]string{
			"priv":  priv,
			"pub":   pub,
			"index": fmt.Sprintf("%d", req.Index),
		}, nil

	case CTOpenEscrow:
		{
			var req openEscrowReq
			if err := decodeStrict(cmd.Payload, &req); err != nil {
				return nil, fmt.Errorf("open escrow payload: %w", err)
			}
			if req.BetAtoms <= 0 {
				return nil, fmt.Errorf("bet_atoms must be > 0")
			}
			compPub, err := hex.DecodeString(req.CompPubkey)
			if err != nil || len(compPub) != 33 {
				return nil, fmt.Errorf("comp_pubkey must be 33-byte hex")
			}
			if req.CSVBlocks <= 0 {
				req.CSVBlocks = 64
			}
			cmtx.Lock()
			cc := cs[handle]
			cmtx.Unlock()
			if cc == nil || cc.c == nil {
				return nil, fmt.Errorf("client not initialized")
			}
			if cc.log != nil {
				cc.log.Debugf("open escrow token_len=%d handle=%d", len(cc.Token), handle)
			}
			if cc.Token == "" {
				return nil, fmt.Errorf("no session token; login first")
			}
			if cc.c.PayoutAddress() == "" {
				return nil, fmt.Errorf("payout address not set; visit Sign Address to verify one before opening escrow")
			}
			ref := cc.c.Referee(cc.Token)
			resp, err := ref.OpenEscrow(cc.ctx, uint64(req.BetAtoms), uint32(req.CSVBlocks), compPub)
			if err != nil {
				return nil, err
			}
			// Persist escrow info locally for history/refund flows.
			info := &client.EscrowInfo{
				EscrowID:        resp.EscrowId,
				DepositAddress:  resp.DepositAddr,
				RedeemScriptHex: resp.RedeemScriptHex,
				PKScriptHex:     resp.PkScriptHex,
				CSVBlocks:       uint32(req.CSVBlocks),
				Status:          "opened",
				KeyIndex:        uint32(req.KeyIndex),
			}
			if err := cc.c.CacheEscrowInfo(info); err != nil && cc.log != nil {
				cc.log.Warnf("failed to cache escrow info %s: %v", resp.EscrowId, err)
			} else if cc.log != nil {
				cc.log.Debugf("cached escrow %s under datadir %s", resp.EscrowId, cc.c.DataDir)
			}
			return map[string]any{
				"escrow_id":              resp.EscrowId,
				"deposit_address":        resp.DepositAddr,
				"pk_script_hex":          resp.PkScriptHex,
				"redeem_script_hex":      resp.RedeemScriptHex,
				"required_confirmations": resp.RequiredConfirmations,
			}, nil
		}

	case CTGetEscrowStatus:
		{
			var req escrowStatusReq
			if err := decodeStrict(cmd.Payload, &req); err != nil {
				return nil, fmt.Errorf("escrow status payload: %w", err)
			}
			if req.EscrowID == "" {
				return nil, fmt.Errorf("escrow_id required")
			}
			cmtx.Lock()
			cc := cs[handle]
			cmtx.Unlock()
			if cc == nil || cc.c == nil {
				return nil, fmt.Errorf("client not initialized")
			}
			if cc.Token == "" {
				return nil, fmt.Errorf("no session token; login first")
			}
			ref := cc.c.Referee(cc.Token)
			resp, err := ref.GetEscrowStatus(cc.ctx, req.EscrowID)
			if err != nil {
				return nil, err
			}
			return map[string]any{
				"escrow_id":              resp.GetEscrowId(),
				"confs":                  resp.GetConfs(),
				"utxo_count":             resp.GetUtxoCount(),
				"ok":                     resp.GetOk(),
				"updated_at_unix":        resp.GetUpdatedAtUnix(),
				"funding_txid":           resp.GetFundingTxid(),
				"funding_vout":           resp.GetFundingVout(),
				"amount_atoms":           resp.GetAmountAtoms(),
				"csv_blocks":             resp.GetCsvBlocks(),
				"required_confirmations": resp.GetRequiredConfirmations(),
				"mature_for_csv":         resp.GetMatureForCsv(),
				"funding_state":          resp.GetFundingState(),
			}, nil
		}

	case CTGetEscrowHistory:
		{
			cmtx.Lock()
			cc := cs[handle]
			cmtx.Unlock()
			if cc == nil || cc.c == nil {
				return nil, fmt.Errorf("client not initialized")
			}
			hist, err := cc.c.GetEscrowHistory()
			if err != nil {
				return nil, err
			}
			return hist, nil
		}

	case CTGetBindableEscrows:
		{
			cmtx.Lock()
			cc := cs[handle]
			cmtx.Unlock()
			if cc == nil || cc.c == nil {
				return nil, fmt.Errorf("client not initialized")
			}
			if cc.Token == "" {
				return nil, fmt.Errorf("no session token; login first")
			}
			escrows, err := cc.c.GetBindableEscrows(cc.ctx, cc.Token)
			if err != nil {
				return nil, err
			}
			return escrows, nil
		}

	case CTRefundEscrow:
		{
			var req struct {
				EscrowID  string `json:"escrow_id"`
				DestAddr  string `json:"dest_addr"`
				FeeAtoms  uint64 `json:"fee_atoms"`
				CSVBlocks uint32 `json:"csv_blocks"`
				UtxoValue uint64 `json:"utxo_value,omitempty"`
			}
			if err := decodeStrict(cmd.Payload, &req); err != nil {
				return nil, fmt.Errorf("refund escrow payload: %w", err)
			}
			if strings.TrimSpace(req.EscrowID) == "" {
				return nil, fmt.Errorf("refund escrow requires escrow_id")
			}
			if strings.TrimSpace(req.DestAddr) == "" {
				return nil, fmt.Errorf("refund escrow requires dest_addr")
			}
			cmtx.Lock()
			cc := cs[handle]
			cmtx.Unlock()
			if cc == nil || cc.c == nil {
				return nil, fmt.Errorf("client not initialized")
			}
			feeAtoms := req.FeeAtoms
			if feeAtoms == 0 {
				feeAtoms = 20000
			}
			res, err := cc.c.RefundEscrow(req.EscrowID, req.DestAddr, feeAtoms, req.CSVBlocks, req.UtxoValue)
			if err != nil {
				return nil, err
			}
			return res, nil
		}

	case CTUpdateEscrowHistory:
		{
			var payload map[string]interface{}
			if err := json.Unmarshal(cmd.Payload, &payload); err != nil {
				return nil, fmt.Errorf("update escrow history payload: %v", err)
			}
			escrowIDRaw, ok := payload["escrow_id"]
			if !ok {
				return nil, fmt.Errorf("update escrow history payload missing escrow_id")
			}
			escrowID := strings.TrimSpace(fmt.Sprint(escrowIDRaw))
			if escrowID == "" {
				return nil, fmt.Errorf("update escrow history payload missing escrow_id")
			}
			info := &client.EscrowInfo{EscrowID: escrowID}
			if txid, ok := payload["funding_txid"].(string); ok {
				info.FundingTxid = strings.TrimSpace(txid)
			}
			if v, exists := payload["funding_vout"]; exists {
				switch vv := v.(type) {
				case float64:
					info.FundingVout = uint32(vv)
				case int:
					info.FundingVout = uint32(vv)
				case int32:
					info.FundingVout = uint32(vv)
				case int64:
					info.FundingVout = uint32(vv)
				}
			}
			if amount, exists := payload["funded_amount"]; exists {
				switch av := amount.(type) {
				case float64:
					info.FundedAmount = uint64(av)
				case int:
					info.FundedAmount = uint64(av)
				case int32:
					info.FundedAmount = uint64(av)
				case int64:
					info.FundedAmount = uint64(av)
				}
			}
			if redeem, ok := payload["redeem_script_hex"].(string); ok {
				info.RedeemScriptHex = strings.TrimSpace(redeem)
			}
			if pk, ok := payload["pk_script_hex"].(string); ok {
				info.PKScriptHex = strings.TrimSpace(pk)
			}
			if csv, exists := payload["csv_blocks"]; exists {
				switch cv := csv.(type) {
				case float64:
					info.CSVBlocks = uint32(cv)
				case int:
					info.CSVBlocks = uint32(cv)
				case int32:
					info.CSVBlocks = uint32(cv)
				case int64:
					info.CSVBlocks = uint32(cv)
				}
			}
			if status, ok := payload["status"].(string); ok {
				info.Status = strings.TrimSpace(status)
			}
			cmtx.Lock()
			cc := cs[handle]
			cmtx.Unlock()
			if cc == nil || cc.c == nil {
				return nil, fmt.Errorf("client not initialized")
			}
			if err := cc.c.UpdateEscrowHistory(info); err != nil {
				return nil, err
			}
			return map[string]string{"status": "updated"}, nil
		}

	case CTDeleteEscrowHistory:
		{
			var req struct {
				EscrowID string `json:"escrow_id"`
			}
			if err := decodeStrict(cmd.Payload, &req); err != nil {
				return nil, fmt.Errorf("delete escrow history payload: %w", err)
			}
			escrowID := strings.TrimSpace(req.EscrowID)
			if escrowID == "" {
				return nil, fmt.Errorf("delete escrow history payload missing escrow_id")
			}
			cmtx.Lock()
			cc := cs[handle]
			cmtx.Unlock()
			if cc == nil || cc.c == nil {
				return nil, fmt.Errorf("client not initialized")
			}
			if err := cc.c.DeleteEscrowHistory(escrowID); err != nil {
				return nil, err
			}
			return map[string]string{"status": "deleted"}, nil
		}

	case CTGetEscrowById:
		{
			var req struct {
				EscrowID string `json:"escrow_id"`
			}
			if err := decodeStrict(cmd.Payload, &req); err != nil {
				return nil, fmt.Errorf("get escrow by id payload: %w", err)
			}
			if req.EscrowID == "" {
				return nil, fmt.Errorf("escrow_id required")
			}
			cmtx.Lock()
			cc := cs[handle]
			cmtx.Unlock()
			if cc == nil || cc.c == nil {
				return nil, fmt.Errorf("client not initialized")
			}
			info, err := cc.c.GetEscrowById(req.EscrowID)
			if err != nil {
				return nil, err
			}
			// If we have a key_index, derive the session private key on-the-fly
			// (safer than storing the private key on disk)
			if idx, ok := info["key_index"].(float64); ok && idx > 0 {
				privHex, _, err := cc.c.DeriveSessionKeyAt(uint64(idx))
				if err == nil {
					info["comp_priv"] = privHex
				}
			}
			return info, nil
		}

	case CTGetFinalizeBundle:
		{
			var req struct {
				MatchID    string `json:"match_id"`
				WinnerSeat int32  `json:"winner_seat"`
			}
			if err := decodeStrict(cmd.Payload, &req); err != nil {
				return nil, fmt.Errorf("finalize bundle payload: %w", err)
			}
			if req.MatchID == "" {
				return nil, fmt.Errorf("match_id required")
			}
			cmtx.Lock()
			cc := cs[handle]
			cmtx.Unlock()
			if cc == nil || cc.c == nil {
				return nil, fmt.Errorf("client not initialized")
			}
			if cc.Token == "" {
				return nil, fmt.Errorf("no session token; login first")
			}
			ref := cc.c.Referee(cc.Token)
			resp, err := ref.GetFinalizeBundle(cc.ctx, req.MatchID, req.WinnerSeat)
			if err != nil {
				return nil, fmt.Errorf("GetFinalizeBundle failed: %w", err)
			}
			// Build inputs array for response
			inputs := make([]map[string]any, 0, len(resp.GetInputs()))
			for _, in := range resp.GetInputs() {
				inputs = append(inputs, map[string]any{
					"input_id":            in.GetInputId(),
					"r_prime_compact_hex": in.GetRPrimeCompactHex(),
					"s_prime_hex":         in.GetSPrimeHex(),
					"input_index":         in.GetInputIndex(),
					"redeem_script_hex":   in.GetRedeemScriptHex(),
				})
			}
			return map[string]any{
				"match_id":     resp.GetMatchId(),
				"branch":       resp.GetBranch(),
				"draft_tx_hex": resp.GetDraftTxHex(),
				"gamma_hex":    resp.GetGammaHex(),
				"inputs":       inputs,
			}, nil
		}

	case CTBindEscrow:
		{
			var req struct {
				TableID   string `json:"table_id"`
				SessionID string `json:"session_id"`
				MatchID   string `json:"match_id"`
				SeatIndex int    `json:"seat_index"`
				Outpoint  string `json:"outpoint"`
			}
			if err := decodeStrict(cmd.Payload, &req); err != nil {
				return nil, fmt.Errorf("bind escrow payload: %w", err)
			}
			if req.TableID == "" {
				return nil, fmt.Errorf("table_id required")
			}
			if strings.TrimSpace(req.Outpoint) == "" {
				return nil, fmt.Errorf("outpoint required")
			}
			cmtx.Lock()
			cc := cs[handle]
			cmtx.Unlock()
			if cc == nil || cc.c == nil {
				return nil, fmt.Errorf("client not initialized")
			}
			if cc.Token == "" {
				return nil, fmt.Errorf("no session token; login first")
			}
			var (
				redeemScriptHex string
				csvBlocks       uint32
			)
			// Attempt to hydrate redeem/csv from cached escrow history matching the outpoint
			parts := strings.Split(req.Outpoint, ":")
			if len(parts) == 2 {
				txid := strings.TrimSpace(parts[0])
				vout := strings.TrimSpace(parts[1])
				if hist, err := cc.c.GetEscrowHistory(); err == nil {
					for _, m := range hist {
						tx, _ := m["funding_txid"].(string)
						if strings.TrimSpace(tx) != txid {
							continue
						}
						var fv string
						switch vv := m["funding_vout"].(type) {
						case float64:
							fv = fmt.Sprintf("%.0f", vv)
						case int:
							fv = fmt.Sprintf("%d", vv)
						case int64:
							fv = fmt.Sprintf("%d", vv)
						case uint64:
							fv = fmt.Sprintf("%d", vv)
						case json.Number:
							fv = vv.String()
						case string:
							fv = strings.TrimSpace(vv)
						}
						if fv != vout {
							continue
						}
						if r, ok := m["redeem_script_hex"].(string); ok {
							redeemScriptHex = strings.TrimSpace(r)
						}
						switch cb := m["csv_blocks"].(type) {
						case float64:
							csvBlocks = uint32(cb)
						case int:
							csvBlocks = uint32(cb)
						case int64:
							csvBlocks = uint32(cb)
						case uint64:
							csvBlocks = uint32(cb)
						case json.Number:
							if n, err := cb.Int64(); err == nil {
								csvBlocks = uint32(n)
							}
						}
						break
					}
				}
			}
			ref := cc.c.Referee(cc.Token)
			resp, err := ref.BindEscrow(cc.ctx, req.TableID, req.SessionID, req.MatchID, uint32(req.SeatIndex), req.Outpoint, redeemScriptHex, csvBlocks)
			if err != nil {
				return nil, err
			}
			return map[string]any{
				"match_id":              resp.GetMatchId(),
				"table_id":              resp.GetTableId(),
				"session_id":            resp.GetSessionId(),
				"seat_index":            resp.GetSeatIndex(),
				"escrow_id":             resp.GetEscrowId(),
				"escrow_ready":          resp.GetEscrowReady(),
				"amount_atoms":          resp.GetAmountAtoms(),
				"required_amount_atoms": resp.GetRequiredAmountAtoms(),
			}, nil
		}

	case CTStartPreSign:
		{
			var req preSignReq
			if err := decodeStrict(cmd.Payload, &req); err != nil {
				return nil, fmt.Errorf("presign payload: %w", err)
			}
			if req.MatchID == "" {
				return nil, fmt.Errorf("match_id required")
			}
			if req.EscrowID == "" || req.CompPriv == "" {
				return nil, fmt.Errorf("escrow_id and comp_priv required")
			}
			compPriv, err := hex.DecodeString(req.CompPriv)
			if err != nil || len(compPriv) == 0 {
				return nil, fmt.Errorf("bad comp_priv")
			}
			compKey := secp256k1.PrivKeyFromBytes(compPriv)
			compPub := compKey.PubKey().SerializeCompressed()
			cmtx.Lock()
			cc := cs[handle]
			cmtx.Unlock()
			if cc == nil || cc.c == nil {
				return nil, fmt.Errorf("client not initialized")
			}
			if cc.Token == "" {
				return nil, fmt.Errorf("no session token; login first")
			}
			ref := cc.c.Referee(cc.Token)
			if err := ref.StartPresign(cc.ctx, req.MatchID, req.TableID, req.EscrowID, compPub, req.CompPriv); err != nil {
				return nil, err
			}
			return map[string]string{"status": "ok"}, nil
		}

	case CTArchiveSessionKey:
		{
			var req struct {
				MatchID    string                 `json:"match_id"`
				EscrowInfo map[string]interface{} `json:"escrow_info,omitempty"`
			}
			if err := decodeStrict(cmd.Payload, &req); err != nil {
				return nil, fmt.Errorf("archive payload: %w", err)
			}
			if req.MatchID == "" {
				return nil, fmt.Errorf("archive: empty match_id")
			}
			if req.EscrowInfo == nil {
				return nil, fmt.Errorf("archive payload requires escrow_info with funding details")
			}

			escrowInfo := &client.EscrowInfo{}
			if id, ok := req.EscrowInfo["escrow_id"].(string); ok {
				escrowInfo.EscrowID = id
			}
			if txid, ok := req.EscrowInfo["funding_txid"].(string); ok {
				escrowInfo.FundingTxid = txid
			}
			if addr, ok := req.EscrowInfo["deposit_address"].(string); ok {
				escrowInfo.DepositAddress = addr
			}
			hasVout := false
			if vout, ok := req.EscrowInfo["funding_vout"].(float64); ok {
				escrowInfo.FundingVout = uint32(vout)
				hasVout = true
			}
			hasAmount := false
			if amount, ok := req.EscrowInfo["funded_amount"].(float64); ok {
				escrowInfo.FundedAmount = uint64(amount)
				hasAmount = true
			}
			if redeem, ok := req.EscrowInfo["redeem_script_hex"].(string); ok {
				escrowInfo.RedeemScriptHex = redeem
			}
			if pk, ok := req.EscrowInfo["pk_script_hex"].(string); ok {
				escrowInfo.PKScriptHex = pk
			}
			hasCSV := false
			if csv, ok := req.EscrowInfo["csv_blocks"].(float64); ok {
				escrowInfo.CSVBlocks = uint32(csv)
				hasCSV = true
			}
			if confirmedHeight, ok := req.EscrowInfo["confirmed_height"].(float64); ok {
				escrowInfo.ConfirmedHeight = uint32(confirmedHeight)
			} else if archived, ok := req.EscrowInfo["archived_at"].(float64); ok {
				// Legacy field name; map to confirmed_height for compatibility.
				escrowInfo.ConfirmedHeight = uint32(archived)
			}
			if status, ok := req.EscrowInfo["status"].(string); ok {
				escrowInfo.Status = status
			}

			switch {
			case escrowInfo.EscrowID == "":
				return nil, fmt.Errorf("escrow_info missing escrow_id")
			case escrowInfo.FundingTxid == "":
				return nil, fmt.Errorf("escrow_info missing funding_txid")
			case !hasVout:
				return nil, fmt.Errorf("escrow_info missing funding_vout")
			case !hasAmount:
				return nil, fmt.Errorf("escrow_info missing funded_amount")
			case escrowInfo.RedeemScriptHex == "":
				return nil, fmt.Errorf("escrow_info missing redeem_script_hex")
			case escrowInfo.PKScriptHex == "":
				return nil, fmt.Errorf("escrow_info missing pk_script_hex")
			case !hasCSV:
				return nil, fmt.Errorf("escrow_info missing csv_blocks")
			}
			// No fallback timestamp; leave height zero if unknown.

			if err := cc.c.ArchiveEscrowSession(req.MatchID, escrowInfo); err != nil {
				return nil, err
			}
			return map[string]string{"status": "archived"}, nil
		}

	case CTStopClient:
		cc.cancel()
		return nil, nil

	case CTGetPokerTables:
		if cc.c == nil {
			return nil, fmt.Errorf("poker client not initialized")
		}
		tables, err := cc.c.GetTables(cc.ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get tables: %v", err)
		}
		// Convert protobuf tables to DTOs with explicit field types
		dtos := make([]*pokerTable, 0, len(tables))
		for _, t := range tables {
			dtos = append(dtos, tableFromProto(t))
		}
		return dtos, nil

	case CTJoinPokerTable:
		var req joinPokerTable
		if err := decodeStrict(cmd.Payload, &req); err != nil {
			return nil, fmt.Errorf("join table payload: %w", err)
		}
		if cc.c == nil {
			return nil, fmt.Errorf("poker client not initialized")
		}
		err := cc.c.JoinTable(cc.ctx, req.TableID)
		if err != nil {
			return nil, fmt.Errorf("failed to join table: %v", err)
		}

		return map[string]string{"status": "joined", "table_id": req.TableID}, nil

	case CTWatchPokerTable:
		var req joinPokerTable
		if err := decodeStrict(cmd.Payload, &req); err != nil {
			return nil, fmt.Errorf("watch table payload: %w", err)
		}
		if cc.c == nil {
			return nil, fmt.Errorf("poker client not initialized")
		}
		err := cc.c.WatchTable(cc.ctx, req.TableID)
		if err != nil {
			return nil, fmt.Errorf("failed to watch table: %v", err)
		}

		return map[string]string{"status": "watching", "table_id": req.TableID}, nil

	case CTCreatePokerTable:
		var req createPokerTable
		if err := decodeStrict(cmd.Payload, &req); err != nil {
			return nil, fmt.Errorf("create table payload: %w", err)
		}
		if cc.c == nil {
			return nil, fmt.Errorf("poker client not initialized")
		}

		// Create TableConfig from request
		config := poker.TableConfig{
			SmallBlind:            req.SmallBlind,
			BigBlind:              req.BigBlind,
			MaxPlayers:            int(req.MaxPlayers),
			MinPlayers:            int(req.MinPlayers),
			BuyIn:                 req.BuyIn,
			StartingChips:         req.StartingChips,
			TimeBank:              time.Duration(req.TimeBankSeconds) * time.Second,
			AutoStartDelay:        time.Duration(req.AutoStartMs) * time.Millisecond,
			AutoAdvanceDelay:      time.Duration(req.AutoAdvanceMs) * time.Millisecond,
			BlindIncreaseInterval: time.Duration(req.BlindIncreaseIntervalSec) * time.Second,
		}

		tableID, err := cc.c.CreateTable(cc.ctx, config)
		if err != nil {
			return nil, fmt.Errorf("failed to create table: %v", err)
		}
		return map[string]string{"status": "created", "table_id": tableID}, nil

	case CTLeavePokerTable:
		if cc.c == nil {
			return nil, fmt.Errorf("poker client not initialized")
		}
		err := cc.c.LeaveTable(cc.ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to leave table: %v", err)
		}
		return map[string]string{"status": "left"}, nil

	case CTUnwatchPokerTable:
		if cc.c == nil {
			return nil, fmt.Errorf("poker client not initialized")
		}
		err := cc.c.UnwatchTable(cc.ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to unwatch table: %v", err)
		}
		return map[string]string{"status": "unwatched"}, nil

	case CTGetPlayerCurrentTable:
		if cc.c == nil {
			return nil, fmt.Errorf("poker client not initialized")
		}
		tid, err := cc.c.GetPlayerCurrentTable(cc.ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get current table: %v", err)
		}
		return map[string]string{"table_id": tid}, nil

	case CTShowCards:
		if cc.c == nil {
			return nil, fmt.Errorf("poker client not initialized")
		}
		err := cc.c.ShowCards(cc.ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to show cards: %v", err)
		}
		return map[string]string{"status": "ok"}, nil

	case CTHideCards:
		if cc.c == nil {
			return nil, fmt.Errorf("poker client not initialized")
		}
		err := cc.c.HideCards(cc.ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to hide cards: %v", err)
		}
		return map[string]string{"status": "ok"}, nil

	case CTMakeBet:
		var req makeBet
		if err := decodeStrict(cmd.Payload, &req); err != nil {
			return nil, fmt.Errorf("make bet payload: %w", err)
		}
		if cc.c == nil {
			return nil, fmt.Errorf("poker client not initialized")
		}
		err := cc.c.Bet(cc.ctx, req.Amount)
		if err != nil {
			return nil, fmt.Errorf("failed to make bet: %v", err)
		}
		return map[string]string{"status": "ok"}, nil

	case CTCallBet:
		if cc.c == nil {
			return nil, fmt.Errorf("poker client not initialized")
		}
		// Call doesn't need currentBet parameter - server handles it
		err := cc.c.Call(cc.ctx, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to call bet: %v", err)
		}
		return map[string]string{"status": "ok"}, nil

	case CTFoldBet:
		if cc.c == nil {
			return nil, fmt.Errorf("poker client not initialized")
		}
		err := cc.c.Fold(cc.ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to fold: %v", err)
		}
		return map[string]string{"status": "ok"}, nil

	case CTCheckBet:
		if cc.c == nil {
			return nil, fmt.Errorf("poker client not initialized")
		}
		err := cc.c.Check(cc.ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to check: %v", err)
		}
		return map[string]string{"status": "ok"}, nil

	case CTGetGameState:
		if cc.c == nil {
			return nil, fmt.Errorf("poker client not initialized")
		}
		tableID := cc.c.GetCurrentTableID()
		if tableID == "" {
			return nil, fmt.Errorf("not currently in a table")
		}
		resp, err := cc.c.PokerService.GetGameState(cc.ctx, &pokerrpc.GetGameStateRequest{
			PlayerId: cc.c.ID.String(),
			TableId:  tableID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get game state: %v", err)
		}
		// Convert protobuf to simple DTO for JSON marshaling
		dto := gameUpdateToDTO(resp.GameState)
		return map[string]interface{}{
			"game_state": dto,
		}, nil

	case CTGetLastWinners:
		if cc.c == nil {
			return nil, fmt.Errorf("poker client not initialized")
		}
		tableID := cc.c.GetCurrentTableID()
		if tableID == "" {
			return nil, fmt.Errorf("not currently in a table")
		}
		resp, err := cc.c.PokerService.GetLastWinners(cc.ctx, &pokerrpc.GetLastWinnersRequest{
			TableId: tableID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get last winners: %v", err)
		}
		// Convert protobuf winners to JSON
		jsonBytes, err := protojson.Marshal(resp)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal winners: %v", err)
		}
		var winnersMap map[string]interface{}
		if err := json.Unmarshal(jsonBytes, &winnersMap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal winners JSON: %v", err)
		}
		return winnersMap, nil

	case CTStartGameStream:
		if cc.c == nil {
			return nil, fmt.Errorf("poker client not initialized")
		}
		err := cc.c.StartGameStream(cc.ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to start game stream: %v", err)
		}
		return map[string]string{"status": "ok"}, nil

	case CTReconnectNow:
		if cc.c == nil {
			return nil, fmt.Errorf("poker client not initialized")
		}
		err := cc.c.ReconnectNow()
		if err != nil {
			return nil, fmt.Errorf("failed to reconnect now: %v", err)
		}
		return map[string]string{"status": "ok"}, nil

	case CTEvaluateHand:
		var req evaluateHand
		if err := decodeStrict(cmd.Payload, &req); err != nil {
			return nil, fmt.Errorf("evaluate hand payload: %w", err)
		}
		if cc.c == nil {
			return nil, fmt.Errorf("poker client not initialized")
		}
		// Convert cards to protobuf format - Card uses strings for suit and value
		cards := make([]*pokerrpc.Card, 0, len(req.Cards))
		for _, c := range req.Cards {
			// Convert int32 to string representation
			// Assuming suit: 0=Spades, 1=Hearts, 2=Diamonds, 3=Clubs
			// and value: 2-14 (2-10, J, Q, K, A)
			suitStr := []string{"Spades", "Hearts", "Diamonds", "Clubs"}[c.Suit]
			valueStr := fmt.Sprintf("%d", c.Value)
			if c.Value == 11 {
				valueStr = "J"
			} else if c.Value == 12 {
				valueStr = "Q"
			} else if c.Value == 13 {
				valueStr = "K"
			} else if c.Value == 14 {
				valueStr = "A"
			}
			cards = append(cards, &pokerrpc.Card{
				Suit:  suitStr,
				Value: valueStr,
			})
		}
		resp, err := cc.c.PokerService.EvaluateHand(cc.ctx, &pokerrpc.EvaluateHandRequest{
			Cards: cards,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate hand: %v", err)
		}
		// Convert protobuf response to JSON
		jsonBytes, err := protojson.Marshal(resp)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal evaluate hand response: %v", err)
		}
		var evalMap map[string]interface{}
		if err := json.Unmarshal(jsonBytes, &evalMap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal evaluate hand JSON: %v", err)
		}
		return evalMap, nil

	case CTSetPlayerReady:
		if cc.c == nil {
			return nil, fmt.Errorf("poker client not initialized")
		}
		err := cc.c.SetPlayerReady(cc.ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to set player ready: %v", err)
		}
		return map[string]string{"status": "ok"}, nil

	case CTSetPlayerUnready:
		if cc.c == nil {
			return nil, fmt.Errorf("poker client not initialized")
		}
		err := cc.c.SetPlayerUnready(cc.ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to set player unready: %v", err)
		}
		return map[string]string{"status": "ok"}, nil

	default:
		return nil, fmt.Errorf("unknown cmd 0x%x", cmd.Type)
	}
}

// handleRegister handles the CTRegister command
func handleRegister(handle uint32, req registerReq) (interface{}, error) {
	cmtx.Lock()
	var cc *clientCtx
	if cs != nil {
		cc = cs[handle]
	}
	cmtx.Unlock()

	if cc == nil {
		return nil, fmt.Errorf("unknown client handle %d", handle)
	}
	if cc.c == nil {
		return nil, fmt.Errorf("poker client not initialized")
	}

	err := cc.c.Register(cc.ctx, req.Nickname)
	if err != nil {
		return nil, fmt.Errorf("failed to register: %v", err)
	}

	// Update nickname in client context
	cmtx.Lock()
	cc.Nick = req.Nickname
	cmtx.Unlock()

	return map[string]string{"status": "ok"}, nil
}

// handleLogin handles the CTLogin command
func handleLogin(handle uint32, req loginReq) (interface{}, error) {
	cmtx.Lock()
	var cc *clientCtx
	if cs != nil {
		cc = cs[handle]
	}
	cmtx.Unlock()

	if cc == nil {
		return nil, fmt.Errorf("unknown client handle %d", handle)
	}
	if cc.c == nil {
		return nil, fmt.Errorf("poker client not initialized")
	}

	clientLoginResp, err := cc.c.Login(cc.ctx, req.Nickname)
	if err != nil {
		return nil, fmt.Errorf("failed to login: %v", err)
	}

	// Update nickname in client context
	cmtx.Lock()
	cc.Nick = clientLoginResp.Nickname
	cc.Token = clientLoginResp.Token
	cmtx.Unlock()
	if cc.log != nil {
		cc.log.Debugf("login success nick=%s token_len=%d", cc.Nick, len(cc.Token))
	}

	// Return loginResp struct (will be JSON marshaled automatically)
	return loginResp{
		Token:    clientLoginResp.Token,
		UserID:   clientLoginResp.UserID,
		Nickname: clientLoginResp.Nickname,
		Address:  clientLoginResp.PayoutAddress,
	}, nil
}

// handleRequestLoginCode asks the server for a short-lived nonce to be signed
// by the user's Decred wallet.
func handleRequestLoginCode(handle uint32) (interface{}, error) {
	cmtx.Lock()
	var cc *clientCtx
	if cs != nil {
		cc = cs[handle]
	}
	cmtx.Unlock()

	if cc == nil {
		return nil, fmt.Errorf("unknown client handle %d", handle)
	}
	if cc.c == nil {
		return nil, fmt.Errorf("poker client not initialized")
	}

	resp, err := cc.c.RequestLoginCode(cc.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to request login code: %v", err)
	}

	return map[string]interface{}{
		"code":         resp.Code,
		"ttl_sec":      int64(resp.TTL.Seconds()),
		"address_hint": resp.AddressHint,
	}, nil
}

// handleResumeSession tries to reuse an existing server session.
func handleResumeSession(handle uint32) (interface{}, error) {
	cmtx.Lock()
	var cc *clientCtx
	if cs != nil {
		cc = cs[handle]
	}
	cmtx.Unlock()

	if cc == nil {
		return nil, fmt.Errorf("unknown client handle %d", handle)
	}
	if cc.c == nil {
		return nil, fmt.Errorf("poker client not initialized")
	}

	session, err := cc.c.ResumeSession(cc.ctx)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, nil
	}

	cmtx.Lock()
	cc.Nick = session.Nickname
	cc.Token = session.Token
	cmtx.Unlock()
	if cc.log != nil {
		cc.log.Debugf("resume session nick=%s token_len=%d", cc.Nick, len(cc.Token))
	}

	return loginResp{
		Token:    session.Token,
		UserID:   session.UserID,
		Nickname: session.Nickname,
		Address:  session.PayoutAddress,
	}, nil
}

func handleSetPayoutAddress(handle uint32, req setPayoutAddressReq) (interface{}, error) {
	cmtx.Lock()
	var cc *clientCtx
	if cs != nil {
		cc = cs[handle]
	}
	cmtx.Unlock()

	if cc == nil {
		return nil, fmt.Errorf("unknown client handle %d", handle)
	}
	if cc.c == nil {
		return nil, fmt.Errorf("poker client not initialized")
	}
	if cc.Token == "" {
		return nil, fmt.Errorf("no session token; login first")
	}

	ref := cc.c.Referee(cc.Token)
	addr, err := ref.SetPayoutAddress(cc.ctx, req.Address, req.Signature, req.Code)
	if err != nil {
		return nil, fmt.Errorf("failed to set payout address: %v", err)
	}

	cc.c.PersistPayoutAddress(addr)

	return map[string]any{
		"ok":      true,
		"address": addr,
	}, nil
}

// handleLogout handles the CTLogout command
func handleLogout(handle uint32) (interface{}, error) {
	cmtx.Lock()
	var cc *clientCtx
	if cs != nil {
		cc = cs[handle]
	}
	cmtx.Unlock()

	if cc == nil {
		return nil, fmt.Errorf("unknown client handle %d", handle)
	}
	if cc.c == nil {
		return nil, fmt.Errorf("poker client not initialized")
	}

	// Clear any persisted session token so the next launch requires re-auth.
	if err := cc.c.ClearSession(); err != nil {
		cc.log.Warnf("failed to clear session on logout: %v", err)
	}
	return map[string]string{"status": "ok"}, nil
}

func handleCreateLockFile(rootDir string) error {
	filePath := filepath.Join(rootDir, clientintf.LockFileName)

	cmtx.Lock()
	defer cmtx.Unlock()

	lf := lfs[filePath]
	if lf != nil {
		// Already running on this DB from this process.
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	lf, err := lockfile.Create(ctx, filePath)
	cancel()
	if err != nil {
		return fmt.Errorf("unable to create lockfile %q: %v", filePath, err)
	}
	lfs[filePath] = lf
	return nil
}

func handleCloseLockFile(rootDir string) error {
	filePath := filepath.Join(rootDir, clientintf.LockFileName)

	cmtx.Lock()
	lf := lfs[filePath]
	delete(lfs, filePath)
	cmtx.Unlock()

	if lf == nil {
		return fmt.Errorf("nil lockfile")
	}
	return lf.Close()
}
