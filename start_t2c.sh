#!/bin/bash
# start_t2c.sh — Démarre le client via LaunchAgent (survit à la fermeture du terminal)
set -euo pipefail
ROOT="$(cd "$(dirname "$0")" && pwd)"
PLIST="$HOME/Library/LaunchAgents/org.tutodecode.t2c-phantom.plist"
UID_NUM=$(id -u)

chmod +x "$ROOT/keep_t2c_alive.sh" 2>/dev/null || true

if [[ ! -f "$PLIST" ]]; then
  echo "❌ LaunchAgent manquant: $PLIST"
  exit 1
fi

launchctl bootout "gui/${UID_NUM}/org.tutodecode.t2c-phantom" 2>/dev/null || true
launchctl bootstrap "gui/${UID_NUM}" "$PLIST"
launchctl kickstart -k "gui/${UID_NUM}/org.tutodecode.t2c-phantom"
sleep 3

echo "✅ T2C Phantom (LaunchAgent KeepAlive)"
echo "   Dashboard : http://127.0.0.1:9090"
echo "   Proxy     : 127.0.0.1:8080"
echo "   Site HTTP : http://tutodecode.t2c"
echo "   Site HTTPS: https://tutodecode.t2c  (asso catalogue only)"
lsof -nP -iTCP:8080 -sTCP:LISTEN 2>/dev/null || echo "⚠️  proxy pas encore up — regarde mac_t2c.log"
open "http://127.0.0.1:9090" 2>/dev/null || true
