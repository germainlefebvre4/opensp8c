## Context

The frontend is a React + Vite + TypeScript SPA with ~100 hardcoded UI strings currently written in French. There is no i18n infrastructure. The goal is to add i18next as the translation layer, make English the default, and expose a manual language selector persisted in localStorage.

## Goals / Non-Goals

**Goals:**
- Add `react-i18next` + `i18next` as the translation layer
- English as default language, French as first translation
- Namespace-scoped JSON locale files (one namespace per feature area)
- Manual `EN | FR` language switcher in the nav bar
- Selected language persisted in `localStorage`

**Non-Goals:**
- Auto-detection via `navigator.language` or `Accept-Language` header
- RTL language support
- Backend API translations
- More than two languages (EN + FR) at this stage
- Number/date/plural formatting (none needed currently)

## Decisions

### i18next over react-intl
`react-i18next` is the standard for React apps without complex ICU message formatting needs. The app has no plural-sensitive strings or date formatting requirements. `react-intl` would add setup overhead without benefit.

### Namespace per feature area (not a single flat file)
8 namespaces (`common`, `navigation`, `kanban`, `detailPanel`, `workspace`, `specs`, `explore`, `dialogs`) keep files small and scoped. Each component only loads what it needs. A single flat file would grow unwieldy and create merge conflicts as the app expands.

### localStorage persistence (not URL param, not cookie)
Language preference is user-specific and should survive navigation. URL params would pollute links shared between users. Cookies require consent overhead. localStorage is the simplest correct choice for a SPA.

### Inline resource loading (not lazy/dynamic import)
With only ~100 strings across 2 languages, there is no meaningful size benefit to lazy-loading locales. Static imports in `i18n.ts` keep the setup simple and avoid async initialization issues.

### `EN | FR` text toggle in the nav bar (not a dropdown, not flags)
The existing nav style is minimal (text-xs, slate palette). A simple two-button toggle fits the aesthetic. Flags imply country rather than language and would require SVG assets. A dropdown is overkill for two options.

## Risks / Trade-offs

- **Missing translations at runtime** → i18next falls back to the key string, so no crash — but untranslated keys surface visibly. Mitigated by completing all translations before shipping.
- **Key drift** — as components evolve, strings may be added without updating locale files. No automated check exists. Mitigated by code review and the tasks list being exhaustive per component.
- **`useTranslation()` namespace coupling** — components must import the correct namespace. A wrong namespace silently returns the key. Mitigated by one namespace per file (no ambiguity about which to use).

## Migration Plan

1. Install dependencies, create `src/i18n.ts`, initialize i18next
2. Create all `locales/en/*.json` + `locales/fr/*.json` files
3. Add `LanguageSwitcher` component, wire into `Layout.tsx`
4. Update each component file one by one (no simultaneous edits needed)
5. No rollback risk — all changes are additive; removing the `useTranslation()` calls reverts to string literals

## Open Questions

None — decisions are fully resolved from the exploration session.
