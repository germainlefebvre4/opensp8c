# Tasks

## 1. Dépendance

- [x] 1.1 Ajouter `remark-gfm` à `frontend/package.json` (`npm install remark-gfm` dans `frontend/`) et vérifier que `npm ls remark-gfm` le liste et que `npm run build` passe

## 2. Composant partagé `Markdown`

- [x] 2.1 Créer `frontend/src/components/Markdown.tsx` (props `children`, `size`, `mermaid`, `components`, `className` ; `remarkGfm` ; classes `prose` ; fusion des `components`) et vérifier qu'il compile (`npm run build`)
- [x] 2.2 Ajouter les styles du code inline (`:not(pre) > code` : fond gris, padding, arrondi, mono, `before/after:content-none`) et vérifier par test que `` `x` `` produit un `<code>` hors `<pre>` sans backtick dans le texte
- [x] 2.3 Surcharger `table` pour l'envelopper dans un conteneur `overflow-x-auto`, styler bordures et en-têtes, et vérifier par test qu'un tableau GFM produit `<table>`, `<th>`, `<td>` dans un conteneur défilant
- [x] 2.4 Déplacer la logique mermaid (`language-mermaid` → `MermaidDiagram`) derrière la prop `mermaid` et vérifier par test que sans la prop un bloc mermaid reste un `<pre><code>`
- [x] 2.5 Écrire `frontend/src/components/Markdown.test.tsx` (tableau, cellule avec code inline/gras, code inline, bloc sans langage hors style inline, bloc avec langage, mermaid activé/désactivé) et vérifier que `npm test -- Markdown` passe

## 3. Migration des usages

- [x] 3.1 Migrer `DocumentationPanel.tsx` (`mermaid`, `size="sm"`), retirer son `CodeRenderer` local, et vérifier que les tests existants passent
- [x] 3.2 Migrer `SpecsPage.tsx` en passant les `h1`-`h3` mémoïsés via `components`, et vérifier que les ancres de titres et la table des matières fonctionnent encore (test ou vérification manuelle sur `/specs`)
- [x] 3.3 Migrer `DetailPanel.tsx` (3 usages, `size="xs"` pour les résumés), `ExplorePanel.tsx`, `ExploreAnonymousPanel.tsx` et `AgentRunPanel.tsx`, et vérifier qu'il ne reste aucun import direct de `react-markdown` hors de `Markdown.tsx` (`grep -rn "from 'react-markdown'" frontend/src`)

## 4. Vérification d'ensemble

- [x] 4.1 Lancer `npm run build`, `npm run lint` et `npm test` dans `frontend/` et vérifier qu'ils passent
- [x] 4.2 Ouvrir `http://localhost:5174/specs?workspace=8b4a8cb8`, sous-onglet Documentation, page `overview`, et vérifier que les tableaux sont rendus avec bordures et que `openspec/` apparaît en code inline sans backticks ; vérifier aussi un message d'exploration en mode rendu
