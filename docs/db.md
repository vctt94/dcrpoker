## DB
```mermaid
erDiagram
  players ||--o{ transactions : has
  players ||--o{ table_participants : sits_in
  players ||--o{ hand_players : participates
  players ||--o{ table_buyins : buys_in
  players ||--o{ table_cashouts : cashes_out
  tables  ||--o{ table_participants : has
  tables  ||--o{ table_buyins : has
  tables  ||--o{ table_cashouts : has
  tables  ||--o{ hands : deals
  tables  ||--o| table_snapshots : snapshot
  hands   ||--o{ hand_players : has
  hands   ||--o{ actions : contains
  hands   ||--o{ board_cards : has

  players {
    string   id PK
    string   name
    int      balance
    datetime created_at
  }

  transactions {
    int      id PK
    string   player_id
    int      amount
    string   type
    string   description
    datetime created_at
  }

  tables {
    string   id PK
    string   host_id
    int      buy_in
    int      min_players
    int      max_players
    int      small_blind
    int      big_blind
    int      min_balance
    int      starting_chips
    int      timebank_ms
    int      autostart_ms
    int      auto_advance_ms
    datetime created_at
  }

  table_participants {
    string   table_id PK
    string   player_id PK
    int      seat
    datetime joined_at
    datetime left_at
    boolean  ready
  }

  table_buyins {
    int      id PK
    string   table_id
    string   player_id
    int      amount
    datetime created_at
  }

  table_cashouts {
    int      id PK
    string   table_id
    string   player_id
    int      amount
    datetime created_at
  }

  hands {
    int      id PK
    string   table_id
    int      hand_no
    datetime started_at
    datetime ended_at
    int      dealer_seat
    int      sb_seat
    int      bb_seat
    string   result_json
  }

  hand_players {
    int      hand_id PK
    string   player_id PK
    int      seat
    int      starting_stack
    string   hole_cards_json
  }

  actions {
    int      id PK
    int      hand_id
    int      ord
    string   street
    int      actor_seat
    string   action
    int      amount
    boolean  is_allin
    datetime created_at
  }

  board_cards {
    int      hand_id PK
    string   street PK
    string   cards_json
  }

  table_snapshots {
    string   table_id PK
    datetime snapshot_at
    blob     payload
  }

```

## Events

The system uses an event-driven architecture with snapshot-based persistence.

### Event Processing Flow

```mermaid
sequenceDiagram
  autonumber
  participant C as Client
  participant T as TableServer
  participant EP as EventProcessor
  participant NH as NotificationHandler
  participant GH as GameStateHandler
  participant PH as PersistenceHandler
  participant DB as Database

  C->>T: RPC calls (JoinTable, MakeBet, CallBet, etc.)
  T->>T: Direct table operations
  T->>EP: PublishEvent(event)
  EP->>NH: Process notifications
  EP->>GH: Process game state updates
  EP->>PH: Process persistence
  NH-->>C: Broadcast notifications
  GH-->>C: Update game streams
  PH->>DB: UpsertSnapshot(table_snapshots)
```

### Event Types

Events are `NotificationType` enum values from the protobuf definition:
- `TABLE_CREATED`, `TABLE_REMOVED`
- `PLAYER_JOINED`, `PLAYER_LEFT`, `PLAYER_READY`
- `GAME_STARTED`, `GAME_ENDED`, `NEW_HAND_STARTED`
- `BET_MADE`, `CALL_MADE`, `CHECK_MADE`, `PLAYER_FOLDED`
- `SHOWDOWN_RESULT`, `PLAYER_ALL_IN`

### Event Processing Architecture

1. **EventProcessor**: Manages a queue of events with worker goroutines
2. **NotificationHandler**: Broadcasts events to connected clients
3. **GameStateHandler**: Updates game state streams for real-time UI
4. **PersistenceHandler**: Saves table snapshots for fast restoration

## State Machine

### Game
```mermaid
%% Game FSM — complete state flow with all intermediate states
stateDiagram-v2
  [*] --> NEW_HAND_DEALING

  NEW_HAND_DEALING: Wait for evStartHand
  NEW_HAND_DEALING --> PRE_DEAL: evStartHand

  PRE_DEAL: Setup positions, post blinds, init hand
  PRE_DEAL --> DEAL: automatic

  DEAL: Deal hole cards to players
  DEAL --> BLINDS: automatic

  BLINDS: Set current player to act
  BLINDS --> PRE_FLOP: automatic

  PRE_FLOP: Pre-flop betting round
  PRE_FLOP --> FLOP: evAdvance (betting complete)
  PRE_FLOP --> SHOWDOWN: evGotoShowdown (fold/win)

  FLOP: Deal flop (3 cards), betting round
  FLOP --> TURN: evAdvance (betting complete)
  FLOP --> SHOWDOWN: evGotoShowdown (fold/win)

  TURN: Deal turn (4th card), betting round
  TURN --> RIVER: evAdvance (betting complete)
  TURN --> SHOWDOWN: evGotoShowdown (fold/win)

  RIVER: Deal river (5th card), betting round
  RIVER --> SHOWDOWN: evAdvance / evGotoShowdown

  SHOWDOWN: Evaluate hands, distribute pots
  SHOWDOWN --> PRE_DEAL: evStartHand (next hand)
  SHOWDOWN --> END: no evStartHand

  END: Terminal state
  END --> [*]
```

### Game State Responsibilities

