package golib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
)

type CmdType = int32

const (
	CTUnknown    CmdType = 0x00
	CTHello      CmdType = 0x01
	CTInitClient CmdType = 0x02

	CTGetUserNick       CmdType = 0x03
	CTStopClient        CmdType = 0x04
	CTGetWRPlayers      CmdType = 0x05
	CTGetWaitingRooms   CmdType = 0x06
	CTJoinWaitingRoom   CmdType = 0x07
	CTCreateWaitingRoom CmdType = 0x08
	CTLeaveWaitingRoom  CmdType = 0x09
	// Settlement-related commands
	CTGenerateSessionKey CmdType = 0x0a
	CTOpenEscrow         CmdType = 0x0b
	CTStartPreSign       CmdType = 0x0c
	CTBindEscrow         CmdType = 0x0d
	// Archive current session key into historic dir using match_id
	CTArchiveSessionKey   CmdType = 0x0e
	CTDeriveSessionKey    CmdType = 0x0f
	CTGetEscrowStatus     CmdType = 0x30
	CTGetEscrowHistory    CmdType = 0x31
	CTGetFinalizeBundle   CmdType = 0x32 // Get gamma + presigs for settlement finalization
	CTGetEscrowById       CmdType = 0x33 // Get single escrow info by ID (includes comp_priv)
	CTGetBindableEscrows  CmdType = 0x34 // Get currently bindable escrows
	CTRefundEscrow        CmdType = 0x35 // Build CSV refund tx for a historic escrow
	CTUpdateEscrowHistory CmdType = 0x36
	CTDeleteEscrowHistory CmdType = 0x37

	// Poker-specific commands
	CTGetPlayerCurrentTable   CmdType = 0x10
	CTLoadConfig              CmdType = 0x11
	CTGetPokerTables          CmdType = 0x12
	CTJoinPokerTable          CmdType = 0x13
	CTCreatePokerTable        CmdType = 0x14
	CTLeavePokerTable         CmdType = 0x15
	CTWatchPokerTable         CmdType = 0x16
	CTCreateDefaultConfig     CmdType = 0x17
	CTCreateDefaultServerCert CmdType = 0x18
	CTUpdateConfig            CmdType = 0x20
	CTShowCards               CmdType = 0x19
	CTHideCards               CmdType = 0x1a
	CTMakeBet                 CmdType = 0x1b
	CTCallBet                 CmdType = 0x1c
	CTFoldBet                 CmdType = 0x1d
	CTCheckBet                CmdType = 0x1e
	CTGetGameState            CmdType = 0x1f
	CTGetLastWinners          CmdType = 0x20
	CTEvaluateHand            CmdType = 0x21
	CTSetPlayerReady          CmdType = 0x22
	CTSetPlayerUnready        CmdType = 0x23
	CTStartGameStream         CmdType = 0x27
	CTUnwatchPokerTable       CmdType = 0x2d

	// Auth commands
	CTRegister         CmdType = 0x24
	CTLogin            CmdType = 0x25
	CTLogout           CmdType = 0x26
	CTResumeSession    CmdType = 0x28
	CTRequestLoginCode CmdType = 0x29
	CTSetPayoutAddress CmdType = 0x2a
	CTGetPayoutAddress CmdType = 0x2b
	CTReconnectNow     CmdType = 0x2c

	CTCreateLockFile        CmdType = 0x60
	CTCloseLockFile         CmdType = 0x61
	CTGetRunState           CmdType = 0x83
	CTEnableBackgroundNtfs  CmdType = 0x84
	CTDisableBackgroundNtfs CmdType = 0x85
	CTEnableProfiler        CmdType = 0x86
	CTZipTimedProfilingLogs CmdType = 0x87
	CTEnableTimedProfiling  CmdType = 0x89
	CTReadLogPage           CmdType = 0x8a

	NTUINotification    CmdType = 0x1001
	NTClientStopped     CmdType = 0x1002
	NTLogLine           CmdType = 0x1003
	NTNOP               CmdType = 0x1004
	NTWRCreated         CmdType = 0x1005
	NTPokerNotification CmdType = 0x1006
	NTGameUpdate        CmdType = 0x1007
	NTPresignError      CmdType = 0x1008
)

