## ADDED Requirements

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

## MODIFIED Requirements

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
