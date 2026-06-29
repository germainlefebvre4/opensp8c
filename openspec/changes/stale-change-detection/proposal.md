## Why

Les changements en cours qui n'ont plus d'activité depuis plusieurs jours génèrent du bruit sur le board kanban. Un indicateur passif de staleness crée une pression saine sans perturber le workflow.

## What Changes

- Calcul de `days_since_activity` sur chaque change à partir du `mtime` de `tasks.md`
- Nouveau champ `is_stale` (booléen) calculé côté backend selon un seuil configurable
- Seuil configurable via `openspec/config.yaml` avec valeur par défaut de 7 jours
- Badge discret `⚠ Nj` affiché inline dans `ChangeCard` pour les changes stale

## Capabilities

### New Capabilities
- `stale-change-detection`: Détection et affichage passif des changes sans activité récente

### Modified Capabilities
- `kanban-board`: Ajout de champs `days_since_activity` et `is_stale` dans la réponse API `/changes`

## Impact

- `backend/internal/openspec/change.go` : `Change` struct + `loadChange()` + `ListChanges()`
- `openspec/config.yaml` : nouveau champ `stale_threshold_days`
- `frontend/src/hooks/useChanges.ts` : mise à jour du type `Change`
- `frontend/src/components/ChangeCard.tsx` : badge inline conditionnel
