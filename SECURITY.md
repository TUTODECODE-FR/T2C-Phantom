# Politique de Sécurité (T2C-Phantom)

La transparence et la confiance sont au cœur du réseau T2C Phantom. Ce dépôt publie le client complet pour qu'il puisse être audité.

## 🛡️ Modèle de menaces

Objectif assumé : **protéger l'adresse IP** des utilisateurs et des bénévoles, et **garantir l'authenticité** des sites. T2C Phantom n'est pas un outil d'anonymat complet comme Tor. Détails : [docs/FONCTIONNEMENT.md](docs/FONCTIONNEMENT.md).

| Menace | Protection |
|---|---|
| Un bénévole modifie un site (virus, contenu altéré) | Chaque morceau est vérifié contre l'empreinte signée par TUTODECODE ; morceau rejeté, bénévole banni, signalé et contrôlé |
| Le relais lit ou modifie les échanges entre pairs | Chiffrement de bout en bout Noise IK ; le relais n'a pas la clé de signature et ne peut pas falsifier un site |
| Le relais ou l'origine récupère ton IP | Ils ne la reçoivent pas : seul Cloudflare la voit. L'origine refuse toute requête portant un en-tête d'IP |
| Un vieux catalogue est rejoué | Version signée strictement croissante, retour en arrière refusé |
| Flood ou création massive d'identités | Cloudflare devant le relais, limites par clé, preuve de travail pour créer une identité |
| Le certificat local est volé | Il est limité aux noms `.t2c` (Name Constraints) : il ne permet d'imiter aucun autre site |
| Le proxy capte toute la navigation | Le fichier PAC n'envoie que les `*.t2c` ; tout le reste est en connexion directe |

**Ce qui reste visible (limites assumées)**
- **Cloudflare** voit ton IP et les sites `.t2c` demandés.
- **Le relais** voit ta clé pseudonyme, les sites demandés et avec quelles clés tu échanges, mais jamais ton IP.
- Si même Cloudflare ne doit pas connaître ton IP, utilise un VPN ou Tor *avant* T2C Phantom (logiciels tiers, sous ta responsabilité).
- Un poste compromis (logiciel malveillant, keylogger) est hors de portée de toute protection réseau.

Le code du relais et de l'origine n'est pas publié. La sécurité des utilisateurs n'en dépend pas : tout ce qui protège ta vie privée et l'intégrité des sites s'exécute dans le client public de ce dépôt.

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
- **Remédiation** : Après un correctif, une nouvelle release est générée par GitHub Actions (avec `SHA256SUMS.txt`) et recopiée sur GitLab.
