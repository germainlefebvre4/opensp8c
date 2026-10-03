# Tasks

## 1. Composant SubTabs partagé

- [x] 1.1 Créer `frontend/src/components/SubTabs.tsx` (props `tabs`, `active`, `onChange`, `trailing?`, `aria-label`), rendu identique à la barre actuelle de SpecsPage/TimelinePage ; vérifier avec `SubTabs.test.tsx` (onglet actif marqué, `onChange` appelé au clic, `trailing` aligné à droite et non cliquable, absence de `trailing` sans zone vide)
- [x] 1.2 Remplacer le helper `subTabClass` et les boutons inline de `SpecsPage.tsx` et `TimelinePage.tsx` par `SubTabs`, sans toucher au reste de ces fichiers ; vérifier que `cd frontend && npm test` reste vert et que le rendu de la barre est inchangé à l'œil sur `/specs` et `/timeline`

## 2. Settings

- [x] 2.1 Dans `SettingsPage.tsx`, supprimer l'en-tête `<h1>` et la barre inline, utiliser `SubTabs` avec le nom du workspace en `trailing` (conserver `parseSettingsTab` et la logique `?tab=`) ; vérifier avec `SettingsPage.test.tsx` mis à jour : plus de titre « Réglages », nom du workspace visible en fin de barre, `tab=columns` toujours persisté, défaut `agent-pool` pour une valeur absente ou inconnue
- [x] 2.2 Nettoyer les imports devenus inutiles (`SettingsIcon`) et vérifier `cd frontend && npm run lint && npx tsc -b` sans erreur

## 3. Agents : sous-onglets Workers et Runs

- [x] 3.1 Ajouter `parseAgentsTab(tabParam, runParam)` (un `tab` valide l'emporte, sinon `run` valide ⇒ `runs`, sinon `workers`) avec un commentaire sur la disjonction des valeurs avec Settings ; vérifier avec un test unitaire couvrant `tab=runs`, `tab=workers`, `run` seul, `tab` inconnu avec et sans `run`
- [x] 3.2 Ajouter les clés `tabs.workers` et `tabs.runs` aux locales `agents` fr et en ; vérifier que `agents.i18n.test.ts` passe
- [x] 3.3 Refondre `AgentsPage.tsx` : retirer le `<h1>`, afficher `SubTabs` (Workers / Runs, `aria-label` issu de `title`), répartir résumé + table des workers sous Workers et la section `recent-runs` sous Runs, garder la rangée « colonne principale + `AgentRunPanel` » commune ; vérifier visuellement en lançant l'app (le panneau reste ouvert au changement de sous-onglet)
- [x] 3.4 Implémenter les écritures d'URL de la décision 3 du design (clic worker ⇒ `run` + `tab=workers`, clic run ⇒ `run` + `tab=runs`, sélecteur du panneau ⇒ `tab` inchangé, fermeture ⇒ `run` retiré, clic Workers ⇒ `tab` retiré sauf si `run` présent) ; vérifier avec des tests d'interaction dans `AgentsPage.interaction.test.tsx`
- [x] 3.5 Adapter `AgentsPage.test.tsx` et `AgentsPage.interaction.test.tsx` : les scénarios sur `run-row` et « Aucun run… » passent par `?tab=runs` ou un clic sur « Runs », ajouter les cas « `run=` seul ouvre Runs », « clic worker reste sur Workers », « `tab` inconnu ignoré » ; vérifier que `cd frontend && npm test` est entièrement vert

## 4. Vérification d'ensemble

- [x] 4.1 Lancer l'app et parcourir Specs, Timeline, Agents et Settings sur un workspace : barre de sous-onglets identique et sans titre sur les quatre pages, rechargement de `?tab=runs` et `?tab=columns` conservé ; vérifier aussi `cd frontend && npm run build` sans erreur
- [x] 4.2 Exécuter `openspec validate align-workspace-subtabs --strict` et confirmer que le résultat est valide
