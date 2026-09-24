# Spec Delta

## MODIFIED Requirements

### Requirement: Artifacts conservés après reset
Le reset SHALL uniquement vider `tasks.md` et effacer les champs `launched` et `order` du `.openspec.yaml` du changement (voir `kanban-ready-column`). Les fichiers `proposal.md`, `design.md`, et les specs dans `specs/` SHALL être conservés intacts. Le changement retourne à l'état "to-explore" tout en préservant le travail de réflexion antérieur.

#### Scenario: Proposal et design conservés après reset
- **WHEN** `PATCH /changes/{name}/tasks/reset` est exécuté avec succès
- **THEN** `proposal.md` et `design.md` existent toujours avec leur contenu intact

#### Scenario: Status retourné to-explore après reset
- **WHEN** `tasks.md` est vidé
- **THEN** `GET /changes/{name}` retourne `kanban_status: "to-explore"` pour ce changement

#### Scenario: État de lancement effacé après reset
- **WHEN** `PATCH /changes/{name}/tasks/reset` est exécuté avec succès pour un changement dont le `.openspec.yaml` contient `launched` et/ou `order`
- **THEN** ces deux champs sont retirés du `.openspec.yaml` du changement
