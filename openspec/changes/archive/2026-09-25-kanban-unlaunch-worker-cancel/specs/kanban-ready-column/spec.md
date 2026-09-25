# Spec Delta: kanban-ready-column

## MODIFIED Requirements

### Requirement: Marqueur persistant "lancé" par change

Chaque change SHALL avoir un marqueur booléen persistant `launched`, stocké dans son fichier `.openspec.yaml`. Un change dont `tasks.md` existe avec au moins une tâche, et dont `launched` est absent ou `false`, SHALL avoir le statut Kanban `ready`. Un change dans les mêmes conditions de `tasks.md` mais avec `launched: true` (ou le champ absent, voir compatibilité ascendante) SHALL avoir le statut Kanban `todo` (ou son statut dérivé habituel si des tâches sont déjà cochées). Si le fichier `.openspec.yaml` n'existe pas lors d'une écriture du marqueur `launched` (ex. promotion ou rétrogradation d'une tâche legacy), le backend SHALL créer automatiquement le fichier `.openspec.yaml` avec la valeur demandée au lieu de renvoyer une erreur système.

#### Scenario: Change avec tasks.md généré et non lancé
- **WHEN** un change a un fichier `tasks.md` non vide et `launched: false` (ou absent après création via le flux Ready) dans son `.openspec.yaml`
- **THEN** `GET /changes` retourne `kanban_status: "ready"` pour ce change

#### Scenario: Change lancé avec tâches non démarrées
- **WHEN** un change a un fichier `tasks.md` non vide, `launched: true`, et aucune tâche cochée
- **THEN** `GET /changes` retourne `kanban_status: "todo"` pour ce change

#### Scenario: Compatibilité ascendante — champ absent traité comme lancé
- **WHEN** un change possède un `tasks.md` non vide mais son `.openspec.yaml` ne contient aucun champ `launched` (change créé avant l'introduction de cette fonctionnalité, ou créé hors du flux applicatif Ready)
- **THEN** le champ est traité comme `launched: true` et le change conserve son statut dérivé habituel (`todo`, `in-progress` ou `done`), sans jamais apparaître en `ready`

#### Scenario: Création automatique de .openspec.yaml si absent à l'écriture
- **WHEN** l'action de promotion ou de rétrogradation est exécutée pour un change ne possédant pas de fichier `.openspec.yaml`
- **THEN** le backend initialise et crée le fichier `.openspec.yaml` avec le marqueur `launched` approprié, sans erreur HTTP 500

### Requirement: Rétrogradation manuelle To Do → Ready

Le backend SHALL exposer une action symétrique permettant de remarquer un change `launched: false`, le retirant de l'éligibilité au ramassage par l'Agent Pool. Cette action SHALL supporter un paramètre optionnel `force=true`. Si un worker de l'Agent Pool est actuellement actif sur ce change : sans `force=true`, l'action SHALL être refusée avec une erreur 409 Conflict ; avec `force=true`, le backend SHALL interrompre le worker actif (`cancel()`), conserver le code et les commits dans le git worktree (ou réinitialiser les modifications non validées si l'exécution n'est pas isolée dans un worktree), écrire `launched: false` dans le `.openspec.yaml`, et renvoyer un statut 204. Dans l'interface Kanban, glisser une carte ayant un worker actif de `To Do` vers `Ready` SHALL ouvrir un dialogue de confirmation demandant à l'utilisateur de valider l'interruption du worker avant d'appeler l'endpoint avec `force=true`.

#### Scenario: Rétrogradation réussie
- **WHEN** l'utilisateur déclenche la rétrogradation d'un change en colonne `To Do` sans worker actif (drag vers `Ready` ou bouton dédié)
- **THEN** le backend écrit `launched: false` dans le `.openspec.yaml` du change, le watcher SSE émet `change_updated`, et la carte apparaît en colonne `Ready`

#### Scenario: Rétrogradation refusée pendant exécution
- **WHEN** l'action de rétrogradation est appelée sans `force=true` pour un change dont `worker_active` vaut `true`
- **THEN** le backend refuse l'action avec une erreur 409 Conflict, et le change reste `launched: true`

#### Scenario: Rétrogradation forcée avec worker actif
- **WHEN** l'action de rétrogradation est appelée avec `force=true` pour un change dont un worker est actif
- **THEN** le backend interrompt le subprocess de l'agent associé, retire le worker de la liste active, écrit `launched: false` dans `.openspec.yaml`, et émet les événements SSE `pool_updated` et `change_updated`

#### Scenario: Rétrogradation conserve le code du git worktree
- **WHEN** un worker actif est interrompu via rétrogradation forcée et qu'il opérait dans un git worktree dédié (`~/.opensp8c/worktrees/wt-<name>`)
- **THEN** le dossier worktree et la branche `feature/<name>` avec ses modifications et commits restent intacts sur le disque

#### Scenario: Confirmation UI lors du drag d'une carte avec worker actif
- **WHEN** l'utilisateur glisse-dépose une carte ayant `worker_active = true` de `To Do` vers `Ready`
- **THEN** l'interface ouvre une modale de confirmation d'interruption du worker ; la confirmation déclenche l'appel avec `force=true` et l'annulation laisse la carte en `To Do`
