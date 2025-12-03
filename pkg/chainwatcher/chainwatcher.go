package chainwatcher

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/decred/dcrd/chaincfg/chainhash"
	"github.com/decred/dcrd/rpcclient/v8"
	"github.com/decred/slog"
)

// EscrowUTXO is the minimal UTXO view needed for escrow tracking.
type EscrowUTXO struct {
	Txid        string
	Vout        uint32
	Value       uint64
	PkScriptHex string
	Confs       uint32
}

type DepositUpdate struct {
	PkScriptHex string
	Confs       uint32
	UTXOCount   int
	OK          bool
	At          time.Time
	UTXOs       []*EscrowUTXO
}

type ChainWatcher struct {
	log  slog.Logger
	dcrd *rpcclient.Client

	mu      sync.RWMutex
	subs    map[string]map[chan DepositUpdate]struct{} // pkHex -> set(chan)
	pkBytes map[string][]byte                          // pkHex -> script bytes
	known   map[string]map[string]*EscrowUTXO          // pkHex -> (txid:vout -> utxo)
}

var (
	// ErrTxNotFound indicates the transaction does not exist on chain.
	ErrTxNotFound = errors.New("tx not found")
	// ErrVoutNotFound indicates the requested output index is out of range for the tx.
	ErrVoutNotFound = errors.New("vout not found")
	// ErrOutpointSpent indicates the tx/vout existed but is no longer unspent.
	ErrOutpointSpent = errors.New("outpoint already spent")
)

func NewChainWatcher(log slog.Logger, c *rpcclient.Client) *ChainWatcher {
	return &ChainWatcher{
		log:     log,
		dcrd:    c,
		subs:    make(map[string]map[chan DepositUpdate]struct{}),
		pkBytes: make(map[string][]byte),
		known:   make(map[string]map[string]*EscrowUTXO),
	}
}

func (w *ChainWatcher) Stop() {} // nothing to stop

// LookupUTXO fetches a specific txid:vout if pkScript matches (best-effort).
func (w *ChainWatcher) LookupUTXO(outpoint string, pkScriptHex string) (*EscrowUTXO, error) {
	w.mu.RLock()
	scriptBytes := w.pkBytes[strings.ToLower(pkScriptHex)]
	w.mu.RUnlock()
	parts := strings.Split(outpoint, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("outpoint must be txid:vout")
	}
	vout, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid vout")
	}
	var h chainhash.Hash
	if err := chainhash.Decode(&h, parts[0]); err != nil {
		return nil, fmt.Errorf("invalid txid")
	}
	ctx := context.Background()

	// First attempt to fetch current UTXO view.
	res, err := w.dcrd.GetTxOut(ctx, &h, uint32(vout), 0, true)
	if err != nil || res == nil {
		// Distinguish between "never existed" and "already spent".
		vtx, errTx := w.dcrd.GetRawTransactionVerbose(ctx, &h)
		if errTx != nil || vtx == nil {
			return nil, ErrTxNotFound
		}
		if int(vout) >= len(vtx.Vout) {
			return nil, ErrVoutNotFound
		}
		// Tx and vout exist but there is no unspent output at this index.
		return nil, ErrOutpointSpent
	}
	// Verify pkScript matches if we have one.
	if scriptBytes != nil {
		if b, err := hex.DecodeString(res.ScriptPubKey.Hex); err == nil {
			if !bytes.Equal(b, scriptBytes) {
				return nil, fmt.Errorf("pkScript mismatch")
			}
		}
	}
	// value comes as float; convert to atoms conservatively
	atoms := uint64(res.Value*1e8 + 0.5)
	return &EscrowUTXO{
		Txid:        parts[0],
		Vout:        uint32(vout),
		Value:       atoms,
		PkScriptHex: pkScriptHex,
		Confs:       uint32(res.Confirmations),
	}, nil
}

// Subscribe registers interest in updates for the given pkScript hex.
func (w *ChainWatcher) Subscribe(pkScriptHex string) (<-chan DepositUpdate, func()) {
	k := strings.ToLower(pkScriptHex)
	if b, err := hex.DecodeString(k); err == nil {
		w.mu.Lock()
		w.pkBytes[k] = b
		w.mu.Unlock()
	}

	ch := make(chan DepositUpdate, 8)
	w.mu.Lock()
	if _, ok := w.subs[k]; !ok {
		w.subs[k] = make(map[chan DepositUpdate]struct{})
	}
	w.subs[k][ch] = struct{}{}
	n := len(w.subs[k])
	w.mu.Unlock()
	w.log.Infof("watcher: subscribed pk=%s (subs=%d)", k, n)

	unsub := func() {
		w.mu.Lock()
		if set, ok := w.subs[k]; ok {
			delete(set, ch)
			if len(set) == 0 {
				delete(w.subs, k)
				delete(w.known, k)
				delete(w.pkBytes, k)
			}
		}
		rem := 0
		if set, ok := w.subs[k]; ok {
			rem = len(set)
		}
		w.mu.Unlock()
		w.log.Infof("watcher: unsubscribed pk=%s (subs=%d)", k, rem)
	}
	return ch, unsub
}

