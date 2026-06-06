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
- ❌ **Interdit** : Copier le code pour des projets tiers, le revendre, le déployer dans un réseau d'entreprise privé ou institutionnel.

<img src="https://raw.githubusercontent.com/TUTODECODE-FR/T2DECODE/main/assets/separator.svg" width="100%" height="4">

## 🧠 Comment ça fonctionne ? (L'Architecture T2C)

Pour comprendre T2C Phantom, il faut comprendre notre philosophie : **Le Sous-marin et la Vitrine**.

### 1. La Vitrine (T2DECODE)
T2DECODE est l'application éducative visible par l'utilisateur. Elle contient l'IA locale (Ollama), les simulateurs de hacking, les cours interactifs et l'interface graphique.

### 2. Le Sous-marin (T2C Phantom)
Ce dépôt. C'est le moteur réseau `libp2p` en Go qui tourne silencieusement en arrière-plan. Son rôle est unique : **connecter des machines qui ne sont pas censées pouvoir se connecter.**

#### A. Le contournement de la censure (DPI Evasion & NAT Traversal)
Aujourd'hui, les réseaux sont verrouillés (pare-feux d'entreprise, box internet, routeurs stricts). T2C Phantom utilise des techniques d'évasion avancées :
- **Circuit Relay v2** : Si deux ordinateurs sont bloqués par un pare-feu, ils se rejoignent sur un Relais public Azure.
- **WebSocket Secure (WSS)** : Pour passer les pare-feux Deep Packet Inspection (DPI) très agressifs, T2C Phantom déguise le trafic P2P en trafic web HTTPS standard (Port 443). Le pare-feu n'y voit que du feu.

#### B. Le "Chiffrement Emboîté" (Sécurité absolue)
T2C Phantom utilise un modèle Zero-Trust. Même si le trafic passe par notre Relais Azure, **Microsoft ne peut rien lire**.
1. **La couche externe (HTTPS)** : Le pare-feu voit un trafic web sécurisé.
2. **La couche interne (Protocole Noise)** : Les données à l'intérieur du tunnel HTTPS sont chiffrées une seconde fois de bout-en-bout avec des clés privées générées uniquement sur votre machine. Le Relais Azure n'est qu'un "tuyau opaque".

#### C. Annuaire Décentralisé (Kademlia DHT)
Il n'y a **aucun serveur central** qui liste les sites. Pour trouver un pair, T2C Phantom interroge le réseau mondial (DHT). 
Pour faciliter l'usage humain, T2C utilise un fichier local `carnet.json` agissant comme un résolveur DNS décentralisé (ex: `tutodecode.t2c` -> `izm3xft3...t2c`).

<img src="https://raw.githubusercontent.com/TUTODECODE-FR/T2DECODE/main/assets/separator.svg" width="100%" height="4">

## 🚀 Installation Plug & Play

Oubliez les configurations complexes ! Rendez-vous dans la section **Releases** de ce dépôt et téléchargez l'exécutable pour votre plateforme.

### ⚙️ Sous le capot (En 3 secondes chrono)
1. ✅ **Déploiement Automatique** : Téléchargement du binaire Go ultra-léger pour votre architecture.
2. ✅ **Bootstrapping DHT** : Récupération automatique du fichier `carnet.json` (l'annuaire décentralisé P2P).
3. ✅ **Connexion Furtive** : Établissement du tunnel chiffré via notre technologie Circuit Relay v2 (NAT/DPI Traversal).

<img src="https://raw.githubusercontent.com/TUTODECODE-FR/T2DECODE/main/assets/separator.svg" width="100%" height="4">

## 💻 Comment Utiliser T2C Phantom ?

Une fois le binaire téléchargé, ouvrez votre terminal (ou invite de commande) dans le dossier du programme.

### 1. Lancer le tunnel furtif (Le Proxy Local)
Pour percer un pare-feu d'entreprise ou de réseau public, exécutez la commande suivante. Elle ouvre un proxy local (sur le port 8080) et masque votre trafic via le Relais Azure :
```bash
# Sur Mac/Linux :
./t2c start --proxy 8080 --use-relay /ip4/20.244.4.32/tcp/443/wss/p2p/12D3KooWBEK7H1KXCgpUbD4pFeGJVjVoFfUhnwPH1ywvsTbn6BBr

# Sur Windows :
t2c.exe start --proxy 8080 --use-relay /ip4/20.244.4.32/tcp/443/wss/p2p/12D3KooWBEK7H1KXCgpUbD4pFeGJVjVoFfUhnwPH1ywvsTbn6BBr
```

### 2. Naviguer sur le réseau "Fantôme"
Laissez le terminal ouvert. Dans un autre terminal, ou dans la configuration proxy de votre navigateur, vous pouvez maintenant accéder aux sites P2P cachés (les adresses `.t2c`) !

Si vous possédez le fichier `carnet.json` fourni (qui agit comme un annuaire DNS), vous pouvez taper directement les noms lisibles :
```bash
curl -x http://localhost:8080 http://tutodecode.t2c
```
*Le trafic passera de votre ordinateur, au Relais Azure chiffré (pour esquiver la censure), jusqu'au serveur privé de destination (Le "Sous-marin").*

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
