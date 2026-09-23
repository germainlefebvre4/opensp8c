# Tasks

## 1. Modèle de données et parseur partagé

- [ ] 1.1 Étendre le type `Message` (`hooks/exploreChat.ts`) avec `toolCalls?: ToolCall[]` et définir `ToolCall = { id, name, target, status: 'pending' | 'done', resultPreview? }` ; vérifier que `frontend/src/hooks/exploreChat.test.ts` compile toujours
- [ ] 1.2 Extraire de `extractText()` une fonction partagée qui, en plus du texte, capture les blocs `tool_use` rencontrés et les rattache au message assistant en construction (dérivation de `target` par nom d'outil : `file_path` pour Read/Write/Edit, `pattern` pour Grep, `command` tronqué pour Bash, repli générique sinon) ; vérifier avec un nouveau test dans `exploreChat.test.ts` couvrant un flux contenant un bloc `tool_use` suivi de texte
- [ ] 1.3 Mettre à jour le `ToolCall` correspondant (`status: 'done'`, `resultPreview` tronqué) quand un `tool_result` avec le même `tool_use_id` est reçu, sans créer de nouveau message ; vérifier avec un test couvrant l'ordre `tool_use` puis `tool_result` puis texte
- [ ] 1.4 Remplacer la copie dupliquée dans `hooks/useConversationRun.ts` (`ParsedMessage`, `extractText`) par un import de la fonction partagée de 1.2/1.3 ; vérifier que `npm run test` passe toujours pour les tests existants de ce hook
- [ ] 1.5 Vérifier qu'un message assistant sans `tool_use` en amont garde `toolCalls` vide/undefined (pas de régression sur les messages texte simples) via un test dédié

## 2. Composant de ligne d'appel d'outil

- [ ] 2.1 Créer `frontend/src/components/ToolCallRow.tsx` : ligne compacte (icône lucide-react + nom de l'outil + `target`), état local `expanded` (replié par défaut), jamais de JSON brut affiché
- [ ] 2.2 Ajouter le mapping icône par nom d'outil (Read, Write, Edit, Grep, Bash, Glob, repli générique pour un outil non listé)
- [ ] 2.3 Afficher `resultPreview` au dépliage quand `status === 'done'`, et un état visuel "en cours" quand `status === 'pending'` ; vérifier manuellement les deux états dans Storybook/le navigateur (pas de test de rendu existant à étendre pour ce composant : en ajouter un si le projet a un runner de test de composants, sinon vérifier via `npm run dev`)

## 3. Flux plein-largeur — panneaux de chat actifs

- [ ] 3.1 Dans `ExploreAnonymousPanel.tsx`, remplacer le rendu en bulle (`self-end`/`self-start`, `max-w-[85%]`, `bg-blue-600`/`bg-slate-100`) par un bloc `w-full` avec libellé de rôle, en conservant le rendu raw/rendered existant (toggle `mode`) et le curseur `partial` (▊)
- [ ] 3.2 Insérer les `ToolCallRow` (2.1) associées à `msg.toolCalls` au-dessus du texte de chaque message assistant, avant le rendu raw/rendered du contenu
- [ ] 3.3 Répercuter le même changement dans `ExplorePanel.tsx` (structure identique, cf. `explore-markdown-toggle` qui exige la cohérence entre les deux panels) ; vérifier visuellement les deux panels côte à côte avec `npm run dev`
- [ ] 3.4 Retirer la contrainte `max-w-[85%]` du conteneur de `QuestionCard` dans les deux panels sans modifier `QuestionCard.tsx` lui-même ; vérifier que la carte de question occupe toute la largeur tout en gardant son style actuel

## 4. Flux plein-largeur — relecture en lecture seule

- [ ] 4.1 Adapter `DetailPanel.tsx` (onglet log, lignes ~252-269) pour utiliser le même bloc plein-largeur et les mêmes `ToolCallRow` que les panels actifs, à partir des messages déjà enrichis de `toolCalls` par le parseur partagé (section 1)
- [ ] 4.2 Vérifier que l'onglet de relecture n'affiche ni champ de saisie ni bouton d'envoi (comportement lecture seule inchangé) après le changement de rendu

## 5. i18n

- [ ] 5.1 Ajouter les clés de libellé de rôle et d'état replié/déplié nécessaires dans `frontend/src/locales/{en,fr}/explore.json` ; vérifier qu'aucune clé n'est laissée en dur dans le JSX (`npm run lint`)

## 6. Vérification de non-régression

- [ ] 6.1 Exécuter `npm run test` (frontend) et confirmer que les suites `exploreChat.test.ts` et les tests de `useConversationRun` passent
- [ ] 6.2 Vérifier manuellement dans le navigateur : scroll-lock et bouton "Défiler vers le bas" (`explore-session`, `anonymous-explore-session`), bulle d'attente animée (`explore-waiting-indicator`), et cycle complet d'une `QuestionCard` (affichage → réponse → réduction en résumé, `explore-structured-questions`) — ces comportements ne doivent pas régresser avec le nouveau flux
