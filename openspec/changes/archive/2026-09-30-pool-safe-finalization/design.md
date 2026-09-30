# Design

## Context

Voir `proposal.md` (Why) pour la motivation. État actuel, vérifié dans le code :

- `runWorker` (`pool/worker.go`) enchaîne : provisionnement → un subprocess agent unique → tour `/opsx:apply` → validation → boucle de guérison → contrôle de `tasks.md` → finalisation. Le chemin de finalisation est `MergeAndCleanup` (`pool/worktree.go`) en `full-autonomy`, ou rien du tout en `hitl-review` (l'issue `awaiting-review` n'est qu'une ligne du journal).
- Le worktree est créé par `git worktree add -b feature/<change> <path>` : il part du **dernier commit** de la branche courante, sans les fichiers non suivis. Aucun code du pool ne fait `git add` ni `git commit`.
- `Manager.tick` (toutes les 5 s) redistribue tout changement `todo` qui n'est ni actif ni dans `pausedWorkers`. Le statut Kanban est dérivé de `tasks.md` **de l'arbre principal** (`deriveStatus`), qui ne bouge pas pendant qu'un worker coche les tâches dans son worktree.
- `Manager.Stop` annule les contextes puis vide `activeWorkers` et `pausedWorkers` ; le worker annulé finit plus tard par `pause(...)`, qui re-remplit `pausedWorkers`. `runWorker` supprime son entrée par identifiant numérique.
- Les subprocess sont lancés par `session.StartSubprocess` (partagé avec l'exploration) via `exec.CommandContext` sans groupe de processus ; `runValidationCommand` n'a ni délai ni `WaitDelay`.
- `worktreesDir()` est `~/.opensp8c/worktrees`, global à tous les workspaces et non injectable (les tests y écrivent).

Le serveur est local, un seul utilisateur, agent Claude en `full-autonomy` : les choix ci-dessous privilégient la **non-perte de travail** et la **visibilité des échecs** sur la performance ou la généralité.

## Goals / Non-Goals

**Goals:**
- Aucun chemin de finalisation ne peut détruire du travail non committé de l'agent.
- Un run n'est `completed` ou `awaiting-review` que si du travail réel, vérifié et committé existe dans la branche.
- Tout échec de finalisation devient une pause visible avec une raison, jamais une boucle silencieuse.
- Plus aucun processus orphelin après annulation, arrêt du pool ou arrêt du serveur.

**Non-Goals:**
- Implémenter le flux HITL côté backend (statut `to-review`, routes d'approbation/correction) : change séparé.
- Persister l'état « en attente de revue » au-delà du redémarrage du serveur.
- Rendre les délais configurables dans l'UI ou Settings (constantes du package dans ce change).
- Commiter automatiquement les artefacts d'un change produit par `ff` dans l'arbre principal de l'utilisateur.
- Durcir l'API HTTP ou le chat d'exploration.

## Decisions

### D1. Finalisation ordonnée : commit → fusion → suppression non forcée

Nouvel ordre dans `runWorker`, après validation et contrôle de complétion :

```
  completion OK
       |
       v
  Commit(worktree)  --(échec)--> pause(raison), worktree conservé
       |
       v
  produit du travail ? --non--> pause("aucun travail produit")
       |
       v
  full-autonomy ?                 hitl-review ?
       |                               |
  mergeMu.Lock                    awaiting-review
  merge --no-ff                   + changement ajouté à reviewChanges
       |  \                        (branche + worktree conservés)
    ok |   \ échec
       v    v
  remove   merge --abort (le nôtre seulement)
  (sans    pause(raison), branche + worktree conservés
  --force)
  branch -d
```

- **Commit** : `git -C <worktree> add -A` puis `git commit -m "feat(<change>): apply OpenSpec change"` **seulement si** `git status --porcelain` n'est pas vide (un agent qui a déjà committé ne produit pas de commit vide). Le `.gitignore` du projet borne ce qui est ajouté ; la liste des fichiers committés est écrite dans le journal du run.
- **« Travail produit »** = le worktree était sale **ou** `git rev-list --count <branche courante du dépôt>..feature/<change>` > 0. Sinon : pause.
- **Suppression** : `git worktree remove` **sans** `--force`, puis `git branch -d` (et non `-D`) : si le merge n'a pas réellement intégré la branche, git refuse et on le signale.

*Alternatives écartées* : (a) garder `--force` mais committer avant — laisse un chemin destructeur latent si le commit est sauté ; (b) `git merge --squash` — perd l'historique de l'agent et ne change pas le risque ; (c) laisser l'agent committer via le prompt — non déterministe, c'est exactement la situation actuelle.

### D2. Merge : branche courante, sérialisé, n'annule que son propre merge

- Un `mergeMu sync.Mutex` par `Manager` (un `Manager` par workspace, donc par dépôt) sérialise les fusions.
- Avant de fusionner : `git rev-parse -q --verify MERGE_HEAD` dans le dépôt. S'il existe, on ne touche à rien et on met en pause (« un merge est déjà en cours »). `merge --abort` n'est exécuté que si **notre** `git merge` vient d'échouer.
- La cible est la branche **actuellement extraite** du dépôt (comportement actuel conservé), nommée dans le journal du run. On ne fige pas une branche cible à la création du worker : l'utilisateur peut légitimement changer de branche.
- En cas d'échec : pause avec `git`'s stderr tronqué dans la raison ; branche et worktree conservés. La reprise explicite (`/resume`) relance le worker, qui retrouve son worktree (spec « Reprise »), constate que tout est coché, committé, et retente la fusion sans refaire de travail d'agent significatif.

*Alternative écartée* : fusionner dans une branche d'intégration dédiée — plus sûr mais change le modèle mental (l'utilisateur attend le merge dans sa branche) ; à reconsidérer si les conflits sont fréquents.

### D3. Pré-contrôle : le change doit être dans le worktree

Après `Provision` et avant de lancer l'agent : si `<worktree>/openspec/changes/<change>/tasks.md` n'existe pas, pause immédiate (« le changement doit être committé avant d'être lancé »), sans consommer de tour ni d'agent. Le contrôle vit dans `runWorker` (pas dans `Provision`, qui reste purement git).

*Alternatives écartées* : copier le dossier du change dans le worktree — le merge échouerait ensuite (« untracked working tree files would be overwritten ») dans l'arbre principal qui contient les mêmes fichiers non suivis ; committer automatiquement dans l'arbre de l'utilisateur — écriture non demandée dans son dépôt. Le coût est une étape manuelle (`git add/commit` du change) ; signalée explicitement à l'utilisateur.

### D4. Complétion stricte et résultat d'agent en erreur

- `ParseTaskProgress` renvoie `(0,0)` si le fichier est absent : le contrôle devient `total == 0 || done < total` ⇒ pause, avec deux raisons distinctes (liste absente/vide vs tâches restantes).
- `runTurn` remplace `isTurnCompleteLine` par un classement : `turnOK`, `turnError(motif)`, `turnNone`. Pour l'événement `result` de Claude, `is_error == true` ou un `subtype` commençant par `error` ⇒ `turnError` avec `subtype`/`result` comme motif ; `runTurn` renvoie une erreur typée que `runWorker` transforme en pause. Les lignes traduites Gemini/Antigravity (`message_complete`) restent `turnOK` (hors périmètre : l'utilisateur utilise Claude).

### D5. Délais : inactivité pour l'agent, durée maximale pour la validation

- Tour d'agent : `time.AfterFunc`/timer réarmé à **chaque ligne lue** ; à expiration, annulation du contexte du subprocess (`procCancel`) et pause « agent inactif depuis X ». Valeur : **30 min d'inactivité** — Claude n'émet aucune sortie pendant l'exécution d'un outil (un `go test` long), donc une valeur courte provoquerait des faux positifs.
- Validation : `context.WithTimeout` de **20 min** autour de chaque commande + `cmd.WaitDelay = 5s`, pause « validation trop longue ».
- Ce sont des constantes du package `pool` (testables via variable surchargeable). Configuration utilisateur reportée (voir Questions ouvertes).

### D6. Groupes de processus

- Nouveau fichier `session/procgroup_unix.go` (`//go:build unix`) : `applyProcessGroup(cmd)` positionne `SysProcAttr{Setpgid: true}` et `cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }` ; `procgroup_other.go` est un no-op (le projet n'est pas compilé pour Windows aujourd'hui, mais on évite de casser la compilation).
- Appliqué à `StartSubprocess` (donc aussi aux sessions d'exploration, bénéfice gratuit) et à `runValidationCommand`, avec `WaitDelay` pour débloquer `Wait` quand un petit-enfant garde le pipe ouvert.
- `SIGKILL` direct sur le groupe plutôt que `SIGTERM` puis `SIGKILL` : le travail de l'agent est dans des fichiers du worktree (pas d'état à flusher), et un arrêt déterministe vaut mieux qu'un agent qui ignore `SIGTERM`.
- Ordre de teardown corrigé : le `defer` actuel exécute `proc.Wait()` **avant** `procCancel()` (LIFO). Il est remplacé par un seul `defer` : `CloseStdin` → attente bornée (5 s) de la sortie → `procCancel` → `Wait`.

### D7. État des workers : pauses, arrêt, identifiants

- Dans `runWorker`, la closure `pause` (et le chemin de provisionnement) vérifie d'abord `ctx.Err()` : un worker annulé enregistre l'issue `stopped` et **ne** passe **pas** par `pauseWorker`.
- `Manager.Start` réinitialise `pausedWorkers` et `reviewChanges` (les maps sont déjà remises à zéro par `Stop`, le défaut est la course avec le worker qui finit après).
- La suppression dans le `defer` de `runWorker` devient `if m.activeWorkers[w.ID] == w { delete(...) }`.
- `Manager.StopAndWait(ctx)` : `Stop()` puis attente de `m.workers` bornée par `ctx`. `Registry.StopAll(ctx)` l'appelle sur chaque pool ; `main.go` l'appelle avant `srv.Shutdown`.

### D8. Changes en attente de revue

`Manager.reviewChanges map[string]bool` (clé : nom du change), alimentée quand le worker termine en `hitl-review`, lue par `tick` au même endroit que `pausedWorkers`. Volontairement en mémoire : `Stop`/`Start` la vident, donc après un redémarrage le change est redistribué, `Provision` réutilise le worktree, l'agent constate que tout est fait et le worker revient à `awaiting-review` pour un coût d'un tour.

### D9. Worktrees par workspace, reprise de l'ancien emplacement, suppression prudente

- `NewWorktreeController(repoRoot, workspaceID, worktreesRoot)` ; chemin : `<worktreesRoot>/<workspaceID>/wt-<change>`. `worktreesRoot` par défaut `~/.opensp8c/worktrees`, surchargeable par la variable d'environnement `OPENSP8C_WORKTREES_DIR` (lue une fois à la construction du `Manager`) et par le champ pour les tests, qui utilisent désormais `t.TempDir()`.
- **Reprise de l'ancien emplacement** : avant de créer, `Provision` teste si `<worktreesRoot>/wt-<change>` figure dans `git worktree list --porcelain` de ce dépôt sur `feature/<change>` ; si oui, il le renvoie tel quel. Cela protège les worktrees existants (dont ceux avec des modifications non committées).
- **`Remove`** : `git worktree remove <path>` sans `--force`. Si le répertoire n'existe plus, `git worktree prune`. **Plus de `os.RemoveAll` de secours.** En cas d'échec (modifications non committées, verrou), l'erreur remonte.
- **`Discard`** (annulation explicite par l'utilisateur) : seul chemin qui utilise `--force`, et seulement après avoir vérifié que le chemin est un worktree enregistré du dépôt ; sinon erreur.

## Risks / Trade-offs

- **`git add -A` embarque des fichiers parasites** (artefacts de build non ignorés) → mitigation : le `.gitignore` du projet, liste des fichiers committés dans le journal, et le mode `hitl-review` laisse la branche relisible avant fusion.
- **Étape manuelle avant lancement (le change doit être committé)** → friction réelle après `ff`, mais la pause explicite remplace une perte silencieuse ; option d'auto-commit en question ouverte.
- **Délais trop courts ou trop longs** → valeurs généreuses (30 min / 20 min) choisies pour éviter les faux positifs ; constantes faciles à ajuster.
- **`SIGKILL` du groupe** peut laisser un état applicatif incohérent côté outils lancés par l'agent (ex. lockfile) → accepté : c'est le comportement attendu d'un arrêt forcé, et le worktree est conservé.
- **État « en revue » perdu au redémarrage** → coût d'un tour d'agent à la reprise, pas de perte de données.
- **Changement d'emplacement des worktrees** → mitigé par la reprise de l'ancien emplacement ; risque résiduel si un worktree legacy est référencé par un autre dépôt (alors il n'est pas repris et `Provision` échoue en pause lisible).
- **Fusion dans la branche courante** : si l'utilisateur est sur une branche de travail inattendue, le merge y atterrit → le nom de la branche cible est journalisé ; pas de contrainte de branche dans ce change.

## Migration Plan

1. Déployer le backend : aucun changement de données ni d'API. Les worktrees existants sont repris depuis l'ancien emplacement.
2. Avant le premier run en `full-autonomy`, committer le change dans le dépôt (sinon pause explicite).
3. Rollback : revenir au binaire précédent ; les commits créés par la plateforme dans `feature/<change>` restent valides, les worktrees au nouvel emplacement restent des worktrees git ordinaires.

## Open Questions

- Faut-il rendre les délais (inactivité, validation) configurables dans Settings ? Peut attendre un retour d'usage, sans impact sur les specs.
- Faut-il proposer, en option, que la plateforme committe elle-même le change dans l'arbre principal au moment du « Launch » pour supprimer l'étape manuelle de D3 ? À décider après usage réel.
