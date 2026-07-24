## 1. Mises à jour des types et propriétés

- [x] 1.1 Mettre à jour les interfaces de propriétés dans `ExploreBottomPanel.tsx` et `ExploreAnonymousBottomPanel.tsx` pour accepter `height: number | string`, `isMaximized?: boolean` et `onMaximizeToggle?: () => void`.
- [x] 1.2 Mettre à jour les interfaces de `ExplorePanel.tsx` et `ExploreAnonymousPanel.tsx` pour recevoir `isMaximized?: boolean` et `onMaximizeToggle?: () => void`.

## 2. Intégration du bouton de maximisation dans les Headers

- [x] 2.1 Importer les icônes `Maximize2` et `Minimize2` de `lucide-react` dans `ExplorePanel.tsx` et `ExploreAnonymousPanel.tsx`.
- [x] 2.2 Ajouter le bouton de bascule de maximisation dans le header de `ExplorePanel.tsx`, juste avant le bouton de fermeture (X).
- [x] 2.3 Ajouter le bouton de bascule de maximisation dans le header de `ExploreAnonymousPanel.tsx`, juste avant le bouton de fermeture (X).

## 3. Comportement et limites de redimensionnement des Bottom Panels

- [x] 3.1 Porter `MAX_HEIGHT_RATIO` de `0.7` à `0.9` dans `ExploreBottomPanel.tsx` et `ExploreAnonymousBottomPanel.tsx`.
- [x] 3.2 Dans `ExploreBottomPanel.tsx`, désactiver l'événement de redimensionnement (`handleMouseDown`) si `isMaximized` est vrai, et changer le style/curseur du drag handle.
- [x] 3.3 Dans `ExploreAnonymousBottomPanel.tsx`, désactiver l'événement de redimensionnement (`handleMouseDown`) si `isMaximized` est vrai, et changer le style/curseur du drag handle.

## 4. Intégration de l'état et de la mise en page dans la Page Kanban

- [x] 4.1 Ajouter l'état `panelMaximized` dans `KanbanPage.tsx` (`const [panelMaximized, setPanelMaximized] = useState(false)`).
- [x] 4.2 Masquer conditionnellement la barre de recherche et la section des colonnes Kanban/DetailPanel si `panelMaximized` est vrai.
- [x] 4.3 Passer la hauteur `panelMaximized ? '100%' : panelHeight`, ainsi que `isMaximized` et `onMaximizeToggle` à `ExploreAnonymousBottomPanel` et `ExploreBottomPanel` dans `KanbanPage.tsx`.

## 5. Validation et Tests

- [x] 5.1 Vérifier le bon redimensionnement manuel jusqu'à 90% de la hauteur de l'écran.
- [x] 5.2 Vérifier le bouton de maximisation/minimisation dans l'exploration nommée et anonyme (ghost card).
- [x] 5.3 S'assurer que le drag handle est inactif en mode maximisé.
