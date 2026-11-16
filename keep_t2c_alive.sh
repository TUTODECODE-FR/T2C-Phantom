#!/bin/bash
# Garde le dashboard + seeder T2C allumés (LaunchAgent / KeepAlive).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

BIN="$ROOT/t2c_dev"
[[ -x "$BIN" ]] || BIN="$ROOT/t2c"
[[ -x "$BIN" ]] || { echo "no binary"; exit 1; }

pkill -f '[t]2c_dev dashboard-client' 2>/dev/null || true
pkill -f '[t]2c dashboard-client' 2>/dev/null || true
sleep 1

"$BIN" dashboard-client >> "$ROOT/mac_t2c.log" 2>&1 &
DASH_PID=$!
echo "$DASH_PID" > "${TMPDIR:-/tmp}/t2c_dash.pid"
sleep 2

# démarre le seeder/proxy si arrêté
curl -s -X POST -H 'Host: 127.0.0.1:9090' -H 'Content-Type: application/json' \
  http://127.0.0.1:9090/api/client/start \
  -d '{"disable_seeding":false}' >/dev/null 2>&1 || true

# reste vivant tant que le dashboard tourne (KeepAlive relancera le script s'il meurt)
wait "$DASH_PID"
