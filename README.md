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
- **Ton IP reste cachée** — seul le point d'entrée (Cloudflare) la voit ; les relais, le serveur et les autres pairs ne voient que ta clé.
- **Sites authentiques** — chaque site est signé par TUTODECODE et vérifié morceau par morceau : personne ne peut en modifier un octet en route.
- **Mode anonyme** — dès que le réseau compte plusieurs relais, tes demandes traversent un circuit chiffré en couches : aucun relais ne relie ton IP à ce que tu consultes. Cellules de taille fixe, trafic leurre et délais aléatoires brouillent aussi l'analyse du trafic.
- **Bénévole en option** — tu choisis les sites que tu acceptes d'héberger ; tes copies sont chiffrées sur ton disque et contrôlées régulièrement.
- **Sites officiels** — `tutodecode.t2c` (portail de tous les sites), `cours.t2c` (cours de TUTODECODE, fichier unique `cours.tdc` signé) et `repo.t2c` (code source, clonable avec `git` sans GitHub).

Voir aussi : [Guide complet](docs/GUIDE.md) · [Glossaire](docs/GUIDE.md#16-glossaire) · [Comment ça marche](docs/FONCTIONNEMENT.md) · [Guide d'utilisation](docs/UTILISATION.md) · [Sécurité](SECURITY.md)

<img src="https://raw.githubusercontent.com/TUTODECODE-FR/T2DECODE/main/assets/separator.svg" width="100%" height="4">

## Installation

Télécharge la dernière version dans les **[Releases](https://github.com/TUTODECODE-FR/T2C-Phantom/releases)** (aussi publiées sur [GitLab](https://gitlab.com/tutodecode-org/T2C-Phantom/-/releases)) :

| Système | Fichier | Remarque |
|---|---|---|
| macOS (Intel et Apple Silicon) | `T2C-Phantom-macOS-universal.zip` | 1er lancement : clic droit → **Ouvrir** (app pas encore signée par Apple) |
| Windows 10 / 11 | `T2C-Phantom-Windows-amd64.zip` | « Informations complémentaires » → **Exécuter quand même** si SmartScreen s'affiche |
| Linux | `T2C-Phantom-Linux-amd64.tar.gz` | nécessite `libwebkit2gtk-4.1-0` ; `libnss3-tools` pour la configuration automatique |
| Ligne de commande | `t2c-cli-<version>-<système>-<architecture>` (`darwin-arm64`, `darwin-amd64`, `linux-amd64`, `linux-arm64`, `windows-amd64`) | serveurs, scripts, sans interface ([mode d'emploi](docs/GUIDE.md#10-ligne-de-commande)) |

Vérifie le fichier téléchargé avec `SHA256SUMS.txt`, publié avec chaque release, dans le dossier où se trouvent les deux fichiers :

```bash
shasum -a 256 -c SHA256SUMS.txt --ignore-missing     # macOS
sha256sum -c SHA256SUMS.txt --ignore-missing         # Linux
```

```powershell
# Windows (PowerShell) : affiche la ligne correspondante si le fichier est intact
Select-String -Path SHA256SUMS.txt -Pattern (Get-FileHash T2C-Phantom-Windows-amd64.zip -Algorithm SHA256).Hash.ToLower()
```

Le fichier doit être marqué `OK` (ou, sous Windows, la ligne doit s'afficher). Sinon, **ne l'ouvre pas** : retélécharge-le, un téléchargement interrompu suffit à fausser la somme.

## Démarrage

1. **Ouvre T2C Phantom.** Un assistant te guide une seule fois : certificat limité aux `.t2c`, choix du navigateur (Firefox conseillé), proxy, puis connexion au réseau pair à pair.
2. **Ouvre un site du carnet** d'un clic, ou tape son adresse `.t2c`.
3. **Deviens bénévole si tu veux** : choisis les sites que tu acceptes d'héberger ; tes copies sont chiffrées sur ton disque.

> **Le premier chargement d'une page peut prendre quelques secondes : c'est normal.** Les morceaux du site traversent deux relais chiffrés et viennent de bénévoles. Seule la page ouverte est téléchargée, puis gardée en cache ; un gros fichier affiche sa progression dans le navigateur.

Tout le reste de ta navigation ne change pas. À la fermeture de l'application, tes réglages d'origine sont remis. Tout savoir (utilisation, bénévolat, fonctionnement, sécurité, dépannage) : **[Guide complet](docs/GUIDE.md)**.

## Trajet d'une requête

```
Mode normal
  Navigateur ──PAC (.t2c seulement)──▶ proxy local 127.0.0.1 + certificat local .t2c
      ──TLS──▶ Cloudflare (voit : IP, clé, site demandé)
      ──▶ relais (voit : clé, site demandé ; jamais l'IP)
      ──chiffré de bout en bout (Noise)──▶ bénévole (voit : clé)
      └─ repli : serveur d'origine, joint uniquement par le relais (réseau privé)

Mode anonyme (au moins deux relais)
  Navigateur ──▶ proxy local ──TLS──▶ Cloudflare (voit : IP, clé ; pas le site)
      ──▶ relais d'entrée (voit : clé ; pas le site, cellules chiffrées)
      ──▶ [relais intermédiaire] ──▶ relais de sortie (voit : site ; ni IP ni clé, pseudonyme jetable)
      ──chiffré de bout en bout──▶ bénévole (voit : pseudonyme jetable)
  Dans le circuit : cellules de 4 tailles fixes seulement, cellules leurres en continu,
  délai aléatoire à la sortie → ni la taille ni le rythme ne trahissent l'activité.

À chaque étape : chaque morceau reçu est vérifié contre l'empreinte signée par TUTODECODE.
```

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

## Démarrage rapide (développeurs)

Prérequis : [Go](https://go.dev/dl/) 1.26.11 ou plus récent (version exigée par `go.mod`) ; pour l'application, Wails v2 (`go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.1`) et, sous Linux, `sudo apt install libgtk-3-dev libwebkit2gtk-4.1-dev`.

```bash
git clone https://github.com/TUTODECODE-FR/T2C-Phantom.git && cd T2C-Phantom
go test ./...                                   # tests du client (protocole, signatures, circuits, stockage)

KEY=985c6d08ed885e08d69611f861595b7d4d6ae7a11a0cfc4c6a7f8267fa2d984f   # clé publique racine de TUTODECODE
go build -ldflags "-X t2c/pkg/trust.RootPubKeyHex=$KEY" -o t2c ./cmd/t2c
go build -ldflags "-X t2c/pkg/trust.RootPubKeyHex=$KEY" -o t2c-peer ./cmd/t2c-peer

export T2C_NO_BROWSER_SETUP=1 T2C_DATA_DIR=./donnees-test   # isole tout dans ./donnees-test
./t2c doctor                                    # diagnostic (relais joignables, port, dossier)
./t2c-peer list                                 # catalogue signé et bénévoles en ligne
./t2c start -proxy 18080 -no-volunteer          # client + proxy local ; laisse-le tourner (Ctrl+C pour arrêter)
```

Dans un **second terminal**, depuis le même dossier :

```bash
curl -x http://127.0.0.1:18080 http://tutodecode.t2c/                                              # HTTP
curl --cacert ./donnees-test/asso-ca/asso-ca.crt -x http://127.0.0.1:18080 https://tutodecode.t2c/  # HTTPS, certificat local
```

Application de bureau :

```bash
cd desktop && wails dev                                            # mode développement
wails build -ldflags "-X t2c/pkg/trust.RootPubKeyHex=$KEY"         # résultat dans desktop/build/bin/ (« T2C Phantom.app », T2C-Phantom.exe ou T2C-Phantom)
```

`T2C_NO_BROWSER_SETUP=1` empêche toute modification des réglages de ton système et de tes navigateurs ; `T2C_DATA_DIR` isole l'identité et les données de test. Organisation du code : [ARCHITECTURE.md](ARCHITECTURE.md). Les releases sont construites par GitHub Actions ([release.yml](.github/workflows/release.yml)) puis recopiées sur GitLab, fichiers et sommes de contrôle vérifiés avant la copie.

<img src="https://raw.githubusercontent.com/TUTODECODE-FR/T2DECODE/main/assets/separator.svg" width="100%" height="4">

## FAQ

**Est-ce un VPN ou Tor ?** Non. T2C Phantom ne touche qu'aux sites `.t2c`. Pour le reste de ta navigation, rien ne change.

**Qui voit quoi ?** Cloudflare voit ton IP (et, hors mode anonyme, les sites `.t2c` demandés) ; les relais voient ta clé pseudonyme, jamais ton IP ; en mode anonyme, aucun relais ne voit à la fois ta clé et ce que tu consultes. Le contenu échangé entre pairs est chiffré de bout en bout. Détails : [Guide complet](docs/GUIDE.md#8-qui-voit-quoi).

**Pourquoi passer par Cloudflare, un acteur centralisé ?** Choix pragmatique, assumé et limité :
- **Ce qu'il apporte** : il cache l'adresse du relais (personne ne peut l'attaquer directement) et absorbe les attaques par saturation, ce qu'un petit relais associatif ne peut pas faire seul.
- **Ce qu'il voit** : ton IP, comme tout point d'entrée (une connexion révèle toujours l'adresse de celui qui l'ouvre). Le nom `.t2c` ne circule jamais sur le réseau : le TLS est négocié avec le **domaine du relais** (SNI et en-tête Host du relais). Cloudflare termine toutefois ce TLS : en mode normal, il pourrait lire les demandes (empreintes des sites, rapprochables du catalogue public) ; **en mode anonyme, il ne voit que des cellules chiffrées pour les relais**, ainsi que leur taille et leur rythme, jamais le site demandé.
- **Ce qu'il ne peut pas faire** : lire les échanges entre membres (chiffrés de bout en bout) ni modifier un site (signatures vérifiées par ton client).
- **Si même lui ne doit pas connaître ton IP** : passe par un VPN ou Tor avant T2C Phantom. La fédération permet aussi d'ajouter des relais d'entrée qui ne dépendent pas de Cloudflare.

**Un observateur peut-il deviner mon activité en regardant le trafic ?** C'est beaucoup plus difficile en mode anonyme : toutes les cellules ont l'une de 4 tailles fixes (remplissage aléatoire), des cellules leurres circulent en permanence, et le relais de sortie ajoute un délai aléatoire. Ni la taille, ni le rythme, ni le moment ne permettent facilement de relier une entrée à une sortie. Comme pour Tor, un adversaire qui contrôle à la fois l'entrée et la sortie d'un circuit garde des moyens de corrélation : la diversité des opérateurs de relais reste la meilleure protection.

T2C Phantom n'est pas un outil d'anonymat absolu comme Tor : son objectif est de **protéger ton adresse IP** et de **garantir l'authenticité des sites**, avec un mode anonyme qui coupe le lien entre ton adresse et ce que tu consultes.

**Un bénévole peut-il me servir un site piégé ?** Non : chaque morceau est comparé à l'empreinte signée par TUTODECODE ; un morceau modifié est rejeté, le bénévole est banni et signalé.

**Le certificat installé est-il dangereux ?** Il est généré sur ta machine et **ne peut signer que des noms `.t2c`** (contrainte inscrite dans le certificat) : même volé, il ne permettrait pas d'imiter ta banque ou un autre site.

**C'est lent ?** Un peu, au premier passage, et c'est le prix de l'anonymat : chaque demande traverse deux relais, avec des cellules de taille fixe et de petits délais aléatoires. Depuis la v2.5, l'application ne télécharge que la page ouverte (pas le site entier), garde ce qu'elle a reçu en cache, et envoie les gros fichiers au fil de l'eau. Plus il y a de bénévoles et de relais, plus c'est rapide.

**Hors ligne ?** Les sites déjà ouverts restent consultables depuis le cache ; pour en récupérer de nouveaux, il faut joindre le relais.

<img src="https://raw.githubusercontent.com/TUTODECODE-FR/T2DECODE/main/assets/separator.svg" width="100%" height="4">

## Roadmap

- **v2.0** — pairs pseudonymes, bénévoles audités, sites signés, application multi-OS, configuration automatique des navigateurs.
- **v2.1** — zéro confiance : clé racine hors ligne, serveur sans clé de publication, relais sans secret.
- **v2.2** — assistant de premier lancement, carnet de sites, bénévolat choisi site par site avec copies chiffrées, liste de relais signée.
- **v2.3** — fédération de relais : plusieurs points d'entrée reliés entre eux.
- **v2.4** — mode anonyme : routage en oignon à travers les relais, pseudonyme jetable, échec fermé ; relais cachés (aucune écoute sur Internet) ; protection contre l'analyse de trafic (cellules de taille fixe, trafic leurre, délais aléatoires) ; correctifs issus d'un pentest.
- **v2.5** — sites chargés à la demande (seulement la page ouverte, gros fichiers au fil de l'eau) ; liste des relais signée par la clé racine (un serveur piraté ne peut plus ajouter de relais espion) ; clé racine chiffrée par phrase de passe ; sites officiels `tutodecode.t2c`, `cours.t2c` (`cours.tdc` signé, vérifié aussi par T2DECODE) et `repo.t2c` ; vérification avant envoi (CI locale et pentest sur le poste du développeur).
- **Plus tard** — applications signées (Apple / Windows), mises à jour automatiques.

### Idées pour la suite

- **Relais bénévoles derrière Cloudflare** : un bénévole fait tourner un relais avec un tunnel Cloudflare fourni par l'association (jeton révocable). Son adresse IP n'est jamais publiée, et l'association reste seule à décider qui devient relais (liste signée).
- **Cloudflare devant le relais principal** : le VPS ne voit plus les adresses IP des utilisateurs.
- **Releases signées** par la clé racine, vérifiables depuis `tutodecode.t2c`.
- **Mode « relais bénévole » dans l'application**, activable en un clic.
- **Mises à jour de sites par différence** : ne renvoyer aux bénévoles que les morceaux qui ont changé.
- **Recherche dans les sites** et **favoris hors ligne** dans le carnet.

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
