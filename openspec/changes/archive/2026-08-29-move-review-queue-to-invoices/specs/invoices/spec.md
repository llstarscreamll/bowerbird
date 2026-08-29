## ADDED Requirements

### Requirement: Review queue lists unresolved invoice lines

Authorized tenant users MUST be able to list invoice lines that are unmatched or have soft suggestions pending review, across invoices for the tenant.

- For `suggested` lines, the system MUST include the top matching items and their scores, including the human-readable item names.
- The review queue MUST be exposed as part of the invoices capability (API and product navigation), not as a catalog master concern.

#### Scenario: View review queue

- **WHEN** user opens the invoice match review queue
- **THEN** they see lines needing resolution (`unmatched` or `suggested`), including the top suggested items with their names and match scores

#### Scenario: Review queue entry from invoices

- **WHEN** user is on the invoices master
- **THEN** they can navigate to the review queue without going through the catalog master

### Requirement: User applies catalog link decisions on invoice lines

Authorized users MUST be able to apply a link decision on an invoice line from the invoice detail view or the review queue: link to an existing item, reject (`never_match`), or create a provisional catalog item from the line evidence.

- The system MUST allow the user to search the catalog by name or SKU when linking to an existing item.
- On link: the line MUST become `linked` with method `manual`, MUST be locked when requested, and when remember is requested the system MUST persist catalog match memory (and supplier alias when hard identity evidence exists) so future identical evidence auto-links.
- On reject: the line status MUST become `rejected` and, when remember is requested, the system MUST record `never_match` memory for the evidence.
- On create provisional: the system MUST create a provisional catalog item from the line's description/code, link the line to it, remember the decision, and lock the link.
- Applying a decision MUST update the invoice header aggregate `linking_status` accordingly.
- Persisting link state (`item_id`, `link_status`, `link_method`, `link_locked`, suggestions clearance as applicable, header `linking_status`) MUST be owned by the invoices capability.

#### Scenario: Manual link with search

- **WHEN** user searches for an item and selects it to link a line with remember and lock
- **THEN** the line is linked to the selected item, locked, the decision is remembered, and the invoice linking status is updated

#### Scenario: Reject line

- **WHEN** user chooses to reject a line with remember
- **THEN** the line is marked as rejected and the system remembers not to auto-link that evidence in the future

#### Scenario: Create provisional item from line

- **WHEN** user chooses to create a new item from a line
- **THEN** a provisional catalog item is created, the line is linked to it (locked), the decision is remembered, and the invoice linking status is updated

## MODIFIED Requirements

### Requirement: Manual link from invoice UI

Authorized users MUST be able to set or clear a line's item link from the invoice detail view or the invoices review queue, optionally remembering the decision per catalog match memory, and optionally locking the link.

- The system SHALL allow the user to resolve line links using the same search, link, reject, and create provisional capabilities as the invoices review queue.
- For each line item, the system MUST display its catalog linking status and any suggested items (with their names and scores).

#### Scenario: Manual correct and remember

- **WHEN** a user assigns item I to a line and enables remember
- **THEN** the line becomes `linked` with method `manual` (locked if requested) and future evidence follows catalog memory rules

#### Scenario: Resolve line from invoice detail

- **WHEN** user views an invoice and resolves an unmatched line using the inline catalog tools
- **THEN** the line is linked, the decision is remembered, and the invoice's overall linking status is updated

### Requirement: View invoice details

The system SHALL display the full details of an invoice, including header information, tax totals, and line items.

- For each line item, the system MUST display its catalog linking status and any suggested items (with their names and scores).
- The system SHALL allow the user to resolve line links directly from the invoice detail view, utilizing the same search, link, reject, and create provisional capabilities as the invoices review queue.

#### Scenario: Resolve line from invoice detail

- **WHEN** user views an invoice and resolves an unmatched line using the inline catalog tools
- **THEN** the line is linked, the decision is remembered, and the invoice's overall linking status is updated