type cmd struct {
	Type         CmdType
	ID           int32
	ClientHandle int32
	Payload      []byte
}

// strict JSON decode (reject unknown fields & trailing data)
func decodeStrict(b []byte, out any) error {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	// disallow trailing junk
	if dec.More() {
		return fmt.Errorf("unexpected trailing data")
	}
	return nil
}

func (cmd *cmd) decode(to interface{}) error {
	return decodeStrict(cmd.Payload, to)
}

type CmdResult struct {
	ID      int32
	Type    int32
	Err     error
	Payload []byte
}

type CmdResultLoopCB interface {
	F(id int32, typ int32, payload string, err string)
	UINtfn(text string, nick string, ts int64)
}

// buffer to avoid transient producer>consumer bursts
var cmdResultChan = make(chan *CmdResult, 256)

func call(cmd *cmd) *CmdResult {
	var v interface{}
	var err error

	decode := func(to interface{}) bool {
		err = cmd.decode(to)
		if err != nil {
			err = fmt.Errorf("unable to decode input payload: %v; full payload: %s", err, spew.Sdump(cmd.Payload))
		}
		return err == nil
	}

	// Handle calls that do not need a client.
	switch cmd.Type {
	case CTHello:
		var name string
		if decode(&name) {
			v, err = handleHello(name)
		}

	case CTInitClient:
		var initClient initClient
		if decode(&initClient) {
			v, err = handleInitClient(uint32(cmd.ClientHandle), initClient)
		}
	case CTLoadConfig:
		// Accept a string payload (filepath or datadir) to load config from Go.
		var pathOrDir string
		if decode(&pathOrDir) {
			v, err = handleLoadConfig(pathOrDir)
		}

	case CTCreateDefaultConfig:
		var args createDefaultConfigArgs
		if decode(&args) {
			v, err = handleCreateDefaultConfig(args)
		}

	case CTCreateDefaultServerCert:
		var certPath string
		if decode(&certPath) {
			v, err = handleCreateDefaultServerCert(certPath)
		}

	case CTUpdateConfig:
		var args updateConfigArgs
		if decode(&args) {
			v, err = handleUpdateConfig(args)
		}

	case CTCreateLockFile:
		var args string
		if decode(&args) {
			err = handleCreateLockFile(args)
		}

	case CTCloseLockFile:
		var args string
		if decode(&args) {
			err = handleCloseLockFile(args)
		}

	case CTGetRunState:
		v = runState{
			ClientRunning: isClientRunning(uint32(cmd.ClientHandle)),
		}

	case CTEnableProfiler:
		var addr string
		if decode(&addr) {
			if addr == "" {
				addr = "0.0.0.0:8118"
			}
			fmt.Printf("Enabling profiler on %s\n", addr)
			go func() {
				if err := http.ListenAndServe(addr, nil); err != nil {
					fmt.Printf("Unable to listen on profiler %s: %v\n", addr, err)
				}
			}()
		}

	case CTEnableTimedProfiling:
		var args string
		if decode(&args) {
			go globalProfiler.Run(args)
		}

	case CTZipTimedProfilingLogs:
		var dest string
		if decode(&dest) {
			err = globalProfiler.zipLogs(dest)
		}

	case CTReadLogPage:
		var args readLogPageArgs
		if decode(&args) {
			v, err = handleReadLogPage(args)
		}

	case CTRegister:
		var req registerReq
		if decode(&req) {
			v, err = handleRegister(uint32(cmd.ClientHandle), req)
		}

	case CTLogin:
		var req loginReq
		if decode(&req) {
			v, err = handleLogin(uint32(cmd.ClientHandle), req)
		}

	case CTResumeSession:
		v, err = handleResumeSession(uint32(cmd.ClientHandle))

	case CTRequestLoginCode:
		v, err = handleRequestLoginCode(uint32(cmd.ClientHandle))

	case CTSetPayoutAddress:
		var req setPayoutAddressReq
		if decode(&req) {
			v, err = handleSetPayoutAddress(uint32(cmd.ClientHandle), req)
		}

	case CTGetPayoutAddress:
		{
			cmtx.Lock()
			cc := cs[uint32(cmd.ClientHandle)]
			cmtx.Unlock()
			if cc == nil || cc.c == nil {
				v, err = nil, fmt.Errorf("poker client not initialized")
			} else {
				v = map[string]string{"payout_address": cc.c.PayoutAddress()}
			}
		}

	case CTLogout:
		v, err = handleLogout(uint32(cmd.ClientHandle))

	default:
		// Calls that need a client. Figure out the client.
		cmtx.Lock()
		var client *clientCtx
		if cs != nil {
			client = cs[uint32(cmd.ClientHandle)]
		}
		cmtx.Unlock()

		if client == nil {
			err = fmt.Errorf("unknown client handle %d", cmd.ClientHandle)
		} else {
			v, err = handleClientCmd(uint32(cmd.ClientHandle), client, cmd)
		}
	}

	var resPayload []byte
	if err == nil {
		// Marshals null when v is nil — consistent for all calls.
		resPayload, err = json.Marshal(v)
	}

	return &CmdResult{ID: cmd.ID, Type: int32(cmd.Type), Err: err, Payload: resPayload}
}

