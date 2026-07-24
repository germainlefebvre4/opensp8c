## 1. Backend : Stockage Isolé et API CRUD

- [x] 1.1 Définir le dossier `drafts/` géré par le backend à côté de `preferences.json` (avec fonction utilitaire `draftsPath`).
- [x] 1.2 Implémenter le endpoint `GET /api/workspaces/{id}/explorations/{ghostId}/draft` (lecture de `drafts/<ghostId>.json` ou retour d'une structure vide).
- [x] 1.3 Implémenter le endpoint `PUT /api/workspaces/{id}/explorations/{ghostId}/draft` (écriture sécurisée du JSON).
- [x] 1.4 Implémenter le endpoint `DELETE /api/workspaces/{id}/explorations/{ghostId}/draft` (suppression propre du fichier).
- [x] 1.5 Mettre à jour `DeleteGhost` pour appeler la suppression du brouillon associé.
- [x] 1.6 Ajouter des tests unitaires backend pour valider le bon fonctionnement du CRUD d'exploration.

## 2. Backend : Synchronisation Live via WebSocket & SSE

- [x] 2.1 Modifier le processeur de streaming `serveWS` pour détecter le marqueur `ghost_draft_updated` sur stdout du subprocess.
- [x] 2.2 Sauvegarder automatiquement le brouillon extrait dans `drafts/<ghostId>.json` lors de la détection du marqueur.
- [x] 2.3 Diffuser l'événement SSE `draft_updated` à l'aide du service Watcher.
- [x] 2.4 Mettre à jour le mécanisme de nettoyage périodique (`retention.go`) pour purger les brouillons d'explorations périmées ou orphelines.

## 3. Backend : Intégration de la Promotion (Fast-Forward)

- [x] 3.1 Modifier `PromoteGhost` pour charger le fichier `drafts/<ghostId>.json` associé s'il existe.
- [x] 3.2 Transmettre les tâches et la description de ce brouillon au subprocess FF (ou les insérer dans le `tasks.md` final post-génération).
- [x] 3.3 Supprimer proprement le fichier de brouillon de tâche à la réussite de la promotion dans `runPromoteFF`.

## 4. Frontend : Panneau de Brouillon Interactif Split-Screen

- [x] 4.1 Ajouter les appels d'API du brouillon d'exploration dans le client `frontend/src/lib/api.ts`.
- [x] 4.2 Créer le composant frontend `DraftSidePanel` (affichage de la description, liste de tâches, case à cocher, ajout de tâche).
- [x] 4.3 Adapter `ExplorePanel` pour implémenter un affichage en double volet (split-screen) Chat / Brouillon.
- [x] 4.4 S'abonner aux événements SSE `draft_updated` pour actualiser l'état du brouillon à l'écran.
- [x] 4.5 Mettre en place un mécanisme de débounce de 500ms sur l'édition manuelle du brouillon pour déclencher automatiquement l'appel de mise à jour `PUT`.

## 5. Frontend : Rendu de la Ghost Card sur le Kanban

- [x] 5.1 Adapter l'API de listing des changes pour retourner un compteur compact de tâches de brouillon dans les Ghost Cards.
- [x] 5.2 Mettre à jour l'affichage des Ghost Cards dans la colonne "To Explore" de `KanbanPage.tsx` pour afficher un indicateur visuel distinctif (badge violet "N draft tasks" ou jauge pointillée) si un brouillon existe.
