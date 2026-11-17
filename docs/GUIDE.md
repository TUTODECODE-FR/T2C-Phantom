# Guide complet de T2C Phantom

Ce guide explique tout : ce qu'est T2C Phantom, ce que tu peux en faire, comment l'installer et l'utiliser, et comment il fonctionne à l'intérieur.

- [1. En bref](#1-en-bref)
- [2. Ce que tu peux faire](#2-ce-que-tu-peux-faire)
- [3. Installer](#3-installer)
- [4. Premier lancement](#4-premier-lancement)
- [5. Au quotidien](#5-au-quotidien)
- [6. Devenir bénévole](#6-devenir-bénévole)
- [7. Comment ça fonctionne](#7-comment-ça-fonctionne)
- [8. Qui voit quoi](#8-qui-voit-quoi)
- [9. Technologies utilisées](#9-technologies-utilisées)
- [10. Ligne de commande](#10-ligne-de-commande)
- [11. Données locales](#11-données-locales)
- [12. Dépannage](#12-dépannage)
- [13. Désinstaller](#13-désinstaller)
- [14. Limites connues](#14-limites-connues)
- [15. Questions fréquentes](#15-questions-fréquentes)

---

## 1. En bref

**T2C Phantom** est le réseau pair à pair de l'association **TUTODECODE**. Il sert des sites en `.t2c` (par exemple `tutodecode.t2c`) :

- **sans compte** : ton identité est une clé aléatoire créée sur ton ordinateur, sans nom ni adresse e-mail ;
- **sans exposer ton adresse IP** aux autres membres : personne dans le réseau ne se connecte directement à toi ;
- **sans site falsifiable** : chaque site est signé par TUTODECODE et chaque morceau est vérifié ;
- **grâce aux bénévoles** : les membres qui le souhaitent gardent une copie des sites et la redistribuent, ce qui soulage le serveur ;
- **avec un mode anonyme** : quand le réseau compte plusieurs relais, tes demandes traversent un circuit chiffré en couches, et aucun relais ne peut relier ton adresse à ce que tu consultes.

Le reste de ta navigation n'est **jamais** concerné : seules les adresses `.t2c` passent par T2C Phantom.

---

## 2. Ce que tu peux faire

| Action | Où |
|---|---|
| Ouvrir un site `.t2c` dans ton navigateur habituel | Carnet de sites → **Ouvrir**, ou directement `https://tutodecode.t2c` dans le navigateur |
| Vérifier qu'un site répond et qu'il est authentique | Carnet de sites → **Tester** |
| Ouvrir une adresse `.t2c` qui n'est pas dans la liste | Champ en bas du carnet |
| Aider le réseau en hébergeant des sites | Section **Bénévolat** |
| Choisir les sites hébergés, la place disque et le débit d'envoi | Section **Bénévolat** |
| Voir ton identité, ton relais, l'état de l'anonymat | **Réglages** |
| Utiliser un relais précis | **Réglages** → Relais préféré |
| Refaire la configuration (certificat, navigateur, proxy) | **Réglages** → Refaire l'assistant |
| Désactiver le proxy | **Réglages** → Désactiver le proxy |
| Utiliser T2C Phantom sans interface (serveur, Raspberry Pi) | Ligne de commande, [section 10](#10-ligne-de-commande) |

---

## 3. Installer

Télécharge le fichier de ton système dans les **Releases** du dépôt, puis vérifie-le avec `SHA256SUMS.txt` :

```bash
shasum -a 256 -c SHA256SUMS.txt --ignore-missing     # macOS
sha256sum -c SHA256SUMS.txt --ignore-missing         # Linux
```

```powershell
Get-FileHash T2C-Phantom-Windows-amd64.zip           # Windows : compare avec SHA256SUMS.txt
```

| Système | Fichier | Installation |
|---|---|---|
| macOS (Intel et Apple Silicon) | `T2C-Phantom-macOS-universal.zip` | Dézippe, puis clic droit sur **T2C Phantom** → **Ouvrir** (l'application n'est pas signée par Apple). Elle propose de se déplacer dans **Applications**. |
| Windows 10 / 11 | `T2C-Phantom-Windows-amd64.zip` | Dézippe et lance `T2C-Phantom.exe`. Si SmartScreen s'affiche : **Informations complémentaires** → **Exécuter quand même**. Elle propose de s'installer avec un raccourci sur le Bureau et dans le menu Démarrer. |
| Linux (x86_64) | `T2C-Phantom-Linux-amd64.tar.gz` | `tar xzf T2C-Phantom-Linux-amd64.tar.gz && ./T2C-Phantom`. Prérequis Debian/Ubuntu : `sudo apt install libwebkit2gtk-4.1-0 libnss3-tools`. Elle propose de s'installer dans `~/.local/bin` avec une entrée de menu. |
| Ligne de commande | `t2c-cli-<version>-<système>-<architecture>` | Voir [section 10](#10-ligne-de-commande). |

---

## 4. Premier lancement

Un **assistant en 4 étapes** s'affiche une seule fois.

1. **Certificat.** T2C Phantom crée sur ton ordinateur un certificat qui ne peut signer **que** les adresses `.t2c`. Il est impossible de s'en servir pour imiter un autre site (contrainte de nom reconnue par tous les navigateurs modernes). Ton système peut demander ton mot de passe pour l'ajouter. S'il est déjà installé, l'étape est validée d'office.
2. **Navigateur.** Choisis le navigateur qui ouvrira les sites. **Firefox est conseillé** : il prend en compte le certificat et le proxy sans aucun réglage. Chrome, Edge, Safari, Brave, Zen et LibreWolf fonctionnent aussi.
3. **Proxy.** Un fichier de configuration automatique (PAC) envoie **uniquement** les adresses `.t2c` vers T2C Phantom ; tout le reste de ta navigation part directement, comme avant. Si ton navigateur était ouvert, ferme-le puis rouvre-le.
4. **Réseau.** T2C Phantom crée ton identité (une clé aléatoire, quelques secondes de calcul la première fois), puis rejoint le réseau.

Ensuite, l'application n'affiche plus que le carnet de sites et le bénévolat. L'assistant peut être relancé à tout moment depuis les **Réglages**.

---

## 5. Au quotidien

- **Laisse l'application ouverte** pendant que tu consultes des sites `.t2c`. À la fermeture, tes réglages proxy d'origine sont remis ; à l'ouverture suivante, tout se reconfigure seul.
- **En haut de la fenêtre** :
  - *Connecté*, *Connexion…* ou *Déconnecté* : l'état du lien avec le réseau ;
  - *Anonyme* : tes demandes passent par un circuit en oignon ([section 7.7](#77-mode-anonyme-routage-en-oignon)).
- **Carnet de sites** : la liste officielle, signée par TUTODECODE. **Ouvrir** lance le site dans ton navigateur ; **Tester** vérifie qu'il répond, que sa signature et son contenu sont intacts.
- **Réglages** : ton identité pseudonyme, le relais utilisé, l'état de l'anonymat, le nombre de morceaux reçus des bénévoles et du serveur, et les faux morceaux rejetés.

---

## 6. Devenir bénévole

Le bénévolat est **désactivé par défaut**. Pour aider le réseau :

1. Section **Bénévolat** → active l'interrupteur.
2. Coche les sites que tu acceptes d'héberger, ou **Tous les sites du catalogue**.
3. Fixe la **place maximale** (2 048 Mo par défaut) et l'**envoi maximal** (1 024 Ko/s par défaut).
4. **Enregistrer.**

Ce qui se passe ensuite :

- Ton application demande au réseau un **ticket** pour chaque site choisi : une autorisation signée, liée à ta clé et à cette version du site.
- Elle télécharge une copie **vérifiée morceau par morceau**, puis l'annonce au relais.
- Tes copies sont **chiffrées sur ton disque** (AES-256-GCM, clé locale aléatoire) : aucun fichier lisible ou modifiable à la main. À chaque envoi, le morceau est déchiffré puis revérifié ; une copie altérée n'est **jamais servie**, elle est supprimée puis retéléchargée proprement.
- Le relais te **contrôle** régulièrement : il demande un morceau au hasard combiné à un nombre aléatoire, ce que seul un bénévole qui possède vraiment le morceau intact peut calculer. Réussite : ton score monte. Échecs répétés : tu es retiré, puis banni.
- Quand TUTODECODE publie une **nouvelle version** d'un site, ta copie est mise à jour automatiquement et l'ancienne est supprimée.
- Les autres membres ne connaissent que ta **clé pseudonyme**, jamais ton adresse. En mode anonyme, toi non plus tu ne vois que des pseudonymes jetables.

Pour arrêter : désactive l'interrupteur. Tes copies sont supprimées et tu es retiré de l'annuaire immédiatement.

---

## 7. Comment ça fonctionne

### 7.1 Architecture en étoile

```
              Internet                        réseau privé
  ┌────────┐                                (inaccessible depuis Internet)
  │ membre │──┐
  └────────┘  │   ┌──────────────┐   ┌───────────┐      ┌─────────────────┐
  ┌────────┐  ├──▶│ point d'entrée│──▶│  relais   │─────▶│ serveur d'origine│
  │ membre │──┤   │ (Cloudflare)  │   │ principal │      │  (TUTODECODE)    │
  └────────┘  │   └──────────────┘   └─────┬─────┘      └─────────────────┘
  ┌────────┐  │                            │ fédération (liaisons chiffrées)
  │bénévole│──┘                   ┌────────┴────────┐
  └────────┘                      │ autres relais   │  (publics ou cachés)
                                  └─────────────────┘
```

- **Personne ne se connecte directement à personne.** Chaque application ne parle qu'à un relais. Les membres ne connaissent les uns des autres que des **clés pseudonymes**, et leurs échanges sont **chiffrés de bout en bout** : le relais les transmet sans pouvoir les lire.
- **Le serveur d'origine** n'est pas sur Internet : seul le relais principal le joint, par un réseau privé chiffré. Les autres relais passent par le relais principal et ne connaissent jamais son adresse.
- **Le point d'entrée public** est Cloudflare : c'est le seul à voir les adresses IP des membres. Tant que le nom de domaine n'est pas actif, le relais principal sert lui-même de point d'entrée et voit donc ces adresses.

### 7.2 Identité pseudonyme

- Ton identité est une **clé Ed25519 aléatoire**, créée sur ton ordinateur et gardée dans `identity.json`. Ton identifiant (32 caractères) en est l'empreinte.
- Pour limiter la création massive de fausses identités, chaque clé doit porter une **preuve de travail** (22 bits, quelques secondes de calcul une seule fois).
- À chaque connexion, le relais envoie un défi aléatoire que tu signes : il vérifie que tu possèdes la clé, sans jamais apprendre qui tu es.
- Une seconde clé (X25519), signée par la première, sert au chiffrement de bout en bout.
- Supprimer `identity.json` crée une identité neuve, sans lien avec l'ancienne.

### 7.3 Authenticité et intégrité des sites

- TUTODECODE publie un **catalogue signé** (Ed25519) : la liste des sites et l'empreinte de chacun. Sa version ne peut qu'augmenter : un ancien catalogue rejoué est refusé.
- Chaque site est un fichier découpé en **morceaux de 256 Ko**. Son **manifeste signé** donne l'empreinte SHA-256 de chaque morceau.
- Ton application connaît la **clé publique de TUTODECODE**, inscrite dans le programme. Elle refuse tout document mal signé et tout morceau dont l'empreinte ne correspond pas.
- La **clé racine** de TUTODECODE n'est sur **aucun serveur** : les sites sont signés hors ligne, puis envoyés au serveur sous forme de paquet qu'il revérifie entièrement.

Conséquence : **ni un bénévole, ni un relais, ni Cloudflare, ni même un serveur piraté ne peut modifier un site** sans que ce soit détecté. Un morceau faux est rejeté et son expéditeur banni de ton côté.

### 7.4 Téléchargement d'un site

1. Ton application résout l'adresse (`tutodecode.t2c`) dans le catalogue signé.
2. Elle demande au relais quels bénévoles hébergent cette version, classés par score.
3. Elle télécharge plusieurs morceaux en parallèle chez les meilleurs bénévoles, par des sessions **chiffrées de bout en bout** (Noise IK).
4. Chaque morceau est vérifié ; en cas d'échec, elle passe à un autre bénévole.
5. En dernier recours, le morceau vient du serveur d'origine, via le relais (avec un cache côté relais).
6. Le site, vérifié, est servi à ton navigateur par le proxy local, en HTTPS grâce au certificat `.t2c`.

### 7.5 Bénévoles, tickets et score

- **Ticket** : autorisation signée, de durée limitée, liée à une clé et à une version de site. Sans ticket valide, un bénévole n'est pas proposé aux autres.
- **Contrôles** : défis de stockage aléatoires (empreinte d'un nombre aléatoire et d'un morceau). Impossible à réussir sans posséder le morceau intact.
- **Score et bannissement** : publiés dans une **liste signée** (`hosts.json`). Les membres choisissent d'abord les bénévoles les mieux notés, à la fois par le réseau et par leur propre expérience.
- **Signalement** : un membre qui reçoit un faux morceau le signale ; le bénévole est contrôlé aussitôt.

### 7.6 Plusieurs relais (fédération)

- Le réseau peut compter **plusieurs relais**, tenus par des opérateurs de confiance et inscrits avec leur clé dans la liste signée. Ton application en choisit un au hasard, ce qui répartit la charge.
- Les relais se relient entre eux par une liaison **authentifiée des deux côtés** : seuls ceux de la liste signée sont acceptés. Ils partagent l'annuaire des bénévoles et se transmettent les échanges chiffrés : tu peux télécharger chez un bénévole connecté à un autre relais.
- Un relais ne retransmet jamais une trame reçue d'un autre relais (un seul saut entre relais).
- Un **relais caché** n'écoute rien sur Internet : c'est lui qui se connecte au relais principal par un réseau privé. Son adresse n'est publiée nulle part ; il ne sert que de relais intermédiaire ou de sortie dans les circuits anonymes.

### 7.7 Mode anonyme (routage en oignon)

Dès que la liste signée compte au moins deux relais, l'application passe **d'elle-même** en mode anonyme :

- Tes demandes traversent un **circuit** de 2 ou 3 relais différents, chiffré une fois par relais (AES-256-GCM). Chaque relais ne retire qu'une couche.
- Avec chaque relais, ton application négocie des **clés éphémères** (échange X25519 authentifié). Même une clé de relais volée plus tard ne permet pas de déchiffrer un échange passé.
- Le **relais d'entrée** connaît ton adresse IP, mais ne peut rien lire de ce que tu demandes.
- Le **relais de sortie** lit les demandes (quel site, quels morceaux), mais ne connaît ni ton IP ni ton identité : seulement un **pseudonyme jetable**, renouvelé toutes les 10 minutes.
- Les **bénévoles** ne voient que ce pseudonyme.
- **Échec fermé** : si le circuit tombe, rien n'est envoyé en clair à sa place ; l'application en reconstruit un.
- **Contre l'analyse de trafic** :
  - **tailles fixes** : chaque cellule est complétée (remplissage aléatoire) jusqu'à l'une de 4 tailles (1 Ko, 8 Ko, 65 Ko, 260 Ko) ; une recherche, une demande de catalogue et une cellule leurre sont indiscernables ;
  - **trafic leurre** : l'application envoie de fausses cellules toutes les 2 à 8 secondes, et le relais de sortie y répond parfois par de fausses réponses ; le rythme du trafic ne trahit plus l'activité réelle ;
  - **délais aléatoires** : le relais de sortie attend 10 à 80 ms avant de traiter une petite demande (au plus 10 à 15 ms pour les morceaux de site), ce qui casse la corrélation des temps entre l'entrée et la sortie du circuit.

Si quelqu'un interroge un relais ou un bénévole sur « qui a demandé quoi », il ne peut pas répondre : il ne possède pas l'information.

### 7.8 Serveur d'origine en « zéro confiance »

- La **clé racine** reste hors ligne, chez l'éditeur. Le serveur ne détient qu'une **clé de service déléguée** (180 jours au plus), qui ne peut signer que les tickets et la liste des bénévoles, jamais un site ni le catalogue.
- Le relais s'authentifie auprès du serveur en **signant chaque requête** avec sa propre clé (horodatage et nombre aléatoire : non rejouable). Le serveur ne stocke aucun secret partagé, seulement des clés publiques.
- Le serveur refuse toute requête qui porterait une adresse IP d'utilisateur.

### 7.9 Intégration au navigateur

- **Certificat local** limité aux noms `.t2c` (contrainte de nom critique) : même volé, il ne permet d'imiter aucun autre site.
- **Fichier PAC** : seuls les `*.t2c` sont envoyés au proxy local (`127.0.0.1:8080`), tout le reste part directement.
- Firefox, Zen et LibreWolf sont réglés pour lire les certificats du système.
- Le proxy local n'accepte que les sites du catalogue signé et refuse tout le reste (ce n'est pas un proxy ouvert).

---

## 8. Qui voit quoi

| | Ton adresse IP | Ta clé | Les sites que tu demandes | Le contenu échangé entre membres |
|---|---|---|---|---|
| **Cloudflare** (point d'entrée) | oui | oui | mode normal : oui ; mode anonyme : **non** | non (chiffré de bout en bout) |
| **Relais d'entrée** | non (oui tant qu'il sert lui-même de point d'entrée) | oui | mode normal : oui ; mode anonyme : **non** | non |
| **Relais de sortie** (mode anonyme) | non | **non** : pseudonyme jetable | oui | non |
| **Bénévoles** | non | mode normal : oui ; mode anonyme : pseudonyme jetable | seulement ce qu'ils te fournissent | — |
| **Serveur d'origine** | non | seulement si tu es bénévole | non : il ne reçoit que des demandes anonymes du relais | — |

Précisions :

- Le nom `.t2c` ne circule **jamais** sur le réseau : ton proxy local l'intercepte, et la connexion vers Cloudflare est établie avec le **domaine du relais** (SNI TLS et en-tête Host du relais), dans les deux modes.
- Cloudflare **termine** cette connexion TLS : il voit en clair les trames échangées avec le relais. En mode normal, ces trames contiennent les demandes (empreintes des sites, rapprochables du catalogue public). En mode anonyme, elles ne contiennent que des cellules chiffrées en couches pour les relais : Cloudflare n'en connaît que la taille et le rythme.
- Le contenu échangé avec les bénévoles est chiffré de bout en bout (Noise) dans tous les cas.
- Le relais et le serveur ne lisent ni n'enregistrent aucun en-tête contenant une adresse IP, et le serveur refuse toute requête qui en porterait une.

---

## 9. Technologies utilisées

| Domaine | Technologie |
|---|---|
| Langage | Go |
| Application de bureau | Wails (interface web intégrée, sans ressource distante) |
| Identité et signatures | Ed25519, preuve de travail de 22 bits |
| Chiffrement de bout en bout entre membres | Noise IK (X25519, ChaCha20-Poly1305, BLAKE2s) |
| Circuits anonymes | échange X25519 authentifié à clés éphémères, couches AES-256-GCM, cellules de taille fixe, trafic leurre, délais aléatoires |
| Intégrité | SHA-256 par morceau de 256 Ko, manifestes et catalogue signés |
| Copies des bénévoles | AES-256-GCM sur le disque, revérifiées à chaque lecture |
| Transport | WebSocket sur TLS 1.2/1.3, sous-protocole `t2c.v2` obligatoire |
| Relais ↔ serveur | réseau privé chiffré, requêtes signées Ed25519 non rejouables |
| Navigateur | certificat local à contrainte de nom `.t2c`, fichier PAC |
| Publication | GitHub Actions (applications macOS, Windows, Linux et outils en ligne de commande), sommes SHA-256, copie sur GitLab |

---

## 10. Ligne de commande

Pour un serveur, un Raspberry Pi ou sans interface graphique, télécharge l'archive `t2c-cli-…` de ton système. Elle contient `t2c` et `t2c-peer`.

```bash
./t2c doctor          # diagnostic : clé racine, relais, port 8080, dossier de données
./t2c setup           # configure les navigateurs (une fois) : certificat .t2c + proxy PAC
./t2c start           # client + proxy .t2c sur 127.0.0.1:8080
./t2c unsetup         # remet les réglages proxy d'origine
./t2c id              # affiche ton identité pseudonyme
./t2c dashboard       # tableau de bord dans le navigateur (http://127.0.0.1:9090)
```

| Option de `t2c start` | Rôle |
|---|---|
| `-proxy 8080` | port du proxy local (`0` = désactivé) |
| `-host site1,site2` | sites à héberger en tant que bénévole |
| `-no-volunteer` | n'héberger aucune copie |
| `-relay wss://…` | relais préféré (ou variable `T2C_RELAY_URL`) |
| `-carnet fichier.json` | alias locaux (`{"cours.t2c": "tutodecode"}`) |
| `-metrics-port 2112` | métriques Prometheus sur `127.0.0.1` |

Client minimal, sans proxy :

```bash
./t2c-peer list                       # catalogue signé et bénévoles en ligne
./t2c-peer fetch tutodecode ./site    # télécharge et vérifie un site dans ./site
./t2c-peer host tutodecode            # devient bénévole pour ce site (reste connecté)
```

---

## 11. Données locales

| Système | Dossier |
|---|---|
| macOS | `~/Library/Caches/t2c-phantom/` |
| Windows | `%LocalAppData%\t2c-phantom\` |
| Linux | `~/.cache/t2c-phantom/` |

- `peer/identity.json` : ton identité. **Ne la partage pas.**
- `peer/hosted/` : tes copies de bénévole (chiffrées) ; `peer/hosting.json` : tes choix.
- `peer/relays.json` : les relais appris dans la liste signée.
- `peer/sites/` : les sites vérifiés, prêts à être servis.
- `asso-ca/` : le certificat local `.t2c`.

La variable `T2C_DATA_DIR` permet de changer ce dossier.

---

## 12. Dépannage

| Symptôme | Solution |
|---|---|
| « Connexion… » qui ne passe pas à « Connecté » | Vérifie ta connexion Internet. L'application essaie tour à tour tous les relais connus. En ligne de commande : `t2c doctor`. |
| Le navigateur affiche « non sécurisé » sur un `.t2c` | **Réglages** → **Refaire l'assistant**, puis ferme complètement et rouvre le navigateur. |
| « … n'est pas dans le catalogue signé » | Ce site n'existe pas ou plus : seul le catalogue officiel est servi. |
| Un site non `.t2c` ne charge plus | Quitte et relance l'application (le PAC est remis). En dernier recours : `t2c unsetup`. |
| Le badge « Anonyme » n'apparaît pas | Le réseau ne compte pour l'instant qu'un relais : le mode anonyme s'activera tout seul dès le deuxième. |
| Port 8080 déjà utilisé | Lance l'application avec `T2C_PROXY_PORT=8081`, ou `t2c start -proxy 8081`. |
| Linux : « installation automatique impossible » | `sudo apt install libnss3-tools`, puis **Refaire l'assistant**. |

---

## 13. Désinstaller

1. Quitte l'application normalement : le proxy d'origine est remis. En ligne de commande : `t2c unsetup`.
2. Supprime l'application et le dossier de données ([section 11](#11-données-locales)).
3. Facultatif : retire le certificat **« tutodecode CA »** :
   - macOS : Trousseau d'accès → session → Certificats ;
   - Windows : `certmgr.msc` → Autorités de certification racines de confiance ;
   - Firefox : Réglages → Certificats → Autorités.

---

## 14. Limites connues

- **Le point d'entrée voit ton IP.** Toute connexion révèle l'adresse de celui qui l'ouvre ; c'est Cloudflare (ou le relais principal tant que le domaine n'est pas actif). Si même lui ne doit pas la connaître, utilise un VPN ou Tor avant T2C Phantom.
- **L'anonymat suppose des relais indépendants.** Il repose sur le fait que les relais d'un circuit ne s'entendent pas entre eux, comme pour Tor. Des relais tenus par un même opérateur ne protègent pas contre cet opérateur. Les tailles fixes, le trafic leurre et les délais rendent la corrélation bien plus difficile, sans la rendre impossible pour un adversaire qui observe à la fois l'entrée et la sortie.
- **Le bénévolat n'est pas anonyme vis-à-vis de son relais**, qui sait quels sites tu héberges.
- **La bande passante des relais** est la limite du réseau : tout le trafic entre membres passe par eux.
- **Applications non signées** par Apple et Microsoft : un avertissement s'affiche au premier lancement.
- **Pas de mise à jour automatique** pour l'instant : installe les nouvelles versions depuis les Releases.
- Un **poste compromis** (logiciel malveillant) est hors de portée de toute protection réseau.

---

## 15. Questions fréquentes

**Est-ce que T2C Phantom change toute ma navigation ?**
Non. Seules les adresses `.t2c` passent par lui ; tout le reste part directement, comme avant.

**Un bénévole peut-il me servir un site modifié ou un virus ?**
Non. Chaque morceau est vérifié contre l'empreinte signée par TUTODECODE. Un morceau faux est rejeté, et le bénévole est banni de ton côté puis contrôlé par le réseau.

**Est-ce que les autres membres voient mon adresse IP ?**
Non. Personne ne se connecte directement à toi : tout passe par les relais, et les membres ne connaissent que des clés pseudonymes.

**Héberger des sites, c'est risqué pour moi ?**
Tu n'héberges que les sites que tu choisis, signés par TUTODECODE. Les copies sont chiffrées sur ton disque, la place et le débit sont plafonnés, et les autres membres ne connaissent que ta clé pseudonyme.

**Que se passe-t-il si je perds mon identité ?**
Rien de grave : l'application en crée une nouvelle. Tu perds seulement ton score de bénévole.

**Comment signaler un problème ?**
Bug ou question : les issues du dépôt ou `contact@tutodecode.org`. Faille de sécurité : en privé, voir [SECURITY.md](../SECURITY.md).
