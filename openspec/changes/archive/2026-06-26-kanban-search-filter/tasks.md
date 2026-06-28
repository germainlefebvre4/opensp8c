## 1. State et logique de filtrage dans KanbanPage

- [x] 1.1 Ajouter le state `searchQuery` (useState) dans `KanbanPage`
- [x] 1.2 Calculer `filteredChanges` en filtrant `changes` par `change.name.toLowerCase().includes(searchQuery.toLowerCase())`
- [x] 1.3 Calculer `filteredArchived` de la même façon sur `archivedChanges`
- [x] 1.4 Passer `filteredChanges` et `filteredArchived` aux colonnes respectives

## 2. Composant barre de recherche

- [x] 2.1 Ajouter l'input de recherche dans `KanbanPage` au-dessus du `<div>` des colonnes
- [x] 2.2 Afficher le bouton `×` conditionnellement (visible si `searchQuery !== ""`)
- [x] 2.3 Brancher le bouton `×` pour réinitialiser `searchQuery` à `""`

## 3. Vérification

- [x] 3.1 Vérifier que la saisie filtre bien les colonnes actives en temps réel
- [x] 3.2 Vérifier que la colonne Archived est filtrée de la même façon
- [x] 3.3 Vérifier que les colonnes à 0 résultat restent visibles avec badge à 0
- [x] 3.4 Vérifier que le bouton `×` réinitialise l'affichage complet
