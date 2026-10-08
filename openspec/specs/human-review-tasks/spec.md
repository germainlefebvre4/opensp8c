# human-review-tasks Specification

## Purpose

Définit le marqueur qui distingue, dans `tasks.md`, une tâche de validation humaine (que l'agent ne peut pas cocher) d'une tâche d'implémentation, sa lecture par le backend et son effet sur le décompte des tâches restantes.

## Requirements

### Requirement: Marqueur de tâche de validation humaine
Une tâche de `tasks.md` (ligne commençant par `- [ ]`, `- [x]` ou `- [X]`) SHALL être une tâche de validation humaine lorsque sa ligne contient le commentaire HTML `<!-- human review required -->`. La reconnaissance SHALL ignorer la casse et les espaces autour du texte du commentaire (`<!--human review required-->` est reconnu) et SHALL accepter le marqueur n'importe où sur la ligne de la tâche. Le marqueur SHALL NE PAS changer la façon dont la tâche est comptée comme cochée ou non, ni la façon dont elle est cochée (le toggle conserve la ligne entière, marqueur compris). Un marqueur situé sur une ligne qui n'est pas une tâche SHALL être ignoré.

#### Scenario: Tâche marquée
- **WHEN** `tasks.md` contient la ligne `- [ ] 4.2 Parcours manuel de la navigation <!-- human review required -->`
- **THEN** cette tâche est une tâche de validation humaine, décochée

#### Scenario: Marqueur tolérant
- **WHEN** une tâche contient `<!--Human Review Required-->`
- **THEN** elle est reconnue comme tâche de validation humaine

#### Scenario: Marqueur hors tâche
- **WHEN** une ligne qui ne commence pas par `- [` contient le commentaire
- **THEN** aucune tâche n'est marquée

#### Scenario: Tâche cochée conservant son marqueur
- **WHEN** l'utilisateur coche une tâche marquée
- **THEN** la ligne passe en `- [x]` et conserve le commentaire

### Requirement: Lecture des tâches de validation humaine
Chaque tâche renvoyée par `GET /api/workspaces/{id}/changes/{name}` SHALL porter l'indicateur `human_review` (`true` pour une tâche marquée, absent ou `false` sinon). Le texte `text` de la tâche SHALL NE PAS contenir le commentaire du marqueur : le texte renvoyé est celui de la ligne sans le marqueur, espaces de bordure retirés. L'entrée d'activité d'un toggle (`kanban.task_toggled`) SHALL également désigner la tâche par son texte sans marqueur.

#### Scenario: Détail d'un change avec une tâche marquée
- **WHEN** le détail est demandé pour un change dont `tasks.md` contient `- [ ] 4.2 Parcours manuel <!-- human review required -->`
- **THEN** la tâche est renvoyée avec `text` valant `4.2 Parcours manuel`, `done` à `false` et `human_review` à `true`

#### Scenario: Tâche ordinaire
- **WHEN** une tâche ne porte pas le marqueur
- **THEN** `human_review` est absent ou `false` et `text` est inchangé

#### Scenario: Activité d'un toggle
- **WHEN** l'utilisateur coche une tâche marquée
- **THEN** l'entrée d'activité mentionne le texte de la tâche sans le commentaire

### Requirement: Décompte des tâches restantes distinguant les tâches humaines
Le backend SHALL pouvoir distinguer, parmi les tâches non cochées d'un `tasks.md`, celles qui portent le marqueur de validation humaine et celles qui ne le portent pas. Les compteurs `tasks_done` et `tasks_total` exposés aux clients SHALL continuer à compter toutes les tâches, marquées ou non : une tâche marquée décochée est une tâche non faite pour la progression affichée.

#### Scenario: Compteurs inchangés
- **WHEN** un `tasks.md` contient 10 tâches dont 8 cochées et 2 marquées décochées
- **THEN** `tasks_done` vaut 8 et `tasks_total` vaut 10

#### Scenario: Distinction des restantes
- **WHEN** un `tasks.md` contient 3 tâches décochées dont 2 marquées
- **THEN** le backend sait que 2 tâches restantes sont de validation humaine et 1 ne l'est pas

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
