package poker

import (
	"fmt"
	"sort"
)

// pot represents a pot of chips in the game
type pot struct {
	amount      int64  // Total amount in the pot
	eligibility []bool // len == len(players); seat-aligned mask
}

// potManager manages multiple pots, including the main pot and side pots
type potManager struct {
	mu          RWLock        // Protects all potManager fields
	pots        []*pot        // Main pot followed by side pots
	currentBets map[int]int64 // Current bet for each player in this round
	totalBets   map[int]int64 // Total bet for each player across all rounds
}

// newPot creates a new pot with the given amount
func newPot(nPlayers int) *pot {
	return &pot{
		amount:      0,
		eligibility: make([]bool, nPlayers),
	}
}

// makeEligible marks a player as eligible to win this pot
func (p *pot) makeEligible(playerIndex int) {
	p.eligibility[playerIndex] = true
}

// isEligible checks if a player is eligible to win this pot
func (p *pot) isEligible(playerIndex int) bool {
	return p.eligibility[playerIndex]
}

func NewPotManager(nPlayers int) *potManager {
	return &potManager{
		pots:        []*pot{newPot(nPlayers)}, // placeholder; real amounts built later
		currentBets: make(map[int]int64),
		totalBets:   make(map[int]int64),
	}
}

// AddBet adds a bet and immediately rebuilds pots to handle side pot creation
func (pm *potManager) addBet(playerIndex int, amount int64, players []*Player) {
	// Pre-compute fold status BEFORE acquiring pot manager lock to avoid deadlock
	foldStatus := make([]bool, len(players))
	for i, p := range players {
		if p != nil {
			foldStatus[i] = (p.GetCurrentStateString() == "FOLDED")
		}
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.currentBets[playerIndex] += amount
	pm.totalBets[playerIndex] += amount
	pm.rebuildPotsIncremental(players, foldStatus)
}

// getTotalPot returns the total amount across all pots
func (pm *potManager) getTotalPot() int64 {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	var total int64
	for _, pot := range pm.pots {
		total += pot.amount
	}
	return total
}

// getCurrentBet returns the current bet for a player
func (pm *potManager) getCurrentBet(playerIndex int) int64 {
	return pm.currentBets[playerIndex]
}

// getTotalBet returns the total bet for a player across all rounds
func (pm *potManager) getTotalBet(playerIndex int) int64 {
	return pm.totalBets[playerIndex]
}

// RebuildPotsIncremental rebuilds the pot structure from pm.totalBets and player states.
// It first handles the uncontested case (exactly one non-folded player), then falls back
// to layered side-pot construction by contribution thresholds.
func (pm *potManager) RebuildPotsIncremental(players []*Player) {
	// Pre-compute fold status BEFORE acquiring pot manager lock to avoid deadlock
	foldStatus := make([]bool, len(players))
	for i, p := range players {
		if p != nil {
			foldStatus[i] = (p.GetCurrentStateString() == "FOLDED")
		}
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.rebuildPotsIncremental(players, foldStatus)
}

// rebuildPotsIncremental rebuilds the pot structure from pm.totalBets.
// CRITICAL: foldStatus must be pre-computed BEFORE calling (no Player access under pm.mu)
func (pm *potManager) rebuildPotsIncremental(players []*Player, foldStatus []bool) {
	mustHeld(&pm.mu)
	n := len(players)
	if n == 0 {
		pm.pots = []*pot{newPot(0)}
		return
	}

	// Count non-folded ("alive") players and remember the last alive seat.
	alive := 0
	lastAlive := -1
	for i := 0; i < n; i++ {
		if players[i] != nil && !foldStatus[i] {
			alive++
			lastAlive = i
		}
	}

	// Uncontested short-circuit: one alive player => a single pot equal to
	// the sum of all contributions (including folded players' prior bets).
	if alive == 1 {
		total := int64(0)
		for i := 0; i < n; i++ {
			total += pm.totalBets[i]
		}
		p := newPot(n)
		p.amount = total
		if lastAlive >= 0 {
			p.makeEligible(lastAlive)
		}
		pm.pots = []*pot{p}
		return
	}

	// Collect unique positive contribution thresholds from totalBets.
	seen := make(map[int64]struct{}, n)
	for i := 0; i < n; i++ {
		if tb := pm.totalBets[i]; tb > 0 {
			seen[tb] = struct{}{}
		}
	}

	// If nobody has put chips in, keep a single empty pot scaffold.
	if len(seen) == 0 {
		pm.pots = []*pot{newPot(n)}
		return
	}

	// Sort thresholds ascending to build layered (capped) pots.
	levels := make([]int64, 0, len(seen))
	for v := range seen {
		levels = append(levels, v)
	}
	sort.Slice(levels, func(i, j int) bool { return levels[i] < levels[j] })

	pots := make([]*pot, 0, len(levels)+1)
	prev := int64(0)

	// Build one capped layer per threshold.
	for _, lvl := range levels {
		p := newPot(n)
		amt := int64(0)

		for i := 0; i < n; i++ {
			tb := pm.totalBets[i]

			// Eligible if player is alive and has contributed at least up to this cap.
			if players[i] != nil && !foldStatus[i] && tb >= lvl {
				p.makeEligible(i)
			}

			// Each player contributes the slice of their bet between (prev, lvl].
			if tb > prev {
				upTo := tb
				if upTo > lvl {
					upTo = lvl
				}
				if upTo > prev {
					amt += (upTo - prev)
				}
			}
		}

		p.amount = amt
		pots = append(pots, p)
		prev = lvl
	}

	// Final uncapped overage (above the highest threshold), if any.
	top := levels[len(levels)-1]
	over := newPot(n)
	hasOver := false
	for i := 0; i < n; i++ {
		tb := pm.totalBets[i]
		if tb > top {
			over.amount += tb - top
			if players[i] != nil && !foldStatus[i] {
				over.makeEligible(i)
			}
			hasOver = true
		}
	}
	if hasOver {
		pots = append(pots, over)
	}

	pm.pots = pots
}

// distributePots distributes all pots to showdown winners.
// Robust to accidental calls on uncontested pots and idempotent:
// pots are zeroed after payout so re-entry is a no-op.
// distributePots pays out all pots. Safe to call multiple times (pots are zeroed after payout).
func (pm *potManager) distributePots(players []*Player) error {
	// Pre-compute fold status
	foldStatus := make([]bool, len(players))
	for i, p := range players {
		if p != nil {
			foldStatus[i] = (p.GetCurrentStateString() == "FOLDED")
		}
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()
	for pi, pot := range pm.pots {
		// Idempotent: skip empty/already-settled pots.
		if pot.amount <= 0 {
			continue
		}

		// Collect eligible & not-folded players.
		if len(pot.eligibility) != len(players) {
			return fmt.Errorf("[pot %d] eligibility len %d != players len %d",
				pi, len(pot.eligibility), len(players))
		}
		var alive []int
		for idx, elig := range pot.eligibility {
			if idx < 0 || idx >= len(players) {
				return fmt.Errorf("[pot %d] eligibility idx %d out of range (players=%d)", pi, idx, len(players))
			}
			if elig && players[idx] != nil && !foldStatus[idx] {
				alive = append(alive, idx)
			}
		}

		// Uncontested pot path.
		if len(alive) == 1 {
			w := alive[0]
			if players[w] != nil {
				if err := players[w].credit(pot.amount); err != nil {
					return fmt.Errorf("[pot %d] failed to credit winner: %w", pi, err)
				}
			}
			pm.pots[pi].amount = 0
			for j := range pm.pots[pi].eligibility {
				pm.pots[pi].eligibility[j] = false
			}
			continue
		}
		if len(alive) == 0 {
			return fmt.Errorf("[pot %d] no eligible alive players; pot=%d", pi, pot.amount)
		}

		// Showdown: find best hand(s) safely.
		var winners []int
		var best *HandValue
		for _, idx := range alive {
			var hv *HandValue
			if players[idx] != nil {
				players[idx].mu.RLock()
				hv = players[idx].handValue
				players[idx].mu.RUnlock()
			}
			if hv == nil {
				return fmt.Errorf("[pot %d] player %d eligible at showdown but HandValue == nil", pi, idx)
			}
			if best == nil {
				best = hv
				winners = []int{idx}
				continue
			}
			cmp := CompareHands(*hv, *best)
			if cmp > 0 {
				best = hv
				winners = []int{idx}
			} else if cmp == 0 {
				winners = append(winners, idx)
			}
		}
		if len(winners) == 0 {
			return fmt.Errorf("[pot %d] showdown produced no winners", pi)
		}

		// Split pot; first winner gets remainder.
		share := pot.amount / int64(len(winners))
		rem := pot.amount % int64(len(winners))
		for i, idx := range winners {
			add := share
			if i == 0 && rem > 0 {
				add += rem
			}
			if players[idx] != nil {
				if err := players[idx].credit(add); err != nil {
					return fmt.Errorf("[pot %d] failed to credit winner %d: %w", pi, idx, err)
				}
			}
		}

		// Mark pot as settled.
		pm.pots[pi].amount = 0
		for j := range pm.pots[pi].eligibility {
			pm.pots[pi].eligibility[j] = false
		}
	}
	return nil
}

// ReturnUncalledBet returns any uncalled portion of the top bet to the bettor.
// It handles the special "no-call" case by refunding down to the bettor's forced amount.
//
// forced[i] = player's forced contribution for THIS street.
//
//	Preflop heads-up: forced = [smallBlind, bigBlind] (e.g., [10, 20])
//	Later streets or non-blind players: 0
//
// Returns (hiPlayer, refunded, error). Caller can decide when to rebuild pots.
func (pm *potManager) returnUncalledBet(forced []int64) (int, int64, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	n := len(pm.currentBets)
	if n == 0 || len(forced) != n {
		return -1, 0, fmt.Errorf("invalid input")
	}

	// Find highest and second-highest current bet.
	var hi, second int64
	hiPlayer := -1
	for i, b := range pm.currentBets {
		if b > hi {
			second = hi
			hi = b
			hiPlayer = i
		} else if b > second {
			second = b
		}
	}
	if hiPlayer < 0 || hi <= second {
		return -1, 0, nil // nothing to refund
	}

	// "No-call" if every non-aggressor put in no more than their forced amount.
	noCall := true
	for i, b := range pm.currentBets {
		if i == hiPlayer {
			continue
		}
		if b > forced[i] {
			noCall = false
			break
		}
	}

	cap := second
	if noCall {
		cap = forced[hiPlayer] // refund down to bettor's forced amount (e.g., SB=10 preflop)
	}

	uncalled := hi - cap
	if uncalled <= 0 {
		return -1, 0, nil
	}

	// Refund and adjust contributions.
	pm.currentBets[hiPlayer] -= uncalled
	pm.totalBets[hiPlayer] -= uncalled
	if pm.currentBets[hiPlayer] < 0 {
		pm.currentBets[hiPlayer] = 0
	}
	if pm.totalBets[hiPlayer] < 0 {
		pm.totalBets[hiPlayer] = 0
	}

	return hiPlayer, uncalled, nil
}
