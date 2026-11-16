# Guide d'utilisation

## Application (recommandé)

### Installer

Télécharge le fichier de ton système dans les [Releases](https://github.com/TUTODECODE-FR/T2C-Phantom/releases), puis vérifie-le avec `SHA256SUMS.txt` :

```bash
shasum -a 256 T2C-Phantom-macOS-universal.zip   # macOS / Linux (sha256sum)
```

```powershell
Get-FileHash T2C-Phantom-Windows-amd64.zip       # Windows
```

| Système | Installation |
|---|---|
| macOS | dézippe, glisse **T2C Phantom** dans Applications ; premier lancement : clic droit → **Ouvrir** → Ouvrir |
| Windows | dézippe, lance `T2C-Phantom.exe` ; si SmartScreen s'affiche : « Informations complémentaires » → **Exécuter quand même** |
| Linux | `tar xzf T2C-Phantom-Linux-amd64.tar.gz` puis `./T2C-Phantom` ; prérequis : `sudo apt install libwebkit2gtk-4.1-0 libnss3-tools` (Debian/Ubuntu) |

### Premier lancement

1. La connexion démarre toute seule. L'écran indique **Connecté · identité pseudonyme** et ton identifiant de 32 caractères.
2. Une fenêtre propose **Configurer automatiquement** : coche l'accord, puis valide.
   - macOS demande ton mot de passe de session (ajout du certificat au trousseau).
   - Windows affiche un avertissement de sécurité sur le certificat : clique **Oui**.
3. Redémarre ton navigateur s'il était ouvert, puis ouvre **`https://tutodecode.t2c`**.

Si la CA est déjà installée, l'application ne te redemande rien.

### Au quotidien

- **Laisse l'application ouverte** pendant que tu consultes des sites `.t2c`. À la fermeture, ton proxy d'origine est remis ; à la réouverture, tout se reconfigure automatiquement.
- **Bénévole** (activé par défaut) : les sites que tu visites sont gardés et redistribués aux autres pairs. Pour le désactiver : **Déconnecter**, décoche « Bénévole (héberger une copie) », puis relance la connexion.
- **Onglets** :
  - *Testeur .t2c* : vérifie qu'un site répond ;
  - *Réseau live* : ton identité, les sites hébergés, les bénévoles rencontrés et leur score, le catalogue signé ;
  - *Association TUTODECODE* : informations légales.

## Ligne de commande

Pour un serveur, un Raspberry Pi ou sans interface graphique, télécharge l'archive `t2c-cli-…` de ton système. Elle contient `t2c` et `t2c-peer`.

```bash
./t2c doctor          # diagnostic : clé racine, DNS du relais, port 8080, dossier de données
./t2c setup           # configure les navigateurs (une fois) — certificat .t2c + proxy PAC
./t2c start           # client + proxy .t2c sur 127.0.0.1:8080, bénévole pour les sites visités
./t2c unsetup         # remet les réglages proxy d'origine
./t2c id              # affiche ton identité pseudonyme
./t2c dashboard       # tableau de bord dans le navigateur (http://127.0.0.1:9090)
```

Options de `t2c start` :

| Option | Rôle |
|---|---|
| `-proxy 8080` | port du proxy local (`0` = désactivé) |
| `-no-volunteer` | ne pas héberger de copie |
| `-host site1,site2` | héberger ces sites dès le démarrage |
| `-relay wss://…` | autre relais (variable `T2C_RELAY_URL`) |
| `-carnet fichier.json` | alias locaux (`{"cours.t2c": "tutodecode"}`) |
| `-metrics-port 2112` | métriques Prometheus sur `127.0.0.1` |

Client minimal, sans proxy :

```bash
./t2c-peer list                       # catalogue signé + bénévoles en ligne
./t2c-peer fetch tutodecode ./site    # télécharge et vérifie un site dans ./site
./t2c-peer host tutodecode            # devient bénévole pour ce site (reste connecté)
```

## Où sont mes données ?

| Système | Dossier |
|---|---|
| macOS | `~/Library/Caches/t2c-phantom/` |
| Windows | `%LocalAppData%\t2c-phantom\` |
| Linux | `~/.cache/t2c-phantom/` |

On y trouve :
- `peer/identity.json` : ton identité, **ne la partage pas** ;
- `peer/hosted/` : les copies hébergées ;
- `peer/sites/` : les sites vérifiés ;
- `asso-ca/` : le certificat local `.t2c`.

La variable `T2C_DATA_DIR` permet de changer ce dossier.

## Dépannage

| Symptôme | Solution |
|---|---|
| « relais injoignable » | vérifie ta connexion Internet ; `t2c doctor` teste la résolution DNS du relais |
| le navigateur affiche « non sécurisé » sur un `.t2c` | relance **Configurer automatiquement**, puis redémarre complètement le navigateur |
| `xxx.t2c n'est pas dans le catalogue signé` | ce site n'existe pas (ou plus) : seul le catalogue officiel est servi |
| un site non `.t2c` ne charge plus | quitte et relance l'application (le PAC est remis) ; en dernier recours `t2c unsetup` |
| Linux : « installation automatique impossible » | `sudo apt install libnss3-tools` (ou `nss-tools`), puis relance la configuration |
| port 8080 déjà utilisé | `T2C_PROXY_PORT=8081` avant de lancer l'application (ou `t2c start -proxy 8081`) |

## Désinstaller

1. Dans l'application, quitte normalement : le proxy d'origine est remis. En ligne de commande : `t2c unsetup`.
2. Supprime l'application et le dossier de données ci-dessus.
3. Facultatif : retire le certificat **« tutodecode CA »**.
   - macOS : Trousseau d'accès → session → Certificats ;
   - Windows : `certmgr.msc` → Autorités de certification racines de confiance ;
   - Firefox : Réglages → Certificats → Autorités.

## Signaler un problème

- Bug ou question : [issues GitHub](https://github.com/TUTODECODE-FR/T2C-Phantom/issues) ou `contact@tutodecode.org`.
- Faille de sécurité : **en privé** (voir [SECURITY.md](../SECURITY.md)).
