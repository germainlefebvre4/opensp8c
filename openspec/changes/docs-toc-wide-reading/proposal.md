# Proposal

## Why

Dans le sous-onglet « Documentation » de la vue Specs, le contenu est plafonné à `max-w-3xl` (768 px) et collé à gauche : sur un écran large, la moitié de la zone reste vide, le texte est étroit et les gros diagrammes Mermaid (dont `max-width` est borné par la zone) sont rendus plus petits que nécessaire. Les pages font plusieurs écrans et il n'y a aucune navigation interne : l'utilisateur doit scroller pour retrouver une section. Le sous-onglet « Spécifications » dispose déjà d'une table des matières, mais pas la documentation.

## What Changes

- **Table des matières à droite** de la zone de contenu du sous-onglet Documentation : titres h2 et h3 de la page affichée, cliquables (défilement doux vers la section), avec surlignage de la section en cours de lecture. Le h1 (titre de la page) n'y figure pas.
- **TOC adaptatif à la zone, pas à la fenêtre** : sa largeur est fluide entre un minimum et un maximum, et il disparaît lorsque la zone devient trop étroite pour conserver une largeur de lecture confortable au texte principal. La mesure porte sur la largeur réelle du panneau (qui varie avec le repli de la sidebar de l'application), pas sur la largeur de la fenêtre.
- **Texte principal élargi** : le plafond de largeur du contenu passe d'environ 768 px à environ 1024 px (`max-w-5xl`) pour tout le contenu (texte, tableaux, code, diagrammes). Les diagrammes Mermaid ne sont pas agrandis artificiellement : les grands profitent de la largeur gagnée, les petits conservent leur taille naturelle.
- **Identifiants de titres fiables** : le TOC et le rendu Markdown dérivent leurs ancres d'une source unique, ce qui évite les entrées mortes quand un titre contient du code inline (par ex. ``### Agent role settings (`preferences` + `agents`)``) et ignore les lignes `#` situées dans des blocs de code.

Hors périmètre :
- Le sous-onglet « Spécifications » et son TOC actuel (`SpecsPage`) restent inchangés.
- Aucun changement backend ni de format des pages générées.
- Pas de TOC inter-pages ni de recherche dans la documentation.

## Capabilities

### New Capabilities

Aucune.

### Modified Capabilities

- `spec-documentation-view`: ajout d'une table des matières cliquable et adaptative pour la page de documentation affichée, et d'une largeur de lecture élargie du contenu.

## Impact

- Frontend uniquement : `frontend/src/components/DocumentationPanel.tsx`, `frontend/src/components/TableOfContents.tsx` (réutilisé, adaptations mineures), nouveau module partagé pour l'extraction des titres et le calcul des ancres, éventuellement `frontend/src/components/MermaidDiagram.tsx` (style du conteneur).
- Tests frontend du panneau de documentation et du module d'ancres.
- Clés i18n existantes (`toc.title`) réutilisées ; aucune nouvelle dépendance attendue (container queries natives de Tailwind v4).
- Documentation : `docs/opensp8c/` n'est pas modifiée à la main (régénérée par la génération de documentation).
