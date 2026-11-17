<div align="center">
  <img src="assets/phantom_logo.png" width="160" height="160" alt="T2C Phantom">

  # T2C Phantom

  **« Le savoir ne devrait pas toujours dépendre d'une connexion centralisée. »**

  <p>
    <a href="https://github.com/TUTODECODE-FR/T2C-Phantom/actions/workflows/ci.yml"><img src="https://github.com/TUTODECODE-FR/T2C-Phantom/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
    <a href="https://github.com/TUTODECODE-FR/T2C-Phantom/actions/workflows/release.yml"><img src="https://github.com/TUTODECODE-FR/T2C-Phantom/actions/workflows/release.yml/badge.svg" alt="Release"></a>
    <a href="https://github.com/TUTODECODE-FR/T2C-Phantom/actions/workflows/osv-scanner.yml"><img src="https://github.com/TUTODECODE-FR/T2C-Phantom/actions/workflows/osv-scanner.yml/badge.svg" alt="OSV"></a>
    <a href="https://github.com/TUTODECODE-FR/T2C-Phantom/actions/workflows/codeql.yml"><img src="https://github.com/TUTODECODE-FR/T2C-Phantom/actions/workflows/codeql.yml/badge.svg" alt="CodeQL"></a>
    <a href="https://github.com/TUTODECODE-FR/T2C-Phantom/actions/workflows/scorecard.yml"><img src="https://github.com/TUTODECODE-FR/T2C-Phantom/actions/workflows/scorecard.yml/badge.svg" alt="Scorecard"></a>
  </p>

  <b>Le réseau de partage de l'écosystème éducatif TUTODECODE / T2DECODE.</b><br>
  <i>Pairs pseudonymes · Bénévoles audités · Chiffré de bout en bout · Sites signés</i>
</div>

<img src="https://raw.githubusercontent.com/TUTODECODE-FR/T2DECODE/main/assets/separator.svg" width="100%" height="4">

## Présentation

T2C Phantom te permet d'ouvrir les sites **`.t2c`** de TUTODECODE (par exemple `https://tutodecode.t2c`) dans ton navigateur habituel. Ces sites ne sont pas servis par un seul serveur : ils sont **copiés et redistribués par des bénévoles**, comme toi, pendant que l'application tourne.

