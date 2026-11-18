# T2C Phantom — vue d'ensemble

Ce document décrit T2C Phantom en entier : ce que c'est, de quoi il est fait, comment chaque pièce fonctionne, ce qui le protège, et ce qu'il deviendra après le chantier en cours. Pour l'utilisation pas à pas, voir le [Guide complet](GUIDE.md).

---

## 1. En une phrase

**T2C Phantom est un réseau pair à pair qui sert les sites et les cours de TUTODECODE sans serveur central à abattre, sans qu'on puisse les falsifier, et sans que quiconque puisse savoir qui consulte quoi.**

Trois promesses, qui ne dépendent pas de la bonne foi des machines du réseau :

1. **Authenticité** : tout contenu est signé par la clé de TUTODECODE et vérifié par l'application de chaque utilisateur.
2. **Anonymat** : ton adresse IP n'est jamais reliée à ce que tu consultes.
3. **Résilience** : les sites sont copiés chez des bénévoles ; plus il y a de membres, plus le réseau est rapide et difficile à couper.

---

## 2. Les pièces du réseau

```
  Éditeur (poste hors ligne)            Utilisateurs
  ┌─────────────────────────┐          ┌───────────────────────────┐
  │ clé racine chiffrée      │          │ T2C Phantom (app/CLI)     │
  │ publie : sites, cours,   │          │  proxy local .t2c         │
  │ liste des relais         │          │  vérifie toutes signatures│
  └───────────┬─────────────┘          └─────────────┬─────────────┘
              │ paquet signé                           │ WebSocket TLS
              ▼                                        ▼
  ┌─────────────────────────┐   réseau   ┌───────────────────────────┐
  │ Serveur d'origine        │◀── privé ──│ Relais (fédération)       │
  │ stocke et distribue,     │  requêtes  │ entrée, milieu, sortie des│
  │ aucune clé de publication│  signées   │ circuits ; annuaire des   │
  └─────────────────────────┘            │ bénévoles                  │
                                          └─────────────┬─────────────┘
                                                        │
                                          ┌─────────────▼─────────────┐
                                          │ Bénévoles                  │
                                          │ copies chiffrées des sites │
                                          │ servent des morceaux       │
                                          └───────────────────────────┘
```

| Pièce | Rôle | Ce qu'elle détient |
|---|---|---|
| **Poste de l'éditeur** | crée et signe tout ce qui fait foi | la **clé racine**, chiffrée par phrase de passe, jamais en ligne |
| **Serveur d'origine** | stocke les sites, les distribue au réseau, tient le score des bénévoles | une clé de service limitée (liste des bénévoles, tickets) ; **aucune clé capable de signer un site** |
| **Relais** | point d'entrée des utilisateurs, circuits anonymes, annuaire des bénévoles, fédération entre relais | sa propre clé ; **aucun contenu en clair**, aucune clé de publication |
| **Bénévoles** | hébergent des copies chiffrées et servent des morceaux aux autres | des copies vérifiées des sites qu'ils ont choisis |
| **Application T2C Phantom** | ouvre les sites `.t2c` dans ton navigateur, vérifie tout | ton identité pseudonyme (clé aléatoire) |
| **T2DECODE** | application d'apprentissage ; peut recevoir les cours par T2C Phantom | la clé **publique** de TUTODECODE, pour vérifier `cours.tdc` |

### Les programmes

| Programme | Pour qui | Ce qu'il fait |
|---|---|---|
| **T2C Phantom** (macOS, Windows, Linux) | tout le monde | application de bureau : assistant de premier lancement, carnet de sites, bénévolat, connexion/déconnexion |
| **`t2c`** | serveurs, scripts | la même chose en ligne de commande (`t2c start`, `t2c doctor`…) |
| **`t2c-peer`** | tests, bénévoles sans interface | un membre minimal du réseau |
| **`t2c-relay`** | opérateurs de relais | le relais |
| **`t2c-serv`** | l'éditeur et le serveur d'origine | publier, signer, exporter/importer, servir |
| **`t2c-forge`** | conteneurs de fabrication | fabrique `repo.t2c` à partir des dépôts de code |

### Les sites officiels

| Adresse | Contenu |
|---|---|
| `tutodecode.t2c` | portail : la liste de tous les sites, lue en direct dans le catalogue vérifié par ton application |
| `cours.t2c` | les cours : un fichier unique **`cours.tdc`** signé, chaque cours seul en `.tdc`, une page par cours |
| `repo.t2c` | le code source : lisible en ligne, en `.zip`, ou clonable avec `git` à travers T2C |

