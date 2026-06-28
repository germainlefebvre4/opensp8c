## Purpose

Expose Kanban status counters per workspace, both via the API and as visual badges in the sidebar.

## Requirements

### Requirement: Compteurs Kanban dans la liste des workspaces
L'API `GET /api/workspaces` SHALL retourner pour chaque workspace un champ `task_counts` contenant le nombre de changes par statut Kanban (`to-explore`, `todo`, `in-progress`, `done`). Le calcul SHALL être effectué en lisant le répertoire `openspec/changes/` du workspace à chaque requête.

#### Scenario: Listing de workspaces avec des changes
- **WHEN** l'API reçoit une requête `GET /api/workspaces`
- **THEN** chaque workspace dans la réponse inclut un objet `task_counts` avec les quatre statuts et leur nombre respectif (0 si aucun change dans ce statut)

#### Scenario: Workspace sans changes
- **WHEN** un workspace n'a aucun change dans son répertoire `openspec/changes/`
- **THEN** `task_counts` retourne `{ "to-explore": 0, "todo": 0, "in-progress": 0, "done": 0 }`

#### Scenario: Répertoire changes inexistant
- **WHEN** le répertoire `openspec/changes/` n'existe pas dans le workspace
- **THEN** `task_counts` retourne tous les compteurs à 0 sans erreur

### Requirement: Badges de statut Kanban dans le sidebar
Le `WorkspaceSidebar` SHALL afficher des badges colorés inline pour chaque workspace, indiquant le nombre de changes par statut Kanban actif. Seuls les statuts avec un compteur supérieur à 0 SHALL être affichés. Le statut `done` SHALL ne pas être affiché dans le sidebar.

#### Scenario: Workspace avec changes actifs
- **WHEN** un workspace a des changes dans les statuts `to-explore`, `todo` ou `in-progress`
- **THEN** le sidebar affiche un badge coloré par statut non-nul (violet, slate, amber) à droite du nom du workspace

#### Scenario: Workspace sans changes actifs
- **WHEN** un workspace n'a aucun change dans les statuts `to-explore`, `todo` ou `in-progress`
- **THEN** aucun badge n'est affiché pour ce workspace (statuts à 0 sont masqués)

#### Scenario: Mise à jour des badges
- **WHEN** un changement de statut Kanban survient dans un workspace
- **THEN** les badges du sidebar se mettent à jour dans les 15 secondes suivantes
