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
| **Relais officiels** | point d'entrée des utilisateurs, circuits anonymes, annuaire des bénévoles, fédération entre relais | sa propre clé ; **aucun contenu en clair**, aucune clé de publication |
| **Relais cachés** (chaque application) | transportent les circuits des autres membres, au milieu, sans rien voir ; aucune adresse publiée | une clé de relais distincte de l'identité du membre |
| **Bénévoles** | hébergent des copies chiffrées et servent des morceaux aux autres | des copies vérifiées des sites qu'ils ont choisis |
| **Application T2C Phantom** | ouvre les sites `.t2c` dans ton navigateur, vérifie tout | ton identité pseudonyme (clé aléatoire) |
| **T2DECODE** | application d'apprentissage ; peut recevoir les cours par T2C Phantom | la clé **publique** de TUTODECODE, pour vérifier `cours.tdc` |

### Les programmes

| Programme | Pour qui | Ce qu'il fait |
|---|---|---|
| **T2C Phantom** (macOS, Windows, Linux) | tout le monde | application de bureau : assistant de premier lancement, carnet de sites, bénévolat, connexion/déconnexion |
| **`t2c`** | serveurs, scripts | la même chose en ligne de commande (`t2c start`, `t2c doctor`…) |
| **`t2c-peer`** | tests, bénévoles sans interface | un membre minimal du réseau |
| **`t2c-relay`** | ceux qui ont un serveur et veulent soutenir le réseau | le relais complet (dépôt séparé `T2C-Phantom-relay`, dont le cœur est aussi intégré à chaque application) |
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

- Tes demandes traversent un **circuit** de plusieurs relais, chiffré une fois par relais. Chaque relais ne retire qu'une couche. Quand c'est possible, le relais du **milieu** est le relais caché d'un autre membre : il ne voit ni qui tu es ni ce que tu demandes.
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
| **Relais du milieu** (souvent le relais caché d'un autre membre) | non | non | non |
| **Relais de sortie** | non | oui, sous un pseudonyme jetable | non |
| **Bénévoles** | non | seulement les morceaux qu'ils fournissent, à un pseudonyme | — |
| **Serveur d'origine** | non | non | — |

Le nom `.t2c` ne circule jamais sur Internet : le proxy local l'intercepte, et la connexion part vers le **domaine du relais**.

## 6. Si un maillon est piraté

| Piraté | Ce qu'il peut faire | Ce qu'il ne peut pas faire |
|---|---|---|
| **Relais** | couper ou ralentir ; voir les adresses IP de ceux qui entrent par lui | faux contenu ; savoir **qui** lit **quoi** (il faudrait aussi la sortie du même circuit) |
| **Bénévole** | envoyer de faux morceaux, aussitôt rejetés (il est banni) | modifier un site ; identifier qui demande |
| **Relais caché d'un membre** | ralentir ou couper les circuits qui passent par lui (l'application en rebâtit un autre) | lire quoi que ce soit, savoir qui ou quoi : il n'est jamais ni entrée ni sortie |
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

## 8. Soutenir le réseau

### 8.1 Le principe : chaque membre soutient l'anonymat de tous

Sans bénévoles, le réseau repose sur les seuls relais de l'association : tout fonctionne, mais tout le trafic passe par eux, et l'anonymat repose sur un petit nombre de relais. **Chaque personne qui a T2C Phantom renforce le réseau**, à trois niveaux, tous automatiques :

| Niveau | Qui | Ce que ça apporte | Ce que la personne risque |
|---|---|---|---|
| **Relais caché** | **chaque application**, activé par défaut (désactivable) | plus de relais différents dans les circuits : **la corrélation devient beaucoup plus difficile** | rien : ni adresse publiée, ni contenu lisible, débit plafonné (128 Ko/s par défaut) |
| **Bénévole hébergeur** | ceux qui cochent des sites | les sites sont servis par les membres, plus vite, même si un serveur tombe | rien : copies chiffrées, seulement du contenu signé |
| **Relais d'entrée** | ceux qui ont un serveur, ou un VPN qui laisse entrer les connexions | **soulage les relais de l'association** : les membres entrent aussi par eux | l'adresse du serveur ou du VPN est visible, jamais celle de leur domicile |

