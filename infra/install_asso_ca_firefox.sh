#!/bin/bash
# Installe la CA asso T2C dans Firefox (NSS) — une fois, plug-and-play ensuite.
set -euo pipefail
OFFICIAL="${HOME}/Library/Caches/t2c-phantom/asso-ca/official-from-server.crt"
LOCAL="${HOME}/Library/Caches/t2c-phantom/asso-ca/asso-ca.crt"
# Priorité : entité CA donnée par le serv, sinon CA locale MITM
CA="$OFFICIAL"
NICK="T2C Asso Official CA (serv)"
if [[ ! -f "$CA" ]]; then
  CA="$LOCAL"
  NICK="T2C Asso Local CA"
fi
if [[ ! -f "$CA" ]]; then
  echo "CA absente — lance d'abord T2C Phantom (récupère la CA du serv)."
  exit 1
fi
echo "Trust: $CA ($NICK)"

CERTUTIL=""
if command -v certutil >/dev/null 2>&1; then
  CERTUTIL=$(command -v certutil)
elif [[ -x /opt/homebrew/opt/nss/bin/certutil ]]; then
  CERTUTIL=/opt/homebrew/opt/nss/bin/certutil
elif [[ -x /usr/local/opt/nss/bin/certutil ]]; then
  CERTUTIL=/usr/local/opt/nss/bin/certutil
fi

if [[ -z "$CERTUTIL" ]]; then
  echo "Installe nss: brew install nss"
  exit 1
fi

shopt -s nullglob
PROFILES=()
for root in \
  "$HOME/Library/Application Support/Firefox/Profiles" \
  "$HOME/Library/Application Support/zen/Profiles" \
  "$HOME/Library/Application Support/LibreWolf/Profiles" \
  "$HOME/.mozilla/firefox" \
  "$HOME/snap/firefox/common/.mozilla/firefox" \
  "$HOME/.zen/profiles" \
  "$HOME/.librewolf"
do
  [[ -d "$root" ]] || continue
  for p in "$root"/*; do
    [[ -d "$p" ]] && PROFILES+=( "$p" )
  done
done
if [[ ${#PROFILES[@]} -eq 0 ]]; then
  echo "Aucun profil Firefox trouvé — ouvre Firefox une fois puis relance ce script"
  exit 1
fi

for p in "${PROFILES[@]}"; do
  [[ -d "$p" ]] || continue
  echo "→ profil $p"
  "$CERTUTIL" -A -n "$NICK" -t "C,," -i "$CA" -d "sql:$p" 2>/dev/null \
    || "$CERTUTIL" -A -n "$NICK" -t "C,," -i "$CA" -d "dbm:$p" 2>/dev/null \
    || echo "   (skip)"
done
# Aussi la CA locale MITM si on a trusté l'entité serv (padlock .t2c)
if [[ -f "$OFFICIAL" && -f "$LOCAL" && "$CA" == "$OFFICIAL" ]]; then
  for p in "${PROFILES[@]}"; do
    [[ -d "$p" ]] || continue
    "$CERTUTIL" -A -n "T2C Asso Local CA" -t "C,," -i "$LOCAL" -d "sql:$p" 2>/dev/null || true
  done
fi

echo "✅ Entité CA trustée. Redémarre Firefox, puis ouvre https://tutodecode.t2c"
