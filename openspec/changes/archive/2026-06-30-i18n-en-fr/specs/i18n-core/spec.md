## ADDED Requirements

### Requirement: Application supports multiple languages
The system SHALL support English (EN) and French (FR) as UI languages. English SHALL be the default language when no preference is stored.

#### Scenario: Default language on first load
- **WHEN** a user opens the application for the first time (no localStorage entry for language)
- **THEN** the UI SHALL display all text in English

#### Scenario: Locale files cover all UI strings
- **WHEN** the application renders any component
- **THEN** all user-visible text SHALL be sourced from translation keys, not hardcoded strings

### Requirement: User can switch the UI language
The system SHALL provide a visible language switcher allowing the user to toggle between EN and FR.

#### Scenario: Language switcher is present in the nav bar
- **WHEN** the user views any page
- **THEN** a language switcher (EN | FR) SHALL be visible in the top navigation bar

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
