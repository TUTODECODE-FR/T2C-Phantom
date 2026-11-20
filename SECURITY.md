# Politique de sécurité

La transparence et la confiance sont au cœur du réseau T2C Phantom. Ce dépôt publie le client complet pour qu'il puisse être audité.

## Modèle de menaces

Objectif assumé : **protéger l'adresse IP** des membres et des bénévoles, et **garantir l'authenticité** des sites. T2C Phantom ne promet pas un anonymat absolu : il réduit ce que chaque acteur peut voir, sans le supprimer. Détails : [docs/FONCTIONNEMENT.md](docs/FONCTIONNEMENT.md).

### Qui voit quoi (état actuel du réseau)

Deux modes dans l'application (Réglages) : **Partagé** (par défaut) et **Maximal**. « Pseudonyme » = clé aléatoire, sans nom ni compte ; « jetable » = clé neuve, sans lien avec ton identité habituelle.

| Acteur | Ton adresse IP | Les sites que tu demandes | Notes |
|---|---|---|---|
| **Relais officiels de l'association** (VPS, relais maison), utilisés comme **entrée** | **oui** | non (cellules chiffrées en couches) | voient aussi ton pseudonyme de connexion, et donc que tu utilises T2C |
| **Relais officiel utilisé comme sortie** (toujours en Maximal ; en Partagé seulement à défaut de sortie bénévole) | non | **oui**, sous un pseudonyme jetable | |
| **Aquilenet** (hébergeur du VPS) | **oui** (au niveau réseau, comme tout hébergeur) | non (TLS) | ne fait que transporter |
| **Tailscale** (Tailscale Funnel, porte d'entrée du relais maison) | **oui** | non (le TLS va jusqu'au relais) | voit aussi le nom `*.ts.net` demandé ; société hors UE |
| **sslip.io** (service DNS du nom `185-233-100-85.sslip.io`) | celle de ton résolveur DNS (parfois une partie de la tienne) | non | apprend qu'un appareil cherche le relais, donc utilise T2C |
| **Cloudflare** (prévu, **pas encore en place**) | oui | non en mode anonyme ; la taille et le rythme des cellules | remplacera l'exposition directe du VPS |
| **Garde d'entrée bénévole** (Partagé seulement, activée par défaut, désactivable : « Entrer par des relais bénévoles » ; jamais en Maximal ni quand tu héberges) | **oui** | non | une seule garde, ancienne d'au moins 30 jours, gardée 60 jours ; identité jetable |
| **Relais bénévole du milieu** (relais caché d'un autre membre) | non | non | ne voit que des cellules chiffrées |
| **Sortie bénévole** (Partagé : relais caché d'un membre qui accepte d'être sortie) | non | **oui**, sous un pseudonyme jetable | ne voit jamais ton adresse |
| **Hébergeurs** (bénévoles qui stockent les sites) | non | les morceaux qu'ils fournissent, à un pseudonyme | contenu chiffré de bout en bout (Noise) |
| **Serveur d'origine** | non | non | connaît les clés des bénévoles hébergeurs |
| **Ton fournisseur d'accès / réseau local** | — | non | voit que tu te connectes aux relais (donc que tu utilises T2C) |

Dans tous les modes, aucun relais n'est censé connaître **à la fois** ton adresse et tes demandes. Mais un adversaire qui tient (ou observe) l'entrée **et** la sortie d'un même circuit peut les relier par le moment et le volume du trafic ; en Maximal, l'association tient les deux bouts. Les cellules de taille fixe, le trafic leurre et les délais aléatoires rendent cette corrélation plus difficile, pas impossible.

Hors mode anonyme (un seul relais joignable), le relais d'entrée voit ton adresse **et** tes demandes.

### Menaces et protections

| Menace | Protection |
|---|---|
| Un bénévole modifie un site (virus, contenu altéré) | Chaque morceau est vérifié contre l'empreinte signée par TUTODECODE ; morceau rejeté, bénévole banni, signalé et contrôlé |
| Le relais lit ou modifie les échanges entre pairs | Chiffrement de bout en bout Noise IK ; le relais n'a pas la clé de signature et ne peut pas falsifier un site |
| L'origine récupère ton IP | Elle ne la reçoit pas et refuse toute requête portant un en-tête d'IP. Le relais d'entrée, lui, la voit (voir le tableau) |
| Un vieux catalogue est rejoué | Version signée strictement croissante, retour en arrière refusé |
| Le serveur d'origine est piraté | La clé racine n'y est pas : elle reste hors ligne chez l'éditeur. Le serveur n'a qu'une clé de service déléguée (180 jours au plus) : tickets, liste des bénévoles, consignes réseau. La racine peut la **révoquer** (liste des relais signée) : ses listes, tickets et consignes sont alors refusés partout. Une consigne signée par cette clé n'exclut pas plus de 64 relais ; au-delà, la racine doit signer |
| Un relais officiel piraté fait exclure des bénévoles honnêtes | Le pilote automatique du serveur n'exclut un bénévole que s'il est signalé par **deux** relais officiels distincts (un seul quand un seul envoie des rapports), et jamais plus de 10 % de l'annuaire : au-delà, un humain décide |
| Le relais (VPS) est piraté | Il ne détient aucun secret de l'origine : chaque requête vers l'origine est signée par sa propre clé, horodatée et non rejouable. Il ne peut ni falsifier un site ni joindre le réseau de l'origine ; il voit en revanche les adresses IP de ceux qui entrent par lui |
| Un relais relie ton IP à ce que tu consultes | Mode anonyme : circuit en oignon à travers plusieurs relais ; l'entrée connaît l'IP sans lire les demandes, la sortie lit les demandes sans connaître l'IP (pseudonyme jetable) |
| Un observateur déduit l'activité de la taille ou du rythme du trafic | Cellules de 4 tailles fixes (remplissage aléatoire), cellules leurres, délai aléatoire au relais de sortie. Protection partielle |
| Requêtes forgées ou non conformes sur le relais | Seule une ouverture WebSocket annonçant le sous-protocole `t2c.v2` est acceptée (sinon 404 muet) ; identification exigée en quelques secondes, refus générique, messages hors protocole sanctionnés puis bannissement |
| Flood ou création massive d'identités | Limites par clé, preuve de travail pour créer une identité ou un relais bénévole, montée progressive des nouveaux relais |
| Une variable d'environnement remplace la clé racine | Ignorée : `T2C_ROOT_PUBKEY` ne sert qu'aux versions de développement sans clé intégrée ; la release vérifie que la clé compilée est celle versionnée dans le dépôt |
| Un site `.t2c` fait charger au navigateur des ressources du web ouvert (qui verraient ton IP) | Le proxy impose à chaque réponse `.t2c` une politique de contenu (CSP) qui n'autorise que le site lui-même et les autres `.t2c`, ainsi que `Referrer-Policy: no-referrer` et `X-Content-Type-Options: nosniff` |
| Une page du web ouvert sonde le proxy (savoir si tu es membre, deviner ton historique au temps de réponse, surcharger le réseau) | Le proxy refuse aussitôt (403, avant toute recherche) les requêtes venues d'une page hors `.t2c` (Origin, Referer, `Sec-Fetch-Site`). Complet en `https://` ; en `http://`, le navigateur n'envoie pas ces indices, une page peut donc encore sonder |
| Une page web pilote l'application (tableau de bord dans le navigateur) | `/api/*` refusé depuis une autre origine ou un autre nom d'hôte (rebinding DNS), actions en POST seulement, interface jamais affichée dans un cadre |
| Le certificat local est volé | Contraintes inscrites dans le certificat : noms `.t2c` seulement, aucune adresse IP, usage « serveur web » seulement. Un navigateur à jour refuse tout certificat de cette CA pour un autre site ; un logiciel ancien qui ignore ces contraintes n'est pas protégé. La clé n'est lisible que par ton compte (0600) |
| Le proxy capte toute la navigation | Le fichier PAC n'envoie que les `*.t2c` ; tout le reste est en connexion directe. Un PAC laissé par un arrêt brutal est retiré au lancement suivant, et le PAC n'est activé que si le proxy a bien obtenu son port |

**Limites connues (à lire)**
- **WebRTC** : un script d'un site `.t2c` peut demander ton adresse IP par WebRTC (STUN), qui ne passe ni par le proxy ni par la CSP. T2C ne modifie pas les réglages de tes profils de navigateur habituels (cela casserait tes appels vidéo). Pour t'en protéger, utilise un profil réservé aux `.t2c` avec WebRTC désactivé (Firefox : `about:config` → `media.peerconnection.enabled` = `false`).
- **Navigation hors `.t2c`** : un site peut toujours t'envoyer, par un lien ou une redirection de la page entière, vers une adresse du web ouvert ; cette page verra alors ton IP, comme n'importe quel lien.
- **Firefox** : le réglage `security.enterprise_roots.enabled` n'est écrit que si `certutil` manque, à l'étape « certificat » que tu lances toi-même, jamais dans LibreWolf ; il est retiré à la désactivation.
- **Le relais d'entrée** (officiel, garde bénévole, ou leur hébergeur / tunnel) voit ton IP. Si personne d'autre que toi ne doit la connaître, passe par un VPN ou Tor *avant* T2C Phantom (logiciels tiers, sous ta responsabilité).
- Un poste compromis (logiciel malveillant, keylogger) est hors de portée de toute protection réseau.

### Anciennes clés de la v1 : compromises

L'historique public du dépôt contient six clés privées Ed25519 de la version 1 (format libp2p, fichiers `data/client.txt`, `data/serveur.txt`, `data/relay.txt`, `data/nodeA.txt`, `data/nodeB.txt` et `data/user_key.txt`, retirés depuis). **Elles sont publiques, donc compromises.** Elles ne sont utilisées par aucune version actuelle (ni l'application, ni les relais, ni le serveur, ni la clé racine) et ne doivent jamais l'être. Un document signé par l'une d'elles n'a aucune valeur.

