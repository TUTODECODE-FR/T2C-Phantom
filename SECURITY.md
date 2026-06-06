# Politique de Sécurité (T2C-Phantom)

La transparence et la confiance sont au cœur du réseau T2C Phantom. Bien que ce dépôt soit sous licence stricte de propriété intellectuelle, une partie de son architecture est ouverte à l'audit.

## 🛡️ Modèle de Menaces (Threat Model)

Pour comprendre de quoi T2C Phantom vous protège, voici notre modèle de menace et nos choix de conception assumés :

### 1. Pare-feux Strict et Censure (DPI)
- **Le Risque :** Un réseau d'entreprise ou une dictature bloque tout trafic non standard (ex: bloque les VPNs, bloque SSH, bloque le P2P).
- **Notre Protection :** T2C Phantom encapsule son trafic dans des WebSockets Sécurisés (WSS) sur le port 443. Pour le pare-feu, cela ressemble à une simple navigation web HTTPS vers un site classique. L'évasion est totale.

### 2. Le Man-In-The-Middle (L'Administrateur Réseau / L'Hébergeur Cloud)
- **Le Risque :** L'administrateur de votre réseau force le déchiffrement SSL (Inspection SSL), ou l'hébergeur du Relais tente de lire vos paquets.
- **Notre Protection :** Le **Chiffrement Emboîté (Double Encryption)**. L'architecture utilise le protocole cryptographique *Noise* de `libp2p`. Même si l'hébergeur cloud supprime la couche HTTPS externe, les paquets internes restent chiffrés de bout en bout. Le Relais Cloud n'est qu'un "tuyau opaque" aveugle.

### 3. Empoisonnement du Réseau et DHT Sniffing (Sybil Attacks)
- **Le Risque :** Des pirates tentent de copier notre code pour créer des faux relais afin d'empoisonner le réseau (Attaques Sybil) ou d'enregistrer les requêtes de nos utilisateurs.
- **Notre Protection (Transparence Hybride) :** Nous avons fait le choix assumé de garder **le code source du Relais et l'architecture du serveur ("Le Sous-marin") strictement confidentiels**. En ne publiant pas les clés d'infrastructure, nous empêchons les acteurs malveillants de cloner facilement notre réseau pour l'espionner.

### 4. Confidentialité des Métadonnées (L'Hébergeur du Relais)
- **La Réalité Technique :** Notre Relais public est actuellement hébergé chez Microsoft Azure. Comme expliqué au point 2, Microsoft ne peut absolument **rien lire** de vos données. Cependant, techniquement, Microsoft (ou tout autre hébergeur) voit nécessairement l'adresse IP publique de la machine qui se connecte au Relais.
- **Le Remède Absolu (VPN / Tor) :** Si votre modèle de menace exige que même l'hébergeur du Relais ignore votre adresse IP physique, la seule protection absolue est d'utiliser un VPN ou le réseau Tor *avant* de vous connecter à T2C Phantom.
- **⚠️ Clause de non-responsabilité :** L'association TUTODECODE garantit l'inviolabilité absolue de son tunnel P2P décentralisé. En revanche, nous ne garantissons ni la fiabilité ni la sécurité des logiciels VPN tiers ou de Tor que vous choisiriez d'utiliser en surcouche.

---

## 📦 Nomenclature Logicielle (SBOM)

La liste complète des dépendances logicielles (Software Bill of Materials) est disponible dans :
- `go.mod` — dépendances déclarées.
- `go.sum` — versions exactes résolues et hashes d'intégrité cryptographique.

## 🚨 Divulgation Responsable (Vulnerability Reporting)

N'ouvrez **pas** d'issues GitHub publiques pour signaler des vulnérabilités critiques (escalade de privilèges, désanonymisation, ou exécution de code à distance).
Utilisez l'un de ces canaux privés :
- Avis de sécurité privé GitHub (Security Advisories).
- Email direct : `contact@tutodecode.org`.

Lors de votre signalement, veuillez inclure :
- La version ou le commit affecté(e).
- Les étapes précises (Proof of Concept) pour reproduire la faille.
- L'impact potentiel sur le chiffrement ou la confidentialité.

## ⏱️ Objectifs de Réponse

L'Association TUTODECODE s'engage sur les délais suivants :
- **Accusé de réception** : dans les 72 heures.
- **Évaluation et Tri** : dans les 7 jours ouvrables.
- **Remédiation** : Après un correctif, une nouvelle release est générée, signée cryptographiquement, et les nœuds du réseau sont invités à se mettre à jour.
