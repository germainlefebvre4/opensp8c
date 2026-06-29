## ADDED Requirements

### Requirement: Transitions de drag autorisées
Le Kanban SHALL autoriser uniquement deux transitions par drag-and-drop :
- `to-explore → todo` : dépose sur la colonne **To Do** (déclenche ff)
- `todo → to-explore` ou `in-progress → to-explore` : dépose sur la colonne **To Explore** (reset tasks)

Toute autre combinaison source/cible SHALL être rejetée visuellement (drop non accepté). Les cartes des colonnes **Done** et **Archived** SHALL être non-draggables.

#### Scenario: Drag valide to-explore vers todo
- **WHEN** l'utilisateur dépose une carte de la colonne **To Explore** sur la colonne **To Do**
- **THEN** le drop est accepté et le flow ff est déclenché

#### Scenario: Drag valide todo/in-progress vers to-explore
- **WHEN** l'utilisateur dépose une carte de la colonne **To Do** ou **In Progress** sur la colonne **To Explore**
- **THEN** le drop est accepté et une confirmation est demandée avant reset

#### Scenario: Drag invalide (done ou archived comme source)
- **WHEN** l'utilisateur tente de drag une carte des colonnes **Done** ou **Archived**
- **THEN** la carte ne peut pas être saisie (drag désactivé)

#### Scenario: Drag invalide (cible non autorisée)
- **WHEN** l'utilisateur drag une carte vers une colonne non autorisée pour cette source (ex: to-explore → in-progress)
- **THEN** la colonne cible refuse visuellement le drop (indicateur de rejet) et la carte retourne à sa position d'origine

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
