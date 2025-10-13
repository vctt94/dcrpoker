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
	// Archive current session key into historic dir using match_id
	CTArchiveSessionKey CmdType = 0x0e

	// Poker-specific commands
	CTGetPlayerCurrentTable   CmdType = 0x10
	CTLoadConfig              CmdType = 0x11
	CTGetPokerTables          CmdType = 0x12
	CTJoinPokerTable          CmdType = 0x13
	CTCreatePokerTable        CmdType = 0x14
	CTLeavePokerTable         CmdType = 0x15
	CTGetPokerBalance         CmdType = 0x16
	CTCreateDefaultConfig     CmdType = 0x17
	CTCreateDefaultServerCert CmdType = 0x18

	CTCreateLockFile        CmdType = 0x60
	CTCloseLockFile         CmdType = 0x61
	CTGetRunState           CmdType = 0x83
	CTEnableBackgroundNtfs  CmdType = 0x84
	CTDisableBackgroundNtfs CmdType = 0x85
	CTEnableProfiler        CmdType = 0x86
	CTZipTimedProfilingLogs CmdType = 0x87
	CTEnableTimedProfiling  CmdType = 0x89

	NTUINotification CmdType = 0x1001
	NTClientStopped  CmdType = 0x1002
	NTLogLine        CmdType = 0x1003
	NTNOP            CmdType = 0x1004
	NTWRCreated      CmdType = 0x1005
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
	Type    CmdType
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
			v, err = handleClientCmd(client, cmd)
		}
	}

	var resPayload []byte
	if err == nil {
		// Marshals null when v is nil — consistent for all calls.
		resPayload, err = json.Marshal(v)
	}

	return &CmdResult{ID: cmd.ID, Type: cmd.Type, Err: err, Payload: resPayload}
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

func notify(typ CmdType, payload interface{}, err error) {
	var resPayload []byte
	if err == nil {
		if b, mErr := json.Marshal(payload); mErr == nil {
			resPayload = b
		} else {
			err = fmt.Errorf("notify marshal: %w", mErr)
		}
	}
	r := &CmdResult{Type: typ, Err: err, Payload: resPayload}
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
		return &CmdResult{Type: NTNOP, Payload: []byte{}}
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
	switch r.Type {
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
			switch r.Type {
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
				cb.F(r.ID, int32(r.Type), payload, errMsg)
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
