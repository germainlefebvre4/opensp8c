# workspace-attention-signals Specification

## Purpose

Indiquer, pour chaque projet de la sidebar, quels changes attendent une action de l'utilisateur et pourquoi (revue, worker en pause, validation humaine, vérification échouée), afin qu'il repère d'un coup d'œil où intervenir et puisse y accéder directement.

## Requirements

### Requirement: Signaux d'action exposés par workspace
L'API `GET /api/workspaces` SHALL retourner pour chaque workspace un champ `attention` : la liste des changes qui portent au moins un signal d'action, chacun avec la liste ordonnée de ses signaux. Un signal SHALL avoir un type parmi `review`, `paused`, `hitl`, `verify-failed` et, selon le type, une raison textuelle. Un change sans signal SHALL NE PAS figurer dans `attention`. Le champ SHALL être recalculé à chaque requête, dans le même passage que `task_counts`, et SHALL valoir une liste vide (jamais absent) pour un workspace sans signal.

#### Scenario: Workspace sans signal
- **WHEN** aucun change du workspace ne porte de signal d'action
- **THEN** `attention` est une liste vide

#### Scenario: Change avec un signal
- **WHEN** un change du workspace est au statut `to-review`
- **THEN** `attention` contient ce change avec un signal de type `review`

#### Scenario: Change absent des signaux
- **WHEN** un change est `todo`, `in-progress` avec un worker actif, ou `done` sans autre condition de signal
- **THEN** il ne figure pas dans `attention`

### Requirement: Types de signaux d'action
Le système SHALL produire les signaux suivants :
- `review` : le change est au statut `to-review` (aucune raison).
- `paused` : un worker du pool détient le change et est en pause ; la raison SHALL être la raison de blocage du worker.
- `hitl` : une tâche non cochée du `tasks.md` du change porte le marqueur de validation humaine ; il SHALL y avoir un signal par tâche concernée et la raison SHALL être le texte de la tâche sans le marqueur. Le `tasks.md` pris en compte SHALL être celui de la branche `feature/<change>` lorsque le change en possède une, comme pour la progression affichée dans le Kanban.
- `verify-failed` : la vérification du change est à l'état `failed`.

Les vérifications `queued`, `running` et `waiting` SHALL NE PAS produire de signal. Un même change MAY porter plusieurs signaux simultanément.

#### Scenario: Worker en pause avec raison
- **WHEN** un worker du pool détient le change `fix-export` et est en pause avec la raison « tests en échec »
- **THEN** `fix-export` porte un signal `paused` dont la raison est « tests en échec »

#### Scenario: Tâches de validation humaine
- **WHEN** le `tasks.md` du change `add-search` contient deux tâches non cochées avec le marqueur de validation humaine et une troisième cochée
- **THEN** `add-search` porte deux signaux `hitl`, un par tâche non cochée, dont la raison est le texte de la tâche sans le marqueur

#### Scenario: Tâche humaine cochée
- **WHEN** toutes les tâches marquées d'un change sont cochées
- **THEN** il ne porte aucun signal `hitl`

#### Scenario: Vérification échouée
- **WHEN** la vérification d'un change est à l'état `failed`
- **THEN** le change porte un signal `verify-failed`

#### Scenario: Vérification en cours ou en file
- **WHEN** la vérification d'un change est `queued`, `running` ou `waiting`
- **THEN** le change ne porte aucun signal `verify-failed`

#### Scenario: Cumul de signaux
- **WHEN** un change est en pause et porte aussi une tâche de validation humaine non cochée
- **THEN** il figure une seule fois dans `attention`, avec un signal `paused` et un signal `hitl`

### Requirement: Compteur des changes en attente d'action
La ligne 1 de chaque projet de la sidebar SHALL afficher, à droite du nom, le nombre de changes distincts présents dans `attention`, sans texte superflu (un nombre seul). Ce compteur SHALL être masqué lorsque sa valeur est 0. Un change portant plusieurs signaux SHALL compter pour 1.

#### Scenario: Compteur affiché
- **WHEN** un projet a trois changes portant des signaux, dont un en porte cinq
- **THEN** la ligne 1 affiche le nombre 3

