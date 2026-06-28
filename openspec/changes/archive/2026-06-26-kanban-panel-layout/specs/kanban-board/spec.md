## MODIFIED Requirements

### Requirement: Application pleine largeur avec colonnes auto-adaptées
Le Kanban Board SHALL occuper toute la largeur disponible de la zone de contenu lorsqu'aucun panel latéral n'est ouvert. Lorsqu'un panel est ouvert, les colonnes SHALL partager l'espace horizontal avec le panel selon un layout flex : colonnes en `flex: 1` et panel en largeur fixe (`420px`). Les colonnes SHALL être scrollables horizontalement si leur largeur minimale combinée dépasse l'espace disponible.

#### Scenario: Redimensionnement de la fenêtre sans panel
- **WHEN** l'utilisateur redimensionne la fenêtre du navigateur et aucun panel n'est ouvert
- **THEN** les colonnes s'adaptent automatiquement pour remplir toute la largeur disponible sans débordement horizontal

#### Scenario: Panel ouvert — colonnes réduites
- **WHEN** un DetailPanel ou ExplorePanel est ouvert
- **THEN** les colonnes occupent l'espace restant après le slot de 420px du panel, avec un scroll horizontal si nécessaire

#### Scenario: Panel fermé — colonnes pleine largeur
- **WHEN** le panel est fermé
- **THEN** les colonnes reprennent toute la largeur disponible