`.t2c` désigne un **site** ; `.tdc` désigne un **fichier de cours** (TUTODECODE Script, créé avec TDC-SDK et TDC-Studio, lu par T2DECODE).

---

## 3. La confiance : une seule clé

Toute la sécurité part de la **clé racine** de TUTODECODE.

- Sa partie **publique** est gravée dans T2C Phantom et dans T2DECODE.
- Sa partie **privée** reste sur le poste de l'éditeur, **hors ligne**, **chiffrée par une phrase de passe** (scrypt, puis AES-256-GCM).

Ce qu'elle signe :

| Document | Contenu | Vérifié par |
|---|---|---|
| **Catalogue** | la liste des sites et l'empreinte de chacun | l'application, les relais, le serveur |
| **Manifestes** | l'empreinte de chaque morceau de 256 Ko d'un site | l'application, les bénévoles, les relais |
| **Liste des relais** | les relais officiels et leur clé | l'application, les relais |
| **`cours.tdc`** | tous les cours | T2DECODE lui-même |

Délégations limitées, pour ce qui doit tourner sans l'éditeur :

- le **serveur** reçoit une clé de service qui ne peut signer que la liste des bénévoles et les tickets d'hébergement (180 jours au plus) ;
- un **conteneur de fabrication** peut recevoir une clé qui ne vaut que pour **un seul site** (par exemple `repo.t2c`).

Conséquence : **un serveur, un relais, un bénévole, un routeur ou un conteneur piraté ne peut pas faire accepter un faux site, un faux cours ni un faux relais**. Une ancienne version renvoyée par un attaquant est aussi refusée : chaque document porte un numéro de version croissant, que l'application retient.

---

## 4. Comment ça marche, étape par étape

### 4.1 Publier (éditeur)

1. Sur son poste, l'éditeur prépare les sites et `cours.tdc`.
2. Une commande signe le tout avec la clé racine (la phrase de passe est demandée une fois).
3. Un **paquet** n'envoie que ce qui a changé au serveur d'origine, qui revérifie chaque signature et chaque empreinte avant de l'accepter.

### 4.2 Rejoindre le réseau (utilisateur)

1. Au premier lancement, l'assistant installe un **certificat limité aux noms `.t2c`**, règle le navigateur (fichier PAC : seuls les `.t2c` passent par T2C Phantom) et crée l'**identité** : une clé aléatoire, protégée par une preuve de travail. Pas de compte, pas d'e-mail.
2. L'application se connecte à un relais, récupère le **catalogue** et la **liste des relais**, et vérifie leurs signatures.

### 4.3 Ouvrir une page

1. Le navigateur demande `https://tutodecode.t2c/` ; le proxy local de T2C Phantom répond.
2. L'application lit la **table des matières** du site (un site est une archive zip), puis **seulement les morceaux** qui contiennent la page demandée, et quelques suivants à l'avance.
3. Ces morceaux viennent des **bénévoles** les mieux notés, par des sessions chiffrées de bout en bout. Sinon, ils viennent du serveur d'origine, via le relais.
4. Chaque morceau est vérifié contre le manifeste signé. Un faux morceau est rejeté, son expéditeur banni et signalé.
5. La page s'affiche. Les morceaux restent en cache : la visite suivante est instantanée. Les gros fichiers arrivent au fil de l'eau, avec une barre de progression.

**Le premier chargement peut prendre quelques secondes : c'est le prix de l'anonymat.**

### 4.4 Rester anonyme (routage en oignon)

- Tes demandes traversent un **circuit** de plusieurs relais, chiffré une fois par relais. Chaque relais ne retire qu'une couche.
- Le **relais d'entrée** voit ton adresse IP, mais pas ce que tu demandes.
- Le **relais de sortie** voit ce qui est demandé, mais ni ton adresse ni ton identité : seulement un **pseudonyme jetable**, changé toutes les 10 minutes.
- Contre l'analyse de trafic : cellules de **4 tailles fixes**, **trafic leurre** permanent, **délais aléatoires** à la sortie.
- **Échec fermé** : si un circuit tombe, rien ne part en clair ; l'application en reconstruit un.

### 4.5 Être bénévole

- Dans l'application, on coche les sites qu'on accepte d'héberger, avec une limite d'espace et de débit.
- Les copies sont **chiffrées sur le disque** et revérifiées à chaque envoi.
- Le réseau contrôle régulièrement que le bénévole possède vraiment ce qu'il annonce (défis de stockage). Les tricheurs perdent leur score et sont bannis.
- Les autres membres ne voient de lui qu'une clé pseudonyme, jamais son adresse : tout passe par un relais.

