package server

import (
	"context"

	"github.com/decred/slog"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
	"github.com/vctt94/pokerbisonrelay/pkg/statemachine"
)

type shutdownCoordinator struct {
	log slog.Logger
	sm  *statemachine.Machine[shutdownCoordinator]

	drainDone     chan struct{}
	pendingTables map[string]struct{}
}

type evShutdownBeginDrain struct {
	tableIDs []string
}

type evShutdownTableDrainStarted struct {
	tableID    string
	activeHand bool
}

type evShutdownHandFinished struct {
	tableID string
	reason  string
}

type evShutdownTableRemoved struct {
	tableID string
}

type evShutdownAwaitQuiescence struct {
	reply chan<- chan struct{}
}

type evShutdownStop struct{}

func newShutdownCoordinator(log slog.Logger) *shutdownCoordinator {
	c := &shutdownCoordinator{log: log}
	c.sm = statemachine.New(c, shutdownStateIdle, 64)
	c.sm.Start(context.Background())
	return c
}

func (c *shutdownCoordinator) BeginDrain(tableIDs []string) {
	if c == nil || c.sm == nil {
		return
	}
	c.sm.Send(evShutdownBeginDrain{tableIDs: append([]string(nil), tableIDs...)})
}

func (c *shutdownCoordinator) NotifyDrainStarted(tableID string, activeHand bool) {
	if c == nil || c.sm == nil || tableID == "" {
		return
	}
	c.sm.Send(evShutdownTableDrainStarted{tableID: tableID, activeHand: activeHand})
}

func (c *shutdownCoordinator) NotifyHandFinished(tableID string, reason string) {
	if c == nil || c.sm == nil || tableID == "" {
		return
	}
	c.sm.Send(evShutdownHandFinished{tableID: tableID, reason: reason})
}

func (c *shutdownCoordinator) NotifyTableRemoved(tableID string) {
	if c == nil || c.sm == nil || tableID == "" {
		return
	}
	c.sm.Send(evShutdownTableRemoved{tableID: tableID})
}

func (c *shutdownCoordinator) WaitForQuiescence() {
	if c == nil || c.sm == nil {
		return
	}

	reply := make(chan chan struct{}, 1)
	c.sm.Send(evShutdownAwaitQuiescence{reply: reply})
	done := <-reply
	<-done
}

func (c *shutdownCoordinator) Stop() {
	if c == nil || c.sm == nil {
		return
	}
	c.sm.TrySend(evShutdownStop{})
	c.sm.Stop()
}

func shutdownStateIdle(c *shutdownCoordinator, in <-chan any) statemachine.StateFn[shutdownCoordinator] {
	for ev := range in {
		switch e := ev.(type) {
		case evShutdownBeginDrain:
			c.drainDone = make(chan struct{})
			c.pendingTables = make(map[string]struct{}, len(e.tableIDs))
			for _, tableID := range e.tableIDs {
				if tableID == "" {
					continue
				}
				c.pendingTables[tableID] = struct{}{}
			}
			if c.allTablesQuiesced() {
				c.closeDrainDone()
				return shutdownStateQuiesced
			}
			return shutdownStateDraining
		case evShutdownAwaitQuiescence:
			e.reply <- closedSignal()
		case evShutdownStop:
			return nil
		}
	}
	return nil
}

func shutdownStateDraining(c *shutdownCoordinator, in <-chan any) statemachine.StateFn[shutdownCoordinator] {
	for ev := range in {
		switch e := ev.(type) {
		case evShutdownBeginDrain:
			// Drain session already active; keep the existing waiter.
		case evShutdownTableDrainStarted:
			if !e.activeHand {
				c.log.Debugf("shutdown drain: table %s already quiesced at drain start", e.tableID)
				delete(c.pendingTables, e.tableID)
			}
		case evShutdownHandFinished:
			c.log.Debugf("shutdown drain: table %s quiesced after %s", e.tableID, e.reason)
			delete(c.pendingTables, e.tableID)
		case evShutdownTableRemoved:
			delete(c.pendingTables, e.tableID)
			c.log.Debugf("shutdown drain: removed table %s from coordinator", e.tableID)
		case evShutdownAwaitQuiescence:
			e.reply <- c.drainDone
			continue
		case evShutdownStop:
			c.closeDrainDone()
			return nil
		}

		if c.allTablesQuiesced() {
			c.closeDrainDone()
			return shutdownStateQuiesced
		}
	}
	c.closeDrainDone()
	return nil
}

func shutdownStateQuiesced(c *shutdownCoordinator, in <-chan any) statemachine.StateFn[shutdownCoordinator] {
	for ev := range in {
		switch e := ev.(type) {
		case evShutdownAwaitQuiescence:
			e.reply <- closedSignal()
		case evShutdownStop:
			return nil
		}
	}
	return nil
}

func (c *shutdownCoordinator) allTablesQuiesced() bool {
	if c == nil {
		return true
	}
	return len(c.pendingTables) == 0
}

func (c *shutdownCoordinator) closeDrainDone() {
	if c == nil || c.drainDone == nil {
		return
	}
	select {
	case <-c.drainDone:
	default:
		close(c.drainDone)
	}
}

func closedSignal() chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}

func tableSnapshotHasActiveHand(snapshot *TableSnapshot) bool {
	if snapshot == nil || snapshot.GameSnapshot == nil {
		return false
	}
	switch snapshot.GameSnapshot.Phase {
	case pokerrpc.GamePhase_PRE_FLOP,
		pokerrpc.GamePhase_FLOP,
		pokerrpc.GamePhase_TURN,
		pokerrpc.GamePhase_RIVER:
		return true
	default:
		return false
	}
}

func (s *Server) observeShutdownEvent(event *GameEvent) {
	if s == nil || event == nil || s.shutdownCoordinator == nil || !s.IsDraining() {
		return
	}

	switch event.Type {
	case pokerrpc.NotificationType_SHOWDOWN_RESULT, pokerrpc.NotificationType_GAME_ENDED:
		if tableSnapshotHasActiveHand(event.TableSnapshot) {
			return
		}
		s.shutdownCoordinator.NotifyHandFinished(event.TableID, string(event.Type))
	case pokerrpc.NotificationType_TABLE_REMOVED:
		s.shutdownCoordinator.NotifyTableRemoved(event.TableID)
	}
}
