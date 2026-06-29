## 1. Backend — ConversationStore

- [ ] 1.1 Créer `backend/internal/conversation/store.go` : struct `Store` avec méthodes `Append(wsID, changeName, kind, line []byte) error` et `List(wsID, changeName, kind) ([]RunMeta, error)` et `Load(wsID, changeName, kind, ts string) ([][]byte, error)`
- [ ] 1.2 Initialiser le `ConversationStore` dans `router.go` avec le chemin `<config-dir>/conversations/` (dérivé comme `preferencesPath`)
- [ ] 1.3 Écrire les tests unitaires de `store.go` : append, list (tri antéchronologique), load, fichier partiel

## 2. Backend — ff handler

- [ ] 2.1 Créer `backend/internal/api/handlers/ff.go` avec `FFHandler` (POST /ff, GET /conversations/{kind}, GET /conversations/{kind}/{ts})
- [ ] 2.2 Implémenter le guard "ff déjà actif" : vérifier la clé `wsID + "/__ff__/" + changeName` dans le session manager avant de spawner
- [ ] 2.3 Spawner le subprocess ff avec `StartSubprocess`, injecter `/opsx:ff` comme premier message stdin
- [ ] 2.4 Démarrer la goroutine fanOut ff : écrire chaque ligne stdout dans `ConversationStore.Append` et émettre `ff_started` / `ff_done` / `ff_failed` via le watcher SSE broadcast
- [ ] 2.5 Auto-nettoyage : à la fermeture de `sess.Done()`, retirer la session `__ff__` du manager sans attendre le timeout inactivité
- [ ] 2.6 Implémenter `GET /conversations/{kind}` : appelle `ConversationStore.List`, retourne `[]RunMeta` JSON
- [ ] 2.7 Implémenter `GET /conversations/{kind}/{ts}` : appelle `ConversationStore.Load`, retourne `{ts, messages}` JSON ou 404

## 3. Backend — tasks reset handler

- [ ] 3.1 Créer l'endpoint `PATCH /api/workspaces/{id}/changes/{name}/tasks/reset` dans un handler dédié ou dans `kanban.go`
- [ ] 3.2 Implémenter le guard : retourner 409 si une session `__ff__` est active pour ce changement
- [ ] 3.3 Vider `tasks.md` (os.WriteFile avec contenu vide) et retourner 204

## 4. Backend — SSE events ff + routing

- [ ] 4.1 Étendre le watcher/events pour supporter les events `ff_started`, `ff_done`, `ff_failed` avec champ `name`
- [ ] 4.2 Enregistrer les 4 nouvelles routes dans `router.go` : POST /ff, GET /conversations/{kind}, GET /conversations/{kind}/{ts}, PATCH /tasks/reset

## 5. Frontend — DnD setup

- [ ] 5.1 Ajouter `@dnd-kit/core` et `@dnd-kit/sortable` aux dépendances (`package.json`)
- [ ] 5.2 Wrapper `KanbanPage` avec `<DndContext>` configuré avec les sensors (PointerSensor)
- [ ] 5.3 Implémenter la logique `onDragEnd` dans `KanbanPage` : extraire source/destination et router vers le bon handler (ff ou reset)

## 6. Frontend — colonnes droppables

- [ ] 6.1 Rendre `KanbanColumn` droppable via `useDroppable` de @dnd-kit pour les colonnes `to-explore` et `todo`
- [ ] 6.2 Afficher le highlight visuel sur la colonne cible quand `isOver && isValidTarget`
- [ ] 6.3 Ne pas afficher de highlight sur les colonnes `in-progress`, `done`, `archived` comme cibles (logique de validation dans `onDragEnd`)

## 7. Frontend — cartes draggables

- [ ] 7.1 Rendre `ChangeCard` draggable via `useDraggable` de @dnd-kit pour les statuts `to-explore`, `todo`, `in-progress`
- [ ] 7.2 Désactiver le drag sur les cartes en statut `done` et `archived`
- [ ] 7.3 Désactiver le drag sur les cartes ayant un état ff actif (spinner) ou en erreur

## 8. Frontend — état ff sur les cartes

- [ ] 8.1 Créer le hook `useFfState` qui consomme les events SSE `ff_started` / `ff_done` / `ff_failed` et maintient un map `changeName → "running" | "failed" | null`
- [ ] 8.2 Passer l'état ff à `ChangeCard` et afficher le spinner si `"running"`, l'indicateur d'erreur si `"failed"`
- [ ] 8.3 Déclencher `POST /changes/{name}/ff` dans `onDragEnd` (après fermeture éventuelle de l'ExplorePanel)

## 9. Frontend — dialog de confirmation reset

- [ ] 9.1 Créer le composant `ResetTasksDialog` : message adapté selon `tasks_done > 0` (avertissement) ou `tasks_done === 0` (neutre)
- [ ] 9.2 À la confirmation, appeler `PATCH /changes/{name}/tasks/reset` et invalider la query changes
- [ ] 9.3 À l'annulation, ne rien faire (carte retourne à sa position)

## 10. Frontend — fermeture ExplorePanel avant ff

- [ ] 10.1 Dans `onDragEnd` (to-explore → todo), vérifier si l'ExplorePanel est ouvert pour ce changement (`exploreOpen?.name === changeName`)
- [ ] 10.2 Si ouvert : appeler `DELETE /changes/{name}/explore` puis, à la réponse, déclencher `POST /changes/{name}/ff`
- [ ] 10.3 Si non ouvert : déclencher `POST /changes/{name}/ff` directement

## 11. Frontend — onglet Log dans DetailPanel

- [ ] 11.1 Créer le hook `useConversationRuns(workspaceId, changeName, kind)` qui appelle `GET /changes/{name}/conversations/{kind}`
- [ ] 11.2 Créer le hook `useConversationRun(workspaceId, changeName, kind, ts)` qui appelle `GET /changes/{name}/conversations/{kind}/{ts}`
- [ ] 11.3 Ajouter l'onglet "Log" dans `DetailPanel` : liste des runs ff (timestamps), sélection du run courant (dernier par défaut)
- [ ] 11.4 Afficher les messages du run sélectionné en lecture seule via le renderer de messages existant (réutilisation du rendu ExplorePanel sans zone de saisie)
