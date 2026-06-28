#!/bin/bash
# Implémente le Zero-Trust L3 : seule l'IP du LXC Proxy peut accéder au port de l'application

iptables -F
iptables -P INPUT DROP
iptables -P FORWARD DROP
iptables -P OUTPUT ACCEPT
iptables -A INPUT -i lo -j ACCEPT
iptables -A INPUT -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT

# Autoriser uniquement le trafic entrant depuis l'IP du LXC Proxy
LXC_PROXY_IP="10.0.0.2"
TRACKER_PORT="8080"
iptables -A INPUT -p tcp -s $LXC_PROXY_IP --dport $TRACKER_PORT -j ACCEPT

echo "Règles iptables appliquées."
