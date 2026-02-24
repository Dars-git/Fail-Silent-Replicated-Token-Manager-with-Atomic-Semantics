#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT_DIR"

HOST="${HOST:-127.0.0.1}"
PORT="${1:-50051}"
CONFIG="${CONFIG:-token_config.yml}"

exec go run ./server -host "$HOST" -port "$PORT" -config "$CONFIG"
