# Spec Delta

## MODIFIED Requirements

### Requirement: Sidebar rétractable via bouton toggle

La sidebar SHALL pouvoir être réduite à un rail étroit (`w-10`) en cliquant sur un bouton toggle situé dans son header. Ce bouton SHALL être visible dans les deux états (ouvert et fermé). À l'état ouvert, la sidebar SHALL avoir la largeur `w-72`. À l'état réduit, le rail SHALL afficher, pour chaque projet, une pastille portant les initiales du projet, marquée d'un point d'attention lorsque le projet a au moins un change en attente d'action, avec le nom du projet en infobulle ; un clic sur une pastille SHALL sélectionner le projet.

#### Scenario: Fermeture de la sidebar

- **WHEN** l'utilisateur clique sur le bouton `◀` dans le header de la sidebar
- **THEN** la sidebar se rétracte à `w-10` avec une transition animée
- **THEN** la liste détaillée des projets et le bouton d'ajout deviennent invisibles
- **THEN** le rail affiche une pastille d'initiales par projet
- **THEN** le bouton toggle affiche l'icône `▶`

#### Scenario: Ouverture de la sidebar depuis l'état collapsed

- **WHEN** la sidebar est en état collapsed (`w-10`) et l'utilisateur clique sur le bouton `▶`
- **THEN** la sidebar s'élargit à `w-72` avec une transition animée
- **THEN** le contenu de la sidebar redevient visible
- **THEN** le bouton toggle affiche l'icône `◀`

#### Scenario: Pastille d'initiales avec attention

- **WHEN** la sidebar est réduite et un projet a au moins un change en attente d'action
- **THEN** sa pastille d'initiales porte un point d'attention

#### Scenario: Sélection depuis le rail

- **WHEN** la sidebar est réduite et l'utilisateur clique sur la pastille d'un projet
- **THEN** ce projet devient le workspace actif

### Requirement: Bouton toggle positionné en haut de la sidebar

Le bouton toggle SHALL être positionné dans le header de la sidebar, aligné avec le label "Projets", dans les deux états (ouvert et fermé).

#### Scenario: Position du bouton en état ouvert

- **WHEN** la sidebar est ouverte
- **THEN** le bouton `◀` est visible à droite du label "Projets" dans le header

#### Scenario: Position du bouton en état collapsed

- **WHEN** la sidebar est en état collapsed
- **THEN** le bouton `▶` est visible en haut du rail `w-10`, à la même hauteur que le header
