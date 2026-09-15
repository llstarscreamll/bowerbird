# catalog Specification

## Purpose

Tenant commercial catalog of items (goods, services, assets) with multi-identifier aliases and a resolution pipeline that links invoice lines for analytics now and stable identity for future inventory—without creating stock movements.

## Requirements

### Requirement: Item kinds

The system SHALL store each catalog item with a kind of `goods`, `service`, `asset`, or `unknown`. Kind MAY be updated by an authorized user via the item update capability.

#### Scenario: Create item with kind

- **WHEN** a user creates an item with kind `goods`
- **THEN** the item is persisted with that kind

#### Scenario: Default unknown on provisional mint

- **WHEN** the system provisionally mints an item from an invoice line without a confirmed classification
- **THEN** the item kind is `unknown` unless a documented heuristic sets another kind

#### Scenario: Update kind on existing item

- **WHEN** an authorized user updates an item's kind to `service`
- **THEN** the item is persisted with kind `service`

### Requirement: Item aliases

The system SHALL support multiple external identifiers (aliases) per item. An alias MUST include a scheme (`supplier_sku` or `gtin`), a value, `source` (`invoice` or `manual`), and optional scope (party id, required for `supplier_sku`, forbidden for `gtin`). The tenant-canonical internal code is an attribute of the item, not an alias. `BuyersItemIdentification` is not stored as an alias. Within a tenant, the tuple (scheme, scope party id or none, value) MUST be unique.

#### Scenario: Supplier-scoped SKU

- **WHEN** an alias `supplier_sku` with value `MGND3LA/A` is scoped to party P
- **THEN** the same code scoped to a different party MAY identify a different item

#### Scenario: Global GTIN

- **WHEN** an alias `gtin` with value `7701234567890` exists on item I
- **THEN** the same GTIN MUST NOT identify a different item in the tenant regardless of party

#### Scenario: Reject duplicate alias

- **WHEN** creating an alias that duplicates an existing (scheme, scope, value) for another item in the tenant
- **THEN** the system rejects the change with a conflict error that includes the owning item id

### Requirement: Manage item aliases

Authorized tenant users MUST be able to list, add, and remove aliases on a catalog item. An alias MUST have scheme `supplier_sku` or `gtin`, a non-empty value, `source` (`manual` when created by this surface, `invoice` when attached from invoice evidence), and for `supplier_sku` a party id. A `gtin` alias MUST NOT carry a party id. Adding an alias whose (scheme, scope, value) already exists on another item MUST be rejected with a conflict that identifies the owning item id. Removing an alias MUST NOT rewrite locked invoice line links. The item PATCH/update surface MUST remain free of alias fields; alias writes use this dedicated surface.

#### Scenario: List aliases on item detail

- **WHEN** a user opens a catalog item that has a supplier SKU and a GTIN
- **THEN** both aliases are returned with scheme, value, optional party id, and source

#### Scenario: Add supplier SKU alias

- **WHEN** a user adds `supplier_sku` value `ABC-1` scoped to party P on item I
- **THEN** the alias is persisted with `source` `manual` and future lines from P with that seller SKU can hard-match I

#### Scenario: Add GTIN alias

- **WHEN** a user adds `gtin` value `7701234567890` on item I
- **THEN** the alias is persisted with no party scope and `source` `manual`

#### Scenario: Reject alias owned by another item

- **WHEN** a user adds an alias whose tuple already exists on item J
- **THEN** the system rejects the change with a conflict error that includes item J's id

#### Scenario: Remove alias

- **WHEN** a user removes an alias from an item
- **THEN** the alias is deleted and locked invoice lines that used it remain unchanged

### Requirement: Resolution pipeline trust order

When resolving an invoice line to a catalog item, the system MUST apply matchers in this order and stop at the first decisive result: (1) user-locked existing link, (2) match memory for the line's seller-SKU evidence (or description evidence when seller SKU is absent/unusable), (3) hard identity by agreement of all non-empty hard hits, (4) soft matchers as suggestions only. Soft matchers MUST NOT auto-link in this capability.

