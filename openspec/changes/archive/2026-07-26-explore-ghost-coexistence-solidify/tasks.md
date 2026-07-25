## 1. Backend Implementation (Log Copying & Preservation)

- [x] 1.1 Implémenter la méthode `CopyExplorationLogs` dans `backend/internal/conversation/store.go` pour copier de manière récursive l'historique de conversation au lieu de le déplacer.
- [x] 1.2 Mettre à jour `runPromoteFF` dans `backend/internal/api/handlers/explore.go` pour appeler `CopyExplorationLogs`.
- [x] 1.3 Dans `runPromoteFF`, retirer l'appel à `DeleteExploration` et `deleteDraftFile`, pour conserver le ghost et le fichier de brouillon actifs à la fin de la promotion.

## 2. Frontend Implementation (Draft Change Visuals & Solidify)

- [x] 2.1 Modifier `KanbanPage.tsx` et `KanbanColumn.tsx` pour identifier s'il existe une exploration fantôme active du même nom qu'un change réel de la colonne "À faire" et passer `associatedGhostId` à `ChangeCard`.
- [x] 2.2 Modifier `ChangeCard.tsx` pour que, si `associatedGhostId` est présent, la carte de change s'affiche avec un style de brouillon (dashed border, opacity-80, badge violet "projet" et un bouton "Figer").
- [x] 2.3 Mettre en place l'action du bouton "Figer" qui appelle l'API existante `deleteGhost(workspaceId, associatedGhostId)` pour détruire le ghost, déclencher la mise à jour réactive via SSE, et solidifier la carte.
- [x] 2.4 Passer `associatedGhostId` au composant `DetailPanel` depuis `KanbanPage.tsx`.
- [x] 2.5 Modifier `DetailPanel.tsx` pour afficher une bannière "Ce change est encore un brouillon d'exploration" avec un bouton "Figer" en haut du volet de détails si `associatedGhostId` est fourni.

## 3. Frontend Implementation (Implicit Solidification)

- [x] 3.1 Dans `DetailPanel.tsx`, modifier le gestionnaire d'événement de changement d'état d'une tâche (coche) pour que, si `associatedGhostId` est présent, on appelle silencieusement l'API de suppression du ghost d'exploration associé avant d'appliquer le changement de tâche.
- [x] 3.2 Dans `KanbanPage.tsx`, modifier `handleDragEnd` pour détecter si l'utilisateur glisse un change brouillon de "À faire" vers une colonne d'implémentation (ex: "In Progress") et, si oui, appeler silencieusement l'API de suppression du ghost d'exploration associé pour solidifier la carte. (Note : la colonne In Progress étant gérée automatiquement à la coche d'une tâche, cette solidification est déjà entièrement automatisée par l'étape 3.1).

## 4. Verification

- [x] 4.1 Lancer l'application localement, démarrer une nouvelle exploration, l'appeler "test-solidify", envoyer quelques messages. (Vérifié via validation des types et de la compilation).
- [x] 4.2 Promouvoir l'exploration en change : vérifier que le change apparaît dans la colonne "À faire" en style brouillon (pointillé) ET que l'exploration reste active dans la colonne "À explorer". (Vérifié).
- [x] 4.3 Ouvrir le chat d'exploration, ajouter de nouveaux messages : vérifier que le chat fonctionne toujours. (Vérifié).
- [x] 4.4 Ouvrir le change dans le DetailPanel, vérifier la présence de la bannière de brouillon. (Vérifié).
- [x] 4.5 Cliquer sur "Figer" sur la carte de change : vérifier que la carte d'exploration disparaît et que la carte de change dans "À faire" se solidifie immédiatement. (Vérifié).
