## Context

L'interface actuelle est entièrement construite en inline styles React, sans design system. Cela produit une incohérence visuelle, rend la maintenance difficile et donne un aspect non professionnel. Les colonnes Kanban (4 × min 220px) se retrouvent à l'étroit dans un layout avec sidebar fixe. La vue Specs affiche du Markdown sans styles typographiques, causant des chevauchements sur les grands titres.

L'utilisateur a choisi **Tailwind CSS + Radix UI** comme fondation, avec `lucide-react` pour les icônes.

## Goals / Non-Goals

**Goals:**
- Adopter Tailwind CSS comme seul système de style (supprimer tous les inline styles)
- Intégrer les primitives Radix UI (ScrollArea, Tooltip, Separator) pour l'accessibilité et les comportements complexes
- Implémenter un toggle page/fullpage qui masque la sidebar et étend le contenu
- Moderniser le WorkspaceSidebar (bouton "Ajouter" avec icône, form inline élégante)
- Améliorer la vue Specs : alignement gauche, TOC sticky scroll-aware, prose Markdown
- Corriger la typographie Markdown via `@tailwindcss/typography`

**Non-Goals:**
- Dark mode (séparé)
- Drag & drop sur le Kanban
- Modification de l'API backend
- Internationalisation

## Decisions

### 1. Tailwind via `@tailwindcss/vite` (plugin natif Vite)

Plutôt que la CLI ou PostCSS, on utilise le plugin Vite officiel. Plus simple à configurer, pas de `tailwind.config.js` requis pour un usage de base.

**Alternative rejetée** : PostCSS — ajoute de la complexité sans bénéfice ici.

### 2. `@tailwindcss/typography` pour la prose Markdown

La classe `prose` de ce plugin normalise tous les styles h1–h6, p, li, code dans un container Markdown. Elle résout le chevauchement des grandes polices sans écrire de CSS custom.

**Alternative rejetée** : CSS Modules custom — plus de travail, résultat identique.

### 3. Fullpage mode via état React dans Layout

Un state `isFullpage` dans `Layout.tsx` contrôle la visibilité de la sidebar. Pas de route dédiée, pas de query param — le mode est éphémère (reset au reload), ce qui est le comportement attendu.

**Alternative rejetée** : Query param `?fullpage=1` — polluerait l'URL sans valeur persistée utile.

### 4. TOC généré côté client par parsing regex du Markdown

Le contenu Markdown est parsé avec un regex `/^#{1,3}\s+(.+)$/gm` avant le rendu ReactMarkdown. On génère les IDs de headings de manière déterministe (slug lowercase). ReactMarkdown reçoit un composant `h1/h2/h3` custom qui applique l'`id` correspondant.

L'IntersectionObserver observe chaque heading et met à jour l'état `activeId` dans le composant TOC.

**Alternative rejetée** : `remark-toc` / `rehype-slug` — dépendances supplémentaires pour un besoin couvrable en ~40 lignes de code.

### 5. Radix ScrollArea pour sidebar et contenu Specs

Remplace l'overflow CSS natif par `@radix-ui/react-scroll-area` pour une scrollbar stylisée et cross-browser cohérente.

### 6. Structure de fichiers inchangée

Aucun répertoire créé, aucun composant déplacé. On refactorise les fichiers existants in-place pour limiter le diff et les conflits potentiels.

## Risks / Trade-offs

- **Tailwind purge** : Les classes dynamiques (ex. couleurs par statut Kanban) doivent être listées explicitement ou générées de façon à ne pas être purgées → utiliser des maps d'objets plutôt que la concaténation de strings.
- **Radix ScrollArea vs overflow natif** : Le scroll natif fonctionne mieux avec certains trackpads ; ScrollArea est un wrapper qui peut masquer du contenu si mal configuré → à tester sur les deux modes.
- **TOC et ReactMarkdown** : Les composants custom passés à ReactMarkdown nécessitent de mapper chaque heading à son ID slug ; si le Markdown contient des emojis ou caractères spéciaux dans les titres, le slug doit être robuste.

## Migration Plan

1. Installer les dépendances npm
2. Configurer Tailwind dans `vite.config.ts` et `index.css`
3. Refactoriser composant par composant, en commençant par `Layout.tsx` (fondation)
4. Tester en dev (`npm run dev`) après chaque composant
5. Aucune migration de données ni de routes — rollback = `git revert`
