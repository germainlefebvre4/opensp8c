# kanban-explore-shortcut-card Specification

## Purpose

A static, permanent placeholder card pinned at the top of the "To Explore" Kanban column that gives users a more discoverable way to start a new exploration, without representing any change or ghost record.

## Requirements

### Requirement: Permanent placeholder card in the To Explore column
The "To Explore" column SHALL always render a placeholder card ("+ New exploration") as the first item of its card list, before any change or ghost cards. This card SHALL be shown regardless of whether the column contains other cards, and regardless of search filtering. The placeholder card is not backed by any change or ghost record and SHALL NOT count toward the column's card count badge.

#### Scenario: To Explore column is empty
- **WHEN** the "To Explore" column has no change or ghost cards
- **THEN** the placeholder card is still displayed as the only item in the column

#### Scenario: To Explore column has existing cards
- **WHEN** the "To Explore" column contains one or more change or ghost cards
- **THEN** the placeholder card is displayed first, above all other cards

#### Scenario: Search filter applied
- **WHEN** the user types a search query that filters out all change cards in the "To Explore" column
- **THEN** the placeholder card remains visible

#### Scenario: Card count badge excludes the placeholder
- **WHEN** the "To Explore" column's card count badge is rendered
- **THEN** the count reflects only real change and ghost cards, not the placeholder card

### Requirement: Placeholder card opens the same exploration panel as the header button
Clicking the placeholder card SHALL trigger the exact same action as the existing "+" button in the "To Explore" column header: it opens the anonymous explore chat panel for a new exploration. The header "+" button SHALL remain unchanged and continue to be displayed alongside the placeholder card.

#### Scenario: Click on the placeholder card
- **WHEN** the user clicks the "+ New exploration" placeholder card
- **THEN** the anonymous explore bottom panel opens, identical to clicking the header "+" button

#### Scenario: Header + button still available
- **WHEN** the "To Explore" column is rendered
- **THEN** the header "+" button is still present and opens the same anonymous explore panel when clicked

### Requirement: Placeholder card is not draggable
The placeholder card SHALL NOT be draggable and SHALL NOT be a valid drop target for other cards.

#### Scenario: Attempt to drag the placeholder card
- **WHEN** the user attempts to drag the placeholder card
- **THEN** the card cannot be picked up (drag is disabled)

#### Scenario: Attempt to drop a card onto the placeholder
- **WHEN** the user drags another card and releases it over the placeholder card
- **THEN** the drop is treated as a drop on the "To Explore" column itself, not on the placeholder card
