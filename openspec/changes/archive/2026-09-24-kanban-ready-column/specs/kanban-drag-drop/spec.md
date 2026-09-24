# Spec Delta

## MODIFIED Requirements

### Requirement: Ghost card draggable vers "todo" via flux promote
Les ghost cards (status "to-explore", `is_ghost: true`) nommés SHALL être draggables vers la colonne "ready". Ce drag déclenche le flux de promotion (dialog + FF), pas le flux FF direct des changes normaux.

#### Scenario: Drag ghost card nommé vers todo — dialog de confirmation
- **WHEN** l'utilisateur dépose un ghost card nommé sur la colonne "ready"
- **THEN** le drop est accepté visuellement et une dialog de confirmation s'affiche ("Créer un change ?") avant tout appel API

#### Scenario: Drag ghost card non-nommé bloqué
- **WHEN** l'utilisateur tente de drag un ghost card encore en phase de nommage (label "Exploring...")
- **THEN** le drag est désactivé sur cette carte (non-saisissable)

#### Scenario: Ghost card non draggable vers "to-explore" (retour arrière impossible)
- **WHEN** l'utilisateur tente de drag un ghost card vers une colonne autre que "ready"
- **THEN** le drop est refusé visuellement, la carte retourne à "to-explore"

### Requirement: Transitions de drag autorisées
Le Kanban SHALL autoriser uniquement les transitions suivantes par drag-and-drop :
- `to-explore` (change normal) → `ready` : déclenche FF directement
- `to-explore` (ghost card nommé) → `ready` : déclenche le flux promote (dialog + FF)
- `ready → todo` : promotion (marque le change "lancé", éligible au pool ; ne modifie pas `tasks.md`)
- `todo → ready` : rétrogradation (démarque le change "lancé" ; refusée si un worker est actif sur ce change)
- `ready → to-explore`, `todo → to-explore` ou `in-progress → to-explore` : reset tasks (confirmation requise)

Toute autre combinaison source/cible SHALL être rejetée visuellement (drop non accepté). Les cartes des colonnes **Done** et **Archived** SHALL être non-draggables.

#### Scenario: Drag valide to-explore (change normal) vers todo
- **WHEN** l'utilisateur dépose un change normal (non-ghost) de la colonne "to-explore" sur "ready"
- **THEN** le drop est accepté et FF est déclenché directement sans dialog

#### Scenario: Drag valide to-explore (ghost card) vers todo
- **WHEN** l'utilisateur dépose un ghost card nommé de la colonne "to-explore" sur "ready"
- **THEN** le drop est accepté et la dialog de confirmation s'affiche avant toute action

#### Scenario: Drag valide ready vers todo (promotion)
- **WHEN** l'utilisateur dépose une carte de la colonne "ready" sur "todo"
- **THEN** le drop est accepté, aucune confirmation n'est demandée, et le change est marqué "lancé"

#### Scenario: Drag valide todo vers ready (rétrogradation)
- **WHEN** l'utilisateur dépose une carte de la colonne "todo" sur "ready" et qu'aucun worker n'est actif sur ce change
- **THEN** le drop est accepté, aucune confirmation n'est demandée, et le change est démarqué "lancé"

#### Scenario: Drag rejeté todo vers ready avec worker actif
- **WHEN** l'utilisateur dépose une carte de la colonne "todo" sur "ready" alors qu'un worker du pool d'agents est actif sur ce change
- **THEN** le drop est visuellement accepté mais l'appel API échoue (409), la carte retourne à "todo", et un message d'erreur est affiché

#### Scenario: Drag valide todo/in-progress vers to-explore
- **WHEN** l'utilisateur dépose une carte de "todo" ou "in-progress" sur "to-explore"
- **THEN** le drop est accepté et une confirmation de reset tasks est demandée

#### Scenario: Drag valide ready vers to-explore
- **WHEN** l'utilisateur dépose une carte de la colonne "ready" sur "to-explore"
- **THEN** le drop est accepté et une confirmation de reset tasks est demandée

#### Scenario: Drag invalide (done ou archived comme source)
- **WHEN** l'utilisateur tente de drag une carte des colonnes "Done" ou "Archived"
- **THEN** la carte ne peut pas être saisie (drag désactivé)

#### Scenario: Drag invalide (cible non autorisée)
- **WHEN** l'utilisateur drag une carte vers une colonne non autorisée pour cette source
- **THEN** la colonne cible refuse visuellement le drop et la carte retourne à sa position d'origine

### Requirement: Confirmation avant reset de tasks
Un dialog de confirmation SHALL être affiché avant tout drop sur **To Explore**. Le message SHALL être adapté à l'état des tâches du changement.

#### Scenario: Reset depuis ready (aucune tâche faite, non lancé)
- **WHEN** l'utilisateur confirme le drop d'une carte **Ready** vers **To Explore**
- **THEN** le dialog indique "Réinitialiser les tâches ?" avec un message neutre

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
Si un ExplorePanel est ouvert pour un changement dont la carte est droppée vers **Ready**, le frontend SHALL fermer ce panel (DELETE /changes/{name}/explore) avant de déclencher le ff (POST /changes/{name}/ff).

#### Scenario: Drag avec ExplorePanel ouvert
- **WHEN** l'utilisateur drop une carte vers **Ready** et qu'un ExplorePanel est ouvert pour ce changement
- **THEN** l'ExplorePanel se ferme, puis le ff est déclenché séquentiellement

#### Scenario: Drag sans ExplorePanel ouvert
- **WHEN** l'utilisateur drop une carte vers **Ready** et qu'aucun ExplorePanel n'est ouvert
- **THEN** le ff est déclenché directement sans étape de fermeture
