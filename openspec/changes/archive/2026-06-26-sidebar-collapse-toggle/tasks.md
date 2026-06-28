## 1. Layout.tsx — Refactoring du state et de la nav

- [x] 1.1 Renommer `isFullpage` en `isSidebarOpen` (valeur initiale `true`)
- [x] 1.2 Supprimer l'import `Maximize2` / `Minimize2` et le bouton fullscreen de la nav
- [x] 1.3 Passer `isSidebarOpen` et `onToggle` en props à `WorkspaceSidebar` (supprimer le rendu conditionnel `{!isFullpage && ...}`)

## 2. WorkspaceSidebar.tsx — Toggle et animation

- [x] 2.1 Ajouter les props `isOpen: boolean` et `onToggle: () => void` à l'interface `Props`
- [x] 2.2 Remplacer l'import `FolderOpen` + ajouter `ChevronLeft` / `ChevronRight` depuis lucide-react
- [x] 2.3 Appliquer la classe de largeur dynamique sur `<aside>` : `w-56` quand ouvert, `w-8` quand collapsed, avec `transition-[width] duration-200 ease-in-out overflow-hidden`
- [x] 2.4 Ajouter le bouton toggle dans le header (à côté de "Projets") avec l'icône `ChevronLeft`/`ChevronRight` selon `isOpen`
- [x] 2.5 Masquer le contenu interne en état collapsed : envelopper liste + footer dans un `<div>` avec `opacity-0 pointer-events-none` quand `!isOpen`
- [x] 2.6 Ajouter `aria-label` au bouton toggle (`"Fermer le menu"` / `"Ouvrir le menu"`)
