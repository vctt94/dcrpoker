package client

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/vctt94/pokerbisonrelay/pkg/poker"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
)

// StartGameStream starts receiving real-time game updates for the current table
func (pc *PokerClient) StartGameStream(ctx context.Context) error {
	if ctx.Err() != nil {
		if ctx.Err() == context.Canceled {
			return nil
		}
		return ctx.Err()
	}

	currentTableID := pc.GetCurrentTableID()
	if currentTableID == "" {
		return fmt.Errorf("not currently at a table")
	}

	pc.gameStreamMu.Lock()
	// If already streaming for this table, nothing to do.
	if pc.gameStreamCancel != nil && pc.gameStreamTable == currentTableID {
		pc.gameStreamMu.Unlock()
		return nil
	}

	// Cancel any prior loop before starting a new one.
	if pc.gameStreamCancel != nil {
		pc.gameStreamCancel()
	}

	loopCtx, cancel := context.WithCancel(pc.ctx)
	pc.gameStreamCancel = cancel
	pc.gameStreamCtx = loopCtx
	pc.gameStreamTable = currentTableID
	pc.gameStream = nil
	pc.gameStreamMu.Unlock()

	go pc.runGameStreamLoop(loopCtx, currentTableID)

	pc.log.Infof("Started game stream loop for table %s", currentTableID)
	return nil
}

func (pc *PokerClient) runGameStreamLoop(ctx context.Context, tableID string) {
	backoff := 500 * time.Millisecond
	const maxBackoff = 30 * time.Second

	defer func() {
		pc.gameStreamMu.Lock()
		pc.gameStream = nil
		pc.gameStreamMu.Unlock()
		pc.setGameStreamConnectionState(false, ctx.Err())
	}()

	for {
		if ctx.Err() != nil {
			return
		}

		if current := pc.GetCurrentTableID(); current == "" || current != tableID {
			return
		}

		stream, err := pc.PokerService.StartGameStream(ctx, &pokerrpc.StartGameStreamRequest{
			PlayerId: pc.ID.String(),
			TableId:  tableID,
		})
		if err != nil {
			if ctx.Err() != nil {
				return
			}

			pc.enqueueError(fmt.Errorf("failed to start game stream: %w", err))
			if !waitWithBackoff(ctx, backoff) {
				return
			}
			backoff = capBackoff(backoff, maxBackoff)
			continue
		}

		pc.gameStreamMu.Lock()
		pc.gameStream = stream
		pc.gameStreamMu.Unlock()

		pc.setGameStreamConnectionState(true, nil)
		backoff = 500 * time.Millisecond

		if err := pc.consumeGameStream(ctx, stream, tableID); err != nil {
			if ctx.Err() != nil {
				return
			}

			pc.gameStreamMu.Lock()
			pc.gameStream = nil
			pc.gameStreamMu.Unlock()

			pc.setGameStreamConnectionState(false, err)
			if !waitWithBackoff(ctx, backoff) {
				return
			}
			backoff = capBackoff(backoff, maxBackoff)
			continue
		}

		return
	}
}

// CreateTable creates a new poker table using poker.TableConfig
func (pc *PokerClient) CreateTable(ctx context.Context, config poker.TableConfig) (string, error) {
	// Convert poker.TableConfig to RPC CreateTableRequest
	timeBankSeconds := int32(config.TimeBank.Seconds())
	ctx = pc.withSessionToken(ctx)
	resp, err := pc.LobbyService.CreateTable(ctx, &pokerrpc.CreateTableRequest{
		PlayerId:        pc.ID.String(),
		SmallBlind:      config.SmallBlind,
		BigBlind:        config.BigBlind,
		MaxPlayers:      int32(config.MaxPlayers),
		MinPlayers:      int32(config.MinPlayers),
		MinBalance:      config.MinBalance,
		BuyIn:           config.BuyIn,
		StartingChips:   config.StartingChips,
		TimeBankSeconds: timeBankSeconds,
		AutoStartMs:     int32(config.AutoStartDelay.Milliseconds()),
		AutoAdvanceMs:   int32(config.AutoAdvanceDelay.Milliseconds()),
	})
	if err != nil {
		return "", err
	}

	// Surface server-side validation errors (only message field available now).
	if resp.GetTableId() == "" {
		msg := resp.GetMessage()
		if msg == "" {
			msg = "unknown error"
		}
		return "", fmt.Errorf("failed to create table: %s", msg)
	}

	pc.SetCurrentTableID(resp.TableId)

	// Note: Game stream is not automatically started for table creation
	// It should be started explicitly when needed for real-time updates

	return resp.TableId, nil
}

