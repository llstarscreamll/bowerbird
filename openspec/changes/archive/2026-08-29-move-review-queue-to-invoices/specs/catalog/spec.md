## ADDED Requirements

### Requirement: List catalog items

Authorized tenant users MUST be able to list catalog items (filter by kind and provisional status).

#### Scenario: List catalog items

- **WHEN** a user lists catalog items with optional kind or status filters
- **THEN** the system returns matching items for the tenant

## REMOVED Requirements

### Requirement: List items and review queue

**Reason**: La cola de revisión de líneas de factura pasa a `invoices`. El listado de ítems queda como requisito `List catalog items`.

**Migration**: Usar `List catalog items` en catalog y `Review queue lists unresolved invoice lines` en invoices.

### Requirement: User manually links an invoice line to a catalog item

**Reason**: La decisión manual sobre líneas de factura y la API/UI de decisions pasan a `invoices`; `catalog` solo aporta identidad y match memory vía ACL.

**Migration**: Requisitos de cola y decisions en `invoices`.

### Requirement: User explicitly rejects catalog suggestions for a line

**Reason**: El rechazo se orquesta desde `invoices`; `catalog` persiste `never_match` en match memory cuando el orquestador lo solicita.

**Migration**: Requisito `User applies catalog link decisions on invoice lines` en `invoices`.

### Requirement: User creates a provisional catalog item from an invoice line

**Reason**: La acción de usuario se inicia en `invoices`; `catalog` sigue minteando el ítem provisional cuando el orquestador lo pide (y en el pipeline automático).

**Migration**: Requisito `User applies catalog link decisions on invoice lines` en `invoices` (acción `create_provisional`).