Le code du relais et de l'origine n'est pas publié. La sécurité des utilisateurs n'en dépend pas : tout ce qui protège ta vie privée et l'intégrité des sites s'exécute dans le client public de ce dépôt.

---

## Nomenclature logicielle (SBOM)

La liste complète des dépendances logicielles (Software Bill of Materials) est disponible dans :
- `go.mod` — dépendances déclarées.
- `go.sum` — versions exactes résolues et hashes d'intégrité cryptographique.

## Divulgation responsable

N'ouvrez **pas** d'issues GitHub publiques pour signaler des vulnérabilités critiques (escalade de privilèges, désanonymisation, ou exécution de code à distance).
Utilisez l'un de ces canaux privés :
- Avis de sécurité privé GitHub (Security Advisories).
- Email direct : `contact@tutodecode.org`.

Lors de votre signalement, veuillez inclure :
- La version ou le commit affecté(e).
- Les étapes précises (Proof of Concept) pour reproduire la faille.
- L'impact potentiel sur le chiffrement ou la confidentialité.

## Objectifs de réponse

L'Association TUTODECODE s'engage sur les délais suivants :
- **Accusé de réception** : dans les 72 heures.
- **Évaluation et Tri** : dans les 7 jours ouvrables.
- **Remédiation** : Après un correctif, une nouvelle release est générée par GitHub Actions (avec `SHA256SUMS.txt`) et recopiée sur GitLab.
