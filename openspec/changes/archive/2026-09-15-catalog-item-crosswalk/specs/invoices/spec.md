## ADDED Requirements

### Requirement: Structured UBL line identifiers

When parsing a DIAN UBL 2.1 invoice line, the system MUST persist three independent identifier fields and MUST NOT collapse them into a single `item_code`:

- `buyer_code` from `cac:BuyersItemIdentification/cbc:ID` (trimmed; empty if absent)
- `seller_sku` from `cac:SellersItemIdentification/cbc:ID` (trimmed). If that value is empty and `StandardItemIdentification` is present but is not a classified GTIN, the system MUST store that standard value as `seller_sku`
- `gtin` only when `cac:StandardItemIdentification/cbc:ID` (digits after stripping spaces) has length 8, 12, 13, or 14 and consists entirely of digits; otherwise `gtin` MUST be empty. The system MUST NOT require a GS1 check digit

Financial amounts and description remain authoritative. Clients MUST read and write these three fields; `item_code` MUST NOT appear on the line resource.

#### Scenario: Line with buyer, seller, and GTIN

- **WHEN** a UBL line has BuyersItemIdentification `INT-9`, SellersItemIdentification `MGND3LA/A`, and StandardItemIdentification `7701234567890`
- **THEN** the stored line has `buyer_code` `INT-9`, `seller_sku` `MGND3LA/A`, and `gtin` `7701234567890`

#### Scenario: Standard id that is not a GTIN

- **WHEN** a UBL line has matching Sellers and Standard values `MGND3LA/A`
- **THEN** `seller_sku` is `MGND3LA/A` and `gtin` is empty

#### Scenario: Standard id used as seller fallback

- **WHEN** a UBL line has empty SellersItemIdentification and StandardItemIdentification `ABC-1` (not a GTIN)
- **THEN** `seller_sku` is `ABC-1` and `gtin` is empty

### Requirement: Unlock catalog line link

Authorized users MUST be able to unlock a locked invoice line catalog link. Unlock MUST NOT change `item_id`, link status, or method. After unlock, a subsequent decision MAY change the link. Automatic resolution and invoice reprocessing MUST still preserve links that remain locked.

#### Scenario: Unlock then relink

- **WHEN** a user unlocks a locked line and then links it to a different item
- **THEN** the line becomes linked to the new item with method `manual` and lock according to the new decision

#### Scenario: Unlock does not relink by itself

- **WHEN** a user unlocks a line linked to item I
- **THEN** the line stays linked to item I until a new decision or unlocked reprocessing runs

## MODIFIED Requirements

### Requirement: Invoice line catalog link

Each persisted invoice line MUST support optional `item_id`, link status (`unmatched`, `suggested`, `linked`, `rejected`), link method (`memory`, `hard`, `soft`, `manual`, or unset), suggestion references when soft matchers run, and a user-locked flag. Financial line fields (quantity, prices, taxes, description, `buyer_code`, `seller_sku`, `gtin`) MUST remain authoritative for the document. The line MUST NOT persist a collapsed `item_code`.

#### Scenario: Line linked after resolution

- **WHEN** catalog resolution links a line to an item
- **THEN** the line stores `item_id`, status `linked`, and the method used, without altering line amounts or identifier fields

#### Scenario: Line with suggestions only

- **WHEN** only soft matchers produce candidates
- **THEN** the line status is `suggested`, `item_id` remains null, and suggestion ids are available to the client

### Requirement: Manual link from invoice UI

Authorized users MUST be able to set or clear a line's item link from the invoice detail view or the invoices review queue, optionally remembering the decision per catalog match memory, and optionally locking the link.

- The system SHALL allow the user to resolve line links using the same search, link, reject, create provisional, and unlock capabilities as the invoices review queue.
- For each line item, the system MUST display `buyer_code`, `seller_sku`, `gtin`, catalog linking status, and any suggested items (with their names and scores).
- The client MUST send `remember` and `lock` explicitly. The UI MUST default both to enabled and MUST preview which aliases (usable seller SKU and classified GTIN) will be attached when remember is enabled.

#### Scenario: Manual correct and remember

- **WHEN** a user assigns item I to a line and enables remember
- **THEN** the line becomes `linked` with method `manual` (locked if requested) and future seller-SKU evidence follows catalog memory rules

