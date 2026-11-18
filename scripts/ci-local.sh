#!/bin/sh
# CI locale : les contrôles de GitHub (ci.yml, release.yml), lancés sur ce PC
# avant d'envoyer. N'affiche que les erreurs et un bilan ; journaux complets
# dans .ci-local/. Lancé automatiquement avant « git push github » (hook).
#
#   sh scripts/ci-local.sh           contrôles complets
#   sh scripts/ci-local.sh --rapide  sans -race (2 à 3 fois plus rapide)
set -u
ROOT=$(cd "$(dirname "$0")/.." && pwd)
OUT="$ROOT/.ci-local"
FAIL=0
RACE=-race
[ "${1:-}" = "--rapide" ] && RACE=
mkdir -p "$OUT"

step() { # step <nom> <journal> <commande…>
	name=$1; log="$OUT/$2.log"; shift 2
	printf '%-50s' "$name"
	t0=$(date +%s)
	if "$@" >"$log" 2>&1; then
		echo "OK ($(($(date +%s) - t0))s)"
	else
		echo "ÉCHEC"
		grep -E -- "--- FAIL|\.go:[0-9]+(:[0-9]+)?:|^FAIL|panic:|non formaté|DATA RACE|interdit" "$log" | head -n 20 | sed 's/^/    /'
		echo "    → journal : $log"
		FAIL=$((FAIL + 1))
	fi
}

gofmt_check() { l=$(gofmt -l $(git ls-files '*.go')); [ -z "$l" ] || { echo "$l" | sed 's/$/ : non formaté (gofmt -w)/'; return 1; }; }

# aucun secret ni code privé dans ce dépôt public
secrets_check() {
	bad=$(git ls-files | grep -E '(^|/)(signing|service|relay|site)\.key$|\.token$|\.deleg\.json$|(^|/)secrets/|t2c-editeur' || true)
	[ -z "$bad" ] || { echo "$bad" | sed 's/$/ : fichier interdit dans le dépôt public/'; return 1; }
	! git grep -n -I -E 'BEGIN (RSA|OPENSSH|EC) PRIVATE KEY|glpat-[0-9A-Za-z_-]{20}|ghp_[0-9A-Za-z]{36}' -- . ':!scripts/ci-local.sh' | sed 's/$/ : secret interdit/' | grep .
}

cross() { # compile les outils en ligne de commande pour chaque système publié
	for t in darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64; do
		GOOS=${t%/*} GOARCH=${t#*/} go build -o /dev/null ./cmd/... || { echo "compilation $t interdit"; return 1; }
	done
}

cd "$ROOT" || exit 1
echo "== T2C-Phantom"
step "secrets et fichiers privés" secrets secrets_check
step "formatage (gofmt)" gofmt gofmt_check
step "dépendances (go mod verify)" modverify go mod verify
step "compilation" build go build ./...
step "analyse (go vet)" vet go vet ./...
step "compilation 5 systèmes (outils CLI)" cross cross
step "tests ${RACE:-sans -race}" test go test $RACE -count=1 ./...
if [ -d desktop ]; then
	step "application de bureau (go vet)" desktop sh -c "cd desktop && go vet ./..."
fi

# dépôts privés voisins : ils utilisent ce code (pkg/trust…), on vérifie qu'ils suivent
for d in T2C-Phantom-serv T2C-Phantom-relay; do
	[ -d "$ROOT/../$d" ] || continue
	echo "== $d (privé, non publié)"
	step "compilation + analyse" "$d-vet" sh -c "cd '$ROOT/../$d' && go build ./... && go vet ./..."
	step "tests ${RACE:-sans -race}" "$d-test" sh -c "cd '$ROOT/../$d' && go test $RACE -count=1 ./..."
done

echo
if [ "$FAIL" -eq 0 ]; then echo "Bilan : tout est vert"; else echo "Bilan : $FAIL échec(s)"; fi
exit "$FAIL"
