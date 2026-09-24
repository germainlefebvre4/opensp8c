# Proposal

## Why

Le pool d'agents sélectionne déjà les changements de la colonne Todo, provisionne un worktree git isolé et affiche le statut de chaque worker — mais l'invocation de l'agent elle-même (`invokeAgentApply`/`invokeAgentHeal` dans `backend/internal/pool/worker.go`) est un stub qui se contente d'attendre 2 secondes sans jamais exécuter de travail réel. Lancer le pool ne fait donc jamais progresser un changement : `tasks.md` n'est jamais modifié, et en mode `full-autonomy` le worker merge quand même une branche vide avant de recommencer au tick suivant.

## What Changes

- Remplacer les stubs `invokeAgentApply`/`invokeAgentHeal` par une invocation réelle d'un agent CLI (réutilisation de `session.StartSubprocess` + `session.Manager.ResolveAgentConfig`, sur le modèle déjà utilisé pour le Fast-Forward dans `handlers/explore.go`), avec un seul appel par tentative lançant `/opsx:apply <changeName>` : la boucle tâche-par-tâche est déléguée à la session agent elle-même, pas au code Go.
- Donner à `pool.Manager` accès à la configuration d'agent par workspace/change (`preferences.Service`, `DefaultAgent`, `Env`), dont il ne dispose pas aujourd'hui.
- Ajouter une vérification de complétion de `tasks.md` après une validation réussie, avant toute finalisation : si des tâches restent décochées malgré un `go test` vert, le worker ne merge pas silencieusement un travail incomplet et passe en statut `paused` au lieu de transitionner.
- Faire remonter la sortie de l'agent en cours d'exécution jusqu'au panneau d'état du pool, pour que l'utilisateur voie l'activité réelle d'un worker plutôt que son seul statut (`working`/`testing`/`healing`).
- **Hors périmètre**, traité séparément : le filtre de dépendances du scheduler qui ignore les statuts `to-explore`/`ready`, et le statut `to-review` actuellement jamais produit par le backend.

## Capabilities

### New Capabilities

(aucune)

### Modified Capabilities

- `agent-pool-orchestrator`: la boucle d'auto-correction invoque réellement un agent CLI (au lieu d'un stub simulé) et n'autorise la finalisation (merge ou transition) qu'après vérification que `tasks.md` ne contient plus de tâche non cochée.
- `agent-pool-ui`: le panneau d'état du pool actif expose l'activité en cours de chaque worker (sortie de l'agent), en plus de son statut.

## Impact

- Backend : `backend/internal/pool/worker.go` (invocation réelle, vérification de complétion), `backend/internal/pool/manager.go` (accès à la config d'agent par workspace), `backend/internal/pool/types.go` (champ d'activité/transcript sur `Worker`), `backend/internal/session` (réutilisation de `StartSubprocess`/`ResolveAgentConfig`), `backend/internal/openspec/change.go` (réutilisation de la logique de progression de `tasks.md`).
- Frontend : panneau d'état du pool actif (affichage de l'activité par worker).
- Pas de migration de données ; changement purement additif côté API (nouveau champ dans la réponse de statut du pool).
