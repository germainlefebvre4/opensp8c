# Design

## Context

Six endroits du frontend appellent `<ReactMarkdown>` directement, chacun avec ses propres classes `prose` : `DocumentationPanel`, `SpecsPage` (composants `h1`-`h3` avec ancres pour la table des matières), `DetailPanel` (3 usages), `ExplorePanel`, `ExploreAnonymousPanel`, `AgentRunPanel`. Seul `DocumentationPanel` surcharge `code` (pour mermaid, via `MermaidDiagram`, chargé en lazy). Aucun n'active de plugin remark.

Constats à l'origine du bug (voir proposal.md) : `remark-gfm` est absent, et `@tailwindcss/typography` (déjà activé dans `index.css`) ajoute `code::before/::after { content: "`" }`. `react-markdown` v10 ne fournit plus de prop `inline` aux composants `code` : inline et bloc ne se distinguent que par la structure (`<pre><code>`) ou la classe `language-*`.

## Goals / Non-Goals

**Goals:**
- Un seul composant `Markdown` qui porte tout le rendu commun (GFM, styles tableau/code).
- Rendu des tableaux et du code inline corrigé dans les 6 usages sans changer leur mise en page (taille `sm`/`xs`, alignement).
- Le chargement lazy de mermaid n'est pas dégradé pour les panneaux qui n'en ont pas besoin.

**Non-Goals:**
- Coloration syntaxique des blocs de code.
- Sanitisation HTML ou support du HTML brut dans le Markdown (comportement actuel de `react-markdown` conservé).
- Modification du toggle raw/rendered de l'exploration ou du format des docs générés par le backend.

## Decisions

**1. Composant `Markdown` avec props de variation, pas de wrapper par panneau.**
Props : `children` (contenu), `size?: 'sm' | 'xs'` (défaut `sm`), `mermaid?: boolean`, `components?` (surcharges fusionnées, pour les titres de `SpecsPage`), `className?`. Il rend `<article className="prose prose-slate ...">` + `<ReactMarkdown remarkPlugins={[remarkGfm]}>`. Alternative écartée : un plugin/CSS global dans `index.css` seul, qui ne règle pas les tableaux (la syntaxe n'est pas parsée sans GFM) et laisse la configuration dupliquée.

**2. Style du code inline par sélecteur CSS, pas par détection dans le composant `code`.**
Les styles inline s'appliquent via des variantes Tailwind sur le conteneur `prose` ciblant `:not(pre) > code` (fond gris, padding, arrondi, police mono, `before/after:content-none`). Un bloc est toujours `pre > code` avec ou sans langage, donc il est exclu sans heuristique. Alternative écartée : décider inline/bloc dans `code` d'après `language-*` ou la présence d'un `\n` : fragile pour un bloc sans langage d'une seule ligne. Le composant `code` ne reste surchargé que pour mermaid.

**3. Mermaid optionnel via la prop `mermaid`.**
Quand elle est vraie, `code` délègue à `MermaidDiagram` pour `language-mermaid` (logique déplacée depuis `DocumentationPanel`, repli sur bloc brut inchangé). `MermaidDiagram` reste importé statiquement mais charge le paquet `mermaid` en dynamique, donc aucun coût pour les autres panneaux. Les autres panneaux (chat, détail) affichent un bloc mermaid comme code brut, comme aujourd'hui.

**4. Tableaux : enveloppe défilante via composant `table`.**
Le composant `table` est surchargé pour rendre `<div className="overflow-x-auto"><table>…</table></div>`, ce qui évite que les tableaux larges élargissent le panneau. Bordures et en-têtes par variantes `prose-th:` / `prose-td:` / `prose-table:`, pour rester dans le thème zinc remappé de `index.css`.

**5. Fusion des `components`.**
`Markdown` fusionne : composants internes (`table`, et `code` si mermaid) puis surcharges passées par le panneau. `SpecsPage` continue de passer ses `h1`-`h3` mémoïsés.

## Risks / Trade-offs

- [Les tailles `prose-xs` sont utilisées dans `DetailPanel`/`AgentRunPanel` alors que le plugin typography n'en définit pas] → conserver telle quelle la classe existante via `size`, sans en changer le comportement actuel ; hors périmètre de la corriger.
- [Changement visuel dans des panneaux non signalés (détail, exploration, runs)] → voulu (cohérence) ; le mode `raw` de l'exploration reste intact.
- [Contenu non fiable (messages agent) rendu avec GFM] → GFM n'ajoute pas de HTML brut ; `react-markdown` continue d'ignorer le HTML par défaut. Les liens automatiques GFM restent des `<a>` standard.
- [Tests jsdom : mermaid non rendu] → les tests vérifient le repli et la structure, pas le SVG.

## Open Questions

- Teinte exacte du fond du code inline (zinc-100 proposé, zinc-200 en alternative) : tranchable à l'implémentation sans impact sur specs ni tâches.
