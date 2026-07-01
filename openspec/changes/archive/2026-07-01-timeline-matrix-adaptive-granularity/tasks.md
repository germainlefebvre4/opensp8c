## 1. Bucketing utilities

- [x] 1.1 Add a `Granularity` type (`'day' | 'week' | 'month' | 'quarter'`) and a `bucketKey(date: string, granularity: Granularity): string` function producing `YYYY-MM-DD` / `YYYY-Www` (ISO week-year) / `YYYY-MM` / `YYYY-Qn`
- [x] 1.2 Add a `bucketLabel(key: string, granularity: Granularity): string` function for the short header label per granularity (e.g. `24/03`, `S12 '26`, `mars 24`, `T1 26`)
- [x] 1.3 Unit-test `bucketKey`/`bucketLabel` around ISO week-year boundary dates (e.g. Dec 29–31 and Jan 1–3) to confirm correct week-year assignment and lexicographic sort order — added `vitest` as a new devDependency (`frontend/vitest.config.ts`, `npm run test`) and `frontend/src/lib/bucketing.test.ts` covering the Wikipedia ISO-8601 reference boundary dates plus zero-padded sort order

## 2. Width measurement

- [x] 2.1 Add a local `useContainerWidth` hook (`ResizeObserver` on a ref) with cleanup on unmount
- [x] 2.2 Wire the hook to the matrix's scrollable container in `TimelineSpecMatrix.tsx`

## 3. Granularity state and default selection

- [x] 3.1 Add granularity as component state, initialized lazily (not to a fixed default)
- [x] 3.2 Compute, once per mount (and when the underlying `specs` history span changes), the finest granularity (Jour → Semaine → Mois → Trimestre) whose full bucket count fits at `MIN_CELL_PX` in the width measured at mount; use it to initialize granularity state
- [x] 3.3 Ensure later width changes (resize, detail panel open/close) do not re-run the default computation or change the selected granularity

## 4. Bucket computation and matrix data

- [x] 4.1 Replace the `dates` useMemo with a `buckets` useMemo keyed on `[specs, granularity]`, using `bucketKey` to group and `sort().reverse()` for chronological order
- [x] 4.2 Replace the `matrix` useMemo aggregation to sum change counts per `spec × bucket` instead of per `spec × date`
- [x] 4.3 Remove the hard-coded `visibleDates = dates.slice(0, 20)` cap and the "20 dates les plus récentes" notice

## 5. Elastic column sizing and scroll fallback

- [x] 5.1 Compute `cellWidth = clamp(containerWidth / bucketCount, MIN_CELL_PX=16, MAX_CELL_PX=40)`
- [x] 5.2 When `bucketCount * MIN_CELL_PX > containerWidth`, keep cells at `MIN_CELL_PX` and rely on the existing horizontal `ScrollArea`, initially scrolled to show the most recent bucket
- [x] 5.3 Apply `cellWidth` to column headers and cells (replacing the fixed `min-w-[28px]` / `w-5 h-5`)

## 6. Relative color intensity

- [x] 6.1 Replace `getIntensityClass` (fixed 0/1/2/3+ thresholds) with a function computing bands relative to the max non-zero count in the current `matrix` view
- [x] 6.2 Recompute the max/bands in a `useMemo` keyed on `[matrix]` (which already changes with `specs`/`granularity`)
- [x] 6.3 Keep the hover tooltip showing the exact count regardless of the relative band

## 7. Granularity selector UI

- [x] 7.1 Add a granularity selector control (Jour / Semaine / Mois / Trimestre) — placed inside `TimelineSpecMatrix.tsx`'s own header row (above the grid) rather than in `TimelinePage.tsx`'s mode toggle, to keep the change scoped to one file per the proposal's Impact section; same visual pattern (segmented control) as the `[Changes | Matrice]` toggle
- [x] 7.2 Wire selector changes to update granularity state (no default recomputation on manual change)

## 8. Verification

- [x] 8.1 Manually verify each granularity against real workspace data: column count, labels, sort order, and cell aggregation
- [x] 8.2 Manually verify elastic fit at a wide viewport (columns stretch to fill) and at a narrow one / with the detail panel open (scroll fallback engages, most recent bucket visible first)
- [x] 8.3 Manually verify that opening/closing the detail panel or resizing the window never changes the selected granularity
- [x] 8.4 Update `openspec/specs/timeline-spec-matrix/spec.md` behavior is covered — run through each new/modified scenario by hand

## 9. Fixes from /opsx:verify

- [x] 9.1 Extract `cellWidth` clamp math into `frontend/src/lib/gridSizing.ts` (`computeCellWidth`) and unit-test the floor/fallback branch directly (`gridSizing.test.ts`) — closes the gap where the real dataset never had enough buckets to empirically trigger horizontal-scroll fallback
- [x] 9.2 Reset the default-granularity computation when the history span (min/max date) actually changes, not just once ever — `TimelineSpecMatrix.tsx` now tracks `historySpanKey` and clears `hasSetDefault` when it changes, matching design.md Decision 4 as written
- [x] 9.3 Update design.md (Decision 3) and proposal.md (Impact) to say `useContainerWidth` lives in `frontend/src/hooks/` per the project's existing convention, rather than the originally-planned "local, not shared" placement
- [x] 9.4 Add a one-line comment explaining why `cellWidth`/`squareSize` use inline `style` instead of Tailwind classes (runtime-computed values, not a static class the JIT scanner can see)
- [x] 9.5 Render a "Chargement..." placeholder inside the scroll container while `granularity` is still `null` (between mount and the first width measurement), instead of a bare table with no columns
