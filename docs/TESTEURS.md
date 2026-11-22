# Tester T2C Phantom — guide du testeur

Merci de tester T2C Phantom ! Tu reçois une **version de test interne** (v2.6.0). Merci de **ne pas redistribuer les fichiers** : la version publique sortira quand tout sera validé.

Compte environ **20 minutes**. Aucune compétence technique n'est nécessaire.

---

## 1. Ce que tu as reçu

| Fichier | Pour |
|---|---|
| `T2C-Phantom-macOS-universal.zip` | Mac (Intel et Apple Silicon) |
| `T2C-Phantom-Windows-amd64.zip` | Windows 10 ou 11 |
| `T2C-Phantom-Linux-amd64.tar.gz` | Linux |
| `SHA256SUMS.txt` | vérifier que le fichier est intact |

### Vérifier le fichier (recommandé)

Dans le dossier où se trouvent les fichiers :

- **Mac** (Terminal) : `shasum -a 256 -c SHA256SUMS.txt --ignore-missing`
- **Linux** : `sha256sum -c SHA256SUMS.txt --ignore-missing`
- **Windows** (PowerShell) : `Get-FileHash T2C-Phantom-Windows-amd64.zip`, puis compare le résultat avec la ligne correspondante de `SHA256SUMS.txt`.

Le fichier doit être marqué `OK` (ou, sous Windows, les deux suites de caractères doivent être identiques). Sinon, ne l'ouvre pas et préviens-nous.

## 2. Installer

- **Mac** : décompresse, glisse **T2C Phantom** dans **Applications**. Au premier lancement : **clic droit → Ouvrir**, puis **Ouvrir** (l'application n'est pas encore signée par Apple).
- **Windows** : décompresse, lance `T2C-Phantom.exe`. Si SmartScreen s'affiche : **Informations complémentaires → Exécuter quand même**.
- **Linux** : décompresse, lance `./T2C-Phantom`. Il faut `libwebkit2gtk-4.1-0`, et `libnss3-tools` pour la configuration automatique du navigateur.

## 3. Premier lancement

Un assistant te guide une seule fois :

1. **Certificat** : il ne sert qu'aux adresses `.t2c`. Ton système peut demander ton mot de passe.
2. **Navigateur** : Firefox est conseillé (aucun réglage à faire).
3. **Proxy** : seules les adresses `.t2c` passent par T2C Phantom ; le reste de ta navigation ne change pas. Si ton navigateur était ouvert, ferme-le et rouvre-le.
4. **Réseau** : l'application crée ton identité (une clé au hasard, sans nom ni e-mail) et se connecte.

> **Le premier chargement d'une page peut prendre quelques secondes : c'est normal.** Les pages traversent deux relais chiffrés et viennent en partie d'autres membres.

## 4. Ce qu'il faut tester

Coche au fur et à mesure, et note ce qui ne marche pas.

**Les sites**
- [ ] `https://tutodecode.t2c` s'ouvre et affiche la liste des sites.
- [ ] `https://cours.t2c` s'ouvre ; tu peux ouvrir un cours et télécharger un fichier `.tdc`.
- [ ] `https://repo.t2c` s'ouvre ; tu peux lire un fichier de code et télécharger un `.zip`.
- [ ] Une deuxième visite d'une même page est plus rapide que la première.

**L'application**
- [ ] Le badge **Anonyme** apparaît en haut (il peut mettre une minute après la connexion).
- [ ] Le bouton **Se déconnecter** coupe l'accès aux `.t2c`, puis **Se connecter** le rétablit.
- [ ] Tes autres sites (hors `.t2c`) fonctionnent normalement pendant que l'application tourne, et après l'avoir quittée.

**Aider le réseau** (carte **Bénévolat**)
- [ ] La case **Relais caché** est cochée et indique, après une minute, « relié à … relais officiels ».
- [ ] Coche un site à héberger, enregistre : il apparaît dans « sites hébergés ».
- [ ] Décoche ce que tu veux : tout s'arrête sans erreur.

**Si tu utilises T2DECODE** (facultatif)
- [ ] Dans ses réglages, active **Cours depuis T2C Phantom**, puis redémarre T2DECODE pendant que T2C Phantom tourne : il doit proposer de redémarrer avec les nouveaux cours.

## 5. Ce que l'application fait sur ton ordinateur

- Le **certificat** installé ne peut servir qu'aux adresses `.t2c` : impossible de s'en servir pour imiter un autre site.
- Le **proxy** ne concerne que les `.t2c`.
- Le **relais caché** transporte des données chiffrées pour d'autres membres, au milieu de leurs circuits : il ne voit ni qui ils sont ni ce qu'ils lisent, ton adresse n'est publiée nulle part, et son débit est plafonné (128 Ko/s par défaut). Tu peux le désactiver dans **Bénévolat**.
- Tout ce que tu reçois est signé par TUTODECODE et vérifié : un contenu modifié est refusé.

## 6. Signaler un problème

Écris à **contact@tutodecode.org**, objet **`[Test T2C v2.6.0]`**, avec :

```
Système : (macOS 15, Windows 11, Ubuntu 24.04…)
Navigateur : (Firefox, Chrome, Safari…)
Étape : (installation, assistant, site tutodecode.t2c, bénévolat…)
Ce que j'ai fait :
Ce qui s'est passé :
Ce que j'attendais :
Message d'erreur exact (s'il y en a un) :
```

Joins une **capture d'écran** si possible. N'envoie jamais de mot de passe ni de fichier de ton dossier personnel.

Même si tout marche, un petit message « tout est OK sur Windows 11 + Firefox » nous aide beaucoup.

## 7. Désinstaller

Dans l'application : **Réglages → Désactiver le proxy**, puis quitte-la et supprime-la. Le détail (certificat, données locales) est dans le [Guide complet](GUIDE.md#13-désinstaller).
