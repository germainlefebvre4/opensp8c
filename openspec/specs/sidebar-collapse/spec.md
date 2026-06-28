# Spec: sidebar-collapse

## Purpose

Gestion du comportement rétractable de la sidebar : toggle d'ouverture/fermeture via un bouton dédié, positionnement du bouton toggle, et suppression du bouton fullscreen de la navigation principale.

## Requirements

### Requirement: Sidebar rétractable via bouton toggle

La sidebar SHALL pouvoir être réduite à une bande étroite (`w-8`) en cliquant sur un bouton toggle situé dans son header. Ce bouton SHALL être visible dans les deux états (ouvert et fermé).

#### Scenario: Fermeture de la sidebar

- **WHEN** l'utilisateur clique sur le bouton `◀` dans le header de la sidebar
- **THEN** la sidebar se rétracte à `w-8` avec une transition animée
- **THEN** le contenu de la sidebar (liste projets, bouton ajout) devient invisible
- **THEN** le bouton toggle affiche l'icône `▶`

#### Scenario: Ouverture de la sidebar depuis l'état collapsed

- **WHEN** la sidebar est en état collapsed (`w-8`) et l'utilisateur clique sur le bouton `▶`
- **THEN** la sidebar s'élargit à `w-56` avec une transition animée
- **THEN** le contenu de la sidebar redevient visible
- **THEN** le bouton toggle affiche l'icône `◀`

### Requirement: Bouton toggle positionné en haut de la sidebar

Le bouton toggle SHALL être positionné dans le header de la sidebar, aligné avec le label "Projets", dans les deux états (ouvert et fermé).

#### Scenario: Position du bouton en état ouvert

- **WHEN** la sidebar est ouverte
- **THEN** le bouton `◀` est visible à droite du label "Projets" dans le header

#### Scenario: Position du bouton en état collapsed

- **WHEN** la sidebar est en état collapsed
- **THEN** le bouton `▶` est visible en haut de la bande `w-8`, à la même hauteur que le header

### Requirement: Suppression du bouton fullscreen de la navigation

Le bouton `Maximize2`/`Minimize2` dans la barre de navigation principale SHALL être supprimé.

#### Scenario: Absence du bouton fullscreen

- **WHEN** l'utilisateur consulte l'interface
- **THEN** aucun bouton `Maximize2` ou `Minimize2` n'est présent dans la barre de navigation principale
