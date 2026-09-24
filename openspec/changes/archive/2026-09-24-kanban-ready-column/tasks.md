# Tasks

## 1. Backend — état persistant "lancé"/"ordre"

- [x] 1.1 Dans `backend/internal/openspec/change.go`, ajouter `Launched *bool` (`yaml:"launched,omitempty"`) et `Order *int` (`yaml:"order,omitempty"`) à `openspecMeta`, et `Launched bool`/`Order int` (`json:"order,omitempty"`) à `Change`. Vérifier que `go build ./...` compile.
- [x] 1.2 Changer la signature de `deriveStatus(done, total int)` en `deriveStatus(done, total int, launched bool) string`, insérer la branche `!launched -> "ready"` entre `total == 0` et `done == 0`. Mettre à jour tous les appels dans `change.go` (`loadChange`). Vérifier avec un test unitaire `deriveStatus(0, 3, false) == "ready"` et `deriveStatus(0, 3, true) == "todo"`.
- [x] 1.3 Dans `loadChange`, calculer `effectiveLaunched := meta.Launched == nil || *meta.Launched` (règle de compatibilité ascendante) et le passer à `deriveStatus`. Vérifier avec un test unitaire : un change avec `tasks.md` non vide et un `.openspec.yaml` sans champ `launched` retourne `kanban_status: "todo"` (non-régression).
- [x] 1.4 Ajouter dans `change.go` deux fonctions `SetLaunched(changeRoot string, launched bool) error` et `ClearKanbanState(changeRoot string) error`, sur le modèle lecture/`yaml.Unmarshal`/mutation/`yaml.Marshal`/`os.WriteFile` de `TagChange` (`backend/internal/openspec/tagger.go:158-210`). `ClearKanbanState` met `Launched` et `Order` à `nil`. Vérifier avec un test d'aller-retour lecture/écriture sur un `.openspec.yaml` de fixture.
- [x] 1.5 Ajouter `ReorderReady(changesDir string, orderedNames []string) error` qui, pour chaque nom de la liste, lit son `.openspec.yaml`, lui assigne `Order` séquentiellement (1..N selon la position dans `orderedNames`), et réécrit le fichier. Vérifier avec un test sur 3 changes de fixture : l'ordre des `Order` après appel correspond à `orderedNames`.

## 2. Backend — routes API de promotion/rétrogradation/réordonnancement

- [x] 2.1 Ajouter `KanbanHandler.Launch` (`PATCH /workspaces/{id}/changes/{name}/launch`) : appelle `SetLaunched(changeRoot, true)`, retourne 204, 404 si le change n'existe pas. Vérifier avec un test HTTP.
- [x] 2.2 Ajouter `KanbanHandler.Unlaunch` (`PATCH /workspaces/{id}/changes/{name}/unlaunch`) : retourne 409 si `h.activeWorkerChanges(id)[name]` est vrai (réutiliser le helper existant `kanban.go:43`), sinon appelle `SetLaunched(changeRoot, false)` et retourne 204. Vérifier avec un test HTTP couvrant les deux cas.
- [x] 2.3 Ajouter `KanbanHandler.ReorderReady` (`PUT /workspaces/{id}/ready-order`) : décode `{ "order": ["change-a", ...] }`, appelle `openspec.ReorderReady`. Vérifier avec un test HTTP.
- [x] 2.4 Dans `FFHandler.ResetTasks` (`backend/internal/api/handlers/ff.go:155`), après avoir vidé `tasks.md`, appeler `openspec.ClearKanbanState` sur le dossier du change. Vérifier avec un test asserting que `.openspec.yaml` ne contient plus `launched` ni `order` après reset.
- [x] 2.5 Déclarer les nouvelles routes dans `backend/internal/api/router.go` à côté des routes `changes/{name}` existantes (près de la ligne `tasks/reset`, ~113) : `r.Patch(".../launch", kanbanHandler.Launch)`, `r.Patch(".../unlaunch", kanbanHandler.Unlaunch)`, et `r.Put("/workspaces/{id}/ready-order", kanbanHandler.ReorderReady)`. Vérifier que le serveur démarre et que les routes répondent (pas de 404 routeur).
- [x] 2.6 Faire écrire `launched: false` au moment de la création du change par un Fast-Forward réussi — aussi bien le déclenchement direct (`FFHandler.TriggerFF`) que la promotion de ghost card (`ExploreHandler.PromoteGhost` / `runPromoteFF`). Vérifié par lecture de code (appel `openspec.SetLaunched(changeDir, false)` juste après `proc.Wait()` réussi, avant le broadcast `ff_done`, dans les deux chemins) ; la vérification bout-en-bout avec un vrai FF est couverte par la tâche 9.1.

