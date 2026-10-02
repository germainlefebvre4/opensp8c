# benchmark-metrics-report Specification

## Purpose

Définit comment le benchmark extrait les chronos par phase à partir des données existantes de la plateforme derrière une source de métriques interchangeable, conserve les données brutes avant leur purge, et produit un rapport Markdown comparant les deux méthodes.

## Requirements

### Requirement: Source de métriques interchangeable
Le calcul des chronos et la génération du rapport SHALL dépendre d'une interface de source de métriques et non des fichiers de la plateforme. La source par défaut SHALL combiner le journal de l'observateur (événements et transitions de colonne) et les données existantes de la plateforme (entrées d'activité, runs de conversation et historique git). Remplacer cette source par une autre implémentation (par exemple un module de métriques du backend) SHALL NOT exiger de modifier le calcul des chronos, l'agrégation ni le rapport.

#### Scenario: Calcul indépendant de la source
- **WHEN** une source de métriques de test fournit les mêmes événements horodatés que la source par défaut
- **THEN** les chronos calculés et le rapport généré sont identiques

### Requirement: Observation des événements de la plateforme pendant un run
Pour un run de méthode A, le benchmark SHALL fournir un observateur qui, tant que le run est actif, s'abonne au flux d'événements du workspace de la plateforme et enregistre chaque événement reçu avec son horodatage de réception. À chaque événement de modification du change suivi, l'observateur SHALL en outre relever la colonne Kanban courante du change (to-explore, ready, todo, in-progress, done) via l'API de la plateforme et enregistrer chaque transition de colonne avec son horodatage. L'observateur SHALL NOT modifier l'état de la plateforme. Un run de méthode A pour lequel l'observateur n'a pas été actif de l'exploration au merge SHALL être marqué invalide.

#### Scenario: Événements enregistrés
- **WHEN** la plateforme émet les événements de promotion (`ff_started`, `ff_done`) pendant un run observé
- **THEN** chacun apparaît dans le journal d'événements du run avec son horodatage de réception

#### Scenario: Transition de colonne relevée
- **WHEN** le change passe de la colonne `ready` à la colonne `todo` après le lancement
- **THEN** le journal du run contient une transition `ready` → `todo` avec son horodatage

#### Scenario: Observateur interrompu
- **WHEN** l'observateur a perdu sa connexion au flux d'événements pendant le run et n'a pas pu la rétablir
- **THEN** le run est marqué invalide avec la raison « observation incomplète »

### Requirement: Chronos de la méthode A par phase
Pour un run de méthode A, le benchmark SHALL calculer, à partir du journal de l'observateur et des données persistées de la plateforme :
- le **temps humain** : la phase d'explore (du premier message de l'exploration à `ff_started`) plus l'attente de lancement (de `ff_done` à la transition du change vers la colonne `todo`) ;
- le **temps machine** : la durée du ff (de `ff_started` à `ff_done`), plus la durée de l'exécution du pool (de la transition vers `todo` au commit de merge dans la branche de départ), détaillée en durée active du pool (du marqueur `pool_run_start` au marqueur `pool_run_end` du run de pool) ;
- le **total** : du premier message d'explore au commit de merge, égal au temps humain plus le temps machine.

Un run de pool dont le marqueur de fin porte un résultat autre que `completed` SHALL rendre le run invalide.

#### Scenario: Chronos d'un run de plateforme
- **WHEN** un run de méthode A s'est déroulé de l'explore au merge
- **THEN** le résultat contient le temps humain, le temps machine, la décomposition (explore, attente de lancement, ff, pool) et le total, avec total égal à humain plus machine

#### Scenario: Pool en pause
- **WHEN** le marqueur de fin du run de pool indique `paused`
- **THEN** le run est marqué invalide avec la raison de pause dans son résultat

#### Scenario: Plusieurs runs de pool
- **WHEN** le change a connu plusieurs runs de pool (reprise après pause)
- **THEN** le temps de pool cumule l'ensemble des runs et le nombre de runs est indiqué dans le résultat

### Requirement: Chronos de la méthode B
Pour un run de méthode B, le benchmark SHALL mesurer le temps total entre le lancement de la première exécution de l'agent et la réussite de la validation, ainsi que le temps cumulé passé dans l'agent et dans la validation séparément.

#### Scenario: Chrono baseline avec relance
- **WHEN** un run de méthode B a nécessité une relance
- **THEN** le total inclut les deux exécutions de l'agent et les deux validations, et le résultat indique 1 relance

### Requirement: Conservation des données brutes avant rétention
Le benchmark SHALL conserver, dans le dossier de sortie de chaque run, une copie des données brutes dont il a extrait ses chronos (journal de l'observateur, entrées d'activité, runs de conversation et extrait de l'historique git), de sorte qu'aucun résultat ne dépende de données soumises à la rétention des logs de la plateforme.

#### Scenario: Copie des données de plateforme
- **WHEN** un run de méthode A est collecté
- **THEN** le journal de l'observateur ainsi que les fichiers d'activité et de conversation du change sont copiés dans le dossier du run avant tout calcul

### Requirement: Rapport Markdown
Le benchmark SHALL générer un rapport Markdown comportant : un tableau par run (méthode, validité, temps humain, temps machine, total, relances ou pauses), les agrégats par méthode calculés sur les seuls runs valides (médiane, minimum, maximum et nombre de runs valides sur nombre de runs), l'écart entre les médianes des deux méthodes, la configuration effective et le SHA de départ, et une section « Limites » mentionnant au minimum le biais de familiarité (l'agent connaît ce dépôt et la plateforme a été en partie construite avec elle-même), la taille d'échantillon et l'exclusion de la qualité, du coût et des tokens. Le rapport SHALL être regénérable à partir des seuls résultats bruts.

#### Scenario: Agrégats sur les runs valides
- **WHEN** 3 runs de méthode A dont un invalide sont agrégés
- **THEN** la médiane, le min et le max portent sur les 2 runs valides et le rapport indique « 2/3 valides »

#### Scenario: Section limites toujours présente
- **WHEN** un rapport est généré, quel que soit le nombre de runs
- **THEN** la section « Limites » mentionne le biais de familiarité

#### Scenario: Aucun run valide pour une méthode
- **WHEN** aucune exécution d'une méthode n'est valide
- **THEN** le rapport indique « aucune donnée valide » pour cette méthode au lieu d'une médiane et ne calcule aucun écart
