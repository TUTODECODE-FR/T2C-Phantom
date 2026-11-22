#!/usr/bin/env bash
# Génère les notes de version en comparant le tag précédent et le nouveau.
# Usage : release-notes.sh <tag> [dépôt owner/nom]   (sortie Markdown sur stdout)
# Classement par type de commit (Conventional Commits) ; aucun emoji.
set -euo pipefail

TAG="${1:?tag requis}"
REPO="${2:-TUTODECODE-FR/T2C-Phantom}"
URL="https://github.com/$REPO"

# tag précédent : dernier tag de version (vX.Y.Z) antérieur à celui-ci
if git rev-parse -q --verify "$TAG^{commit}" >/dev/null; then HEAD_REF="$TAG"; else HEAD_REF=HEAD; fi
PREV=$(git tag --list 'v[0-9]*' --sort=-v:refname --merged "$HEAD_REF" \
  | grep -vx "$TAG" | grep -vE -- '-essai|-test' | head -1 || true)
RANGE="${PREV:+$PREV..}$HEAD_REF"

# sujet des commits (hors fusions), sans emoji ni espaces superflus
commits=$(git log --no-merges --format='%h%x09%s' "$RANGE" \
  | perl -CSD -pe 's/[\x{1F000}-\x{1FAFF}\x{2600}-\x{27BF}\x{2B00}-\x{2BFF}\x{FE0F}\x{200D}]//g; s/[ ]{2,}/ /g')

ncommits=$(printf '%s\n' "$commits" | grep -c . || true)
stat=$(git diff --shortstat "${PREV:-$(git rev-list --max-parents=0 "$HEAD_REF" | tail -1)}" "$HEAD_REF" 2>/dev/null \
  | sed -E 's/ files? changed/ fichiers modifiés/; s/ insertions?\(\+\)/ lignes ajoutées/; s/ deletions?\(-\)/ lignes supprimées/' | xargs || true)

# section <titre> <regex de type>
section() {
  local title="$1" re="$2" lines
  lines=$(printf '%s\n' "$commits" | awk -F'\t' -v re="$re" '
    $2 ~ re {
      msg=$2; sha=$1; scope="";
      if (match(msg, /^[a-z]+\(([^)]+)\)!?: /)) {
        scope=substr(msg, RSTART, RLENGTH); sub(/^[a-z]+\(/, "", scope); sub(/\)!?: $/, "", scope);
        sub(/^[a-z]+\([^)]+\)!?: /, "", msg)
      } else { sub(/^[a-z]+!?: /, "", msg) }
      msg=toupper(substr(msg,1,1)) substr(msg,2)
      printf "- %s%s (%s)\n", (scope!="" ? "**" scope "** : " : ""), msg, sha
    }')
  if [ -n "$lines" ]; then printf '### %s\n\n%s\n\n' "$title" "$lines"; fi
}

echo "## T2C Phantom $TAG"
echo
if [ -n "$PREV" ]; then
  echo "Changements depuis $PREV : $ncommits commit(s)${stat:+, $stat}."
  echo "Comparaison complète : $URL/compare/$PREV...$TAG"
else
  echo "Première version publiée : $ncommits commit(s)."
fi
echo

breaking=$(printf '%s\n' "$commits" | awk -F'\t' '$2 ~ /^[a-z]+(\([^)]+\))?!: / {print "- " $2 " (" $1 ")"}')
[ -n "$breaking" ] && printf '### Changements incompatibles\n\n%s\n\n' "$breaking"

section "Nouveautés"                 '^feat([(]|!|:)'
section "Corrections"                '^fix([(]|!|:)'
section "Sécurité"                   '^sec(urity)?([(]|!|:)'
section "Performances"               '^perf([(]|!|:)'
section "Documentation"              '^docs([(]|!|:)'
section "Maintenance"                '^(ci|build|chore|refactor|test|style|deps)([(]|!|:)'
# « Autres » : commits sans préfixe conventionnel
others=$(printf '%s\n' "$commits" | awk -F'\t' '$2 !~ /^[a-z]+(\([^)]+\))?!?: / && $2 != "" {print "- " $2 " (" $1 ")"}')
[ -n "$others" ] && printf '### Autres changements\n\n%s\n\n' "$others"

cat <<EOF
### Téléchargements

| Système | Fichier |
|---|---|
| macOS (Intel et Apple Silicon) | \`T2C-Phantom-macOS-universal.zip\` |
| Windows 10 / 11 | \`T2C-Phantom-Windows-amd64.zip\` |
| Linux (x86_64) | \`T2C-Phantom-Linux-amd64.tar.gz\` |
| Ligne de commande | \`t2c-cli-$TAG-<système>-<architecture>\` |

Installation et premiers pas : $URL/blob/$TAG/docs/UTILISATION.md

### Vérification

\`\`\`bash
sha256sum -c SHA256SUMS.txt --ignore-missing
\`\`\`

Signature : macOS signé Developer ID et notarisé par Apple ; Windows signé par SignPath Foundation dès que la signature est active (sinon SmartScreen affiche un avertissement : Exécuter quand même). [Code signing policy]($URL/blob/$TAG/docs/CODE-SIGNING-POLICY.md) — Free code signing provided by SignPath.io, certificate by SignPath Foundation.
EOF
