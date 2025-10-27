# Poker Bot for Bison Relay

A poker bot implementation for the Bison Relay network that allows users to play poker games using their balance.

## Features

- Player balance management
- Table creation and joining
- Texas Hold'em poker game implementation
- gRPC API for game operations
- SQLite database for persistence

## Getting Started

### Prerequisites

- Go 1.24 or later
- Protocol Buffers compiler (protoc)
- SQLite3

### Installation

1. Clone the repository:
```bash
git clone https://github.com/vctt94/pokerbisonrelay.git
cd pokerbisonrelay
```

2. Install dependencies:
```bash
go mod tidy
```

3. Generate gRPC code:
```bash
./generate.sh
```

### Running the Server

Start the poker server:
```bash
go run cmd/server/main.go
```

The server will start listening on port 50051 by default. You can change the port and database path using flags:
```bash
go run cmd/server/main.go -port :50052 -db custom.db
```

### Running the Client

Test the service using the client:
```bash
go run cmd/client/main.go
```

You can specify the server address and player ID:
```bash
go run cmd/client/main.go -server localhost:50052 -player my-player
```

### Run Tests

```bash
go clean -testcache && go test -tags=lockcheck -v -race ./... -count=10 -timeout 300s > test_output.log 2>&1
```


## API Documentation

The poker service provides the following gRPC endpoints:

- `GetBalance`: Get a player's current balance
- `UpdateBalance`: Update a player's balance
- `CreateTable`: Create a new poker table
- `JoinTable`: Join an existing table
- `LeaveTable`: Leave a table
- `MakeBet`: Place a bet in the current round
- `GetTableState`: Get the current state of a table

## Project Structure

- `poker/`: Core poker game logic
  - `poker.go`: Main poker game types and functions
  - `deck.go`: Deck and card management
  - `game.go`: Game state machine and rules
  - `db.go`: Database operations
- `pokerrpc/`: gRPC service definition and implementation
  - `poker.proto`: Protocol Buffers definition
  - `server.go`: gRPC server implementation
- `cmd/`: Command-line applications
  - `server/`: Server implementation
  - `client/`: Client implementation

## License

This project is licensed under the MIT License - see the LICENSE file for details. 