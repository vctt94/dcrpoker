package poker

import (
	"fmt"
	"sync"
	"time"
)

// Hole represents a player's hole cards for a single hand
type Hole struct {
	cards [2]Card
	count int // number of cards dealt (0, 1, or 2)
}

// NewHole creates a new empty Hole
func NewHole() *Hole {
	return &Hole{
		cards: [2]Card{},
		count: 0,
	}
}

// AddCard adds a card to the hole (max 2 cards)
func (h *Hole) AddCard(c Card) error {
	if h.count >= 2 {
		return fmt.Errorf("hole already has 2 cards")
	}
	h.cards[h.count] = c
	h.count++
	return nil
}

// GetCards returns the hole cards (visible only per retention policy)
func (h *Hole) GetCards() []Card {
	if h.count == 0 {
		return nil
	}
	return h.cards[:h.count]
}

// Clear purges the hole cards (called during cleanup)
func (h *Hole) Clear() {
	h.cards = [2]Card{}
	h.count = 0
}

// Hand represents all per-hand state, owned by Game
// Created at hand start; destroyed after cleanup
type Hand struct {
	id        string
	hole      map[string]*Hole // playerID -> private hole cards
	createdAt time.Time

	mu sync.RWMutex
}

// NewHand creates a new Hand for the given players
func NewHand(playerIDs []string) *Hand {
	h := &Hand{
		id:        fmt.Sprintf("hand_%d", time.Now().UnixNano()),
		hole:      make(map[string]*Hole),
		createdAt: time.Now(),
	}

	// Initialize empty Hole for each player
	for _, pid := range playerIDs {
		h.hole[pid] = NewHole()
	}

	return h
}

// DealCardToPlayer deals a card to a specific player
func (h *Hand) DealCardToPlayer(playerID string, card Card) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	hole, ok := h.hole[playerID]
	if !ok {
		return fmt.Errorf("player %s not in hand", playerID)
	}

	return hole.AddCard(card)
}

// GetPlayerCards returns the cards for a specific player (respecting visibility)
// GetPlayerCards returns the cards for a player.
// Visibility/auth filtering is handled by the server layer.
func (h *Hand) GetPlayerCards(playerID string) []Card {
	h.mu.RLock()
	defer h.mu.RUnlock()

	hole, ok := h.hole[playerID]
	if !ok {
		return nil
	}

	// Return cards if they exist - server layer handles visibility/auth
	// Cards are publicly visible if revealed during showdown
	return hole.GetCards()
}

// CleanupHoleCards purges all hole card data (called after settlement)
func (h *Hand) CleanupHoleCards() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, hole := range h.hole {
		hole.Clear()
	}

}
