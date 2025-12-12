package server

import (
	"sync"
	"time"

	"github.com/decred/slog"
	"github.com/vctt94/pokerbisonrelay/pkg/poker"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
)

// GameEventType is an alias for the RPC NotificationType to maintain compatibility
type GameEventType = pokerrpc.NotificationType

// GameEvent represents an immutable snapshot of a game event
type GameEvent struct {
	Type          GameEventType
	TableID       string
	PlayerIDs     []string // All players who should receive updates
	Amount        int64
	Payload       EventPayload
	Timestamp     time.Time
	TableSnapshot *TableSnapshot
}

// TableSnapshot represents an immutable snapshot of table state
type TableSnapshot struct {
	ID           string
	Players      []*PlayerSnapshot
	GameSnapshot *poker.GameStateSnapshot
	LastShowdown *poker.ShowdownResult
	Config       poker.TableConfig
	State        TableState
	Timestamp    time.Time
}

func (ts *TableSnapshot) playerIDs() []string {
	if ts == nil {
		return nil
	}
	ids := make([]string, 0, len(ts.Players))
	for _, ps := range ts.Players {
		if ps != nil && ps.ID != "" {
			ids = append(ids, ps.ID)
		}
	}
	return ids
}

// PlayerSnapshot represents an immutable snapshot of player state
type PlayerSnapshot struct {
	ID              string
	Name            string
	TableSeat       int
	Balance         int64
	Hand            []poker.Card
	IsReady         bool
	EscrowID        string
	EscrowReady     bool
	PresignComplete bool
	IsDisconnected  bool
	HasFolded       bool
	IsAllIn         bool
	IsDealer        bool
	IsSmallBlind    bool
	IsBigBlind      bool
	IsTurn          bool
	GameState       string
	HandDescription string
	HasBet          int64
	StartingBalance int64
	LastAction      time.Time
}

// TableState represents table-level state
type TableState struct {
	GameStarted     bool
	AllPlayersReady bool
	PlayerCount     int
}

// EventProcessor manages the processing of game events
type EventProcessor struct {
	server   *Server
	log      slog.Logger
	queue    chan *GameEvent
	workers  []*eventWorker
	stopChan chan struct{}
	wg       sync.WaitGroup
	started  bool
	mu       sync.Mutex
}

// eventWorker processes events from the queue
type eventWorker struct {
	id        int
	processor *EventProcessor
	stopChan  chan struct{}
	wg        *sync.WaitGroup
}

// NewEventProcessor creates a new event processor
func NewEventProcessor(server *Server, queueSize, workerCount int) *EventProcessor {
	processor := &EventProcessor{
		server:   server,
		log:      server.log,
		queue:    make(chan *GameEvent, queueSize),
		stopChan: make(chan struct{}),
	}

	// Create workers
	processor.workers = make([]*eventWorker, workerCount)
	for i := 0; i < workerCount; i++ {
		processor.workers[i] = &eventWorker{
			id:        i,
			processor: processor,
			stopChan:  make(chan struct{}),
			wg:        &processor.wg,
		}
	}

	return processor
}

// Start begins processing events
func (ep *EventProcessor) Start() {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	if ep.started {
		return
	}

	ep.started = true
	ep.log.Infof("Starting event processor with %d workers", len(ep.workers))

	// Start all workers
	for _, worker := range ep.workers {
		ep.wg.Add(1)
		go worker.run()
	}
}

// Stop gracefully stops the event processor
func (ep *EventProcessor) Stop() {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	if !ep.started {
		return
	}

	ep.log.Infof("Stopping event processor...")

	// Signal all workers to stop
	close(ep.stopChan)
	for _, worker := range ep.workers {
		close(worker.stopChan)
	}

	// Wait for all workers to finish
	ep.wg.Wait()

	ep.started = false
	ep.log.Infof("Event processor stopped")
}

// PublishEvent publishes an event for processing
func (ep *EventProcessor) PublishEvent(event *GameEvent) {
	ep.mu.Lock()
	started := ep.started
	ep.mu.Unlock()

	if !started {
		ep.log.Errorf("Event processor not started, dropping event: %v", event.Type)
		GetMetrics().IncEventDrop()
		return
	}

	select {
	case ep.queue <- event:
		ep.log.Debugf("Published event: %s for table %s", event.Type, event.TableID)
	default:
		ep.log.Errorf("Event queue full, dropping event: %s for table %s", event.Type, event.TableID)
		GetMetrics().IncEventDrop()
	}
}

// run executes the worker loop
func (w *eventWorker) run() {
	defer w.wg.Done()
	w.processor.log.Debugf("Event worker %d started", w.id)

	for {
		select {
		case <-w.stopChan:
			w.processor.log.Debugf("Event worker %d stopping", w.id)
			return

		case <-w.processor.stopChan:
			w.processor.log.Debugf("Event worker %d stopping (processor shutdown)", w.id)
			return

		case event, ok := <-w.processor.queue:
			if !ok {
				w.processor.log.Debugf("Event worker %d exiting: queue closed", w.id)
				return
			}
			if event != nil {
				w.processEvent(event)
			}
		}
	}
}

// processEvent processes a single event using all registered handlers
func (w *eventWorker) processEvent(event *GameEvent) {
	w.processor.log.Debugf("Worker %d processing event: %s for table %s", w.id, event.Type, event.TableID)

	// TABLE_REMOVED requires synchronous ordering: save -> finalize -> notify.
	if event.Type == pokerrpc.NotificationType_TABLE_REMOVED {
		w.processTableRemoved(event)
		return
	}

	// Process event through all handlers
	w.processNotifications(event)
	w.processGameStateUpdates(event)
	w.processPersistence(event)
}

// processNotifications handles notification broadcasting for the event
func (w *eventWorker) processNotifications(event *GameEvent) {
	handler := NewNotificationHandler(w.processor.server)
	handler.HandleEvent(event)
}

// processGameStateUpdates handles game state broadcasting for the event
func (w *eventWorker) processGameStateUpdates(event *GameEvent) {
	handler := NewGameStateHandler(w.processor.server)
	handler.HandleEvent(event)
}

// processPersistence handles state persistence for the event
func (w *eventWorker) processPersistence(event *GameEvent) {
	handler := NewPersistenceHandler(w.processor.server)
	handler.SaveTableStateAsync(event)
}

// processTableRemoved enforces ordering for table shutdown to avoid races.
func (w *eventWorker) processTableRemoved(event *GameEvent) {
	s := w.processor.server

	// Perform irreversible cleanup while holding the save mutex.
	s.finalizeTableRemoval(event.TableID)

	// Now broadcast to clients that the table is gone.
	handler := NewNotificationHandler(s)
	handler.handleTableRemoved(event)
}
