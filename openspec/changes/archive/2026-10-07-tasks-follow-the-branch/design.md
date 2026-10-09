# Design

## Context

Voir `proposal.md` pour la motivation. État actuel observé :

- Le travail d'un change est committé dans `feature/<change>` ; le worktree est sous `<worktreesRoot>/<workspaceID>/wt-<change>` et peut être recréé depuis la branche (`WorktreeController.Provision`).
- **Lecture** : `ListChanges` et `GetChange` ne surchargent les tâches qu'avec le worktree d'un worker qui tient le change (`activeWorkerChanges` → `ApplyWorktreeProgress`). Sans worker, y compris en To Review, ils lisent le `tasks.md` du dépôt principal.
- **Écriture** : `TaskHandler.toggleRoot` choisit le worktree d'un worker qui tient le change, sinon le dépôt principal. L'écriture dans le worktree n'est pas committée : le worker committe lui-même (`CommitAll`) à la finalisation.
- **Revue** : `ApproveReview` fusionne la *branche* et ne committe rien avant le merge ; une coche non committée dans un worktree serait donc perdue, et un worktree recréé depuis la branche l'effacerait. `RequestCorrection` fournit le schéma à réutiliser : `reviewLock(change)` → `checkReviewable` → `Provision` → écriture de `tasks.md` → `CommitFile` → `publishChangeUpdated`. Le watcher du dépôt principal ne voit aucun de ces changements.
- **Statut de colonne** : `deriveStatus(done, total, launched)` est calculé sur le dépôt principal ; `ApplyWorktreeProgress` le recalcule avec `launched=true` et plafonne `done` à `in-progress` pour un change tenu par un worker.
- **Reset** : `FFHandler.ResetTasks` vide `tasks.md` du dépôt principal et efface `launched` / `order`, sans regarder le pool ni la branche. `DeleteChange` appelle déjà `pool.NewWorktreeController(path, id, "").Cleanup(name)` pour supprimer worktree, branche et marqueur.
- Le format des messages de commit est défini par le change `conventional-commit-messages` (Conventional Commits : type, scope déduit des tags ou des capacités du change, sujet issu du nom, corps `Change:`) ; ce change réutilise sa fonction de construction pour le commit de coche.
- Le worker actif peut être coché dans son worktree (change archivé `resume-paused-worker-after-manual-tasks`) ; ce comportement est conservé tel quel.

## Goals / Non-Goals

**Goals:**
- Que la lecture et l'écriture des tâches visent toujours le même fichier : celui de la branche dès qu'elle existe sans worker, celui du worker sinon, le dépôt principal à défaut.
- Qu'une coche faite en revue survive à la recréation du worktree et soit embarquée par la fusion.
- Qu'un reset ne laisse pas de travail orphelin affiché sur une carte To Explore.

**Non-Goals:**
- Recalculer la colonne à partir de la branche (voir D3).
- Changer le toggle quand un worker tient le change (pas de `409`, pas de commit) ni la reprise en finalisant.
- Marqueur de tâche humaine, triage, désactivation d'Approve : change `hitl-human-validation-tasks`.
- Tester l'application depuis la revue.

## Decisions

### D1. La source des tâches est résolue par une fonction unique, utilisée en lecture et en écriture

Une résolution à trois niveaux, partagée par `ListChanges`, `GetChange` et le toggle :

```
 worker tient le change ?  oui, worktree avec >= 1 tâche  -> worktree du worker   (sans commit)
        |non
 feature/<change> existe, tasks.md de la branche >= 1 tâche -> branche            (commit par coche)
        |non
        v                                                   -> dépôt principal
```

La condition « au moins une tâche » est celle d'`ApplyWorktreeProgress`, ce qui garantit que l'index calculé par le front sur la liste affichée vise le fichier effectivement modifié. Le contenu de la branche est lu par `WorktreeController` : le `tasks.md` du worktree s'il existe (il peut porter une coche non committée d'un travail interrompu), sinon `git show feature/<c>:openspec/changes/<c>/tasks.md`. Cette lecture n'appelle jamais `Provision` : un GET ne crée ni worktree ni commit.

