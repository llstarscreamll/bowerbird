## ADDED Requirements

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

- **WHEN** a catalog item is updated (rename, kind, SKU assignment)
- **THEN** its stored `creation_source` remains unchanged

### Requirement: Filter catalog items by creation source

Authorized tenant users MUST be able to list catalog items filtered by `creation_source` (`manual` or `invoice`).

#### Scenario: Filter manual items

- **WHEN** a user lists catalog items with `creation_source=manual`
- **THEN** only items with `creation_source` `manual` are returned

#### Scenario: Filter invoice items

- **WHEN** a user lists catalog items with `creation_source=invoice`
- **THEN** only items with `creation_source` `invoice` are returned

### Requirement: Catalog UI shows creation source

The PWA catalog master and detail views MUST display the item's `creation_source` in user-facing Spanish labels (e.g. "Manual", "Desde factura").

#### Scenario: Master shows origin column

- **WHEN** a user views the catalog master table
- **THEN** each row shows the creation source label

#### Scenario: Detail shows origin

- **WHEN** a user opens a catalog item detail page
- **THEN** the creation source is visible among the item attributes

## MODIFIED Requirements

### Requirement: Provisional mint on hard identity miss

When hard identity evidence exists (resolved supplier party and non-empty item code) and no memory or alias resolves an item, the system MUST create a provisional item with `creation_source` `invoice`, attach a `supplier_sku` alias scoped to that party, and link the line to the new item with method `hard` (provisional). Description-only lines MUST NOT auto-mint.

#### Scenario: New supplier code mints provisional item

- **WHEN** an invoice line has issuer party P and item code `ABC-1` with no matching alias or memory
- **THEN** the system creates a provisional item (`status` provisional, `creation_source` `invoice`), alias `(supplier_sku, P, ABC-1)`, and links the line

#### Scenario: Empty code does not mint

- **WHEN** an invoice line has an empty item code
- **THEN** the system does not mint an item and leaves the line unmatched (soft suggestions allowed)

### Requirement: Manual create of catalog item

Authorized tenant users MUST be able to create a catalog item without an invoice line. The client MUST supply the item id as a valid ULID. Manual create MUST require a non-empty internal SKU and MUST persist the item with status `confirmed` and `creation_source` `manual`. The system MUST reject create when the internal SKU conflicts with an existing internal SKU in the tenant. The public create contract MUST express the internal SKU as an item attribute and MUST NOT require clients to manage alias resources. Create MUST be atomic with respect to the item and its internal SKU (no persisted item without the required SKU, and no orphan SKU without the item). The create contract MUST NOT include a stockable attribute.

#### Scenario: Successful manual create

- **WHEN** a user creates an item with a client-generated ULID, name, kind, and internal SKU
- **THEN** the item is persisted with status `confirmed`, `creation_source` `manual`, and that internal SKU is available on subsequent reads of the item

#### Scenario: Reject create without internal SKU

- **WHEN** a user attempts to create an item without an internal SKU
- **THEN** the system rejects the request with a validation error

#### Scenario: Reject duplicate internal SKU on create

- **WHEN** a user creates an item whose internal SKU already exists for another item in the tenant
- **THEN** the system rejects the request with a conflict error

#### Scenario: Reject invalid client id

- **WHEN** a user creates an item with an id that is not a valid ULID
- **THEN** the system rejects the request with a validation error

#### Scenario: Reject duplicate item id on create

- **WHEN** a user creates an item with an id that already exists in the tenant
- **THEN** the system rejects the request with a conflict error

### Requirement: List catalog items

Authorized tenant users MUST be able to list catalog items (filter by kind, provisional status, and `creation_source`). List responses MUST include `creation_source`.

#### Scenario: List catalog items

- **WHEN** a user lists catalog items with optional kind, status, or `creation_source` filters
- **THEN** the system returns matching items for the tenant including `creation_source`

### Requirement: View catalog item detail

Authorized tenant users MUST be able to open a single catalog item and see its id, name, kind, status, `creation_source`, timestamps, and internal SKU when present. The detail view of this capability MUST NOT require displaying supplier aliases and MUST NOT expose a stockable attribute.

#### Scenario: Open item detail

- **WHEN** a user opens an item by id
- **THEN** the system returns the item fields including `creation_source` and internal SKU when set

#### Scenario: Item without internal SKU

- **WHEN** a user opens a provisional item that has no internal SKU yet
- **THEN** the system returns the item with an empty/absent internal SKU and its `creation_source`
