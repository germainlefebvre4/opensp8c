# Spec Delta

## ADDED Requirements

### Requirement: Issue `awaiting-verification` d'un worker
Lorsqu'un worker pose un marqueur de vérification à la fin de l'implémentation d'un change (voir `verification-stage`), son exécution SHALL se terminer avec l'issue `awaiting-verification`, enregistrée dans le marqueur de fin du run `pool`. Cette issue SHALL libérer le slot du worker comme toute fin d'exécution, sans pause : le worker n'apparaît ni parmi les workers actifs ni parmi les workers en pause, et le change n'est plus tenu par un worker. Cette issue ne SHALL PAS annuler le worktree ni la branche du change.

#### Scenario: Issue enregistrée
- **WHEN** un worker termine un change dont la vérification est activée
- **THEN** le marqueur de fin du run `pool` porte `awaiting-verification`

#### Scenario: Slot libéré
- **WHEN** l'issue `awaiting-verification` est atteinte avec une taille de pool de 1
- **THEN** le tick suivant peut démarrer un worker sur un autre change

#### Scenario: Worktree conservé
- **WHEN** un worker termine avec l'issue `awaiting-verification`
- **THEN** le worktree et la branche `feature/<change>` existent toujours

### Requirement: Dispatch d'un change dont la vérification a réussi
En plus des changes de la colonne To Do dont les dépendances sont satisfaites, le dispatcher SHALL proposer au tick les changes dont le marqueur de vérification vaut `passed` et qu'aucun worker ne tient, sans vérifier leurs dépendances, qui l'étaient déjà au démarrage du change. Le worker démarré SHALL être `finalizeOnly` et SHALL lever le marqueur à son démarrage. La limite de taille du pool SHALL s'appliquer à ces workers comme aux autres.

#### Scenario: Dispatch d'un change vérifié
- **WHEN** un change porte le marqueur `passed` et qu'un slot est libre
- **THEN** le tick démarre un worker `finalizeOnly` pour ce change et lève le marqueur

#### Scenario: Pas de double dispatch
- **WHEN** un worker, actif ou en pause, tient déjà le change
- **THEN** le tick ne démarre pas de second worker pour ce change

#### Scenario: Limite de taille
- **WHEN** deux changes portent `passed` et que la taille du pool est 1
- **THEN** un seul worker de finalisation démarre par slot libre, l'autre change attend
