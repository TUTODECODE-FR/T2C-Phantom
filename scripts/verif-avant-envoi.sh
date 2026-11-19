#!/bin/sh
# Vérification complète avant envoi (GitHub ou GitLab) : CI locale + pentest
# local. Tout tourne sur ce PC : aucun service en ligne, aucun token Claude.
# En cas d'erreur : notification, rapport court dans .ci-local/rapport.md et
# bouton « Ouvrir dans Claude » (Claude ne lit alors que ce rapport).
set -u
ROOT=$(cd "$(dirname "$0")/.." && pwd)
OUT="$ROOT/.ci-local"
PENTEST="$ROOT/../T2C-Phantom-serv/infra/pentest/pentest-local.sh"
mkdir -p "$OUT"
: >"$OUT/sortie.txt"
t0=$(date +%s)

sh "$ROOT/scripts/ci-local.sh" "$@" 2>&1 | tee -a "$OUT/sortie.txt"
[ -f "$PENTEST" ] && { echo; echo "== Pentest local"; FUZZTIME=${FUZZTIME:-5s} sh "$PENTEST" 2>&1 | tee -a "$OUT/sortie.txt"; }
dur=$(($(date +%s) - t0))

notify() { osascript -e "display notification \"$2\" with title \"T2C Phantom\" subtitle \"$1\"" 2>/dev/null || true; }

if ! grep -q "ÉCHEC" "$OUT/sortie.txt"; then
	rm -f "$OUT/rapport.md"
	notify "Vérification réussie" "Tout est vert en ${dur}s : envoi autorisé."
	exit 0
fi

# rapport court : uniquement les étapes en échec et leurs lignes d'erreur
{
	echo "# Erreurs de la vérification avant envoi ($(date '+%Y-%m-%d %H:%M'))"
	echo
	echo "Dépôt : $ROOT (branche $(git -C "$ROOT" rev-parse --abbrev-ref HEAD 2>/dev/null))"
	echo
	echo '```'
	awk '/ÉCHEC/ {print; keep=1; next} keep && /^    / {print; next} {keep=0}' "$OUT/sortie.txt"
	echo '```'
	echo
	echo "Relancer : sh scripts/verif-avant-envoi.sh"
} >"$OUT/rapport.md"
n=$(grep -c "ÉCHEC" "$OUT/sortie.txt")
notify "$n erreur(s) — envoi bloqué" "Rapport : .ci-local/rapport.md"

PROMPT="Corrige les erreurs décrites dans .ci-local/rapport.md : lis ce rapport puis seulement les fichiers qu'il cite, corrige, et relance sh scripts/verif-avant-envoi.sh pour vérifier."
rep=$(osascript -e "button returned of (display dialog \"Vérification avant envoi : $n erreur(s). L'envoi est bloqué.\" buttons {\"Ignorer\", \"Voir le rapport\", \"Ouvrir dans Claude\"} default button 3 with title \"T2C Phantom\" giving up after 600)" 2>/dev/null)
case "$rep" in
"Ouvrir dans Claude")
	osascript -e "tell application \"Terminal\" to do script \"cd '$ROOT' && claude '$PROMPT'\"" -e 'tell application "Terminal" to activate' >/dev/null 2>&1 ;;
"Voir le rapport")
	open -e "$OUT/rapport.md" ;;
esac
exit 1
