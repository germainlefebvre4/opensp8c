## Purpose

Spec du drag-and-drop sur le Kanban Board : transitions autorisées, indicateurs visuels, blocage pendant ff, et confirmation avant reset.

## Requirements

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
- `to-review → done` : approbation du change en revue (confirmation requise, voir `change-review-actions`)
- `to-review → in-progress` : demande de correction (saisie obligatoire du retour de l'utilisateur, voir `change-review-actions`)

Le drop vers une colonne cible autorisée SHALL être accepté dès que le pointeur se trouve à l'intérieur des bornes de la colonne cible, que cette colonne soit vide ou qu'elle contienne déjà d'autres cartes, et que le curseur soit relâché sur une carte existante ou sur l'espace vide de la colonne.
Toute autre combinaison source/cible SHALL être rejetée visuellement (drop non accepté). La colonne **To Review** SHALL n'être jamais une cible de drop : un changement n'y entre que par la fin de son worker en mode `hitl-review`. Les colonnes **In Progress** et **Done** SHALL n'être une cible de drop que pour une carte provenant de **To Review** (demande de correction, approbation) : leurs autres états sont pilotés par le pool d'agents et l'avancement des tâches, pas par le drag-and-drop. Les cartes des colonnes **Done** et **Archived** SHALL être non-draggables.

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

#### Scenario: Drag to-review vers done (approbation)
- **WHEN** l'utilisateur dépose une carte de la colonne "to-review" sur "done", sur une carte ou l'espace vide
- **THEN** le drop est accepté et une confirmation d'approbation est demandée avant toute fusion

#### Scenario: Drag to-review vers in-progress (demande de correction)
- **WHEN** l'utilisateur dépose une carte de la colonne "to-review" sur "in-progress", sur une carte ou l'espace vide
- **THEN** le drop est accepté et le dialogue de saisie du retour de correction s'ouvre ; la carte reste en "to-review" tant que la demande n'est pas confirmée

#### Scenario: Drag invalide to-review vers une autre colonne
- **WHEN** l'utilisateur dépose une carte de la colonne "to-review" sur une colonne autre que "done" ou "in-progress"
- **THEN** la colonne cible refuse visuellement le drop et la carte retourne à sa position d'origine

### Requirement: Indicateur visuel de drag en cours
Dès le début d'un drag (saisie de la carte), le Kanban SHALL afficher immédiatement un indicateur visuel léger de zone de dépôt (bordure ou fond) sur chacune des colonnes cibles autorisées pour la colonne source de la carte draguée, que le curseur survole ces colonnes ou non. Parmi les colonnes autorisées, celle actuellement survolée par le curseur (sur son espace vide ou sur une de ses cartes) SHALL afficher un indicateur visuel renforcé par rapport aux autres colonnes autorisées non survolées. Les colonnes non autorisées pour cette source ne SHALL afficher aucun indicateur de dépôt, y compris lorsqu'elles sont survolées par le curseur.

#### Scenario: Début du drag — toutes les colonnes autorisées s'allument
- **WHEN** l'utilisateur saisit une carte et commence à la déplacer
- **THEN** toutes les colonnes cibles autorisées pour la colonne source de cette carte affichent immédiatement un indicateur visuel léger, avant même que le curseur ne les survole

#### Scenario: Survol d'une colonne acceptante
- **WHEN** l'utilisateur drag une carte au-dessus d'une colonne qui accepte ce drop (sur son espace vide ou sur une de ses cartes)
- **THEN** cette colonne affiche un indicateur visuel renforcé par rapport aux autres colonnes autorisées, qui conservent leur indicateur léger

#### Scenario: Survol d'une colonne rejetante
- **WHEN** l'utilisateur drag une carte au-dessus d'une colonne qui n'accepte pas ce drop
- **THEN** aucun indicateur, léger ou renforcé, n'est affiché sur cette colonne, même au survol

#### Scenario: Fin du drag
- **WHEN** l'utilisateur relâche la carte, que le drop soit accepté ou refusé
- **THEN** tous les indicateurs de zone de dépôt disparaissent de toutes les colonnes


### Requirement: Blocage du drag pendant ff en cours
Si un subprocess ff est actif pour un changement (spinner visible sur la carte), la carte SHALL être non-draggable jusqu'à réception de l'événement `ff_done` ou `ff_failed`.

#### Scenario: Tentative de drag pendant ff actif
- **WHEN** la carte d'un changement affiche le spinner ff et l'utilisateur tente de la drag
- **THEN** le drag est désactivé sur cette carte et aucune action n'est déclenchée

### Requirement: Confirmation avant reset de tasks
Un dialog de confirmation SHALL être affiché avant tout drop sur **To Explore**. Le message SHALL être adapté à l'état des tâches du changement et à la présence d'une branche : lorsque le changement porte une branche `feature/<change>` (`has_branch`), quelle que soit sa colonne d'origine et même si aucune tâche n'est cochée, le dialog SHALL avertir que le travail réalisé dans la branche, son worktree et son état de revue seront supprimés et ne pourront pas être récupérés.

#### Scenario: Reset depuis ready (aucune tâche faite, non lancé)
- **WHEN** l'utilisateur confirme le drop d'une carte **Ready** vers **To Explore**
- **THEN** le dialog indique "Réinitialiser les tâches ?" avec un message neutre

#### Scenario: Reset depuis todo (aucune tâche faite)
- **WHEN** l'utilisateur confirme le drop d'une carte **To Do** vers **To Explore**
- **THEN** le dialog indique "Réinitialiser les tâches ?" avec un message neutre

#### Scenario: Reset depuis in-progress (tâches partiellement faites)
- **WHEN** l'utilisateur confirme le drop d'une carte **In Progress** vers **To Explore**
- **THEN** le dialog indique "X tâches complétées seront perdues. Continuer ?" avec un message d'avertissement

#### Scenario: Reset d'une carte qui porte une branche
- **WHEN** l'utilisateur dépose vers **To Explore** une carte **Ready** ou **To Do** dont `has_branch` vaut `true`
- **THEN** le dialog affiche le message d'avertissement indiquant que la branche et le travail qu'elle contient seront supprimés, avec le bouton de confirmation en style d'avertissement

#### Scenario: Refus du backend
- **WHEN** le backend répond `409` à la demande de reset (worker actif ou action de revue en cours)
- **THEN** une notification affiche le message d'erreur, la carte retourne à sa colonne d'origine et rien n'est modifié

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

### Requirement: Représentation visuelle de la carte pendant le drag
Pendant un drag actif, une représentation visuelle de la carte SHALL rester visible sous le curseur tant que celui-ci se trouve dans les bornes visibles de la zone regroupant les colonnes du Kanban (To Explore, To Do, In Progress, To Review, Done, Archived), quelle que soit la colonne survolée. Cette représentation SHALL ne pas être tronquée ou masquée par le défilement (`overflow`) d'une colonne. Le défilement d'une colonne ne SHALL pas être déclenché par le déplacement de cette représentation.

#### Scenario: Le curseur traverse plusieurs colonnes pendant le drag
- **WHEN** l'utilisateur drag une carte depuis sa colonne source et déplace le curseur au-dessus d'une autre colonne du Kanban
- **THEN** la représentation visuelle de la carte reste visible sous le curseur, sans être coupée par les bords de la colonne survolée ni par ceux de la colonne source

#### Scenario: Le curseur sort de la zone visible d'une colonne à défilement
- **WHEN** l'utilisateur drag une carte au-dessus d'une colonne dont le contenu dépasse la hauteur visible (colonne scrollable)
- **THEN** aucune scrollbar n'apparaît dans cette colonne du fait du drag, et la représentation de la carte reste visible

#### Scenario: Le curseur atteint la limite de la zone des colonnes
- **WHEN** l'utilisateur déplace le curseur au-delà de la zone visible regroupant les colonnes du Kanban (par exemple au-dessus de l'en-tête de la page ou du panneau de détail)
- **THEN** la représentation visuelle de la carte reste positionnée au bord de cette zone plutôt que de suivre le curseur au-delà

#### Scenario: Fin du drag
- **WHEN** l'utilisateur relâche la carte, que le drop soit accepté ou refusé
- **THEN** la représentation visuelle de la carte disparaît et seule la carte dans sa colonne finale (source ou cible) reste affichée
