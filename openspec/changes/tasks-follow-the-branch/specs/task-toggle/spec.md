# Spec Delta

## MODIFIED Requirements

### Requirement: Cible du toggle lorsqu'un worker tient le change
`PATCH /api/workspaces/{workspaceId}/changes/{changeName}/tasks/{taskIndex}` SHALL choisir le fichier à modifier selon l'état du change. Lorsqu'un worker du pool (actif ou en pause) tient le change visé et que le `tasks.md` de son worktree existe et contient au moins une tâche, le toggle SHALL inverser la tâche dans le `tasks.md` du **worktree** du worker, SHALL NOT modifier le `tasks.md` du dépôt principal et SHALL NOT committer (le worker committe son travail à la finalisation). Lorsqu'aucun worker ne tient le change et que la branche `feature/<change>` existe avec un `tasks.md` contenant au moins une tâche, le toggle SHALL suivre l'exigence « Toggle committé sur la branche d'un change sans worker ». Dans tous les autres cas (aucun worker et aucune branche, worker dont le worktree n'est pas encore provisionné, `tasks.md` absent ou sans tâche à l'emplacement visé), le toggle SHALL s'appliquer au `tasks.md` du dépôt principal, comme avant. Dans chaque cas, l'index SHALL désigner la même tâche que dans la liste renvoyée par `GET /api/workspaces/{id}/changes/{name}` pour ce change.

#### Scenario: Toggle sur un change tenu par un worker en pause
- **WHEN** le worker du change `add-user-auth` est en pause et qu'un PATCH est envoyé sur l'index d'une tâche non cochée
- **THEN** la tâche est cochée dans le `tasks.md` du worktree du worker sans commit, le `tasks.md` du dépôt principal est inchangé et le serveur retourne 200

#### Scenario: Toggle sur un change tenu par un worker actif
- **WHEN** le worker du change `add-user-auth` s'exécute et qu'un PATCH est envoyé sur une tâche
- **THEN** la tâche est inversée dans le `tasks.md` du worktree et le `tasks.md` du dépôt principal est inchangé

#### Scenario: Toggle sans worker
- **WHEN** aucun worker ne tient le change `add-user-auth`, que la branche `feature/add-user-auth` n'existe pas et qu'un PATCH est envoyé sur une tâche
- **THEN** la tâche est inversée dans le `tasks.md` du dépôt principal

