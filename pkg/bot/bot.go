package bot

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/companyzero/bisonrelay/clientrpc/types"
	"github.com/companyzero/bisonrelay/zkidentity"
	"github.com/decred/dcrd/dcrutil/v4"
	kit "github.com/vctt94/bisonbotkit"
	"github.com/vctt94/pokerbisonrelay/pkg/server"
)

const STARTING_CHIPS = 1000

// State holds the state of the poker bot
type State struct {
	db server.Database
	mu sync.RWMutex
}

// NewState creates a new bot state with the given database
func NewState(db server.Database) *State {
	return &State{
		db: db,
	}
}

// HandlePM handles incoming PM commands.
func (s *State) HandlePM(ctx context.Context, bot *kit.Bot, pm *types.ReceivedPM) {
	tokens := strings.Fields(pm.Msg.Message)
	if len(tokens) == 0 {
		return
	}

	cmd := strings.ToLower(tokens[0])
	var uid zkidentity.ShortID
	uid.FromBytes(pm.Uid)
	playerID := uid.String()

	switch cmd {
	case "balance":
		balance, err := s.db.GetPlayerBalance(ctx, playerID)
		if err != nil {
			bot.SendPM(ctx, pm.Nick, "Error checking balance: "+err.Error())
			return
		}
		bot.SendPM(ctx, pm.Nick, fmt.Sprintf("Your current balance is: %.8f DCR",
			dcrutil.Amount(balance).ToCoin()))

	case "help":
		s.handleHelp(ctx, bot, pm)

	default:
		bot.SendPM(ctx, pm.Nick, "Unknown command. Type 'help' for available commands.")
	}
}

func (s *State) handleHelp(ctx context.Context, bot *kit.Bot, pm *types.ReceivedPM) {
	helpMsg := `Available commands:
- balance: Check your current balance
- create <amount> [starting-chips]: Create a new poker table with specified buy-in and optional starting chips (default: 1000)
- join <table-id>: Join an existing poker table
- tables: List all active tables
- help: Show this help message`
	bot.SendPM(ctx, pm.Nick, helpMsg)
}
