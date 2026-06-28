## Context

`KanbanPage` charge l'intégralité des changes du workspace en mémoire via `useChanges(workspaceId)` et `useArchivedChanges(workspaceId)`. Il distribue ensuite ces arrays aux `KanbanColumn` par filtrage sur `kanban_status`. Le filtre de recherche s'insère dans ce flux sans modifier l'API, le backend, ni les composants en aval.

## Goals / Non-Goals

**Goals:**
- Barre de recherche textuelle au-dessus des colonnes Kanban (pleine largeur)
- Filtre instantané et insensible à la casse sur `change.name`, couvrant toutes les colonnes (actives + Archived)
- Colonnes restantes visibles à 0 résultats (structure du board préservée)
- Bouton `×` pour réinitialiser le filtre

**Non-Goals:**
- Recherche cross-workspace
- Filtrage par statut, tags, ou dates
- Raccourci clavier (⌘K) — peut être ajouté en v2
- Modification du backend ou des endpoints API

## Decisions

### State dans KanbanPage (pas de context/store)
`KanbanPage` est déjà le point de coordination entre les données et les colonnes. Ajouter `searchQuery` comme `useState` local maintient la cohérence du flux de données existant sans introduire de couplage inutile.

Alternative rejetée : state global (Zustand/Context) — over-engineering pour un filtre dont la portée est strictement locale à la vue Kanban.

### Filtrage avant distribution aux colonnes
```
const filtered = changes.filter(c =>
  c.name.toLowerCase().includes(searchQuery.toLowerCase())
)
// puis filtered.filter(c => c.kanban_status === col.status) comme avant
```
`KanbanColumn` et `ChangeCard` ne sont pas modifiés — ils reçoivent déjà des tableaux, le filtre est transparent pour eux.

### Colonnes visibles à 0 résultats
Masquer une colonne vide casse le modèle mental de l'utilisateur (il ne sait plus si "In Progress" est vide ou cachée). Les colonnes restent visibles avec leur badge count à 0.

### Archived incluse dans le filtre
Pas de cas spécial pour Archived — elle reçoit `archivedChanges.filter(...)` de la même façon. Comportement cohérent et prévisible.

## Risks / Trade-offs

- **Filtre réinitialisé au changement de workspace** → comportement attendu, `KanbanPage` se remonte sur changement de `workspaceId`
- **Filtre sur `name` uniquement** → les changes avec des noms peu descriptifs sont moins trouvables, mais c'est une contrainte de la convention OpenSpec (noms kebab-case), pas un bug à corriger ici
