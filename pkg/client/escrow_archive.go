package client

import (
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
