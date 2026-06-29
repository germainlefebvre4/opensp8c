## ADDED Requirements

### Requirement: Reset de tasks.md via endpoint dédié
Le backend SHALL exposer `PATCH /api/workspaces/{id}/changes/{name}/tasks/reset` qui vide le contenu de `tasks.md` pour le changement spécifié. Après reset, `deriveStatus(done=0, total=0)` retourne `"to-explore"` et le watcher SSE détecte le write et émet `change_updated` automatiquement. L'endpoint SHALL retourner 409 si un subprocess ff est actif pour ce changement.

#### Scenario: Reset réussi depuis todo
- **WHEN** le frontend appelle `PATCH /changes/{name}/tasks/reset` pour un changement en statut todo (aucun ff actif)
- **THEN** `tasks.md` est vidé, le backend retourne 204, et le watcher émet `change_updated`

#### Scenario: Reset réussi depuis in-progress
- **WHEN** le frontend appelle `PATCH /changes/{name}/tasks/reset` pour un changement en statut in-progress (aucun ff actif)
- **THEN** `tasks.md` est vidé, le backend retourne 204, et le watcher émet `change_updated`

#### Scenario: Reset bloqué pendant ff actif
- **WHEN** le frontend appelle `PATCH /changes/{name}/tasks/reset` pour un changement dont un subprocess ff est actif
- **THEN** le backend retourne 409 Conflict sans modifier `tasks.md`

#### Scenario: Reset d'un changement sans tasks.md
- **WHEN** le frontend appelle `PATCH /changes/{name}/tasks/reset` pour un changement sans fichier `tasks.md`
- **THEN** le backend retourne 204 sans erreur (état déjà cohérent)

### Requirement: Artifacts conservés après reset
Le reset SHALL uniquement vider `tasks.md`. Les fichiers `proposal.md`, `design.md`, et les specs dans `specs/` SHALL être conservés intacts. Le changement retourne à l'état "to-explore" tout en préservant le travail de réflexion antérieur.

#### Scenario: Proposal et design conservés après reset
- **WHEN** `PATCH /changes/{name}/tasks/reset` est exécuté avec succès
- **THEN** `proposal.md` et `design.md` existent toujours avec leur contenu intact

#### Scenario: Status retourné to-explore après reset
- **WHEN** `tasks.md` est vidé
- **THEN** `GET /changes/{name}` retourne `kanban_status: "to-explore"` pour ce changement
