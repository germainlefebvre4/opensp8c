## 1. Refactor Layout — état workspace vers URL

- [x] 1.1 Remplacer `useState<string | null>` par `useSearchParams` dans `Layout.tsx`
- [x] 1.2 Calculer `effectiveId` depuis le param `workspace` avec fallback sur `workspaces[0]?.id`
- [x] 1.3 Ajouter un `useEffect` qui initialise `?workspace=<id>` dans l'URL si le param est absent (replace: true)
- [x] 1.4 Mettre à jour le handler `onSelect` pour écrire le param via `setSearchParams` au lieu de `setActiveId`

## 2. Propagation du param dans la navigation

- [x] 2.1 Mettre à jour les `NavLink` (Kanban / Specs) pour utiliser `to={{ pathname, search: searchParams.toString() }}`

## 3. Vérification

- [x] 3.1 Vérifier que le refresh conserve le workspace sélectionné
- [x] 3.2 Vérifier que la navigation Kanban ↔ Specs conserve le param `workspace`
- [x] 3.3 Vérifier le fallback quand `?workspace=<id_invalide>` est dans l'URL