#### Scenario: Toggle sans worker avec une branche
- **WHEN** aucun worker ne tient le change `add-user-auth`, que `feature/add-user-auth` existe avec une liste de tâches et qu'un PATCH est envoyé sur une tâche
- **THEN** la tâche est inversée dans la branche (voir « Toggle committé sur la branche d'un change sans worker ») et le `tasks.md` du dépôt principal est inchangé

#### Scenario: Worktree sans liste de tâches exploitable
- **WHEN** un worker tient le change mais que son worktree n'est pas encore provisionné, ou que le `tasks.md` de ce worktree est absent ou sans tâche
- **THEN** le toggle s'applique au `tasks.md` du dépôt principal

#### Scenario: Branche sans liste de tâches exploitable
- **WHEN** aucun worker ne tient le change et que la branche existe mais que son `tasks.md` est absent ou sans tâche
- **THEN** le toggle s'applique au `tasks.md` du dépôt principal

#### Scenario: Cohérence avec le DetailPanel
- **WHEN** l'utilisateur coche la dernière tâche d'un change tenu par un worker en pause depuis le DetailPanel
- **THEN** la réponse suivante de `GET /changes/{name}` expose cette tâche cochée et `tasks_done = tasks_total`, sans que le dépôt principal ait été modifié

## ADDED Requirements

### Requirement: Toggle committé sur la branche d'un change sans worker
Lorsqu'aucun worker (actif ou en pause) ne tient le change et que `feature/<change>` existe avec un `tasks.md` contenant au moins une tâche, le toggle SHALL inverser la tâche dans le `tasks.md` du worktree du change, que le backend SHALL recréer à partir de la branche si le worktree a disparu, puis SHALL committer ce seul fichier dans `feature/<change>`, un commit par coche, dont le message est un message Conventional Commits de type `chore` : en-tête `chore(<scope>): Validate task N` (`Reopen task N` pour une décoche, N étant le numéro d'ordre de la tâche, à partir de 1), scope déduit comme pour les autres commits d'un change (voir `change-commit-messages`) ou omis, et corps contenant les lignes `Change: <nom-du-change>` et `Task: <première ligne du texte de la tâche>`. Le toggle SHALL aboutir que le change soit en revue (marqueur présent) ou non, SHALL NOT lever ni poser le marqueur de revue, SHALL NOT modifier le `tasks.md` du dépôt principal et SHALL publier l'événement SSE `change_updated` du change (aucun fichier OpenSpec du dépôt principal n'étant modifié, le watcher reste silencieux). Il SHALL aussi consigner l'entrée d'activité `kanban.task_toggled` habituelle. Le toggle SHALL être sérialisé avec les actions de revue du même change : si une approbation, une demande de correction, un reset ou un autre toggle du change est en cours, il SHALL répondre `409` avec le code `review_busy` sans modifier la branche. Si un worker prend le change entre-temps, il SHALL répondre `409` avec le code `worker_active` sans modifier la branche. Si l'écriture ou le commit échoue, le `tasks.md` du worktree SHALL retrouver son contenu d'origine, la branche SHALL rester inchangée et la réponse SHALL porter un message lisible.

#### Scenario: Coche d'une tâche humaine pendant la revue
- **WHEN** `add-user-auth` est en revue, sans worker, avec 2 tâches non cochées dans `feature/add-user-auth`, et qu'un PATCH coche la première
- **THEN** la tâche est cochée dans le worktree, un commit ne contenant que `tasks.md` est ajouté à `feature/add-user-auth`, le `tasks.md` du dépôt principal est inchangé, `change_updated` est publié et le serveur retourne 200

#### Scenario: Message du commit de coche
- **WHEN** la tâche d'index 8 du change `improve-matrix-change-drilldown-nav` (scope `timeline-spec-matrix`), dont le texte est « 3.2 Parcours manuel de la navigation », est cochée en revue
- **THEN** le commit ajouté à la branche a pour en-tête `chore(timeline-spec-matrix): Validate task 9` et pour corps `Change: improve-matrix-change-drilldown-nav` puis `Task: 3.2 Parcours manuel de la navigation`

#### Scenario: Message du commit de décoche
- **WHEN** la même tâche est ensuite décochée
- **THEN** le commit ajouté a pour en-tête `chore(timeline-spec-matrix): Reopen task 9` et le même corps

#### Scenario: Message sans scope
- **WHEN** aucun scope ne peut être déduit pour le change et qu'une tâche est cochée
- **THEN** l'en-tête du commit est `chore: Validate task N` et le corps contient `Change: <nom>`

#### Scenario: Décocher une tâche
- **WHEN** un PATCH est envoyé sur une tâche déjà cochée d'un change en revue
- **THEN** la tâche est décochée et le changement est committé dans la branche, comme pour une coche

#### Scenario: Worktree disparu
- **WHEN** le worktree d'un change en revue a été supprimé mais que sa branche existe, et qu'un PATCH est envoyé
- **THEN** le worktree est recréé à partir de la branche, la coche y est écrite et committée, et le serveur retourne 200

#### Scenario: Marqueur de revue conservé
- **WHEN** une tâche d'un change en revue est cochée
- **THEN** le marqueur de revue est toujours posé et le change reste en To Review

#### Scenario: Branche sans marqueur ni worker
- **WHEN** la branche d'un change existe, qu'aucun marqueur de revue n'est posé et qu'aucun worker ne le tient (par exemple après une rétrogradation avec `force=true`), et qu'un PATCH est envoyé
- **THEN** la tâche est inversée et committée dans la branche, et le `tasks.md` du dépôt principal est inchangé

#### Scenario: Approbation en cours
- **WHEN** une approbation du change est en cours et qu'un PATCH est envoyé
- **THEN** le serveur retourne `409` avec le code `review_busy` et la branche n'est pas modifiée

#### Scenario: Worker survenu entre-temps
- **WHEN** un worker prend le change entre la résolution de la cible et l'écriture
- **THEN** le serveur retourne `409` avec le code `worker_active` et la branche n'est pas modifiée

#### Scenario: Échec du commit
- **WHEN** le commit de la coche échoue
- **THEN** le `tasks.md` du worktree retrouve son contenu d'origine, la branche est inchangée et la réponse indique l'échec

#### Scenario: Index cohérent avec le détail
- **WHEN** l'utilisateur coche la tâche d'index 1 de la liste affichée par le DetailPanel d'un change en revue
- **THEN** la tâche d'index 1 du `tasks.md` de la branche est inversée, et la réponse suivante de `GET /changes/{name}` l'expose cochée
