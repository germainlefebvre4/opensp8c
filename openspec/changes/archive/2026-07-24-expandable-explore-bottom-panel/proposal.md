## Why

Le panneau d'exploration (bottom panel) est actuellement limité à une hauteur maximale de 70% de l'écran et ne dispose pas d'un moyen rapide de basculer en plein écran ou de s'agrandir de manière significative pour lire de longs messages ou du code généré. Permettre d'agrandir et de maximiser ce panneau améliorera grandement la lisibilité de la conversation et le confort d'utilisation de l'agent d'exploration.

## What Changes

- Ajout d'un bouton de maximisation/minimisation dans le header du panneau d'exploration (`ExplorePanel` et `ExploreAnonymousPanel`).
- Permettre au panneau de s'étendre en mode plein écran / hauteur maximale (superposition absolue sous la barre de navigation).
- Augmentation de la limite de redimensionnement manuel par glisser-déposer (drag-to-resize) jusqu'à 90% de la hauteur de l'écran.
- Conservation de l'état maximisé ou de la hauteur redimensionnée lors des transitions de pages ou des réouvertures.

## Capabilities

### New Capabilities

<!-- No new capabilities, this is a modification of an existing feature. -->

### Modified Capabilities

- `explore-bottom-panel`: Ajout de la fonction de maximisation/minimisation dans le header et mise à jour des contraintes de hauteur (hauteur maximale de drag portée à 90% de l'écran).

## Impact

- **Frontend** :
  - `frontend/src/components/ExploreBottomPanel.tsx` et `frontend/src/components/ExploreAnonymousBottomPanel.tsx` : Modification de `MAX_HEIGHT_RATIO` de `0.7` à `0.9`. Ajout de la gestion de l'état maximisé (`isMaximized`).
  - `frontend/src/components/ExplorePanel.tsx` et `frontend/src/components/ExploreAnonymousPanel.tsx` : Ajout d'un bouton de maximisation/minimisation dans le header.
  - `frontend/src/pages/KanbanPage.tsx` : Gestion et transmission de l'état maximisé ou synchronisation de la hauteur.