## 3. Backend — priorité dans le dispatcher de l'Agent Pool

- [x] 3.1 Dans `backend/internal/pool/scheduler.go`, trier la liste retournée par `GetRunnableChanges()` par `Order` croissant (tri stable, égalité départagée par `Name`) avant de la retourner. Vérifier avec un test dans `scheduler_test.go` : deux changes runnables sans dépendance commune, `Order` différents, la fonction retourne l'ordre attendu.

## 4. Backend — compteurs de workspace

- [x] 4.1 Ajouter `"ready": 0` à la map `counts` initiale dans `backend/internal/api/handlers/workspace.go` (lignes ~29-34). Vérifier via `GET /api/workspaces` que `task_counts` inclut `"ready"` même à 0.

## 5. Frontend — types et client API

- [x] 5.1 Dans `frontend/src/hooks/useChanges.ts`, ajouter `'ready'` à l'union `kanban_status` de `Change`, et ajouter `order?: number`. Vérifier que `tsc --noEmit` passe sans erreur.
- [x] 5.2 Dans `frontend/src/lib/api.ts`, ajouter `launchChange`, `unlaunchChange`, `reorderReady` sur le modèle de `resetTasks`/`triggerFF` (lignes 69-76). Vérifier par un appel manuel depuis la console du navigateur ou un test si la suite en couvre l'équivalent.

## 6. Frontend — colonne Ready et transitions de drag

