<div align="center">
  <img src="https://raw.githubusercontent.com/TUTODECODE-FR/T2DECODE/main/assets/TDC.png" width="160" height="160" alt="T2C Logo">

  # T2C Phantom (Client)

  **« Le savoir ne devrait pas toujours dépendre d'une connexion. »**<br>
  — *Maxime MARTIN CIVET*

  [![Go](https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white&labelColor=1A1D2E)]()
  [![libp2p](https://img.shields.io/badge/libp2p-000000?style=for-the-badge&logo=ipfs&logoColor=white&labelColor=1A1D2E)]()
  [![Proprietary](https://img.shields.io/badge/License-All_Rights_Reserved-D93025?style=for-the-badge&labelColor=1A1D2E)]()
  [![Privacy By Design](https://img.shields.io/badge/Privacy-By%20Design-3DDC84?style=for-the-badge&labelColor=1A1D2E&color=3DDC84)]()
  
  <br>
  <p>
    <b>Le Connecteur Furtif vers l'Écosystème Éducatif T2DECODE.</b><br>
    <i>DPI Evasion · Kademlia DHT · Zéro Serveur Central · Air-Gapped Ready</i>
  </p>
  <br>
</div>

<img src="https://raw.githubusercontent.com/TUTODECODE-FR/T2DECODE/main/assets/separator.svg" width="100%" height="4">

## ⚠️ PROPRIÉTÉ INTELLECTUELLE & LICENCE STRICTE

> [!CAUTION]
> **Ce dépôt est "Source-Available" (Auditable) mais n'est PAS Open Source.**
> Le code de T2C Phantom est maintenu sous propriété intellectuelle stricte (**Tous droits réservés**). 
> L'utilisation, la modification, la distribution ou l'intégration commerciale de ce moteur réseau par des entreprises, gouvernements, ou entités tierces est **formellement interdite** sans accord écrit explicite de TUTODECODE.

**Pourquoi cette protection absolue ?**
Nous ouvrons ce code pour garantir une **transparence totale**. N'importe quel expert en cybersécurité peut auditer ce client pour s'assurer de l'absence de portes dérobées (backdoors) et de l'intégrité du chiffrement. Cependant, l'architecture réseau développée ici a nécessité une R&D massive pour traverser les pare-feux et censures les plus extrêmes. Cette licence vise à empêcher le vol industriel par de grandes entreprises technologiques tout en protégeant les utilisateurs d'infrastructures frauduleuses clonées.

- ✔️ **Autorisé** : Lire, auditer, compiler pour un usage personnel et se connecter au réseau T2DECODE.
- ❌ **Interdit** : Copier le code pour des projets tiers, le revendre, le déployer dans un réseau d'entreprise privé.

<img src="https://raw.githubusercontent.com/TUTODECODE-FR/T2DECODE/main/assets/separator.svg" width="100%" height="4">

## 🎯 Le Concept : "Le Sous-marin et la Vitrine"

L'écosystème **TUTODECODE** est divisé en deux entités distinctes pour une sécurité maximale :

1. **T2DECODE (La Vitrine)** : L'application éducative visuelle. Elle contient l'IA locale (Ollama), les simulateurs de hacking, les cours interactifs et l'interface utilisateur.
2. **T2C Phantom (Le Sous-marin)** : Ce dépôt. C'est un moteur réseau `libp2p` en Go qui tourne silencieusement en arrière-plan.

**Le rôle de ce Client T2C Phantom :**
Si l'application T2DECODE a besoin de télécharger des mises à jour, des cours ou de discuter en Peer-to-Peer, elle passe par T2C Phantom. Phantom se charge alors de contourner la censure de votre réseau (via TCP natif, WebSocket ou QUIC), de trouver d'autres pairs de manière décentralisée (Kademlia DHT) et de rapatrier les données cryptées en essaim (Swarm).

<img src="https://raw.githubusercontent.com/TUTODECODE-FR/T2DECODE/main/assets/separator.svg" width="100%" height="4">

## 🚀 Installation Plug & Play

Oubliez les configurations complexes ! Rendez-vous dans la section **Releases** de ce dépôt et exécutez le script adapté à votre plateforme en 1 clic :

- 🪟 **Windows** : Double-cliquez sur `install_windows.bat`
- 🍎 **macOS / 🐧 Linux** : Exécutez le script shell approprié.
- 📱 **Android** : La librairie native `t2c-android.aar` s'intègre directement à l'APK T2DECODE.

### ⚙️ Sous le capot (En 3 secondes chrono)
1. ✅ **Déploiement Automatique** : Téléchargement du binaire Go ultra-léger pour votre architecture.
2. ✅ **Bootstrapping DHT** : Récupération automatique du fichier `carnet.json` (l'annuaire décentralisé P2P).
3. ✅ **Connexion Furtive** : Établissement du tunnel chiffré via notre technologie Circuit Relay v2 (NAT/DPI Traversal).
4. ✅ **Lancement** : Ouverture automatique du Dashboard de contrôle sur **http://127.0.0.1:9090**.

<img src="https://raw.githubusercontent.com/TUTODECODE-FR/T2DECODE/main/assets/separator.svg" width="100%" height="4">

## ✨ Dashboard d'Administration Local

Le Dashboard local (`http://127.0.0.1:9090`) a été entièrement repensé avec une interface haut de gamme "Noir & Beige", fidèle au design system strict de T2DECODE.

*   🎛️ **Options de Confidentialité Avancées** : Choisissez de masquer votre IP via des relais communautaires ou de router votre trafic anonymement.
*   🤝 **Réseau Pair-à-Pair (P2P)** : Suivez en temps réel la connexion au Swarm pour assurer la robustesse du réseau.
*   🛡️ **Métriques en Temps Réel** : Suivi de l'état cryptographique, analyse de connexion, et diagnostic des pare-feux (DPI).

<img src="https://raw.githubusercontent.com/TUTODECODE-FR/T2DECODE/main/assets/separator.svg" width="100%" height="4">

## 🔐 Conformité & Sécurité

La transparence est la fondation de la confiance. Bien que ce code soit protégé par le droit de la propriété intellectuelle, nous l'ouvrons publiquement à l'audit.

- 🛡️ **Politique de Sécurité** : Consultez notre [SECURITY.md](SECURITY.md) pour notre approche des failles critiques.
- 📜 **Code de Conduite** : Nous appliquons le [Contributor Covenant 2.1](CODE_OF_CONDUCT.md).
- ⚖️ **Gouvernance** : Découvrez comment nous gérons le projet dans le [GOVERNANCE.md](GOVERNANCE.md).

<img src="https://raw.githubusercontent.com/TUTODECODE-FR/T2DECODE/main/assets/separator.svg" width="100%" height="4">

## 🏛️ L'Association TUTODECODE

Le projet est propulsé par l'**Association TUTODECODE** (Économie Sociale et Solidaire).  
Notre but ? Démocratiser l'informatique souveraine, la cybersécurité, et l'apprentissage respectueux de la vie privée.

- 🏢 **Éditeur** : Association Loi 1901 TUTODECODE
- 👤 **Directeur** : Maxime MARTIN CIVET
- 📄 **SIREN** : 102 763 133
- 🌐 **Site Web** : [tutodecode.org](https://tutodecode.org)
- ⚖️ **Preuve Légale** : [Annonce de création au JOAFE](https://www.journal-officiel.gouv.fr/pages/associations-detail-annonce/?q.id=id:202600110336)
- 💬 **Support** : Consultez le [SUPPORT.md](SUPPORT.md)

### 💖 Soutenez l'Indépendance Technologique !
Si nos outils gratuits, hors-ligne et sans publicité vous sont utiles, aidez-nous à financer notre infrastructure éducative :
- ☕ **[Faire un don sécurisé via HelloAsso](https://www.helloasso.com/associations/tutodecode)**

---
<div align="center">
  <i>© 2026 TUTODECODE - Tous droits réservés.</i>
</div>
