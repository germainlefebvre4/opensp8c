## Why

L'interface actuelle est construite entièrement en inline styles ad hoc, sans design system ni cohérence visuelle. Les colonnes Kanban sont trop à l'étroit sur écran standard, la vue Specs manque d'ergonomie pour la lecture longue, et les composants UI (boutons, sidebar) ont une apparence brute non professionnelle.

## What Changes

- Ajout de Tailwind CSS (via `@tailwindcss/vite`) et Radix UI comme fondations UI
- Ajout de `lucide-react` pour l'iconographie
- Remplacement de tous les inline styles par des classes Tailwind cohérentes
- Layout principal : sidebar collapsible + toggle page/fullpage (masque la sidebar)
- Sidebar workspace : bouton "Ajouter un projet" moderne avec icône et form intégrée
- Kanban : colonnes s'étalant en pleine largeur en mode fullpage, design de cards amélioré
- Specs : texte aligné à gauche, Table des Matières sticky à droite, styles prose Markdown (`@tailwindcss/typography`)
- Correction du chevauchement des grandes polices en typographie Markdown

## Capabilities

### New Capabilities

- `ui-layout-modes`: Toggle page/fullpage qui masque la sidebar et étend le contenu sur toute la largeur

### Modified Capabilities

- `specs-view`: Ajout d'une Table des Matières sticky et scroll-aware à droite du contenu, alignement texte gauche, styles prose Markdown

## Impact

- Nouvelles dépendances npm : `tailwindcss`, `@tailwindcss/vite`, `@tailwindcss/typography`, `@radix-ui/react-scroll-area`, `@radix-ui/react-tooltip`, `@radix-ui/react-separator`, `lucide-react`
- `vite.config.ts` : ajout du plugin Tailwind
- `index.css` : remplacement du contenu par les directives Tailwind
- Tous les composants frontend refactorisés (Layout, WorkspaceSidebar, KanbanPage, KanbanColumn, ChangeCard, SpecsPage)
- Aucune modification backend, aucun changement d'API
