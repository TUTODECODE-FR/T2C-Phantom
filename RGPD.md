# Données personnelles et vie privée

Ce document décrit, de façon factuelle, quelles données T2C Phantom traite, où, et pourquoi. Il découle directement du fonctionnement du logiciel décrit dans le [Guide complet](docs/GUIDE.md).

**Responsable** : Association TUTODECODE — `contact@tutodecode.org`.

## Ce que TUTODECODE ne collecte pas

- **Aucun compte**, aucun nom, aucune adresse e-mail, aucun numéro de téléphone.
- **Aucune adresse IP enregistrée** : les relais de l'association et le serveur d'origine n'écrivent aucune adresse IP dans leurs journaux, et le serveur d'origine refuse toute requête qui en porterait une. Les relais d'entrée (et leurs hébergeurs ou tunnels, voir plus bas) voient en revanche ton adresse pendant la connexion.
- **Aucun historique de navigation** conservé côté réseau.
- **Aucun traceur, aucune publicité, aucune mesure d'audience** dans l'application.

## Données traitées

| Donnée | Où | Pourquoi | Durée |
|---|---|---|---|
| **Adresse IP** de l'appareil | Relais d'entrée : relais de l'association (VPS chez Aquilenet, relais maison joint par Tailscale Funnel), ou ta garde bénévole en mode Partagé (voir plus bas) ; et les tiers listés ci-dessous | Établir la connexion : toute connexion Internet révèle l'adresse de celui qui l'ouvre | Le temps de la connexion ; jamais enregistrée par TUTODECODE. Les tiers appliquent leur propre politique |
| **Clé pseudonyme** (identifiant aléatoire de 32 caractères) | Ton appareil ; vue par le relais auquel tu te connectes | Te reconnaître d'une connexion à l'autre sans savoir qui tu es | Tant que tu gardes `identity.json` ; la supprimer crée une identité neuve sans lien avec l'ancienne |
| **Sites demandés** | Hors mode anonyme : le relais. En mode anonyme : le relais de **sortie** (bénévole en Partagé, association en Maximal), sous un pseudonyme jetable, sans ton adresse | Te fournir le site | Non conservés |
| **Version de l'application** | Relais auxquels tu te connectes | Refuser les versions trop anciennes (avis de version signé) | Non conservée |
| **Bénévolat** : clé pseudonyme, sites hébergés, score | Liste signée publique `hosts.json` | Permettre aux membres de choisir des bénévoles fiables | Tant que tu es bénévole ; retiré dès l'arrêt du bénévolat ou l'expiration du ticket |
| **Données locales** : identité, certificat local `.t2c` et sa clé, copies hébergées (chiffrées), sites vérifiés, réglages, journal de sécurité local (`security.log`, peut contenir des noms `.t2c` demandés) | Uniquement sur ton appareil | Fonctionnement de l'application | Jusqu'à leur suppression par toi ([Guide, section 11](docs/GUIDE.md#11-données-locales)) |

Base légale : intérêt légitime de l'association à faire fonctionner le service que tu as installé, au minimum de données nécessaire. Tu peux saisir la CNIL (`www.cnil.fr`) si tu estimes que tes droits ne sont pas respectés.

## Ce qui est activé par défaut (et comment le couper)

Dans l'application, **Réglages → Bénévolat / Anonymat** :

| Réglage | Par défaut | Ce que cela implique | Pour le couper |
|---|---|---|---|
| **Relais caché** (milieu des circuits des autres membres) | activé | ton application relaie des cellules chiffrées pour d'autres membres, avec un plafond de débit ; elle ne voit ni leur adresse ni leurs demandes, et n'ouvre aucun port | décocher « Relais caché » |
| **Être sortie de circuit** | activé (si le relais caché l'est) | ton application transmet les demandes d'autres membres (sites `.t2c` signés uniquement), sous pseudonyme jetable, sans connaître leur adresse | décocher « Accepter d'être sortie » |
| **Entrer par des relais bénévoles** (mode Partagé) | activé | une garde bénévole voit ton adresse IP (jamais tes demandes) | décocher la case, ou choisir le mode **Maximal** |
| **Relais d'entrée** (ta propre adresse ou un VPN) | **désactivé** | si tu l'actives, les membres qui entrent par toi te montrent leur adresse IP, et la tienne (ou celle du VPN) est publiée dans l'annuaire | — |
| **Héberger des sites** | désactivé | ta clé et les sites hébergés apparaissent dans la liste publique des bénévoles | — |

## Tes droits

TUTODECODE ne détient aucune donnée permettant de t'identifier : il n'y a donc pas de fiche à consulter, rectifier ou effacer chez l'association. Tu gardes la maîtrise complète de tes données locales : désinstaller l'application et supprimer son dossier de données efface tout ([Guide, section 13](docs/GUIDE.md#13-désinstaller)).

Pour toute question sur ce document : `contact@tutodecode.org`.

## Destinataires et services tiers

| Tiers | Ce qu'il voit | Où |
|---|---|---|
| **Relais de l'association** (VPS, relais maison) | ton adresse IP quand tu entres par eux ; tes demandes (sous pseudonyme jetable) quand ils sont la sortie | France |
| **Aquilenet** (association, hébergeur du VPS) | les adresses IP qui se connectent au VPS, au niveau réseau ; le contenu est chiffré (TLS) | France (UE) |
| **Tailscale** (Tailscale Funnel, porte d'entrée publique du relais maison) | ton adresse IP et le nom `*.ts.net` demandé ; le contenu est chiffré jusqu'au relais | **hors UE** (société canadienne, infrastructure mondiale) : transfert de données hors de l'Union européenne |
| **sslip.io** (service DNS du nom `185-233-100-85.sslip.io`) | l'adresse de ton résolveur DNS (parfois une partie de la tienne) et le fait que le nom du relais est recherché | hors UE possible |
| **Cloudflare** (prévu devant le VPS, **pas encore en place**) | une fois en place : ton adresse IP, et en mode anonyme seulement des cellules chiffrées | — |
| **Bénévoles** (gardes d'entrée, relais du milieu et de sortie, hébergeurs) | selon leur rôle : ton adresse IP (garde d'entrée, en Partagé), tes demandes sous pseudonyme jetable (sortie), ou des cellules chiffrées seulement | partout |
| **GitHub et GitLab** | hébergement du code et des fichiers à télécharger, selon leurs propres conditions | hors UE |

Le détail, acteur par acteur et mode par mode, est dans [SECURITY.md](SECURITY.md#qui-voit-quoi-état-actuel-du-réseau).
