# Tasks

## 1. Backend — endpoint de suppression

- [x] 1.1 Donner au `KanbanHandler` une référence vers `*pool.Manager` et `*preferences.Service`/`convStore`/`watcher` déjà nécessaires à la cascade ghost (constructeur `NewKanbanHandler`), et vérifier que `main.go`/le point de câblage des handlers compile avec la nouvelle signature.
- [x] 1.2 Implémenter `KanbanHandler.DeleteChange` : vérifie l'existence du dossier `openspec/changes/{name}` (404 sinon), vérifie via `mgr.Status()` qu'aucun `Worker.ActiveChange == name` n'est actif (409 sinon), sinon `os.RemoveAll` du dossier et 204. Vérifier avec un test manuel `curl -X DELETE` sur un change de test.
- [x] 1.3 Ajouter la cascade ghost : rechercher dans `prefs.ListExplorations(workspaceID)` un enregistrement dont `.Name == name`, et si trouvé, exécuter la même séquence que `ExploreHandler.DeleteGhost` (stop session, `DeleteExploration`, suppression du brouillon, `DeleteExplorationLogs`, broadcast `exploration_deleted`). Vérifier qu'un change "solidifié" avec ghost actif fait disparaître les deux enregistrements après l'appel.
- [x] 1.4 Déclarer la route `DELETE /workspaces/{id}/changes/{name}` dans `router.go`, à côté des routes `changes/{name}` existantes. Vérifier que la route répond (pas de 404 routeur).
- [x] 1.5 Écrire/adapter un test backend (sur le modèle de `explore_test.go`) couvrant : suppression réussie, change inexistant (404), worker actif (409), cascade ghost.

## 2. Frontend — appel API et hook

- [x] 2.1 Ajouter `deleteChange(workspaceId, changeName)` dans `frontend/src/lib/api.ts`, sur le modèle de `deleteGhost`, appelant `DELETE /api/workspaces/{workspaceId}/changes/{changeName}`.
- [x] 2.2 Créer `frontend/src/hooks/useDeleteChange.ts` sur le modèle de `useArchive.ts` (mutation + invalidation des query keys `['changes', workspaceId]`). Vérifier que le hook expose `isPending` et l'erreur pour l'affichage dans l'onglet Actions.

## 3. Frontend — DetailPanel : onglet Actions et dialogue de confirmation

- [x] 3.1 Créer `frontend/src/components/DeleteChangeDialog.tsx` sur le modèle de `ResetTasksDialog.tsx` (overlay, titre, message nommant le change, boutons Annuler/Confirmer). Vérifier visuellement dans le navigateur que le style est cohérent avec `ResetTasksDialog`.
- [x] 3.2 Ajouter `'actions'` au type `Tab` de `DetailPanel.tsx` et l'entrée correspondante dans le tableau `tabs`. Vérifier que l'onglet apparaît dans la barre d'onglets pour tout statut différent de `archived`.
- [x] 3.3 Déplacer le footer "Archiver" (`kanban_status === 'done'`, lignes ~390-403) et le footer "To Review" (`kanban_status === 'to-review'`, lignes ~374-387) à l'intérieur du contenu `activeTab === 'actions'`, sans changer leur comportement (les boutons "Approuver & Fusionner"/"Demander correction" restent sans `onClick`). Vérifier qu'un change Done affiche Archiver uniquement dans l'onglet Actions (plus en footer permanent), et idem pour To Review.
- [x] 3.4 Ajouter le bouton Supprimer dans le contenu `activeTab === 'actions'`, visible pour tout statut différent de `archived`, désactivé si un worker de l'Agent Pool est actif sur ce change (nécessite d'exposer ce statut au frontend — cf. tâche 3.5). Au clic, ouvre `DeleteChangeDialog` ; à la confirmation, appelle `useDeleteChange`, puis `onClose()` en cas de succès. Afficher l'erreur en cas d'échec, sur le modèle de `archiveError`.
- [x] 3.5 Exposer au frontend si un worker est actif sur le change (ex. champ sur la réponse de `GET /changes/{name}` ou du listing des changes, alimenté par `mgr.Status()`), et l'utiliser pour désactiver le bouton Supprimer avec une infobulle explicative. Vérifier manuellement en simulant un worker actif (ou en lisant l'état du pool) que le bouton est bien désactivé.

## 4. Vérification bout-en-bout

- [x] 4.1 Lancer l'app (`run` skill ou démarrage manuel), supprimer un change de test en To Do sans worker ni ghost associé, vérifier qu'il disparaît du board et que le dossier `openspec/changes/<name>` n'existe plus sur le disque.
- [x] 4.2 Reproduire le scénario "ghost associé" (solidifier une exploration en To Do puis supprimer la carte solide avant d'avoir "figé") et vérifier que la carte ghost associée disparaît aussi de la colonne To Explore.
- [x] 4.3 Vérifier qu'une carte en colonne Archived affiche bien l'onglet Actions (cf. requirement "Onglet Actions toujours présent") mais sans bouton Supprimer, conformément au requirement de la spec.
