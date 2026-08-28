## ADDED Requirements

### Requirement: Manual create of catalog item

Authorized tenant users MUST be able to create a catalog item without an invoice line. The client MUST supply the item id as a valid ULID. Manual create MUST require a non-empty internal SKU and MUST persist the item with status `confirmed`. The system MUST reject create when the internal SKU conflicts with an existing internal SKU in the tenant. The public create contract MUST express the internal SKU as an item attribute and MUST NOT require clients to manage alias resources. Create MUST be atomic with respect to the item and its internal SKU (no persisted item without the required SKU, and no orphan SKU without the item). The create contract MUST NOT include a stockable attribute.

#### Scenario: Successful manual create

- **WHEN** a user creates an item with a client-generated ULID, name, kind, and internal SKU
- **THEN** the item is persisted with status `confirmed` and that internal SKU is available on subsequent reads of the item

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

### Requirement: Update catalog item

Authorized tenant users MUST be able to rename an item and change its kind. Confirming a provisional item (transition to `confirmed`) MUST require an internal SKU (already assigned or provided in the same update). The system MUST NOT allow reassigning an internal SKU once it is set. The system MUST NOT expose supplier-alias management on the item update surface of this capability. First assignment of an internal SKU (including as part of confirmation) MUST be atomic with the item update. Domain invariants for confirmation and SKU immutability MUST be enforced in the catalog item model (not only at the transport layer). The update contract MUST NOT include a stockable attribute.

#### Scenario: Rename and change kind

- **WHEN** a user renames an item and changes its kind
- **THEN** the changes are persisted and the internal SKU (if any) remains unchanged

#### Scenario: Confirm provisional with internal SKU

- **WHEN** a user confirms a provisional item and supplies a new internal SKU (or one already exists)
- **THEN** the item becomes `confirmed` and the internal SKU is stored if it was missing

#### Scenario: Reject confirm without internal SKU

- **WHEN** a user attempts to confirm a provisional item that has no internal SKU and does not provide one
- **THEN** the system rejects the request with a validation error

#### Scenario: Reject reassignment of internal SKU

- **WHEN** a user attempts to assign a different internal SKU after one was already set
- **THEN** the system rejects the request with a validation or conflict error

### Requirement: View catalog item detail

Authorized tenant users MUST be able to open a single catalog item and see its id, name, kind, status, timestamps, and internal SKU when present. The detail view of this capability MUST NOT require displaying supplier aliases and MUST NOT expose a stockable attribute.

#### Scenario: Open item detail

- **WHEN** a user opens an item by id
- **THEN** the system returns the item fields including internal SKU when set

#### Scenario: Item without internal SKU

- **WHEN** a user opens a provisional item that has no internal SKU yet
- **THEN** the system returns the item with an empty/absent internal SKU

### Requirement: Catalog master navigation for manual CRUD

The catalog master MUST allow authorized users to start creating an item and to open an item's detail (and from detail, edit). Create and edit MUST share the same form component within the catalog feature. List responses used by the master SHOULD include the internal SKU when present so users can recognize items by code.

#### Scenario: Create from master

- **WHEN** a user chooses to create a new item from the catalog master
- **THEN** they reach the create form and can submit a manual create

#### Scenario: Open detail from master

- **WHEN** a user selects an item in the catalog master
- **THEN** they reach the item detail view

### Requirement: Catalog item has no stockable flag

The catalog item model and public item contract MUST NOT include a stockable (or equivalent inventory) flag. Stockability concerns are out of scope for the catalog master capability until an inventory capability defines them.

#### Scenario: Item resource omits stockable

- **WHEN** a client reads or writes a catalog item via the item API
- **THEN** the payload does not include a stockable attribute

## MODIFIED Requirements

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
