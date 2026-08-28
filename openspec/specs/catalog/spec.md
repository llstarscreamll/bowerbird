# catalog Specification

## Purpose

Tenant commercial catalog of items (goods, services, assets) with multi-identifier aliases and a resolution pipeline that links invoice lines for analytics now and stable identity for future inventory—without creating stock movements.

## Requirements

### Requirement: Item kinds

The system SHALL store each catalog item with a kind of `goods`, `service`, `asset`, or `unknown`. Kind MAY be updated by an authorized user.

#### Scenario: Create item with kind

- **WHEN** a user creates an item with kind `goods`
- **THEN** the item is persisted with that kind

#### Scenario: Default unknown on provisional mint

- **WHEN** the system provisionally mints an item from an invoice line without a confirmed classification
- **THEN** the item kind is `unknown` unless a documented heuristic sets another kind

### Requirement: Item aliases

The system SHALL support multiple external identifiers (aliases) per item. An alias MUST include a scheme (at least `supplier_sku` and `internal_sku`), a value, and optional scope (party id for supplier-scoped codes). Within a tenant, the tuple (scheme, scope party id or none, value) MUST be unique.

#### Scenario: Supplier-scoped SKU

- **WHEN** an alias `supplier_sku` with value `MGND3LA/A` is scoped to party P
- **THEN** the same code scoped to a different party MAY identify a different item

#### Scenario: Reject duplicate alias

- **WHEN** creating an alias that duplicates an existing (scheme, scope, value) for another item in the tenant
- **THEN** the system rejects the change with a conflict error

### Requirement: Resolution pipeline trust order

When resolving an invoice line to a catalog item, the system MUST apply matchers in this order and stop at the first decisive result: (1) user-locked existing link, (2) match memory for the line's evidence, (3) hard identity via supplier-scoped alias on item code, (4) soft matchers as suggestions only. Soft matchers MUST NOT auto-link in this capability.

#### Scenario: Memory wins over hard alias

- **WHEN** match memory maps evidence E to item A and a hard alias would map the same code to item B
- **THEN** the system links the line to item A with method `memory`

#### Scenario: Hard alias auto-links

- **WHEN** no lock or memory applies and a `supplier_sku` alias exists for the issuer party and non-empty item code
- **THEN** the system links the line to that item with method `hard`

#### Scenario: Soft match only suggests

- **WHEN** soft matchers return candidates with scores
- **THEN** the system records suggestions and leaves the line unlinked (or keeps prior unlocked state) without auto-applying a soft candidate

### Requirement: Provisional mint on hard identity miss

When hard identity evidence exists (resolved supplier party and non-empty item code) and no memory or alias resolves an item, the system MUST create a provisional item, attach a `supplier_sku` alias scoped to that party, and link the line to the new item with method `hard` (provisional). Description-only lines MUST NOT auto-mint.

#### Scenario: New supplier code mints provisional item

- **WHEN** an invoice line has issuer party P and item code `ABC-1` with no matching alias or memory
- **THEN** the system creates a provisional item (status provisional), alias `(supplier_sku, P, ABC-1)`, and links the line

#### Scenario: Empty code does not mint

- **WHEN** an invoice line has an empty item code
- **THEN** the system does not mint an item and leaves the line unmatched (soft suggestions allowed)

### Requirement: Match memory for user decisions

When a user confirms, corrects, or rejects a line-to-item association and requests that the decision be remembered, the system MUST persist match memory keyed by evidence (party id when known, item code when present, and/or normalized description fingerprint) so future lines with the same evidence reuse that decision. User memory MUST outrank algorithmic hard and soft matchers.

#### Scenario: Remember positive link

- **WHEN** a user links a line (party P, code `X`) to item I and opts to remember
- **THEN** a later line with party P and code `X` auto-links to item I via method `memory`

#### Scenario: Correct bad match

- **WHEN** a line was linked to item A and the user remaps it to item B with remember
- **THEN** future matching evidence for that decision resolves to item B, not A

#### Scenario: Negative memory

- **WHEN** a user marks that evidence must not auto-match a given item
- **THEN** hard auto-link to that item for the same evidence is suppressed until a new positive memory or alias overrides it

### Requirement: User-locked links

A user-confirmed link marked locked MUST NOT be overwritten by later automatic resolution or invoice reprocessing.

#### Scenario: Reprocess preserves lock

- **WHEN** an invoice is reprocessed and a line has a locked user link to item I
- **THEN** the link to item I remains unchanged

### Requirement: Stock side effects forbidden

Catalog resolution and invoice linking MUST NOT create inventory quantities, warehouse records, or stock movements.

#### Scenario: Linked goods line

- **WHEN** a goods-kind item is linked from an invoice line
- **THEN** no stock balance or inventory movement is created

### Requirement: List items and review queue

Authorized tenant users MUST be able to list catalog items (filter by kind and provisional status) and list invoice lines that are unmatched or have soft suggestions pending review.

- For `suggested` lines, the system MUST include the top matching items and their scores, including the human-readable item names.

#### Scenario: Review unmatched lines

- **WHEN** a user opens the match review queue
- **THEN** the system returns lines with status unmatched or suggested for the tenant

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
