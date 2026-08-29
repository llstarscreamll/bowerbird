# invoices Specification

## Purpose

Extend electronic invoice persistence so issuers and lines participate in party and catalog identity—optional links and match status—while CUFE, amounts, and denormalized document fields remain the financial source of truth.

## Requirements

### Requirement: Invoice header party link

When an invoice header is persisted, the system MUST attempt to associate `issuer_party_id` per the parties capability. Failure to resolve a party MUST NOT block invoice persistence when the invoice itself is valid.

#### Scenario: Invoice saved with party

- **WHEN** a valid invoice is persisted and the issuer tax id resolves or creates a party
- **THEN** the invoice header stores `issuer_party_id` in addition to denormalized issuer fields

#### Scenario: Invoice saved without party

- **WHEN** a valid invoice is persisted but no party can be resolved
- **THEN** the invoice header is still stored and `issuer_party_id` is null

### Requirement: Invoice line catalog link

Each persisted invoice line MUST support optional `item_id`, link status (`unmatched`, `suggested`, `linked`, `rejected`), link method (`memory`, `hard`, `soft`, `manual`, or unset), suggestion references when soft matchers run, and a user-locked flag. Financial line fields (quantity, prices, taxes, description, item_code) MUST remain authoritative for the document.

#### Scenario: Line linked after resolution

- **WHEN** catalog resolution links a line to an item
- **THEN** the line stores `item_id`, status `linked`, and the method used, without altering line amounts

#### Scenario: Line with suggestions only

- **WHEN** only soft matchers produce candidates
- **THEN** the line status is `suggested`, `item_id` remains null, and suggestion ids are available to the client

### Requirement: Resolution runs after successful invoice write

Catalog and party resolution for a newly persisted invoice MUST run as part of the ingest path such that a successfully stored invoice either has resolution results applied or a recoverable follow-up without losing the invoice. Resolution failures MUST NOT roll back a already-validated CUFE-unique invoice financial write when isolation requires it; they MUST be observable (log/metric/status) for retry.

#### Scenario: Resolution error after invoice insert

- **WHEN** the invoice financial rows commit and catalog resolution fails transiently
- **THEN** the invoice remains queryable and the system records that linking is pending or failed for retry

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

### Requirement: List invoices

The system SHALL list invoice headers sorted by issue date (newest first).

- The list MUST expose the aggregate catalog linking status of the invoice (`linking_status`).

#### Scenario: View invoices with status

- **WHEN** user views the invoice list
- **THEN** they see the overall catalog linking status for each invoice.

### Requirement: View invoice details

The system SHALL display the full details of an invoice, including header information, tax totals, and line items.

- For each line item, the system MUST display its catalog linking status and any suggested items (with their names and scores).
- The system SHALL allow the user to resolve line links directly from the invoice detail view, utilizing the same search, link, reject, and create provisional capabilities as the invoices review queue.

#### Scenario: Resolve line from invoice detail

- **WHEN** user views an invoice and resolves an unmatched line using the inline catalog tools
- **THEN** the line is linked, the decision is remembered, and the invoice's overall linking status is updated

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

### Requirement: No inventory from invoicing

Persisting or linking invoice lines MUST NOT create inventory or stock side effects.

#### Scenario: Ingest complete

- **WHEN** invoice ingest and catalog linking complete for a goods line
- **THEN** no warehouse or stock movement records exist as a result of that ingest