Hard hits MUST be collected independently from: (a) `buyer_code` equal to a tenant `internal_code`, (b) `gtin` matching an unscoped `gtin` alias, (c) usable `seller_sku` matching a `supplier_sku` alias scoped to the issuer party. If every hard hit points to the same item, the system MUST auto-link with method `hard`. If two or more hard hits point to different items, the system MUST NOT auto-link, MUST NOT mint, and MUST record suggestions for the conflicting items with reason `hard_conflict`. If there are no hard hits, resolution continues to mint (when allowed) or soft suggestions.

#### Scenario: Memory wins over hard alias

- **WHEN** match memory maps seller-SKU evidence E to item A and a hard alias would map the same seller SKU to item B
- **THEN** the system links the line to item A with method `memory`

#### Scenario: Hard alias auto-links

- **WHEN** no lock or memory applies and the only hard hit is a `supplier_sku` alias for the issuer party and usable seller SKU
- **THEN** the system links the line to that item with method `hard`

#### Scenario: Buyer code matches internal code

- **WHEN** no lock or memory applies and `buyer_code` equals item I's `internal_code` and no other hard hit disagrees
- **THEN** the system links the line to item I with method `hard`

#### Scenario: GTIN hard match

- **WHEN** no lock or memory applies and the line GTIN matches a `gtin` alias on item I and no other hard hit disagrees
- **THEN** the system links the line to item I with method `hard`

#### Scenario: Hard identity conflict

- **WHEN** `buyer_code` matches item A and `seller_sku` matches item B
- **THEN** the line is `suggested` with both items, reason `hard_conflict`, and no mint occurs

#### Scenario: Soft match only suggests

- **WHEN** soft matchers return candidates with scores and there is no lock, memory, agreed hard hit, or mint
- **THEN** the system records suggestions and leaves the line unlinked (or keeps prior unlocked state) without auto-applying a soft candidate

### Requirement: Provisional mint on hard identity miss

When no lock, memory, or agreed hard hit applies, the system MUST mint a provisional item only when the issuer party is resolved and the seller SKU is usable. The minted item MUST have `creation_source` `invoice`, a `supplier_sku` alias scoped to that party with `source` `invoice`, and any valid line GTIN attached as a `gtin` alias with `source` `invoice` on the same item. The line MUST link to the new item with method `hard`. Lines without a usable seller SKU MUST NOT auto-mint (including buyer-code-only, GTIN-only, description-only, and low-entropy seller SKU lines). If attaching a GTIN conflicts with another item, mint MUST NOT proceed; the line MUST be `suggested` with `hard_conflict`.

#### Scenario: New supplier code mints provisional item

- **WHEN** an invoice line has issuer party P and usable seller SKU `ABC-1` with no matching alias, memory, or other hard hit
- **THEN** the system creates a provisional item (`status` provisional, `creation_source` `invoice`), alias `(supplier_sku, P, ABC-1)` with `source` `invoice`, and links the line

#### Scenario: Mint attaches GTIN

- **WHEN** the same miss also has a classified GTIN that is not aliased to another item
- **THEN** the minted item also receives a `gtin` alias with `source` `invoice`

#### Scenario: Empty or unusable seller SKU does not mint

- **WHEN** an invoice line has an empty seller SKU, only a buyer code, only a GTIN, or an unusable seller SKU
- **THEN** the system does not mint an item and leaves the line unmatched or suggested (soft or conflict) without a new provisional

### Requirement: Low-entropy seller SKU is not auto-learned

The system MUST treat a seller SKU as unusable for auto-mint and for auto-creating a `supplier_sku` alias when it is empty after normalize, shorter than three characters, or equal (case-insensitive) to a generic token (`1`, `01`, `001`, `n/a`, `na`, `serv`, `servicio`, `item`). Authorized users MAY still add that value as a manual alias. Match memory MUST NOT be keyed by an unusable seller SKU (description fingerprint MAY still apply).

#### Scenario: Generic code does not mint

- **WHEN** an invoice line has party P and seller SKU `1` with no other hard hits
- **THEN** the system does not mint an item and does not persist a `supplier_sku` alias for `1`

#### Scenario: Manual alias allowed for generic code

