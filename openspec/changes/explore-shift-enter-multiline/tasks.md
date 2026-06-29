## 1. ExplorePanel — saisie multiligne

- [x] 1.1 Remplacer `<input type="text">` par `<textarea>` dans `ExplorePanel.tsx`
- [x] 1.2 Ajouter le handler `onKeyDown` : Enter → submit, Shift+Enter → newline natif
- [x] 1.3 Ajouter un `useRef` sur le textarea et un `useEffect` pour l'auto-resize (recalcul à chaque changement de `input`)
- [x] 1.4 Appliquer les styles CSS : `rows="1"`, `resize: none`, `max-height`, `overflow-y: auto`

## 2. ExploreAnonymousPanel — saisie multiligne

- [x] 2.1 Remplacer `<input type="text">` par `<textarea>` dans `ExploreAnonymousPanel.tsx`
- [x] 2.2 Ajouter le handler `onKeyDown` : Enter → submit, Shift+Enter → newline natif
- [x] 2.3 Ajouter un `useRef` sur le textarea et un `useEffect` pour l'auto-resize
- [x] 2.4 Appliquer les styles CSS : `rows="1"`, `resize: none`, `max-height`, `overflow-y: auto`

## 3. Vérification

- [x] 3.1 Vérifier que Shift+Enter insère bien une nouvelle ligne dans ExplorePanel
- [x] 3.2 Vérifier que Shift+Enter insère bien une nouvelle ligne dans ExploreAnonymousPanel
- [x] 3.3 Vérifier que le textarea se redimensionne à la saisie et revient à sa hauteur initiale après envoi
- [x] 3.4 Vérifier que le bouton "Envoyer" fonctionne toujours correctement
