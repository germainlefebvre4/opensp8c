## 1. Traduction (Locales)

- [x] 1.1 Ajouter les clés `createChange` et `createChangeTooltip` dans `frontend/src/locales/fr/explore.json`
- [x] 1.2 Ajouter les clés `createChange` et `createChangeTooltip` dans `frontend/src/locales/en/explore.json`

## 2. Déclaration des Props Callbacks

- [x] 2.1 Ajouter la prop optionnelle `onPromote?: () => void` dans l'interface `Props` de `ExploreAnonymousBottomPanel.tsx` et la passer au sous-composant `ExploreAnonymousPanel`
- [x] 2.2 Ajouter la prop optionnelle `onPromote?: () => void` dans l'interface `Props` de `ExploreAnonymousPanel.tsx`

## 3. Bouton de Promotion Responsive (Header UI)

- [x] 3.1 Ajouter la classe `@container` sur la div parente principale de `ExploreAnonymousPanel.tsx`
- [x] 3.2 Ajouter le bouton "Créer le change" dans le header de `ExploreAnonymousPanel.tsx` à côté des boutons système de droite
- [x] 3.3 Configurer le bouton pour qu'il s'affiche uniquement si `ghostId` et `ghostName` sont présents et appeler `onPromote()` lors du clic
- [x] 3.4 Utiliser les Container Queries natives de Tailwind v4 (`hidden @[350px]:inline`) sur l'étiquette texte du bouton pour masquer le texte et n'afficher que l'icône `Sparkles` sur les volets étroits (< 350px)

## 4. Intégration et Orchestration dans KanbanPage

- [x] 4.1 Dans `frontend/src/pages/KanbanPage.tsx`, implémenter le callback `handlePromoteFromPanel` qui recherche l'exploration fantôme correspondante dans la liste `changes` et met à jour `promoteDialog`
- [x] 4.2 Passer la fonction `handlePromoteFromPanel` en tant que prop `onPromote` au composant `ExploreAnonymousBottomPanel`
