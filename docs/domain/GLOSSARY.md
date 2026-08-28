# Domain Dictionary

This glossary defines the ubiquitous language for the Bowerbird project, mapping Colombian business and e-invoicing concepts (Spanish) to their exact representation in the codebase (English).

**Rule for LLMs and Developers:** ALWAYS use the exact English term defined in the "Code (EN)" column for variables, structs, classes, API routes, and database tables.

## Core E-Invoicing & Legal Entities

| Term (ES / Business)       | Meaning / Context                                                                | Code (EN)                         |
| :------------------------- | :------------------------------------------------------------------------------- | :-------------------------------- |
| **Facturador / Inquilino** | The company that owns the workspace and receives or issues invoices.             | `Tenant` or `Organization`        |
| **Contacto / Tercero**     | Any external legal entity or person trading with the Tenant. UI label: Contacto. | `Party`                           |
| **Proveedor**              | A Party that issues invoices to the Tenant (Accounts Payable).                   | `Supplier` (role)                 |
| **Adquirente / Cliente**   | A Party that receives invoices from the Tenant (Accounts Receivable).            | `Customer` (role)                 |
| **NIT / Cédula**           | The unique tax identification number of a Party.                                 | `TaxID` (avoid using `CompanyID`) |
| **DIAN**                   | The Colombian tax authority.                                                     | `DIAN` (kept as-is)               |

## Documents

| Term (ES / Business)    | Meaning / Context                                                         | Code (EN)                                     |
| :---------------------- | :------------------------------------------------------------------------ | :-------------------------------------------- |
| **Factura Electrónica** | The electronic invoice document validated by DIAN.                        | `Invoice`                                     |
| **Emisor**              | The party that created and sent the invoice.                              | `Issuer` (maps to a `Supplier`)               |
| **Receptor**            | The party that receives the invoice.                                      | `Receiver` (maps to a `Customer` or `Tenant`) |
| **CUFE**                | Unique electronic invoice code (Código Único de Facturación Electrónica). | `CUFE` (kept as-is)                           |
| **UBL**                 | Universal Business Language (XML format used by DIAN).                    | `UBL` (kept as-is)                            |
| **Línea de Factura**    | A single item/row inside an invoice.                                      | `InvoiceLine` or `LineItem`                   |
| **Impuestos**           | Taxes applied to the invoice or line item.                                | `TaxAmount`, `TaxTotal`                       |
| **Fecha de Emisión**    | The date the invoice was issued.                                          | `IssueDate`                                   |

## Catalog & Inventory

| Term (ES / Business)        | Meaning / Context                                            | Code (EN)                                    |
| :-------------------------- | :----------------------------------------------------------- | :------------------------------------------- |
| **Catálogo**                | The master list of products, services, or assets.            | `Catalog`                                    |
| **Producto / Servicio**     | A single entry in the catalog.                               | `Item`                                       |
| **Tipo de Ítem**            | Whether the item is a physical good, service, or asset.      | `Kind` (`Goods`, `Service`, `Asset`)         |
| **SKU / Código Externo**    | A code used by a supplier or internally to identify an item. | `Alias` (e.g., `SupplierSKU`, `InternalSKU`) |
| **Emparejamiento / Enlace** | The act of linking an invoice line item to a catalog item.   | `Match`, `Link` (e.g., `MatchMemory`)        |

## Platform & Integrations

| Term (ES / Business)             | Meaning / Context                                             | Code (EN)                 |
| :------------------------------- | :------------------------------------------------------------ | :------------------------ |
| **Buzón Tributario / Recepción** | The system that receives emails with XML/PDF invoices.        | `Inbox` or `UnifiedInbox` |
| **Conexión**                     | An external integration (e.g., connecting a Gmail account).   | `Connection`              |
| **Permisos / Roles**             | Role-Based Access Control (RBAC) rules for users.             | `Permissions`, `Roles`    |
| **Planes / Suscripciones**       | Feature limits and access levels for a Tenant.                | `Entitlements`            |
| **Credenciales**                 | Encrypted sensitive data (passwords, tokens) for connections. | `Secrets`                 |