### 4.6 Recevoir les cours dans T2DECODE

- Option « Cours depuis T2C Phantom » dans les réglages de T2DECODE.
- Au démarrage, T2DECODE regarde par T2C Phantom si une version plus récente de `cours.tdc` existe.
- Il **vérifie lui-même** l'empreinte et la signature de TUTODECODE, refuse un fichier faux ou plus ancien, installe les cours et propose de redémarrer.
- Sans T2C Phantom lancé, il ne fait rien. Aucune requête vers GitHub ni Internet.

---

## 5. Qui voit quoi

| | Ton adresse IP | Ce que tu consultes | Le contenu des échanges entre membres |
|---|---|---|---|
| **Point d'entrée** (Cloudflare ou relais d'entrée) | oui | non en mode anonyme | non (chiffré) |
| **Relais du milieu** | non | non | non |
| **Relais de sortie** | non | oui, sous un pseudonyme jetable | non |
| **Bénévoles** | non | seulement les morceaux qu'ils fournissent, à un pseudonyme | — |
| **Serveur d'origine** | non | non | — |

Le nom `.t2c` ne circule jamais sur Internet : le proxy local l'intercepte, et la connexion part vers le **domaine du relais**.

## 6. Si un maillon est piraté

| Piraté | Ce qu'il peut faire | Ce qu'il ne peut pas faire |
|---|---|---|
| **Relais** | couper ou ralentir ; voir les adresses IP de ceux qui entrent par lui | faux contenu ; savoir **qui** lit **quoi** (il faudrait aussi la sortie du même circuit) |
| **Bénévole** | envoyer de faux morceaux, aussitôt rejetés (il est banni) | modifier un site ; identifier qui demande |
| **Serveur d'origine** | cesser de servir ; mal noter des bénévoles | signer un site, un cours ou une liste de relais |
| **Conteneur de fabrication** | publier de fausses versions de **son seul site**, jusqu'à l'expiration de sa clé | toucher aux autres sites, au catalogue, aux relais |
| **Routeur ou réseau** | couper | lire ou modifier (tout est chiffré et signé) |
| **Clé racine** | **tout** : c'est le seul point critique | — |

D'où les protections de la clé racine : hors ligne, chiffrée, phrase de passe longue, poste à jour et chiffré (FileVault).

---

## 7. Technologies utilisées

| Domaine | Technologie |
|---|---|
| Langages | Go (réseau, relais, serveur, outils) ; Dart/Flutter (T2DECODE) |
| Application de bureau | Wails : interface web intégrée, aucune ressource distante |
| Identité, signatures | Ed25519 ; preuve de travail de 22 bits pour chaque identité |
| Clé racine | hors ligne, chiffrée : scrypt (N = 2¹⁷), AES-256-GCM |
| Chiffrement entre membres | Noise IK (X25519, ChaCha20-Poly1305, BLAKE2s) |
| Circuits anonymes | échange X25519 à clés éphémères, couches AES-256-GCM, cellules de taille fixe, trafic leurre, délais aléatoires |
| Intégrité | SHA-256 par morceau de 256 Ko ; catalogue, manifestes, liste des relais et cours signés |
| Sites | archive zip déterministe, lue à la demande ; envoi au fil de l'eau des gros fichiers |
| Cours | TUTODECODE Script (`.tdc`) : un seul fichier signé, lu par T2DECODE |
| Copies des bénévoles | AES-256-GCM sur le disque, revérifiées à chaque lecture |
| Transport | WebSocket sur TLS 1.2/1.3, sous-protocole `t2c.v2` obligatoire |
| Relais ↔ serveur | réseau privé chiffré ; chaque requête signée, non rejouable |
| Navigateur | certificat local limité aux `.t2c`, fichier PAC |
| Sites officiels | charte TUTODECODE (police Aileron, domaine public), tout servi par le site lui-même |
| Fabrication de `repo.t2c` | copie locale des dépôts (`git`), mise à jour toutes les 2 minutes, `git clone` en HTTP simple |
| Développement | CI locale et pentest sur le poste du développeur, lancés avant chaque envoi (hook git), sans service en ligne ; GitHub Actions pour les versions publiées |

---

## 8. Ce que T2C Phantom deviendra (chantier en cours)

Testé en interne d'abord ; publié quand tout sera validé, avec le domaine `t2cphantom.eu.org`.

### 8.1 Des relais bénévoles, comme Tor

