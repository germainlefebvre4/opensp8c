# Tasks

## 1. Backend - Extraction et purge multi-questions

- [x] 1.1 Implémenter `ExtractAllGhostQuestions` dans `backend/internal/session/manager.go` pour extraire tous les marqueurs `ghost_question` présents dans un texte ou flux JSONL, et vérifier avec des tests unitaires dans `manager_test.go`.
- [x] 1.2 Mettre à jour `detectGhostQuestion` et `stripGhostQuestionMarker` dans `backend/internal/api/handlers/explore.go` pour diffuser un événement WebSocket `ghost_question` par question détectée et purger intégralement les marqueurs de tous les messages sortants (deltas et texte consolidé).
- [x] 1.3 Écrire des tests dans `backend/internal/api/handlers/explore_test.go` vérifiant que la réception de multiples marqueurs `ghost_question` émet chaque événement et ne laisse aucun JSON brut dans les messages textuels.

## 2. Frontend - Gestion multi-questions et logique de staging

- [x] 2.1 Mettre à jour `appendQuestionMessage` dans `frontend/src/hooks/exploreChat.ts` pour que l'arrivée d'une nouvelle question ne marque plus les questions antérieures non répondues comme `superseded`.
- [x] 2.2 Ajouter dans `frontend/src/hooks/exploreChat.ts` les helpers de consolidation des réponses (`buildConsolidatedUserMessage`, nettoyage défensif du texte de l'assistant) et valider par des tests unitaires dans `frontend/src/hooks/exploreChat.test.ts`.

## 3. Frontend - Saisie directe et staging dans QuestionCard

- [x] 3.1 Adapter `frontend/src/components/QuestionCard.tsx` pour inclure un champ de saisie directe permettant de renseigner une réponse et de la valider localement sans envoi WebSocket direct.
- [x] 3.2 Ajouter dans `QuestionCard.tsx` l'affichage visuel d'une réponse préparée (« Réponse en attente d'envoi ») avec possibilité de ré-éditer ou d'annuler la saisie.
- [x] 3.3 Valider le comportement du composant `QuestionCard` via des tests automatisés ou tests de composants dans `frontend/src/components/QuestionCard.test.tsx`.

## 4. Frontend - Encarts temporaires dans le fil et envoi consolidé

- [x] 4.1 Ajouter dans `frontend/src/components/ExplorePanel.tsx` et `frontend/src/components/ExploreAnonymousPanel.tsx` la gestion de l'état `stagedAnswers` et l'affichage des encarts temporaires dans le fil de discussion (avec boutons Modifier et Supprimer).
- [x] 4.2 Mettre à jour la barre de saisie et la fonction d'envoi : activer le bouton d'envoi dès qu'une réponse est préparée même sans prompt libre, transmettre le message consolidé, fermer les encarts temporaires et conserver les questions non répondues ouvertes.
- [x] 4.3 Mettre à jour `useExploreSession.ts` et `useAnonymousExploreSession.ts` pour enregistrer les réponses consolidées dans l'historique et exécuter la suite de tests complète (`npm test` dans `frontend/`, `go test ./...` dans `backend/`).
