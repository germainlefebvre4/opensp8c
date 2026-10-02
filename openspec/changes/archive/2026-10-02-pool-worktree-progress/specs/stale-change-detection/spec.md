# Spec Delta

## ADDED Requirements

### Requirement: Activité mesurée sur le worktree du worker actif
Pour un change dont un worker du pool est actif (y compris en pause) et dont le `tasks.md` du worktree contient au moins une tâche, `days_since_activity` et `is_stale` SHALL être calculés à partir de la date de dernière modification de ce `tasks.md` du worktree et non de celui du dépôt principal. Sinon, le calcul sur le dépôt principal reste inchangé.

#### Scenario: Agent qui coche des tâches dans le worktree
- **WHEN** le `tasks.md` du dépôt principal n'a pas été modifié depuis 10 jours mais l'agent a modifié celui du worktree il y a quelques minutes
- **THEN** `days_since_activity` vaut `0` et `is_stale` vaut `false`

#### Scenario: Worktree sans tasks.md exploitable
- **WHEN** un worker est actif mais que le `tasks.md` de son worktree est absent ou vide
- **THEN** `days_since_activity` et `is_stale` sont calculés sur le `tasks.md` du dépôt principal