func AsyncCall(typ CmdType, id, clientHandle int32, payload []byte) {
	cmd := &cmd{
		Type:         typ,
		ID:           id,
		ClientHandle: clientHandle,
		Payload:      payload,
	}
	go func() { cmdResultChan <- call(cmd) }()
}

func AsyncCallStr(typ CmdType, id, clientHandle int32, payload string) {
	cmd := &cmd{
		Type:         typ,
		ID:           id,
		ClientHandle: clientHandle,
		Payload:      []byte(payload),
	}
	go func() { cmdResultChan <- call(cmd) }()
}

type notificationDTO struct {
	Type            int32         `json:"type"` // enum as int
	Message         string        `json:"message,omitempty"`
	TableId         string        `json:"tableId,omitempty"`
	PlayerId        string        `json:"playerId,omitempty"`
	Cards           []*cardDTO    `json:"cards,omitempty"`
	Amount          int64         `json:"amount,omitempty"`
	NewBalance      int64         `json:"newBalance,omitempty"`
	Ready           bool          `json:"ready"`
	Started         bool          `json:"started"`
	GameReadyToPlay bool          `json:"gameReadyToPlay"`
	Countdown       int32         `json:"countdown,omitempty"`
	Table           *ntfnTableDTO `json:"table,omitempty"` // Table snapshot for lobby updates (includes players)
	// Settlement fields for GAME_ENDED notifications
	WinnerId   string `json:"winnerId,omitempty"`
	WinnerSeat int32  `json:"winnerSeat,omitempty"`
	MatchId    string `json:"matchId,omitempty"`
	IsWinner   bool   `json:"isWinner,omitempty"`
	// Showdown fields for SHOWDOWN_RESULT notifications
	Winners         []*pokerrpc.Winner         `json:"winners,omitempty"`
	ShowdownPot     int64                      `json:"showdownPot,omitempty"`
	ShowdownPlayers []*pokerrpc.ShowdownPlayer `json:"players,omitempty"`
	Board           []*pokerrpc.Card           `json:"board,omitempty"`
}

