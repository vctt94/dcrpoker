## Getting Started

### Prerequisites
- Go 1.23+ (module uses `go 1.23.4`)
- Protocol Buffers compiler (`protoc`) if regenerating gRPC stubs
- SQLite3 CLI (for inspecting the dev DB)

### Setup
```bash
git clone https://github.com/vctt94/pokerbisonrelay.git
cd pokerbisonrelay
go mod download
```

### Regenerate gRPC code (optional)
Only needed when changing `pkg/rpc/*.proto`. Requires Go + Dart plugins:
```bash
cd pkg/rpc
./regen-clientrpc.sh
```

### Run the server
Starts gRPC poker server with SQLite storage and verbose logs:
```bash
go run ./cmd/pokersrv \
  -host=127.0.0.1 -port=50051 \
  -db=/tmp/poker.sqlite \
  -debuglevel=debug
```
- Metrics: `http://127.0.0.1:50051/metrics` (Prometheus text format).
- Use `--portfile` to write the chosen port to a file for scripts.

### Run the TUI client
```bash
go run ./cmd/client \
  -datadir="$HOME/.pokerclient" \
  -grpchost=127.0.0.1 -grpcport=50051
```

### Tests
- Fast path (unit + pkg): `go test ./pkg/...`
- Full suite (includes e2e with SQLite + gRPC): `go test ./...`
  - Ensure port 50051 is free before running the e2e tests.
  - Set `POKER_SEED` to make shuffles deterministic during debugging.

`go clean -testcache && go test -tags=lockcheck -v -race ./... -count=10 -timeout 300s > test_output.log 2>&1`
