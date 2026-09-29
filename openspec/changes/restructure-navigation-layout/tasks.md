# Tasks

## 1. i18n et composants de navigation

- [ ] 1.1 Ajouter la clé `projects` (« Projects » / « Projets ») dans `frontend/src/locales/{en,fr}/navigation.json` et vérifier que les deux fichiers ont les mêmes clés (`npm run test` dans `frontend`)
- [ ] 1.2 Extraire les onglets Kanban/Specs/Timeline/Agents/Settings de `Layout.tsx` vers un composant `WorkspaceTabs` (conservation du `search` de l'URL, onglet actif en évidence) et vérifier par un test `renderToStaticMarkup` que les 5 liens sont rendus dans l'ordre avec le paramètre `workspace`
- [ ] 1.3 Créer le composant `TopBar` (marque « OpenSpec », entrées « Projects » et « Configuration », `AgentSelector` aligné à droite, entrée active mise en évidence via une prop) et vérifier par test que les éléments sont présents et que l'état actif suit la prop

## 2. AgentSelector et Sidebar

- [ ] 2.1 Adapter `AgentSelector.tsx` : retirer le wrapper `px-2 pb-2` propre à la sidebar, largeur minimale du dropdown, bord droit aligné sur celui du bouton, `left` borné à la fenêtre ; vérifier par test (ou inspection du style calculé) que la largeur est `max(bouton, minimum)` et que le dropdown ne déborde pas à droite
- [ ] 2.2 Retirer de `WorkspaceSidebar.tsx` le lien Configuration, l'`AgentSelector` et la prop `isConfigurationActive` ; vérifier que `npm run build` (tsc) ne signale aucune référence restante
- [ ] 2.3 Réécrire `WorkspaceSidebar.test.tsx` pour vérifier l'absence de lien `/configuration` et d'`agent-selector`, et la présence de la liste, du bouton d'ajout et du toggle ; vérifier que `npm run test` passe

## 3. Layout

- [ ] 3.1 Restructurer `Layout.tsx` : colonne racine `TopBar` puis ligne (sidebar + zone principale avec `WorkspaceTabs` puis `children`), sidebar et onglets masqués sur `/configuration` où seul `children` occupe toute la largeur ; vérifier manuellement dans `npm run dev` les 6 routes et l'absence de scroll parasite de la page
- [ ] 3.2 Mémoriser dans `Layout` la dernière route hors Configuration (pathname + search) et l'utiliser comme cible de « Projects », avec repli sur `/` ; vérifier manuellement : Timeline du workspace B → Configuration → Projects ramène à Timeline de B, et un chargement direct sur `/configuration` puis « Projects » ouvre le Kanban du premier workspace
- [ ] 3.3 Vérifier manuellement les cas limites : aucun workspace (TopBar, sous-menu et invitation d'ajout visibles, Configuration accessible), suppression du workspace mémorisé (retour sur le premier restant), sidebar repliée puis ouverte

## 4. Vérifications d'intégration

- [ ] 4.1 Parcourir chaque onglet (Kanban, Specs, Timeline, Agents, Settings) et le panneau Explore maximisé pour confirmer qu'aucun contenu n'est coupé avec les 2 barres empilées (44 px + 44 px), et que le dropdown de l'agent s'ouvre entièrement visible depuis la TopBar
- [ ] 4.2 Lancer `npm run lint`, `npm run test` et `npm run build` dans `frontend` et confirmer qu'ils passent sans erreur
- [ ] 4.3 Valider les specs avec `openspec validate restructure-navigation-layout --strict` et confirmer l'absence d'erreur
