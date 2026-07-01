## Context

`TimelineSpecMatrix.tsx` currently buckets changes by exact calendar date (`YYYY-MM-DD`), collects all dates into a sorted set, and hard-slices to the 20 most recent (`dates.slice(0, 20)`). Column width is fixed (`min-w-[28px]`), so the grid never adapts to the space it's given — it either wastes width (few dates) or truncates history (many dates), and there is no way to see a retrospective view beyond ~20 days. This change replaces that with a user-selected granularity and a grid that fills the available width. No API or data model changes: `ChangeRef.date` (`YYYY-MM-DD`) stays the source field; bucketing is a pure client-side derivation.

## Goals / Non-Goals

**Goals:**
- Let the user pick the column granularity: Jour, Semaine, Mois, Trimestre.
- Make column width fill the available container width within a legible range, instead of a fixed width and an arbitrary date cap.
- Keep cell color intensity meaningful at any granularity.
- Pick a sensible default granularity automatically on first load.

**Non-Goals:**
- No Semestre granularity (dropped during exploration — Trimestre covers the long-history case well enough).
- No server-side aggregation changes; bucketing remains a client-side derivation over existing data.
- No persistence of the user's granularity choice across sessions/reloads.
- No re-selection of granularity in response to layout resize (window resize, detail panel open/close) — only cell width reflows.

## Decisions

**1. Bucket key format — explicit, zero-padded, lexicographically sortable per granularity.**
Keeping the existing `.sort().reverse()` pattern working (no custom comparator) requires each granularity's key to sort correctly as a plain string:
- Jour: `YYYY-MM-DD` (unchanged)
- Semaine: `YYYY-Www`, using the ISO 8601 week-year and week number (e.g. `2026-W01`) — not the calendar year of any single day in the bucket, since ISO weeks near year boundaries can belong to a different week-year than the calendar date suggests.
- Mois: `YYYY-MM`
- Trimestre: `YYYY-Qn` (Q1 = Jan–Mar, Q2 = Apr–Jun, Q3 = Jul–Sep, Q4 = Oct–Dec)

Alternative considered: key every granularity by its bucket-start date (e.g. week bucket = Monday's ISO date). Rejected — it loses the granularity semantics in the key itself, complicates label derivation, and doesn't sort more correctly than the explicit format.

**2. Elastic cell sizing with clamp + scroll fallback.**
`cellWidth = clamp(containerWidth / bucketCount, MIN_CELL_PX, MAX_CELL_PX)` (16–40px). If `bucketCount * MIN_CELL_PX > containerWidth`, cells stay at `MIN_CELL_PX` and the existing `ScrollArea` horizontal scrollbar handles overflow, anchored so the most recent bucket is initially in view.

Alternatives considered: fixed cell width + always scroll (simpler, but never visually "fills" the space as requested); pure elastic with no floor (illegible at fine granularity over long history — reintroduces the original problem in a different form).

**3. Width measurement via a small local `ResizeObserver` hook.**
First use of `ResizeObserver` in this codebase. Scoped as a local hook (e.g. `useContainerWidth`) inside/near the component rather than a shared utility, since there's no second consumer yet — avoid premature abstraction.

**4. Default granularity computed once at mount.**
Evaluate Jour → Semaine → Mois → Trimestre (finest first) against the full history span and the width measured at mount; pick the finest one whose full bucket count fits at `MIN_CELL_PX` without requiring scroll. Falls back to Trimestre (coarsest) if even that needs scroll. Recomputed only if the underlying data materially changes the history span — never on a pure layout resize.

Alternative considered: recompute the default on every resize. Rejected — decided during exploration that this would silently change the user's chosen view whenever the detail panel opens/closes, undermining the "user chooses" model.

**5. Color intensity relative to the current view's max count.**
Replace fixed absolute thresholds (1/2/3+) with a scale derived from the maximum non-zero count present in the currently visible buckets (e.g. quartile bands: low / mid / high / max). Recomputed whenever the matrix data or granularity changes.

Alternative considered: fixed thresholds tuned per granularity. Rejected as brittle — would need manual retuning per granularity and wouldn't adapt to actual change volume over time.

## Risks / Trade-offs

- [Risk] ISO week-year edge cases could confuse users if a week's label appears to belong to the "wrong" year → Mitigation: label the ISO week-year explicitly (e.g. `S01 '26`), not the calendar year of any single day in the bucket.
- [Risk] Auto-computed default could pick something surprising for datasets with unusual gaps (e.g. one change today, nothing for a year) → Mitigation: it's a one-time default; the user can always override manually.
- [Risk] Relative color scale means the same absolute count can render as a different shade depending on what else is in view → Mitigation: hover tooltip always shows the exact count; color is a relative-glance aid, not the source of truth.
- [Risk] `ResizeObserver` is new infra here; needs a disconnect-on-unmount cleanup to avoid leaks.

## Migration Plan

No data migration. Purely a frontend component change behind the existing "Matrice" toggle. Ships as a normal frontend deploy; rollback is a plain revert of the component change.

## Open Questions

- Exact placement of the granularity selector (inline with the `[Changes | Matrice]` toggle vs. inside the matrix panel) — implementation detail, no behavioral impact.
- Whether "Trimestre" should be visually hinted/disabled when history is too short to produce more than 1–2 buckets — polish, not a hard requirement.
