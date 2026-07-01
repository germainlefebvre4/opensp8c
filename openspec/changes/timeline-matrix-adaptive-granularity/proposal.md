## Why

The Matrice view of the Timeline hard-codes one column per calendar day and caps the grid at the 20 most recent dates. Past ~3 weeks of history this cap hides everything older, so the matrix can never give the retrospective view (months to a year+) it's meant to provide. Users need to control the time scale themselves and get a grid that actually fills the space it's given, instead of a magic-number slice of daily columns.

## What Changes

- Add a granularity selector (Jour / Semaine / Mois / Trimestre) to the Matrice mode, replacing the fixed daily-column, 20-date-cap behavior.
- Bucket change dates by the selected granularity (ISO week for "Semaine", calendar quarter for "Trimestre") instead of by exact date.
- Compute cell width elastically to fill the available container width, clamped to a legible min/max (16px–40px); fall back to horizontal scroll (anchored on the most recent bucket) only when even the minimum width doesn't fit.
- Auto-select a default granularity at mount: the finest granularity whose full-history bucket count still fits legibly in the measured container width. This computation does not re-run on later resizes (e.g. opening/closing the right detail panel) — only cell width re-adjusts within the clamp.
- Replace the fixed absolute color thresholds (1/2/3+) with a scale relative to the max change-count observed in the current view, so intensity stays meaningful at any granularity.
- Remove the "20 dates les plus récentes" notice; replace with granularity-aware framing where relevant (e.g. scroll-fallback hint).

## Capabilities

### New Capabilities
(none)

### Modified Capabilities
- `timeline-spec-matrix`: the grid's column model changes from "one column per day, capped at 20" to "one column per bucket at a user-selected granularity, sized to fill available width." Cell intensity coloring changes from fixed absolute thresholds to a scale relative to the current view's max count.

## Impact

- `frontend/src/components/TimelineSpecMatrix.tsx`: column/bucket computation (`useMemo`), cell sizing, intensity scale, date label formatting, granularity selector UI.
- New: a width-measurement mechanism (e.g. `ResizeObserver`-based hook) — first use of this pattern in the frontend; scope it as a small local hook rather than a shared utility unless another consumer emerges.
- No API or data-shape changes — `ChangeRef.date` (`YYYY-MM-DD`) remains the source field; bucketing is purely client-side derivation.
