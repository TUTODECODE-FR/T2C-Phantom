# Cartographie du client (T2C Phantom v2)

Ce dépôt contient le **client** et le **protocole**. Le relais et le serveur origine sont dans des dépôts privés séparés.

Seul Cloudflare voit l'IP des utilisateurs. Le relais, l'origine et les pairs ne connaissent que des clés aléatoires. Tout contenu est signé par l'origine ou adressé par son hash.

```text
T2C-Phantom/
├── cmd/
│   ├── t2c/          Client : daemon, proxy .t2c, tableau de bord
│   └── t2c-peer/     Client minimal en ligne de commande (list / fetch / host)
├── desktop/          Application de bureau (Wails : macOS, Windows, Linux)
├── docs/             FONCTIONNEMENT.md (comment ça marche), UTILISATION.md (guide)
├── pkg/
│   ├── trust/        Protocole : catalogue, manifestes, hosts.json, tickets signés ; morceaux 256 Ko ; anti-retour arrière
│   ├── peerid/       Protocole : identité pseudonyme (clé Ed25519 + PoW, clé Noise signée)
│   ├── wire/         Protocole : trames WebSocket entre pairs et relais
│   ├── peerclient/   Noise IK de bout en bout, téléchargement parallèle vérifié, bénévolat, score local
│   ├── node/         Daemon : reconnexion, catalogue, alias → dossier local, tickets persistants
│   ├── proxy/        Proxy local .t2c (HTTP + HTTPS via CA locale, catalogue signé uniquement)
│   ├── api/          Tableau de bord (interface intégrée dans api/web) : fenêtre de l'app ou 127.0.0.1:9090
│   ├── sysint/       Configuration automatique : CA .t2c (système + Firefox/Zen) et proxy PAC, par OS
│   ├── metrics/      Compteurs Prometheus (127.0.0.1)
│   ├── rpc/          Pont d'évènements WebSocket pour un futur client Flutter
│   └── utils/        Chemins et journal de sécurité
```

## Points de contrôle côté client

- **Authenticité** : chaque document est vérifié avec la clé racine intégrée au build (`trust.RootPubKeyHex`).
- **Intégrité** : `Manifest.VerifyChunk`. Un morceau faux entraîne un ban local et un signalement du bénévole.
- **Bout en bout** : les échanges entre pairs sont chiffrés en Noise IK ; le relais ne les lit pas.
- **Proxy** : ne sert et ne certifie en HTTPS que les sites du catalogue signé ; le PAC n'y envoie que les `.t2c`.
- **CA locale** : Name Constraints `.t2c` — même volée, sa clé ne peut imiter aucun autre site.
- **Identité** : `peerid` (clé aléatoire + PoW). Supprimer `identity.json` crée une nouvelle identité.
