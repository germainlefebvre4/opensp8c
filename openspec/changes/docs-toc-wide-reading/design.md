# Design

## Context

Voir proposal.md pour la motivation. État actuel, frontend uniquement :

- `DocumentationPanel` rend `[liste des pages w-44] [contenu]`. Le contenu est un `ScrollArea` Radix dont le `Viewport` est l'élément qui défile, avec un article `prose prose-sm` plafonné à `max-w-3xl`.
- `TableOfContents` (déjà utilisé par `SpecsPage`) reçoit `headings` et l'élément défilant (`contentEl`), pose un `IntersectionObserver` pour le surlignage et fait `scrollIntoView` au clic. Il est réutilisable tel quel.
- `SpecsPage` calcule les titres en analysant le markdown brut (`parseHeadings`) et pose les `id` par un composant de titre React (`makeHeadingComponent`). Les deux calculent le slug différemment : un titre contenant du code inline (``### Agent role settings (`preferences` + `agents`)`` dans `architecture.md`) produit deux ids distincts, et les lignes `#` des blocs de code sont prises pour des titres. Le compteur de doublons est une `Map` mutée pendant le rendu (fragile en StrictMode).
- `MermaidDiagram` injecte le SVG de `mermaid.render` sans aucun style de conteneur ; Mermaid pose un `max-width` égal à la taille naturelle du diagramme.
- Tailwind v4 (`@tailwindcss/vite`) : les container queries (`@container`, `@[66rem]:`) et les unités `cqw` sont natives. Les tests Vitest tournent en environnement `node` (fonctions pures, `renderToStaticMarkup`).

## Goals / Non-Goals

**Goals:**
- TOC fiable (ancres toujours cohérentes avec le rendu) et adaptatif à la largeur réelle du panneau.
- Contenu plus large jusqu'à ~1024 px, sans agrandir artificiellement les diagrammes.

**Non-Goals:**
- Migrer `SpecsPage` vers le nouveau module d'ancres (peut suivre dans un autre change ; son comportement est explicitement « inchangé » dans la spec).
- Rendu mobile dédié, TOC repliable manuellement, mémorisation de préférence d'affichage.

## Decisions

### D1. Largeur adaptative par container queries CSS plutôt que par JS
Le panneau racine devient un conteneur (`@container`). Le TOC est `hidden` par défaut et `@[66rem]:block` ; sa largeur est `clamp(11rem, 18cqw, 16rem)`. Le seuil 66rem = liste des pages (11rem) + texte principal minimal (44rem ≈ 70 caractères en `prose-sm`) + TOC minimal (11rem).

Pourquoi : la zone utile varie avec le repli de la sidebar de l'application, que les breakpoints de fenêtre (`lg:`) ne voient pas ; le CSS évite tout re-rendu et tout état.
Alternatives : (a) `useContainerWidth` (existe déjà, ResizeObserver) : fonctionne mais ajoute état et re-rendus à chaque pixel de redimensionnement ; (b) breakpoint de fenêtre comme `SpecsPage` : ne répond pas au besoin. Le seuil et les bornes sont des constantes de classe faciles à ajuster.

### D2. Ancres dérivées du DOM rendu (source unique)
Un hook `useRenderedHeadings(articleRef, content)` parcourt, après rendu (`useLayoutEffect`), les `h2, h3` de l'article, lit leur `textContent` (donc le texte réellement affiché, code inline inclus), calcule les ids avec une fonction pure `buildHeadingIds` (slug Unicode `\p{L}\p{N}`, dédoublonnage `-1`, `-2`, repli `section` si vide, remise à zéro à chaque passe donc idempotente), pose `el.id` et renvoie la liste `{level, text, id}`.

Pourquoi : le TOC et les ancres lisent la même chose (le DOM), donc ne peuvent pas diverger ; les blocs de code sont ignorés naturellement ; pas de compteur muté pendant le rendu. React ne gère pas l'attribut `id` de ces éléments, il n'est donc pas écrasé par les re-rendus.
Alternatives : (a) réutiliser `parseHeadings` + `makeHeadingComponent` : reproduit le bug des titres avec code inline ; (b) plugin rehype posant les ids et collectant les titres : propre mais plus lourd (plugin custom + canal de retour vers le composant, ou dépendances transitives de react-markdown à importer directement).

Seule `buildHeadingIds` est pure et testée unitairement (environnement `node`) ; le hook reste fin.

### D3. Réutilisation de `TableOfContents`, contenu défilant inchangé
Le TOC est placé hors de la zone qui défile (comme dans `SpecsPage`), donc toujours visible, avec son propre `overflow-y-auto`. `contentEl` est le `Viewport` du `ScrollArea`. Le h1 est exclu en ne sélectionnant que `h2, h3`. Au changement de page, le `scrollTop` du viewport est remis à 0 (sinon la position précédente est conservée sur la nouvelle page).

### D4. Plafond de largeur `max-w-5xl`, article centré
`max-w-3xl` devient `max-w-5xl` pour tout le contenu (choix utilisateur : pas de traitement séparé des diagrammes). L'article est centré (`mx-auto`) dans la colonne de contenu : avec un TOC collé au bord droit, un article aligné à gauche laisserait un grand vide entre les deux sur les très grands écrans. Quand le TOC est masqué ou la zone étroite, le rendu est identique à l'alignement à gauche puisque l'article remplit la colonne.

### D5. Diagrammes : contenir, ne pas agrandir
Style sur `.mermaid-diagram` : `overflow-x: auto` en garde-fou, et le `svg` interne avec `max-width: 100%; height: auto`. On ne force pas `width: 100%`, ce qui agrandirait les petits diagrammes au-delà de leur taille naturelle (spec : « sans être agrandis au-delà de leur taille naturelle »). Les gros diagrammes gagnent mécaniquement avec la largeur du contenu.

## Risks / Trade-offs

- [Seuil 66rem mal calibré selon les écrans] → constantes de classe isolées, vérification visuelle à plusieurs largeurs (sidebar ouverte/repliée) pendant l'implémentation.
- [Tableaux ou blocs de code plus larges que 1024 px] → `prose` gère déjà le défilement horizontal des `pre` ; vérifier les tableaux et ajouter `overflow-x-auto` si besoin.
- [Décalage de mise en page quand un diagramme Mermaid se rend après un clic dans le TOC] → le suivi de lecture repose sur `IntersectionObserver` et se recale seul ; le défilement du clic peut atterrir légèrement avant la cible si un diagramme situé au-dessus finit de se rendre après le clic, acceptable.
- [Mutation de `el.id` hors React] → idempotente et limitée à des éléments dont React ne contrôle pas l'attribut ; couverte par la remise à zéro à chaque passe.
- [Deux implémentations d'ancres coexistent jusqu'à une migration de `SpecsPage`] → dette assumée, hors périmètre.
