package poker

import (
	"encoding/json"
	"math/rand"
	"testing"
)

func TestNewDeck(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	deck := newDeck(rng)

	// Check deck size
	if deck.Size() != 52 {
		t.Errorf("Expected deck size 52, got %d", deck.Size())
	}

	// Check that all cards are unique
	seen := make(map[Card]bool)
	for _, card := range deck.cards {
		if seen[card] {
			t.Errorf("Duplicate card found: %v", card)
		}
		seen[card] = true
	}

	// Check that all suits and values are present
	suitCount := make(map[Suit]int)
	valueCount := make(map[Value]int)
	for _, card := range deck.cards {
		suitCount[card.suit]++
		valueCount[card.value]++
	}

	// Check suit distribution
	for suit, count := range suitCount {
		if count != 13 {
			t.Errorf("Expected 13 cards of suit %v, got %d", suit, count)
		}
	}

	// Check value distribution
	for value, count := range valueCount {
		if count != 4 {
			t.Errorf("Expected 4 cards of value %v, got %d", value, count)
		}
	}
}

func TestDeckShuffle(t *testing.T) {
	// Create two decks with the same seed
	rng1 := rand.New(rand.NewSource(42))
	rng2 := rand.New(rand.NewSource(42))
	deck1 := newDeck(rng1)
	deck2 := newDeck(rng2)

	// Both decks should have the same order
	for i := 0; i < 52; i++ {
		if deck1.cards[i] != deck2.cards[i] {
			t.Errorf("Decks with same seed should have same order at position %d", i)
		}
	}

	// Create a deck with a different seed
	rng3 := rand.New(rand.NewSource(43))
	deck3 := newDeck(rng3)

	// This deck should have a different order
	sameOrder := true
	for i := 0; i < 52; i++ {
		if deck1.cards[i] != deck3.cards[i] {
			sameOrder = false
			break
		}
	}
	if sameOrder {
		t.Error("Decks with different seeds should have different orders")
	}
}

func TestDeckDraw(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	deck := newDeck(rng)

	// Draw all cards
	for i := 0; i < 52; i++ {
		card, ok := deck.Draw()
		if !ok {
			t.Errorf("Expected to draw card %d, but deck was empty", i)
		}
		if deck.Size() != 51-i {
			t.Errorf("Expected deck size %d after drawing, got %d", 51-i, deck.Size())
		}
		// Verify the card is valid
		if card.suit == "" || card.value == "" {
			t.Errorf("Drawn card %d is invalid: %v", i, card)
		}
	}

	// Try to draw from empty deck
	card, ok := deck.Draw()
	if ok {
		t.Error("Expected to fail drawing from empty deck")
	}
	if card != (Card{}) {
		t.Error("Expected zero value card when drawing from empty deck")
	}
}

// TestCardJSONSerialization tests that cards can be properly serialized and deserialized
func TestCardJSONSerialization(t *testing.T) {
	// Test cases with various card combinations
	testCases := []struct {
		name string
		card Card
	}{
		{"Ace of Spades", NewCardFromSuitValue(Spades, Ace)},
		{"King of Hearts", NewCardFromSuitValue(Hearts, King)},
		{"Ten of Diamonds", NewCardFromSuitValue(Diamonds, Ten)},
		{"Two of Clubs", NewCardFromSuitValue(Clubs, Two)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Serialize to JSON
			jsonData, err := json.Marshal(tc.card)
			if err != nil {
				t.Fatalf("Failed to marshal card: %v", err)
			}

			// Deserialize from JSON
			var deserializedCard Card
			err = json.Unmarshal(jsonData, &deserializedCard)
			if err != nil {
				t.Fatalf("Failed to unmarshal card: %v", err)
			}

			// Verify the card is the same
			if deserializedCard.GetSuit() != tc.card.GetSuit() {
				t.Errorf("Suit mismatch: expected %s, got %s", tc.card.GetSuit(), deserializedCard.GetSuit())
			}
			if deserializedCard.GetValue() != tc.card.GetValue() {
				t.Errorf("Value mismatch: expected %s, got %s", tc.card.GetValue(), deserializedCard.GetValue())
			}
		})
	}
}

// TestHandSerialization tests that a full hand can be serialized and deserialized
func TestHandSerialization(t *testing.T) {
	// Create a test hand
	hand := []Card{
		NewCardFromSuitValue(Spades, Ace),
		NewCardFromSuitValue(Hearts, King),
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(hand)
	if err != nil {
		t.Fatalf("Failed to marshal hand: %v", err)
	}

	// Deserialize from JSON
	var deserializedHand []Card
	err = json.Unmarshal(jsonData, &deserializedHand)
	if err != nil {
		t.Fatalf("Failed to unmarshal hand: %v", err)
	}

	// Verify the hand is the same
	if len(deserializedHand) != len(hand) {
		t.Fatalf("Hand length mismatch: expected %d, got %d", len(hand), len(deserializedHand))
	}

	for i, card := range hand {
		if deserializedHand[i].GetSuit() != card.GetSuit() {
			t.Errorf("Card %d suit mismatch: expected %s, got %s", i, card.GetSuit(), deserializedHand[i].GetSuit())
		}
		if deserializedHand[i].GetValue() != card.GetValue() {
			t.Errorf("Card %d value mismatch: expected %s, got %s", i, card.GetValue(), deserializedHand[i].GetValue())
		}
	}
}

// testRNG creates a deterministic RNG for testing
func testRNG() *rand.Rand {
	return rand.New(rand.NewSource(42))
}

// TestDeckStateSerialization tests that deck state can be properly saved and restored
func TestDeckStateSerialization(t *testing.T) {
	// Create a deck and draw some cards
	rng := testRNG()
	deck := newDeck(rng)

	// Draw a few cards to change the deck state
	_, _ = deck.Draw()
	_, _ = deck.Draw()
	_, _ = deck.Draw()

	originalSize := deck.Size()

	// Get the deck state
	state := deck.GetState()

	// Serialize to JSON
	jsonData, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Failed to marshal deck state: %v", err)
	}

	// Deserialize from JSON
	var deserializedState DeckState
	err = json.Unmarshal(jsonData, &deserializedState)
	if err != nil {
		t.Fatalf("Failed to unmarshal deck state: %v", err)
	}

	// Create a new deck from the deserialized state
	newRng := testRNG()
	restoredDeck, err := NewDeckFromState(&deserializedState, newRng)
	if err != nil {
		t.Fatalf("Failed to create deck from state: %v", err)
	}

	// Verify the deck has the same size
	if restoredDeck.Size() != originalSize {
		t.Errorf("Deck size mismatch: expected %d, got %d", originalSize, restoredDeck.Size())
	}

	// Verify we can draw the same number of cards
	for i := 0; i < originalSize; i++ {
		card, ok := restoredDeck.Draw()
		if !ok {
			t.Errorf("Failed to draw card %d from restored deck", i)
			break
		}
		if card.GetSuit() == "" || card.GetValue() == "" {
			t.Errorf("Invalid card drawn from restored deck: %s", card.String())
		}
	}
}
