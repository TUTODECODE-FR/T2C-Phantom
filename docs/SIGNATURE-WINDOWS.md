# Signature Windows avec SignPath Foundation

SignPath Foundation signe **gratuitement** les applications des projets libres. Une fois en place, l'alerte SmartScreen de Windows (« Windows a protégé votre ordinateur ») disparaît au fil des téléchargements, car la réputation se construit sur le certificat.

> **État au 2026-11-22 : demande refusée** par SignPath Foundation (visibilité publique insuffisante : étoiles, contributeurs, articles). Recandidater après l'ouverture publique et l'activation de `t2cphantom.eu.org`. La CI ci-dessous reste en place et ne fait rien tant que le jeton n'existe pas.

La CI est déjà prête : dès que le secret `SIGNPATH_API_TOKEN` existe, chaque release envoie l'exécutable Windows à SignPath, attend ton approbation, puis publie le fichier signé, sur GitHub d'abord, puis recopié à l'identique sur GitLab.

## 1. Avant de candidater (déjà fait dans le dépôt)

- Licence libre approuvée par l'OSI : **AGPL-3.0-or-later**.
- Page **[Code signing policy](CODE-SIGNING-POLICY.md)**, liée depuis le README et les notes de chaque version, avec la mention exigée.
- Nom et version du produit inscrits dans l'exécutable à chaque compilation (`T2C Phantom`, version du tag).
- Assistant qui annonce et demande l'accord avant toute modification du système ; désinstallation documentée.

À vérifier de ton côté :
- **authentification à plusieurs facteurs** activée sur GitHub, GitLab et, plus tard, SignPath ;
- les conditions ne disent pas explicitement qu'un dépôt **public** est exigé, mais SignPath vérifie l'origine des compilations et examine le projet. Le plus simple est de candidater **quand les dépôts seront publics**, avec le domaine `t2cphantom.eu.org`.

## 2. Candidater

Sur [signpath.org](https://signpath.org), bouton **Apply**. Indique :
- projet : **T2C Phantom**, association **TUTODECODE** ;
- dépôt : `https://github.com/TUTODECODE-FR/T2C-Phantom` (et `https://gitlab.com/tutodecode-org/T2C-Phantom`) ;
- licence : AGPL-3.0-or-later ;
- politique : lien vers `docs/CODE-SIGNING-POLICY.md` ;
- ce qui est signé : l'application de bureau Windows (`T2C-Phantom.exe`), compilée par GitHub Actions.

L'équipe SignPath examine la demande, ce qui peut prendre quelques semaines.

## 3. Après acceptation, sur app.signpath.io

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

## 4. Dans GitHub (T2C-Phantom → Settings → Secrets and variables → Actions)

| Type | Nom | Valeur |
|---|---|---|
| Secret | `SIGNPATH_API_TOKEN` | jeton de l'utilisateur CI SignPath |
| Variable | `SIGNPATH_ORGANIZATION_ID` | identifiant d'organisation affiché par SignPath |

Variables facultatives, si tu choisis d'autres noms : `SIGNPATH_PROJECT_SLUG` (défaut `T2C-Phantom`), `SIGNPATH_POLICY_SLUG` (défaut `release-signing`), `SIGNPATH_ARTIFACT_SLUG` (défaut `initial`).

## 5. À chaque version

La CI s'arrête sur l'étape « Signature Windows (SignPath Foundation) » : SignPath t'envoie la demande, tu l'**approuves**, et la release continue toute seule. Délai maximum : 1 heure, sinon la release échoue et on la relance.

Pour vérifier sous Windows : clic droit sur `T2C-Phantom.exe` → **Propriétés** → **Signatures numériques** : « SignPath Foundation ».