#### Scenario: Resolve line from invoice detail

- **WHEN** user views an invoice and resolves an unmatched line using the inline catalog tools with remember enabled
- **THEN** the line is linked, the decision is remembered, and the invoice's overall linking status is updated

#### Scenario: Link without remember

- **WHEN** a user links a line to item I with remember disabled
- **THEN** the line is linked and the system does not persist match memory or new aliases from that decision

### Requirement: View invoice details

The system SHALL display the full details of an invoice, including header information, tax totals, and line items.

- For each line item, the system MUST display `buyer_code`, `seller_sku`, `gtin`, its catalog linking status, and any suggested items (with their names and scores).
- The system SHALL allow the user to resolve line links directly from the invoice detail view, utilizing the same search, link, reject, create provisional, and unlock capabilities as the invoices review queue.

#### Scenario: Resolve line from invoice detail

- **WHEN** user views an invoice and resolves an unmatched line using the inline catalog tools with remember enabled
- **THEN** the line is linked, the decision is remembered, and the invoice's overall linking status is updated

### Requirement: Review queue lists unresolved invoice lines

Authorized tenant users MUST be able to list invoice lines that are unmatched or have soft suggestions pending review, across invoices for the tenant.

- For `suggested` lines, the system MUST include the top matching items and their scores, including the human-readable item names.
- Each queue row MUST include `buyer_code`, `seller_sku`, and `gtin`.
- The review queue MUST be exposed as part of the invoices capability (API and product navigation), not as a catalog master concern.

#### Scenario: View review queue

- **WHEN** user opens the invoice match review queue
- **THEN** they see lines needing resolution (`unmatched` or `suggested`), including the top suggested items with their names and match scores, and the line identifier fields

#### Scenario: Review queue entry from invoices

- **WHEN** user is on the invoices master
- **THEN** they can navigate to the review queue without going through the catalog master

### Requirement: User applies catalog link decisions on invoice lines

Authorized users MUST be able to apply a link decision on an invoice line from the invoice detail view or the review queue: link to an existing item, reject (`never_match`), create a provisional catalog item from the line evidence, or unlock a locked link.

- The system MUST allow the user to search the catalog by name, internal code, supplier SKU, or GTIN when linking to an existing item.
- On link: the line MUST become `linked` with method `manual`, MUST be locked only when requested, and when remember is requested the system MUST persist catalog match memory keyed by usable seller SKU (when present) and MUST attach all valid line aliases (`supplier_sku` for usable seller SKU, `gtin` for classified GTIN) to the chosen item with `source` `invoice`. Alias conflicts on another item MUST reject the decision with a conflict that includes the owning item id; the line MUST NOT be mutated.
- On reject: the line status MUST become `rejected` and, when remember is requested, the system MUST record `never_match` memory for the seller-SKU or description evidence.
- On create provisional: the system MUST create a provisional catalog item from the line's description and usable seller SKU (same mint rules as ingest), attach valid aliases, link the line, remember the decision, and lock the link.
- On unlock: the lock flag MUST clear without changing the linked item.
- Applying a decision MUST update the invoice header aggregate `linking_status` accordingly.
- Persisting link state (`item_id`, `link_status`, `link_method`, `link_locked`, suggestions clearance as applicable, header `linking_status`) MUST be owned by the invoices capability.

#### Scenario: Manual link with search

- **WHEN** user searches for an item and selects it to link a line with remember and lock
- **THEN** the line is linked to the selected item, locked, the decision is remembered, valid aliases are attached, and the invoice linking status is updated

#### Scenario: Reject line

- **WHEN** user chooses to reject a line with remember
- **THEN** the line is marked as rejected and the system remembers not to auto-link that evidence in the future

#### Scenario: Create provisional item from line

- **WHEN** user chooses to create a new item from a line that has a usable seller SKU
- **THEN** a provisional catalog item is created, aliases from the line are attached, the line is linked to it (locked), the decision is remembered, and the invoice linking status is updated

#### Scenario: Remember attaches GTIN and seller SKU

- **WHEN** a user links a line that has usable seller SKU `ABC-1` and GTIN `7701234567890` to item I with remember
- **THEN** item I has `supplier_sku` (issuer party, `ABC-1`) and `gtin` (`7701234567890`) with `source` `invoice`
