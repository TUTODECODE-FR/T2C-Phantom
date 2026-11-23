# Signature Windows

Deux voies, une seule active à la fois :

- **Azure Artifact Signing** (voie choisie) : payé par le crédit Azure annuel de Microsoft for Nonprofits, certificat au nom de l'association, signature automatique dans la CI (job `sign-windows` de `release.yml` : app, `t2c.exe`, `t2c-peer.exe`, horodatés).
- **SignPath Foundation** (plan B, refusé pour l'instant) : voir plus bas.

Sans configuration, la release sort avec des exécutables Windows **non signés** et un avertissement dans la CI.

## Azure Artifact Signing

Coût : formule Basic, environ 10 $/mois, à imputer sur le crédit Azure nonprofit. **À vérifier dans le Nonprofit Hub** que le crédit couvre bien ce service (les offres de la Marketplace en sont exclues).

### A. Dans le portail Azure (abonnement du crédit nonprofit)

1. **Abonnement → Fournisseurs de ressources** : enregistrer `Microsoft.CodeSigning`.
2. **Créer un compte Artifact Signing** (« Artifact Signing Accounts ») : région Europe de l'Ouest, formule Basic. Noter son **nom** et son **point de terminaison** (`https://weu.codesigning.azure.net/` pour l'Europe de l'Ouest ; il doit correspondre à la région).
3. **Contrôle d'accès (IAM) du compte** : te donner le rôle « Artifact Signing Identity Verifier » (ancien nom « Trusted Signing Identity Verifier »).
4. **Identity validation → New identity → Public Trust → Organization** : identité de l'association (nom exact, adresse, numéro RNA/SIREN, site, e-mail du domaine si possible). Microsoft vérifie les documents, ce qui peut prendre plusieurs jours. Les conditions d'éligibilité (ancienneté de l'organisation, pays) sont à relire au moment de la demande.
5. **Certificate profiles → Create → Public Trust** : lié à l'identité validée, nom par exemple `t2c-phantom`.

### B. Accès de GitHub sans secret (OIDC)

1. **Microsoft Entra ID → Inscriptions d'applications → Nouvelle** : `github-t2c-signature`. Noter l'**ID d'application (client)** et l'**ID de l'annuaire (locataire)**.
2. Dans cette application, **Certificats et secrets → Informations d'identification fédérées → Ajouter** : scénario « GitHub Actions », organisation `TUTODECODE-FR`, dépôt `T2C-Phantom`, type d'entité **Environment**, nom `signature-windows`. Aucun secret client à créer.
3. Sur le **profil de certificat** (ou le compte), IAM : donner à `github-t2c-signature` le rôle « Artifact Signing Certificate Profile Signer ». Rien d'autre.

### C. Dans GitHub (T2C-Phantom → Settings)

1. **Environments → New environment** `signature-windows` : « Deployment branches and tags » = **Selected** : branche `main` (essais à blanc) et tags `v*`. Optionnel : toi comme « Required reviewer » pour approuver chaque signature.
2. **Secrets and variables → Actions → Variables** (variables du dépôt, pas de secrets) :

| Variable | Valeur |
|---|---|
| `AZURE_CLIENT_ID` | ID d'application de `github-t2c-signature` |
| `AZURE_TENANT_ID` | ID du locataire |
| `AZURE_SUBSCRIPTION_ID` | abonnement du crédit nonprofit |
| `AZURE_SIGNING_ENDPOINT` | ex. `https://weu.codesigning.azure.net/` |
| `AZURE_SIGNING_ACCOUNT` | nom du compte Artifact Signing (**active la signature**) |
| `AZURE_CERT_PROFILE` | nom du profil, ex. `t2c-phantom` |

3. Essai à blanc : `gh workflow run release.yml --ref main -f tag=essai-XXX -f dry_run=true` (tag `essai-…` poussé d'abord), puis vérifier dans le journal du job « Signature Windows (Azure) » : `Valid`, sujet = l'association, horodatage présent.

La réputation SmartScreen se construit ensuite au fil des téléchargements : l'alerte peut rester sur les toutes premières versions signées.

---

# Plan B : SignPath Foundation

SignPath Foundation signe **gratuitement** les applications des projets libres. Une fois en place, l'alerte SmartScreen de Windows (« Windows a protégé votre ordinateur ») disparaît au fil des téléchargements, car la réputation se construit sur le certificat.

> **État au 2026-11-22 : demande refusée** par SignPath Foundation (visibilité publique insuffisante : étoiles, contributeurs, articles). Recandidater après l'ouverture publique et l'activation de `t2cphantom.eu.org`. La CI ci-dessous reste en place et ne fait rien tant que le jeton n'existe pas.

La CI est déjà prête : dès que le secret `SIGNPATH_API_TOKEN` existe, chaque release envoie l'exécutable Windows à SignPath, attend ton approbation, puis publie le fichier signé, sur GitHub d'abord, puis recopié à l'identique sur GitLab.

## B1. Avant de candidater (déjà fait dans le dépôt)

- Licence libre approuvée par l'OSI : **AGPL-3.0-or-later**.
- Page **[Code signing policy](CODE-SIGNING-POLICY.md)**, liée depuis le README et les notes de chaque version, avec la mention exigée.
- Nom et version du produit inscrits dans l'exécutable à chaque compilation (`T2C Phantom`, version du tag).
- Assistant qui annonce et demande l'accord avant toute modification du système ; désinstallation documentée.

À vérifier de ton côté :
- **authentification à plusieurs facteurs** activée sur GitHub, GitLab et, plus tard, SignPath ;
- les conditions ne disent pas explicitement qu'un dépôt **public** est exigé, mais SignPath vérifie l'origine des compilations et examine le projet. Le plus simple est de candidater **quand les dépôts seront publics**, avec le domaine `t2cphantom.eu.org`.

## B2. Candidater

Sur [signpath.org](https://signpath.org), bouton **Apply**. Indique :
- projet : **T2C Phantom**, association **TUTODECODE** ;
- dépôt : `https://github.com/TUTODECODE-FR/T2C-Phantom` (et `https://gitlab.com/tutodecode-org/T2C-Phantom`) ;
- licence : AGPL-3.0-or-later ;
- politique : lien vers `docs/CODE-SIGNING-POLICY.md` ;
- ce qui est signé : l'application de bureau Windows (`T2C-Phantom.exe`), compilée par GitHub Actions.

L'équipe SignPath examine la demande, ce qui peut prendre quelques semaines.

## B3. Après acceptation, sur app.signpath.io

1. **Project** : slug `T2C-Phantom`.
2. **Trusted build system** : GitHub.com, en installant l'application GitHub de SignPath sur le dépôt `TUTODECODE-FR/T2C-Phantom`.
3. **Artifact configuration** (slug `initial`) : l'artefact envoyé par la CI est un zip contenant `T2C-Phantom.exe`.

```xml
<?xml version="1.0" encoding="utf-8"?>
<artifact-configuration xmlns="http://signpath.io/artifact-configuration/v1">
  <zip-file>
    <pe-file path="T2C-Phantom.exe" product-name="T2C Phantom">
      <authenticode-sign/>
    </pe-file>
  </zip-file>
</artifact-configuration>
```

4. **Signing policy** : slug `release-signing`, certificat de SignPath Foundation, **approbation manuelle** par toi.
5. **API token** : crée un utilisateur CI (« CI user ») avec le droit de soumettre sur ce projet, puis copie son jeton.

## B4. Dans GitHub (T2C-Phantom → Settings → Secrets and variables → Actions)

| Type | Nom | Valeur |
|---|---|---|
| Secret | `SIGNPATH_API_TOKEN` | jeton de l'utilisateur CI SignPath |
| Variable | `SIGNPATH_ORGANIZATION_ID` | identifiant d'organisation affiché par SignPath |

Variables facultatives, si tu choisis d'autres noms : `SIGNPATH_PROJECT_SLUG` (défaut `T2C-Phantom`), `SIGNPATH_POLICY_SLUG` (défaut `release-signing`), `SIGNPATH_ARTIFACT_SLUG` (défaut `initial`).

## B5. À chaque version

La CI s'arrête sur l'étape « Signature Windows (SignPath Foundation) » : SignPath t'envoie la demande, tu l'**approuves**, et la release continue toute seule. Délai maximum : 1 heure, sinon la release échoue et on la relance.

Pour vérifier sous Windows : clic droit sur `T2C-Phantom.exe` → **Propriétés** → **Signatures numériques** : « SignPath Foundation ».