- **NEW_HAND_DEALING**: Initial state, waits for `evStartHand` to begin hand
- **PRE_DEAL**: Advances dealer button, calculates blind positions, posts blinds, initializes `currentHand`, starts player hand participation FSMs
- **DEAL**: Deals hole cards from the deck to all active players
- **BLINDS**: Determines first player to act based on blind positions
- **PRE_FLOP**: First betting round with hole cards only
- **FLOP**: Deals 3 community cards, second betting round
- **TURN**: Deals 4th community card, third betting round  
- **RIVER**: Deals 5th community card, final betting round
- **SHOWDOWN**: Hand evaluation, pot distribution, winner determination
- **END**: Terminal state (game stopped)

## Table State Machine

The table tracks lobby readiness and coordinates game lifecycle.

```mermaid
stateDiagram-v2
  [*] --> WAITING_FOR_PLAYERS

  WAITING_FOR_PLAYERS: Not enough ready players
  WAITING_FOR_PLAYERS --> WAITING_FOR_PLAYERS: evUsersChanged (not ready)
  WAITING_FOR_PLAYERS --> PLAYERS_READY: evUsersChanged (all ready)
  WAITING_FOR_PLAYERS --> GAME_ACTIVE: evStartGameReq (if all ready)
  WAITING_FOR_PLAYERS --> WAITING_FOR_PLAYERS: evGameEnded

  PLAYERS_READY: Min players ready, can start
  PLAYERS_READY --> WAITING_FOR_PLAYERS: evUsersChanged (not ready)
  PLAYERS_READY --> PLAYERS_READY: evUsersChanged (still ready)
  PLAYERS_READY --> GAME_ACTIVE: evStartGameReq
  PLAYERS_READY --> WAITING_FOR_PLAYERS: evGameEnded

  GAME_ACTIVE: Game is running
  GAME_ACTIVE --> WAITING_FOR_PLAYERS: evGameEnded
```

### Table State Responsibilities

- **WAITING_FOR_PLAYERS**: Initial state. Waits for enough players to be seated and ready. Responds to player join/ready events (`evUsersChanged`).
- **PLAYERS_READY**: All conditions met to start a game. Server can call `StartGame()` which sends `evStartGameReq` to transition to GAME_ACTIVE.
- **GAME_ACTIVE**: Game is running (hands are being played). Remains active across multiple hands. Returns to WAITING_FOR_PLAYERS on `evGameEnded`.

## Player State Machines

Each player has **two independent state machines** that run concurrently:

### 1. Table Presence FSM
Tracks whether the player is seated at the table. Lives for the entire session.

```mermaid
stateDiagram-v2
  [*] --> SEATED

  SEATED: Player seated at table
  SEATED --> SEATED: evReady
  SEATED --> SEATED: evDeductBlind
  SEATED --> SEATED: evBalanceNotification
  SEATED --> SEATED: evDisconnect
  SEATED --> SEATED: evStartHand, evStartTurn, evEndTurn, evBet, evCall, evFoldReq, evEndHand
  SEATED --> LEFT: evLeave

  LEFT: Player left table (terminal)
  LEFT --> [*]
```

### 2. Hand Participation FSM
Tracks player actions during a single hand. Created when hand starts, destroyed when hand ends.

```mermaid
stateDiagram-v2
  [*] --> ACTIVE

  ACTIVE: Player active in hand, can bet/fold
  ACTIVE --> ACTIVE: evStartTurn
  ACTIVE --> ACTIVE: evEndTurn
  ACTIVE --> ACTIVE: evCall
  ACTIVE --> ACTIVE: evCallDelta
  ACTIVE --> ALL_IN: evBet (balance=0)
  ACTIVE --> ALL_IN: evCallDelta (balance=0)
  ACTIVE --> FOLDED: evFoldReq
  ACTIVE --> [*]: evEndHand

  ALL_IN: Player all-in, cannot act
  ALL_IN --> ALL_IN: evStartTurn
  ALL_IN --> ALL_IN: evEndTurn
  ALL_IN --> [*]: evEndHand

  FOLDED: Player folded, out of hand
  FOLDED --> FOLDED: evStartTurn
  FOLDED --> FOLDED: evEndTurn
  FOLDED --> [*]: evEndHand
```

### Player State Machine Responsibilities

**Table Presence FSM:**
- **SEATED**: Handles readiness, blind deductions (game-commanded), chip additions (winnings/refunds), and disconnect events. Forwards hand-related events to Hand Participation FSM if active.
- **LEFT**: Terminal state when player leaves table.

**Hand Participation FSM:**
- **ACTIVE**: Player can make betting decisions (bet, call, fold). Sets `isTurn` flag when it's player's turn. Auto-transitions to ALL_IN when balance reaches zero.
- **ALL_IN**: Player has committed all chips, cannot make further actions. Passively waits for hand completion.
- **FOLDED**: Player has folded, out of the hand. Sets `hasFolded` flag and waits for hand completion.

## State Machine Coordination

The three layers of state machines coordinate to manage the complete poker game lifecycle:

```
Table FSM (Lobby/Session)
    │
    ├─ Manages: Player readiness, game lifecycle
    │
    └─► Game FSM (Hand Progression)
         │
         ├─ Manages: Dealer rotation, blinds, betting rounds, deck
         │
         └─► Player FSMs (Individual Actions)
              │
              ├─ Table Presence: Seated/Left (session-level)
              └─ Hand Participation: Active/AllIn/Folded (hand-level)
```
