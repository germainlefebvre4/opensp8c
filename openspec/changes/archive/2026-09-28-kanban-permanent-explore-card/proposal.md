# Proposal

## Why

The only way to start a new exploration is the small `+` icon in the "To Explore" column header, which is easy to miss. A permanent, always-visible placeholder card makes the "start a new exploration" gesture more discoverable without changing what it does.

## What Changes

- Add a permanent placeholder card ("+ New exploration") always shown as the first item in the "To Explore" column's card list, even when the column already has cards or a ghost/change is present.
- Clicking the placeholder card triggers the exact same action as the existing header `+` button (opens the anonymous explore chat panel) — no new backend behavior.
- The existing small `+` button in the "To Explore" column header is unchanged and stays alongside the placeholder card.

## Capabilities

### New Capabilities
- `kanban-explore-shortcut-card`: a static, non-draggable placeholder card permanently pinned at the top of the "To Explore" column that opens the same anonymous explore panel as the existing header `+` button.

### Modified Capabilities
_(none — the existing `kanban-board` card-display requirements and the header `+` button behavior are unchanged)_

## Impact

- Frontend only: `frontend/src/components/KanbanColumn.tsx` (render the placeholder card in the "To Explore" column's card list) and its locale strings (`frontend/src/locales/{en,fr}/kanban.json`).
- No backend, API, or data model changes — the placeholder card is not backed by any change or ghost record.
