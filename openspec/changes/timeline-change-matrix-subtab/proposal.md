## Why

The Timeline page switches between its Changes and Matrix views with a pill toggle next to a redundant "Timeline" title. The Specs and Settings pages already use underlined sub-tabs, so Timeline looks inconsistent and wastes a header line that the matrix could use.

## What Changes

- Replace the page header (`<h1>` title + pill toggle) of `TimelinePage` with a sub-tab bar `Changes | Matrix`, styled like the `SpecsPage` sub-tabs.
- Keep the active sub-tab in local component state (not in the URL), like `SpecsPage`. Changes stays the default.
- Keep the `?spec=<name>` deep-link opening the Matrix sub-tab with that spec selected.
- Remove the now-unused `title` key from the `timeline` i18n namespace (en/fr).
- The granularity selector inside the matrix keeps its pill style.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `timeline-spec-matrix`: the requirement "Mode Matrice dans la Timeline" now describes sub-tabs replacing the page header instead of a toggle at the top of the page.

## Impact

- `frontend/src/pages/TimelinePage.tsx` (header markup, unused `List`/`LayoutGrid` imports)
- `frontend/src/locales/{en,fr}/timeline.json` (remove `title`)
- Spec `timeline-spec-matrix` (one MODIFIED requirement). No API or backend changes.