// JoinTable joins an existing poker table and tracks the table ID
func (pc *PokerClient) JoinTable(ctx context.Context, tableID string) error {
	ctx = pc.withSessionToken(ctx)
	resp, err := pc.LobbyService.JoinTable(ctx, &pokerrpc.JoinTableRequest{
		PlayerId: pc.ID.String(),
		TableId:  tableID,
	})
	if err != nil {
		return err
	}

	if !resp.Success {
		msg := resp.GetMessage()
		if msg == "" {
			msg = "unknown error"
		}
		return fmt.Errorf("failed to join table: %s", msg)
	}

	pc.SetCurrentTableID(tableID)

	return nil
}

// LeaveTable leaves the current table and clears the table ID
func (pc *PokerClient) LeaveTable(ctx context.Context) error {
	tableID := pc.GetCurrentTableID()

	if tableID == "" {
		return fmt.Errorf("not currently in a table")
	}

	// Stop game stream first
	pc.stopGameStream()

	ctx = pc.withSessionToken(ctx)
	resp, err := pc.LobbyService.LeaveTable(ctx, &pokerrpc.LeaveTableRequest{
		PlayerId: pc.ID.String(),
		TableId:  tableID,
	})
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("failed to leave table: %s", resp.Message)
	}

	pc.SetCurrentTableID("")

	return nil
}

// GetTables returns all available tables
func (pc *PokerClient) GetTables(ctx context.Context) ([]*pokerrpc.Table, error) {
	resp, err := pc.LobbyService.GetTables(ctx, &pokerrpc.GetTablesRequest{})
	if err != nil {
		return nil, err
	}
	return resp.Tables, nil
}

// GetPlayerCurrentTable returns the current table for the player
func (pc *PokerClient) GetPlayerCurrentTable(ctx context.Context) (string, error) {
	resp, err := pc.LobbyService.GetPlayerCurrentTable(ctx, &pokerrpc.GetPlayerCurrentTableRequest{
		PlayerId: pc.ID.String(),
	})
	if err != nil {
		return "", err
	}
	return resp.TableId, nil
}

// SetPlayerReady sets the player ready status
func (pc *PokerClient) SetPlayerReady(ctx context.Context) error {
	tableID := pc.GetCurrentTableID()

	if tableID == "" {
		return fmt.Errorf("not currently in a table")
	}

	ctx = pc.withSessionToken(ctx)
	resp, err := pc.LobbyService.SetPlayerReady(ctx, &pokerrpc.SetPlayerReadyRequest{
		PlayerId: pc.ID.String(),
		TableId:  tableID,
	})
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("failed to set ready: %s", resp.Message)
	}

	return nil
}

// SetPlayerUnready sets the player unready status
func (pc *PokerClient) SetPlayerUnready(ctx context.Context) error {
	tableID := pc.GetCurrentTableID()

	if tableID == "" {
		return fmt.Errorf("not currently in a table")
	}

	ctx = pc.withSessionToken(ctx)
	resp, err := pc.LobbyService.SetPlayerUnready(ctx, &pokerrpc.SetPlayerUnreadyRequest{
		PlayerId: pc.ID.String(),
		TableId:  tableID,
	})
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("failed to set unready: %s", resp.Message)
	}

	return nil
}

