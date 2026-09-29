# Spec Delta

## MODIFIED Requirements

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
