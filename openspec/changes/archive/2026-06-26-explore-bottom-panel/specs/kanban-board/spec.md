## MODIFIED Requirements

### Requirement: Ouvrir l'ExplorePanel au clic sur une carte To Explore
L'utilisateur SHALL pouvoir cliquer sur une carte dans la colonne **To Explore** pour ouvrir le bottom panel de conversation. Le panel SHALL s'afficher sous les colonnes Kanban (layout flex-col), sans masquer ni comprimer les colonnes. La carte entière est la zone cliquable.

#### Scenario: Clic sur carte en colonne To Explore
- **WHEN** l'utilisateur clique sur une carte dans la colonne **To Explore**
- **THEN** le bottom panel de chat s'ouvre sous les colonnes Kanban, les colonnes restant visibles et interactibles au-dessus

#### Scenario: Bottom panel ne masque pas les colonnes
- **WHEN** le bottom panel est ouvert
- **THEN** les colonnes Kanban restent visibles et interactibles dans la partie supérieure de l'écran

### Requirement: Application pleine largeur avec colonnes auto-adaptées
Le Kanban Board SHALL occuper toute la largeur disponible de la zone de contenu, que le DetailPanel soit ouvert ou non. Lorsque le DetailPanel est ouvert, les colonnes SHALL partager l'espace horizontal avec lui selon un layout flex : colonnes en `flex: 1` et DetailPanel en largeur fixe (`420px`). Le bottom panel d'exploration n'affecte pas la largeur des colonnes. Les colonnes SHALL être scrollables horizontalement si leur largeur minimale combinée dépasse l'espace disponible.

#### Scenario: Redimensionnement de la fenêtre sans panel
- **WHEN** l'utilisateur redimensionne la fenêtre du navigateur et aucun panel n'est ouvert
- **THEN** les colonnes s'adaptent automatiquement pour remplir toute la largeur disponible sans débordement horizontal

#### Scenario: DetailPanel ouvert — colonnes réduites
- **WHEN** le DetailPanel est ouvert
- **THEN** les colonnes occupent l'espace restant après le slot de 420px du DetailPanel, avec un scroll horizontal si nécessaire

#### Scenario: DetailPanel fermé — colonnes pleine largeur
- **WHEN** le DetailPanel est fermé
- **THEN** les colonnes reprennent toute la largeur disponible

#### Scenario: Bottom panel ouvert — largeur colonnes inchangée
- **WHEN** le bottom panel d'exploration est ouvert
- **THEN** les colonnes conservent leur largeur (le bottom panel n'affecte que la hauteur disponible)