func notificationToDTO(n *pokerrpc.Notification) *notificationDTO {
	dto := &notificationDTO{
		Type: int32(n.Type),
	}
	if n.Message != "" {
		dto.Message = n.Message
	}
	if n.TableId != "" {
		dto.TableId = n.TableId
	}
	if n.PlayerId != "" {
		dto.PlayerId = n.PlayerId
	}
	if len(n.Cards) > 0 {
		dto.Cards = make([]*cardDTO, 0, len(n.Cards))
		for _, c := range n.Cards {
			dto.Cards = append(dto.Cards, cardToDTO(c))
		}
	}
	if n.Amount != 0 {
		dto.Amount = n.Amount
	}
	if n.NewBalance != 0 {
		dto.NewBalance = n.NewBalance
	}
	dto.Ready = n.Ready
	dto.Started = n.Started
	dto.GameReadyToPlay = n.GameReadyToPlay
	if n.Countdown != 0 {
		dto.Countdown = n.Countdown
	}
	// Include table snapshot if present (for PLAYER_JOINED, PLAYER_LEFT, etc.)
	if n.Table != nil {
		dto.Table = tableFromProtoForNtfn(n.Table)
	}
	// Settlement fields for GAME_ENDED
	if n.WinnerId != "" {
		dto.WinnerId = n.WinnerId
	}
	if n.WinnerSeat != 0 {
		dto.WinnerSeat = n.WinnerSeat
	}
	if n.MatchId != "" {
		dto.MatchId = n.MatchId
	}
	dto.IsWinner = n.IsWinner
	// Carry full showdown payload so Flutter can render last showdown correctly.
	if n.Showdown != nil {
		if n.Showdown.Pot != 0 {
			dto.ShowdownPot = n.Showdown.Pot
		}
		if len(n.Showdown.Winners) > 0 {
			dto.Winners = n.Showdown.Winners
		}
		if len(n.Showdown.Players) > 0 {
			dto.ShowdownPlayers = n.Showdown.Players
		}
		if len(n.Showdown.Board) > 0 {
			dto.Board = n.Showdown.Board
		}
	} else if len(n.Winners) > 0 {
		// Fallback to top-level winners slice if present.
		dto.Winners = n.Winners
	}
	return dto
}

func notify(typ CmdType, payload interface{}, err error) {
	var resPayload []byte
	if err == nil && payload != nil {
		// Convert protobuf Notification to simple DTO for JSON marshaling
		if n, ok := payload.(*pokerrpc.Notification); ok {
			dto := notificationToDTO(n)
			if b, mErr := json.Marshal(dto); mErr == nil {
				resPayload = b
			} else {
				err = fmt.Errorf("notify json marshal: %w", mErr)
			}
		} else {
			// Use standard JSON for other types
			if b, mErr := json.Marshal(payload); mErr == nil {
				resPayload = b
			} else {
				err = fmt.Errorf("notify json marshal: %w", mErr)
			}
		}
	}
	r := &CmdResult{Type: int32(typ), Err: err, Payload: resPayload}
	// non-blocking to avoid deadlocks under bursty shutdowns
	select {
	case cmdResultChan <- r:
	default:
		fmt.Println("notify: dropping CmdResult due to full channel")
	}
}

func NextCmdResult() *CmdResult {
	select {
	case r := <-cmdResultChan:
		return r
	case <-time.After(time.Second): // Timeout.
		return &CmdResult{Type: int32(NTNOP), Payload: []byte{}}
	}
}

var (
	cmdResultLoopsMtx   sync.Mutex
	cmdResultLoops      = map[int32]chan struct{}{}
	cmdResultLoopsLive  atomic.Int32
	cmdResultLoopsCount int32
)

// Minimal UI notification shape to decouple from BR client package.
type uiNtfn struct {
	Text      string `json:"text"`
	FromNick  string `json:"from_nick"`
	Timestamp int64  `json:"timestamp"`
}

// emitBackgroundNtfns emits background notifications to the callback object.
func emitBackgroundNtfns(r *CmdResult, cb CmdResultLoopCB) {
	switch CmdType(r.Type) {
	case NTUINotification:
		var n uiNtfn
		if err := json.Unmarshal(r.Payload, &n); err != nil {
			return
		}
		cb.UINtfn(n.Text, n.FromNick, n.Timestamp)
	default:
		// Ignore every other notification.
	}
}

