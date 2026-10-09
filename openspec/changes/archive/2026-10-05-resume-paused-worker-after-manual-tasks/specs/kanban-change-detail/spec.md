# Spec Delta

## ADDED Requirements

### Requirement: Actions de reprise dans le DetailPanel d'un change en pause
`GET /api/workspaces/{id}/changes/{name}` SHALL exposer, lorsqu'un worker du pool tient le change, l'identifiant du worker (`worker_id`) et, s'il est en pause, sa raison de blocage lisible (`worker_blocked_reason`). Le `DetailPanel` d'un change avec `worker_paused = true` SHALL afficher un bandeau contenant cette raison de blocage et deux boutons, « Reprendre » et « Reprendre en finalisant », ayant le même effet que sur la carte et dans le panneau d'état du pool. « Reprendre en finalisant » SHALL être désactivé tant que `tasks_done` est inférieur à `tasks_total`, avec le nombre de tâches restantes indiqué, et SHALL devenir actif dès que la dernière tâche est cochée depuis ce panneau, sans rechargement de la page. Un échec de reprise SHALL afficher le message du backend dans le bandeau. Le bandeau SHALL NOT apparaître pour un change sans worker en pause.

#### Scenario: Bandeau d'un change en pause
- **WHEN** l'utilisateur ouvre le DetailPanel du changement `add-user-auth`, dont le worker est en pause avec la raison « Validation réussie mais tâches restantes incomplètes (9/10) dans tasks.md »
- **THEN** le panneau affiche cette raison, un bouton « Reprendre » actif et un bouton « Reprendre en finalisant » désactivé indiquant 1 tâche restante

#### Scenario: Cocher la dernière tâche débloque la finalisation
- **WHEN** l'utilisateur coche la dernière tâche depuis le DetailPanel d'un change dont le worker est en pause
- **THEN** la tâche est enregistrée dans le worktree du worker (voir `task-toggle`) et « Reprendre en finalisant » devient actif

#### Scenario: Reprise en finalisant depuis le détail
- **WHEN** l'utilisateur clique sur « Reprendre en finalisant » dans le bandeau
- **THEN** l'application demande la reprise avec `finalize_only`, et le bandeau disparaît dès que le worker n'est plus en pause

#### Scenario: Pas de bandeau sans pause
- **WHEN** l'utilisateur ouvre le DetailPanel d'un change tenu par un worker actif ou par aucun worker
- **THEN** aucun bandeau ni bouton de reprise n'est affiché