*Alternative :* ne lire que le worktree et le provisionner à la demande. Écartée : un GET aurait des effets de bord (création de worktree), et le spec de la revue exige déjà que la lecture fonctionne sans worktree.

### D2. Un toggle sur la branche s'écrit, se committe et se publie sous le verrou de revue

Nouvelle opération du `Manager` (même fichier que les actions de revue) : `ToggleBranchTask(ctx, workspaceID, workspacePath, change, index)`.

1. `reviewLock(change).TryLock()` : si le verrou est pris (approbation, correction, reset, autre toggle), `ErrReviewBusy` → `409` `review_busy`. `TryLock` plutôt que `Lock` pour qu'un toggle n'attende pas la validation d'une approbation, qui peut durer des minutes.
2. `hasWorkerFor(change)` : un worker est apparu → `ErrWorkerActive` → `409` `worker_active`.
3. `Provision(change)` (recrée le worktree depuis la branche s'il manque), `openspec.ToggleTask(worktreeDir, change, index)` (réutilisé tel quel : il ne dépend que de la racine), puis `CommitFile(change, rel, message)`. Le message est construit avec la fonction de `conventional-commit-messages` : en-tête `chore(<scope>): Validate task N` à la coche, `chore(<scope>): Reopen task N` à la décoche (N = index + 1 ; le type est toujours `chore`, une coche ne modifie que `tasks.md`), corps `Change: <nom>` puis `Task: <première ligne du texte de la tâche>`. Le scope est déduit comme pour les autres commits du change (premier `tags.components`, sinon première capacité de `specs/`, sinon omis).
4. En cas d'échec d'écriture ou de commit, le fichier est restauré à son contenu d'origine (comme `RequestCorrection`).
5. `publishChangeUpdated(workspaceID, change)`, puis le handler consigne l'entrée d'activité `kanban.task_toggled` existante.

Un commit par coche est assumé : c'est une piste d'audit de la validation humaine, visible dans l'historique du merge `--no-ff`. Le marqueur de revue n'est ni posé ni levé : la coche ne change pas l'état de revue.

*Alternatives :* (a) écrire sans committer et laisser Approve committer : une coche serait perdue si le worktree est recréé avant, ou fusionnée sans trace ; (b) faire attendre le toggle sur le verrou (`Lock`) : bloquerait la requête pendant la validation d'Approve ; (c) fusionner plusieurs coches en un commit par `--amend` : réécrit l'historique de la branche pour un gain cosmétique ; (d) reprendre l'en-tête du commit du worker (`feat(<scope>): <Sujet>`) pour chaque coche : N commits `feat` identiques seraient comptés comme N fonctionnalités par un outil de changelog et ne diraient pas ce qui a changé.

### D3. La branche fait foi pour les tâches et les compteurs, jamais pour la colonne

`openspec.ApplyBranchProgress(ch, content)` ne modifie que `TasksDone` / `TasksTotal` : ni `KanbanStatus` ni `IsStale`. La colonne reste dérivée du marqueur de revue, de `launched` et du dépôt principal. Conséquences voulues :

- Un change rétrogradé en Ready (arrêt forcé : worktree et commits conservés) reste en Ready au lieu de sauter en In Progress, ce que ferait `ApplyWorktreeProgress` avec son `launched=true` forcé.
- Une branche cochée N/N ne place pas le change en Done et ne lui offre pas « Sync & Archive », alors que rien n'est fusionné.
- Le glisser-déposer garde son sens : le placement dans Ready / To Do est une décision de l'utilisateur.

Conséquence acceptée : la carte peut afficher un compteur (« 10 / 11 ») qui contredit sa colonne (To Do) dans l'état « branche seule ». C'est un signal que du travail existe, pas une erreur.

Pour un change tenu par un worker, la surcharge existante (compteurs *et* colonne) est conservée.

### D4. Détection groupée des branches, lecture du contenu seulement pour les cartes concernées

`ListChanges` ne doit pas faire N appels git. Une fonction (`openspec.FeatureBranches(workspacePath) map[string]bool`, un seul `git for-each-ref refs/heads/feature/`) alimente `has_branch` ; le contenu des tâches n'est lu que pour les changes qui portent une branche et qu'aucun worker ne tient. `ReviewMarkers` conserve son appel unique ; le `branchExists` par change qu'elle déclenche pour un marqueur est remplacé par la même map.

### D5. Le reset nettoie la branche, avant de toucher aux fichiers du change

`FFHandler.ResetTasks` reçoit le registre de pools et suit cet ordre :

1. refus existant si un ff tourne ;
2. si `feature/<change>` existe : `reviewLock.TryLock` (`review_busy`), refus `409` `worker_active` si un worker actif tient le change, libération d'un worker en pause (`ReleasePausedForChange`) ;
3. `WorktreeController.Cleanup(change)` (supprime worktree, branche, marqueur) — en cas d'échec : `500`, rien d'autre n'est modifié ;
4. seulement ensuite : vidage de `tasks.md` et `ClearKanbanState`, puis `change_updated` publié explicitement (le marqueur disparaît sans écriture OpenSpec) et entrée d'activité.

Le nettoyage précède l'écriture pour qu'un échec laisse le change intact et re-tentable. Cette destruction irréversible est précédée d'une confirmation explicite côté UI (D6). `DeleteChange` donne déjà le précédent (`Cleanup` avec une racine de worktrees par défaut) ; le reset l'imite.

*Alternative :* refuser le reset tant qu'une branche existe. Écartée : l'utilisateur n'aurait aucun moyen de repartir de zéro depuis le Kanban.

### D6. Frontend : `has_branch` et la confirmation de reset

`Change` (`useChanges.ts`) gagne `has_branch?: boolean`. `ResetTasksDialog` affiche l'avertissement de perte de travail (nouveau message dédié, bouton en style d'avertissement) quand `change.has_branch` est vrai, même si `tasks_done` vaut 0 : aujourd'hui `hasProgress` se limite à `tasks_done > 0`, calculé sur le dépôt principal. `KanbanPage` affiche l'erreur du backend (`worker_active`, `review_busy`) dans une notification et ramène la carte dans sa colonne.

