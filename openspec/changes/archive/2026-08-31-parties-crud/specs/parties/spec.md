## ADDED Requirements

### Requirement: Manual party creation

Usuarios autorizados MUST poder crear un contacto manualmente vía API y PWA con nombre, NIT y al menos un rol (`supplier` y/o `customer`). El contacto creado manualmente MUST persistirse con status `confirmed`.

#### Scenario: Create confirmed party

- **WHEN** un usuario envía nombre, NIT no vacío y al menos un rol válido que no existe en el tenant
- **THEN** el sistema persiste el contacto con status `confirmed` y devuelve su id estable

#### Scenario: Reject create without roles

- **WHEN** un usuario intenta crear un contacto sin ningún rol
- **THEN** el sistema rechaza la operación con error de validación

#### Scenario: Reject create with duplicate NIT

- **WHEN** un usuario intenta crear un contacto cuyo NIT ya existe en el tenant
- **THEN** el sistema rechaza la operación con error de conflicto

### Requirement: Party update

Usuarios autorizados MUST poder actualizar nombre y roles de un contacto existente. El NIT MUST NOT ser modificable tras la creación.

#### Scenario: Update name and roles

- **WHEN** un usuario envía un PATCH con nombre y/o roles válidos (al menos un rol si se envían roles)
- **THEN** el sistema persiste los cambios y devuelve el contacto actualizado

#### Scenario: Reject tax id change

- **WHEN** un usuario envía un PATCH que incluye `tax_id`
- **THEN** el sistema ignora o rechaza el cambio de NIT; el NIT almacenado permanece igual

#### Scenario: Reject update without roles

- **WHEN** un usuario envía roles vacíos
- **THEN** el sistema rechaza la operación con error de validación

### Requirement: Parties CRUD UI

La PWA MUST exponer flujo completo de consulta y edición de contactos siguiendo el patrón del catálogo.

#### Scenario: Navigate to create

- **WHEN** el usuario está en la página maestra de Contactos
- **THEN** ve un botón "Nuevo Contacto" que navega a `/parties/new`

#### Scenario: View party detail

- **WHEN** el usuario hace clic en una fila de la tabla de contactos
- **THEN** navega a la página de detalle del contacto

#### Scenario: Edit party

- **WHEN** el usuario abre detalle o edición de un contacto
- **THEN** puede modificar nombre y roles (checkboxes); el NIT se muestra solo lectura

#### Scenario: Success notification on save

- **WHEN** se crea o actualiza un contacto exitosamente
- **THEN** el sistema muestra toast con texto "Contacto" (ej. "Contacto guardado exitosamente")

## MODIFIED Requirements

### Requirement: Party identity by tax id

The system SHALL represent each tenant party with a stable identifier, legal/display name, and tax id (NIT). Within a tenant, tax id MUST be unique when present and non-empty. Once assigned at creation, tax id MUST NOT change.

#### Scenario: Create party with tax id

- **WHEN** a party is created with a non-empty tax id that is not already used in the tenant
- **THEN** the system persists the party and returns its stable id

#### Scenario: Reject duplicate tax id

- **WHEN** a party create would reuse another party's tax id in the same tenant
- **THEN** the system rejects the change with a conflict error

#### Scenario: Tax id immutable on update

- **WHEN** an existing party is updated
- **THEN** its stored tax id remains unchanged regardless of request payload

### Requirement: Party roles

The system SHALL allow a party to hold one or more roles among `supplier` and `customer`. Manual create and update MUST require at least one role.

#### Scenario: Mark issuer as supplier

- **WHEN** a party is bootstrapped or updated from an invoice issuer
- **THEN** the party includes the `supplier` role (and may later also hold `customer`)

#### Scenario: Require at least one role on manual write

- **WHEN** a user creates or updates a party via the manual API/UI path
- **THEN** the party MUST retain at least one of `supplier` or `customer`

### Requirement: Parties UI Terminology

The UI SHALL use the term "Contactos" (Contacts) exclusively when referring to `parties` entities, instead of "Contrapartes".

#### Scenario: Navigating the PWA menus

- **WHEN** the user views the main side navigation or layout headers
- **THEN** they see "Contactos" as the menu item instead of "Contrapartes"

#### Scenario: Viewing the Parties master page

- **WHEN** the user opens the `/parties` route
- **THEN** the page title reads "Contactos" and actions read "Nuevo Contacto"

#### Scenario: Receiving success/error notifications

- **WHEN** the system emits a toast notification regarding a `party` (e.g. creation or update)
- **THEN** the notification text uses the term "Contacto" (e.g., "Contacto guardado exitosamente")

#### Scenario: Viewing Invoice details

- **WHEN** the user views an invoice detail page that displays the issuer or receiver
- **THEN** visual labels or headers that refer to the external entity use the term "Contacto" instead of "Contraparte"
