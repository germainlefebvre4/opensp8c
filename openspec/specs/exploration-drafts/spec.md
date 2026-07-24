## Purpose

Gérer la persistance isolée et le cycle de vie complet des brouillons d'exploration côté backend (stockage isolé sous `drafts/<ghostId>.json`, API CRUD) et côté frontend (vue split-screen interactive, édition des tâches à la volée, bouton "Sauvegarder le brouillon").

## Requirements

### Requirement: Stockage isolé des brouillons côté backend
Le backend SHALL créer et administrer un sous-répertoire `drafts/` situé au même niveau que `preferences.json`. Ce répertoire contiendra les brouillons d'exploration transitoires sous la forme `drafts/<ghostId>.json`.

#### Scenario: Dossier drafts créé si absent
- **WHEN** le backend démarre ou tente d'écrire un brouillon alors que le dossier `drafts/` n'existe pas
- **THEN** le backend crée automatiquement le dossier `drafts/` de manière récursive sans erreur

#### Scenario: Format JSON du fichier brouillon
- **WHEN** un brouillon est écrit sur le disque dans `drafts/<ghostId>.json`
- **THEN** son format respecte exactement le schéma `{ "ghostId": string, "workspaceId": string, "name": string, "description": string, "tasks": Array<{ "id": string, "text": string, "done": boolean }>, "lastSavedAt": string }`

### Requirement: API CRUD pour les brouillons d'exploration
Le backend SHALL exposer un ensemble d'endpoints HTTP pour permettre au frontend de manipuler de façon autonome les brouillons associés aux Ghost Cards d'un workspace actif.

#### Scenario: Récupération de brouillon existant (GET)
- **WHEN** `GET /api/workspaces/{id}/explorations/{ghostId}/draft` est reçu ET que le fichier `drafts/<ghostId>.json` existe
- **THEN** le serveur retourne un code 200 OK avec le contenu JSON complet du brouillon

#### Scenario: Récupération de brouillon inexistant (GET)
- **WHEN** `GET /api/workspaces/{id}/explorations/{ghostId}/draft` est reçu ET que le fichier `drafts/<ghostId>.json` n'existe pas
- **THEN** le serveur retourne un code 200 OK avec une structure de brouillon par défaut vide : `{ "ghostId": "<ghostId>", "workspaceId": "<workspaceId>", "name": "", "description": "", "tasks": [] }`

#### Scenario: Sauvegarde ou mise à jour de brouillon (PUT)
- **WHEN** `PUT /api/workspaces/{id}/explorations/{ghostId}/draft` est reçu avec un payload JSON valide dans le body
- **THEN** le backend écrit ou remplace le fichier `drafts/<ghostId>.json` avec ce payload, met à jour le champ `lastSavedAt` à l'heure actuelle, et retourne un code 200 OK

#### Scenario: Suppression du brouillon (DELETE)
- **WHEN** `DELETE /api/workspaces/{id}/explorations/{ghostId}/draft` est reçu
- **THEN** le backend supprime physiquement le fichier `drafts/<ghostId>.json` s'il existe et retourne un code 204 No Content

### Requirement: Détection automatique des tâches dans le chat et émission d'événement
Le subprocess de session d'exploration (Claude/Gemini) SHALL pouvoir émettre un marqueur d'événement structuré `{"event":"ghost_draft_updated","draft":{...}}` sur stdout. Le backend SHALL détecter ce marqueur dans le flux, sauvegarder les tâches de brouillon reçues dans le fichier `drafts/<ghostId>.json`, et diffuser l'événement aux clients abonnés via le flux SSE (`/api/workspaces/{id}/events`).

#### Scenario: Détection du marqueur ghost_draft_updated
- **WHEN** le stdout du subprocess d'une session anonyme émet une ligne JSON contenant `"event":"ghost_draft_updated"`
- **THEN** le backend parse la trame, écrit le nouveau brouillon sous `drafts/<ghostId>.json`, et diffuse un événement SSE de type `draft_updated` avec les détails du brouillon au frontend

### Requirement: Panneau de brouillon interactif split-screen côté frontend
L'interface de chat d'exploration (`ExplorePanel`) SHALL proposer un affichage split-screen (panneau double) : la conversation à gauche, et le panneau de brouillon de tâche interactif à droite.

#### Scenario: Réception de draft_updated met à jour le panneau latéral
- **WHEN** le frontend reçoit un événement SSE `draft_updated` ou des informations d'un brouillon actualisé via WebSocket
- **THEN** le panneau de brouillon à droite affiche instantanément la description mise à jour et la liste des tâches déduites avec des cases à cocher

#### Scenario: Édition manuelle des tâches par l'utilisateur
- **WHEN** l'utilisateur ajoute, supprime, renomme ou coche une tâche de brouillon dans le panneau latéral droit
- **THEN** l'état local du brouillon est mis à jour et un appel `PUT` vers le endpoint d'enregistrement est automatiquement déclenché (avec débouncing de 500ms) pour persister l'action côté serveur

#### Scenario: Enregistrement manuel via bouton de sauvegarde
- **WHEN** l'utilisateur clique sur le bouton "Sauvegarder le brouillon" dans le panneau latéral
- **THEN** une requête `PUT` immédiate est émise vers l'API, et un indicateur "Brouillon sauvegardé" s'affiche temporairement
