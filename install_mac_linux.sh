#!/bin/bash

echo "========================================="
echo " Installation de T2C Phantom... "
echo "========================================="

OS="$(uname -s)"
ARCH="$(uname -m)"

if [ "$OS" = "Darwin" ]; then
    if [ "$ARCH" = "arm64" ]; then
        FILE="t2c-darwin-arm64"
    else
        FILE="t2c-darwin-amd64"
    fi
elif [ "$OS" = "Linux" ]; then
    FILE="t2c-linux-amd64"
else
    echo "OS non supporté par ce script."
    exit 1
fi

URL="https://github.com/TUTODECODE-FR/T2C-Phantom/releases/latest/download/$FILE"

echo "Téléchargement de $FILE depuis GitHub..."
curl -L -o t2c "$URL"

if [ -f "t2c" ]; then
    chmod +x t2c
    echo "Téléchargement réussi ! Lancement en arrière-plan..."
    ./t2c dashboard &
    echo "L'interface web s'ouvre sur http://127.0.0.1:9090"
    sleep 2
    open http://127.0.0.1:9090 2>/dev/null || xdg-open http://127.0.0.1:9090 2>/dev/null
else
    echo "Erreur lors du téléchargement."
    exit 1
fi
