# parties Specification

## Purpose

Tenant counterparties (suppliers and customers) identified primarily by tax id, bootstrapped from electronic invoice issuers so catalog aliases and spend analytics can be scoped per trading partner.

## Requirements

### Requirement: Party identity by tax id

The system SHALL represent each tenant party with a stable identifier, legal/display name, and tax id (NIT / CompanyID). Within a tenant, tax id MUST be unique when present and non-empty.

#### Scenario: Create party with tax id

- **WHEN** a party is created with a non-empty tax id that is not already used in the tenant
- **THEN** the system persists the party and returns its stable id

#### Scenario: Reject duplicate tax id

- **WHEN** a party create or update would reuse another party's tax id in the same tenant
- **THEN** the system rejects the change with a conflict error

### Requirement: Party roles

The system SHALL allow a party to hold one or more roles among `supplier` and `customer` without requiring separate party records per role.

#### Scenario: Mark issuer as supplier

- **WHEN** a party is bootstrapped or updated from an invoice issuer
- **THEN** the party includes the `supplier` role (and may later also hold `customer`)

### Requirement: Bootstrap party from invoice issuer

When an electronic invoice is persisted, the system MUST resolve or create a party from the issuer tax id and name, and associate that party with the invoice header.

#### Scenario: First invoice from a supplier

- **WHEN** an invoice is persisted whose issuer tax id does not match any existing party
- **THEN** the system creates a provisional party with that tax id and name, marks it as supplier, and links the invoice header to the new party

#### Scenario: Repeat issuer

- **WHEN** an invoice is persisted whose issuer tax id matches an existing party
- **THEN** the system links the invoice header to that party without creating a duplicate

#### Scenario: Missing issuer tax id

- **WHEN** an invoice is persisted without a usable issuer tax id
- **THEN** the system leaves the invoice party link unset and does not invent a party

### Requirement: Document fidelity preserved

Invoice header denormalized issuer name and tax id fields MUST remain the document source of truth even when a party link exists.

#### Scenario: Party renamed later

- **WHEN** a linked party's display name is updated after an invoice was stored
- **THEN** the invoice header's stored issuer name remains unchanged

### Requirement: List and get parties

Authorized tenant users MUST be able to list parties (filterable by role and search on name/tax id) and retrieve a party by id.

#### Scenario: Filter suppliers

- **WHEN** a user lists parties with role `supplier`
- **THEN** only parties that include the supplier role are returned
