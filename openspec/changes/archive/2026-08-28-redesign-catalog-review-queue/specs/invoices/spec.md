## MODIFIED Requirements

### Requirement: List invoices

The system SHALL list invoice headers sorted by issue date (newest first).

- The list MUST expose the aggregate catalog linking status of the invoice (`linking_status`).

#### Scenario: View invoices with status

- **WHEN** user views the invoice list
- **THEN** they see the overall catalog linking status for each invoice.

### Requirement: View invoice details

The system SHALL display the full details of an invoice, including header information, tax totals, and line items.

- For each line item, the system MUST display its catalog linking status and any suggested items (with their names and scores).
- The system SHALL allow the user to resolve line links directly from the invoice detail view, utilizing the same search, link, reject, and create provisional capabilities as the catalog review queue.

#### Scenario: Resolve line from invoice detail

- **WHEN** user views an invoice and resolves an unmatched line using the inline catalog tools
- **THEN** the line is linked, the decision is remembered, and the invoice's overall linking status is updated.
