# Design

## Context

`ExploreAnonymousPanel.tsx` et `ExplorePanel.tsx` rendent aujourd'hui chaque message avec un `<div className="max-w-[85%] ... self-end|self-start bg-blue-600|bg-slate-100 ...">` (voir proposal.md - Why). `DetailPanel.tsx:252-269` réutilise le même markup pour la relecture en lecture seule. Le modèle de données partagé, `Message` (`hooks/exploreChat.ts:25-30` et sa quasi-copie `ParsedMessage` dans `hooks/useConversationRun.ts:4-7`), ne porte que `{ role, content, partial?, question? }` : les blocs `content` de type `tool_use`/`tool_result` du flux `stream-json` du subprocess sont filtrés par `extractText()`, dupliquée dans les deux hooks. `QuestionCard` a déjà un style détaché (carte blanche bordée) indépendant du style de bulle. Voir specs/explore-message-layout, specs/explore-markdown-toggle (delta) et specs/explore-structured-questions (delta) pour le comportement attendu.

## Goals / Non-Goals

**Goals:**
- Un seul endroit qui transforme les messages bruts du subprocess (texte + `tool_use`/`tool_result`) en une structure affichable, réutilisé par les deux hooks au lieu de dupliquer `extractText()`.
- Un composant de ligne d'appel d'outil réutilisable entre `ExploreAnonymousPanel`, `ExplorePanel` et `DetailPanel`.
- Aucune régression sur le comportement existant non visé par ce change : scroll-lock, waiting indicator, toggle raw/rendered, structured questions (seul leur habillage visuel change, pas leur logique).

**Non-Goals:**
- Fusionner `ExploreAnonymousPanel` et `ExplorePanel` en un seul composant : ils restent deux fichiers, seul le rendu des tours et des appels d'outils est factorisé.
- Grouper plusieurs appels d'outils d'un même tour sous une puce unique (approche envisagée puis écartée en faveur d'une ligne par appel, cf. exploration) : chaque appel garde sa propre ligne repliable.
- Modifier le protocole backend ou le format du flux `stream-json` : les blocs `tool_use`/`tool_result` sont déjà présents sur stdout, seul le frontend cesse de les jeter.

## Decisions

### Un parseur partagé plutôt que deux
`extractText()` existe en deux copies presque identiques. On extrait une fonction partagée (dans `hooks/exploreChat.ts`, importée par `useConversationRun.ts`) qui, en plus du texte, retourne les `tool_use` rencontrés avant le texte courant sous forme d'un tableau `toolCalls: ToolCall[]` attaché au message assistant en construction. Alternative écartée : garder les deux copies et ajouter la capture des tool_use dans chacune séparément — rejetée, car c'est exactement le genre de duplication qui a permis au filtrage des tool_use de rester incohérent jusqu'ici.

`ToolCall = { id: string; name: string; target: string; status: 'pending' | 'done'; resultPreview?: string }`. `id` = l'`id` du bloc `tool_use` (sert à apparier le `tool_result` correspondant via son `tool_use_id`). `target` est dérivé du nom de l'outil : `input.file_path` pour Read/Write/Edit, `input.pattern` pour Grep, `input.command` (tronqué) pour Bash, sinon une représentation compacte du premier champ de `input`.

### Rattachement au message assistant qui suit
Un `tool_use` est rattaché au message assistant en cours de construction (celui qui accumule le texte suivant), pas à un message séparé : un message assistant du modèle `Message` gagne un champ optionnel `toolCalls?: ToolCall[]`, jamais un rôle ou un type de message à part. Si aucun texte ne suit jamais (tour 100% outils), le message assistant existe quand même avec `content: ''` et ses `toolCalls`, pour rester affichable par les composants existants sans branche spéciale.

### Résultat différé
Le `tool_result` correspondant arrive généralement après que le message a déjà été poussé dans l'historique affiché. Le hook met à jour le `ToolCall` en place (`status: 'done'`, `resultPreview`) par son `id`, dans le message déjà présent dans l'état — pas de nouveau message. Tant que le résultat n'est pas arrivé, la ligne reste `status: 'pending'` (affichée repliée, dépliage sans effet visible autre que "en cours").

### Composant `ToolCallRow` partagé
Nouveau composant `frontend/src/components/ToolCallRow.tsx`, état local `expanded` (non contrôlé, replié par défaut), utilisé par les trois surfaces d'affichage (`ExploreAnonymousPanel`, `ExplorePanel`, `DetailPanel`). Icônes via `lucide-react` (déjà utilisé pour Code/Eye) : mapping `nom d'outil -> icône` avec une icône générique de repli pour un outil non listé. Le `resultPreview` est affiché tel quel (déjà tronqué en amont par le parseur, jamais le JSON brut du bloc).

### Suppression du style de bulle, pas d'alignement
Les classes `self-end`/`self-start`, `max-w-[85%]`, `bg-blue-600`/`bg-slate-100` sont retirées du rendu de tour ; le tour devient un bloc `w-full` avec un libellé de rôle (texte + icône légère) au-dessus du contenu. `QuestionCard` n'est pas touché dans sa structure interne, seul son conteneur perd la contrainte `max-w-[85%]`.

## Risks / Trade-offs

- [Un tour avec beaucoup d'appels d'outils (ex. exploration qui fait 6 `Read` de suite) devient visuellement long, une ligne par appel] → Accepté pour ce change car c'est le comportement observé dans Claude Code/Codex ; un regroupement par tour (option écartée en exploration) reste une évolution possible mais indépendante.
- [Le parseur partagé change un comportement utilisé par deux hooks en même temps] → Couvert par les tests existants d'`exploreChat.test.ts`, à étendre pour la capture des `tool_use`/`tool_result` avant de toucher `useConversationRun.ts`.
- [`tool_result` qui arrive alors que le message assistant est encore `partial: true`] → La mise à jour du `ToolCall` par `id` est indépendante de l'accumulation du texte (`content`) du même message ; les deux mutations ne se marchent pas dessus.

## Migration Plan

Changement frontend pur, sans migration de données ni de schéma persisté (la clé localStorage `explore-view-mode` n'est pas modifiée). Déploiement en une fois, pas de flag : rollback = revert du commit.
