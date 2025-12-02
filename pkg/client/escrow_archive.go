package client

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// EscrowInfo represents the data we need to store about an escrow for potential refund.
type EscrowInfo struct {
	EscrowID        string `json:"escrow_id"`
	DepositAddress  string `json:"deposit_address,omitempty"`
	FundingTxid     string `json:"funding_txid"`
	FundingVout     uint32 `json:"funding_vout"`
	FundedAmount    uint64 `json:"funded_amount"`
	RedeemScriptHex string `json:"redeem_script_hex"`
	PKScriptHex     string `json:"pk_script_hex"`
	CSVBlocks       uint32 `json:"csv_blocks"`
	Status          string `json:"status,omitempty"`
	ConfirmedHeight uint32 `json:"confirmed_height,omitempty"`
	Confs           uint32 `json:"confs,omitempty"`
	KeyIndex        uint32 `json:"key_index,omitempty"` // session key derivation index (safer than storing priv key)
}

// SessionArchive captures the escrow info we persist per match/escrow.
type SessionArchive struct {
	MatchID string      `json:"match_id"`
	Escrow  *EscrowInfo `json:"escrow_info"`
}

func (pc *PokerClient) historyDir() string {
	if pc.DataDir == "" {
		return ""
	}
	return filepath.Join(pc.DataDir, "history_session")
}

