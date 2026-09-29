## Purpose

Core internationalization capability providing multi-language support (English and French) for the application UI, including a language switcher, session persistence, and namespace-scoped translation keys.

## Requirements

### Requirement: Application supports multiple languages
The system SHALL support English (EN) and French (FR) as UI languages. English SHALL be the default language when no preference is stored. Every user-visible string rendered by the application SHALL be sourced from a translation key with both an English and a French value; no page or component SHALL render a hardcoded string in either language.

#### Scenario: Default language on first load
- **WHEN** a user opens the application for the first time (no localStorage entry for language)
- **THEN** the UI SHALL display all text in English

#### Scenario: Locale files cover all UI strings
- **WHEN** the application renders any page or component, including modals, dialogs, tooltips, empty states, and error messages
- **THEN** all user-visible text SHALL be sourced from translation keys, not hardcoded strings, in both the English and French builds

#### Scenario: French translation differs from a copy-pasted English default
- **WHEN** a translation key's French value is reviewed
- **THEN** it SHALL be an actual French translation, not the English string duplicated verbatim (except for terms intentionally kept identical, such as proper nouns or technical labels)

### Requirement: User can switch the UI language
The system SHALL provide a visible language switcher, located in Configuration > Langue, allowing the user to toggle between EN and FR.

#### Scenario: Language switcher is present in the nav bar
- **WHEN** the user views any page
- **THEN** no language switcher SHALL be visible in the top navigation bar - it has moved to Configuration > Langue (see "Language switcher is present in Configuration")

#### Scenario: Language switcher is present in Configuration
- **WHEN** the user opens Configuration > Langue
- **THEN** a language switcher (EN | FR) SHALL be visible

#### Scenario: Switching to French
- **WHEN** the user selects FR in the language switcher
- **THEN** all UI text SHALL update to French immediately without a page reload

#### Scenario: Switching to English
- **WHEN** the user selects EN in the language switcher
- **THEN** all UI text SHALL update to English immediately without a page reload

### Requirement: Language preference is persisted
The system SHALL remember the user's language choice across sessions using localStorage.

#### Scenario: Language persists after reload
- **WHEN** the user selects a language and reloads the page
- **THEN** the previously selected language SHALL be applied on load

#### Scenario: Language persists across navigation
- **WHEN** the user selects a language and navigates between pages (Kanban, Specs, Timeline)
- **THEN** the selected language SHALL remain active

### Requirement: Translation namespaces are scoped by feature
The system SHALL organize translation keys into 8 namespaces: `common`, `navigation`, `kanban`, `detailPanel`, `workspace`, `specs`, `explore`, `dialogs`.

#### Scenario: Component loads only its namespace
- **WHEN** a component calls `useTranslation('kanban')`
- **THEN** it SHALL have access to the kanban namespace keys and the common namespace keys

#### Scenario: Missing translation key falls back to key string
- **WHEN** a translation key is missing from the active language's namespace
- **THEN** the system SHALL display the key string rather than crashing

### Requirement: Translation keys used in code match locale file keys
The system SHALL ensure every translation key referenced via `t()` in the application code resolves to a key that exists, with the same casing, in the locale files for the namespace it is called against.

#### Scenario: Key casing mismatch is caught before release
- **WHEN** a component calls `t('some.key')` and no key with that exact casing exists in the locale files for its namespace
- **THEN** this SHALL be treated as a defect (translation missing or key renamed) rather than silently rendering the raw key or a hardcoded default string

#### Scenario: New column or feature keys are added to every supported locale
- **WHEN** a new translation key is introduced for a UI element
- **THEN** it SHALL be added to both the `en` and `fr` locale files before the feature ships, with no `defaultValue` fallback left as the only translation for a supported locale
