## ADDED Requirements

### Requirement: Party creation source

The system SHALL persist an immutable `creation_source` on each party with allowed values `manual` and `invoice`. The value MUST be set only at creation time and MUST NOT change on update, role changes, or subsequent invoice linking. `creation_source` records the first birth channel of the party, not later enrichment.

#### Scenario: Manual create sets manual source

- **WHEN** a user creates a party via the manual API/UI path
- **THEN** the persisted party has `creation_source` `manual`

#### Scenario: Invoice bootstrap sets invoice source

- **WHEN** the system creates a new party from an invoice issuer that did not match an existing tax id
- **THEN** the persisted party has `creation_source` `invoice`

#### Scenario: Existing party keeps original source

- **WHEN** an invoice is linked to a party that already existed (matched by tax id)
- **THEN** the party's `creation_source` remains unchanged

#### Scenario: Update does not change creation source

- **WHEN** a party is updated via PATCH (name or roles)
- **THEN** its stored `creation_source` remains unchanged

### Requirement: Filter parties by creation source

Authorized tenant users MUST be able to list parties filtered by `creation_source` (`manual` or `invoice`).

#### Scenario: Filter manual parties

- **WHEN** a user lists parties with `creation_source=manual`
- **THEN** only parties with `creation_source` `manual` are returned

#### Scenario: Filter invoice parties

- **WHEN** a user lists parties with `creation_source=invoice`
- **THEN** only parties with `creation_source` `invoice` are returned

### Requirement: Parties UI shows creation source

The PWA Contactos master and detail views MUST display the party's `creation_source` in user-facing Spanish labels (e.g. "Manual", "Desde factura").

#### Scenario: Master shows origin column

- **WHEN** a user views the Contactos master table
- **THEN** each row shows the creation source label

#### Scenario: Detail shows origin

- **WHEN** a user opens a party detail page
- **THEN** the creation source is visible among the party attributes

## MODIFIED Requirements

### Requirement: Manual party creation

Usuarios autorizados MUST poder crear un contacto manualmente vía API y PWA con nombre, NIT y al menos un rol (`supplier` y/o `customer`). El contacto creado manualmente MUST persistirse con status `confirmed` y `creation_source` `manual`.

#### Scenario: Create confirmed party

- **WHEN** un usuario envía nombre, NIT no vacío y al menos un rol válido que no existe en el tenant
- **THEN** el sistema persiste el contacto con status `confirmed`, `creation_source` `manual` y devuelve su id estable

#### Scenario: Reject create without roles

- **WHEN** un usuario intenta crear un contacto sin ningún rol
- **THEN** el sistema rechaza la operación con error de validación

#### Scenario: Reject create with duplicate NIT

- **WHEN** un usuario intenta crear un contacto cuyo NIT ya existe en el tenant
- **THEN** el sistema rechaza la operación con error de conflicto

### Requirement: Bootstrap party from invoice issuer

When an electronic invoice is persisted, the system MUST resolve or create a party from the issuer tax id and name, and associate that party with the invoice header. A newly created party MUST have `creation_source` `invoice`.

#### Scenario: First invoice from a supplier

- **WHEN** an invoice is persisted whose issuer tax id does not match any existing party
- **THEN** the system creates a provisional party with that tax id and name, `creation_source` `invoice`, marks it as supplier, and links the invoice header to the new party

#### Scenario: Repeat issuer

- **WHEN** an invoice is persisted whose issuer tax id matches an existing party
- **THEN** the system links the invoice header to that party without creating a duplicate and without changing `creation_source`

#### Scenario: Missing issuer tax id

- **WHEN** an invoice is persisted without a usable issuer tax id
- **THEN** the system leaves the invoice party link unset and does not invent a party

### Requirement: List and get parties

Authorized tenant users MUST be able to list parties (filterable by role, `creation_source`, and search on name/tax id) and retrieve a party by id. List and get responses MUST include `creation_source`.

#### Scenario: Filter suppliers

- **WHEN** a user lists parties with role `supplier`
- **THEN** only parties that include the supplier role are returned

#### Scenario: Get includes creation source

- **WHEN** a user retrieves a party by id
- **THEN** the response includes `creation_source`
