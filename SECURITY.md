# Politique de Sécurité (T2C-Phantom)

## Périmètre

Cette politique couvre :
- Le code source du client d'administration hébergé dans ce dépôt.
- Les binaires générés pour les administrateurs locaux.

Ce projet étant un client d'administration hors-ligne, aucune API distante externe à l'infrastructure d'administration n'est impliquée.

## Nomenclature Logicielle (SBOM)

La liste complète des dépendances logicielles est disponible dans :
- `go.mod` — dépendances déclarées.
- `go.sum` — versions exactes résolues et hashes d'intégrité cryptographique.

## Divulgation Responsable

N'ouvrez pas d'issues GitHub publiques pour signaler des vulnérabilités critiques d'escalade de privilèges ou d'exécution de code à distance.
Utilisez l'un de ces canaux privés :
- Avis de sécurité privé GitHub.
- Email : `contact@tutodecode.org`.

Lors de votre signalement, veuillez inclure :
- La version/le tag affecté(e).
- Les étapes pour reproduire.
- Un résumé de l'impact.

## Objectifs de Réponse

- Délai d'accusé de réception : dans les 72 heures.
- Délai de tri : dans les 7 jours ouvrables.
- Après un correctif, une nouvelle release est générée et signée.
