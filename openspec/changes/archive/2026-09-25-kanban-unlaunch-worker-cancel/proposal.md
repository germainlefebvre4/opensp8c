# Proposal: Interruption d'un Worker à la Rétrogradation To Do -> Ready

## Why

Aujourd'hui, lorsqu'un change est dans la colonne **To Do** et qu'un worker de l'Agent Pool est en cours d'exécution dessus, tenter de déplacer la carte vers la colonne **Ready** échoue avec une erreur HTTP 409 Conflict (`"cannot demote: an Agent Pool worker is active on this change"`). 
Ce blocage affecte particulièrement les changes créés avant l'introduction de la colonne Ready (qui sont marqués rétroactivement comme `launched: true` et immédiatement ramassés par l'Agent Pool dès son démarrage), mais aussi tout utilisateur souhaitant retirer une tâche de la file d'attente d'exécution sans devoir couper l'intégralité du pool.
De plus, si le fichier `.openspec.yaml` d'une tâche legacy est absent du disque, `SetLaunched` échoue en erreur 500, interceptée aveuglément par l'interface avec ce même message d'erreur.

## What Changes

- **Annulation ciblée de worker (`force=true`)** : `PATCH /api/workspaces/{id}/changes/{name}/unlaunch?force=true` interrompt le worker actif sur le change spécifié (`cancel()`), tout en conservant son code et ses commits dans son git worktree (`~/.opensp8c/worktrees/wt-<name>`), puis rétrograde le change vers la colonne `Ready` (`launched: false`).
- **Confirmation utilisateur dans le Kanban** : Lors d'un drag & drop de `To Do` vers `Ready` d'une carte ayant un worker actif, un dialogue de confirmation explicite demande à l'utilisateur s'il souhaite interrompre le worker et rétrograder la tâche.
- **Bouton d'arrêt direct sur la carte** : Ajout d'un bouton d'action d'arrêt directement sur `ChangeCard` à côté du badge animé CPU lorsqu'un worker est actif, permettant d'interrompre le worker et de rétrograder la tâche en `Ready` en un clic.
- **Résilience de `SetLaunched`** : `SetLaunched` crée le fichier `.openspec.yaml` s'il est absent du dossier du change au lieu d'échouer avec `os.ErrNotExist`.
- **Typage des erreurs UI** : L'interface distingue désormais les erreurs 409 (worker actif non forcé) des autres erreurs serveur ou réseau (500).

## Capabilities

### Modified Capabilities
- `kanban-ready-column`: Ajout du paramètre `force=true` sur l'action de rétrogradation To Do -> Ready permettant d'interrompre un worker actif tout en conservant le code du git worktree, dialogue de confirmation UI, et création automatique de `.openspec.yaml` si absent.
- `kanban-board`: Ajout d'un bouton d'arrêt direct du worker sur la carte Kanban (`ChangeCard`) quand `worker_active` est vrai, rétrogradant la tâche vers `Ready`.

## Impact

- **Backend** :
  - `internal/pool/manager.go` : ajout de `CancelWorkerForChange(name string) bool`.
  - `internal/api/handlers/kanban.go` : gestion du paramètre `force` dans `KanbanHandler.Unlaunch`.
  - `internal/openspec/change.go` : tolérance du `.openspec.yaml` manquant dans `SetLaunched`.
- **Frontend** :
  - `src/lib/api.ts` : support du paramètre `force` dans `unlaunchChange`.
  - `src/pages/KanbanPage.tsx` : affichage du dialogue de confirmation d'interruption du worker au drop vers `Ready` et gestion affinée des erreurs.
  - `src/components/ChangeCard.tsx` : bouton d'arrêt rapide du worker à côté du badge CPU.
  - Traductions `i18n` (en/fr) pour les dialogues et messages associés.
