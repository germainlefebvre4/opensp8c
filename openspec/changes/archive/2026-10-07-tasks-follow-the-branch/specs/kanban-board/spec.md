# Spec Delta

## ADDED Requirements

### Requirement: Compteurs de tâches issus de la branche et indicateur de branche
Pour chaque change de `GET /api/workspaces/{id}/changes` qui porte une branche `feature/<change>` et qu'aucun worker du pool ne tient, `tasks_done` et `tasks_total` SHALL provenir du `tasks.md` de la branche (le worktree du change s'il existe, sinon la branche) lorsque celui-ci contient au moins une tâche, de sorte que la carte affiche la progression réelle du travail y compris en To Review. La colonne (`kanban_status`) SHALL NOT être recalculée à partir de ces compteurs : elle reste dérivée du marqueur de revue, de l'état « lancé » et du `tasks.md` du dépôt principal, de sorte qu'une branche entièrement cochée ne place jamais un change en Done et ne lui offre pas l'action de synchronisation et d'archivage. Pour un change tenu par un worker, la surcharge existante (compteurs et colonne du worktree) SHALL rester inchangée. Chaque change de la liste SHALL aussi exposer `has_branch` (booléen, `true` lorsque `feature/<change>` existe, absent ou `false` sinon). La détection des branches SHALL se faire par une lecture groupée des références du dépôt, sans appel git par change, et le contenu des branches SHALL n'être lu que pour les changes qui en portent une.

#### Scenario: Carte d'un change en revue
- **WHEN** un change en revue a 10 tâches cochées sur 10 dans sa branche et `0/10` dans le dépôt principal
- **THEN** sa carte est en To Review et affiche « 10 / 10 »

#### Scenario: Branche entièrement cochée sans marqueur
- **WHEN** un change non lancé porte une branche à 10 tâches cochées sur 10, sans worker ni marqueur de revue
- **THEN** sa carte reste dans la colonne dérivée du dépôt principal (Ready), affiche « 10 / 10 » et n'est pas proposée à l'archivage

#### Scenario: Change rétrogradé avec travail conservé
- **WHEN** un change rétrogradé en Ready par un arrêt forcé porte une branche à 7 tâches cochées sur 10
- **THEN** sa carte reste en Ready, affiche « 7 / 10 » et `has_branch` vaut `true`

#### Scenario: Change sans branche
- **WHEN** un change n'a pas de branche `feature/<change>`
- **THEN** ses compteurs proviennent du `tasks.md` du dépôt principal et `has_branch` est absent ou `false`

#### Scenario: Change tenu par un worker
- **WHEN** un worker tient le change
- **THEN** les compteurs et la colonne suivent le worktree du worker, comme avant

#### Scenario: Branche sans liste de tâches
- **WHEN** la branche du change existe mais que son `tasks.md` est absent ou sans tâche
- **THEN** les compteurs proviennent du dépôt principal et `has_branch` vaut `true`