- **Pseudonyme** — tu n'es qu'une clé générée au hasard sur ta machine. Pas de compte, pas d'email.
- **Ton IP reste cachée** — seul Cloudflare (le point d'entrée) la voit ; le relais, le serveur et les autres pairs ne voient que ta clé.
- **Sites authentiques** — chaque site est signé par TUTODECODE et vérifié morceau par morceau : personne ne peut en modifier un octet en route.
- **Bénévole en option** — tu peux héberger une copie des sites que tu visites pour aider le réseau ; ta copie est contrôlée régulièrement.

Voir aussi : [Comment ça marche](docs/FONCTIONNEMENT.md) · [Guide d'utilisation](docs/UTILISATION.md) · [Sécurité](SECURITY.md)

<img src="https://raw.githubusercontent.com/TUTODECODE-FR/T2DECODE/main/assets/separator.svg" width="100%" height="4">

## Installation

Télécharge la dernière version dans les **[Releases](https://github.com/TUTODECODE-FR/T2C-Phantom/releases)** (aussi publiées sur [GitLab](https://gitlab.com/tutodecode-org/T2C-Phantom/-/releases)) :

| Système | Fichier | Remarque |
|---|---|---|
| macOS (Intel et Apple Silicon) | `T2C-Phantom-macOS-universal.zip` | 1er lancement : clic droit → **Ouvrir** (app pas encore signée par Apple) |
| Windows 10 / 11 | `T2C-Phantom-Windows-amd64.zip` | « Informations complémentaires » → **Exécuter quand même** si SmartScreen s'affiche |
| Linux | `T2C-Phantom-Linux-amd64.tar.gz` | nécessite `libwebkit2gtk-4.1` ; `libnss3-tools` pour la configuration automatique |
| Ligne de commande | `t2c-cli-…` | serveurs, scripts, sans interface |

Vérifie le fichier téléchargé avec `SHA256SUMS.txt` (publié avec chaque release).

## Démarrage

1. **Ouvre T2C Phantom.** Un assistant te guide une seule fois : certificat limité aux `.t2c`, choix du navigateur (Firefox conseillé), proxy, puis connexion au réseau pair à pair.
2. **Ouvre un site du carnet** d'un clic, ou tape son adresse `.t2c`.
3. **Deviens bénévole si tu veux** : choisis les sites que tu acceptes d'héberger ; tes copies sont chiffrées sur ton disque.

Tout le reste de ta navigation ne change pas. À la fermeture de l'application, tes réglages d'origine sont remis. Détails, ligne de commande et dépannage : **[Guide d'utilisation](docs/UTILISATION.md)**.

<img src="https://raw.githubusercontent.com/TUTODECODE-FR/T2DECODE/main/assets/separator.svg" width="100%" height="4">

## Licence
> [!WARNING]
> Ce dépôt public publie le **client** (application, daemon et protocole) en source-available : vous pouvez l'auditer et prouver l'absence de porte dérobée. **Ce n'est PAS open source** (pas GPLv3, pas MIT). Tous droits réservés TUTODECODE.
> Le **code du relais et du serveur origine** n'est pas publié.
> Utilisation, modification, distribution ou intégration commerciale par un tiers : **interdites** sans accord écrit de TUTODECODE.

**Pourquoi cette protection (Transparence Hybride) ?**
Tout ce qui tourne sur ta machine est public et vérifiable : c'est là que se jouent ta vie privée et l'intégrité des sites. La sécurité du réseau ne dépend pas du secret du serveur — les sites sont signés et chaque morceau est vérifié par ton client.

<img src="https://raw.githubusercontent.com/TUTODECODE-FR/T2DECODE/main/assets/separator.svg" width="100%" height="4">

<img src="https://raw.githubusercontent.com/TUTODECODE-FR/T2DECODE/main/assets/separator.svg" width="100%" height="4">

## Compiler depuis les sources

```bash
git clone https://github.com/TUTODECODE-FR/T2C-Phantom.git && cd T2C-Phantom
# clé publique racine du réseau TUTODECODE
KEY=985c6d08ed885e08d69611f861595b7d4d6ae7a11a0cfc4c6a7f8267fa2d984f
go build -ldflags "-X t2c/pkg/trust.RootPubKeyHex=$KEY" -o t2c ./cmd/t2c           # ligne de commande
cd desktop && wails build -ldflags "-X t2c/pkg/trust.RootPubKeyHex=$KEY"           # application (Wails v2)
```

Organisation du code : [ARCHITECTURE.md](ARCHITECTURE.md). Les releases sont construites par GitHub Actions ([release.yml](.github/workflows/release.yml)) et recopiées sur GitLab.

<img src="https://raw.githubusercontent.com/TUTODECODE-FR/T2DECODE/main/assets/separator.svg" width="100%" height="4">

## FAQ

**Est-ce un VPN ou Tor ?** Non. T2C Phantom ne touche qu'aux sites `.t2c`. Pour le reste de ta navigation, rien ne change.

**Qui voit quoi ?** Cloudflare voit ton IP et les sites `.t2c` que tu demandes ; le relais voit ta clé pseudonyme et avec quelles clés tu échanges, jamais ton IP ; les bénévoles ne voient que ta clé. Le contenu échangé entre pairs est chiffré de bout en bout. Détails : [Comment ça marche](docs/FONCTIONNEMENT.md#qui-voit-quoi).

**Un bénévole peut-il me servir un site piégé ?** Non : chaque morceau est comparé à l'empreinte signée par TUTODECODE ; un morceau modifié est rejeté, le bénévole est banni et signalé.

**Le certificat installé est-il dangereux ?** Il est généré sur ta machine et **ne peut signer que des noms `.t2c`** (contrainte inscrite dans le certificat) : même volé, il ne permettrait pas d'imiter ta banque ou un autre site.

**Hors ligne ?** Les sites déjà ouverts restent consultables depuis le cache ; pour en récupérer de nouveaux, il faut joindre le relais.

<img src="https://raw.githubusercontent.com/TUTODECODE-FR/T2DECODE/main/assets/separator.svg" width="100%" height="4">

## Roadmap

- **v2.0** — pairs pseudonymes, bénévoles audités, sites signés, application multi-OS, configuration automatique des navigateurs.
- **v2.1** — zéro confiance : clé racine hors ligne, serveur sans clé de publication, relais sans secret.
- **v2.2** — assistant de premier lancement, carnet de sites, bénévolat choisi site par site avec copies chiffrées, liste de relais signée.
- **Ensuite** — relais bénévoles (plusieurs points d'entrée), puis routage en oignon à travers eux pour l'anonymat complet.
- **Plus tard** — applications signées (Apple / Windows), mises à jour automatiques.

<img src="https://raw.githubusercontent.com/TUTODECODE-FR/T2DECODE/main/assets/separator.svg" width="100%" height="4">

## Soutenir le projet

T2C-Phantom et T2DECODE sont développés de manière indépendante et à but non lucratif. Si vous appréciez notre démarche d'éducation souveraine et sans traçage, vous pouvez soutenir notre association loi 1901 :

[![Soutenir TUTODECODE sur Liberapay](https://liberapay.com/assets/widgets/donate.svg)](https://liberapay.com/tutodecode/donate)

### Indicateurs de l'Association sur LiberaPay

<div align="center">
  <a href="https://liberapay.com/tutodecode/donate">
    <img alt="Reçoit sur Liberapay" src="https://img.shields.io/liberapay/receives/tutodecode.svg?logo=liberapay">
  </a>
  <a href="https://liberapay.com/tutodecode/donate">
    <img alt="Patrons sur Liberapay" src="https://img.shields.io/liberapay/patrons/tutodecode.svg?logo=liberapay">
  </a>
  <a href="https://liberapay.com/tutodecode/donate">
    <img alt="Objectif sur Liberapay" src="https://img.shields.io/liberapay/goal/tutodecode.svg?logo=liberapay">
  </a>
</div>

### Liens Liberapay
*   [Profil Principal](https://liberapay.com/tutodecode/)
*   [Faire un don direct](https://liberapay.com/tutodecode/donate)
*   [Widget d'encaissements](https://liberapay.com/tutodecode/receiving)
*   [Widget de dons](https://liberapay.com/tutodecode/giving)

<img src="https://raw.githubusercontent.com/TUTODECODE-FR/T2DECODE/main/assets/separator.svg" width="100%" height="4">

## Association TUTODECODE — mentions légales

Le projet T2DECODE est développé et soutenu par l'**Association TUTODECODE**, structure relevant de l'Économie Sociale et Solidaire (ESS).  
Notre mission est de démocratiser la maîtrise des infrastructures informatiques et de la cybersécurité en fournissant des outils souverains et respectueux de la vie privée.

- **Éditeur** : Association Loi 1901 TUTO DECODE
- **Directeur de Publication** : Maxime MARTIN CIVET
- **SIREN** : 102 763 133
- **Site Web Officiel** : [https://tutodecode.org](https://tutodecode.org)
- **Preuve Légale** : [Annonce de création parue au JOAFE](https://www.journal-officiel.gouv.fr/pages/associations-detail-annonce/?q.id=id:202600110336)
- **Engagement de Confidentialité** : [Consulter notre Politique RGPD](RGPD.md)

---
<div align="center">
  <i>© 2026 TUTODECODE - Tous droits réservés.</i>
</div>
