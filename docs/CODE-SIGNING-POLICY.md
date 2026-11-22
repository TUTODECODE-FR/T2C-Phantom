# Code signing policy

**Politique de signature du code de T2C Phantom**

Free code signing provided by [SignPath.io](https://signpath.io), certificate by [SignPath Foundation](https://signpath.org).

## Ce qui est signé

| Système | Signature | Par qui |
|---|---|---|
| **Windows** | Authenticode | certificat de **SignPath Foundation**, via [SignPath.io](https://signpath.io) |
| **macOS** | Developer ID + notarisation Apple | certificat **Developer ID Application: TUTODECODE (TS97M57BJV)** |
| **Linux** | sommes SHA-256 publiées avec chaque version (`SHA256SUMS.txt`) | — |

Seuls les programmes **construits par l'intégration continue GitHub Actions à partir du code source de ce dépôt** sont signés : l'application T2C Phantom et ses outils en ligne de commande. Aucun programme compilé sur un poste personnel, aucun binaire de tiers n'est signé avec ces certificats.

## Rôles

| Rôle | Membres |
|---|---|
| **Auteurs** (committers de confiance) | [@itswinancher](https://github.com/itswinancher) |
| **Relecteurs** (revoient toute contribution externe avant fusion) | [@itswinancher](https://github.com/itswinancher) |
| **Approbateurs** (approuvent chaque demande de signature, une par une) | [@itswinancher](https://github.com/itswinancher) |

Tous les membres utilisent l'**authentification à plusieurs facteurs** sur GitHub, GitLab et SignPath.

## Processus

1. Une version est taguée (`vX.Y.Z`) dans ce dépôt.
2. GitHub Actions compile l'application à partir de ce tag ; le nom du produit et la version du tag sont inscrits dans les métadonnées du fichier.
3. L'exécutable Windows non signé est transmis à SignPath, qui vérifie qu'il provient bien de cette compilation.
4. Un approbateur **valide manuellement** la demande de signature.
5. Le fichier signé est publié sur GitHub, puis recopié à l'identique sur GitLab (sommes de contrôle vérifiées).

## Ce que fait le programme sur l'ordinateur

- **Modifications du système, toujours annoncées** : au premier lancement, un assistant explique puis demande l'accord avant d'installer un certificat limité aux adresses `.t2c` et de régler le proxy pour les seules adresses `.t2c`. Tout est retiré à la fermeture de l'application ou depuis ses réglages.
- **Désinstallation** : voir le [Guide complet](GUIDE.md#13-désinstaller).

## Vie privée

T2C Phantom ne contacte que le réseau T2C Phantom (relais et membres) pour afficher les sites `.t2c`. Il ne transmet aucune donnée personnelle, aucune statistique d'usage, et ne contacte aucun service tiers. Le **relais caché** (désactivable dans « Bénévolat ») transporte des données chiffrées pour d'autres membres, sans pouvoir les lire. Détails : [Données personnelles et RGPD](../RGPD.md).
