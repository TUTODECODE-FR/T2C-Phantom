# 🗺️ Cartographie de l'Architecture (T2C-Phantom)
**Statut : Backend V1 & Infrastructure DMZ validés et figés.**

Cette radiographie présente l'arborescence complète du dépôt ainsi que la répartition stricte des responsabilités de chaque composant pour le réseau T2C.

---

## 🏗️ Structure du Dépôt

```text
📁 T2C-Phantom/
├── 📁 cmd/
│   └── 📁 t2c/
│       └── ⚙️ main.go           # Point d'entrée principal. Démarre le démon, le pont P2P/RPC, et initialise le serveur de métriques Prometheus.
├── 📁 infra/
│   ├── 📁 chaos/
│   │   └── ⚙️ monkey.sh         # Script Bash de Chaos Engineering. Injecte de la latence, détruit des paquets (tc netem) et gèle aléatoirement les processus (SIGSTOP) pour valider la résilience du système de Failover.
│   ├── 📁 lxc-proxy/
│   │   └── 🔒 nginx.conf        # Bouclier frontal DMZ. Force le TLS, transmet les en-têtes X-Forwarded-For et bloque instantanément les attaques par déni de service (Rate Limiting de 5 req/s).
│   └── 📁 lxc-serveur/
│       └── 🔒 iptables.sh       # Mur pare-feu applicatif. Isole le LXC cerveau et bloque drastiquement tout mouvement latéral, n'autorisant que le port d'application restreint.
├── 📁 pkg/
│   ├── 📁 crypto/
│   │   ├── 🔒 merkle.go         # Le mur Zero-Trust. Vérifie cryptographiquement chaque hash de fragment en RAM *avant* toute écriture disque pour annihiler les attaques TOCTOU et les empoisonnements (Fuzzing).
│   │   └── ⚙️ merkle_test.go    # Tests unitaires du modèle Zero-Trust et résistance au fuzzing.
│   ├── 📁 metrics/
│   │   └── ⚙️ metrics.go        # SOC de supervision (Security Operations Center). Exporte les métriques d'attaque et de réseau via Prometheus (`t2c_blocks_corrupted_total`, `t2c_failovers_total`, `t2c_peers_banned_total`).
│   ├── 📁 p2p/
│   │   ├── 🔒 downloader.go     # Le Moteur de Survie. Orchestre la vérification de blocs, dénonce le pair malveillant en cas de corruption, et déclenche le Failover automatique transparent vers le serveur HTTP (Proxy).
│   │   ├── ⚙️ downloader_test.go# Tests du basculement automatique et rejets des blocs empoisonnés.
│   │   └── ⚙️ node.go           # Moteur réseau libp2p. Intègre la stéganographie DPI via WebRTC-Direct pour outrepasser les pares-feux stricts des universités.
│   ├── 📁 proxy/
│   │   └── ⚙️ server.go         # API Gateway HTTP. Expose des endpoints RESTful.
│   ├── 📁 rpc/
│   │   ├── ⚙️ server.go         # Pont de communication inter-process. Diffuse les évènements d'avancement et d'état en WebSocket au futur client Flutter de façon asynchrone.
│   │   └── ⚙️ server_test.go    # Tests de charge prouvant la fluidité du pont WebSocket.
│   └── 📁 tracker/
│       ├── 🔒 middleware.go     # Anti-Rejeu et Authentification. Intercepte les requêtes de l'UI. Valide les en-têtes `X-T2C-Timestamp`, le `Nonce` (anti-rejeu) et garantit la légitimité avec la vérification de la signature asymétrique Ed25519.
│       ├── 🔒 tracker.go        # L'Immunité du Réseau. L'Annuaire central qui collecte les dénonciations depuis le proxy (X-Forwarded-For) et expulse par blacklist tout noeud après quorum (Review bombing impossible).
│       └── ⚙️ tracker_test.go   # Tests du système immunitaire (Bannissement par consensus d'IP).
├── ⚙️ go.mod                  # Fichier de définition des dépendances Go.
└── ⚙️ go.sum                  # Hashes de vérification des bibliothèques.
```

---

## 🔍 Synthèse des Points de Contrôle de Sécurité

- 🔒 **Le mur Zero-Trust (Vérification RAM)** : Situé dans `pkg/crypto/merkle.go`, il recalcule les sommes SHA-256 en mémoire à chaque réception de segment (256 KB). La moindre corruption appelle le failover sans toucher le disque (Protection TOCTOU).
- 🔒 **L'Anti-Rejeu & Authentification** : Situé dans `pkg/tracker/middleware.go`. Il rejette toute requête vieille de plus de 5 minutes, garde en mémoire cache les Nonces déjà vus, et vérifie de manière implacable les clés Ed25519 sur chaque route.
- 🔒 **Le Moteur de Survie (Failover)** : Localisé dans `pkg/p2p/downloader.go`. Si la vérification `merkle.go` crashe, ce moteur éjecte le socket suspect et bifurque la requête vers le LXC sécurisé sans interrompre l'expérience utilisateur.
- 🔒 **L'Immunité P2P (Tracker / Ban)** : Dans `pkg/tracker/tracker.go`. Assure un comptage par IP entrante (via le `X-Forwarded-For` de Nginx) et décrète un ban irrévocable (niveau 1) au bout de 3 consensus.
- 🔒 **Le Hardening de la DMZ** : Situé dans l'infrastructure via les scripts Nginx (`infra/lxc-proxy/nginx.conf`) pour le Rate Limiting impitoyable et Netfilter (`infra/lxc-serveur/iptables.sh`) pour le verrouillage du réseau intra-LXC.

---

> [!NOTE]
> Le code Backend / Infra est audité, cartographié et stable. Transition prête vers la phase Frontend (Dart/Flutter).