// StartNotifier starts the notification stream to receive server notifications
func (pc *PokerClient) StartNotificationStream(ctx context.Context) error {
	if err := pc.validate(); err != nil {
		return fmt.Errorf("cannot start notifier: %v", err)
	}

	pc.ntfnLoopMu.Lock()
	if pc.ntfnLoopRunning {
		pc.ntfnLoopMu.Unlock()
		return nil
	}
	pc.ntfnLoopRunning = true
	pc.ntfnLoopMu.Unlock()

	go pc.runNotificationLoop(ctx)

	return nil
}

func (pc *PokerClient) runNotificationLoop(ctx context.Context) {
	defer func() {
		pc.ntfnLoopMu.Lock()
		pc.ntfnLoopRunning = false
		pc.ntfnLoopMu.Unlock()
		pc.setConnectionState(false, ctx.Err())
	}()

	backoff := 500 * time.Millisecond
	const maxBackoff = 30 * time.Second

	for {
		if ctx.Err() != nil {
			return
		}

		stream, err := pc.startNotificationStream(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			pc.setConnectionState(false, err)
			if !waitWithBackoff(ctx, backoff) {
				return
			}
			backoff = capBackoff(backoff, maxBackoff)
			continue
		}

		pc.notifier = stream
		pc.setConnectionState(true, nil)

		if err := pc.reSyncStateAfterReconnect(ctx); err != nil && ctx.Err() == nil {
			pc.log.Warnf("state re-sync after reconnect failed: %v", err)
		}

		backoff = 500 * time.Millisecond
		if err := pc.consumeNotificationStream(ctx, stream); err != nil {
			if ctx.Err() != nil {
				return
			}

			pc.setConnectionState(false, err)
			if !waitWithBackoff(ctx, backoff) {
				return
			}
			backoff = capBackoff(backoff, maxBackoff)
			continue
		}

		return
	}
}

func (pc *PokerClient) startNotificationStream(ctx context.Context) (pokerrpc.LobbyService_StartNotificationStreamClient, error) {
	return pc.LobbyService.StartNotificationStream(ctx, &pokerrpc.StartNotificationStreamRequest{
		PlayerId: pc.ID.String(),
	})
}

func (pc *PokerClient) consumeNotificationStream(ctx context.Context, stream pokerrpc.LobbyService_StartNotificationStreamClient) error {
	for {
		ntfn, err := stream.Recv()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if isTransportClosing(err) {
				return err
			}

			pc.enqueueError(fmt.Errorf("notification stream error: %v", err))
			return err
		}

		pc.handleNotification(ctx, ntfn)
	}
}

