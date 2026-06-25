#!/bin/bash
# start_t2c.sh — Lance le daemon T2C Phantom proprement (mode utilisateur Mac)
# Usage: ./start_t2c.sh

RELAY_PEER="12D3KooWBEK7H1KXCgpUbD4pFeGJVjVoFfUhnwPH1ywvsTbn6BBr"
# Raw TCP sur le port 443 pour le Mac (fallback auto-relay via IPFS si rejeté)
RELAY_TCP="/ip4/20.244.4.32/tcp/443/p2p/$RELAY_PEER"
RELAY_ADDR="$RELAY_TCP"
# Bootstrap via DHT libp2p public + relay Azure
BOOTSTRAP="/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN,/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa,$RELAY_TCP"
CARNET="./data/carnet.json"
PROXY_PORT=8080
UI_PORT=9090
LOG="mac_t2c.log"
KEY_FILE="./data/user_key.txt"

# Tuer les anciens processus T2C
pkill -f "t2c" 2>/dev/null
sleep 1

# Générer une clé persistante si elle n'existe pas encore
if [ ! -f "$KEY_FILE" ]; then
    echo "🔑 Génération d'une clé persistante..."
    ./t2c genkey 2>/dev/null | grep "Private Key:" | awk -F': ' '{print $2}' > "$KEY_FILE"
    echo "✅ Clé sauvegardée dans $KEY_FILE"
fi

USER_KEY=$(cat "$KEY_FILE" 2>/dev/null)

if [ -z "$USER_KEY" ]; then
    echo "⚠️  Impossible de lire la clé. Utilisation d'une identité éphémère."
    KEY_ARG=""
else
    KEY_ARG="--key $USER_KEY"
fi

echo "🚀 Lancement du daemon T2C (proxy HTTP :$PROXY_PORT)..."
nohup ./t2c start $KEY_ARG \
    --port 4002 \
    --proxy $PROXY_PORT \
    --bootstrap "$BOOTSTRAP" \
    --use-relay "$RELAY_ADDR" \
    --carnet "$CARNET" \
    > "$LOG" 2>&1 &

DAEMON_PID=$!
echo "   Daemon PID: $DAEMON_PID"

sleep 3

echo "🖥️  Lancement de l'interface web (port $UI_PORT)..."
nohup ./t2c dashboard-client > /dev/null 2>&1 &
UI_PID=$!
echo "   UI PID: $UI_PID"

sleep 2

echo ""
echo "=== STATUT ==="
cat "$LOG"
echo ""
echo "✅ Interface disponible sur : http://127.0.0.1:$UI_PORT"
open "http://127.0.0.1:$UI_PORT" 2>/dev/null
