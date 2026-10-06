# Spec Delta

## ADDED Requirements

### Requirement: Drop vers Done refusé tant que des tâches restent à valider
Le drop d'une carte de **To Review** sur **Done** SHALL être refusé lorsque la carte indique des tâches non cochées (`tasks_done` inférieur à `tasks_total`) : aucune confirmation d'approbation ne s'affiche, aucun appel n'est fait au backend, la carte retourne dans To Review et une notification indique le nombre de tâches à valider. Si les compteurs étaient périmés et que le backend refuse l'approbation avec le code `tasks_pending`, la carte SHALL retourner dans To Review et le message du refus SHALL être affiché. Cette exigence précise la transition `to-review → done` de « Transitions de drag autorisées » : l'approbation par drag n'est possible que lorsque toutes les tâches sont cochées.

#### Scenario: Drop avec des tâches restantes
- **WHEN** l'utilisateur dépose sur Done une carte To Review affichant « 8 / 10 »
- **THEN** aucune confirmation n'est demandée, la carte reste dans To Review et une notification indique « 2 tâches à valider »

#### Scenario: Drop avec toutes les tâches cochées
- **WHEN** l'utilisateur dépose sur Done une carte To Review affichant « 10 / 10 »
- **THEN** la confirmation d'approbation s'affiche comme avant

#### Scenario: Compteurs périmés
- **WHEN** la confirmation est acceptée mais que le backend répond `409` avec le code `tasks_pending`
- **THEN** la carte reste dans To Review et le message du refus est affiché