#### Scenario: Aucun signal
- **WHEN** aucun change du projet ne porte de signal
- **THEN** aucun compteur n'est affiché sur la ligne 1

#### Scenario: Mise à jour
- **WHEN** un signal apparaît ou disparaît dans un workspace
- **THEN** le compteur de la sidebar se met à jour dans les 15 secondes suivantes

### Requirement: Vue dépliée des signaux d'un projet
Chaque projet de la sidebar SHALL pouvoir être déplié par un contrôle (chevron) situé sur sa ligne 1 ; ce contrôle SHALL être affiché pour tout projet porteur d'au moins un signal. La vue dépliée SHALL lister chaque change concerné une seule fois, avec son nom, suivi d'une ligne par signal du change (libellé du type de signal et, le cas échéant, sa raison). Les changes SHALL être triés du plus bloquant au moins bloquant selon leur signal le plus bloquant (`paused`, puis `verify-failed`, puis `hitl`, puis `review`), puis par nom ; les signaux d'un change SHALL suivre le même ordre. Lorsqu'un change porte plus de trois signaux `hitl`, seuls les trois premiers SHALL être listés, suivis d'une ligne « +N autres ». Une raison trop longue pour la largeur de la sidebar SHALL être tronquée et son texte complet disponible en infobulle.

#### Scenario: Dépliage
- **WHEN** l'utilisateur déplie un projet porteur de signaux
- **THEN** la vue dépliée liste ses changes concernés, chacun une seule fois, avec une ligne par signal

#### Scenario: Change à signaux multiples
- **WHEN** un change porte un signal `paused` et deux signaux `hitl`
- **THEN** son nom apparaît une fois, suivi de trois lignes : la pause avec sa raison puis les deux validations humaines

#### Scenario: Ordre des changes
- **WHEN** un projet a un change `review` et un change `paused`
- **THEN** le change `paused` est listé avant le change `review`

#### Scenario: Au-delà de trois validations humaines
- **WHEN** un change porte cinq signaux `hitl`
- **THEN** trois sont listés, suivis d'une ligne « +2 autres »

#### Scenario: Raison longue
- **WHEN** la raison d'un signal dépasse la largeur disponible
- **THEN** elle est tronquée avec une ellipse et son texte complet apparaît en infobulle

#### Scenario: Projet sans signal
- **WHEN** un projet ne porte aucun signal
- **THEN** aucun contrôle de dépliage n'est affiché pour ce projet

### Requirement: État de dépliage conservé pour la session
Les projets SHALL être repliés par défaut. L'état déplié de chaque projet SHALL être indépendant des autres, conservé en mémoire pendant la session de l'application et SHALL NE PAS être persisté au-delà (rechargement de la page). Un projet déplié dont les signaux disparaissent SHALL se présenter replié, sans contrôle de dépliage.

#### Scenario: État par défaut
- **WHEN** l'application est chargée
- **THEN** tous les projets sont repliés

#### Scenario: Indépendance des projets
- **WHEN** l'utilisateur déplie le projet A
- **THEN** le projet B reste dans son état précédent

#### Scenario: Non persisté
- **WHEN** l'utilisateur déplie un projet puis recharge la page
- **THEN** le projet est de nouveau replié

### Requirement: Ouverture d'un change depuis la sidebar
Un clic sur un change de la vue dépliée SHALL sélectionner le workspace du projet, afficher son Kanban et ouvrir le DetailPanel de ce change. Si le change n'existe plus au moment de l'ouverture, le Kanban SHALL s'afficher sans DetailPanel, sans erreur.

#### Scenario: Ouverture d'un change
- **WHEN** l'utilisateur clique sur le change `feat-auth` d'un projet B alors que le projet A est actif, sur l'onglet Specs
- **THEN** le workspace B est actif, le Kanban est affiché et le DetailPanel de `feat-auth` est ouvert

#### Scenario: Change disparu
- **WHEN** le change a été archivé ou supprimé entre l'affichage et le clic
- **THEN** le Kanban du projet s'affiche sans DetailPanel et sans message d'erreur
