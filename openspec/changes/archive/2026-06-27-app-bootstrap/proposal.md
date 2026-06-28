## Why

Il n'existe pas d'interface visuelle pour suivre l'avancement des changements OpenSpec à travers les projets. OpenSP8C comble ce manque en offrant un Kanban Board couplé à Claude Code pour piloter le workflow OpenSpec sans quitter l'interface.

## What Changes

- Création d'une application complète (backend Go + frontend React) à partir de zéro
- Interface Kanban Board avec 4 colonnes : To Explore, To Do, In Progress, Done
- Gestion multi-workspace : chaque workspace est un répertoire de projet portant son propre `openspec/`
- Intégration Claude Code en subprocess pour la session d'exploration dans la colonne To Explore
- Visualisation des specs actées dans un onglet dédié
- Configuration des workspaces via `config.yaml` avec explorateur de fichiers

## Capabilities

### New Capabilities

- `workspace-management` : Ajout, sélection et persistance de workspaces (projets) via `config.yaml`
- `kanban-board` : Affichage et gestion des changements OpenSpec sous forme de Kanban (To Explore / To Do / In Progress / Done), avec drag & drop et transition de statut
- `explore-session` : Lancement de Claude Code en subprocess non-interactif pour le chat d'exploration, proxifié via WebSocket depuis le backend Go
- `specs-view` : Liste et affichage des spécifications actées (`openspec/specs/`) du workspace actif
- `change-archive` : Action d'archivage d'un changement Done depuis l'UI, sans interaction utilisateur, avec affichage du résultat dans le chat

### Modified Capabilities

## Impact

- Nouveau projet : aucun code existant à modifier
- Dépendance externe : binaire `claude` (Claude Code CLI) installé sur la machine hôte
- Dépendance externe : binaire `openspec` installé sur la machine hôte
- Backend expose une API REST + WebSocket ; frontend consomme cette API
- Le statut Kanban est stocké dans le champ `kanban_status` du fichier `.openspec.yaml` de chaque change
