## Purpose

Spec for the agent selector dropdown component. Covers visibility, positioning, lifecycle behavior, and backend API reliability.

## Requirements

### Requirement: Dropdown visible malgré overflow-hidden des ancêtres
Le dropdown de sélection d'agent SHALL être entièrement visible même lorsque ses ancêtres portent `overflow: hidden`. Il SHALL utiliser `position: fixed` pour s'affranchir du clipping CSS. Le sélecteur étant placé dans la barre principale, le dropdown SHALL avoir une largeur minimale suffisante pour afficher le nom et la version de chaque agent sans troncature, et SHALL être aligné sur le bord droit de son bouton sans déborder de la fenêtre.

#### Scenario: Ouverture dans le sidebar
- **WHEN** l'utilisateur clique sur le bouton d'agent dans la barre principale
- **THEN** le dropdown complet (avec tous les agents) est visible à l'écran, sans être tronqué

#### Scenario: Alignement sous le bouton
- **WHEN** le dropdown s'ouvre
- **THEN** son bord supérieur est positionné directement sous le bouton (4px de marge) et son bord droit est aligné sur celui du bouton

#### Scenario: Bouton plus étroit que la liste
- **WHEN** le bouton d'agent est plus étroit que la largeur minimale du dropdown
- **THEN** le dropdown s'affiche à sa largeur minimale, sans troncature des libellés ni débordement hors de la fenêtre

### Requirement: Fermeture du dropdown lors du resize
Le dropdown SHALL se fermer automatiquement si la fenêtre est redimensionnée pendant qu'il est ouvert.

#### Scenario: Resize avec dropdown ouvert
- **WHEN** le dropdown est ouvert et l'utilisateur redimensionne la fenêtre
- **THEN** le dropdown se ferme

### Requirement: Endpoint /api/agents répond toujours
L'endpoint `/api/agents` SHALL toujours répondre en moins de 5 secondes, même si une CLI détectée ne répond pas.

#### Scenario: CLI qui bloque (ex: gh copilot --version)
- **WHEN** une commande de détection de version prend plus de 3 secondes
- **THEN** l'agent est retourné avec `installed: false` et l'endpoint répond normalement

#### Scenario: Toutes les CLIs absentes
- **WHEN** aucune CLI d'agent n'est installée
- **THEN** l'endpoint retourne un tableau de 5 agents tous avec `installed: false`
