# Design

## Context

Voir `proposal.md` - Why. Points d'implémentation pertinents observés dans le code actuel :

- `App.tsx` : si `workspaces.length === 0`, l'app rend `<WorkspaceSetup />` seul, sans jamais monter `<Layout>` — donc sans nav ni sidebar. Sinon, `<Layout>` monte un `<Routes>` avec `/`, `/specs`, `/timeline` (qui font `workspaceId ? <Page/> : null`), `/agents` et `/settings` (qui ne dépendent pas de `workspaceId`).
- `Layout.tsx` construit sa barre de nav en mappant un tableau `{path, label}` sur des `NavLink`, chacun avec `search: searchParams.toString()` pour préserver `?workspace=`.
- `WorkspaceSidebar.tsx` gère déjà une liste de projets vide sans erreur (boucle `.map` sur `[]`, bouton "Ajouter un projet" toujours affiché) — aucune modification nécessaire côté sidebar elle-même.
- `AgentSelector.tsx` ouvre `AgentSettingsModal` via une icône engrenage à côté du sélecteur d'agent.
- Les hooks `useAgentPreferences.ts` (`usePreferences`, `usePatchPreferences`, `useAgents`, `useAgentSpecializations`) exposent déjà tout ce dont Configuration a besoin ; aucun nouvel endpoint backend n'est requis.

## Goals / Non-Goals

**Goals:**
- Rendre `Layout` (nav + sidebar) toujours monté, y compris quand `workspaces.length === 0`.
- Isoler l'entrée "Configuration" du tableau de tabs existant pour qu'elle ne porte jamais `?workspace=`.
- Regrouper dans une seule page à onglets internes (`Agents` / `CLI`) le contenu aujourd'hui dispersé entre le dropdown sidebar (registre) et `AgentSettingsModal` (variables + mode question native).

**Non-Goals:**
- Ne touche pas à l'état de chargement initial (`isLoading`) : le court instant de chargement des workspaces continue de ne rien afficher.
- Ne modifie aucun endpoint backend ni le schéma de `preferences.json`.
- Ne change pas le contenu ni le comportement de `/agents` (pool), `/settings` (spécialisations), ni du sélecteur d'agent par défaut dans la sidebar — seule leur *accessibilité* sans workspace change, pas leur logique interne.

## Decisions

### 1. `App.tsx` : retirer le branchement plein-écran, garder `Layout` toujours monté
Au lieu de `if (workspaces.length === 0) return <WorkspaceSetup />`, `App.tsx` rend toujours `<Layout>{workspaceId => <Routes>...}</Routes>}</Layout>` dès que `isLoading` est faux. Pour les routes qui dépendent d'un workspace (`/`, `/specs`, `/timeline`), le fallback `workspaceId ? <Page/> : null` devient `workspaceId ? <Page/> : <NoWorkspaceState/>`, un petit composant partagé qui reprend le contenu du formulaire actuel de `WorkspaceSetup` (chemin + nom + bouton), simplement rendu dans la zone de contenu plutôt qu'en plein écran.
**Alternative envisagée** : garder deux arbres de rendu séparés (un "app vide" minimal dupliquant juste une barre avec le bouton Configuration, un "app pleine" avec `Layout` complet). Rejetée : duplique la nav, risque de divergence future, complexifie `Layout` sans bénéfice.

### 2. Bouton Configuration hors du tableau de tabs
`Layout.tsx` ajoute un élément séparé après la boucle de tabs (avant `LanguageSwitcher`) : un `NavLink to="/configuration"` qui n'inclut PAS `search: searchParams.toString()`, stylé différemment (pilule avec bordure plutôt que soulignement d'onglet) pour marquer visuellement qu'il n'est pas un onglet lié au workspace.

### 3. Une seule page `ConfigurationPage` avec état d'onglet interne
Suit le pattern déjà utilisé par `SettingsPage` (pas de sous-routes) : un `useState<'agents' | 'cli'>` local pilote l'affichage. Route unique `/configuration`.
**Alternative envisagée** : sous-routes `/configuration/agents` et `/configuration/cli`. Rejetée pour cohérence avec `SettingsPage` existante et parce qu'aucun besoin de lien profond n'a été exprimé.

### 4. Suppression de `AgentSettingsModal` et de son icône dans `AgentSelector`
Le contenu JSX (variables recommandées, variables personnalisées, toggle mode question native) est déplacé tel quel dans le sous-onglet CLI de `ConfigurationPage`, branché sur les mêmes hooks (`usePreferences`, `usePatchPreferences`). `AgentSelector.tsx` perd son bouton engrenage et l'état `showSettings` associé ; il ne garde que le dropdown de sélection d'agent par défaut.

### 5. Sous-onglet Agents : nouvelle vue en lecture seule, pas de nouveau hook
Réutilise `useAgents()` (déjà utilisé par le dropdown) pour lister `id/label/installed/version`, rendu en tableau simple, sans action.

## Risks / Trade-offs

- [Rendre `Layout` toujours monté change le comportement existant de `/agents` et `/settings`, qui étaient déjà (silencieusement) inaccessibles sans workspace] → Effet de bord positif et voulu (cf. delta `workspace-management`), mais à vérifier explicitement en test manuel pour éviter une régression sur ces deux pages en `workspaces.length === 0`.
- [Le formulaire `WorkspaceSetup` a été conçu pour un centrage plein écran ; réutilisé dans la zone de contenu, son style doit être adapté à un conteneur plus étroit] → Ajustement de style mineur, à traiter dans les tasks d'implémentation, pas un risque d'architecture.
- [Deux emplacements historiques du statut d'installation des agents (dropdown sidebar existant, nouveau registre Configuration) doivent rester cohérents] → Les deux consomment le même hook `useAgents()` / `GET /api/agents`, donc pas de double source de vérité.
