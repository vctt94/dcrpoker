package golib

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/companyzero/bisonrelay/client/clientintf"
	"github.com/companyzero/bisonrelay/clientrpc/types"
	"github.com/companyzero/bisonrelay/lockfile"
	"github.com/vctt94/pokerbisonrelay/pkg/poker"
)

const (
	appName = "pongui"
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

func handleClientCmd(cc *clientCtx, cmd *cmd) (interface{}, error) {
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
		cc.log.Infof("GenerateSessionKey called (stub implementation)")
		// Stub implementation - return dummy keys
		return map[string]string{"priv": "stub-private-key", "pub": "stub-public-key"}, nil

	case CTOpenEscrow:
		if es == nil {
			// Initialize with dummy data for demo purposes
			es = &escrowState{
				EscrowId:       "demo-escrow-123",
				DepositAddress: "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
				PkScriptHex:    "76a91462e907b15cbf27d5425399ebf6f0fb50ebb88e1888ac",
			}
		}
		return map[string]any{
			"escrow_id":       es.EscrowId,
			"deposit_address": es.DepositAddress,
			"pk_script_hex":   es.PkScriptHex,
		}, nil

	case CTStartPreSign:
		{
			var req preSignReq
			if err := decodeStrict(cmd.Payload, &req); err != nil {
				return nil, fmt.Errorf("presign payload: %w", err)
			}
			cc.log.Infof("start presign match_id=%q (stub implementation)", req.MatchID)
			return map[string]string{"status": "ok"}, nil
		}

	case CTArchiveSessionKey:
		{
			var req struct {
				MatchID string `json:"match_id"`
			}
			if err := decodeStrict(cmd.Payload, &req); err != nil {
				return nil, fmt.Errorf("archive payload: %w", err)
			}
			if req.MatchID == "" {
				return nil, fmt.Errorf("archive: empty match_id")
			}
			cc.log.Infof("ArchiveSessionKey called: matchID=%s (stub implementation)", req.MatchID)
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

		notify(NTLogLine, fmt.Sprintf("CTJoinPokerTable ok: player=%s table=%s", cc.ID.String(), req.TableID), nil)
		return map[string]string{"status": "joined", "table_id": req.TableID}, nil

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
			SmallBlind:     req.SmallBlind,
			BigBlind:       req.BigBlind,
			MaxPlayers:     int(req.MaxPlayers),
			MinPlayers:     int(req.MinPlayers),
			MinBalance:     req.MinBalance,
			BuyIn:          req.BuyIn,
			StartingChips:  req.StartingChips,
			TimeBank:       time.Duration(req.TimeBankSeconds) * time.Second,
			AutoStartDelay: time.Duration(req.AutoStartMs) * time.Millisecond,
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

	case CTGetPokerBalance:
		if cc.c == nil {
			return nil, fmt.Errorf("poker client not initialized")
		}
		balance, err := cc.c.GetBalance(cc.ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get balance: %v", err)
		}
		return map[string]int64{"balance": balance}, nil

	case CTGetPlayerCurrentTable:
		if cc.c == nil {
			return nil, fmt.Errorf("poker client not initialized")
		}
		tid, err := cc.c.GetPlayerCurrentTable(cc.ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get current table: %v", err)
		}
		return map[string]string{"table_id": tid}, nil

	default:
		return nil, fmt.Errorf("unknown cmd 0x%x", cmd.Type)
	}
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