- [x] 6.1 Dans `frontend/src/pages/KanbanPage.tsx`, insérer `{ title: t('columns.ready'), status: 'ready' }` dans `leadingColumns` entre `to-explore` et `todo`. Ajouter une entrée `'ready'` à `STATUS_STYLES` dans `frontend/src/components/KanbanColumn.tsx`. Vérifier visuellement dans le navigateur que la colonne Ready s'affiche entre To Explore et To Do.
- [x] 6.2 Mettre à jour `VALID_DROPS` dans `KanbanPage.tsx` selon la spec `kanban-drag-drop` : `'to-explore': ['ready']`, `'ready': ['to-explore', 'todo']`, `'todo': ['ready', 'in-progress']`. Mettre à jour `handleDragEnd` : le drop sur `'ready'` déclenche `triggerFF` (au lieu du drop sur `'todo'` aujourd'hui) ; le drop `ready -> todo` appelle `launchChange` ; le drop `todo -> ready` appelle `unlaunchChange` (capturer le 409 et afficher une erreur) ; tout drop sur `'to-explore'` continue d'ouvrir `resetDialog`. Vérifié manuellement dans l'app (backend+frontend de démo) en glissant des cartes à travers chaque transition (to-explore→ready déclenche le FF, ready↔todo appellent launch/unlaunch, {ready,todo,in-progress}→to-explore ouvrent le reset dialog) ; a aussi nécessité d'ajouter `'ready'` à `DRAGGABLE_STATUSES` (`ChangeCard.tsx`) et d'étendre la détection de ghost "brouillon" à la colonne Ready (`KanbanColumn.tsx`), sans quoi les cartes Ready n'étaient ni saisissables ni reconnues comme brouillon — conséquence directe du déplacement de la zone d'atterrissage du FF vers Ready, couverte par les specs `kanban-board`/`kanban-drag-drop` de ce même change.
- [x] 6.3 Mettre à jour la condition de promotion de ghost card dans `handleDragEnd` (actuellement `if (targetStatus === 'todo')`) pour cibler `'ready'`. Vérifié manuellement : le drag d'un ghost card nommé vers Ready ouvre `promoteDialog`.
- [x] 6.4 Mettre à jour les textes `promoteGhostDialog.body` dans `frontend/src/locales/fr/kanban.json` et `en/kanban.json` pour référencer la colonne Ready au lieu de "À faire"/"To Do". Vérifié visuellement le texte de la dialog de promotion.

## 7. Frontend — réordonnancement par drag-and-drop dans Ready

- [x] 7.1 Ajouter la dépendance `@dnd-kit/sortable` au `frontend/package.json`. Vérifier que `npm install` et `npm run build` réussissent.
- [x] 7.2 Dans la colonne Ready, envelopper la liste des cartes dans un `SortableContext` (via `@dnd-kit/sortable`), et appeler `reorderReady(workspaceId, newOrderedNames)` à la fin d'un drag interne, avec mise à jour optimiste de l'ordre local (`qc.setQueryData`) avant le prochain refetch. Vérifié manuellement (backend+frontend de démo) : réordonner 3+ cartes dans Ready persiste après rechargement complet de la page.
  - Écart assumé par rapport au libellé de la tâche : implémenté avec UN SEUL `DndContext` partagé (celui inter-colonnes existant) plutôt qu'un second `DndContext` indépendant. `ChangeCard` utilise désormais `useSortable` (au lieu de `useDraggable`) pour toutes les colonnes ; seule la colonne Ready l'enveloppe dans un vrai `SortableContext`, les autres colonnes reçoivent un tableau `items` vide donc `useSortable` s'y comporte comme un simple draggable (aucun effet de réordonnancement, cohérent avec le comportement existant). `handleDragEnd` distingue un drop sur une carte (réordonnancement intra-Ready) d'un drop sur une colonne (transition de statut). C'est le pattern officiel `@dnd-kit` pour combiner sortable-par-conteneur et drag inter-conteneurs sous un seul contexte ; deux `DndContext` réellement imbriqués auraient un risque de conflit de capture du pointeur entre sensors. Ajout de `collisionDetection={closestCenter}` sur le `DndContext` pour un ciblage plus fiable entre carte et colonne.

## 8. i18n

- [x] 8.1 Ajouter la clé `columns.ready` dans `frontend/src/locales/fr/kanban.json` et `frontend/src/locales/en/kanban.json`, cohérente avec le style des entrées `toDo`/`toExplore` existantes. Vérifié visuellement que l'en-tête de colonne s'affiche dans les deux locales.

## 9. Vérification manuelle de bout en bout

- [x] 9.1 Vérifié avec l'app réelle (backend+frontend lancés contre un workspace de démo isolé, pas le projet réel) : le drag d'un change `to-explore` vers `Ready` déclenche le FF (`POST /ff` 202) et la carte atterrit en `Ready` ; ajout d'un test d'intégration `TestManager_SkipsReadyAndPicksLowestOrder` (`backend/internal/pool/manager_ready_integration_test.go`) qui démarre un vrai `pool.Manager` sur un mélange de changes `ready`/`todo` (rangs `order` distincts) et confirme qu'un worker ne prend jamais un change `ready` et pioche toujours le `todo` de rang le plus bas en premier — couvre la portion backend/scheduling du scénario. N'a pas fait tourner un Agent Pool avec un vrai agent codant réellement les tâches (hors périmètre d'une vérification de composant, coûteux et non déterministe) ; le comportement UI/API ci-dessus et le dispatch du pool sont ce que ce projet peut vérifier de façon fiable et reproductible.
- [x] 9.2 Confirmé par lecture seule sur le vrai `openspec/changes/activity-conversation-timeline/.openspec.yaml` du projet (aucune modification) : il ne contient pas de champ `launched`. Un test ciblant `openspec.ListChanges` sur ce même dossier réel confirme `kanban_status: "todo"` (`launched: true` effectif par compatibilité ascendante, 0/30 tâches), donc le change n'apparaît jamais en `Ready`. Couvre aussi par le test unitaire générique `TestListChangesLaunchedAbsentIsBackwardCompatible` (tâche 1.3) et par la démo navigateur (le change `legacy-no-launched` de test s'affichait bien en `To Do`).
