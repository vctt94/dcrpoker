package golib

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/companyzero/bisonrelay/client/clientintf"
	"github.com/companyzero/bisonrelay/clientrpc/types"
	"github.com/companyzero/bisonrelay/lockfile"
	"github.com/vctt94/pokerbisonrelay/pkg/poker"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	appName = "bisonpoker"
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
			TableId: tableID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get game state: %v", err)
		}
		// Convert protobuf to JSON using protojson
		jsonBytes, err := protojson.Marshal(resp.GameState)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal game state: %v", err)
		}
		var gameStateMap map[string]interface{}
		if err := json.Unmarshal(jsonBytes, &gameStateMap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal game state JSON: %v", err)
		}
		return map[string]interface{}{
			"game_state": gameStateMap,
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
