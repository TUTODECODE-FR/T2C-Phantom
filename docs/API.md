# API et Commandes T2C

## Commandes du CLI

### Générer une identité
```bash
t2c genkey
```
Génère une nouvelle paire de clés Ed25519. Affiche l'adresse `.t2c` (publique) et la clé privée encodée en Base64.

### Démarrer le démon
```bash
t2c start --key <PRIVATE_KEY_BASE64> [--port 4001] [--proxy 8080] [--expose 8000] [--bootstrap /dns4/node/tcp/4001/p2p/PEER_ID]
```

#### Options :
- `--key` : Requis. La clé privée en Base64 générée par `genkey`.
- `--port` : Optionnel. Le port TCP sur lequel le nœud P2P écoutera (défaut : 4001).
- `--proxy` : Optionnel. Active un proxy HTTP sur le port local spécifié pour intercepter les requêtes `.t2c` du navigateur ou de curl.
- `--expose` : Optionnel. Lie un service HTTP local existant (ex: un site sur 8000) au démon P2P. Toute connexion P2P entrante sera redirigée vers localhost:8000.
- `--bootstrap` : Optionnel. Un ou plusieurs (séparés par des virgules) multiaddrs libp2p pour rejoindre un réseau existant. (Sur un réseau local, le mDNS est activé automatiquement et permet la découverte sans bootstrap).

## Exemple de flux (Sprint 1)

**Sur l'ordinateur A (Serveur Web)** :
1. Démarrer un serveur web : `python3 -m http.server 8000`
2. Créer une clé T2C : `t2c genkey` (Noter l'adresse `[ADRESSE_A].t2c` et la clé privée).
3. Lancer T2C et exposer le port 8000 : `t2c start --key [CLE_PRIVEE] --expose 8000 --port 4001`

**Sur l'ordinateur B (Client)** :
1. Créer une clé T2C : `t2c genkey` (Noter la clé privée).
2. Lancer T2C en tant que proxy : `t2c start --key [CLE_PRIVEE_B] --proxy 8080 --port 4002`
3. Ouvrir le navigateur et configurer le proxy HTTP sur `127.0.0.1:8080`.
4. Naviguer vers `http://[ADRESSE_A].t2c`.
5. La page s'affiche de manière chiffrée de bout en bout via l'Overlay Network.
