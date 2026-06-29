## 1. Backend — ConversationStore

- [x] 1.1 Créer `backend/internal/conversation/store.go` : struct `Store` avec méthodes `Append(wsID, changeName, kind, line []byte) error` et `List(wsID, changeName, kind) ([]RunMeta, error)` et `Load(wsID, changeName, kind, ts string) ([][]byte, error)`
- [x] 1.2 Initialiser le `ConversationStore` dans `router.go` avec le chemin `<config-dir>/conversations/` (dérivé comme `preferencesPath`)
- [x] 1.3 Écrire les tests unitaires de `store.go` : append, list (tri antéchronologique), load, fichier partiel

## 2. Backend — ff handler

- [x] 2.1 Créer `backend/internal/api/handlers/ff.go` avec `FFHandler` (POST /ff, GET /conversations/{kind}, GET /conversations/{kind}/{ts})
- [x] 2.2 Implémenter le guard "ff déjà actif" : vérifier la clé `wsID + "/__ff__/" + changeName` dans le session manager avant de spawner
- [x] 2.3 Spawner le subprocess ff avec `StartSubprocess`, injecter `/opsx:ff` comme premier message stdin
- [x] 2.4 Démarrer la goroutine fanOut ff : écrire chaque ligne stdout dans `ConversationStore.Append` et émettre `ff_started` / `ff_done` / `ff_failed` via le watcher SSE broadcast
- [x] 2.5 Auto-nettoyage : à la fermeture de `sess.Done()`, retirer la session `__ff__` du manager sans attendre le timeout inactivité
- [x] 2.6 Implémenter `GET /conversations/{kind}` : appelle `ConversationStore.List`, retourne `[]RunMeta` JSON
- [x] 2.7 Implémenter `GET /conversations/{kind}/{ts}` : appelle `ConversationStore.Load`, retourne `{ts, messages}` JSON ou 404

## 3. Backend — tasks reset handler

- [x] 3.1 Créer l'endpoint `PATCH /api/workspaces/{id}/changes/{name}/tasks/reset` dans un handler dédié ou dans `kanban.go`
- [x] 3.2 Implémenter le guard : retourner 409 si une session `__ff__` est active pour ce changement
- [x] 3.3 Vider `tasks.md` (os.WriteFile avec contenu vide) et retourner 204

## 4. Backend — SSE events ff + routing

- [x] 4.1 Étendre le watcher/events pour supporter les events `ff_started`, `ff_done`, `ff_failed` avec champ `name`
- [x] 4.2 Enregistrer les 4 nouvelles routes dans `router.go` : POST /ff, GET /conversations/{kind}, GET /conversations/{kind}/{ts}, PATCH /tasks/reset

## 5. Frontend — DnD setup

- [x] 5.1 Ajouter `@dnd-kit/core` et `@dnd-kit/sortable` aux dépendances (`package.json`)
- [x] 5.2 Wrapper `KanbanPage` avec `<DndContext>` configuré avec les sensors (PointerSensor)
- [x] 5.3 Implémenter la logique `onDragEnd` dans `KanbanPage` : extraire source/destination et router vers le bon handler (ff ou reset)

## 6. Frontend — colonnes droppables

- [x] 6.1 Rendre `KanbanColumn` droppable via `useDroppable` de @dnd-kit pour les colonnes `to-explore` et `todo`
- [x] 6.2 Afficher le highlight visuel sur la colonne cible quand `isOver && isValidTarget`
- [x] 6.3 Ne pas afficher de highlight sur les colonnes `in-progress`, `done`, `archived` comme cibles (logique de validation dans `onDragEnd`)

## 7. Frontend — cartes draggables

- [x] 7.1 Rendre `ChangeCard` draggable via `useDraggable` de @dnd-kit pour les statuts `to-explore`, `todo`, `in-progress`
- [x] 7.2 Désactiver le drag sur les cartes en statut `done` et `archived`
- [x] 7.3 Désactiver le drag sur les cartes ayant un état ff actif (spinner) ou en erreur

## 8. Frontend — état ff sur les cartes

- [x] 8.1 Créer le hook `useFfState` qui consomme les events SSE `ff_started` / `ff_done` / `ff_failed` et maintient un map `changeName → "running" | "failed" | null`
- [x] 8.2 Passer l'état ff à `ChangeCard` et afficher le spinner si `"running"`, l'indicateur d'erreur si `"failed"`
- [x] 8.3 Déclencher `POST /changes/{name}/ff` dans `onDragEnd` (après fermeture éventuelle de l'ExplorePanel)

## 9. Frontend — dialog de confirmation reset

- [x] 9.1 Créer le composant `ResetTasksDialog` : message adapté selon `tasks_done > 0` (avertissement) ou `tasks_done === 0` (neutre)
- [x] 9.2 À la confirmation, appeler `PATCH /changes/{name}/tasks/reset` et invalider la query changes
- [x] 9.3 À l'annulation, ne rien faire (carte retourne à sa position)

## 10. Frontend — fermeture ExplorePanel avant ff

- [x] 10.1 Dans `onDragEnd` (to-explore → todo), vérifier si l'ExplorePanel est ouvert pour ce changement (`exploreOpen?.name === changeName`)
- [x] 10.2 Si ouvert : appeler `DELETE /changes/{name}/explore` puis, à la réponse, déclencher `POST /changes/{name}/ff`
- [x] 10.3 Si non ouvert : déclencher `POST /changes/{name}/ff` directement

## 11. Frontend — onglet Log dans DetailPanel

- [x] 11.1 Créer le hook `useConversationRuns(workspaceId, changeName, kind)` qui appelle `GET /changes/{name}/conversations/{kind}`
- [x] 11.2 Créer le hook `useConversationRun(workspaceId, changeName, kind, ts)` qui appelle `GET /changes/{name}/conversations/{kind}/{ts}`
- [x] 11.3 Ajouter l'onglet "Log" dans `DetailPanel` : liste des runs ff (timestamps), sélection du run courant (dernier par défaut)
- [x] 11.4 Afficher les messages du run sélectionné en lecture seule via le renderer de messages existant (réutilisation du rendu ExplorePanel sans zone de saisie)
