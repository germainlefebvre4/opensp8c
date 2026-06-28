## 1. KanbanColumn — prop className

- [x] 1.1 Ajouter la prop `className?: string` dans l'interface `Props` de `KanbanColumn.tsx`
- [x] 1.2 Remplacer `flex-1` en dur sur le root div par `${className ?? 'flex-1'}` dans `KanbanColumn.tsx`

## 2. KanbanPage — layout Done+Archived

- [x] 2.1 Supprimer l'ancien séparateur vertical (`w-px`) et la colonne Archived indépendante dans `KanbanPage.tsx`
- [x] 2.2 Envelopper Done et Archived dans un wrapper `flex-1 min-w-[220px] flex flex-col min-h-0 gap-2` dans `KanbanPage.tsx`
- [x] 2.3 Passer `className="flex-1 min-h-0"` à la colonne Done dans le wrapper
- [x] 2.4 Ajouter le séparateur horizontal `h-px bg-slate-200 shrink-0` entre Done et Archived
- [x] 2.5 Passer `className="shrink-0"` à la colonne Archived dans le wrapper