func (pc *PokerClient) handleNotification(ctx context.Context, ntfn *pokerrpc.Notification) {
	if ntfn == nil {
		pc.log.Debug("received nil notification")
		return
	}

	if pc.ntfns == nil {
		pc.log.Error("notification manager is nil, skipping notification handling")
		pc.enqueueUpdate(ntfn)
		return
	}

	ts := time.Now()
	switch ntfn.Type {
	case pokerrpc.NotificationType_TABLE_CREATED:
		if ntfn.Table != nil {
			pc.ntfns.notifyTableCreated(ntfn.Table, ts)
		}

	case pokerrpc.NotificationType_TABLE_REMOVED:
		// Nothing additional; UI will refresh tables on receipt.

	case pokerrpc.NotificationType_PLAYER_JOINED:
		if ntfn.Table != nil {
			pc.ntfns.notifyPlayerJoined(ntfn.Table, ntfn.PlayerId, ts)
		}

	case pokerrpc.NotificationType_PLAYER_LEFT:
		if ntfn.Table != nil {
			pc.ntfns.notifyPlayerLeft(ntfn.Table, ntfn.PlayerId, ts)
		}

	case pokerrpc.NotificationType_GAME_STARTED:
		if ntfn.Started {
			pc.ntfns.notifyGameStarted(ntfn.TableId, ts)
			if current := pc.GetCurrentTableID(); current == "" {
				pc.SetCurrentTableID(ntfn.TableId)
			}
			if err := pc.StartGameStream(ctx); err != nil {
				err = fmt.Errorf("failed to start game stream: %w", err)
				pc.log.Error(err)
				pc.enqueueError(err)
			}
			pc.log.Infof("Game started for table %s", ntfn.TableId)
		}

	case pokerrpc.NotificationType_NEW_HAND_STARTED:
		pc.log.Debugf("New hand started for table %s", ntfn.TableId)

	case pokerrpc.NotificationType_GAME_ENDED:
		pc.ntfns.notifyGameEnded(ntfn.TableId, ntfn.Message, ts)
		pc.log.Info(ntfn.Message)

	case pokerrpc.NotificationType_BET_MADE:
		pc.ntfns.notifyBetMade(ntfn.PlayerId, ntfn.Amount, ts)
		if ntfn.PlayerId == pc.ID.String() {
			pc.Lock()
			pc.BetAmt = ntfn.Amount
			pc.Unlock()
		}

		if strings.Contains(ntfn.Message, "called") {
			pc.ntfns.notifyPlayerCalled(ntfn.PlayerId, ntfn.Amount, ts)
		} else if strings.Contains(ntfn.Message, "raised") {
			pc.ntfns.notifyPlayerRaised(ntfn.PlayerId, ntfn.Amount, ts)
		} else if strings.Contains(ntfn.Message, "checked") {
			pc.ntfns.notifyPlayerChecked(ntfn.PlayerId, ts)
		}

	case pokerrpc.NotificationType_PLAYER_FOLDED:
		pc.ntfns.notifyPlayerFolded(ntfn.PlayerId, ts)

	case pokerrpc.NotificationType_PLAYER_READY:
		pc.ntfns.notifyPlayerReady(ntfn.PlayerId, ntfn.Ready, ts)
		if ntfn.PlayerId == pc.ID.String() {
			pc.Lock()
			pc.IsReady = ntfn.Ready
			pc.Unlock()
		}

	case pokerrpc.NotificationType_PLAYER_UNREADY:
		pc.ntfns.notifyPlayerReady(ntfn.PlayerId, false, ts)
		if ntfn.PlayerId == pc.ID.String() {
			pc.Lock()
			pc.IsReady = false
			pc.Unlock()
		}

	case pokerrpc.NotificationType_ALL_PLAYERS_READY:
		// UI can transition states on receipt.

	case pokerrpc.NotificationType_BALANCE_UPDATED:
		pc.ntfns.notifyBalanceUpdated(ntfn.PlayerId, ntfn.NewBalance, ts)

	case pokerrpc.NotificationType_TIP_RECEIVED:
		fromID := ntfn.PlayerId
		toID := pc.ID
		amount := ntfn.Amount
		message := ntfn.Message
		pc.ntfns.notifyTipReceived(fromID, toID.String(), amount, message, ts)

	case pokerrpc.NotificationType_SHOWDOWN_RESULT:
		pc.ntfns.notifyShowdownResult(ntfn.TableId, ntfn.Winners, ts)

	case pokerrpc.NotificationType_NEW_ROUND:
		// UI will fetch new state on receipt.

	case pokerrpc.NotificationType_SMALL_BLIND_POSTED:
		pc.ntfns.notifyBetMade(ntfn.PlayerId, ntfn.Amount, ts)
		pc.log.Infof("Small blind posted: %d chips by %s", ntfn.Amount, ntfn.PlayerId)

	case pokerrpc.NotificationType_BIG_BLIND_POSTED:
		pc.ntfns.notifyBetMade(ntfn.PlayerId, ntfn.Amount, ts)
		pc.log.Infof("Big blind posted: %d chips by %s", ntfn.Amount, ntfn.PlayerId)

	case pokerrpc.NotificationType_CALL_MADE:
		pc.ntfns.notifyPlayerCalled(ntfn.PlayerId, ntfn.Amount, ts)
		pc.ntfns.notifyBetMade(ntfn.PlayerId, ntfn.Amount, ts)
		pc.log.Debugf("Player %s called %d", ntfn.PlayerId, ntfn.Amount)

	case pokerrpc.NotificationType_CHECK_MADE:
		pc.ntfns.notifyPlayerChecked(ntfn.PlayerId, ts)
		pc.log.Debugf("Player %s checked", ntfn.PlayerId)

	case pokerrpc.NotificationType_CARDS_SHOWN:
		pc.log.Debugf("Player %s showed cards", ntfn.PlayerId)

	case pokerrpc.NotificationType_CARDS_HIDDEN:
		pc.log.Debugf("Player %s hid cards", ntfn.PlayerId)

	case pokerrpc.NotificationType_PLAYER_ALL_IN:
		pc.log.Infof("Player %s is all-in with amount %d", ntfn.PlayerId, ntfn.Amount)

	case pokerrpc.NotificationType_ESCROW_FUNDING:
		pc.log.Infof("Escrow funding: %s", ntfn.Message)

	default:
		pc.log.Debug("received unknown notification type", "type", ntfn.Type)
	}

	// Enqueue for golib handler to update escrow cache with confirmed_height
	pc.enqueueUpdate(ntfn)
}

