# Spec Delta

## MODIFIED Requirements

### Requirement: Compteurs Kanban dans la liste des workspaces
L'API `GET /api/workspaces` SHALL retourner pour chaque workspace un champ `task_counts` contenant le nombre de changes par statut Kanban (`to-explore`, `ready`, `todo`, `in-progress`, `verifying`, `to-review`, `done`). Le calcul SHALL être effectué en lisant le répertoire `openspec/changes/` du workspace à chaque requête.

#### Scenario: Listing de workspaces avec des changes
- **WHEN** l'API reçoit une requête `GET /api/workspaces`
- **THEN** chaque workspace dans la réponse inclut un objet `task_counts` avec les sept statuts et leur nombre respectif (0 si aucun change dans ce statut)

#### Scenario: Workspace sans changes
- **WHEN** un workspace n'a aucun change dans son répertoire `openspec/changes/`
- **THEN** `task_counts` retourne `{ "to-explore": 0, "ready": 0, "todo": 0, "in-progress": 0, "verifying": 0, "to-review": 0, "done": 0 }`

#### Scenario: Répertoire changes inexistant
- **WHEN** le répertoire `openspec/changes/` n'existe pas dans le workspace
- **THEN** `task_counts` retourne tous les compteurs à 0 sans erreur

#### Scenario: Change en vérification
- **WHEN** un workspace contient un change dont le statut est `verifying`
- **THEN** `task_counts.verifying` vaut 1 et ce change n'est compté dans aucun autre statut
