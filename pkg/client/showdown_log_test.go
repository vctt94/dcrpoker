package client

import (
	"testing"

	"github.com/vctt94/pokerbisonrelay/pkg/rpc/grpc/pokerrpc"
)

func TestBuildShowdownLogLines(t *testing.T) {
	ntfn := &pokerrpc.Notification{
		TableId: "table-1",
		Showdown: &pokerrpc.Showdown{
			HandId: "hand-9",
			Round:  9,
			Pot:    120,
			Board: []*pokerrpc.Card{
				{Value: "A", Suit: "Spades"},
				{Value: "K", Suit: "Hearts"},
				{Value: "10", Suit: "Clubs"},
			},
			Winners: []*pokerrpc.Winner{
				{
					PlayerId: "hero",
					HandRank: pokerrpc.HandRank_STRAIGHT,
					Winnings: 120,
				},
			},
			Players: []*pokerrpc.ShowdownPlayer{
				{
					PlayerId:   "hero",
					Name:       "Hero",
					HoleCards:  []*pokerrpc.Card{{Value: "Q", Suit: "Diamonds"}, {Value: "J", Suit: "Clubs"}},
					FinalState: pokerrpc.PlayerState_PLAYER_STATE_IN_GAME,
				},
				{
					PlayerId:   "villain",
					Name:       "Villain",
					HoleCards:  []*pokerrpc.Card{{Value: "2", Suit: "Spades"}, {Value: "2", Suit: "Hearts"}},
					FinalState: pokerrpc.PlayerState_PLAYER_STATE_FOLDED,
				},
			},
		},
	}

	lines := buildShowdownLogLines(ntfn, nil, "hero")
	if len(lines) != 3 {
		t.Fatalf("expected 3 showdown log lines, got %d: %v", len(lines), lines)
	}

	if got := lines[0]; got != "showdown table=table-1 hand=hand-9 round=9 pot=120 board=As Kh Tc" {
		t.Fatalf("unexpected showdown header: %q", got)
	}
	if got := lines[1]; got != "showdown winners=Hero(+120, Straight)" {
		t.Fatalf("unexpected winner summary: %q", got)
	}
	if got := lines[2]; got != "showdown hands=*Hero[Qd Jc; in-game], Villain[2s 2h; folded]" {
		t.Fatalf("unexpected hand summary: %q", got)
	}
}

func TestBuildShowdownLogLinesFallsBackToLastGameUpdate(t *testing.T) {
	ntfn := &pokerrpc.Notification{
		TableId: "table-2",
		Showdown: &pokerrpc.Showdown{
			HandId: "hand-10",
			Round:  10,
			Pot:    75,
			Winners: []*pokerrpc.Winner{
				{
					PlayerId: "hero",
					HandRank: pokerrpc.HandRank_PAIR,
					Winnings: 75,
				},
			},
			Players: []*pokerrpc.ShowdownPlayer{
				{
					PlayerId:   "hero",
					FinalState: pokerrpc.PlayerState_PLAYER_STATE_IN_GAME,
				},
			},
		},
	}
	lastUpdate := &pokerrpc.GameUpdate{
		Players: []*pokerrpc.Player{
			{
				Id:   "hero",
				Name: "Hero",
				Hand: []*pokerrpc.Card{
					{Value: "A", Suit: "Spades"},
					{Value: "A", Suit: "Clubs"},
				},
			},
		},
		CommunityCards: []*pokerrpc.Card{
			{Value: "K", Suit: "Hearts"},
			{Value: "Q", Suit: "Diamonds"},
			{Value: "J", Suit: "Clubs"},
		},
	}

	lines := buildShowdownLogLines(ntfn, lastUpdate, "hero")
	if len(lines) != 3 {
		t.Fatalf("expected 3 showdown log lines with fallback state, got %d: %v", len(lines), lines)
	}

	if got := lines[0]; got != "showdown table=table-2 hand=hand-10 round=10 pot=75 board=Kh Qd Jc" {
		t.Fatalf("unexpected fallback showdown header: %q", got)
	}
	if got := lines[1]; got != "showdown winners=Hero(+75, Pair)" {
		t.Fatalf("unexpected fallback winner summary: %q", got)
	}
	if got := lines[2]; got != "showdown hands=*Hero[As Ac; in-game]" {
		t.Fatalf("unexpected fallback hand summary: %q", got)
	}
}

func TestBuildShowdownLogLinesSkipsWatcherLogs(t *testing.T) {
	ntfn := &pokerrpc.Notification{
		TableId: "table-watch",
		Showdown: &pokerrpc.Showdown{
			HandId: "hand-watch",
			Round:  3,
			Pot:    45,
			Winners: []*pokerrpc.Winner{
				{
					PlayerId: "p1",
					HandRank: pokerrpc.HandRank_PAIR,
					Winnings: 45,
				},
			},
		},
	}
	lastUpdate := &pokerrpc.GameUpdate{
		Players: []*pokerrpc.Player{
			{Id: "p1", Name: "Player 1"},
			{Id: "p2", Name: "Player 2"},
		},
	}

	lines := buildShowdownLogLines(ntfn, lastUpdate, "watcher")
	if lines != nil {
		t.Fatalf("expected no showdown logs for watcher, got %v", lines)
	}
}
