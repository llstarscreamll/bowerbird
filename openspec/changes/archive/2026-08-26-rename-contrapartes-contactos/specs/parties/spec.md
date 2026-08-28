## MODIFIED Requirements

### Requirement: Parties UI Terminology

The UI SHALL use the term "Contactos" (Contacts) exclusively when referring to `parties` entities, instead of "Contrapartes".

#### Scenario: Navigating the PWA menus

- **WHEN** the user views the main side navigation or layout headers
- **THEN** they see "Contactos" as the menu item instead of "Contrapartes"

#### Scenario: Viewing the Parties master page

- **WHEN** the user opens the `/parties` route
- **THEN** the page title reads "Contactos" and actions read "Nuevo Contacto" (or similar) instead of "Contraparte"

#### Scenario: Receiving success/error notifications

- **WHEN** the system emits a toast notification regarding a `party` (e.g. creation, update, deletion)
- **THEN** the notification text uses the term "Contacto" (e.g., "Contacto guardado exitosamente") instead of "Contraparte"

#### Scenario: Viewing Invoice details

- **WHEN** the user views an invoice detail page that displays the issuer or receiver
- **THEN** visual labels or headers that refer to the external entity use the term "Contacto" instead of "Contraparte"