func (w *ChainWatcher) broadcastUpdate(pk string, u DepositUpdate) {
	w.mu.RLock()
	set := w.subs[pk]
	chs := make([]chan DepositUpdate, 0, len(set))
	for ch := range set {
		chs = append(chs, ch)
	}
	w.mu.RUnlock()

	for _, ch := range chs {
		select {
		case ch <- u:
		default: /* drop if slow */
		}
	}
}

// ProcessTxAcceptedHash should be called from dcrd OnTxAccepted notifications.
func (w *ChainWatcher) ProcessTxAcceptedHash(ctx context.Context, hash *chainhash.Hash) {
	if hash == nil {
		return
	}

	v, err := w.dcrd.GetRawTransactionVerbose(ctx, hash)
	if err != nil || v == nil {
		return
	}

	// snapshot subs + pkBytes
	w.mu.RLock()
	if len(w.subs) == 0 {
		w.mu.RUnlock()
		return
	}
	keys := make([]string, 0, len(w.subs))
	for k := range w.subs {
		keys = append(keys, k)
	}
	pkb := make(map[string][]byte, len(keys))
	for _, k := range keys {
		pkb[k] = w.pkBytes[k]
	}
	w.mu.RUnlock()

	discoveredByPk := make(map[string][]*EscrowUTXO)
	for i, out := range v.Vout {
		spk, err := hex.DecodeString(out.ScriptPubKey.Hex)
		if err != nil {
			continue
		}
		for _, k := range keys {
			if b := pkb[k]; b != nil && bytes.Equal(spk, b) {
				// value comes as float; convert to atoms conservatively
				atoms := uint64(out.Value*1e8 + 0.5)
				discoveredByPk[k] = append(discoveredByPk[k], &EscrowUTXO{
					Txid: v.Txid, Vout: uint32(i), Value: atoms, PkScriptHex: k,
				})
			}
		}
	}
	if len(discoveredByPk) == 0 {
		return
	}

	now := time.Now()
	for k, list := range discoveredByPk {
		w.mu.Lock()
		m := w.known[k]
		if m == nil {
			m = make(map[string]*EscrowUTXO)
			w.known[k] = m
		}
		for _, u := range list {
			m[fmt.Sprintf("%s:%d", u.Txid, u.Vout)] = u
		}
		cur := make([]*EscrowUTXO, 0, len(m))
		for _, u := range m {
			cur = append(cur, u)
		}
		w.mu.Unlock()

		w.broadcastUpdate(k, DepositUpdate{
			PkScriptHex: k, Confs: 0, UTXOCount: len(cur),
			OK: len(cur) > 0, At: now, UTXOs: cur,
		})
	}
}

// ProcessBlockConnected should be called from dcrd OnBlockConnected notifications.
func (w *ChainWatcher) ProcessBlockConnected(ctx context.Context) {
	// copy keys to check
	w.mu.RLock()
	keys := make([]string, 0, len(w.known))
	for k := range w.known {
		keys = append(keys, k)
	}
	w.mu.RUnlock()

	now := time.Now()
	for _, k := range keys {
		// copy ids so we can RPC without holding the lock
		w.mu.RLock()
		km := w.known[k]
		ids := make([]string, 0, len(km))
		for id := range km {
			ids = append(ids, id)
		}
		w.mu.RUnlock()

		cur := make([]*EscrowUTXO, 0, len(ids))
		minConfs := int64(^uint32(0))

		for _, id := range ids {
			w.mu.RLock()
			u := w.known[k][id]
			w.mu.RUnlock()
			if u == nil {
				continue
			}

			var h chainhash.Hash
			if err := chainhash.Decode(&h, u.Txid); err != nil {
				continue
			}

			res, err := w.dcrd.GetTxOut(ctx, &h, u.Vout, 0, true)
			if err != nil || res == nil {
				// spent/unknown → evict
				w.mu.Lock()
				if set := w.known[k]; set != nil {
					delete(set, id)
					if len(set) == 0 {
						delete(w.known, k)
					}
				}
				w.mu.Unlock()
				continue
			}

			cur = append(cur, u)
			if res.Confirmations < minConfs {
				minConfs = res.Confirmations
			}
		}

		var confs uint32
		if len(cur) == 0 || minConfs < 0 {
			confs = 0
		} else if minConfs > int64(^uint32(0)) {
			confs = ^uint32(0)
		} else {
			confs = uint32(minConfs)
		}

		w.broadcastUpdate(k, DepositUpdate{
			PkScriptHex: k, Confs: confs, UTXOCount: len(cur),
			OK: len(cur) > 0, At: now, UTXOs: cur,
		})
	}
}
