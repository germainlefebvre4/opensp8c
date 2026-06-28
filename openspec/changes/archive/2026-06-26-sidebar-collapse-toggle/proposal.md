## Why

Le bouton "plein écran" dans la nav principale (icône `Maximize2`) masque la sidebar mais son emplacement et son iconographie sont trompeurs. Déplacer cette action dans la sidebar elle-même, sous forme de toggle rétractable, est plus naturel et cohérent avec les conventions des outils dev modernes.

## What Changes

- Suppression du bouton `Maximize2`/`Minimize2` dans la barre de navigation principale
- Ajout d'un bouton `◀/▶` dans le header de la `WorkspaceSidebar` (à côté du label "Projets")
- La sidebar ne disparaît plus brutalement : elle se **rétracte** à `w-8` en laissant visible uniquement le bouton de ré-ouverture
- Transition animée (`transition-all duration-200`) entre l'état ouvert (`w-56`) et fermé (`w-8`)
- Le contenu de la sidebar (`opacity-0`) disparaît lors de la fermeture
- Le bouton `▶` en état fermé est positionné en haut, aligné avec le header

## Capabilities

### New Capabilities

- `sidebar-collapse`: Toggle de la sidebar avec rétraction animée — bouton dans le header, état collapsed réduit à une bande `w-8`

### Modified Capabilities

_(aucune — les specs existantes ne couvrent pas ce composant)_

## Impact

- `frontend/src/components/Layout.tsx` : suppression du bouton fullscreen, la gestion du state reste dans Layout et est passée en prop
- `frontend/src/components/WorkspaceSidebar.tsx` : ajout du bouton toggle, gestion de l'affichage collapsed
- Aucune dépendance backend, aucune API touchée