func sanitizeForFile(s string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9_-]+`)
	return strings.Trim(re.ReplaceAllString(s, "_"), "_")
}

// CacheEscrowInfo stores interim escrow metadata keyed by escrow ID.
func (pc *PokerClient) CacheEscrowInfo(info *EscrowInfo) error {
	if info == nil || strings.TrimSpace(info.EscrowID) == "" {
		return fmt.Errorf("escrow info missing escrow_id")
	}
	dir := pc.historyDir()
	if dir == "" {
		return fmt.Errorf("no data directory configured")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	path := filepath.Join(dir, fmt.Sprintf("escrow_%s.json", sanitizeForFile(info.EscrowID)))

	// Merge with existing info to avoid losing fields when partial updates arrive.
	if curBytes, err := os.ReadFile(path); err == nil {
		var cur EscrowInfo
		if err := json.Unmarshal(curBytes, &cur); err == nil {
			mergeEscrowInfo(&cur, info)
			info = &cur
		}
	}

	blob, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, blob, 0600)
}

// ArchiveEscrowSession stores escrow info under history_session for later recovery.
func (pc *PokerClient) ArchiveEscrowSession(matchID string, info *EscrowInfo) error {
	matchID = strings.TrimSpace(matchID)
	if matchID == "" {
		return fmt.Errorf("match_id required")
	}
	if info == nil {
		return fmt.Errorf("escrow info required")
	}
	dir := pc.historyDir()
	if dir == "" {
		return fmt.Errorf("no data directory configured")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	file := fmt.Sprintf("%s_%s.json", sanitizeForFile(matchID), sanitizeForFile(info.EscrowID))
	path := filepath.Join(dir, file)
	archive := SessionArchive{MatchID: matchID, Escrow: info}
	blob, err := json.MarshalIndent(archive, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, blob, 0600)
}

func mergeEscrowInfo(dst, src *EscrowInfo) {
	if dst == nil || src == nil {
		return
	}
	if dst.DepositAddress == "" {
		dst.DepositAddress = src.DepositAddress
	}
	if dst.FundingTxid == "" {
		dst.FundingTxid = src.FundingTxid
	}
	if dst.FundingVout == 0 {
		dst.FundingVout = src.FundingVout
	}
	if dst.FundedAmount == 0 {
		dst.FundedAmount = src.FundedAmount
	}
	if dst.RedeemScriptHex == "" {
		dst.RedeemScriptHex = src.RedeemScriptHex
	}
	if dst.PKScriptHex == "" {
		dst.PKScriptHex = src.PKScriptHex
	}
	if dst.CSVBlocks == 0 {
		dst.CSVBlocks = src.CSVBlocks
	}
	if src.Status != "" {
		dst.Status = src.Status
	}
	if src.ConfirmedHeight != 0 {
		dst.ConfirmedHeight = src.ConfirmedHeight
	}
	if dst.KeyIndex == 0 && src.KeyIndex != 0 {
		dst.KeyIndex = src.KeyIndex
	}
}

// GetEscrowById returns the cached escrow info for a specific escrow ID.
func (pc *PokerClient) GetEscrowById(escrowID string) (map[string]interface{}, error) {
	escrowID = strings.TrimSpace(escrowID)
	if escrowID == "" {
		return nil, fmt.Errorf("escrow_id required")
	}
	dir := pc.historyDir()
	if dir == "" {
		return nil, fmt.Errorf("no data directory configured")
	}
	path := filepath.Join(dir, fmt.Sprintf("escrow_%s.json", sanitizeForFile(escrowID)))
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("escrow %s not found in cache", escrowID)
		}
		return nil, err
	}
	var info map[string]interface{}
	if err := json.Unmarshal(b, &info); err != nil {
		return nil, fmt.Errorf("failed to parse escrow info: %w", err)
	}
	return info, nil
}

// GetEscrowHistory returns all cached escrow infos.
func (pc *PokerClient) GetEscrowHistory() ([]map[string]interface{}, error) {
	dir := pc.historyDir()
	if dir == "" {
		return nil, fmt.Errorf("no data directory configured")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var entriesWithHeight []struct {
		info   map[string]interface{}
		height uint64
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), "escrow_") || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var info map[string]interface{}
		if err := json.Unmarshal(b, &info); err == nil {
			height := extractConfirmedHeight(info)
			if height == 0 {
				continue
			}
			entriesWithHeight = append(entriesWithHeight, struct {
				info   map[string]interface{}
				height uint64
			}{info: info, height: height})
		}
	}
	sort.Slice(entriesWithHeight, func(i, j int) bool {
		return entriesWithHeight[i].height > entriesWithHeight[j].height
	})
	out := make([]map[string]interface{}, 0, len(entriesWithHeight))
	for _, it := range entriesWithHeight {
		out = append(out, it.info)
	}
	return out, nil
}

// UpdateEscrowHistory merges the provided escrow info into the cached history
// file for that escrow ID (if present). This allows UI flows to adjust funding
// hints and status for refund tooling.
func (pc *PokerClient) UpdateEscrowHistory(info *EscrowInfo) error {
	if info == nil || strings.TrimSpace(info.EscrowID) == "" {
		return fmt.Errorf("escrow info requires escrow_id")
	}
	dir := pc.historyDir()
	if dir == "" {
		return fmt.Errorf("no data directory configured")
	}
	path := filepath.Join(dir, fmt.Sprintf("escrow_%s.json", sanitizeForFile(info.EscrowID)))
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var cur EscrowInfo
	if err := json.Unmarshal(b, &cur); err != nil {
		return err
	}
	mergeEscrowInfo(&cur, info)
	blob, err := json.MarshalIndent(&cur, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, blob, 0600)
}

// DeleteEscrowHistory removes the cached escrow history entry for the given id.
// This only affects local refund metadata; it does not touch server-side state.
func (pc *PokerClient) DeleteEscrowHistory(escrowID string) error {
	escrowID = strings.TrimSpace(escrowID)
	if escrowID == "" {
		return fmt.Errorf("escrow_id required")
	}
	dir := pc.historyDir()
	if dir == "" {
		return fmt.Errorf("no data directory configured")
	}
	path := filepath.Join(dir, fmt.Sprintf("escrow_%s.json", sanitizeForFile(escrowID)))
	if err := os.Remove(path); err != nil {
		return err
	}
	return nil
}

// GetBindableEscrows returns escrows that are currently bindable for play.
// It filters cached escrows using the referee's funding state and sorts them
// by confirmed_height descending (newest confirmations first).
func (pc *PokerClient) GetBindableEscrows(ctx context.Context, token string) ([]map[string]interface{}, error) {
	dir := pc.historyDir()
	if dir == "" {
		return nil, fmt.Errorf("no data directory configured")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	type candidate struct {
		escrowID string
		info     map[string]interface{}
		height   uint64
	}
	var cands []candidate

	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), "escrow_") || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var info map[string]interface{}
		if err := json.Unmarshal(b, &info); err != nil {
			continue
		}

		// Extract escrow_id (top-level or under escrow_info).
		var idRaw any
		if v, ok := info["escrow_id"]; ok {
			idRaw = v
		} else if escRaw, ok := info["escrow_info"]; ok {
			if m, ok := escRaw.(map[string]interface{}); ok {
				idRaw = m["escrow_id"]
			}
		}
		escrowID := strings.TrimSpace(fmt.Sprint(idRaw))
		if escrowID == "" {
			continue
		}

		// Require a funding outpoint; unfunded escrows are not bindable.
		// Check nested escrow_info first for compatibility.
		var txid string
		var vout any
		if escRaw, ok := info["escrow_info"]; ok {
			if m, ok := escRaw.(map[string]interface{}); ok {
				if v, ok := m["funding_txid"]; ok {
					txid = strings.TrimSpace(fmt.Sprint(v))
				}
				if v2, ok := m["funding_vout"]; ok {
					vout = v2
				}
			}
		}
		if txid == "" {
			if v, ok := info["funding_txid"]; ok {
				txid = strings.TrimSpace(fmt.Sprint(v))
			}
		}
		if vout == nil {
			if v2, ok := info["funding_vout"]; ok {
				vout = v2
			}
		}
		if txid == "" || vout == nil {
			continue
		}

		height := extractConfirmedHeight(info)
		cands = append(cands, candidate{
			escrowID: escrowID,
			info:     info,
			height:   height,
		})
	}

	if len(cands) == 0 {
		return nil, nil
	}

	ref := pc.Referee(token)
	type result struct {
		info   map[string]interface{}
		height uint64
	}
	var out []result

	for _, c := range cands {
		if ctx != nil && ctx.Err() != nil {
			return nil, ctx.Err()
		}
		resp, err := ref.GetEscrowStatus(ctx, c.escrowID)
		if err != nil {
			// Skip escrows the referee no longer knows about.
			continue
		}
		// Only escrows with a single live UTXO and not CSV-matured are bindable.
		if resp.GetUtxoCount() != 1 {
			continue
		}
		if resp.GetMatureForCsv() {
			continue
		}
		if !resp.GetOk() {
			continue
		}

		// Merge cached info with live status fields.
		m := make(map[string]interface{}, len(c.info)+8)
		for k, v := range c.info {
			m[k] = v
		}
		m["escrow_id"] = resp.GetEscrowId()
		m["confs"] = resp.GetConfs()
		m["utxo_count"] = resp.GetUtxoCount()
		m["mature_for_csv"] = resp.GetMatureForCsv()
		m["required_confirmations"] = resp.GetRequiredConfirmations()
		if tx := resp.GetFundingTxid(); tx != "" {
			m["funding_txid"] = tx
		}
		if v := resp.GetFundingVout(); v != 0 {
			m["funding_vout"] = v
		}
		if amt := resp.GetAmountAtoms(); amt != 0 {
			m["funded_amount"] = amt
		}

		out = append(out, result{info: m, height: c.height})
	}

	if len(out) == 0 {
		return nil, nil
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].height == out[j].height {
			// Tie-breaker: escrow_id for deterministic ordering.
			var ei, ej string
			if id, ok := out[i].info["escrow_id"].(string); ok {
				ei = id
			}
			if id, ok := out[j].info["escrow_id"].(string); ok {
				ej = id
			}
			return ei < ej
		}
		return out[i].height > out[j].height
	})

	res := make([]map[string]interface{}, 0, len(out))
	for _, it := range out {
		res = append(res, it.info)
	}
	return res, nil
}

// extractConfirmedHeight returns the confirmed block height (or conf count) if present.
func extractConfirmedHeight(info map[string]interface{}) uint64 {
	if info == nil {
		return 0
	}
	raw := info["confirmed_height"]
	if escRaw, ok := info["escrow_info"]; ok {
		if m, ok := escRaw.(map[string]interface{}); ok {
			raw = m["confirmed_height"]
		}
	}
	switch v := raw.(type) {
	case float64:
		if v > 0 {
			return uint64(v)
		}
	case json.Number:
		if n, err := v.Int64(); err == nil && n > 0 {
			return uint64(n)
		}
	case int:
		if v > 0 {
			return uint64(v)
		}
	case int64:
		if v > 0 {
			return uint64(v)
		}
	case uint64:
		return v
	}
	return 0
}
