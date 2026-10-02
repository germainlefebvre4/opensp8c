# Design

## Context

`VALID_DROPS` est une constante locale de `KanbanPage.tsx`. Elle sert deux fois : de garde-fou dans `handleDragEnd` (`allowed.includes(targetStatus)`) et de source à `validDropSources`, qui pilote l'éclairage des colonnes dans `KanbanColumn`. Les branches de `handleDragEnd` ne traitent que `→ ready`, `→ todo` et `→ to-explore`. Toute entrée de la table sans branche correspondante produit un drop accepté visuellement et ignoré. Les cartes `to-review` ne sont pas draggables (`DRAGGABLE_STATUSES` dans `ChangeCard.tsx`), donc leurs entrées sont inatteignables.

## Goals / Non-Goals

**Goals:**
- Une seule définition des transitions, dont le contenu correspond à la spec `kanban-drag-drop`.
- Un test qui échoue si une transition hors spec est ajoutée sans branche de traitement.

**Non-Goals:**
- Ne pas toucher à `kanbanCollision.ts` ni au rendu du drag overlay.
- Ne pas traiter les `catch { /* ignore */ }` de `handleDragEnd`.
- Ne pas modifier le backend ni `DRAGGABLE_STATUSES`.

## Decisions

**Extraire la table dans `frontend/src/lib/kanbanDrops.ts`** (constante `VALID_DROPS`), importée par `KanbanPage.tsx`. Elle devient testable sans monter la page, comme `kanbanCollision.ts` et `unlaunchError.ts`. Alternative écartée : la garder dans `KanbanPage.tsx` et la tester via un rendu complet, ce qui est lourd et fragile avec dnd-kit.

**Supprimer les entrées au lieu de les implémenter.** `in-progress`, `to-review` et `done` sont dérivés de l'avancement des tâches et du pool. Un drag manuel les rendrait incohérents avec `tasks.md`. Alternative écartée : implémenter des endpoints de transition, plus coûteux et contraire à la spec.

**Le test compare la table à la liste exacte de la spec**, sous forme d'égalité de la table entière. Un test ciblé sur les seules entrées retirées ne détecterait pas une nouvelle dérive.

## Risks / Trade-offs

- [Le test duplique la spec] → Acceptable : il sert de garde-fou, et un écart force à mettre à jour la spec et le test ensemble.
- [Un usage de `validDropSources` hors `KanbanPage` ignore la nouvelle table] → Les tests existants de `KanbanColumn` passent la prop explicitement ; on vérifie qu'aucun autre appelant ne dépend des anciennes entrées.
