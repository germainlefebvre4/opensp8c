# Design

## Context

Un worker en pause est copié dans `Manager.pausedWorkers` (`pool/worker.go`, `pauseWorker`) pour rester visible après la fin de sa goroutine. Trois lectures de cette map comptent le worker comme « tenant » le change : `Status()` (donc `activeWorkerChanges` côté handlers, qui alimente `worker_active`), `tick()` (exclusion du dispatcher) et `hasWorkerFor` (actions de revue). Seul `ResumeWorker`, `Stop()` et `Start()` la modifient.

`Unlaunch` (`handlers/kanban.go`) décide `held` via `activeWorkerChanges`, donc vrai pour un worker en pause, mais `CancelAndWaitForChange` ne parcourt que `activeWorkers` : pour un worker en pause il répond `found=false` sans erreur, le handler écrit `launched: false` et la pause survit. Voir proposal.md pour le symptôme observé.

## Goals / Non-Goals

**Goals:**
- La rétrogradation libère toujours la pause du change, quelle que soit la façon dont elle est déclenchée (drag, bouton).
- L'API et l'UI distinguent un worker qui s'exécute d'un worker bloqué.

**Non-Goals:**
- Corriger la cause d'une pause (par exemple un worktree périmé : change `refresh-stale-worktree-on-launch`).
- Modifier la reprise explicite, les actions de revue ou l'arrêt du pool.
- Supprimer la branche ou le worktree d'un change rétrogradé : ils sont conservés, comme pour une rétrogradation forcée.

## Decisions

**1. Libération côté `Manager`, pas côté handler.** Ajouter `ReleasePausedForChange(change) bool` : sous `m.mu`, supprime les entrées de `pausedWorkers` dont `ActiveChange == change`, appelle `broadcastLocked()` si une entrée a été retirée. Même verrou et même mécanisme que `ResumeWorker`, donc pas de nouvelle voie d'écriture vers la map. *Alternative écartée* : étendre `CancelAndWaitForChange` pour qu'il libère aussi les pauses. Cette fonction décrit « annuler et attendre un worker » (retour `found/result/timedOut`) ; un worker en pause n'a rien à attendre, mélanger les deux brouille son contrat.

**2. `Unlaunch` : libération systématique avant d'écrire `launched: false`.**
```
held actif ?  non force -> 409
              force     -> cancelAndWait (timeout 503, merged 409)
puis : ReleasePausedForChange(change)   // pause existante ou survenue entre-temps
puis : SetLaunched(false) -> 204
```
Le refus 409 sans `force` ne vise que le worker **actif**. Libérer après `cancelAndWait` évite un zombie si le worker passe en pause juste avant d'honorer l'annulation ; `pauseWorker` refuse déjà de se publier quand son contexte est annulé, la libération est donc une ceinture supplémentaire, pas la protection principale. Si la libération a lieu mais que l'écriture de `launched` échoue, la pause est perdue sans que le change soit rétrogradé : acceptable, le change reste `To Do` et sera redistribué au tick suivant, ce qui est le comportement d'une reprise.

**3. Source de vérité des indicateurs : un état par change, pas une map de chemins.** `activeWorkerChanges` renvoie aujourd'hui `map[change]worktreePath`. Il renvoie à la place `map[change]heldWorker{WorktreePath, Paused}` construit depuis `Status()` (qui contient déjà `Worker.Status`). `ListChanges` et `GetChange` posent `WorkerActive = !Paused` et `WorkerPaused = Paused`. L'overlay de progression du worktree (`ApplyWorktreeProgress`) reste appliqué aux deux cas.

**4. `worker_active` change de sens (API interne).** Il ne couvre plus les workers en pause. Les consommateurs sont tous dans ce dépôt : `KanbanPage` (modale), `ChangeCard` (badge, bouton d'arrêt), `DetailPanel` (désactivation de la suppression). Ce dernier SHALL désactiver la suppression pour `worker_active || worker_paused`, car le backend continue de refuser la suppression d'un change tenu par un worker en pause. *Alternative écartée* : garder `worker_active` vrai pour tout worker tenant et ajouter `worker_paused` en plus. Moins de modifications, mais les deux indicateurs ne seraient plus exclusifs et chaque consommateur devrait redécouvrir la nuance.

**5. UI.** `KanbanPage.handleDragEnd` : modale d'interruption uniquement si `worker_active`. Pour `worker_paused`, appel direct de `unlaunchChange` sans `force`. `ChangeCard` : badge « en pause » (distinct, tooltip reprenant l'idée « worker bloqué, rétrograder le libère »), sans bouton d'arrêt. La raison de blocage reste affichée dans le panneau du pool tant que la pause existe, et disparaît avec elle.

## Risks / Trade-offs

- **[Un consommateur de `worker_active` oublié lit `false` pour un worker en pause]** → inventaire exhaustif fait (voir Decision 4) ; une tâche vérifie par recherche qu'aucune autre occurrence n'existe, et la suppression est couverte par un test.
- **[Un utilisateur libère une pause par la rétrogradation et perd la raison de blocage]** → la raison est aussi journalisée dans le run `pool` du worker (`pool_run_end`), consultable après coup.
- **[Libération puis nouvelle promotion rejoue la même pause si la cause demeure]** → comportement voulu ici : cette change ne traite pas la cause ; le change `refresh-stale-worktree-on-launch` supprime le cas du worktree périmé.
