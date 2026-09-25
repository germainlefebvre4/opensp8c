# Design: Annulation de Worker et Rétrogradation To Do -> Ready

## Context

L'Agent Pool exécute des agents autonomes ou supervisés dans des git worktrees isolés (`~/.opensp8c/worktrees/wt-<name>`). 
Chaque worker possède son propre `context.CancelFunc` stocké sur sa structure [`Worker`](file:///home/glefebvre/Documents/Dev/Perso/OpenSpec/opensp8c/backend/internal/pool/types.go#L37).
Actuellement :
1. [`KanbanHandler.Unlaunch`](file:///home/glefebvre/Documents/Dev/Perso/OpenSpec/opensp8c/backend/internal/api/handlers/kanban.go#L228) vérifie `h.activeWorkerChanges(id)[name]`. Si un worker est actif, la requête est immédiatement rejetée avec HTTP 409 Conflict.
2. [`pool.Manager`](file:///home/glefebvre/Documents/Dev/Perso/OpenSpec/opensp8c/backend/internal/pool/manager.go) n'expose qu'une méthode globale `Stop()`, annulant tous les workers simultanément.
3. [`openspec.SetLaunched`](file:///home/glefebvre/Documents/Dev/Perso/OpenSpec/opensp8c/backend/internal/openspec/change.go#L296) tente de lire `.openspec.yaml` avec `os.ReadFile`. Si le fichier n'existe pas (ex. tâche legacy créée avant la colonne Ready), la fonction renvoie une erreur brute entraînant une 500.
4. Dans [`KanbanPage.tsx`](file:///home/glefebvre/Documents/Dev/Perso/OpenSpec/opensp8c/frontend/src/pages/KanbanPage.tsx#L233), tout échec de `unlaunchChange` affiche le toast d'erreur `"Cannot demote: an Agent Pool worker is active..."` sans distinction du code HTTP.

## Goals / Non-Goals

**Goals :**
- Permettre l'interruption ciblée d'un worker sur un change donné via `PATCH .../unlaunch?force=true`.
- Préserver le git worktree et la branche `feature/<change>` lors de l'interruption pour ne pas perdre le code et les commits déjà réalisés par l'agent.
- Proposer un dialogue de confirmation dans l'interface Kanban lors d'un glisser-déposer vers `Ready` d'une carte ayant un worker actif.
- Ajouter un bouton d'arrêt direct sur la [`ChangeCard`](file:///home/glefebvre/Documents/Dev/Perso/OpenSpec/opensp8c/frontend/src/components/ChangeCard.tsx) à côté de l'indicateur CPU pour interrompre le worker et rétrograder vers `Ready`.
- Rendre [`SetLaunched`](file:///home/glefebvre/Documents/Dev/Perso/OpenSpec/opensp8c/backend/internal/openspec/change.go#L296) tolérant à l'absence de `.openspec.yaml` en créant le fichier si nécessaire.
- Améliorer la gestion d'erreur dans l'UI pour distinguer les erreurs 409 des erreurs 500 / réseau.

**Non-Goals :**
- Réimplémenter la boucle globale de planification de l'Agent Pool.
- Permettre de laisser une tâche dans `To Do` avec un worker arrêté sans la rétrograder (car le scheduler de l'Agent Pool la relancerait immédiatement au tick suivant).
- Gérer l'annulation sur des changements situés dans la colonne `In Progress` ou `To Review`.

## Decisions

### 1. Méthode `CancelWorkerForChange(name string) bool` sur `pool.Manager`
- **Choix** : Ajouter une méthode `CancelWorkerForChange(name string) bool` dans [`internal/pool/manager.go`](file:///home/glefebvre/Documents/Dev/Perso/OpenSpec/opensp8c/backend/internal/pool/manager.go). Sous verrou `m.mu.Lock()`, elle itère sur `m.activeWorkers` à la recherche du change correspondant et appelle son `w.CancelFunc()`.
- **Rationnel** : Évite d'interrompre les autres workers du pool qui travaillent en parallèle sur d'autres changes.
- **Alternatives considérées** : Arrêter tout le pool puis redémarrer (trop agressif et perturbe les autres tâches).

### 2. Paramètre `force=true` sur l'endpoint `PATCH /unlaunch`
- **Choix** : Réutiliser la route existante [`PATCH /api/workspaces/{id}/changes/{name}/unlaunch`](file:///home/glefebvre/Documents/Dev/Perso/OpenSpec/opensp8c/backend/internal/api/handlers/kanban.go#L228) en acceptant un paramètre optionnel `force=true`.
- **Comportement** :
  - Sans `force=true` : conserve le comportement de sécurité existant (409 Conflict si un worker est actif).
  - Avec `force=true` : invoque `CancelWorkerForChange(name)`, passe à `SetLaunched(changeDir, false)`, et renvoie 204.
- **Alternatives considérées** : Créer un endpoint séparé `POST .../changes/{name}/cancel` (nécessite d'enchaîner deux requêtes côté frontend avec risque d'incohérence entre annulation et rétrogradation).

### 3. Conservation du code dans le Git Worktree
- **Choix** : Lors de l'annulation du worker, on **ne supprime pas** le worktree (`wt.CleanupDiscard` n'est pas appelé). Le subprocess de l'agent est simplement arrêté.
- **Rationnel** : Le code produit dans la branche `feature/<name>` reste intact. Lors d'un futur relancement (promotion `Ready -> To Do`), [`WorktreeController.Provision`](file:///home/glefebvre/Documents/Dev/Perso/OpenSpec/opensp8c/backend/internal/pool/worktree.go#L30) réutilise la branche existante et ses commits. Si le code était directement sur la branche principale hors worktree, les modifications non validées seraient nettoyées.

### 4. Bouton d'action direct sur la carte Kanban
- **Choix** : Ajouter une icône Stop cliquable sur [`ChangeCard.tsx`](file:///home/glefebvre/Documents/Dev/Perso/OpenSpec/opensp8c/frontend/src/components/ChangeCard.tsx) à côté du badge CPU lorsque `worker_active` est vrai.
- **Action** : Un clic ouvre la modale de confirmation d'interruption du worker. À la confirmation, l'API `unlaunchChange(workspaceId, changeName, true)` est appelée.
- **Rationnel** : Stopper le worker tout en laissant la carte en `To Do` conduirait le scheduler à la relancer automatiquement au tick suivant (dans les 5 secondes). Lier l'arrêt du worker à la rétrogradation vers `Ready` garantit la cohérence du statut.

### 5. Résilience de `SetLaunched`
- **Choix** : Dans [`internal/openspec/change.go`](file:///home/glefebvre/Documents/Dev/Perso/OpenSpec/opensp8c/backend/internal/openspec/change.go#L296), si `os.ReadFile(metaPath)` renvoie une erreur telle que `os.IsNotExist(err)`, initialiser une structure `openspecMeta{Schema: "spec-driven"}` vide au lieu de retourner l'erreur.
- **Rationnel** : Corrige définitivement le bug des tâches créées sans `.openspec.yaml`.

## Risks / Trade-offs

- **Délai d'arrêt du subprocess CLI** : L'annulation de contexte (`procCtx`) ferme `stdin` et attend la sortie du processus. Si un modèle est en train de streamer une réponse, cela peut prendre quelques centaines de millisecondes.
  - *Atténuation* : `SetLaunched` écrit immédiatement la nouvelle configuration et l'invalidation React Query / SSE met à jour l'interface immédiatement.
- **Course concurrente lors de la complétion du worker** : Le worker pourrait se terminer naturellement juste au moment où l'utilisateur clique sur confirmer.
  - *Atténuation* : `CancelWorkerForChange` renvoie un booléen indicatif ; même si le worker s'est terminé entre-temps, `SetLaunched` s'exécute quand même avec succès (idempotent).
