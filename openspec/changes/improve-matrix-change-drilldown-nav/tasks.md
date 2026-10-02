# Tasks

## 1. DetailPanel : retour optionnel

- [ ] 1.1 Ajouter à `DetailPanel` les props optionnelles `onBack`, `backLabel` et `onActionDone` ; `onActionDone` est appelé après archivage, suppression et correction (repli sur `onClose`). Vérifier que les tests existants du Kanban passent sans modification.
- [ ] 1.2 Afficher dans le header, quand `onBack` est fourni, une ligne « ← {backLabel} » au-dessus du titre (icône `ArrowLeft`, `aria-label` via la clé `detailPanel:backTo`). Ajouter la clé aux locales fr et en.
- [ ] 1.3 Tests dans `DetailPanel.test.tsx` : sans `onBack` aucun bouton retour ; avec `onBack` le clic l'appelle et le X appelle `onClose` ; après archivage, suppression ou correction `onActionDone` est appelé et `onClose` ne l'est pas. Vérifier `npm test -- DetailPanel` au vert.

## 2. TimelinePage : navigation en pile

- [ ] 2.1 Brancher le `DetailPanel` de la Matrice : `onBack` et `onActionDone` ramènent à la liste de la spec (`selectedChange = null`), `onClose` désélectionne le change et la spec ; passer `backLabel={selectedSpec}`. Vérifier à la main (ou par test) que ← revient à la liste et que X referme tout.
- [ ] 2.2 Remplacer le rendu conditionnel du slot droit par un conteneur `relative` : l'en-tête de spec et `SpecHistoryView` restent montés, le `DetailPanel` est superposé (`absolute inset-0`, fond blanc) ; marquer la liste `inert` / `aria-hidden` pendant le drill-down. Vérifier que le scroll de la liste est identique avant et après un aller-retour.
- [ ] 2.3 Ajouter l'état `lastViewedChange` (renseigné à l'ouverture d'un change, réinitialisé au changement de spec, à la fermeture complète et au passage à l'onglet Changes) et le transmettre à `SpecHistoryView` via `selectedChangeName`. Vérifier le surlignage du dernier change consulté au retour.
- [ ] 2.4 Gérer Échap dans `TimelinePage` (mode Matrice, spec sélectionnée) : retour si un change est ouvert, fermeture du panel sinon ; ignoré si `defaultPrevented`, focus dans un champ de saisie ou `[role="dialog"]` présent. Vérifier par les tests de 2.5.
- [ ] 2.5 Créer `pages/TimelinePage.test.tsx` (mêmes conventions que `KanbanPage.review.test.tsx`) couvrant les scénarios de la spec : ouverture du drill-down avec « ← nom-de-la-spec », retour via le bouton, X qui ferme tout, retour après action sur le change, Échap (drill-down, liste, ignoré avec dialogue ou saisie), surlignage du dernier change consulté. Vérifier `npm test -- TimelinePage` au vert.

## 3. Vérification finale

- [ ] 3.1 Lancer la suite frontend complète (`npm test`), le lint et le typecheck ; tout doit être vert.
- [ ] 3.2 Parcours manuel dans l'application (Timeline → Matrice → clic sur une spec → clic sur un change → ←, X, Échap, scroll d'une spec à nombreux changes) et vérifier que le Kanban n'a pas changé (ouverture et fermeture du `DetailPanel`).