Le DetailPanel n'a pas besoin de changer : il affiche `tasks` tel que renvoyé, et son toggle appelle toujours le même endpoint ; l'invalidation des requêtes `changes` / `change-detail` après un toggle existe déjà. Le panneau de revue reste inchangé.

## Risks / Trade-offs

- [Un worker démarre entre `hasWorkerFor` et le commit (le dispatcher ne prend pas `reviewLock`)] → fenêtre de l'ordre de la milliseconde, même classe de risque que celle déjà acceptée pour le toggle sur worker actif ; `hasWorkerFor` est revérifié juste avant d'écrire.
- [`Provision` d'un change à branche seule crée un worktree comme effet de bord d'un toggle] → identique à `RequestCorrection` ; limité à une action explicite de l'utilisateur.
- [Péremption (`IsStale`) calculée sur la date du `tasks.md` du dépôt principal, que les coches de la branche ne touchent pas] → ne concerne que les colonnes in-progress / done, rarement en état « branche seule » ; documenté, non corrigé ici.
- [Coches antérieures faites dans le `tasks.md` du dépôt principal pour un change qui a une branche : devenues invisibles, et fichier modifié non committé que la fusion peut refuser d'écraser] → pas de migration automatique (cas hérité) ; le message d'erreur d'Approve existant (« Fusion refusée ») reste lisible, et `git checkout -- openspec/changes/<c>/tasks.md` règle le cas.
- [Un change To Explore dont le dépôt principal est vide mais dont une branche porte des tâches (résidu d'un reset ancien)] → afficherait des compteurs sur une carte To Explore ; le reset nettoie désormais la branche, donc le cas ne se produit plus qu'avec des données historiques.
- [Lecture de `git show` à chaque rafraîchissement du détail] → un seul processus git par ouverture de détail, uniquement pour les changes à branche sans worktree ; aucun coût pour la liste hors cartes à branche.
- [Un commit par coche allonge l'historique de la branche] → assumé comme piste d'audit (D2).

## Migration Plan

Aucune migration de données. Le déploiement est un simple redémarrage du backend. Retour arrière : revert du change ; les commits de coche déjà ajoutés aux branches restent valides (de petits commits sur `tasks.md`).
