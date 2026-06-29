## 1. Hooks — état `waiting`

- [x] 1.1 Ajouter `const [waiting, setWaiting] = useState(false)` dans `useExploreSession`
- [x] 1.2 Appeler `setWaiting(true)` dans `send()` de `useExploreSession` (avant l'envoi WebSocket)
- [x] 1.3 Appeler `setWaiting(false)` quand `extractText(data)` retourne une chaîne non-vide dans `useExploreSession`
- [x] 1.4 Appeler `setWaiting(false)` dans `onclose` et `onerror` de `useExploreSession`
- [x] 1.5 Exposer `waiting` dans le retour de `useExploreSession`
- [x] 1.6 Répéter les étapes 1.1–1.5 pour `useAnonymousExploreSession`

## 2. Composant — bulle animée (CSS)

- [x] 2.1 Ajouter les styles d'animation CSS trois-points dans `index.css` (ou équivalent global) : `@keyframes` avec `opacity` et `animation-delay` sur trois `<span>`
- [x] 2.2 Créer le sous-composant (ou fragment inline) `TypingBubble` qui rend la bulle animée dans le style des messages assistant existants

## 3. ExploreAnonymousPanel — intégration

- [x] 3.1 Ajouter la prop `assistantName?: string` (défaut `"Claude"`) à l'interface `Props` de `ExploreAnonymousPanel`
- [x] 3.2 Lire `waiting` depuis `useAnonymousExploreSession`
- [x] 3.3 Afficher `TypingBubble` dans le fil de messages quand `waiting === true`
- [x] 3.4 Ajouter un `useEffect` sur `waiting` : lancer un `setTimeout(5000)` qui active `showSlowLabel`, le nettoyer quand `waiting` repasse à `false`
- [x] 3.5 Afficher `"{assistantName} réfléchit..."` dans la bulle quand `showSlowLabel === true`

## 4. ExplorePanel — intégration

- [x] 4.1 Ajouter la prop `assistantName?: string` (défaut `"Claude"`) à l'interface `Props` de `ExplorePanel`
- [x] 4.2 Lire `waiting` depuis `useExploreSession`
- [x] 4.3 Afficher `TypingBubble` dans le fil de messages quand `waiting === true`
- [x] 4.4 Ajouter le même `useEffect` timeout 5s / `showSlowLabel` que dans `ExploreAnonymousPanel`
- [x] 4.5 Afficher `"{assistantName} réfléchit..."` dans la bulle quand `showSlowLabel === true`

## 5. Vérification

- [x] 5.1 Envoyer un message dans une session anonyme et vérifier que la bulle apparaît immédiatement
- [x] 5.2 Vérifier que la bulle disparaît dès l'arrivée du premier token streamé
- [x] 5.3 Vérifier que le label "Claude réfléchit..." apparaît après ~5s (simulable en coupant le réseau temporairement)
- [x] 5.4 Vérifier que l'input reste actif et utilisable pendant l'attente
- [x] 5.5 Vérifier le même comportement dans `ExplorePanel` (session nommée)
<!-- vérification manuelle requise -->
