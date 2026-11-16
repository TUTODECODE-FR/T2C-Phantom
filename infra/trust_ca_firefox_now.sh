#!/bin/bash
# Installe tutodecode-ca.crt dans Firefox (ferme Firefox d'abord).
set -euo pipefail
CA="${HOME}/Library/Caches/t2c-phantom/asso-ca/asso-ca.crt"
NICK="tutodecode CA"
CERTUTIL="/opt/homebrew/opt/nss/bin/certutil"
[[ -f "$CA" ]] || { echo "CA manquante: $CA — lance T2C Phantom d'abord"; exit 1; }
[[ -x "$CERTUTIL" ]] || CERTUTIL="$(command -v certutil || true)"
[[ -n "${CERTUTIL}" ]] || { echo "brew install nss"; exit 1; }

cp -f "$CA" "${HOME}/Desktop/tutodecode-ca.crt" 2>/dev/null || true

echo "→ Ferme Firefox…"
osascript -e 'quit app "Firefox"' >/dev/null 2>&1 || true
sleep 2
killall -9 firefox 2>/dev/null || true
sleep 1

PROF=""
INI="${HOME}/Library/Application Support/Firefox/profiles.ini"
if [[ -f "$INI" ]]; then
  PATH0=$(awk -F= '/^Default=1$/{d=1} d&&/^Path=/{print $2; exit}' "$INI" || true)
  [[ -z "${PATH0:-}" ]] && PATH0=$(awk -F= '/^Path=/{print $2; exit}' "$INI" || true)
  if [[ -n "${PATH0:-}" ]]; then
    if [[ "$PATH0" = /* ]]; then PROF="$PATH0"; else PROF="${HOME}/Library/Application Support/Firefox/${PATH0}"; fi
  fi
fi
if [[ -z "$PROF" || ! -d "$PROF" ]]; then
  PROF="${HOME}/Library/Application Support/Firefox/Profiles/hmo2j378.default-release"
fi
echo "→ Profil: $PROF"
rm -f "$PROF/lock" "$PROF/.parentlock" 2>/dev/null || true
"$CERTUTIL" -D -n "T2C Asso Local CA" -d "sql:$PROF" 2>/dev/null || true
"$CERTUTIL" -D -n "$NICK" -d "sql:$PROF" 2>/dev/null || true
"$CERTUTIL" -A -n "$NICK" -t "CT,C,C" -i "$CA" -d "sql:$PROF"
"$CERTUTIL" -L -d "sql:$PROF" | grep -i tutodecode || true
echo "✅ CA tutodecode trustée. Relance Firefox → https://tutodecode.t2c (proxy 127.0.0.1:8080)"
open -a Firefox
