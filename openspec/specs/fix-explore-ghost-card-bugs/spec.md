## Purpose

TBD - Merged from fix-explore-ghost-card-bugs change.

## Requirements

### Requirement: Correct Ghost Card Life Cycle
The backend SHALL correctly parse and propagate ghost card creation and naming events to the WebSocket of the anonymous session, ensuring correct UI rendering and Kanban display regardless of start warning conditions.

#### Scenario: Real-time update
- **WHEN** the first message is sent in an anonymous session
- **THEN** the ghost card is created and visible in the Kanban, and the "Create Change" button is visible in the chat header
