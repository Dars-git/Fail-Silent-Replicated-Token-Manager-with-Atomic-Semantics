# PR Change Documentation

This document summarizes the changes on branch `refactor-config-indexing-and-tests`.

## Scope

The PR includes:

1. Scalability refactor in config lookup paths.
2. Server transport abstraction for better testability and flexibility.
3. Integration testing with 3 in-process gRPC servers.
4. Deterministic timestamp generation hooks for stable tests.
5. Configuration validation at load time.
6. Context propagation and cancellation handling in broadcast fan-out.
7. Quorum helper logic with table-driven tests.
8. Lightweight observability (counters + latency snapshots).
9. Benchmarks for config and server hot paths.

## Main Code Changes

### 1. Config indexing and validation

File: `internal/config/config.go`

- Added one-time in-memory indexes (`sync.Once`) for:
  - `serverByPort`
  - `serverByName`
  - `tokenByID`
  - `replicasByToken`
- Lookup operations now use map access instead of repeated linear scans.
- Added `validate()` in `Load()` to reject invalid config early:
  - duplicate server names
  - duplicate server ports
  - duplicate token IDs
  - invalid server ports
  - tokens with no readers/writers/replicas
  - unknown readers/writers/replicas

### 2. Server transport abstraction

File: `server/tokenserver.go`

- Added `peerTransport` interface:
  - `WriteBroadcast(...)`
  - `ReadBroadcast(...)`
- Added `grpcTransport` implementation for production use.
- `TokenManagerServer` now depends on `peerTransport` instead of hard-wiring direct gRPC dial logic in request flow.
- Added constructor `newTokenManagerServer(...)` and `ensureDefaults()` for safe defaults.

### 3. Deterministic WTS hooks

File: `server/tokenserver.go`

- Added injectable hooks:
  - `nowUnixNano func() int64`
  - `nextSequence func() uint64`
- `nextWTS()` now uses hooks, enabling deterministic tests.

### 4. Context propagation

File: `server/tokenserver.go`

- `startBroadcast` and `startReadBroadcast` now derive per-peer timeout contexts from parent request context.
- Cancellation/timeouts now propagate from top-level RPC contexts to broadcast calls.

### 5. Quorum helpers

File: `server/tokenserver.go`

- Added reusable quorum helpers:
  - `requiredQuorum(totalReplicas int) int`
  - `hasQuorum(ackCount, totalReplicas int) bool`
- `WriteToken` and `ReadToken` quorum checks now use helper logic.

### 6. Lightweight observability

Files:
- `server/metrics.go`
- `server/tokenserver.go`

- Added atomic counters for call volumes and error classes.
- Added latency accumulation and average latency reporting for main RPCs.
- Added `MetricsSnapshot()` API on `TokenManagerServer`.

## Test Additions

### Unit tests

- `internal/config/config_test.go`
  - default name assignment
  - lookup behavior
  - replica dedupe/filter/sort
  - validation error scenarios
- `internal/tokenstore/store_test.go`
  - `UpsertIfNewer` first/newer/stale/empty behavior
- `server/tokenserver_test.go`
  - write/read broadcast and request edge cases
- `server/tokenserver_advanced_test.go`
  - transport abstraction behavior
  - context cancellation propagation
  - deterministic WTS
  - quorum table tests
  - metrics assertions
- `client/tokenclient_test.go`
  - output formatting and nil response handling

### Integration tests

- `server/tokenserver_integration_test.go`
  - real 3-node in-process cluster
  - write/read success across replicas
  - negative-ack scenario still meeting quorum

### Benchmarks

- `internal/config/config_benchmark_test.go`
  - `BenchmarkServerByName`
  - `BenchmarkTokenByID`
  - `BenchmarkReplicaServers`
- `server/tokenserver_benchmark_test.go`
  - `BenchmarkWriteTokenSelfReplica`
  - `BenchmarkReadTokenSelfReplica`

## Validation Performed

Commands run successfully:

- `go test ./...`
- `go test -race ./...`
- `go test -run=^$ -bench . ./internal/config ./server`

## Why this is more scalable

- Config and token lookups moved from repeated scan (`O(n)`) to indexed map lookup (`O(1)` expected).
- Replica resolution is precomputed and reused.
- Server fan-out logic is now abstraction-based, enabling non-network test paths and easier future transport optimization.
- Added observability to measure behavior under load and guide further tuning.
