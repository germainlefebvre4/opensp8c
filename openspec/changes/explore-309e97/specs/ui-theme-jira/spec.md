## ADDED Requirements

### Requirement: Jira-style Neutral Palette
The system SHALL use Atlassian Design System neutral colors for all Tailwind `slate` utility classes.

#### Scenario: Application rendering neutral colors
- **WHEN** the application renders components using Tailwind `slate` classes (e.g., `slate-50`, `slate-900`)
- **THEN** the colors displayed MUST match the corresponding Atlassian Neutral hex codes (N10 through N900).

### Requirement: Jira-style Brand Palette
The system SHALL use Atlassian Design System blue colors for all Tailwind `blue` utility classes.

#### Scenario: Application rendering brand colors
- **WHEN** the application renders components using Tailwind `blue` classes (e.g., `blue-50`, `blue-600`)
- **THEN** the colors displayed MUST match the corresponding Atlassian Blue hex codes (B50 through B500).
