# Proposal

## Why

En mode `hitl-review`, l'implémentation d'un change laisse souvent des tâches que seul l'utilisateur peut valider (parcours manuel dans l'application, vérification visuelle, commande à lancer). Le worker n'a pas le droit de finaliser tant qu'une tâche reste décochée : il passe en `paused` et la carte reste bloquée dans **In Progress**, alors que le travail de l'agent est terminé et que c'est à l'utilisateur de jouer. Le bon endroit pour attendre cette validation est **To Review**, où le travail est relisible (diff) et où, avec le change `tasks-follow-the-branch`, les tâches peuvent être cochées sur la bonne branche.

## What Changes

- **Marqueur de tâche humaine** : une tâche de `tasks.md` portant le commentaire HTML `<!-- human review required -->` est une tâche de validation humaine. Le marqueur est invisible au rendu Markdown ; l'interface l'affiche comme un badge et ne montre pas le commentaire dans le texte de la tâche.
- **Tour de triage (hitl-review)** : après l'application des tâches, si des tâches restent décochées sans marqueur, le worker envoie un unique tour à l'agent : pour chaque tâche restante, la terminer, ou la marquer comme validation humaine. L'ordre devient : application, triage, validation (avec guérison), contrôle de complétion, commit.
- **Contrôle de complétion** : en `hitl-review`, les tâches décochées portant le marqueur ne comptent plus comme restantes ; une tâche décochée sans marqueur reste un motif de pause, comme aujourd'hui. En `full-autonomy`, rien ne change : toutes les tâches doivent être cochées.
- **Directive donnée au worker** : en `hitl-review`, l'agent reçoit dans son prompt système la consigne de ne jamais cocher une tâche marquée et de marquer, plutôt que de cocher, une tâche qui exige un humain.
- **Traçabilité** : chaque tâche marquée par le triage est consignée dans l'activité du change.
- **Approbation conditionnée** : `POST …/review/approve` refuse en `409` (code `tasks_pending`, avec le nombre de tâches restantes) tant que le `tasks.md` de la branche contient une tâche décochée. Dans l'interface, « Approuver & Fusionner » reste visible mais désactivé avec l'info-bulle « N tâches à valider », et le drop d'une carte To Review sur Done est refusé avec une notification.
- **Reprise après correction** : le contrôle de complétion de la reprise suit la même règle (les tâches marquées restantes n'empêchent pas le retour en revue).
- **Docs** : `docs/opensp8c/workflows.md` décrit le flux (triage, review avec tâches à valider, approbation conditionnée).

Dépend de `tasks-follow-the-branch` (lecture des tâches de la branche en revue, toggle committé sur la branche, compteurs de carte issus de la branche) : sans lui, l'utilisateur ne pourrait pas cocher les tâches en revue et les compteurs afficheraient le dépôt principal.

Hors périmètre : lancement de l'application depuis la revue (tester l'application) ; badge « à valider » sur la carte du Kanban ; prise en compte du marqueur dans « Reprendre en finalisant » (qui exige toujours que tout soit coché) ; verrou technique contre l'abus du marqueur par l'agent.

## Capabilities

### New Capabilities

- `human-review-tasks`: marqueur de tâche de validation humaine dans `tasks.md`, sa lecture (texte nettoyé, indicateur `human_review`) et son effet sur le décompte des tâches restantes.

### Modified Capabilities

- `agent-pool-orchestrator`: contrôle de complétion tenant compte du marqueur en `hitl-review`, tour de triage, directive de validation humaine, traçabilité des tâches marquées.
- `change-review-actions`: approbation refusée tant que des tâches restent à valider, interface correspondante, reprise après correction alignée sur la nouvelle règle de complétion.
- `kanban-drag-drop`: le drop To Review vers Done est refusé tant que des tâches restent à valider.
- `kanban-change-detail`: les tâches de validation humaine sont signalées dans la liste des tâches du DetailPanel.

## Impact

- Backend : `internal/openspec/change.go` (`Task.HumanReview`, lecture du marqueur, statistiques de tâches), `internal/pool/worker.go` (triage, contrôle de complétion, directive), `internal/pool/review_actions.go` (refus d'approbation), `internal/api/handlers/review_actions.go` (code `tasks_pending`).
- Frontend : `DetailPanel.tsx` (badge, bouton désactivé), `KanbanPage.tsx` (drop refusé), `lib/api.ts` (code d'erreur), `hooks/useChangeDetail.ts` (champ `human_review`), locales fr/en (`dialogs`, `detailPanel`).
- API : champ optionnel `human_review` sur les tâches du détail ; nouvelle réponse `409` `tasks_pending` (avec `remaining`) pour l'approbation.
- Docs : `docs/opensp8c/workflows.md`.
