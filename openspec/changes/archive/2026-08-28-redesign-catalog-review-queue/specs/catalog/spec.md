## MODIFIED Requirements

### Requirement: Review queue lists unresolved invoice lines

The system SHALL list invoice lines that are pending catalog resolution (`unmatched` or `suggested`) to allow manual intervention.

- For `suggested` lines, the system MUST include the top matching items and their scores, including the human-readable item names.

#### Scenario: View review queue

- **WHEN** user views the catalog review queue
- **THEN** they see lines needing resolution, including the top suggested items with their names and match scores.

### Requirement: User manually links an invoice line to a catalog item

The system SHALL allow a user to manually select a catalog item for an invoice line and remember the decision.

- The system MUST allow the user to search the catalog by name or SKU to find the correct item.
- If the user selects an item, the line is updated to `linked` via `manual` method.
- The system MUST memorize the mapping (Party, ItemCode, Description -> ItemID) so future identical lines are auto-linked.
- The system MUST lock the line's link to prevent automated overrides.

#### Scenario: Manual link with search

- **WHEN** user searches for an item and selects it to link a line
- **THEN** the line is linked to the selected item, locked, and the decision is remembered.

## ADDED Requirements

### Requirement: User explicitly rejects catalog suggestions for a line

The system SHALL allow a user to reject linking an invoice line, teaching the system to "never match" it to specific items or broadly.

- The system MUST record a `never_match` memory for the line's evidence.
- The line's status becomes `rejected`.

#### Scenario: Reject line

- **WHEN** user chooses to reject a line
- **THEN** the line is marked as rejected and the system remembers not to auto-link it in the future.

### Requirement: User creates a provisional catalog item from an invoice line

The system SHALL allow a user to rapidly create a new provisional catalog item using the data from an unresolved invoice line.

- The system MUST create a new item using the line's description and item code.
- The system MUST link the line to the newly created item.
- The system MUST remember the decision.

#### Scenario: Create provisional item

- **WHEN** user chooses to create a new item from a line
- **THEN** a provisional item is created and the line is linked to it.
