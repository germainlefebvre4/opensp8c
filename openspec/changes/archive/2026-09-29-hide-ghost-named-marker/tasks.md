# Tasks

## 1. Backend : retrait du marker ghost_named

- [x] 1.1 Ajouter un filtre à état de markers `ghost_*` (alimenté par deltas, vidage en fin de bloc) dans `backend/internal/session` ou `handlers`, avec tests unitaires table-driven : marker en un bloc, fragmenté token par token, mélangé à du texte, `{` légitime invalidé, marker jamais complété ; vérifier avec `go test ./internal/...`
- [x] 1.2 Étendre le nettoyage regex des messages consolidés à `ghost_named` (valeurs décodées) et l'appliquer dans la goroutine de sortie de `explore.go` pour les sessions anonymes ; tests dans `explore_test.go` sur le flux réel `2026-09-29T15-12-38Z.jsonl` (rejouer les messages et vérifier qu'aucun texte relayé ne contient `ghost_named`) ; vérifier avec `go test ./internal/api/handlers/...`
- [x] 1.3 Vérifier que l'événement WS `ghost_named` et le renommage du ghost (suffixe de collision inclus) restent inchangés : les tests existants de `manager_test.go` / `explore_test.go` passent

## 2. Frontend : nettoyage et notice

- [x] 2.1 Ajouter `stripGhostMarkers` dans `exploreChat.ts` (markers complets + suffixe partiel) et l'utiliser dans `mergeAssistantText` sur le contenu accumulé ; tests dans `exploreChat.test.ts` (fragmentation token par token, suffixe partiel retenu puis restitué, `ghost_question` toujours nettoyé) ; vérifier avec `npm test` dans `frontend/`
- [x] 2.2 Ajouter le rôle `notice` au type `Message`, l'insérer sur l'événement WS `ghost_named` dans `useAnonymousExploreSession.ts` sans doublon (remplacement si existante) et le persister via `saveMessages` ; tests sur le helper d'insertion (nom initial, collision, rechargement) ; vérifier avec `npm test`
- [x] 2.3 Rendre la notice en ligne compacte (icône + texte) dans `ExploreAnonymousPanel`, `ExplorePanel`, identique en mode brut et rendu ; ajouter la clé `namedNotice` dans `locales/fr/explore.json` et `locales/en/explore.json` ; vérifier avec `npm run build` et le test i18n existant

## 3. Documentation et intégration

- [x] 3.1 Mettre à jour la section markers de `docs/opensp8c/architecture.md` (filtre à état, `ghost_named` désormais retiré, notice) ; vérifier que le texte correspond au code livré
- [x] 3.2 Vérification bout en bout : lancer l'app, ouvrir une exploration anonyme et confirmer qu'aucun JSON n'apparaît pendant le streaming, que la notice apparaît une fois avec le bon nom et qu'un rechargement ne la duplique pas
