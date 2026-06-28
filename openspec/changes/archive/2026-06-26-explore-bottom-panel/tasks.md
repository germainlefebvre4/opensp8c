## 1. Backend — Session buffer et auto-injection

- [x] 1.1 Ajouter `messages [][]byte`, `outCh chan []byte` et un mutex à `session.Session`
- [x] 1.2 Créer une goroutine de fan-out dans `Manager.Start` : lit stdout du subprocess, ajoute au buffer (cap 500), pousse dans `outCh`
- [x] 1.3 Implémenter la logique de cap à 500 messages (sliding window) dans le buffer
- [x] 1.4 Dans `Manager.Start`, après spawn du subprocess, envoyer le message d'initialisation `/opsx:explore <changeName>` sur stdin uniquement si la session vient d'être créée

## 2. Backend — Handler WebSocket avec replay

- [x] 2.1 Modifier `ExploreHandler.HandleWS` : à la connexion, rejouer le buffer existant (`session.Messages()`) avant de passer en mode stream live
- [x] 2.2 Remplacer la goroutine de lecture stdout par une consommation du `outCh` channel
- [x] 2.3 Vérifier que la fermeture WebSocket ne stoppe pas la goroutine de fan-out (subprocess reste vivant)

## 3. Frontend — ExploreBottomPanel

- [x] 3.1 Créer `frontend/src/components/ExploreBottomPanel.tsx` avec header (nom du changement + bouton close), drag handle et zone de chat
- [x] 3.2 Implémenter le drag-to-resize : `onMouseDown` sur le handle, listeners `mousemove`/`mouseup` sur `window`, hauteur clampée entre 200px et 70vh
- [x] 3.3 Intégrer le composant `ExplorePanel` existant dans `ExploreBottomPanel` (réutiliser sans modification)

## 4. Frontend — Layout KanbanPage

- [x] 4.1 Modifier `KanbanPage` pour passer en layout `flex-col` : zone supérieure (`flex-1`) = colonnes + DetaiPanel, zone inférieure = `ExploreBottomPanel` (conditionnel)
- [x] 4.2 Supprimer le slot latéral droit `w-[420px]` pour le type `explore` dans `activePanel`
- [x] 4.3 Ajouter un état `exploreOpen: { name: string } | null` et `panelHeight: number` (défaut 320px) dans `KanbanPage`
- [x] 4.4 Passer `onOpen` des colonnes `To Explore` pour déclencher `exploreOpen` au lieu du slot latéral
- [x] 4.5 Conserver le comportement du DetailPanel inchangé (slot latéral droit 420px)
