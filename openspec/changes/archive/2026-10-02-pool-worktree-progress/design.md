# Design

## Context

`loadChange` (`openspec/change.go`) calcule progression, colonne et staleness à partir du seul `tasks.md` du dépôt principal. Les workers du pool travaillent dans un worktree (`~/.opensp8c/worktrees/<ws>/wt-<change>`) dont le `tasks.md` n'est lu que par `runWorker` pour son contrôle de fin. Le watcher SSE ne surveille que `<workspace>/openspec`. `Manager.Status` expose déjà les workers actifs **et en pause** avec leur `WorktreePath` (renseigné après le provisionnement). Voir proposal.md pour la motivation.

## Goals / Non-Goals

**Goals:**
- La progression et la colonne reflètent le travail de l'agent en direct pendant le run.
- Aucune écriture dans le dépôt principal.
- Réutiliser le flux SSE `change_updated` existant, sans changement frontend.

**Non-Goals:**
- Dériver le statut `to-review` (le mode `hitl-review` est hors périmètre).
- Surveiller d'autres fichiers du worktree que `tasks.md`.
- Persister la progression du worktree après la fin du worker.

## Decisions

**1. Surcouche à la lecture (pas de synchronisation).** `ListChanges` et `GetChangeDetail` reçoivent, pour chaque change avec worker actif, le chemin de son worktree. Une fonction du paquet `openspec` recalcule alors `tasks_done`/`tasks_total`, la liste des tâches et la staleness depuis `<worktree>/openspec/changes/<change>/tasks.md`, puis redérive la colonne. Alternative écartée : recopier les coches dans le dépôt principal, qui le salit et provoque des conflits au merge.

**2. Repli sur le dépôt principal.** Si le `tasks.md` du worktree est absent ou a `total == 0`, aucune surcouche n'est appliquée. Cela couvre un worker pas encore provisionné (`WorktreePath` vide) et un worktree incohérent.

**3. Plafonnement à In Progress.** Avec un worker actif, la colonne dérivée est bornée : `done` devient `in-progress`. Le passage à Done est réservé au merge, qui fait coïncider le dépôt principal et le worker retiré. `todo` reste `todo` si rien n'est coché, `in-progress` sinon.

**4. Surveillance dans le pool, pas dans le `WatcherService`.** `runWorker` démarre, une fois le worktree provisionné, un watcher fsnotify sur le **répertoire** `openspec/changes/<change>` du worktree (un fichier réécrit par renommage atomique ferait perdre un watch posé sur le fichier). Les événements sur `tasks.md` sont debouncés à 150 ms puis diffusés via le `Broadcaster` du Manager comme `Event{Type: "change_updated", Name: change}`. Le watcher s'arrête par `defer` à la sortie de `runWorker` (succès, pause, annulation). Alternative écartée : étendre `WatcherService` à des chemins hors workspace, qui lui ferait connaître le cycle de vie des workers.

**5. Chemin du worktree via `Status`.** `activeWorkerChanges` du handler passe d'un `map[string]bool` à un `map[string]string` (change → `WorktreePath`). `WorkerActive` reste vrai dès que le change a une entrée, même avec un chemin vide.

**6. Fin de run.** Le merge écrit le `tasks.md` complet dans le dépôt principal alors que le worker est encore actif : l'événement fsnotify du dépôt principal arrive donc avec le plafonnement encore actif. Le `notify()` (`pool_updated`) émis à la sortie du worker, qui provoque déjà le rechargement de la liste des changes, fait ensuite apparaître Done.

## Risks / Trade-offs

- [Le watcher du worktree rate un événement] → un `pool_updated` accompagne chaque changement de statut du worker et recharge la liste ; le prochain rechargement corrige l'état.
- [Worktree legacy ou déplacé] → le chemin vient de `Worker.WorktreePath`, déjà résolu par `Provision`, pas reconstruit.
- [Worker en pause : worktree figé] → la surcouche reste appliquée ; c'est voulu, le change garde sa progression partielle tant que le worker le tient. La surveillance est arrêtée avec la goroutine.
- [Barre à 100 % en In Progress pendant validation et merge] → comportement voulu et documenté dans la spec ; le badge worker explique l'état.