- **WHEN** a user adds `supplier_sku` `1` scoped to party P on item I
- **THEN** the alias is persisted and later lines from P with seller SKU `1` may hard-match I

### Requirement: Match memory for user decisions

When a user confirms, corrects, or rejects a line-to-item association and requests that the decision be remembered, the system MUST persist match memory keyed by evidence (party id when known, usable seller SKU when present, and/or normalized description fingerprint) so future lines with the same evidence reuse that decision. User memory MUST outrank algorithmic hard and soft matchers. GTIN and buyer code MUST NOT be the memory evidence key; they participate only as hard identity.

#### Scenario: Remember positive link

- **WHEN** a user links a line (party P, seller SKU `X`) to item I and opts to remember
- **THEN** a later line with party P and seller SKU `X` auto-links to item I via method `memory`

#### Scenario: Correct bad match

- **WHEN** a line was linked to item A and the user remaps it to item B with remember
- **THEN** future matching evidence for that decision resolves to item B, not A

#### Scenario: Negative memory

- **WHEN** a user marks that evidence must not auto-match a given item
- **THEN** hard auto-link to that item for the same seller-SKU evidence is suppressed until a new positive memory or alias overrides it

### Requirement: User-locked links

A user-confirmed link marked locked MUST NOT be overwritten by later automatic resolution or invoice reprocessing.

#### Scenario: Reprocess preserves lock

- **WHEN** an invoice is reprocessed and a line has a locked user link to item I
- **THEN** the link to item I remains unchanged

### Requirement: Catalog item creation source

The system SHALL persist an immutable `creation_source` on each catalog item with allowed values `manual` and `invoice`. The value MUST be set only at creation time and MUST NOT change on update, rename, kind change, or confirmation (provisional → confirmed). `creation_source` records the first birth channel of the item.

#### Scenario: Manual create sets manual source

- **WHEN** a user creates a catalog item via the manual API/UI path
- **THEN** the persisted item has `creation_source` `manual`

#### Scenario: Provisional mint sets invoice source

- **WHEN** the system provisionally mints an item from invoice line evidence
- **THEN** the persisted item has `creation_source` `invoice`

#### Scenario: Confirm provisional keeps invoice source

- **WHEN** a user confirms a provisional item that was minted from an invoice
- **THEN** the item becomes `confirmed` but `creation_source` remains `invoice`

#### Scenario: Update does not change creation source

- **WHEN** a catalog item is updated (rename, kind, internal code assignment)
- **THEN** its stored `creation_source` remains unchanged

### Requirement: Stock side effects forbidden

Catalog resolution and invoice linking MUST NOT create inventory quantities, warehouse records, or stock movements.

#### Scenario: Linked goods line

- **WHEN** a goods-kind item is linked from an invoice line
- **THEN** no stock balance or inventory movement is created

### Requirement: List catalog items

Authorized tenant users MUST be able to list catalog items (filter by kind, provisional status, and `creation_source`). List responses MUST include `creation_source`.

#### Scenario: List catalog items

- **WHEN** a user lists catalog items with optional kind, status, or `creation_source` filters
- **THEN** the system returns matching items for the tenant including `creation_source`

### Requirement: Filter catalog items by creation source

Authorized tenant users MUST be able to list catalog items filtered by `creation_source` (`manual` or `invoice`).

#### Scenario: Filter manual items

- **WHEN** a user lists catalog items with `creation_source=manual`
- **THEN** only items with `creation_source` `manual` are returned

#### Scenario: Filter invoice items

- **WHEN** a user lists catalog items with `creation_source=invoice`
- **THEN** only items with `creation_source` `invoice` are returned

### Requirement: Manual create of catalog item

Authorized tenant users MUST be able to create a catalog item without an invoice line. The client MUST supply the item id as a valid ULID. Manual create MUST require a non-empty internal code and MUST persist the item with status `confirmed` and `creation_source` `manual`. The system MUST reject create when the internal code conflicts with an existing internal code in the tenant. The public create contract MUST express the internal code as an item attribute (`internal_code`) persisted on the catalog item (not as an alias resource). Create MUST be atomic with respect to the item and its internal code. The create contract MUST NOT include a stockable attribute.

