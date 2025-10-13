## DB
```mermaid
erDiagram
  players ||--o{ transactions : has
  players ||--o{ table_participants : sits_in
  tables  ||--o{ table_participants : has
  tables  ||--o{ table_buyins : has
  tables  ||--o{ table_cashouts : has
  tables  ||--o{ hands : deals
  hands   ||--o{ events : emits
  tables  ||--o{ events : emits_table
  tables  ||--|| table_snapshots : snapshot

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
    int      min_players
    int      max_players
    int      starting_chips
    int      small_blind
    int      big_blind
    int      timebank_ms
    int      autostart_ms
    int      seed
    datetime created_at
  }

  table_participants {
    int      id PK
    string   table_id
    string   player_id
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
    int      deck_seed
    string   deck_commit
  }

  events {
    int      id PK
    string   table_id
    int      hand_id
    int      ord
    string   type
    string   payload
    datetime created_at
  }

  table_snapshots {
    int      id PK
    string   table_id
    datetime snapshot_at
    string   payload
  }

```

## Events

```mermaid
sequenceDiagram
  autonumber
  participant C as Client
  participant T as TableServer
  participant G as GameFSM
  participant DB as DBEvents

  C->>T: SeatPlayer | SetReady | StartHand | Bet | Call | Check | Fold
  T->>G: Dispatch event
  G->>G: Mutate in memory state
  G-->>T: Send notifications
  T->>DB: AppendEvent tableId handId type payload
  DB-->>T: Return ord

  T->>DB: LoadSnapshot tableId
  DB-->>T: Snapshot blob
  T->>G: Hydrate from snapshot or init fresh
  T->>DB: EventsFrom tableId fromOrd
  DB-->>T: Ordered events
  T->>G: Replay events to reducer


```

## State Machine
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

  note right of WAITING_FOR_PLAYERS
    Checks allPlayersReady():
    - At least MinPlayers seated
    - All users have IsReady=true
  end note

  note right of GAME_ACTIVE
    StartGame() sends evStartGameReq
    which triggers Game FSM startup.
    Subsequent hands stay in GAME_ACTIVE.
  end note
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
  SEATED --> SEATED: evReady, evDeductBlind, evAddChips, evDisconnect
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
  ACTIVE --> ACTIVE: evStartTurn, evEndTurn, evBet, evCallDelta
  ACTIVE --> ALL_IN: balance=0 (from bet/call)
  ACTIVE --> FOLDED: evFoldReq
  ACTIVE --> [*]: evEndHand

  ALL_IN: Player all-in, cannot act
  ALL_IN --> [*]: evEndHand

  FOLDED: Player folded, out of hand
  FOLDED --> [*]: evEndHand

  note right of ACTIVE
    Auto-transitions to ALL_IN
    when balance reaches 0
    after any bet or call
  end note

  note right of ALL_IN
    Initialized directly to ALL_IN
    if player is all-in from
    posting blinds (HandleStartHand)
  end note
```

### Player State Machine Responsibilities

**Table Presence FSM:**
- **SEATED**: Handles readiness, blind deductions (game-commanded), chip additions (winnings/refunds), and disconnect events. Forwards hand-related events to Hand Participation FSM if active.
- **LEFT**: Terminal state when player leaves table.

**Hand Participation FSM:**
- **ACTIVE**: Player can make betting decisions (bet, call, fold). Sets `isTurn` flag when it's player's turn. Auto-transitions to ALL_IN when balance reaches zero.
- **ALL_IN**: Player has committed all chips, cannot make further actions. Passively waits for hand completion.
- **FOLDED**: Player has folded, out of the hand. Sets `hasFolded` flag and waits for hand completion.

### Key Design Notes

1. **Separation of Concerns**: Table presence persists across hands; hand participation is created/destroyed per hand.
2. **Flag-Based State**: FSMs set durable flags (`hasFolded`, `isAllIn`) that persist after FSM stops, ensuring showdown logic sees correct state.
3. **Event Forwarding**: Table presence FSM forwards unknown events to hand participation FSM when active.
4. **Synchronous Initialization**: Game's `statePreDeal` posts blinds first, then calls `HandleStartHand()` which detects all-in condition and initializes FSM in correct state (ACTIVE or ALL_IN).
5. **Event Naming Convention**: Events like `evDeductBlind` and `evAddChips` use action-oriented names to clarify they are game-commanded state updates (not player actions). This follows the command pattern where Game commands Player FSM to update state, maintaining encapsulation and thread safety.

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

### Typical Flow

1. **Table: WAITING_FOR_PLAYERS**
   - Players join, mark ready
   - `evUsersChanged` → checks `allPlayersReady()`
   - Transitions to `PLAYERS_READY`

2. **Table: PLAYERS_READY → GAME_ACTIVE**
   - Server calls `StartGame()`
   - Table sends `evStartGameReq` to Table FSM
   - Table creates Game instance and starts Game FSM
   - Game FSM receives `evStartHand` → `statePreDeal`

3. **Game: statePreDeal → stateDeal → stateBlinds → statePreFlop**
   - `statePreDeal`: Advances dealer, posts blinds, creates Hand Participation FSMs
   - `stateDeal`: Deals hole cards
   - `stateBlinds`: Sets first player to act
   - `statePreFlop`: Begins first betting round

4. **Game: Betting Rounds (PRE_FLOP → FLOP → TURN → RIVER)**
   - Game sets `currentPlayer`, sends `evStartTurn`
   - Player Hand Participation FSM handles betting actions
   - Player FSM sends action responses back to Game
   - Game validates, updates pot, advances to next player
   - On round complete: Game sends `evAdvance` → next street

5. **Game: SHOWDOWN**
   - Game evaluates hands, distributes pots
   - Game sends `GameEventShowdownComplete` to Table
   - Table stores result, publishes to clients
   - Auto-start timer triggers `startNewHand()`

6. **Subsequent Hands**
   - Table stays in `GAME_ACTIVE`
   - `startNewHand()` calls `Game.ResetForNewHandFromUsers()`
   - Game resets deck, clears `currentHand`
   - Game receives `evStartHand` → loops back to `statePreDeal`
   - Player Hand Participation FSMs destroyed and recreated each hand

7. **Game Ends**
   - Game sends `evGameEnded` to Table FSM
   - Table transitions `GAME_ACTIVE` → `WAITING_FOR_PLAYERS`
   - Player Table Presence FSMs remain active (players still seated)
   - Player Hand Participation FSMs destroyed

### Critical Synchronization Points

1. **Blind Posting Before Hand Start**: `statePreDeal` posts blinds BEFORE calling `HandleStartHand()`, ensuring players see correct balances and can detect all-in from blinds.

2. **PRE_FLOP Wait**: `StartGame()` and `startNewHand()` wait for Game FSM to reach `statePreFlop` before broadcasting `NEW_HAND_STARTED`, ensuring clients see complete state (blinds posted, current player set).

3. **Event Forwarding**: Player Table Presence FSM forwards unhandled events to Hand Participation FSM, allowing betting actions to reach the correct handler.

4. **Flag Persistence**: Player FSMs set flags (`hasFolded`, `isAllIn`) that outlive the FSM lifecycle, ensuring showdown sees correct state after FSM stops.
