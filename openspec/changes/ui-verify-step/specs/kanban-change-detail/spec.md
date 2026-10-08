# Spec Delta

## ADDED Requirements

### Requirement: Captures de la vérification UI dans le DetailPanel
Lorsque le bandeau de vérification du DetailPanel affiche le rapport d'une étape `ui` (voir `verification-stage`), il SHALL afficher sous le rapport les preuves listées par `GET …/verification/report` sous forme de vignettes, chacune servie par `GET …/verification/artifacts/{run}/{name}`. Un clic sur une vignette SHALL l'afficher en grand dans une visionneuse que l'utilisateur peut fermer. Lorsque le rapport n'a pas de preuve, aucune zone de vignettes ne SHALL être affichée. Le bandeau SHALL aussi afficher les lignes `TASK-VERIFIED:` retenues et ignorées du rapport. Un état `waiting` SHALL y être libellé « en attente de l'interface ».

#### Scenario: Vignettes affichées
- **WHEN** l'utilisateur ouvre un change dont le rapport UI liste deux captures
- **THEN** deux vignettes sont affichées sous le rapport

#### Scenario: Visionneuse
- **WHEN** l'utilisateur clique sur une vignette
- **THEN** la capture s'affiche en grand et peut être fermée

#### Scenario: Aucune preuve
- **WHEN** le rapport ne liste aucune capture
- **THEN** aucune zone de vignettes n'est affichée

#### Scenario: Tâches vérifiées
- **WHEN** le rapport indique une tâche cochée et une ligne `TASK-VERIFIED:` ignorée
- **THEN** le bandeau distingue la tâche cochée de la ligne ignorée

#### Scenario: État d'attente
- **WHEN** la vérification du change attend le verrou UI
- **THEN** le bandeau affiche « en attente de l'interface »
