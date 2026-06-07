<div align="center">
  <img src="https://raw.githubusercontent.com/TUTODECODE-FR/T2DECODE/main/assets/TDC.png" width="160" height="160" alt="T2C Logo">

  # T2C Phantom

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

> [!WARNING]
> **⚠️ Avertissement Légal & Éthique**
> T2C Phantom est un réseau P2P éducatif géré par l'association TUTODECODE. Son but est de garantir l'accès à la connaissance en s'affranchissant de la centralisation (en hébergeant un registre de paquets décentralisé). **T2C Phantom n'est PAS un outil conçu pour contourner illégalement des systèmes de sécurité ni pour être utilisé à des fins illégales.**

## ⚠️ PROPRIÉTÉ INTELLECTUELLE & LICENCE STRICTE

> [!CAUTION]
> **Ce dépôt est "Source-Available" (Auditable) mais n'est PAS Open Source.**
> Le code de T2C Phantom est maintenu sous propriété intellectuelle stricte (**Tous droits réservés**). 
> L'utilisation, la modification, la distribution ou l'intégration commerciale de ce moteur réseau par des entreprises, gouvernements, ou entités tierces est **formellement interdite** sans accord écrit explicite de TUTODECODE.

**Pourquoi cette protection absolue ? (Transparence Hybride)**
Nous adoptons une approche hybride pour la sécurité du réseau. La mécanique côté client est auditable pour prouver l'absence de portes dérobées. En revanche, **le code source du Relais et de nos serveurs est gardé strictement confidentiel**. Ce choix architectural vise à empêcher les acteurs malveillants d'étudier notre infrastructure pour créer des nœuds empoisonnés (Attaques Sybil) et préserver l'intégrité de la topologie du réseau T2C.

- ✔️ **Autorisé** : Lire, auditer, compiler pour un usage personnel et se connecter au réseau T2DECODE.
- ❌ **Interdit** : Copier le code pour des projets tiers, le revendre, le déployer dans un réseau d'entreprise privé ou institutionnel.

<img src="https://raw.githubusercontent.com/TUTODECODE-FR/T2DECODE/main/assets/separator.svg" width="100%" height="4">

## 🧠 L'Architecture P2P & Modèle Théorique

T2C Phantom repose sur un modèle d'architecture décentralisée (Swarm/Mesh) garantissant l'accès à la connaissance sans point de défaillance unique. 

### 1. Le Modèle "Swarm" (Essaim)
Contrairement à une architecture Client-Serveur classique où tout le monde se connecte à un serveur central vulnérable (Cloud), T2C Phantom crée un **maillage**. Chaque utilisateur qui lance le daemon devient un "nœud" du réseau. L'application *T2DECODE* utilise silencieusement ce réseau en arrière-plan pour interroger l'essaim et télécharger les nouveaux cours disponibles de manière chiffrée. Plus il y a de nœuds, plus le réseau est rapide et résilient.

### 2. Résilience en Milieux Restreints (DPI Evasion & WSS)
Pour garantir un accès continu dans les environnements restrictifs (pare-feux d'entreprises, réseaux universitaires, zones censurées), T2C Phantom utilise des techniques d'évasion de pointe :
- **Circuit Relay v2** : Permet de contourner les restrictions NAT en s'appuyant sur des nœuds relais communautaires pour faire transiter le trafic de manière anonyme.
- **WebSocket Secure (WSS)** : T2C Phantom encapsule son protocole P2P binaire à l'intérieur d'un flux HTTPS standard (Port 443). Pour les équipements d'inspection (DPI), le trafic ressemble à une navigation web chiffrée classique, le rendant intraçable et impossible à bloquer sans couper tout le trafic HTTPS.

### 3. "Chiffrement Emboîté" (Zero-Trust)
Le réseau applique un chiffrement absolu de bout-en-bout (Protocole Noise / AES-GCM). Même si le trafic transite par des nœuds relais, ceux-ci n'ont aucun accès au contenu. Ils agissent comme des routeurs aveugles.

### 4. Annuaire Décentralisé (Kademlia DHT)
Il n'y a **aucun serveur central d'adressage**. Pour localiser les données, le réseau interroge une table de hachage distribuée (DHT) mondiale. Pour l'expérience utilisateur, T2C Phantom intègre un résolveur DNS décentralisé (le fichier `carnet.json`), permettant de traduire des noms humains (ex: `tutodecode.t2c`) en adresses cryptographiques.

<img src="https://raw.githubusercontent.com/TUTODECODE-FR/T2DECODE/main/assets/separator.svg" width="100%" height="4">

## 🚀 Guide Rapide : Post-Téléchargement

Le binaire est conçu pour être 100% autonome et sans dépendances (Air-Gapped ready). Une fois téléchargé depuis la section **Releases**, voici comment l'utiliser :

### 1. Démarrer le Nœud Local
Ouvrez votre terminal dans le dossier contenant l'exécutable et lancez le démon. Pour vous connecter au réseau en contournant les pare-feux, utilisez notre Relais Public Officiel :

```bash
# Sur Mac/Linux :
./t2c start --proxy 8080 --use-relay /ip4/20.244.4.32/tcp/443/wss/p2p/12D3KooWBEK7H1KXCgpUbD4pFeGJVjVoFfUhnwPH1ywvsTbn6BBr

# Sur Windows :
t2c.exe start --proxy 8080 --use-relay /ip4/20.244.4.32/tcp/443/wss/p2p/12D3KooWBEK7H1KXCgpUbD4pFeGJVjVoFfUhnwPH1ywvsTbn6BBr
```
*Le nœud démarrera instantanément, générera une identité éphémère (si aucune clé n'est fournie), et se connectera silencieusement à l'essaim P2P.*

### 2. Naviguer sur le réseau "Fantôme"
Laissez le terminal tourner en arrière-plan. Votre machine dispose désormais d'un accès au sous-réseau T2C via le proxy local `127.0.0.1:8080`. 

Si vous disposez du fichier `carnet.json` agissant comme annuaire DNS, vous pouvez interroger les nœuds du réseau de manière transparente :
```bash
curl -x http://localhost:8080 http://tutodecode.t2c
```
*Le trafic traversera vos pare-feux locaux de manière chiffrée, rebondira sur le relais, et atteindra sa destination P2P de manière intraçable.*

<img src="https://raw.githubusercontent.com/TUTODECODE-FR/T2DECODE/main/assets/separator.svg" width="100%" height="4">

## ✨ Dashboard d'Administration Local

Le Dashboard local (`http://127.0.0.1:9090`) a été entièrement repensé avec une interface haut de gamme "Noir & Beige", fidèle au design system strict de T2DECODE.

*   🎛️ **Options de Confidentialité Avancées** : Choisissez de masquer votre IP via des relais communautaires ou de router votre trafic anonymement.
*   🤝 **Réseau Pair-à-Pair (P2P)** : Suivez en temps réel la connexion au Swarm pour assurer la robustesse du réseau.
*   🛡️ **Métriques en Temps Réel** : Suivi de l'état cryptographique, analyse de connexion, et diagnostic des pare-feux (DPI).

<img src="https://raw.githubusercontent.com/TUTODECODE-FR/T2DECODE/main/assets/separator.svg" width="100%" height="4">

## 🔐 Conformité & Sécurité

La transparence est la fondation de la confiance. Bien que ce code soit protégé par le droit de la propriété intellectuelle, nous l'ouvrons publiquement à l'audit.

- 🛡️ **Politique de Sécurité et Modèle de Menace** : Consultez notre [SECURITY.md](SECURITY.md) pour comprendre de quoi (et de qui) T2C Phantom vous protège.
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
