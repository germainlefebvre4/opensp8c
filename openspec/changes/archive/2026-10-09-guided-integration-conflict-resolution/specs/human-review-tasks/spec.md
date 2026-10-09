# Spec Delta

## ADDED Requirements

### Requirement: Réouverture d'une tâche de validation humaine par une correction
Lorsqu'une demande de correction est faite avec la réouverture des tâches de validation humaine (voir `change-review-actions`), toute tâche de `tasks.md` portant le marqueur `<!-- human review required -->` et cochée SHALL repasser de `- [x]` ou `- [X]` à `- [ ]`, le reste de la ligne, marqueur compris, étant conservé octet pour octet. Les tâches sans marqueur, les tâches marquées déjà décochées et toutes les autres lignes du fichier SHALL NOT être modifiées. Après réouverture, la tâche SHALL être comptée comme non faite dans `tasks_done` et `tasks_total`, `human_review` SHALL rester vrai, et le contrôle de complétion d'un worker `hitl-review` SHALL continuer à l'ignorer (elle est de validation humaine). L'agent d'implémentation SHALL NOT cocher cette tâche, comme toute tâche marquée ; elle peut ensuite être cochée par l'utilisateur ou, après un verdict `PASS`, par la vérification UI, qui la traite alors comme une tâche marquée non cochée.

#### Scenario: Tâche marquée cochée rouverte
- **WHEN** `tasks.md` contient `- [x] 4.2 Parcours manuel <!-- human review required -->` et qu'une correction demande la réouverture
- **THEN** la ligne devient `- [ ] 4.2 Parcours manuel <!-- human review required -->`

#### Scenario: Autres tâches inchangées
- **WHEN** la réouverture s'applique à un `tasks.md` contenant une tâche d'implémentation cochée, une tâche marquée décochée et une tâche marquée cochée
- **THEN** seule la tâche marquée cochée est modifiée et le reste du fichier est identique

#### Scenario: Compteurs après réouverture
- **WHEN** un `tasks.md` de 10 tâches, toutes cochées dont une marquée, subit la réouverture
- **THEN** `tasks_done` vaut 9, `tasks_total` vaut 10 et la tâche marquée est renvoyée avec `done` à `false` et `human_review` à `true`

#### Scenario: Revérification par la vérification UI
- **WHEN** la vérification UI d'un change dont une tâche marquée a été rouverte conclut `PASS` et déclare cette tâche vérifiée
- **THEN** la tâche est cochée par la plateforme comme n'importe quelle tâche marquée non cochée
