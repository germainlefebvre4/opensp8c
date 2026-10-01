# Proposal

## Why

Dans le sous-onglet « Documentation » de la vue Specs, le Markdown est mal rendu : les tableaux apparaissent comme des lignes de pipes brutes (`react-markdown` suit CommonMark pur, sans l'extension GFM, et `remark-gfm` n'est pas installé) et le code inline s'affiche avec des backticks littéraux et sans style (le plugin `@tailwindcss/typography` injecte `code::before/::after` avec des backticks). Les pages générées dans `docs/opensp8c/` contiennent pourtant des tableaux et du code inline. Le même défaut existe dans les 5 autres usages de `<ReactMarkdown>` de l'application, chacun configuré à la main : corriger uniquement la documentation laisserait un rendu incohérent.

## What Changes

- Ajouter la dépendance `remark-gfm` au frontend.
- Introduire un composant partagé `Markdown` qui centralise : le plugin GFM, les classes `prose` (tailles `sm` / `xs`), la suppression des backticks automatiques du code inline, le style du code inline (fond gris, police mono, arrondi, sans backticks, à la GitHub) et le style des tableaux (bordures, en-têtes, défilement horizontal si débordement).
- Le composant distingue code inline et bloc de code : le style inline ne s'applique jamais aux blocs ``` ```.
- Le rendu des blocs `mermaid` (avec repli sur le bloc brut en cas d'erreur) devient une option du composant, activée pour la documentation.
- Migrer les 6 usages existants vers ce composant : `DocumentationPanel`, `SpecsPage` (en conservant ses ancres de titres h1–h3), `DetailPanel`, `ExplorePanel`, `ExploreAnonymousPanel`, `AgentRunPanel`.
- Ajouter des tests de rendu (tableau, code inline, bloc de code, mermaid).

## Capabilities

### New Capabilities
- `markdown-rendering`: contrat de rendu Markdown commun à toute l'interface (tableaux GFM, code inline stylé sans backticks, blocs de code distincts du code inline).

### Modified Capabilities
- `spec-documentation-view`: l'exigence « Rendu Markdown avec diagrammes Mermaid » s'étend aux tableaux et au code inline de la page de documentation affichée.

## Impact

- Frontend uniquement : nouveau `frontend/src/components/Markdown.tsx` (+ test), modification des 6 composants/pages listés, `frontend/package.json` et `package-lock.json` (`remark-gfm`).
- Aucun changement backend ni d'API.
- Changement visuel : tous les panneaux affichant du Markdown (détail de change, exploration, runs d'agent, specs brutes) rendent désormais les tableaux et le code inline. Le toggle raw/rendered de l'exploration (`explore-markdown-toggle`) est inchangé : seul le mode rendu est amélioré.
