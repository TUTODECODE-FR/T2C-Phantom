#!/bin/bash

echo "========================================="
echo " Configuration T2C Phantom (ADMIN) "
echo "========================================="

if [ ! -f "t2c-linux-admin" ]; then
    echo "[!] Le binaire t2c-linux-admin est introuvable."
    echo "Veuillez transférer ce fichier sur le serveur via scp avant d'exécuter ce script."
    exit 1
fi

chmod +x t2c-linux-admin

echo "1) Configurer cette machine comme Relais Public (Ex: VM Cloud)"
echo "2) Exposer un port local via le réseau T2C (Ex: Proxmox)"
read -p "Votre choix (1 ou 2) : " choix

if [ ! -f "admin_key.txt" ]; then
    echo "Génération d'une nouvelle identité..."
    ./t2c-linux-admin genkey | grep "Private Key:" | awk '{print $3}' > admin_key.txt
fi

KEY=$(cat admin_key.txt)

if [ "$choix" = "1" ]; then
    echo "Lancement du Relais..."
    # Si la commande --is-relay n'est pas reconnue, le dashboard-server est l'équivalent
    ./t2c-linux-admin start --key "$KEY" --is-relay || ./t2c-linux-admin dashboard-server
elif [ "$choix" = "2" ]; then
    read -p "Port local à exposer (ex: 8006 pour Proxmox) : " port
    read -p "Adresse du relais (/ip4/.../p2p/...) : " relay
    echo "Exposition du port $port via $relay..."
    ./t2c-linux-admin start --key "$KEY" --expose "$port" --use-relay "$relay"
else
    echo "Choix invalide."
fi
