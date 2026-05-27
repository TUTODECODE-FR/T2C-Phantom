<div align="center">
  <img src="https://raw.githubusercontent.com/TUTODECODE-FR/T2DECODE/main/assets/TDC.png" width="160" height="160" alt="T2C Logo">

  # T2C Phantom
  
  **« Le savoir ne devrait pas toujours dépendre d'une connexion... ni d'un censeur. »**<br>
  — *Maxime MARTIN CIVET*

  [![Go](https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white&labelColor=1A1D2E)]()
  [![libp2p](https://img.shields.io/badge/libp2p-000000?style=for-the-badge&logo=ipfs&logoColor=white&labelColor=1A1D2E)]()
  [![Proprietary](https://img.shields.io/badge/License-All_Rights_Reserved-D93025?style=for-the-badge&labelColor=1A1D2E)]()
  [![Privacy By Design](https://img.shields.io/badge/Privacy-By%20Design-3DDC84?style=for-the-badge&labelColor=1A1D2E&color=3DDC84)]()
  
  <br>
  <p>
    <b>Client furtif pour le réseau décentralisé T2DECODE (Overlay Network).</b><br>
    <i>Chiffrement de bout en bout · DHT Kademlia · Zéro Serveur Central · Zéro Télémétrie</i>
  </p>
  <br>
</div>

<img src="https://raw.githubusercontent.com/TUTODECODE-FR/T2DECODE/main/assets/separator.svg" width="100%" height="4">

## 🎯 Raison d'Être de T2C Phantom

T2C Phantom est un **client de connexion réseau souverain** permettant d'accéder à des ressources éducatives et des services cachés hébergés sur le réseau P2P décentralisé T2DECODE.

Ce logiciel établit un tunnel chiffré de bout-en-bout via le protocole Kademlia DHT (libp2p), garantissant l'anonymat et la résistance à la censure.

<img src="https://raw.githubusercontent.com/TUTODECODE-FR/T2DECODE/main/assets/separator.svg" width="100%" height="4">

## 📥 Téléchargement et Installation Plug & Play

Pour une expérience sans configuration, **téléchargez directement le script d'installation spécial pour votre système** depuis la section "Releases" :
- **Windows** : Téléchargez et lancez `install_windows.bat`.
- **macOS / Linux** : Téléchargez et exécutez `install_mac_linux.sh`.
- **Android** : La librairie native `t2c-android.aar` est générée automatiquement. (Elle est conçue pour être importée dans notre APK Android complet).

Ces scripts se chargeront de :
1. Télécharger le bon exécutable silencieusement.
2. **Télécharger le fichier le plus important du réseau : `carnet.json`.** (Ce fichier contient l'annuaire décentralisé pour trouver les ressources).
3. Lancer l'application et ouvrir automatiquement votre navigateur sur **http://127.0.0.1:9090**.

Si vous souhaitez le lancer manuellement, vous pouvez télécharger les binaires, vous assurer que le `carnet.json` est placé dans le dossier `data/`, et exécuter :
```bash
./t2c dashboard
```

<img src="https://raw.githubusercontent.com/TUTODECODE-FR/T2DECODE/main/assets/separator.svg" width="100%" height="4">

## ⚠️ Avertissement Légal & Propriété Intellectuelle

Ce code source est publié publiquement **uniquement à des fins de transparence, d'audit de sécurité et de démonstration technique**. 

**Tous les droits sont réservés (All Rights Reserved).** 
La licence de ce dépôt N'EST PAS une licence open-source (ni GPL, ni MIT, ni Apache). 

Vous êtes autorisé à :
- Lire et auditer le code source.
- Télécharger et exécuter les binaires fournis pour accéder au réseau T2DECODE.

Vous n'êtes **PAS** autorisé à :
- Copier, modifier, ou créer des œuvres dérivées de ce code.
- Redistribuer ou publier ce code (même modifié) sur un autre dépôt ou site web.
- Utiliser cette technologie à des fins commerciales ou pour créer votre propre réseau concurrent.

*Toute violation de ces conditions entraînera des poursuites pour violation de droits d'auteur.*

<img src="https://raw.githubusercontent.com/TUTODECODE-FR/T2DECODE/main/assets/separator.svg" width="100%" height="4">

## 🏛️ L'Association TUTODECODE (Mentions Légales)

Le projet T2DECODE et le réseau T2C Phantom sont développés et soutenus par l'**Association TUTODECODE**, structure relevant de l'Économie Sociale et Solidaire (ESS).  
Notre mission est de démocratiser la maîtrise des infrastructures informatiques et de la cybersécurité défensive en fournissant des outils souverains, auditables et respectueux de la vie privée.

Dans une démarche absolue de transparence et de rigueur, l'association publie ses identifiants légaux officiels :

- **Éditeur** : Association Loi 1901 TUTODECODE
- **Directeur de Publication** : Maxime MARTIN CIVET
- **SIREN** : 102 763 133
- **Site Web Officiel** : [https://tutodecode.org](https://tutodecode.org)
- **Preuve Légale** : [Annonce de création parue au Journal Officiel de la République Française (JOAFE)](https://www.journal-officiel.gouv.fr/pages/associations-detail-annonce/?q.id=id:202600110336)

### 💖 Soutien Financier (Dons)
Si nos outils gratuits et sans publicité vous font gagner du temps ou enrichissent votre parcours professionnel, vous pouvez soutenir l'association TUTODECODE. Les dons servent exclusivement à pérenniser l'hébergement de nos services, le maintien des noms de domaine et la continuité de nos actions éducatives.
- ➡️ **[Faire un don sécurisé à l'association via HelloAsso](https://www.helloasso.com/associations/tutodecode)**

---
© 2026 TUTODECODE - Tous droits réservés.
