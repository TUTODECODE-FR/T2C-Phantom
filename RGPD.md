# Données personnelles et vie privée

Ce document décrit, de façon factuelle, quelles données T2C Phantom traite, où, et pourquoi. Il découle directement du fonctionnement du logiciel décrit dans le [Guide complet](docs/GUIDE.md).

**Responsable** : Association TUTODECODE — `contact@tutodecode.org`.

## Ce que TUTODECODE ne collecte pas

- **Aucun compte**, aucun nom, aucune adresse e-mail, aucun numéro de téléphone.
- **Aucune adresse IP enregistrée** : les relais et le serveur d'origine n'écrivent aucune adresse IP dans leurs journaux, et le serveur d'origine refuse toute requête qui en porterait une.
- **Aucun historique de navigation** conservé côté réseau.
- **Aucun traceur, aucune publicité, aucune mesure d'audience** dans l'application.

## Données traitées

| Donnée | Où | Pourquoi | Durée |
|---|---|---|---|
| **Adresse IP** de l'appareil | Point d'entrée du réseau (Cloudflare, ou le relais principal tant que le nom de domaine n'est pas actif) | Établir la connexion : toute connexion Internet révèle l'adresse de celui qui l'ouvre | Le temps de la connexion ; non enregistrée par TUTODECODE. Cloudflare applique sa propre politique de confidentialité. |
| **Clé pseudonyme** (identifiant aléatoire de 32 caractères) | Ton appareil ; vue par le relais auquel tu te connectes | Te reconnaître d'une connexion à l'autre sans savoir qui tu es | Tant que tu gardes `identity.json` ; la supprimer crée une identité neuve sans lien avec l'ancienne |
| **Sites demandés** | Relais (en mode normal) ; relais de sortie, sous un pseudonyme jetable (en mode anonyme) | Te fournir le site | Non conservés |
| **Bénévolat** : clé pseudonyme, sites hébergés, score | Liste signée publique `hosts.json` | Permettre aux membres de choisir des bénévoles fiables | Tant que tu es bénévole ; retiré dès l'arrêt du bénévolat ou l'expiration du ticket |
| **Données locales** : identité, certificat local `.t2c`, copies hébergées (chiffrées), sites vérifiés, réglages | Uniquement sur ton appareil | Fonctionnement de l'application | Jusqu'à leur suppression par toi ([Guide, section 11](docs/GUIDE.md#11-données-locales)) |

## Tes droits

TUTODECODE ne détient aucune donnée permettant de t'identifier : il n'y a donc pas de fiche à consulter, rectifier ou effacer chez l'association. Tu gardes la maîtrise complète de tes données locales : désinstaller l'application et supprimer son dossier de données efface tout ([Guide, section 13](docs/GUIDE.md#13-désinstaller)).

Pour toute question sur ce document : `contact@tutodecode.org`.

## Services tiers

- **Cloudflare** : point d'entrée du réseau (protection contre les attaques par saturation). Il voit l'adresse IP de connexion.
- **GitHub et GitLab** : hébergement du code et des fichiers à télécharger. Le téléchargement depuis ces plateformes est soumis à leurs propres conditions.
