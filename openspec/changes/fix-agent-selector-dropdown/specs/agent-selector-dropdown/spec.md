## ADDED Requirements

### Requirement: Dropdown visible malgré overflow-hidden des ancêtres
Le dropdown de sélection d'agent SHALL être entièrement visible même lorsque ses ancêtres portent `overflow: hidden`. Il SHALL utiliser `position: fixed` pour s'affranchir du clipping CSS.

#### Scenario: Ouverture dans le sidebar
- **WHEN** l'utilisateur clique sur le bouton "Agent…" dans le sidebar
- **THEN** le dropdown complet (avec tous les agents) est visible à l'écran, sans être tronqué

#### Scenario: Alignement sous le bouton
- **WHEN** le dropdown s'ouvre
- **THEN** son bord supérieur est positionné directement sous le bouton (4px de marge), aligné à gauche avec lui, de même largeur

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
