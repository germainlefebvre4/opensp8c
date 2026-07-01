## Context

La TimelinePage actuelle consomme uniquement `useChanges` + `useArchivedChanges` et dérive sa heatmap depuis `tags.components` (LLM). L'endpoint `/specs/overview` — introduit par `specs-history-view` — expose déjà l'index inversé `spec → []ChangeRef` qui permet de construire `changeName → []specName` côté client. La SpecsPage héberge actuellement un mode Historique (`SpecHistoryView`) qui logiquement appartient à la Timeline.

Aucun nouveau endpoint backend n'est nécessaire : les deux endpoints existants (`/changes`, `/archived-changes`, `/specs/overview`) fournissent toutes les données.

## Goals / Non-Goals

**Goals:**
- Enrichir le mode Changes de la Timeline avec les delta specs (liens vers SpecsPage) et une heatmap basée sur les specs formelles
- Ajouter un mode Matrice (grille spec × temps) avec drill-down spec → change history → DetailPanel
- Migrer SpecHistoryView de SpecsPage vers TimelinePage (mode Matrice)
- Lien de deep-link depuis SpecsPage vers Timeline Matrice avec spec pré-sélectionnée

**Non-Goals:**
- Modification du backend (aucune)
- Pagination ou virtualisation (28 specs, charge négligeable)
- Filtrage dans le mode Matrice (à envisager en suivant si le nombre de specs explose)
- Persistance du mode actif (Changes vs Matrice) — état UI local non persisté

## Decisions

**1. Fusion client-side depuis `/specs/overview` (pas d'ajout sur Change struct)**

Le `/specs/overview` retourne `{ specs: [{name, changes:[]}] }`. On inverse côté client :
```ts
const changeToSpecs = useMemo(() => {
  const map: Record<string, string[]> = {}
  for (const spec of overview.specs)
    for (const ref of spec.changes)
      (map[ref.name] ??= []).push(spec.name)
  return map
}, [overview.specs])
```
Zéro modification backend, zéro nouveau type. La spec-name devient l'identifiant clé du lien (exact match avec les dossiers `openspec/specs/<name>/`).

**2. Séparation visuelle spec chips vs component chips**

- `spec_chips` = `changeToSpecs[change.name]` → liens cliquables (naviguent `/specs?selected=<name>`)
- `extra_comps` = `change.tags?.components` filtrés sur `!knownSpecs.has(c) && !specChips.includes(c)` → chips de filtre seulement (items LLM sans spec formelle : `api-router`, `change-status`...)

**3. Heatmap depuis delta specs (remplace tags.components)**

La heatmap "Composants fréquents" est remplacée par "Specs actives" calculée depuis `overview.specs` : count de changes par spec name. Plus précis et fiable que les tags LLM.

**4. Mode Matrice = SpecHistoryView + grille temporelle**

Le mode Matrice comprend deux vues :
- **Grille** (défaut) : `TimelineSpecMatrix` — grille spec × date, intensité colorée
- **Panel droit** : quand une spec est sélectionnée, `SpecHistoryView` filtré sur cette spec + `DetailPanel` quand un change est cliqué

`SpecHistoryView` est réutilisé sans modification — il accepte un `SpecOverview` et émet `onChangeClick`. En mode Matrice, on lui passe un `SpecOverview` filtré sur la spec sélectionnée.

**5. Deep-link spec via param URL `?spec=<name>`**

`TimelinePage` lit `searchParams.get('spec')`. Si présent, le mode Matrice s'active et le panel droit s'ouvre directement sur cette spec. SpecsPage génère le lien avec `useSearchParams`. Aucune modification du router nécessaire.

**6. SpecsPage perd le mode Historique**

On retire : `mode`, `setMode`, `useSpecsOverview`, `SpecHistoryView`, `historyDetailOpen`. On ajoute : un lien discret "Voir l'historique →" dans le header de la spec sélectionnée. La SpecsPage redevient stateless sur ce point.

## Risks / Trade-offs

- **Nommage spec vs component** : un même concept peut avoir un nom différent entre `specs/<name>` et `tags.components`. La fusion est un union non-normalisée — le gap est visible (ex. `kanban-change-search` vs `kanban-board`). Acceptable pour l'instant ; une table de mapping pourra être introduite plus tard.
- **Performance grille** : 28 specs × ~20 jours = 560 cellules. Rendu direct sans virtualisation est correct. Au-delà de 100 specs, à revoir.
- **SpecHistoryView context** : le composant est conçu pour un `SpecOverview` complet. Pour le mode Matrice, on lui passe un overview filtré sur 1 spec. Le rendu est correct (1 ligne dans la liste), mais la stats bar ("28 specs, 31 changes") sera trompeuse si non adaptée. Mitigation : en mode Matrice, masquer la stats bar ou la contextualiser ("1 spec sélectionnée").
