# Spec Delta

## ADDED Requirements

### Requirement: Coche d'une tâche marquée par la vérification UI
Une tâche portant le marqueur de validation humaine PEUT être cochée par la plateforme à l'issue d'une vérification UI réussie (voir `ui-verification-step`), sans intervention de l'utilisateur. Cette coche SHALL suivre les règles du marqueur : la ligne passe en `- [x]` et conserve le commentaire, les compteurs `tasks_done` et `tasks_total` la comptent comme n'importe quelle tâche cochée, et `human_review` reste vrai pour cette tâche. L'agent d'implémentation SHALL continuer à ne jamais cocher une tâche marquée : seule la vérification UI de la plateforme coche une telle tâche automatiquement.

#### Scenario: Tâche cochée par la vérification
- **WHEN** la vérification UI d'un change réussit et déclare vérifiée la tâche `- [ ] 4.2 Parcours manuel <!-- human review required -->`
- **THEN** `tasks.md` de la branche contient `- [x] 4.2 Parcours manuel <!-- human review required -->`

#### Scenario: Lecture après coche
- **WHEN** le détail du change est demandé après cette coche
- **THEN** la tâche est renvoyée avec `done` à `true` et `human_review` à `true`

#### Scenario: Décompte
- **WHEN** un `tasks.md` contient 10 tâches dont 9 cochées, parmi lesquelles une tâche marquée cochée par la vérification UI
- **THEN** `tasks_done` vaut 9 et `tasks_total` vaut 10