#### Scenario: Successful manual create

- **WHEN** a user creates an item with a client-generated ULID, name, kind, and internal code
- **THEN** the item is persisted with status `confirmed`, `creation_source` `manual`, and that internal code is available on subsequent reads of the item

#### Scenario: Reject create without internal code

- **WHEN** a user attempts to create an item without an internal code
- **THEN** the system rejects the request with a validation error

#### Scenario: Reject duplicate internal code on create

- **WHEN** a user creates an item whose internal code already exists for another item in the tenant
- **THEN** the system rejects the request with a conflict error

#### Scenario: Reject invalid client id

- **WHEN** a user creates an item with an id that is not a valid ULID
- **THEN** the system rejects the request with a validation error

#### Scenario: Reject duplicate item id on create

- **WHEN** a user creates an item with an id that already exists in the tenant
- **THEN** the system rejects the request with a conflict error

### Requirement: Update catalog item

Authorized tenant users MUST be able to rename an item and change its kind. Confirming a provisional item (transition to `confirmed`) MUST require an internal code (already assigned or provided in the same update). The system MUST NOT allow reassigning an internal code once it is set. The system MUST NOT expose supplier-alias management on the item update surface of this capability. First assignment of an internal code (including as part of confirmation) MUST be persisted on the catalog item in the same update. Domain invariants for confirmation and internal-code immutability MUST be enforced in the catalog item model (not only at the transport layer). The update contract MUST NOT include a stockable attribute.

#### Scenario: Rename and change kind

- **WHEN** a user renames an item and changes its kind
- **THEN** the changes are persisted and the internal code (if any) remains unchanged

#### Scenario: Confirm provisional with internal code

- **WHEN** a user confirms a provisional item and supplies a new internal code (or one already exists)
- **THEN** the item becomes `confirmed` and the internal code is stored if it was missing

#### Scenario: Reject confirm without internal code

- **WHEN** a user attempts to confirm a provisional item that has no internal code and does not provide one
- **THEN** the system rejects the request with a validation error

#### Scenario: Reject reassignment of internal code

- **WHEN** a user attempts to assign a different internal code after one was already set
- **THEN** the system rejects the request with a validation or conflict error

### Requirement: View catalog item detail

Authorized tenant users MUST be able to open a single catalog item and see its id, name, kind, status, `creation_source`, timestamps, internal code when present, and its aliases (scheme, value, party id when scoped, source). The detail view MUST NOT expose a stockable attribute.

#### Scenario: Open item detail

- **WHEN** a user opens an item by id
- **THEN** the system returns the item fields including `creation_source`, internal code when set, and the alias list

#### Scenario: Item without internal code

- **WHEN** a user opens a provisional item that has no internal code yet
- **THEN** the system returns the item with an empty/absent internal code, its `creation_source`, and any aliases

### Requirement: Catalog master navigation for manual CRUD

The catalog master MUST allow authorized users to start creating an item and to open an item's detail (and from detail, edit). Create and edit MUST share the same form component within the catalog feature. List responses used by the master SHOULD include the internal code when present so users can recognize items by code.

#### Scenario: Create from master

- **WHEN** a user chooses to create a new item from the catalog master
- **THEN** they reach the create form and can submit a manual create

#### Scenario: Open detail from master

- **WHEN** a user selects an item in the catalog master
- **THEN** they reach the item detail view

### Requirement: Catalog UI shows creation source

The PWA catalog master and detail views MUST display the item's `creation_source` in user-facing Spanish labels (e.g. "Manual", "Desde factura").

#### Scenario: Master shows origin column

- **WHEN** a user views the catalog master table
- **THEN** each row shows the creation source label

#### Scenario: Detail shows origin

- **WHEN** a user opens a catalog item detail page
- **THEN** the creation source is visible among the item attributes

### Requirement: Catalog item has no stockable flag

The catalog item model and public item contract MUST NOT include a stockable (or equivalent inventory) flag. Stockability concerns are out of scope for the catalog master capability until an inventory capability defines them.

#### Scenario: Item resource omits stockable

- **WHEN** a client reads or writes a catalog item via the item API
- **THEN** the payload does not include a stockable attribute
