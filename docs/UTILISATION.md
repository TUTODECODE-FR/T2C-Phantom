# Guide d'utilisation

> Version détaillée et à jour : [Guide complet](GUIDE.md).

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
| macOS | dézippe et ouvre **T2C Phantom** (premier lancement : clic droit → **Ouvrir** → Ouvrir). L'application propose de se **déplacer dans Applications**. |
| Windows | dézippe et lance `T2C-Phantom.exe` (si SmartScreen s'affiche : « Informations complémentaires » → **Exécuter quand même**). L'application propose de **s'installer** dans `%LocalAppData%\Programs\T2C Phantom`, avec un raccourci sur le **Bureau** et dans le **menu Démarrer**. |
| Linux | `tar xzf T2C-Phantom-Linux-amd64.tar.gz` puis `./T2C-Phantom` (prérequis : `sudo apt install libwebkit2gtk-4.1-0 libnss3-tools` sur Debian/Ubuntu). L'application propose de **s'installer** dans `~/.local/bin`, avec une entrée dans le menu et un raccourci sur le Bureau. |

Pour ne plus voir cette proposition : bouton « Ne plus demander » sur macOS, ou variable `T2C_NO_INSTALL_PROMPT=1`.

### Premier lancement

Un assistant en 4 étapes s'affiche une seule fois :

1. **Certificat** : installe le certificat local, limité aux adresses `.t2c` (impossible de s'en servir pour imiter un autre site). macOS demande ton mot de passe de session, Windows affiche un avertissement : accepte. S'il est déjà installé, l'étape est validée d'office.
2. **Navigateur** : choisis le navigateur qui ouvrira les sites. **Firefox est conseillé** : il applique le certificat et le proxy sans aucun réglage.
3. **Proxy** : seules les adresses `.t2c` passent par T2C Phantom ; tout le reste de ta navigation est inchangé. Si ton navigateur était ouvert, ferme-le et rouvre-le.
4. **Réseau** : création de ton identité (une clé aléatoire, sans nom ni adresse) et connexion au réseau pair à pair.

L'assistant peut être relancé à tout moment : **Réglages**, **Refaire l'assistant**.

### Au quotidien

- **Carnet de sites** : la liste des sites du catalogue signé. **Ouvrir** lance le site dans ton navigateur, **Tester** vérifie qu'il répond (signature et contenu). Le champ du bas accepte n'importe quelle adresse `.t2c`.
- **Bénévolat** (désactivé par défaut) : active-le, puis choisis les sites que tu acceptes d'héberger, ou « Tous les sites ». Tu fixes la place maximale et le débit d'envoi.
  - Tes copies sont **chiffrées sur ton disque** et revérifiées à chaque envoi : une copie altérée n'est jamais servie, elle est supprimée puis retéléchargée.
  - Quand TUTODECODE publie une nouvelle version d'un site, ta copie est mise à jour automatiquement et l'ancienne est supprimée.
  - Les autres membres ne connaissent que ta clé pseudonyme.
- **Laisse l'application ouverte** pendant que tu consultes des sites `.t2c`.

## Ligne de commande

Pour un serveur, un Raspberry Pi ou sans interface graphique, télécharge l'archive `t2c-cli-…` de ton système. Elle contient `t2c` et `t2c-peer`.

```bash
./t2c doctor          # diagnostic : clé racine, relais joignables, port du proxy, dossier de données
./t2c setup           # configure les navigateurs (une fois) — certificat .t2c + proxy PAC
./t2c start           # client + proxy .t2c sur 127.0.0.1:8080 (bénévole seulement avec -host)
./t2c unsetup         # remet les réglages proxy d'origine
./t2c id              # affiche ton identité pseudonyme
./t2c dashboard       # tableau de bord dans le navigateur (http://127.0.0.1:9090)
```

Options de `t2c start` :

| Option | Rôle |
|---|---|
| `-proxy 8080` | port du proxy local (`0` = désactivé) |
| `-no-volunteer` | n'héberger aucune copie |
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
- `peer/hosted/` : les copies hébergées (chiffrées) et `peer/hosting.json`, tes choix de bénévole ;
- `peer/sites/` : les sites vérifiés ;
- `asso-ca/` : le certificat local `.t2c`.

La variable `T2C_DATA_DIR` permet de changer ce dossier.

## Dépannage

| Symptôme | Solution |
|---|---|
| « relais injoignable » | vérifie ta connexion Internet ; l'application essaie tour à tour tous les relais connus ; `t2c doctor` pour un diagnostic |
| le navigateur affiche « non sécurisé » sur un `.t2c` | **Réglages** → **Refaire l'assistant**, puis redémarre complètement le navigateur |
| `xxx.t2c n'est pas dans le catalogue signé` | ce site n'existe pas (ou plus) : seul le catalogue officiel est servi |
| un site non `.t2c` ne charge plus | quitte et relance l'application (le PAC est remis) ; en dernier recours `t2c unsetup` |
| Linux : « installation automatique impossible » | `sudo apt install libnss3-tools` (ou `nss-tools`), puis **Refaire l'assistant** |
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
