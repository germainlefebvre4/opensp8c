## Context

Le `WorkspaceSidebar` affiche des pastilles colorées par statut Kanban pour chaque workspace. Ces pastilles sont construites à partir de `BADGE_COLORS` et `BADGE_ORDER`, deux constantes définies dans le composant. Le backend fournit déjà `task_counts.done` via l'API `/workspaces`, mais le frontend ne l'utilise pas. Par ailleurs, un `group-hover:hidden` appliqué aux pastilles les masque lors du survol de l'item.

## Goals / Non-Goals

**Goals:**
- Afficher la pastille "Done" avec la couleur `bg-emerald-500` (cohérente avec le dot Kanban)
- Conserver les pastilles visibles lors du survol d'un item (supprimer `group-hover:hidden`)

**Non-Goals:**
- Modifier le backend ou le type `Workspace`
- Changer l'ordre des autres pastilles
- Modifier la couleur ou le style des pastilles existantes

## Decisions

**Couleur `bg-emerald-500`** : cohérente avec `STATUS_STYLES['done'].dot` dans `KanbanColumn.tsx`. Toutes les autres pastilles sidebar matchent déjà leur contrepartie Kanban (`violet-400`, `slate-400`, `amber-400`).

**Suppression de `group-hover:hidden`** : le bouton X a son propre mécanisme d'apparition (`opacity-0 group-hover:opacity-100`), les deux éléments coexistent sans conflit. La suppression ne nécessite pas de réorganisation du layout — le flex container `gap-1` absorbe naturellement les deux.

**Ordre des pastilles** : `['to-explore', 'todo', 'in-progress', 'done']` — "done" ajouté en fin, reflétant l'ordre naturel du workflow Kanban.

## Risks / Trade-offs

- [Espace limité] Si un projet a des compteurs non nuls dans les 4 colonnes + le bouton X au survol, la ligne peut être dense → le texte `truncate` absorbe la compression, pas de débordement
- Aucun risque de régression backend (la clé `done` est déjà présente dans `task_counts`)