// reSyncStateAfterReconnect refreshes client-side state after a notification stream reconnect.
func (pc *PokerClient) reSyncStateAfterReconnect(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if _, err := pc.GetTables(ctx); err != nil && ctx.Err() == nil {
		pc.log.Warnf("failed to refresh tables after reconnect: %v", err)
	}

	gameUpdate, err := pc.reSyncTableState(ctx)
	if err != nil {
		return err
	}

	if err := pc.restoreGameStreamIfNeeded(ctx, gameUpdate); err != nil {
		return err
	}

	return nil
}

// reSyncTableState restores the active table ID and emits the latest game state snapshot.
func (pc *PokerClient) reSyncTableState(ctx context.Context) (*pokerrpc.GameUpdate, error) {
	tableID, err := pc.GetPlayerCurrentTable(ctx)
	if err != nil {
		return nil, fmt.Errorf("get current table: %w", err)
	}

	if tableID == "" {
		pc.SetCurrentTableID("")
		pc.Lock()
		pc.IsReady = false
		pc.Unlock()
		return nil, nil
	}

	pc.SetCurrentTableID(tableID)

	resp, err := pc.PokerService.GetGameState(ctx, &pokerrpc.GetGameStateRequest{
		TableId:  tableID,
		PlayerId: pc.ID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("get game state: %w", err)
	}

	gameUpdate := resp.GetGameState()
	if gameUpdate != nil {
		pc.syncReadyStatusFromGameUpdate(gameUpdate)
		pc.enqueueUpdate(gameUpdate)
	}

	return gameUpdate, nil
}

func (pc *PokerClient) syncReadyStatusFromGameUpdate(update *pokerrpc.GameUpdate) {
	if update == nil {
		return
	}

	for _, player := range update.Players {
		if player.Id == pc.ID.String() {
			pc.Lock()
			pc.IsReady = player.IsReady
			pc.Unlock()
			return
		}
	}
}

func (pc *PokerClient) restoreGameStreamIfNeeded(ctx context.Context, update *pokerrpc.GameUpdate) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if pc.GetCurrentTableID() == "" {
		return nil
	}

	if err := pc.StartGameStream(ctx); err != nil {
		return fmt.Errorf("restore game stream: %w", err)
	}

	// Game updates are already enqueued via reSyncTableState.
	return nil
}