**Plus il y a de relais, mieux c'est pour tout le monde** :
- les circuits passent par des relais de personnes différentes : pour espionner, il faudrait contrôler l'entrée **et** la sortie d'un même circuit, ce qui devient d'autant plus improbable qu'il y a de relais ;
- le trafic se répartit : plus de goulot, plus de point unique à surcharger ou à couper ;
- un relais qui tombe passe inaperçu : l'application en choisit un autre.

Un argument que Tor n'a pas : **un relais T2C ne transporte que du contenu signé par TUTODECODE**, jamais « tout Internet ». Soutenir le réseau ne fait courir aucun des risques juridiques d'un relais de sortie Tor.

### 8.2 Le relais caché de chaque application (déjà en place)

- L'application n'écoute rien et ne publie aucune adresse : c'est elle qui se connecte aux relais officiels.
- Les relais officiels l'admettent **automatiquement** si sa clé a coûté une preuve de travail (24 bits : quelques secondes une seule fois), avec **au plus 8 relais par réseau** : en lancer des milliers depuis quelques machines ne sert à rien. Elle n'a que des droits réduits : transporter des cellules de circuit, rien d'autre (ni annuaire, ni accès au serveur, ni sortie).
- Elle ne sert qu'au **milieu** des circuits : entrée officielle → relais caché → sortie officielle. Elle ne voit ni qui est le membre (connu du relais d'entrée) ni ce qu'il demande (relais de sortie).
- Sa clé n'a **aucun lien** avec l'identité du membre, et elle n'est jamais utilisée dans ses propres circuits. Il porte au plus 64 circuits à la fois, avec un débit plafonné.
- Limite assumée : tant que les relais cachés ne sont reliés qu'aux relais officiels, leurs cellules passent deux fois par ceux-ci (entrée et sortie). Ils renforcent l'anonymat, pas la bande passante ; ce sont les relais publics bénévoles (8.4) qui soulagent.
- L'annuaire des relais cachés est signé par chaque relais officiel, pas par le serveur.

### 8.3 Des circuits choisis comme Tor (en place, testé en interne)

- Un relais bénévole peut aussi accepter d'être la **sortie** des circuits (réglage dans l'application, activé par défaut). Il voit alors les demandes, sous un pseudonyme jetable, jamais l'adresse du membre, connue du seul relais d'entrée de l'association.
- **Jamais deux relais du même réseau** dans un circuit : chaque relais officiel publie, pour chaque bénévole, une « famille » (empreinte salée de son réseau, renouvelée chaque jour), jamais son adresse.
- **Montée progressive** : un relais n'est proposé qu'après 24 heures, puis choisi de plus en plus souvent pendant 14 jours. Une armée de relais créés d'un coup ne pèse presque rien.
- Deux modes dans l'application :

| Mode | Circuit | Effet |
|---|---|---|
| **Partagé** (par défaut) | entrée de l'association ou **garde** bénévole → bénévole → bénévole d'une autre famille en sortie | la garde voit l'adresse IP, jamais ce qui est demandé ; la sortie voit les demandes, jamais l'adresse ; **aucun relais de l'association ne voit plus ce qui est demandé** |
| **Maximal** | entrée de l'association (jamais un bénévole) → bénévole → sortie de l'association | seuls les relais de l'association voient l'adresse ou les demandes |

- **Gardes d'entrée** (comme Tor) : un membre qui entre par un relais bénévole public (réglage « Entrer par des relais bénévoles (ils verront mon adresse IP) », activé par défaut en Partagé, jamais en Maximal ni pour un hébergeur) n'utilise qu'**une** garde, et un secours, choisies parmi les bénévoles présents depuis au moins 30 jours (ancienneté tenue par les relais officiels, remise à zéro après 48 h d'absence), gardées 60 jours ; changées seulement après des échecs répétés, une exclusion ou l'expiration. Identité jetable neuve à chaque connexion. L'annuaire des bénévoles est demandé par le circuit à une sortie officielle, jamais au relais d'entrée bénévole.
- **Un relais caché ne se connecte jamais à un bénévole** : il ne prolonge un circuit que vers les relais officiels auxquels il est déjà relié ; seuls les bénévoles publics se joignent entre eux.
- **Consignes en échec fermé** : quand une consigne expire, ses exclusions et son mode urgence restent appliqués (relais et applications) jusqu'à la suivante ; en urgence, les relais officiels refusent les bénévoles et les relais d'entrée bénévoles refusent les nouveaux membres.

