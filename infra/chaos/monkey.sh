#!/bin/bash
# 🐒 T2C-Phantom Chaos Monkey
# Ce script sème le chaos sur l'interface réseau locale pour prouver la résilience du Failover et de la UI.

INTERFACE="lo" # On limite au loopback pour ne pas casser la vraie connexion (Wi-Fi/Eth) de la soutenance.
LATENCY="800ms"
JITTER="200ms"
LOSS="10%"

echo "======================================================"
echo "🐒 DÉMARRAGE DU CHAOS MONKEY (T2C-Phantom)"
echo "======================================================"
echo "⚠️ L'infrastructure va subir une instabilité majeure."
echo "Interface ciblée : $INTERFACE"
echo "Latence injectée : $LATENCY ± $JITTER"
echo "Perte de paquets : $LOSS"
echo "Appuyez sur CTRL+C pour arrêter et restaurer le réseau."
echo "======================================================"

# Fonction de nettoyage (restauration) en cas d'arrêt
cleanup() {
    echo -e "\n🧹 Arrêt du Chaos Monkey. Restauration de l'interface $INTERFACE..."
    sudo tc qdisc del dev $INTERFACE root 2>/dev/null
    echo "✅ Réseau restauré."
    exit 0
}

trap cleanup SIGINT SIGTERM

# 1. Injection de la latence et perte de paquets (Détérioration P2P)
sudo tc qdisc add dev $INTERFACE root netem delay $LATENCY $JITTER loss $LOSS 2>/dev/null || sudo tc qdisc change dev $INTERFACE root netem delay $LATENCY $JITTER loss $LOSS

echo "💥 Chaos réseau injecté via 'tc'."

# 2. Boucle de Chaos Processeur (Tuer aléatoirement des instances p2p)
while true; do
    sleep $((RANDOM % 10 + 5)) # Pause entre 5 et 15 secondes
    
    # Choisir aléatoirement une action destructrice
    ACTION=$((RANDOM % 3))
    
    case $ACTION in
        0)
            echo "🦍 [CHAOS] Gel d'un processus P2P (SIGSTOP)..."
            # Gèle un process 't2c' aléatoire s'il y en a (simulation d'un crash RAM)
            PIDS=$(pgrep t2c)
            if [ -n "$PIDS" ]; then
                TARGET=$(echo "$PIDS" | shuf -n 1)
                kill -STOP $TARGET 2>/dev/null
                echo "   -> Processus $TARGET gelé."
                
                # Le réveiller plus tard
                (sleep 5; kill -CONT $TARGET 2>/dev/null; echo "   -> Processus $TARGET réveillé.") &
            fi
            ;;
        1)
            echo "🌪️ [CHAOS] Pic de latence extrême (2000ms)..."
            sudo tc qdisc change dev $INTERFACE root netem delay 2000ms loss 50%
            sleep 3
            sudo tc qdisc change dev $INTERFACE root netem delay $LATENCY $JITTER loss $LOSS
            echo "   -> Latence revenue à la normale du chaos."
            ;;
        2)
            echo "👀 [CHAOS] Le singe observe (Rien à signaler ce tour-ci)."
            ;;
    esac
done
