#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT_DIR"

S2_SLEEP_MS="${S2_SLEEP_MS:-0}"
S2_NEG_ACK="${S2_NEG_ACK:-false}"

cleanup() {
  for pid in "${P1:-}" "${P2:-}" "${P3:-}"; do
    if [[ -n "${pid}" ]] && kill -0 "$pid" >/dev/null 2>&1; then
      kill "$pid" >/dev/null 2>&1 || true
    fi
  done
}
trap cleanup EXIT

echo "[build] building server and client"
mkdir -p bin
go build -o bin/tokenserver_app ./server
go build -o bin/tokenclient_app ./client

echo "[start] starting 3 servers"
./bin/tokenserver_app -port 50051 -config token_config.yml > /tmp/tokenserver-50051.log 2>&1 &
P1=$!
./bin/tokenserver_app -port 50052 -config token_config.yml -sleep-ms "$S2_SLEEP_MS" -negative-ack="$S2_NEG_ACK" > /tmp/tokenserver-50052.log 2>&1 &
P2=$!
./bin/tokenserver_app -port 50053 -config token_config.yml > /tmp/tokenserver-50053.log 2>&1 &
P3=$!

sleep 2

echo "[demo] write on writer node s1"
./bin/tokenclient_app -write -id 1 -name abc -low 5 -mid 25 -high 100 -host 127.0.0.1 -port 50051

echo "[demo] read from replica s2"
./bin/tokenclient_app -read -id 1 -host 127.0.0.1 -port 50052

echo "[demo] unauthorized write (should fail authorization)"
./bin/tokenclient_app -write -id 1 -name denied -low 1 -mid 1 -high 1 -host 127.0.0.1 -port 50052 || true

echo "[logs] tailing short server logs"
for f in /tmp/tokenserver-50051.log /tmp/tokenserver-50052.log /tmp/tokenserver-50053.log; do
  echo "---- ${f} ----"
  tail -n 20 "$f" || true
  echo
 done

echo "[done] demo finished"
