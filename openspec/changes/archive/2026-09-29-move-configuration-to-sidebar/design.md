# Design

## Context

`frontend/src/components/Layout.tsx` currently renders two independent navigation surfaces: `WorkspaceSidebar` (left panel, workspace-scoped: default agent selector + project list, collapsible between `w-64` and `w-8`) and a top `<nav>` bar that mixes workspace tabs (Kanban/Specs/Timeline/Agents/Réglages) with two global controls that don't belong to any workspace tab: the `Configuration` link (styled as a separate pill, `Layout.tsx:75-87`) and `LanguageSwitcher` (`Layout.tsx:88`). See `proposal.md` - Why for the motivation to consolidate the two global controls into the left panel.

`WorkspaceSidebar` currently has one collapse behavior: everything below the header toggle (`w-full` content block) is wrapped in a single `div` with `opacity-0 pointer-events-none` when collapsed (`WorkspaceSidebar.tsx:60`). That block must be split so the new Configuration entry stays interactive while collapsed, but the agent selector / project list stay hidden.

`ConfigurationPage.tsx` already has a working two-tab pattern (`agents` | `cli`, local `useState`) that the new `Langue` tab follows directly - no new pattern needed.

## Goals / Non-Goals

**Goals:**
- Move the `Configuration` entry point from the top nav into `WorkspaceSidebar`, above the "Projets" block, active/highlighted when `/configuration` is the current route.
- Keep the `Configuration` entry reachable (icon only) when the sidebar is collapsed, without making the rest of the collapsed content interactive.
- Move `LanguageSwitcher` out of the top nav into a new `Langue` tab in `ConfigurationPage`, reusing the component as-is.

**Non-Goals:**
- No change to `LanguageSwitcher`'s internal behavior (still `i18n.changeLanguage`, still reads `i18n.language`) - only its mount location changes.
- No change to how workspaces are selected/added/removed, or to the sidebar's `w-64`/`w-8` collapse mechanics themselves.
- No new global entries beyond `Configuration` - the sidebar becoming "the place for global nav" is a placement decision for this one entry, not a new extensible menu system (no need to build a generic "global nav items" abstraction for a single item).

## Decisions

- **Entry placement: top of the sidebar, above "Projets".** Chosen over a footer/gear-icon placement (confirmed with the user) because the sidebar's stated role going forward is "menu bar for global features", which reads as primary nav rather than a secondary settings shortcut - top placement also leaves room for any future global entry to sit in the same block without competing with the footer's "add project" action.
- **Collapsed-state split.** `WorkspaceSidebar`'s single collapsible content `div` becomes two independent blocks: (1) the `Configuration` link, always rendered and clickable, icon-only (no label) when `isOpen` is false; (2) the existing `AgentSelector` + project list + add-project form, still wrapped in the current `opacity-0 pointer-events-none` block when collapsed. This is the minimal change that satisfies "reachable when collapsed" without touching the collapse animation itself.
- **Active state.** The `Configuration` link uses the same `useLocation().pathname === '/configuration'` check already computed in `Layout.tsx` (`isConfigurationRoute`), passed down to `WorkspaceSidebar` as a prop, rather than duplicating route-matching logic inside the sidebar.
- **No sidebar auto-open on collapsed click.** Clicking the collapsed Configuration icon navigates to `/configuration` but does not call `onToggle` - consistent with the confirmed scenario that the panel's open/closed state is user-controlled and orthogonal to navigation.
- **Langue tab reuses `LanguageSwitcher` unmodified.** `ConfigurationPage`'s tab union grows from `'agents' | 'cli'` to `'agents' | 'cli' | 'language'`; the new tab body is just `<LanguageSwitcher />`. No new translation keys needed for the switcher itself (its EN/FR labels are hardcoded language codes, not translated strings), only for the new tab label (`configuration.json` → `tabs.language`) and its accessible name if needed.
- **Top nav simplification.** `Layout.tsx`'s top `<nav>` keeps only the workspace tab `NavLink`s; the `Configuration` `NavLink` and `<LanguageSwitcher />` are removed from it.

## Risks / Trade-offs

- [Removing the top-nav Configuration link changes a URL entry point users may have muscle memory for] → The route `/configuration` itself is unchanged, only the link's location moves; no redirect/back-compat needed since it's the same route.
- [Splitting the sidebar's collapsible content into two blocks could visually misalign the collapsed icon column if not tested in both themes/widths] → Verify manually in the browser (collapsed and expanded) as part of implementation, per the project's standard "test the golden path in a browser" practice for frontend changes.