// CmdResultLoop runs the loop that fetches async results in a goroutine and
// calls cb.F() with the results. Returns an ID that may be passed to
// StopCmdResultLoop to stop this goroutine.
//
// If onlyBgNtfns is specified, only background notifications are sent.
func CmdResultLoop(cb CmdResultLoopCB, onlyBgNtfns bool) int32 {
	cmdResultLoopsMtx.Lock()
	id := cmdResultLoopsCount + 1
	cmdResultLoopsCount += 1
	ch := make(chan struct{})
	cmdResultLoops[id] = ch
	cmdResultLoopsLive.Add(1)
	cmdResultLoopsMtx.Unlock()

	// onlyBgNtfns == true when this is called from the native plugin
	// code while the flutter engine is _not_ attached to it.
	deliverBackgroundNtfns := onlyBgNtfns

	cmtx.Lock()
	if cs != nil && cs[0x12131400] != nil {
		cc := cs[0x12131400]
		cc.log.Infof("CmdResultLoop: starting new run for pid %d id %d",
			os.Getpid(), id)
	}
	cmtx.Unlock()

	go func() {
		minuteTicker := time.NewTicker(time.Minute)
		defer minuteTicker.Stop()
		startTime := time.Now()
		wallStartTime := startTime.Round(0)
		lastTime := startTime
		lastCPUTimes := make([]cpuTime, 6)

		defer func() {
			cmtx.Lock()
			if cs != nil && cs[0x12131400] != nil {
				elapsed := time.Since(startTime).Truncate(time.Millisecond)
				elapsedWall := time.Since(wallStartTime).Truncate(time.Millisecond)
				cc := cs[0x12131400]
				cc.log.Infof("CmdResultLoop: finishing "+
					"goroutine for pid %d id %d after %s (wall %s)",
					os.Getpid(), id, elapsed, elapsedWall)
			}
			cmtx.Unlock()
		}()

		for {
			var r *CmdResult
			select {
			case r = <-cmdResultChan:
			case <-minuteTicker.C:
				// This is being used to debug background issues
				// on mobile. It may be removed in the future.
				go reportCmdResultLoop(startTime, lastTime, id, lastCPUTimes)
				lastTime = time.Now()
				continue

			case <-ch:
				return
			}

			// Process the special commands that toggle calling
			// native code with background ntfn events.
			switch CmdType(r.Type) {
			case CTEnableBackgroundNtfs:
				deliverBackgroundNtfns = true
				continue
			case CTDisableBackgroundNtfs:
				deliverBackgroundNtfns = false
				continue
			}

			// If the flutter engine is attached to the process,
			// deliver the event so that it can be processed.
			if !onlyBgNtfns {
				var errMsg, payload string
				if r.Err != nil {
					errMsg = r.Err.Error()
				}
				if len(r.Payload) > 0 {
					payload = string(r.Payload)
				}
				cb.F(r.ID, r.Type, payload, errMsg)
			}

			// Emit a background ntfn if the flutter engine is
			// detached or if it is attached but paused/on
			// background.
			if deliverBackgroundNtfns {
				emitBackgroundNtfns(r, cb)
			}
		}
	}()

	return id
}

// StopCmdResultLoop stops an async goroutine created with CmdResultLoop. Does
// nothing if this goroutine is already stopped.
func StopCmdResultLoop(id int32) {
	cmdResultLoopsMtx.Lock()
	ch := cmdResultLoops[id]
	delete(cmdResultLoops, id)
	cmdResultLoopsLive.Add(-1)
	cmdResultLoopsMtx.Unlock()
	if ch != nil {
		close(ch)
	}
}

// StopAllCmdResultLoops stops all async goroutines created by CmdResultLoop.
func StopAllCmdResultLoops() {
	cmdResultLoopsMtx.Lock()
	chans := cmdResultLoops
	cmdResultLoops = map[int32]chan struct{}{}
	cmdResultLoopsLive.Store(0)
	cmdResultLoopsMtx.Unlock()
	for _, ch := range chans {
		close(ch)
	}
}

// ClientExists returns true if the client with the specified handle is running.
func ClientExists(handle int32) bool {
	cmtx.Lock()
	exists := cs != nil && cs[uint32(handle)] != nil
	cmtx.Unlock()
	return exists
}

func LogInfo(id int32, s string) {
	cmtx.Lock()
	if cs != nil && cs[uint32(id)] != nil {
		cs[uint32(id)].log.Info(s)
	} else {
		fmt.Println(s)
	}
	cmtx.Unlock()
}
