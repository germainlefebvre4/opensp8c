## Purpose

Spec du drag-and-drop sur le Kanban Board : transitions autorisées, indicateurs visuels, blocage pendant ff, et confirmation avant reset.

## Requirements

### Requirement: Ghost card draggable vers "todo" via flux promote
Les ghost cards (status "to-explore", `is_ghost: true`) nommés SHALL être draggables vers la colonne "todo". Ce drag déclenche le flux de promotion (dialog + FF), pas le flux FF direct des changes normaux.

#### Scenario: Drag ghost card nommé vers todo — dialog de confirmation
- **WHEN** l'utilisateur dépose un ghost card nommé sur la colonne "todo"
- **THEN** le drop est accepté visuellement et une dialog de confirmation s'affiche ("Créer un change ?") avant tout appel API

#### Scenario: Drag ghost card non-nommé bloqué
- **WHEN** l'utilisateur tente de drag un ghost card encore en phase de nommage (label "Exploring...")
- **THEN** le drag est désactivé sur cette carte (non-saisissable)

#### Scenario: Ghost card non draggable vers "to-explore" (retour arrière impossible)
- **WHEN** l'utilisateur tente de drag un ghost card vers une colonne autre que "todo"
- **THEN** le drop est refusé visuellement, la carte retourne à "to-explore"

### Requirement: Transitions de drag autorisées
Le Kanban SHALL autoriser uniquement les transitions suivantes par drag-and-drop :
- `to-explore` (change normal) → `todo` : déclenche FF directement
- `to-explore` (ghost card nommé) → `todo` : déclenche le flux promote (dialog + FF)
- `todo → to-explore` ou `in-progress → to-explore` : reset tasks (confirmation requise)

Toute autre combinaison source/cible SHALL être rejetée visuellement (drop non accepté). Les cartes des colonnes **Done** et **Archived** SHALL être non-draggables.

#### Scenario: Drag valide to-explore (change normal) vers todo
- **WHEN** l'utilisateur dépose un change normal (non-ghost) de la colonne "to-explore" sur "todo"
- **THEN** le drop est accepté et FF est déclenché directement sans dialog

#### Scenario: Drag valide to-explore (ghost card) vers todo
- **WHEN** l'utilisateur dépose un ghost card nommé de la colonne "to-explore" sur "todo"
- **THEN** le drop est accepté et la dialog de confirmation s'affiche avant toute action

#### Scenario: Drag valide todo/in-progress vers to-explore
- **WHEN** l'utilisateur dépose une carte de "todo" ou "in-progress" sur "to-explore"
- **THEN** le drop est accepté et une confirmation de reset tasks est demandée

#### Scenario: Drag invalide (done ou archived comme source)
- **WHEN** l'utilisateur tente de drag une carte des colonnes "Done" ou "Archived"
- **THEN** la carte ne peut pas être saisie (drag désactivé)

#### Scenario: Drag invalide (cible non autorisée)
- **WHEN** l'utilisateur drag une carte vers une colonne non autorisée pour cette source
- **THEN** la colonne cible refuse visuellement le drop et la carte retourne à sa position d'origine

### Requirement: Indicateur visuel de drag en cours
Pendant un drag actif, la colonne cible autorisée SHALL afficher un indicateur visuel de zone de dépôt (highlight de bordure ou fond). Les colonnes non autorisées pour cette source ne SHALL pas afficher d'indicateur de dépôt.

#### Scenario: Survol d'une colonne acceptante
- **WHEN** l'utilisateur drag une carte au-dessus d'une colonne qui accepte ce drop
- **THEN** la colonne affiche un highlight visuel (bordure ou fond légèrement coloré)

#### Scenario: Survol d'une colonne rejetante
- **WHEN** l'utilisateur drag une carte au-dessus d'une colonne qui n'accepte pas ce drop
- **THEN** aucun highlight n'est affiché sur cette colonne

### Requirement: Blocage du drag pendant ff en cours
Si un subprocess ff est actif pour un changement (spinner visible sur la carte), la carte SHALL être non-draggable jusqu'à réception de l'événement `ff_done` ou `ff_failed`.

#### Scenario: Tentative de drag pendant ff actif
- **WHEN** la carte d'un changement affiche le spinner ff et l'utilisateur tente de la drag
- **THEN** le drag est désactivé sur cette carte et aucune action n'est déclenchée

### Requirement: Confirmation avant reset de tasks
Un dialog de confirmation SHALL être affiché avant tout drop sur **To Explore**. Le message SHALL être adapté à l'état des tâches du changement.

#### Scenario: Reset depuis todo (aucune tâche faite)
- **WHEN** l'utilisateur confirme le drop d'une carte **To Do** vers **To Explore**
- **THEN** le dialog indique "Réinitialiser les tâches ?" avec un message neutre

#### Scenario: Reset depuis in-progress (tâches partiellement faites)
- **WHEN** l'utilisateur confirme le drop d'une carte **In Progress** vers **To Explore**
- **THEN** le dialog indique "X tâches complétées seront perdues. Continuer ?" avec un message d'avertissement

#### Scenario: Annulation de la confirmation
- **WHEN** l'utilisateur clique "Annuler" dans le dialog de confirmation
- **THEN** la carte retourne à sa colonne d'origine sans modification

### Requirement: Fermeture de l'ExplorePanel avant déclenchement ff
Si un ExplorePanel est ouvert pour un changement dont la carte est droppée vers **To Do**, le frontend SHALL fermer ce panel (DELETE /changes/{name}/explore) avant de déclencher le ff (POST /changes/{name}/ff).

#### Scenario: Drag avec ExplorePanel ouvert
- **WHEN** l'utilisateur drop une carte vers **To Do** et qu'un ExplorePanel est ouvert pour ce changement
- **THEN** l'ExplorePanel se ferme, puis le ff est déclenché séquentiellement

#### Scenario: Drag sans ExplorePanel ouvert
- **WHEN** l'utilisateur drop une carte vers **To Do** et qu'aucun ExplorePanel n'est ouvert
- **THEN** le ff est déclenché directement sans étape de fermeture
