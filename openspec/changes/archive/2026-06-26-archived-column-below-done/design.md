## Context

La colonne Archived est actuellement une 5e colonne horizontale indépendante dans `KanbanPage.tsx`, séparée de Done par un diviseur vertical `<div className="w-px...">`. `KanbanColumn` a `flex-1` en dur sur son root div, ce qui lui donne une largeur égale aux autres colonnes dans le flex horizontal.

## Goals / Non-Goals

**Goals:**
- Repositionner Archived verticalement sous Done dans le même slot horizontal
- 4 slots d'égale largeur horizontale (To Explore, To Do, In Progress, Done+Archived)
- Done prend la hauteur disponible ; Archived est compact (hauteur auto)
- Séparateur horizontal entre Done et Archived

**Non-Goals:**
- Tout changement backend, hooks, logique d'archivage
- Modification du comportement de pagination Archived (reste 5/+5)
- Modification du style des cartes

## Decisions

### D1 — Prop `className` sur `KanbanColumn`

`KanbanColumn` a `flex-1` en dur sur son root div. Pour contrôler son comportement dans un contexte `flex-col` (vertical), on expose une prop `className?: string` qui override `flex-1` sur le root div.

```
root div: className={`${className ?? 'flex-1'} min-w-[220px] bg-slate-50...`}
```

- Done dans le slot : `className="flex-1 min-h-0"` → prend l'espace vertical disponible
- Archived dans le slot : `className="shrink-0"` → hauteur auto basée sur le contenu

**Alternatif écarté** : wrapper div externe contrôlant la taille. Crée une double imbrication inutile et masque l'intention.

### D2 — Wrapper `flex-col` pour le slot Done+Archived

Dans `KanbanPage.tsx`, le slot Done+Archived est un `div` avec `flex-1 min-w-[220px] flex flex-col min-h-0 gap-2` — il prend la même largeur que les autres colonnes (via `flex-1`) et organise Done + séparateur + Archived en vertical.

```tsx
<div className="flex-1 min-w-[220px] flex flex-col min-h-0 gap-2">
  <KanbanColumn status="done" className="flex-1 min-h-0" ... />
  <div className="h-px bg-slate-200 shrink-0" />
  <KanbanColumn status="archived" className="shrink-0" maxVisible={5} ... />
</div>
```

Le `min-h-0` sur le wrapper et sur Done est nécessaire pour que `flex-1` ne déborde pas au-delà du conteneur parent (comportement flex standard).

### D3 — Suppression du séparateur vertical

L'ancien `<div className="w-px bg-slate-200 shrink-0 self-stretch mx-1" />` est supprimé. Le séparateur visuel est désormais le `h-px` horizontal entre Done et Archived.

## Risks / Trade-offs

- **`min-h-0` obligatoire** : sans ce flag, les navigateurs ne respectent pas la contrainte de hauteur dans un flex-col → Done déborde. C'est un piège CSS classique ; la prop `className` doit l'inclure explicitement.
- **Archived sans `flex-1`** : `shrink-0` sur Archived garantit qu'il ne "vole" pas d'espace à Done si la liste est courte. Sans ça, les deux se partageraient l'espace 50/50.

## Migration Plan

Changement purement frontend, aucune migration de données. Déploiement direct.

## Open Questions

_(aucune)_
