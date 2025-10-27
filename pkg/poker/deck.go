package poker

import (
	"encoding/json"
	"fmt"
	"math/rand"
)

// Suit represents a card suit
type Suit string

const (
	Spades   Suit = "♠"
	Hearts   Suit = "♥"
	Diamonds Suit = "♦"
	Clubs    Suit = "♣"
)

// Value represents a card value
type Value string

const (
	Ace   Value = "A"
	Two   Value = "2"
	Three Value = "3"
	Four  Value = "4"
	Five  Value = "5"
	Six   Value = "6"
	Seven Value = "7"
	Eight Value = "8"
	Nine  Value = "9"
	Ten   Value = "10"
	Jack  Value = "J"
	Queen Value = "Q"
	King  Value = "K"
)

// Card represents a playing card
type Card struct {
	suit  Suit
	value Value
}

// CardJSON represents a card for JSON serialization
type CardJSON struct {
	Suit  string `json:"suit"`
	Value string `json:"value"`
}

// MarshalJSON implements json.Marshaler interface for Card
func (c Card) MarshalJSON() ([]byte, error) {
	return json.Marshal(CardJSON{
		Suit:  string(c.suit),
		Value: string(c.value),
	})
}

// UnmarshalJSON implements json.Unmarshaler interface for Card
func (c *Card) UnmarshalJSON(data []byte) error {
	var cardJSON CardJSON
	if err := json.Unmarshal(data, &cardJSON); err != nil {
		return err
	}

	// Validate and convert suit
	switch cardJSON.Suit {
	case "♠", "s", "S", "spades", "Spades":
		c.suit = Spades
	case "♥", "h", "H", "hearts", "Hearts":
		c.suit = Hearts
	case "♦", "d", "D", "diamonds", "Diamonds":
		c.suit = Diamonds
	case "♣", "c", "C", "clubs", "Clubs":
		c.suit = Clubs
	default:
		return fmt.Errorf("invalid suit: %s", cardJSON.Suit)
	}

	// Validate and convert value
	switch cardJSON.Value {
	case "A", "a", "ace", "Ace":
		c.value = Ace
	case "K", "k", "king", "King":
		c.value = King
	case "Q", "q", "queen", "Queen":
		c.value = Queen
	case "J", "j", "jack", "Jack":
		c.value = Jack
	case "10", "T", "t", "ten", "Ten":
		c.value = Ten
	case "9", "nine", "Nine":
		c.value = Nine
	case "8", "eight", "Eight":
		c.value = Eight
	case "7", "seven", "Seven":
		c.value = Seven
	case "6", "six", "Six":
		c.value = Six
	case "5", "five", "Five":
		c.value = Five
	case "4", "four", "Four":
		c.value = Four
	case "3", "three", "Three":
		c.value = Three
	case "2", "two", "Two":
		c.value = Two
	default:
		return fmt.Errorf("invalid value: %s", cardJSON.Value)
	}

	return nil
}

// String returns a string representation of the card
func (c Card) String() string {
	return string(c.value) + string(c.suit)
}

// GetSuit returns the card's suit
func (c Card) GetSuit() string {
	return string(c.suit)
}

// GetValue returns the card's value
func (c Card) GetValue() string {
	return string(c.value)
}

// Deck represents a deck of cards
type Deck struct {
	cards []Card
	rng   *rand.Rand
}

// NewDeck creates a new deck of cards with the given random number generator
func NewDeck(rng *rand.Rand) *Deck {
	deck := &Deck{
		cards: make([]Card, 0, 52),
		rng:   rng,
	}

	// Create all 52 cards
	suits := []Suit{Spades, Hearts, Diamonds, Clubs}
	values := []Value{Ace, Two, Three, Four, Five, Six, Seven, Eight, Nine, Ten, Jack, Queen, King}

	for _, suit := range suits {
		for _, value := range values {
			deck.cards = append(deck.cards, Card{suit: suit, value: value})
		}
	}

	// Shuffle the deck
	deck.Shuffle()

	return deck
}

// Shuffle randomizes the order of cards in the deck
func (d *Deck) Shuffle() {
	d.rng.Shuffle(len(d.cards), func(i, j int) {
		d.cards[i], d.cards[j] = d.cards[j], d.cards[i]
	})
}

// Draw removes and returns the top card from the deck
func (d *Deck) Draw() (Card, bool) {
	if len(d.cards) == 0 {
		return Card{}, false
	}
	card := d.cards[0]
	d.cards = d.cards[1:]
	return card, true
}

// Size returns the number of cards remaining in the deck
func (d *Deck) Size() int {
	return len(d.cards)
}

// NewCardFromSuitValue creates a new Card with the given suit and value
// This is needed because Card fields are unexported
func NewCardFromSuitValue(suit Suit, value Value) Card {
	return Card{suit: suit, value: value}
}

// initializeDeck creates a new deck of cards
func initializeDeck() []Card {
	// Create a new deck with a deterministic seed for testing
	rng := rand.New(rand.NewSource(42))
	deck := NewDeck(rng)

	// Convert to slice for compatibility with existing code
	cards := make([]Card, len(deck.cards))
	copy(cards, deck.cards)
	return cards
}

// GetCards returns the remaining cards in the deck (for persistence)
func (d *Deck) GetCards() []Card {
	return d.cards
}

// SetCards sets the remaining cards in the deck (for restoration)
func (d *Deck) SetCards(cards []Card) {
	d.cards = make([]Card, len(cards))
	copy(d.cards, cards)
}

// NewDeckFromCards creates a deck from a specific set of cards (for restoration)
func NewDeckFromCards(cards []Card, rng *rand.Rand) *Deck {
	deck := &Deck{
		cards: make([]Card, len(cards)),
		rng:   rng,
	}
	copy(deck.cards, cards)
	return deck
}

// DeckState represents the serializable state of a deck
type DeckState struct {
	RemainingCards []Card `json:"remaining_cards"`
	Seed           int64  `json:"seed,omitempty"` // Optional: for deterministic restoration
}

// GetState returns the current state of the deck for persistence
func (d *Deck) GetState() *DeckState {
	cards := make([]Card, len(d.cards))
	copy(cards, d.cards)
	return &DeckState{
		RemainingCards: cards,
	}
}

// RestoreState restores the deck from a saved state
func (d *Deck) RestoreState(state *DeckState) error {
	if state == nil {
		return fmt.Errorf("deck state is nil")
	}

	// Restore the remaining cards
	d.cards = make([]Card, len(state.RemainingCards))
	copy(d.cards, state.RemainingCards)

	return nil
}

// NewDeckFromState creates a new deck from a saved state
func NewDeckFromState(state *DeckState, rng *rand.Rand) (*Deck, error) {
	if state == nil {
		return nil, fmt.Errorf("deck state is nil")
	}

	deck := &Deck{
		cards: make([]Card, len(state.RemainingCards)),
		rng:   rng,
	}
	copy(deck.cards, state.RemainingCards)

	return deck, nil
}
