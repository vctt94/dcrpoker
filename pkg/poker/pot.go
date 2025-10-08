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

// potManager manages multiple pots, including the main pot and side pots
type potManager struct {
	pots        []*pot        // Main pot followed by side pots
	currentBets map[int]int64 // Current bet for each player in this round
	totalBets   map[int]int64 // Total bet for each player across all rounds
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
	pm.currentBets[playerIndex] += amount
	pm.totalBets[playerIndex] += amount
	pm.rebuildPotsIncremental(players)
}

// getTotalPot returns the total amount across all pots
func (pm *potManager) getTotalPot() int64 {
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

// rebuildPotsIncremental rebuilds pots based on current TotalBets and player status.
// Called after each bet to maintain proper side-pot structure.
func (pm *potManager) rebuildPotsIncremental(players []*Player) {
	n := len(players)
	if n == 0 {
		pm.pots = []*pot{newPot(0)}
		return
	}

	// Collect unique positive bet thresholds.
	seen := make(map[int64]struct{}, n)
	for i := 0; i < n; i++ {
		if b := pm.totalBets[i]; b > 0 {
			seen[b] = struct{}{}
		}
	}

	// If everyone is at 0, a single empty pot is enough.
	if len(seen) == 0 {
		pm.pots = []*pot{newPot(n)}
		return
	}

	// Sort thresholds ascending.
	levels := make([]int64, 0, len(seen))
	for b := range seen {
		levels = append(levels, b)
	}
	sort.Slice(levels, func(i, j int) bool { return levels[i] < levels[j] })

	pots := make([]*pot, 0, len(levels)+1)
	prev := int64(0)

	// Build capped pots for each threshold.
	for _, lvl := range levels {
		p := newPot(n)
		amt := int64(0)

		for i := 0; i < n; i++ {
			tb := pm.totalBets[i]
			// Eligible if not folded and contributed at least to this level.
			if players[i] != nil && !(players[i].GetCurrentStateString() == "FOLDED") && tb >= lvl {
				p.makeEligible(i)
			}
			// Contribution into this layer is clamp(tb, prev..lvl) - prev.
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

	// Final uncapped overage pot above the highest level (e.g., raises not all-in capped).
	top := levels[len(levels)-1]
	over := newPot(n)
	hasOver := false

	for i := 0; i < n; i++ {
		tb := pm.totalBets[i]
		if tb > top {
			over.amount += tb - top
			if players[i] != nil && !(players[i].GetCurrentStateString() == "FOLDED") {
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
			if elig && players[idx] != nil && !(players[idx].GetCurrentStateString() == "FOLDED") {
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

// ReturnUncalledBet returns any uncalled portion of a bet to the player who made it
func (pm *potManager) returnUncalledBet(players []*Player) error {
	var hi, second int64
	hiPlayer := -1

	for idx, bet := range pm.currentBets {
		if bet > hi {
			second = hi
			hi = bet
			hiPlayer = idx
		} else if bet > second {
			second = bet
		}
	}

	if hiPlayer >= 0 && hi > second {
		uncalled := hi - second
		if players[hiPlayer] != nil {
			if err := players[hiPlayer].credit(uncalled); err != nil {
				return fmt.Errorf("failed to return uncalled bet to player %d: %w", hiPlayer, err)
			}
		}
		pm.currentBets[hiPlayer] -= uncalled
		pm.totalBets[hiPlayer] -= uncalled

		// Rebuild pots after refund to reflect the new totals
		pm.rebuildPotsIncremental(players)
	}
	return nil
}
