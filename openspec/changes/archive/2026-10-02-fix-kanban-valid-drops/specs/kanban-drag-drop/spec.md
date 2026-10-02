# Spec Delta

## MODIFIED Requirements

### Requirement: Transitions de drag autorisées
Le Kanban SHALL autoriser uniquement les transitions suivantes par drag-and-drop :
- `to-explore` (change normal) → `ready` : déclenche FF directement
- `to-explore` (ghost card nommé) → `ready` : déclenche le flux promote (dialog + FF)
- `ready → todo` : promotion (marque le change "lancé", éligible au pool ; ne modifie pas `tasks.md`)
- `todo → ready` : rétrogradation (démarque le change "lancé" ; refusée si un worker est actif sur ce change)
- `ready → to-explore`, `todo → to-explore` ou `in-progress → to-explore` : reset tasks (confirmation requise)

Le drop vers une colonne cible autorisée SHALL être accepté dès que le pointeur se trouve à l'intérieur des bornes de la colonne cible, que cette colonne soit vide ou qu'elle contienne déjà d'autres cartes, et que le curseur soit relâché sur une carte existante ou sur l'espace vide de la colonne.
Toute autre combinaison source/cible SHALL être rejetée visuellement (drop non accepté). Les colonnes **In Progress**, **To Review** et **Done** SHALL n'être jamais une cible de drop : ces états sont pilotés par le pool d'agents et l'avancement des tâches, pas par le drag-and-drop. Les cartes des colonnes **Done** et **Archived** SHALL être non-draggables.

#### Scenario: Drag valide to-explore (change normal) vers todo
- **WHEN** l'utilisateur dépose un change normal (non-ghost) de la colonne "to-explore" sur "ready", que "ready" soit vide ou contienne des cartes
- **THEN** le drop est accepté et FF est déclenché directement sans dialog

#### Scenario: Drag valide to-explore (ghost card) vers todo
- **WHEN** l'utilisateur dépose un ghost card nommé de la colonne "to-explore" sur "ready", que "ready" soit vide ou contienne des cartes
- **THEN** le drop est accepté et la dialog de confirmation s'affiche avant toute action

#### Scenario: Drag valide ready vers todo (promotion)
- **WHEN** l'utilisateur dépose une carte de la colonne "ready" sur "todo", que "todo" soit vide ou contienne des cartes
- **THEN** le drop est accepté, aucune confirmation n'est demandée, et le change est marqué "lancé"

#### Scenario: Drag valide todo vers ready (rétrogradation)
- **WHEN** l'utilisateur dépose une carte de la colonne "todo" sur "ready" et qu'aucun worker n'est actif sur ce change
- **THEN** le drop est accepté, aucune confirmation n'est demandée, et le change est démarqué "lancé"

#### Scenario: Drag rejeté todo vers ready avec worker actif
- **WHEN** l'utilisateur dépose une carte de la colonne "todo" sur "ready" alors qu'un worker du pool d'agents est actif sur ce change
- **THEN** le drop est visuellement accepté mais l'appel API échoue (409), la carte retourne à "todo", et un message d'erreur est affiché

#### Scenario: Drag valide todo/in-progress vers to-explore
- **WHEN** l'utilisateur dépose une carte de "todo" ou "in-progress" sur "to-explore", sur une carte ou l'espace vide
- **THEN** le drop est accepté et une confirmation de reset tasks est demandée

#### Scenario: Drag valide ready vers to-explore
- **WHEN** l'utilisateur dépose une carte de la colonne "ready" sur "to-explore", sur une carte ou l'espace vide
- **THEN** le drop est accepté et une confirmation de reset tasks est demandée

#### Scenario: Drag invalide (done ou archived comme source)
- **WHEN** l'utilisateur tente de drag une carte des colonnes "Done" ou "Archived"
- **THEN** la carte ne peut pas être saisie (drag désactivé)

#### Scenario: Drag invalide (cible non autorisée)
- **WHEN** l'utilisateur drag une carte vers une colonne non autorisée pour cette source
- **THEN** la colonne cible refuse visuellement le drop et la carte retourne à sa position d'origine

#### Scenario: Drag vers In Progress, To Review ou Done refusé
- **WHEN** l'utilisateur drag une carte de "todo" ou "in-progress" vers "in-progress", "to-review" ou "done", ou survole ces colonnes pendant le drag
- **THEN** ces colonnes n'affichent aucun indicateur de dépôt, le drop n'est pas accepté, aucune action n'est déclenchée et la carte retourne à sa position d'origine
