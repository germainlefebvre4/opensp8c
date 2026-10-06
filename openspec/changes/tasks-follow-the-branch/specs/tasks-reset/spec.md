# Spec Delta

## MODIFIED Requirements

### Requirement: Artifacts conservés après reset
Le reset SHALL vider `tasks.md` et effacer les champs `launched` et `order` du `.openspec.yaml` du changement (voir `kanban-ready-column`) et, lorsque le changement porte une branche `feature/<change>`, supprimer aussi son worktree, sa branche et son marqueur de revue (voir « Reset d'un change qui porte une branche »). Les fichiers `proposal.md`, `design.md`, et les specs dans `specs/` du dépôt principal SHALL être conservés intacts. Le changement retourne à l'état "to-explore" tout en préservant le travail de réflexion antérieur.

#### Scenario: Proposal et design conservés après reset
- **WHEN** `PATCH /changes/{name}/tasks/reset` est exécuté avec succès
- **THEN** `proposal.md` et `design.md` existent toujours avec leur contenu intact

#### Scenario: Status retourné to-explore après reset
- **WHEN** `tasks.md` est vidé
- **THEN** `GET /changes/{name}` retourne `kanban_status: "to-explore"` pour ce changement

#### Scenario: État de lancement effacé après reset
- **WHEN** `PATCH /changes/{name}/tasks/reset` est exécuté avec succès pour un changement dont le `.openspec.yaml` contient `launched` et/ou `order`
- **THEN** ces deux champs sont retirés du `.openspec.yaml` du changement

## ADDED Requirements

### Requirement: Reset d'un change qui porte une branche
Lorsque le change visé porte une branche `feature/<change>`, `PATCH /api/workspaces/{id}/changes/{name}/tasks/reset` SHALL, avant de vider `tasks.md`, supprimer le worktree, la branche et le marqueur de revue du change, de sorte que le travail committé soit perdu et que les tâches de la branche ne s'affichent plus. Le reset SHALL retourner `409` avec le code `worker_active` et ne rien modifier lorsqu'un worker actif tient le change, et `409` avec le code `review_busy` lorsqu'une action de revue ou un toggle du change est en cours. Un worker en pause qui tient le change SHALL être libéré. Si la suppression de la branche ou du worktree échoue, le reset SHALL répondre `500` avec un message lisible et SHALL NOT modifier `tasks.md` ni le `.openspec.yaml`. Après un reset réussi, le backend SHALL publier `change_updated` du change et `GET /changes/{name}` SHALL retourner `has_branch` absent ou `false`. Un reset d'un change sans branche SHALL se comporter comme avant.

#### Scenario: Reset d'un change en Ready avec une branche conservée
- **WHEN** un change rétrogradé en Ready (branche à 7 tâches cochées sur 10) est réinitialisé
- **THEN** son worktree, sa branche et son marqueur sont supprimés, `tasks.md` est vidé, `launched` et `order` sont effacés et le serveur retourne 204

#### Scenario: Reset bloqué par un worker actif
- **WHEN** le reset est demandé pour un change tenu par un worker actif
- **THEN** le serveur retourne `409` avec le code `worker_active` et ni la branche ni `tasks.md` ne sont modifiés

#### Scenario: Reset libérant un worker en pause
- **WHEN** le reset est demandé pour un change tenu par un worker en pause
- **THEN** le worker est libéré, la branche et le worktree sont supprimés, `tasks.md` est vidé et le serveur retourne 204

#### Scenario: Action de revue en cours
- **WHEN** une approbation du change est en cours et que le reset est demandé
- **THEN** le serveur retourne `409` avec le code `review_busy` et rien n'est modifié

#### Scenario: Échec du nettoyage de la branche
- **WHEN** la suppression de la branche ou du worktree échoue
- **THEN** le serveur retourne `500` avec un message lisible, `tasks.md` et le `.openspec.yaml` sont inchangés

#### Scenario: Reset d'un change sans branche
- **WHEN** le reset est demandé pour un change qui ne porte pas de branche
- **THEN** le comportement est celui de « Reset de tasks.md via endpoint dédié »
