# Design

## Context

Aujourd'hui, `Layout.tsx` rend un `WorkspaceSidebar` pleine hauteur (Configuration, `AgentSelector`, liste des projets) à gauche, et à droite une `<nav>` de 44 px (marque « OpenSpec » + 5 onglets) au-dessus du contenu. La `<nav>` ne dépend pas de la route : elle reste affichée sur `/configuration`. Le workspace actif vient du query param `?workspace=`, réinitialisé par un `useEffect` sauf sur `/configuration`. Les routes (`/`, `/specs`, `/timeline`, `/agents`, `/settings`, `/configuration`) sont déclarées dans `App.tsx` et ne changent pas. Voir `proposal.md` pour la motivation.

Cible :

```
+--------------------------------------------------------------------+
| OpenSpec | Projects  Configuration                 [Agent: Claude v]|  TopBar
+------------------+-------------------------------------------------+
| Sidebar          | Kanban Specs Timeline Agents Settings           |  WorkspaceTabs
| (projets)        +-------------------------------------------------+
|                  | children(workspaceId)                           |
+------------------+-------------------------------------------------+

/configuration : TopBar + children en pleine largeur (pas de Sidebar ni de WorkspaceTabs)
```

## Goals / Non-Goals

**Goals:**
- Séparer visuellement le global (TopBar, Sidebar) du spécifique au workspace (WorkspaceTabs).
- Conserver les routes, le query param `workspace` et le contenu des pages.
- Rendre Configuration accessible en permanence depuis la TopBar.

**Non-Goals:**
- Surcharge de Configuration par workspace dans Settings (change ultérieur).
- Renommage technique « workspace » → « project », changement de routes ou de l'API.
- Refonte visuelle des pages elles-mêmes, ou de `ConfigurationPage` en interne.
- Persistance de l'état de la sidebar ou de la dernière route entre sessions.

## Decisions

**1. Trois composants dédiés, orchestrés par `Layout`.** `TopBar` (nouveau), `WorkspaceTabs` (les onglets extraits de la `<nav>` actuelle) et `WorkspaceSidebar` (allégé). `Layout` reste seul propriétaire de la logique de workspace actif et d'URL, et choisit la structure selon `location.pathname === '/configuration'`. Alternative : une route imbriquée avec deux layouts (`WorkspaceLayout` / `ConfigLayout`) dans `App.tsx`. Écartée : la sélection du workspace effectif et la synchronisation de l'URL sont déjà dans `Layout` et servent aux deux zones ; scinder obligerait à dupliquer ou remonter cette logique pour un gain faible.

**2. Structure du DOM.** Colonne racine `h-screen flex-col overflow-hidden` : `TopBar` (`shrink-0`, h-11), puis une ligne `flex-1 flex overflow-hidden` contenant la sidebar (hors Configuration) et la zone principale. La zone principale contient `WorkspaceTabs` (hors Configuration) puis `children`. Sur Configuration, la zone principale n'a que `children`. La sidebar démarrant sous la TopBar, son en-tête « Projects » et son bouton de repli restent inchangés.

**3. `AgentSelector` déplacé tel quel, avec un ajustement de positionnement.** Le composant garde son `position: fixed` (nécessaire, `overflow-hidden` sur les ancêtres). Aujourd'hui `width = bouton` et `left = bouton.left` ; dans la TopBar le bouton est étroit et proche du bord droit. On passe à `min-width` (~12rem, `Math.max(bouton, min)`) avec un `left` calculé pour aligner le bord droit sur celui du bouton et borné à `>= 8px`. Le wrapper `px-2 pb-2` propre à la sidebar est retiré au profit d'un simple wrapper de barre. Le repli à la fermeture sur `resize` est conservé.

**4. Mémoire de la dernière route projets.** `Layout` (jamais démonté) garde un `useRef`/`useState` avec le dernier `pathname + search` hors `/configuration`, mis à jour à chaque navigation. L'entrée « Projects » de la TopBar pointe vers cette valeur, ou vers `/` (le `useEffect` existant ajoute alors `?workspace=<premier>`) si aucune. Si le workspace mémorisé n'existe plus, le calcul existant de `effectiveId` retombe sur `workspaces[0]`, ce qui couvre le scénario « workspace supprimé ». L'état reste en mémoire : un rechargement sur `/configuration` retombe sur le Kanban, ce que la spec accepte. Alternative : `sessionStorage`. Écartée : complexité pour un besoin marginal, et le comportement du rechargement sur `/configuration` est déjà défini.

**5. Marquage actif de la TopBar.** « Configuration » est actif si la route est `/configuration`, « Projects » sinon, calculés depuis `Layout` plutôt que via `NavLink` pour « Projects », qui n'a pas de route propre.

**6. Sidebar allégée.** Retrait du `NavLink` Configuration, de `AgentSelector` et de la prop `isConfigurationActive`. Le bloc visible en mode réduit ne contient plus que le toggle : la structure `pointer-events-none` / `opacity-0` du contenu est conservée. `WorkspaceSidebar.test.tsx` (qui teste l'entrée Configuration) est réécrit pour vérifier son absence et la présence du reste.

**7. i18n.** Ajout de `projects` dans `navigation.json` (en : « Projects », fr : « Projets »), utilisé par la TopBar. Le libellé de la sidebar continue d'utiliser `workspace.projects`. Le vocabulaire du code reste « workspace ».

## Risks / Trade-offs

- [Pages qui supposent la hauteur `h-screen` de la zone de droite (Kanban, Explore, DetailPanel, panneaux `fixed`)] → la hauteur disponible baisse de 44 px (TopBar) puis de 44 px (onglets) au lieu d'une seule barre : vérifier visuellement chaque onglet ; les pages utilisent `flex-1`/`h-full` et non des hauteurs en `vh`, donc le risque est faible.
- [Le panneau Explore maximisé se réfère à « l'espace sous la barre de navigation globale »] → il occupe déjà l'espace disponible de la zone de contenu ; la sémantique reste correcte, à vérifier manuellement.
- [Le dropdown de l'agent près du bord droit peut déborder] → calcul du `left` borné à la fenêtre et alignement droit sur le bouton.
- [Perte de repères au passage sur Configuration (la sidebar disparaît)] → assumé (option A retenue) ; « Projects » ramène à la dernière page consultée.
- [Le contenu de Settings est encore global alors que l'onglet est décrit comme propre au workspace] → assumé et documenté ; la surcharge par workspace est un change ultérieur, sans effet sur cette structure.
- [`ui-layout-modes` décrit encore un toggle page/fullpage déjà retiré par `sidebar-collapse`] → incohérence préexistante, hors périmètre.

## Migration Plan

Modification purement frontend, sans données ni API. Déploiement direct ; retour arrière par revert du commit. Les URLs existantes (`/?workspace=…`, `/specs?workspace=…`, `/configuration`) restent valides.
