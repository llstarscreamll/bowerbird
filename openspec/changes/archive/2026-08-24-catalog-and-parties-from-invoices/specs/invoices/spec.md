## Purpose

Extend electronic invoice persistence so issuers and lines participate in party and catalog identity—optional links and match status—while CUFE, amounts, and denormalized document fields remain the financial source of truth.

## ADDED Requirements

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

Authorized users MUST be able to set or clear a line's item link from the invoice (or review queue), optionally remembering the decision per catalog match memory, and optionally locking the link.

#### Scenario: Manual correct and remember

- **WHEN** a user assigns item I to a line and enables remember
- **THEN** the line becomes `linked` with method `manual` (locked if requested) and future evidence follows catalog memory rules

### Requirement: No inventory from invoicing

Persisting or linking invoice lines MUST NOT create inventory or stock side effects.

#### Scenario: Ingest complete

- **WHEN** invoice ingest and catalog linking complete for a goods line
- **THEN** no warehouse or stock movement records exist as a result of that ingest
