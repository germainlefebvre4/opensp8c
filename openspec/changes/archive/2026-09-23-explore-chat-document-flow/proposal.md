# Proposal

## Why

Le fil de chat des panneaux d'exploration (bulles alignées à gauche/à droite, `bg-blue-600`/`bg-slate-100`, largeur 85%) casse la cohérence de lecture d'une conversation et masque totalement les appels d'outils (Read, Grep, Write, ...) exécutés par l'agent — `extractText()` les filtre aujourd'hui, seul le texte final apparaît. On veut se rapprocher de l'UX Claude Code / Codex / Antigravity : les tours occupent toute la largeur disponible et les actions d'outils sont visibles sous forme de lignes repliables, sans reproduire le bruit brut (JSON) de ces appels.

## What Changes

- Suppression du rendu en bulles (`self-end`/`self-start`, fond coloré, `max-w-[85%]`) dans `ExploreAnonymousPanel`, `ExplorePanel` et l'onglet de relecture en lecture seule de `DetailPanel` : les tours utilisateur et assistant s'affichent en pleine largeur, différenciés par un léger label de rôle plutôt que par un alignement gauche/droite.
- Capture des blocs `tool_use`/`tool_result` aujourd'hui filtrés par `extractText()` (dans `hooks/exploreChat.ts` et sa duplication dans `hooks/useConversationRun.ts`) et rattachement au message assistant qui les suit.
- Affichage de chaque appel d'outil capturé comme une ligne compacte (icône + nom d'outil + cible), repliée par défaut, dépliable pour voir un aperçu du résultat ; jamais le JSON brut de l'appel.
- `QuestionCard` conserve son style actuel (carte blanche bordée, distincte du fil) mais s'étire sur toute la largeur disponible au lieu de sa largeur actuelle contrainte par le flux de bulles.
- Le toggle raw/rendered existant (`explore-markdown-toggle`) continue de s'appliquer au texte de la réponse assistant, désormais à l'intérieur du bloc plein-largeur plutôt que d'une bulle.

## Capabilities

### New Capabilities
- `explore-message-layout`: flux de lecture plein-largeur pour les tours utilisateur/assistant (fini les bulles gauche/droite) et affichage repliable des appels d'outils au-dessus de la réponse assistant qui les suit ; partagé par `ExploreAnonymousPanel`, `ExplorePanel` et la relecture en lecture seule dans `DetailPanel`.

### Modified Capabilities
- `explore-markdown-toggle`: les scénarios qui décrivent le rendu comme une bulle (`bg-blue-600`, `prose ... max-w-none` à l'intérieur d'une bulle assistant) sont mis à jour pour décrire un bloc plein-largeur ; le comportement raw/rendered par rôle et sa persistance localStorage sont inchangés.
- `explore-structured-questions`: le scénario qui décrit la distinction visuelle de la carte de question par contraste avec « le style de bulle standard (`bg-slate-100`) » est mis à jour pour la décrire par contraste avec le flux plein-largeur (`explore-message-layout`), sans bulle standard à opposer.

## Impact

- Frontend : `frontend/src/components/ExploreAnonymousPanel.tsx`, `frontend/src/components/ExplorePanel.tsx`, `frontend/src/components/DetailPanel.tsx` (onglet log), `frontend/src/components/QuestionCard.tsx` (largeur), `frontend/src/hooks/exploreChat.ts` et `frontend/src/hooks/useConversationRun.ts` (parsing `tool_use`/`tool_result` au lieu de les filtrer), nouveau composant de ligne d'appel d'outil.
- Aucun changement backend : les blocs `tool_use`/`tool_result` sont déjà présents dans le stream stdout du subprocess ; seul le filtrage côté frontend change.
- `frontend/src/locales/{en,fr}/explore.json` : nouvelles clés pour les libellés de rôle et l'état replié/déplié des appels d'outils.
