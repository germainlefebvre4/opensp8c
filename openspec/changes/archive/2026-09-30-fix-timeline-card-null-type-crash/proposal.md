# Proposal

## Why

La page Timeline peut faire planter toute l'application (écran blanc) lorsqu'un change expose `tags` sans `tags.type`. Le backend sérialise alors `"type": null` (`Tags.Type` nil n'est pas normalisé, contrairement à `AgentSpecialization`), et `TimelineChangeCard.tsx:43` appelle `c.tags?.type.map(...)` : le `?.` protège `tags` mais pas `type`, d'où un `TypeError` au rendu. Aucun `ErrorBoundary` n'existe dans `frontend/src`, donc l'exception démonte l'application entière. Le problème a été relevé lors de vérifications UI d'un change précédent et laissé de côté ; on le traite maintenant dans un change dédié.

## What Changes

- Frontend : `TimelineChangeCard` tolère un `tags.type` absent ou `null` (aucun badge de type rendu, pas d'erreur), comme le font déjà `ChangeCard`, `DetailPanel`, `TimelinePage` et `KanbanPage`.
- Backend : `loadChange` normalise un `Tags.Type` nil en tableau vide, de sorte que l'API expose toujours `tags.type` comme un tableau (`[]`), à l'image de `agent_specialization`.
- Tests de régression : un test Go (tags sans `type` -> `type: []`) et un test Vitest pour `TimelineChangeCard` (tags sans `type`, `type` null, `type` vide).
- Hors périmètre : ajout d'un `ErrorBoundary` global (changement plus large, à traiter séparément) et durcissement de `'○'.repeat(5 - complexity)` si `complexity > 5`.

## Capabilities

### New Capabilities
<!-- Aucune -->

### Modified Capabilities
- `change-tags`: le backend garantit que `tags.type` est toujours un tableau dans la réponse API, y compris quand le `.openspec.yaml` porte `tags` sans `type`.
- `change-timeline`: une entrée de la timeline dont les tags n'ont pas de `type` (absent, `null` ou vide) s'affiche sans erreur et sans badge de type.

## Impact

- `backend/internal/openspec/change.go` (`loadChange`) et `backend/internal/openspec/change_test.go`.
- `frontend/src/components/TimelineChangeCard.tsx` et un nouveau `frontend/src/components/TimelineChangeCard.test.tsx`.
- Aucun changement d'API au sens contrat : `tags.type` était déjà documenté comme tableau ; on supprime seulement le cas `null` qui violait ce contrat.
