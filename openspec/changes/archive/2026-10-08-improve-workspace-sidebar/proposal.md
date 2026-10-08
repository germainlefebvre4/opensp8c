# Proposal

## Why

La sidebar des projets n'affiche que des pastilles de comptage colorées, sans légende, sur une ligne unique où elles tronquent le nom du projet. Elle ne dit pas ce qui attend l'utilisateur (changes à revoir, workers en pause, tâches de validation humaine, vérification échouée), omet deux statuts du Kanban (`ready` et `verifying`, déjà renvoyés par l'API), et ses actions sont fragiles : la suppression d'un projet est un `X` visible au survol seulement, sans confirmation, et seul le nom du projet est cliquable, sans sémantique clavier ni ARIA.

## What Changes

- Chaque projet passe sur **deux lignes** : ligne 1 = nom + nombre de changes qui attendent une action de l'utilisateur ; ligne 2 = **barre segmentée** de répartition des changes par statut (7 segments : `to-explore`, `ready`, `todo`, `in-progress`, `verifying`, `to-review`, `done`) + total. Chaque segment a une infobulle (« 5 en cours »).
- Chaque projet est **dépliable** (chevron) : la vue dépliée liste, par change concerné, un titre et **une ligne par signal d'action**. Signaux retenus : change à revoir (`to-review`), worker en pause avec sa raison, tâche de validation humaine (HITL) en attente avec son texte, vérification échouée (`failed`). L'état déplié est gardé en mémoire de session (non persisté). Un clic sur un change ouvre ce change dans le Kanban avec son DetailPanel.
- L'API `GET /api/workspaces` expose un champ `attention` par workspace (liste de changes avec leurs signaux), calculé dans le même passage que `task_counts`, sans nouvel endpoint ni polling supplémentaire.
- Les couleurs de statut proviennent d'**une table unique partagée** entre `KanbanColumn` et la sidebar (`ready` = indigo, `verifying` = teal, déjà utilisés par le Kanban), à la place de `BADGE_COLORS` aujourd'hui dupliquée et dérivée.
- La sidebar s'élargit modérément (`w-64` → `w-72`, 288px), compatible avec le budget de `kanban-fit-width`, et son état replié devient un **rail** (initiales du projet + point d'attention) au lieu d'une bande vide.
- La ligne projet entière est cliquable, avec sémantique accessible (`role`, `aria-current`, navigation clavier).
- La suppression passe dans un **menu `…`** toujours visible et accessible au clavier, avec une **confirmation** rappelant que le dossier du projet n'est pas supprimé.
- Les erreurs d'ajout d'un projet (doublon, pas de dossier `openspec/`) sont affichées dans la sidebar.
- Correction de dérives de specs : `sidebar-collapse` (largeur ouverte `w-56`, réalité `w-64`) et `workspace-kanban-counts` (« `done` non affiché », alors que `sidebar-done-badge` l'affiche).

## Capabilities

### New Capabilities

- `workspace-attention-signals`: calcul et exposition, par workspace, des changes qui attendent une action de l'utilisateur et de leurs signaux (revue, pause avec raison, validation humaine, vérification échouée), et leur présentation dans la sidebar (compteur, vue dépliée, navigation vers le change).

### Modified Capabilities

- `app-navigation-layout`: la sidebar affiche la ligne projet à deux niveaux, la barre segmentée et le menu d'actions ; sa largeur ouverte passe à `w-72`.
- `sidebar-collapse`: la largeur ouverte est `w-72` ; l'état replié est un rail d'initiales avec point d'attention, au lieu d'une bande vide.
- `sidebar-done-badge`: la pastille `done` devient le segment `done` de la barre segmentée, avec la même couleur que le Kanban.
- `workspace-kanban-counts`: les badges inline deviennent la barre segmentée à 7 statuts (`ready` et `verifying` inclus) ; `done` est affiché ; `task_counts` documente `verifying`.
- `workspace-management`: la suppression passe par une confirmation depuis un menu d'actions ; les erreurs d'ajout sont affichées dans la sidebar ; la ligne projet est sélectionnable au clavier.

## Impact

- Backend : `backend/internal/api/handlers/workspace.go` (`List` : champ `attention`, utilise déjà `poolReg`), `backend/internal/workspace/workspace.go` (type `Workspace`), `backend/internal/openspec/change.go` (exposition des tâches HITL en attente), calcul des signaux dans un nouveau fichier dédié du paquet `openspec` ou `handlers`. Le backend a déjà des modifications locales non commitées sur ces fichiers.
- Frontend : `frontend/src/components/WorkspaceSidebar.tsx` (refonte), nouveau module de couleurs de statut partagé avec `frontend/src/components/KanbanColumn.tsx`, `frontend/src/hooks/useWorkspaces.ts` (type `Workspace`), `frontend/src/components/Layout.tsx` (largeur, état replié), `frontend/src/pages/KanbanPage.tsx` (ouverture d'un change depuis la sidebar), locales `workspace` en/fr.
- Interaction avec `kanban-fit-width` : la sidebar passe de 256px à 288px ; `kanbanLayout.ts` mesure le conteneur de la ligne (hors sidebar), donc le palier « tout déplié, panel 420 » reste atteint jusqu'à ~1904px de fenêtre. Pas de changement de `kanbanLayout.ts`.
- Pas de dépendance nouvelle ; pas de changement de `config.yaml`.
