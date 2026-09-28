# Tasks

## 1. Structure de l'app toujours montée

- [ ] 1.1 Dans `frontend/src/App.tsx`, retirer le branchement `if (workspaces.length === 0) return <WorkspaceSetup />` : `<Layout>` monte toujours dès que `isLoading` est faux. Vérifier avec `npm run build` (compilation TypeScript) que le composant `WorkspaceSetup` n'est plus importé sans usage, ou qu'il est réutilisé par la tâche 1.2.
- [ ] 1.2 Créer un composant `NoWorkspaceState` (repris du contenu actuel de `WorkspaceSetup.tsx`, adapté à un conteneur de contenu plutôt qu'à un centrage plein écran) et l'utiliser comme fallback (`workspaceId ? <Page/> : <NoWorkspaceState/>`) pour les routes `/`, `/specs`, `/timeline` dans `App.tsx`. Vérifier manuellement (`npm run dev`, aucun workspace configuré) que ces trois routes affichent l'invitation dans la zone de contenu, nav et sidebar restant visibles.
- [ ] 1.3 Ajouter la route `<Route path="/configuration" element={<ConfigurationPage />} />` dans `App.tsx` (la page elle-même est créée en tâche 2). Vérifier que `npm run build` compile sans erreur une fois `ConfigurationPage` créée.

## 2. Page Configuration

- [ ] 2.1 Créer `frontend/src/pages/ConfigurationPage.tsx` avec un état interne `useState<'agents' | 'cli'>('agents')` pilotant deux sous-onglets, suivant le pattern de `SettingsPage.tsx`. Vérifier que la page s'affiche sans erreur console sur `/configuration`.
- [ ] 2.2 Sous-onglet Agents : tableau en lecture seule listant `id/label/installed/version` via le hook `useAgents()` existant (`frontend/src/hooks/useAgentPreferences.ts`), sans contrôle d'édition. Écrire un test `frontend/src/pages/ConfigurationPage.test.tsx` couvrant l'affichage du statut installé/non installé et de la version, et vérifier `npm run test`.
- [ ] 2.3 Sous-onglet CLI : déplacer le JSX de `AgentSettingsModal.tsx` (variables recommandées `GOOGLE_CLOUD_PROJECT`/`GEMINI_MODEL`/`GEMINI_SANDBOX` avec placeholder système et mention override, variables personnalisées ajout/suppression, toggle mode question native) dans ce sous-onglet, branché sur `usePreferences`/`usePatchPreferences`. Étendre `ConfigurationPage.test.tsx` pour couvrir : affichage du placeholder système, bascule override/valeur système, ajout/suppression d'une variable personnalisée, activation du toggle mode question native. Vérifier `npm run test`.
- [ ] 2.4 Ajouter les clés de traduction FR/EN pour Configuration (`frontend/src/locales/fr/configuration.json`, `frontend/src/locales/en/configuration.json`, plus la clé `configuration` dans `navigation.json` des deux locales), en reprenant le texte existant de `dialogs.json` (`agentSettings.*`) et `settings.json` là où le contenu est repris tel quel. Vérifier qu'aucune clé de traduction manquante n'apparaît dans la console (mode dev) en naviguant sur `/configuration` dans les deux langues.

## 3. Navigation et nettoyage du sélecteur d'agent

- [ ] 3.1 Dans `frontend/src/components/Layout.tsx`, ajouter un bouton/`NavLink` "Configuration" séparé de la boucle des tabs existants (pas de `search: searchParams.toString()`), positionné avant `LanguageSwitcher`, stylé distinctement (pilule bordée). Vérifier manuellement que l'URL de `/configuration` ne porte jamais `?workspace=`.
- [ ] 3.2 Dans `frontend/src/components/AgentSelector.tsx`, retirer le bouton engrenage, l'état `showSettings` et l'import/usage de `AgentSettingsModal`. Vérifier `npm run build` et `npm run test` (adapter le test existant du composant s'il en dépend).
- [ ] 3.3 Supprimer `frontend/src/components/AgentSettingsModal.tsx` (contenu repris en tâche 2.3). Vérifier qu'aucune référence résiduelle ne subsiste : `grep -r "AgentSettingsModal" frontend/src` ne retourne rien.

## 4. Vérification d'intégration

- [ ] 4.1 Test manuel de bout en bout (`npm run dev`, backend démarré) : avec un `config.yaml` sans workspace, vérifier que Configuration, Agents (pool) et Réglages restent accessibles et fonctionnels, que Kanban/Specs/Timeline affichent `NoWorkspaceState`, puis qu'ajouter un premier projet fait réapparaître normalement le Kanban sans perte d'état sur la page Configuration si elle reste ouverte dans un autre onglet du navigateur.
- [ ] 4.2 Lancer `npm run lint` et `npm run build` sur l'ensemble du frontend pour confirmer l'absence de régression suite à la suppression de `AgentSettingsModal` et à la restructuration de `App.tsx`/`Layout.tsx`.
