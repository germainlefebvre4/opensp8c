# Spec Delta

## ADDED Requirements

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
