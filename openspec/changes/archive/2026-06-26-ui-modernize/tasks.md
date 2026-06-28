## 1. Setup des dépendances

- [x] 1.1 Installer `tailwindcss`, `@tailwindcss/vite` et `@tailwindcss/typography` dans `frontend/`
- [x] 1.2 Installer `@radix-ui/react-scroll-area`, `@radix-ui/react-tooltip`, `@radix-ui/react-separator` dans `frontend/`
- [x] 1.3 Installer `lucide-react` dans `frontend/`
- [x] 1.4 Configurer le plugin Tailwind dans `frontend/vite.config.ts`
- [x] 1.5 Remplacer le contenu de `frontend/src/index.css` par les directives Tailwind (`@import "tailwindcss"`) et supprimer `App.css`

## 2. Layout principal

- [x] 2.1 Refactoriser `Layout.tsx` : remplacer tous les inline styles par des classes Tailwind
- [x] 2.2 Ajouter l'état `isFullpage` dans `Layout.tsx` avec toggle dans la nav
- [x] 2.3 Conditionner l'affichage de la sidebar selon `isFullpage`
- [x] 2.4 Ajouter le bouton icône Lucide (ex. `Maximize2` / `Minimize2`) pour le toggle dans la barre de navigation

## 3. WorkspaceSidebar

- [x] 3.1 Refactoriser `WorkspaceSidebar.tsx` avec classes Tailwind (fond, border, items)
- [x] 3.2 Remplacer le bouton "Ajouter un projet" par un bouton moderne avec icône `PlusCircle` de lucide-react
- [x] 3.3 Styliser le formulaire d'ajout inline (input, boutons Ajouter/Annuler) avec Tailwind
- [x] 3.4 Intégrer `@radix-ui/react-scroll-area` pour la liste des workspaces

## 4. Kanban

- [x] 4.1 Refactoriser `KanbanColumn.tsx` avec classes Tailwind (fond gris, border-radius, gap)
- [x] 4.2 Attribuer une couleur d'accent par statut (badge, header) via une map de classes Tailwind
- [x] 4.3 Refactoriser `ChangeCard.tsx` avec classes Tailwind (card avec shadow, hover state)
- [x] 4.4 Vérifier que le Kanban s'étale correctement en mode fullpage (overflow-x: auto sur le container de colonnes)

## 5. Vue Specs

- [x] 5.1 Refactoriser `SpecsPage.tsx` : layout 3 colonnes (liste / contenu / TOC) avec Tailwind
- [x] 5.2 Aligner le texte à gauche dans le panneau de contenu (`text-left`)
- [x] 5.3 Ajouter la classe `prose` de `@tailwindcss/typography` sur le container ReactMarkdown
- [x] 5.4 Créer le composant `TableOfContents.tsx` : parse les headings h1/h2/h3 du Markdown avec regex
- [x] 5.5 Passer des composants h1/h2/h3 custom à ReactMarkdown pour injecter les `id` slugifiés
- [x] 5.6 Implémenter l'IntersectionObserver dans `TableOfContents.tsx` pour mettre en évidence la section active
- [x] 5.7 Masquer le panneau TOC si le contenu ne contient aucun heading
- [x] 5.8 Intégrer `@radix-ui/react-scroll-area` sur la liste des specs et le panneau de contenu

## 6. Vérification

- [x] 6.1 Vérifier le rendu Kanban en mode page et fullpage (colonnes visibles, scroll horizontal OK)
- [x] 6.2 Vérifier la vue Specs : TOC générée, clic navigue, highlight scroll-aware
- [x] 6.3 Vérifier qu'aucune grande police ne chevauche son container en typographie Markdown
- [x] 6.4 Vérifier l'accessibilité des composants Radix (focus keyboard, aria)
- [x] 6.5 Supprimer les imports de `App.css` et vérifier qu'aucun inline style ne subsiste (ExplorePanel/DetailPanel hors scope)
