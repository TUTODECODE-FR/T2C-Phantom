# Code signing policy

**Politique de signature du code de T2C Phantom**

**Windows : pas encore signé.** La demande à SignPath Foundation a été refusée le 2026-11-22 (projet pas encore assez connu publiquement) ; une nouvelle demande sera faite après l'ouverture publique du projet. D'ici là, Windows affiche l'alerte SmartScreen : « Informations complémentaires » → « Exécuter quand même », après avoir vérifié la somme SHA-256.

## Ce qui est signé

| Système | Signature | Par qui |
|---|---|---|
| **Windows** | **pas encore signé** (sommes SHA-256 publiées) | prévu : Authenticode, via SignPath Foundation ou un autre fournisseur |
| **macOS** | Developer ID + notarisation Apple | certificat **Developer ID Application: TUTODECODE (TS97M57BJV)** |
| **Linux** | sommes SHA-256 publiées avec chaque version (`SHA256SUMS.txt`) | — |

Seuls les programmes **construits par l'intégration continue GitHub Actions à partir du code source de ce dépôt** sont signés : l'application T2C Phantom et ses outils en ligne de commande. Aucun programme compilé sur un poste personnel, aucun binaire de tiers n'est signé avec ces certificats.

## Rôles

| Rôle | Membres |
|---|---|
| **Auteurs** (committers de confiance) | [@itswinancher](https://github.com/itswinancher) |
| **Relecteurs** (revoient toute contribution externe avant fusion) | [@itswinancher](https://github.com/itswinancher) |
| **Approbateurs** (approuvent chaque demande de signature, une par une) | [@itswinancher](https://github.com/itswinancher) |

Tous les membres utilisent l'**authentification à plusieurs facteurs** sur GitHub, GitLab et Apple.

## Processus

1. Une version est taguée (`vX.Y.Z`) dans ce dépôt.
2. GitHub Actions compile l'application à partir de ce tag ; le nom du produit et la version du tag sont inscrits dans les métadonnées du fichier.
3. L'application macOS est signée avec le certificat Developer ID puis notarisée par Apple. (Windows, dès qu'une signature sera en place : un approbateur validera manuellement chaque demande.)
4. Les sommes SHA-256 de tous les fichiers sont publiées avec la version.
5. Le fichier signé est publié sur GitHub, puis recopié à l'identique sur GitLab (sommes de contrôle vérifiées).

## Ce que fait le programme sur l'ordinateur

- **Modifications du système, toujours annoncées** : au premier lancement, un assistant explique puis demande l'accord avant d'installer un certificat limité aux adresses `.t2c` et de régler le proxy pour les seules adresses `.t2c`. Tout est retiré à la fermeture de l'application ou depuis ses réglages.
- **Désinstallation** : voir le [Guide complet](GUIDE.md#13-désinstaller).

## Vie privée

T2C Phantom ne contacte que le réseau T2C Phantom (relais et membres) pour afficher les sites `.t2c`. Il ne transmet aucune donnée personnelle, aucune statistique d'usage, et ne contacte aucun service tiers. Le **relais caché** (désactivable dans « Bénévolat ») transporte des données chiffrées pour d'autres membres, sans pouvoir les lire. Détails : [Données personnelles et RGPD](../RGPD.md).
