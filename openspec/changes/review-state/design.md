# Design

## Context

Le statut Kanban est calculé par `deriveStatus` (`openspec/change.go`) à partir du seul `tasks.md` du workspace principal et de l'état « lancé ». Or le travail d'un worker vit dans `feature/<change>` et son worktree : tant que rien n'est fusionné, main reste à `0/N` et le change reste en `todo`. `Manager.reviewChanges` (map en RAM) n'existe que pour que le dispatcher saute le change, et `Stop`/`Start` la vident. Voir `proposal.md` - Why.

Contraintes observées :
- `openspec.ListChanges` a cinq appelants : tick du dispatcher (`pool/manager.go`), handlers Kanban, compteurs de workspaces (`handlers/workspace.go`), `explore.go` et `activity/gitwatch.go`. Un statut de revue calculé ailleurs que dans `ListChanges` divergerait entre eux.
- `openspec` est importé par `pool` : `openspec` ne peut pas importer `pool`.
- Le worker enregistre déjà un état par change dans la configuration git du dépôt (`branch.feature/<change>.opensp8c-base`, `worktree.go`), que git supprime avec la branche.
- Le scheduler traite déjà `to-review` comme dépendance en attente (`scheduler.go`) et ne distribue que le statut `todo`.
- `pool-worktree-progress` (livré) applique, dans les handlers Kanban, une surcouche `ApplyWorktreeProgress` qui réécrit `KanbanStatus` d'un change tenu par un worker, y compris en pause (`activeWorkerChanges`), en lisant le `tasks.md` du worktree. Elle s'applique après `ListChanges`/`loadChange`, donc après la dérivation du marqueur.

## Goals / Non-Goals

**Goals:**
- Un état « en revue » durable (redémarrages du pool et du backend) qui n'ajoute aucun fichier modifié dans le workspace.
- Un seul point de dérivation du statut, utilisé par tous les consommateurs.
- Retirer `reviewChanges` sans changer le comportement visible du dispatcher.

**Non-Goals:**
- Lever le marqueur (approbation, correction, suppression) : c'est `review-actions`.
- Lire le contenu de la branche (`review-panel`).
- La progression en direct d'un change tenu par un worker : déjà livrée par `pool-worktree-progress`.

## Decisions

**1. Le marqueur vit dans la configuration git, attaché à la branche.** Clé `branch.feature/<change>.opensp8c-review`, valeur : horodatage RFC 3339 de l'entrée en revue (utile à l'affichage futur). Alternatives écartées :
- *Flag dans `.openspec.yaml`* (comme `launched`) : modifie l'arbre du dépôt principal, donc salit `git status`, peut gêner la fusion de la branche et n'est pas supprimé avec la branche.
- *Dérivation depuis le journal de runs* (dernier `pool_run_end`) : aucune écriture, mais rien n'invalide l'état après une approbation ou une correction, et le journal est facultatif (store nul) donc non fiable.
- La configuration git est commune à tous les worktrees : un marqueur posé depuis le worktree est lu depuis le dépôt principal.

**2. La dérivation vit dans `openspec.ListChanges` et `GetChangeDetail`.** Une lecture unique `git config --get-regexp '^branch\..*\.opensp8c-review$'` dans le dossier du workspace donne l'ensemble des changes en revue ; `loadChange` applique ensuite la règle de priorité (`to-review` ⟶ prime sur `ready`/`todo`/`in-progress`, jamais sur `archived`). La constante du suffixe de clé est définie dans `openspec` et réutilisée par `pool` pour l'écriture, ce qui évite un import cyclique et une signature de `ListChanges` modifiée. Si git échoue ou si le dossier n'est pas un dépôt, l'ensemble est vide et aucune erreur n'est remontée. Un marqueur n'est retenu que si `feature/<change>` existe (via la même sortie de config : git ne garde la section que tant que la branche existe, un contrôle explicite couvre le cas d'une section créée à la main).

**3. Priorité du marqueur sur la surcouche worktree.** Un change en revue n'est tenu par aucun worker (le worker est retiré de `activeWorkers` à sa fin, le marqueur est levé avant toute reprise), donc les deux mécanismes ne se rencontrent pas en pratique. Par sécurité, `ApplyWorktreeProgress` ne modifie jamais le statut d'un change déjà `to-review` (il peut toujours mettre à jour la progression) : la priorité est ainsi garantie par le code et non par la seule absence de cas.

**4. Écriture dans `WorktreeController`** : `MarkReview(change)` (config git), `ClearReview(change)` et `HasReview(change)` à côté de `recordBase`/`BaseBranch`. Contrairement à `recordBase` (best effort), l'échec de `MarkReview` n'est pas silencieux : le worker passe en `paused` avec une raison lisible, pour ne jamais présenter comme « en revue » un change dont l'état n'est pas enregistré.

**5. Suppression de `reviewChanges`.** Plus de map, plus de filtre dans `tick` : `GetRunnableChanges` ne retient que `todo`, et un change `to-review` n'en fait pas partie. Les dépendants restent bloqués (le scheduler compte déjà `to-review` comme en attente). `Stop`/`Start` n'ont plus rien à nettoyer côté revue.

**6. Événement SSE explicite.** Poser un marqueur ne touche aucun fichier sous `openspec/`, le watcher n'émet donc rien. Le `Manager` publie `watcher.Event{Type: "change_updated", Name: change}` via son `Broadcaster` après `MarkReview`, ce qui déclenche l'invalidation existante côté frontend (liste et détail). La levée (change `review-actions`) utilisera le même chemin.

**7. Compteurs.** `handlers/workspace.go` initialise `task_counts` avec la clé `to-review` (aujourd'hui une clé inconnue serait ajoutée dynamiquement mais absente à 0) et le sidebar affiche un badge bleu pour `to-review`, hors `done`.

## Risks / Trade-offs

- [Un appel `git` supplémentaire à chaque `ListChanges` (tick du dispatcher, compteurs rafraîchis toutes les 15 s, handlers)] → un seul `git config --get-regexp` par appel, ignoré si `.git` est absent ; si la mesure montre un coût, mise en cache par date de modification de `.git/config`.
- [Contention du verrou `config.lock` de git entre l'écriture du marqueur et d'autres écritures de configuration (`recordBase`)] → l'échec est remonté en pause du worker, et la reprise explicite du worker (existante) relance la finalisation, qui réécrit le marqueur de façon idempotente.
- [Marqueur persistant sans action disponible tant que `review-actions` n'est pas livré] → le change reste en To Review ; l'utilisateur peut le supprimer (suppression existante). Livrer `review-actions` juste après, ou masquer les boutons inactifs en attendant.
- [Changement de comportement visible : un change redémarré en revue n'est plus relancé] → c'est l'objectif ; la spec de l'orchestrateur est modifiée en conséquence (marquée BREAKING dans le proposal).
- [Branche supprimée à la main sans passer par le backend] → git supprime la section de configuration de la branche avec elle ; un marqueur orphelin éventuel est ignoré par la dérivation.
