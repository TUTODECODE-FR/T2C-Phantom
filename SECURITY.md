# Politique de Sécurité (T2C-Phantom)

La transparence et la confiance sont au cœur du réseau T2C Phantom. Bien que ce dépôt soit sous licence stricte de propriété intellectuelle, son code est ouvert à l'audit.

## 🛡️ Modèle de Menaces (Threat Model)

Pour comprendre de quoi T2C Phantom vous protège, voici notre modèle de menace :

### 1. Pare-feux Strict et Censure (DPI)
- **Le Risque :** Un réseau d'entreprise ou une dictature bloque tout trafic non standard (ex: bloque les VPNs, bloque SSH, bloque le P2P).
- **Notre Protection :** T2C Phantom encapsule son trafic dans des WebSockets Sécurisés (WSS) sur le port 443. Pour le pare-feu, cela ressemble à une simple navigation web HTTPS vers un site classique. L'évasion est totale.

### 2. Le Man-In-The-Middle (L'Administrateur Réseau / L'Hébergeur Cloud)
- **Le Risque :** L'administrateur de votre réseau force le déchiffrement SSL (Inspection SSL), ou l'hébergeur du Relais (ex: Microsoft Azure) tente de lire vos paquets.
- **Notre Protection :** Le **Chiffrement Emboîté (Double Encryption)**. L'architecture utilise le protocole cryptographique *Noise* de `libp2p`. Même si l'hébergeur cloud supprime la couche HTTPS externe, les paquets internes restent chiffrés de bout en bout entre les deux clients T2C Phantom. Le Relais Cloud n'est qu'un "tuyau opaque" aveugle.

### 3. Les Attaquants sur le réseau P2P (DHT Sniffing)
- **Le Risque :** Un pirate rejoint le réseau T2C pour écouter qui cherche quoi, ou tente de cibler votre adresse IP physique.
- **Notre Protection :** Si vous utilisez le mode Relais (`--use-relay`), votre véritable adresse IP physique (ex: votre box à la maison) n'est jamais exposée sur le réseau P2P mondial. Le réseau ne voit que l'adresse IP publique de votre relais. *(Note : La confidentialité des métadonnées requiert l'usage conjoint d'un VPN externe si l'anonymat absolu vis-à-vis du relais lui-même est requis).*

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