### 8.4 Ce qui arrive ensuite (testé en interne, publié avec le domaine `t2cphantom.eu.org`)

1. **Les relais d'entrée volontaires**, en un clic dans l'application : « ma propre adresse » (l'application vérifie qu'elle est joignable et explique comment ouvrir le port) ou « un VPN avec redirection de port » (guide pas à pas, vérification automatique). Et pour ceux qui ont un serveur : `t2c-relay` en une commande. **C'est l'étape qui soulage vraiment les relais de l'association**, puisque les membres entrent aussi par eux.
2. **Cloudflare devant le relais principal** : le relais de l'association ne reçoit plus les adresses IP des membres (seul Cloudflare, en façade, les voit) et rejette toute connexion directe qui ne présente pas le certificat client de Cloudflare.

### 8.5 Mises à jour : automatiques, signées, progressives

- **Les relais se mettent à jour tout seuls** : chaque nouvelle version est publiée sur le réseau lui-même (site `relais.t2c`), **signée par une clé dédiée** de l'association, hors ligne. Un relais refuse toute version non signée ou plus ancienne que la sienne. Le déploiement est progressif : quelques relais d'abord, puis tous s'ils restent en bonne santé.
- **Les applications** reçoivent par le réseau un avis de version signé par TUTODECODE :
  - une **mise à jour ordinaire** est simplement signalée : pas besoin d'avoir toujours la dernière version, tout continue de fonctionner ;
  - une **mise à jour importante** (faille corrigée) est annoncée clairement, avec un bouton pour la télécharger ;
  - si une faille met **les autres membres** en danger, l'avis fixe une **version minimale** : en dessous, les relais refusent l'application, qui explique pourquoi et comment se mettre à jour.
- Les relais bénévoles peuvent choisir « mises à jour automatiques » (par défaut) ou « me prévenir seulement ». Le code est public, et chaque version peut être reconstruite à l'identique pour vérifier qu'elle correspond au code.

### 8.6 Au bout du chantier

T2C Phantom sera **un réseau de diffusion du savoir de TUTODECODE porté par ses membres** :

- chaque application soutient l'anonymat de tous, les bénévoles hébergent, les relais volontaires transportent ;
- **sans point unique** à abattre ni à acheter : si les relais de l'association tombent, ceux des membres prennent le relais ;
- **inaltérable** : tout est signé par une clé unique, hors ligne ;
- **anonyme par défaut**, et d'autant plus fort qu'il y a de membres ;
- **à jour tout seul**, sans jamais accepter une version non signée.

## 9. Limites honnêtes

- **Le point d'entrée voit ton adresse IP** (Cloudflare ou un relais), comme toute connexion. Pour que même lui ne la voie pas : un VPN ou Tor avant T2C Phantom.
- **L'anonymat repose sur la diversité des relais** : un adversaire qui contrôlerait à la fois l'entrée et la sortie d'un circuit pourrait corréler. Les relais cachés des membres et les règles de choix des circuits rendent cela de plus en plus coûteux à mesure que le réseau grandit, sans le rendre impossible, comme pour Tor.
- **Le relais caché utilise un peu de débit** chez chaque membre (plafonné, désactivable).
- **Tout maillon peut couper le service** ; seule la multiplication des relais et des bénévoles y répond.
- **La clé racine est le point critique** : volée et cassée, elle permettrait de tout signer. Il n'existe pas encore de révocation : il faudrait publier de nouvelles applications.
- **Plus lent qu'un site classique** au premier chargement : c'est le prix de l'anonymat et de la vérification.

## 10. Idées pour la suite

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
| **Relais caché** | relais intégré à chaque application : il transporte les circuits des autres au milieu, sans adresse publiée et sans rien voir |
| **Circuit** | chemin chiffré en couches à travers plusieurs relais |
| **Bénévole** | membre qui héberge des copies de sites |
| **Pseudonyme jetable** | identité temporaire utilisée dans un circuit |
| **Délégation** | autorisation limitée donnée par la clé racine à une autre clé |
