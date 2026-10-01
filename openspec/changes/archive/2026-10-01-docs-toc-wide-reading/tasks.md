# Tasks

## 1. Identifiants de titres fiables

- [x] 1.1 Créer `frontend/src/lib/headingIds.ts` avec `slugify` (Unicode `\p{L}\p{N}`, repli `section`) et `buildHeadingIds(headings)` (dédoublonnage `-1`, `-2`, idempotent), puis `headingIds.test.ts` couvrant doublons, accents, titre vide, titre avec ponctuation/code inline ; vérifier avec `cd frontend && npx vitest run src/lib/headingIds.test.ts`
- [x] 1.2 Créer le hook `frontend/src/hooks/useRenderedHeadings.ts` : parcourt `h2, h3` de l'article après rendu (`useLayoutEffect`, dépendant du contenu), pose `el.id` via `buildHeadingIds` et renvoie `{level, text, id}[]` ; vérifier que `npx tsc -b` et `npm run lint` passent depuis `frontend/`

## 2. Mise en page de la documentation

- [x] 2.1 Dans `DocumentationPanel.tsx`, faire du panneau un conteneur (`@container`), passer l'article de `max-w-3xl` à `max-w-5xl` centré (`mx-auto`) et remettre `scrollTop` du viewport à 0 au changement de page ; vérifier visuellement (zone large : contenu jusqu'à ~1024 px centré ; changement de page : retour en haut)
- [x] 2.2 Ajouter à droite du contenu, hors zone défilante, un `<aside>` contenant `TableOfContents` alimenté par `useRenderedHeadings` et le viewport comme `contentEl` ; classes `hidden @[66rem]:block`, largeur `clamp(11rem, 18cqw, 16rem)`, `overflow-y-auto`, non rendu si aucun titre h2/h3 ; vérifier qu'un clic défile vers la section, que le surlignage suit le défilement et que le h1 n'est pas listé
- [x] 2.3 Styler `.mermaid-diagram` (`overflow-x: auto`, `svg` en `max-width: 100%; height: auto`) dans `MermaidDiagram.tsx` ou `index.css` ; vérifier dans `architecture.md` que le gros graphe est plus grand qu'avant et qu'un petit diagramme garde sa taille naturelle
- [x] 2.4 Contrôler le débordement des tableaux et blocs de code de `docs/opensp8c/*.md` à 1024 px et ajouter `overflow-x-auto` si nécessaire ; vérifier qu'aucun scroll horizontal de page n'apparaît

## 3. Vérification d'intégration

- [x] 3.1 Lancer l'application (skill `run`) et, sur chaque page de `docs/opensp8c/`, vérifier : toutes les entrées du TOC sont cliquables (dont « Agent role settings (`preferences` + `agents`) » dans architecture), aucune ligne de bloc de code n'y figure, le TOC disparaît en réduisant la fenêtre ou en ouvrant la sidebar de l'application et réapparaît en l'élargissant
- [x] 3.2 Vérifier que le sous-onglet « Spécifications » est inchangé (TOC `lg:` existant) puis lancer `cd frontend && npx vitest run && npm run lint && npm run build` sans erreur
