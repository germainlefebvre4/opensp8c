## Why

Le Kanban Board est fonctionnel mais les cartes ne sont pas interactives (clic impossible) et le layout ne tire pas parti de l'espace disponible — les colonnes ne prennent pas toute la hauteur et l'application ne s'étend pas en pleine largeur. Ces frictions ralentissent la navigation quotidienne entre les changes.

## What Changes

- Les cartes du Kanban deviennent entièrement cliquables : clic sur une carte de la colonne **To Explore** ouvre l'ExplorePanel (session de conversation) ; clic sur une carte dans les autres colonnes ouvre un nouveau **DetailPanel** latéral affichant le détail riche du change
- Le **DetailPanel** affiche : nom, statut, progression des tâches avec liste détaillée, et le contenu des artifacts (proposal, design) — plus les actions de changement de statut et d'archivage
- Les boutons d'action (`→ To Do`, `Archiver`) migrent du corps de la carte vers le DetailPanel — la carte est épurée
- Les colonnes Kanban s'étendent sur toute la hauteur disponible (alignement `stretch`)
- L'application occupe toute la largeur de la page, les colonnes se distribuent équitablement dans l'espace disponible
- Un nouveau endpoint `GET /api/workspaces/{id}/changes/{name}` retourne le détail complet d'un change (tasks avec texte, contenu des artifacts)

## Capabilities

### New Capabilities

- `kanban-change-detail`: Panneau latéral de détail d'un change — liste des tâches, contenu des artifacts (proposal, design), actions (changement de statut, archivage) — accessible au clic depuis les colonnes To Do, In Progress et Done

### Modified Capabilities

- `kanban-board`: Les cards deviennent cliquables (comportement différencié selon la colonne), les boutons d'action disparaissent de la carte, les colonnes prennent toute la hauteur, l'app occupe toute la largeur

## Impact

- **Backend** : `internal/openspec/change.go` (nouvelle fonction `GetChangeDetail`), `internal/api/handlers/kanban.go` (nouveau handler `GetChange`), `internal/api/router.go` (nouvelle route)
- **Frontend** : `components/ChangeCard.tsx` (suppression boutons, ajout onClick), `components/KanbanColumn.tsx` (layout hauteur), `pages/KanbanPage.tsx` (gestion activePanel, layout largeur), nouveaux `hooks/useChangeDetail.ts` et `components/DetailPanel.tsx`
- **Aucune dépendance externe** nouvelle, aucun breaking change sur l'API existante
