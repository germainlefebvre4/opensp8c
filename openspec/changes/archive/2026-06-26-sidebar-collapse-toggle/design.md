## Context

La `WorkspaceSidebar` est actuellement affichée ou masquée brutalement via un state `isFullpage` dans `Layout.tsx`. Le bouton de contrôle (`Maximize2`/`Minimize2`) est dans la nav principale, ce qui est sémantiquement incorrect — il n'agit pas sur la page entière mais uniquement sur la sidebar.

Deux composants concernés :
- `Layout.tsx` : détient le state, rend conditionnellement la sidebar et affiche le bouton fullscreen
- `WorkspaceSidebar.tsx` : sidebar pure, n'a pas connaissance de son état ouvert/fermé

## Goals / Non-Goals

**Goals:**
- Sidebar rétractable (pas de disparition brutale) avec transition CSS animée
- Bouton toggle dans le header de la sidebar, visible dans les deux états
- État collapsed réduit à `w-8` avec uniquement l'icône `▶`
- Supprimer le bouton `Maximize2` de la nav

**Non-Goals:**
- Persistance de l'état collapsed entre sessions (pas de localStorage)
- Raccourci clavier
- Sidebar redimensionnable par drag

## Decisions

### State ownership : Layout.tsx

Le state `isSidebarOpen` reste dans `Layout.tsx` et est passé en prop à `WorkspaceSidebar`. Alternatives envisagées :
- State dans `WorkspaceSidebar` → rejété car `Layout` contrôle la mise en page globale et pourrait avoir besoin de l'état (ex: adapter la nav)
- Context React → over-engineering pour un state binaire simple

### Rétraction plutôt que disparition

La sidebar passe de `w-56` à `w-8` au lieu de `display: none`. Ça évite un re-mount du composant (conservation du scroll, des inputs en cours) et permet l'animation.

### Contenu masqué par `overflow-hidden` + `opacity`

En état collapsed, le contenu interne est masqué via `opacity-0` et `pointer-events-none` combinés au `overflow-hidden` de l'aside. Pas de rendu conditionnel, pas de démontage des hooks internes.

### Icône : `ChevronLeft`/`ChevronRight` (lucide-react)

Plus sémantique que `Maximize2`/`Minimize2` pour un toggle de panneau latéral. Déjà disponible dans lucide-react.

## Risks / Trade-offs

- [Animation] Le `transition-all` peut être coûteux sur de vieux appareils → Mitigation : `transition-[width,opacity]` ciblé
- [Accessibilité] Le bouton doit avoir un `aria-label` explicite selon l'état
- [Layout shift] Le contenu principal s'élargit lors du collapse → comportement attendu et souhaité, pas de mitigation nécessaire
