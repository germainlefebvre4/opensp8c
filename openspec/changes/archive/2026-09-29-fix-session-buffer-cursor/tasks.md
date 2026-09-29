## 1. Backend : curseur absolu

- [x] 1.1 Ajouter le champ `dropped` à `Session` (`backend/internal/session/manager.go`) et un helper `appendMessage` qui évince la plus ancienne entrée en incrémentant `dropped`, sous `msgMu`
- [x] 1.2 Utiliser `appendMessage` dans `startFanOut` et `InjectMessage` (suppression de la duplication)
- [x] 1.3 Adapter `Snapshot` (curseur = `dropped + len`) et `MessagesSince` (`start = cursor - dropped` clampé à `[0, len]`, retour `dropped + len`)
- [x] 1.4 Mettre à jour les commentaires de `MessagesSince` / `Snapshot` pour décrire le curseur absolu

## 2. Backend : tests

- [x] 2.1 Test `MessagesSince` : livraison de chaque nouveau message après saturation (pousser plus de `maxMessages` entrées en lisant à chaque étape, aucune perte ni doublon, ordre conservé)
- [x] 2.2 Test curseur périmé : reprise à la plus ancienne entrée disponible
- [x] 2.3 Test `Snapshot` puis `MessagesSince` sur buffer saturé : pas de doublon ni de trou entre replay et live
- [x] 2.4 Test `InjectMessage` sur buffer saturé (mêmes garanties)

## 3. Frontend : fin de tour

- [x] 3.1 Ajouter `isTurnEnd(data)` dans `frontend/src/hooks/exploreChat.ts` (`result` ou `message_complete`) avec tests unitaires dans `exploreChat.test.ts`
- [x] 3.2 Appeler `setWaiting(false)` sur fin de tour dans `useExploreSession.ts`
- [x] 3.3 Même comportement dans `useAnonymousExploreSession.ts`

## 4. Vérification

- [x] 4.1 Lancer les tests Go (`go test ./internal/session/... ./internal/api/...`) et les tests frontend
- [x] 4.2 Reproduire manuellement : session Explore avec plusieurs tours et appels d'outils dépassant 500 événements, vérifier que texte final et résultats d'outils s'affichent sans rechargement
