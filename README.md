# Fail-Silent Replicated Token Manager with Atomic Semantics

This project restores your distributed systems final project from the submitted README.

## Features

- Replicated token manager across multiple servers.
- Authorization checks for token-level readers and writers.
- Atomic semantics via **read-impose-write-all (quorum)** style flow.
- Write broadcast + quorum acknowledgement.
- Read broadcast + max timestamp selection + write-back.
- Fail-silent simulation support:
  - server sleep/latency mode
  - negative-ack mode

## Project layout

- `server/tokenserver.go`: token server and replication protocol logic.
- `client/tokenclient.go`: command-line client for read/write operations.
- `token_management/token_pb.proto`: RPC contract from your project README.
- `token_management/token_pb.pb.go`: message structures used by the server/client.
- `token_management/token_pb_grpc.pb.go`: RPC stubs used by the server/client.
- `token_config.yml`: server + token authorization + replication config.
- `service-run.sh`: run full 3-server demo locally.

## Prerequisites

- Go 1.22+

## Install dependencies

```bash
go mod tidy
```

## Optional proto generation (original README flow)

Your original command was:

```bash
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative token_management/token_pb.proto
```

The restored project already includes working stubs, so generation is optional.

## Run servers manually

```bash
go build -o tokenserver ./server
./tokenserver -port 50051 -config token_config.yml
./tokenserver -port 50052 -config token_config.yml
./tokenserver -port 50053 -config token_config.yml
```

## Run client manually

Write:

```bash
go build -o tokenclient ./client
./tokenclient -write -id 1 -name abc -low 5 -mid 25 -high 100 -host 127.0.0.1 -port 50051
```

Read:

```bash
./tokenclient -read -id 1 -host 127.0.0.1 -port 50052
```

## Run all-in-one demo script

```bash
./service-run.sh
```

## Fail-silent scenarios

Simulate latency on server 50052:

```bash
S2_SLEEP_MS=4000 ./service-run.sh
```

Simulate negative ack on server 50052:

```bash
S2_NEG_ACK=true ./service-run.sh
```

## Notes

- Timestamps are generated as monotonic write timestamps (`wts`) and compared lexicographically.
- Token values include: `partial_value`, `name`, `low`, `mid`, `high`.
- Authorization and replication membership are controlled from `token_config.yml`.