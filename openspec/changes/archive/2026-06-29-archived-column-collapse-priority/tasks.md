## 1. Fix layout flex — priorité Done sur Archived

- [x] 1.1 Dans `KanbanPage.tsx`, remplacer `className="shrink-0"` de la `KanbanColumn` Archived par `className="max-h-[40%] overflow-y-auto"`
- [x] 1.2 Vérifier que la colonne Done conserve `className="flex-1 min-h-0"` (aucun changement requis)
- [x] 1.3 Passer `maxVisible={3}` (au lieu de 5) à la `KanbanColumn` Archived dans `KanbanPage.tsx`

## 2. Ajout du bouton collapse sur KanbanColumn

- [x] 2.1 Ajouter prop `collapsible?: boolean` à l'interface `Props` de `KanbanColumn.tsx`
- [x] 2.2 Ajouter `useState<boolean>(false)` pour `collapsed` dans `KanbanColumn`
- [x] 2.3 Afficher un bouton chevron (ChevronDown / ChevronUp selon l'état) dans le header de la colonne, visible uniquement si `collapsible === true`
- [x] 2.4 Conditionner le rendu de la liste de cartes et du bouton "Afficher plus" à `!collapsed`
- [x] 2.5 Passer `collapsible={true}` à la `KanbanColumn` Archived dans `KanbanPage.tsx`

## 3. Vérification visuelle

- [x] 3.1 Tester le comportement au redimensionnement vertical : Done conserve l'espace, Archived se scroll ou se compresse
- [x] 3.2 Tester l'ouverture du bottom panel : Done reste lisible, Archived ne déborde pas
- [x] 3.3 Tester le collapse/expand : header Archived reste visible, chevron change d'état
- [x] 3.4 Tester "Afficher plus" sur Archived : incréments de 3, bouton disparaît quand tout est visible
