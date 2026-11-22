# Signature Apple : T2C Phantom (notarisé) et T2DECODE (App Store)

Ce guide se fait **une seule fois**. Ensuite, chaque version taguée est signée, notarisée et envoyée à Apple automatiquement, avec le même numéro de version que dans GitHub et GitLab.

| App | Distribution | Résultat |
|---|---|---|
| **T2C Phantom** | téléchargement (GitHub, GitLab) | signée **Developer ID** et **notarisée** : aucune alerte « app non fiable », double-clic suffit |
| **T2DECODE** | **Mac App Store** (et téléchargement notarisé) | envoyée dans **App Store Connect** à chaque tag, soumise à l'examen d'Apple si tu le veux |

Pourquoi T2C Phantom n'est pas sur l'App Store : Apple y impose le « bac à sable », qui interdit de régler le proxy du système et d'installer le certificat `.t2c`. La notarisation donne la même confiance à macOS.

Personne d'autre que toi ne doit voir les certificats, mots de passe et clés ci-dessous : tu les crées et tu les colles toi-même dans GitHub. Ils ne sont jamais écrits dans un dépôt.

---

## 1. Clé API App Store Connect (pour les deux apps)

1. [App Store Connect](https://appstoreconnect.apple.com) → **Users and Access** → **Integrations** → **App Store Connect API** → **Team Keys** → **+**.
2. Nom : `CI TUTODECODE`, accès : **App Manager**.
3. Télécharge le fichier `AuthKey_XXXXXXXXXX.p8` (**une seule fois possible**) et note le **Key ID** et l'**Issuer ID** affichés en haut.
4. Encode la clé :

```bash
base64 -i ~/Downloads/AuthKey_XXXXXXXXXX.p8 | pbcopy
```

Le contenu est dans le presse-papiers, prêt à coller dans le secret `ASC_KEY_P8_BASE64`. Range ensuite le `.p8` hors ligne avec tes autres secrets.

## 2. Certificat Developer ID (T2C Phantom, et T2DECODE hors App Store)

Ton Mac a déjà **« Developer ID Application: TUTODECODE (TS97M57BJV) »**.

1. Ouvre **Trousseaux d'accès** → **Mes certificats** → clic droit sur ce certificat → **Exporter…** → format **.p12**, avec un mot de passe solide.
2. Encode-le :

```bash
base64 -i ~/Desktop/DeveloperID.p12 | pbcopy
```

Puis supprime le `.p12` exporté : il reste dans ton trousseau.

## 3. Certificats et profil Mac App Store (T2DECODE)

Sur [developer.apple.com](https://developer.apple.com/account/resources/certificates/list), dans l'équipe qui possède l'app T2DECODE :

1. **Certificates → +** → **Apple Distribution**. Installe-le (double-clic), puis exporte-le en `.p12`.
2. **Certificates → +** → **Mac Installer Distribution**. Même chose.
3. **Profiles → +** → **Mac App Store Connect** → App ID `org.t2decode.app` → certificat Apple Distribution → télécharge le `.provisionprofile`.
4. Encode les trois fichiers avec `base64 -i FICHIER | pbcopy`, un par un.

## 4. Secrets GitHub

### Dépôt `TUTODECODE-FR/T2C-Phantom` → Settings → Secrets and variables → Actions

| Secret | Contenu |
|---|---|
| `MACOS_CERT_P12_BASE64` | le `.p12` Developer ID encodé (étape 2) |
| `MACOS_CERT_PASSWORD` | son mot de passe |
| `MACOS_CERT_IDENTITY` | `Developer ID Application: TUTODECODE (TS97M57BJV)` |
| `ASC_KEY_ID` | Key ID (étape 1) |
| `ASC_ISSUER_ID` | Issuer ID (étape 1) |
| `ASC_KEY_P8_BASE64` | la clé `.p8` encodée (étape 1) |

Quand tout est en place, ajoute la **variable** (pas un secret) `MACOS_SIGN_REQUIRED` = `1` : une version non signée ne pourra plus partir par erreur.

### Dépôt `TUTODECODE-FR/T2DECODE`

Les mêmes six secrets que ci-dessus, plus :

| Secret | Contenu |
|---|---|
| `MAS_APP_CERT_P12_BASE64` | `.p12` Apple Distribution (étape 3.1) |
| `MAS_INSTALLER_CERT_P12_BASE64` | `.p12` Mac Installer Distribution (étape 3.2) |
| `MAS_CERT_PASSWORD` | mot de passe de ces deux `.p12` (le même pour les deux) |
| `MAS_PROVISION_PROFILE_BASE64` | profil `.provisionprofile` (étape 3.3) |
| `MAS_APP_IDENTITY` | `Apple Distribution: TUTODECODE (ÉQUIPE)`, tel qu'affiché dans le Trousseau |
| `MAS_INSTALLER_IDENTITY` | `3rd Party Mac Developer Installer: TUTODECODE (ÉQUIPE)` |

Variables (facultatives) : `APPSTORE_SUBMIT` = `1` pour **soumettre automatiquement à l'examen d'Apple**, et `APPSTORE_AUTO_RELEASE` = `1` pour **publier dès l'approbation**.

## 5. T2DECODE : GitHub d'abord, GitLab reçoit

GitHub est le dépôt principal : la CI y compile et y signe tout. GitLab reçoit ensuite le code (`mirror-gitlab.yml`, à chaque envoi sur `main` et à chaque tag) et les fichiers signés de chaque version (job `gitlab` de `release.yml`).

1. GitLab → projet `tutodecode-org/T2DECODE` → **Settings → Access tokens** → jeton de projet, rôle **Maintainer**, droits **api** et **write_repository**, expiration 1 an.
2. GitHub → `TUTODECODE-FR/T2DECODE` → **Settings → Secrets and variables → Actions** → secret `GITLAB_TOKEN` = ce jeton.
3. Variable facultative `GITLAB_PROJECT` si le projet GitLab change de chemin (défaut `tutodecode-org/T2DECODE`).

Sans ce secret, GitHub fonctionne normalement et GitLab n'est simplement pas mis à jour.

## 6. Publier une version

```bash
git tag -a v1.0.5 -m "T2DECODE 1.0.5"
git push github v1.0.5
```

Pour T2DECODE : le tag part sur GitHub, la CI le recopie sur GitLab, et :
1. compile l'app Mac avec la version `1.0.5` et un numéro de build croissant ;
2. la signe pour l'App Store et l'envoie dans **App Store Connect**, où elle apparaît dans **TestFlight** au bout de 10 à 30 minutes ;
3. si `APPSTORE_SUBMIT` = `1`, la soumet à l'examen d'Apple.

**Apple examine chaque version** (en général 1 à 2 jours) : ni nous ni personne ne peut l'éviter. Avec `APPSTORE_AUTO_RELEASE` = `1`, elle est publiée dès son approbation.

Pour T2C Phantom : même chose avec la CI de release habituelle. L'app et les outils macOS sortent signés et notarisés.

## 7. Vérifier

Sur un Mac, après téléchargement :

```bash
spctl --assess --type execute --verbose=2 "/Applications/T2C Phantom.app"
```

La réponse attendue est `accepted` et `source=Notarized Developer ID`.

Premier essai sans rien publier : GitHub → **Actions** → **Release** → **Run workflow** avec `dry_run` coché. Les fichiers signés sont téléchargeables dans les « artifacts » de l'exécution.

## Windows

Voir [Signature Windows avec SignPath Foundation](SIGNATURE-WINDOWS.md).

## Snap Store (T2DECODE, snapcraft.io/t2decode)

La CI `snap.yml` de T2DECODE compile le snap (amd64 et arm64) à chaque tag, avec la version du tag, et le publie sur le canal **stable**. Les ordinateurs qui ont installé le snap se mettent ensuite à jour tout seuls.

Il suffit d'un secret dans le dépôt GitHub de T2DECODE, `SNAPCRAFT_STORE_CREDENTIALS`, limité à ce snap et valable un an :

```bash
snapcraft export-login --snaps t2decode --acls package_access,package_push,package_release --expires 2027-11-22 -
```

La commande te demande de te connecter à ton compte Snapcraft, puis affiche un long texte : colle-le **entier** comme valeur du secret. Un fichier `snapcraft_credentials.txt` qui traîne dans ton dossier personnel peut contenir d'anciens identifiants : s'ils ne servent plus, supprime-le.
