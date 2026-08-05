#!/bin/bash

# Script d'installation unifié de service T2C-Phantom (Unix)
# Doit être exécuté avec les droits d'administration (sudo)

if [ "$EUID" -ne 0 ]; then
  echo "❌ Veuillez exécuter ce script en tant que root (sudo ./install.sh)."
  exit 1
fi

OS="$(uname -s)"

echo "========================================="
echo " Installation du Service T2C-Phantom      "
echo "========================================="

# 1. Copie du binaire
if [ -f "t2c-linux-amd64" ] && [ "$OS" = "Linux" ]; then
  cp t2c-linux-amd64 /usr/local/bin/t2c-phantom
  chmod +x /usr/local/bin/t2c-phantom
elif [ -f "t2c-mac-v2" ] && [ "$OS" = "Darwin" ]; then
  cp t2c-mac-v2 /usr/local/bin/t2c-phantom
  chmod +x /usr/local/bin/t2c-phantom
elif [ -f "t2c" ]; then
  cp t2c /usr/local/bin/t2c-phantom
  chmod +x /usr/local/bin/t2c-phantom
else
  echo "⚠️ Binaire 't2c-phantom' non trouvé dans le répertoire courant. Installation sans binaire."
fi

# 2. Configuration selon l'OS
if [ "$OS" = "Linux" ]; then
  echo "[+] Configuration du service Systemd sous Linux..."
  
  cat <<EOF > /etc/systemd/system/t2c-phantom.service
[Unit]
Description=T2C-Phantom Secure P2P Node Service
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/t2c-phantom start --metrics-port 2112
Restart=always
RestartSec=5

# 🛡️ SECURITE : Exécution sous l'utilisateur nobody
User=nobody
Group=nogroup
NoNewPrivileges=yes
AmbientCapabilities=CAP_NET_BIND_SERVICE
CapabilityBoundingSet=CAP_NET_BIND_SERVICE

# 🛡️ SECURITE : Système de fichiers verrouillé en lecture seule
ProtectSystem=strict
ProtectHome=yes
PrivateTmp=yes
ReadWritePaths=/tmp/t2c_cache
TemporaryFileSystem=/tmp:rw,noexec,nosuid,nodev
TemporaryFileSystem=/tmp/t2c_cache:rw,noexec,nosuid,nodev

# 🛡️ SECURITE : Filtres Seccomp (Anti shell / Anti execve)
SystemCallFilter=@system-service
SystemCallFilter=~execve
SystemCallErrorNumber=EPERM

PrivateDevices=yes
ProtectKernelLogs=yes
ProtectKernelModules=yes
ProtectKernelTunables=yes
RestrictNamespaces=yes

[Install]
WantedBy=multi-user.target
EOF

  systemctl daemon-reload
  systemctl enable t2c-phantom
  systemctl start t2c-phantom
  echo "✅ Service Systemd 't2c-phantom' installé et démarré !"

elif [ "$OS" = "Darwin" ]; then
  echo "[+] Configuration de l'agent Launchd sous macOS..."

  cat <<EOF > /Library/LaunchDaemons/org.tutodecode.t2c-phantom.plist
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>org.tutodecode.t2c-phantom</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/t2c-phantom</string>
        <string>start</string>
        <string>--metrics-port</string>
        <string>2112</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>UserName</key>
    <string>nobody</string>
    <key>StandardOutPath</key>
    <string>/var/log/t2c-phantom.out.log</string>
    <key>StandardErrorPath</key>
    <string>/var/log/t2c-phantom.err.log</string>
</dict>
</plist>
EOF

  launchctl load -w /Library/LaunchDaemons/org.tutodecode.t2c-phantom.plist
  echo "✅ Agent Launchd 'org.tutodecode.t2c-phantom' installé et démarré !"
else
  echo "❌ Système d'exploitation non supporté pour l'installation automatique."
  exit 1
fi