- **N'importe qui avec un serveur peut faire tourner un relais**, en une commande, sans rien demander à l'association.
- Les relais **s'annoncent tout seuls**. Le réseau les teste automatiquement (joignables, protocole conforme, preuve de travail, 24 h en ligne), les inscrit dans un **annuaire automatique**, et les retire tout seuls s'ils tombent ou trichent.
- Pour un bénévole chez lui : seulement derrière un VPN qui permet de recevoir des connexions (redirection de port) ou derrière son propre tunnel. Son adresse personnelle n'est jamais publiée.
- Les relais de l'association restent dans la **liste signée par la clé racine** : c'est le socle de confiance.

### 8.2 Des circuits choisis comme Tor

- Jamais deux relais du même réseau dans un circuit.
- Un nouveau relais n'est presque pas utilisé au début, puis de plus en plus sur plusieurs semaines. Espionner demanderait des centaines de serveurs pendant des mois.
- Deux modes dans l'application :

| Mode | Circuit | Effet |
|---|---|---|
| **Partagé** (par défaut) | relais bénévoles d'opérateurs différents | réseau rapide et sans goulot ; anonymat très fort, comme Tor |
| **Maximal** | au moins un relais de l'association | anonymat appuyé sur les relais de l'association ; plus lent |

Dans les deux modes, **personne ne peut modifier un contenu**.

Un argument que Tor n'a pas : **un relais T2C ne transporte que du contenu signé par TUTODECODE**, jamais « tout Internet ». Faire tourner un relais ne fait donc courir aucun des risques juridiques d'un relais de sortie Tor.

### 8.3 Cloudflare devant le relais principal

- Les utilisateurs entrent par Cloudflare ; le relais principal ne voit plus leurs adresses IP.
- Le relais refuse toute connexion qui ne vient pas de Cloudflare (certificat client).
- Sans domaine : un Worker gratuit sur `workers.dev`. Avec `t2cphantom.eu.org` : directement sur le domaine.

### 8.4 Au bout du chantier

T2C Phantom sera **un réseau de diffusion du savoir de TUTODECODE** :

- **porté par ses membres** : bénévoles qui hébergent, relais bénévoles qui transportent ;
- **sans point unique** à abattre ni à acheter ;
- **inaltérable** : tout est signé par une seule clé, hors ligne ;
- **anonyme par défaut** ;
- avec ses propres sites (portail, cours, code source) et l'intégration directe des cours dans T2DECODE.

---

## 9. Limites honnêtes

- **Le point d'entrée voit ton adresse IP** (Cloudflare ou un relais), comme toute connexion. Pour que même lui ne la voie pas : un VPN ou Tor avant T2C Phantom.
- **L'anonymat repose sur la diversité des relais** : un adversaire qui contrôlerait à la fois l'entrée et la sortie d'un circuit pourrait corréler. Les règles de choix des circuits rendent cela coûteux, pas impossible, comme pour Tor.
- **Tout maillon peut couper le service** ; seule la multiplication des relais et des bénévoles y répond.
- **La clé racine est le point critique** : volée et cassée, elle permettrait de tout signer. Il n'existe pas encore de révocation : il faudrait publier de nouvelles applications.
- **Plus lent qu'un site classique** au premier chargement : c'est le prix de l'anonymat et de la vérification.

## 10. Idées pour la suite

- Mode « relais bénévole » activable en un clic dans l'application.
- Releases signées par la clé racine, vérifiables depuis `tutodecode.t2c`.
- Clé racine sur une clé physique (YubiKey) : impossible à copier.
- Mise à jour des sites par différence : ne renvoyer que les morceaux modifiés.
- Recherche dans les sites, favoris hors ligne.
- Tableau de remerciement des bénévoles et relais (« ton relais a servi 1 200 visiteurs ce mois-ci »).
- Applications signées par Apple et Microsoft, mises à jour automatiques.

## 11. Petit glossaire

| Terme | Sens |
|---|---|
| **`.t2c`** | adresse d'un site du réseau |
| **`.tdc`** | fichier de cours (TUTODECODE Script) |
| **Clé racine** | la clé de TUTODECODE qui signe tout ce qui fait foi |
| **Catalogue** | la liste signée des sites |
| **Manifeste** | la liste signée des empreintes des morceaux d'un site |
| **Morceau** | bloc de 256 Ko d'un site, vérifiable seul |
| **Relais** | machine qui fait entrer les membres dans le réseau et transporte les circuits |
| **Circuit** | chemin chiffré en couches à travers plusieurs relais |
| **Bénévole** | membre qui héberge des copies de sites |
| **Pseudonyme jetable** | identité temporaire utilisée dans un circuit |
| **Délégation** | autorisation limitée donnée par la clé racine à une autre clé |
