## Why

Lors des sessions d'exploration anonymes d'opensp8c (lancées via le bouton "+" du Kanban), l'utilisateur et l'IA échangent des idées et formalisent des tâches. Actuellement, ces tâches déduites restent sous forme de conversation informelle dans le chat et ne sont matérialisées sur le Kanban qu'après la promotion finale (qui crée les fichiers de change réels sur le disque du workspace).

Ce changement résout ce manque de visibilité intermédiaire en introduisant des "brouillons de tâches d'exploration" persistés de manière isolée au sein de l'écosystème opensp8c (sans polluer le dépôt Git du projet de l'utilisateur), et visualisables de façon dynamique sur la Ghost Card du Kanban ainsi que dans un panneau latéral interactif dédié au sein du chat.

## What Changes

- **Brouillons isolés d'exploration** : Stockage des brouillons de tâches et de descriptions au format JSON dans un sous-dossier privé `drafts/` d'opensp8c, à côté de `preferences.json`.
- **Nouveaux endpoints API d'exploration** : Endpoints pour récupérer, enregistrer ou supprimer les fichiers de brouillons d'une Ghost Card.
- **Panneau de brouillon live dans le Chat** : Ajout d'une vue split-screen dans le panel d'exploration montrant les tâches déduites ou modifiables par l'utilisateur.
- **Visualisation des tâches de brouillon sur le Kanban** : Affichage d'un compteur de tâches de brouillon ou d'une barre de progression pointillée sur les Ghost Cards dans la colonne *To Explore*.
- **Intégration avec la Promotion (Fast-Forward)** : Lors du déclenchement du promote, le contenu du brouillon JSON privé est utilisé pour peupler et générer automatiquement le fichier `tasks.md` final dans le projet de l'utilisateur.

## Capabilities

### New Capabilities
- `exploration-drafts`: Cette capability gère le cycle de vie complet des brouillons d'exploration côté backend (stockage isolé sous `drafts/<ghostId>.json`, API CRUD) et côté frontend (vue split-screen interactive, édition des tâches à la volée, bouton "Sauvegarder le brouillon").

### Modified Capabilities
- `explore-ghost-card`: Modification pour afficher les métadonnées de brouillon (nombre de tâches déduites) directement sur la carte de statut "Exploring..." dans la colonne "To Explore".
- `exploration-promote-to-change`: Modification du endpoint `/promote` pour consommer le fichier de brouillon `drafts/<ghostId>.json` lors de l'exécution de FF pour créer le `tasks.md` réel.

## Impact

- **Backend** :
  - Nouvel helper `draftsPath` et dossier `drafts/` gérés par le serveur d'opensp8c.
  - Endpoints `GET` / `PUT` / `DELETE` pour `/api/workspaces/{id}/explorations/{ghostId}/draft`.
  - Modification de `DeleteGhost` pour supprimer le fichier de brouillon associé.
  - Modification de `/promote` pour passer le brouillon de tâches au subprocess FF.
- **Frontend** :
  - Mise à jour de `ExplorePanel` pour supporter un mode d'affichage split-screen / panneau de brouillon interactif.
  - Nouveaux hooks et appels d'API d'enregistrement de brouillon.
  - Mise à jour de l'affichage de la Ghost Card dans `KanbanPage`.
