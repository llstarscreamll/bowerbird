# Domain Dictionary

This glossary defines the ubiquitous language for the Bowerbird project, mapping Colombian business and e-invoicing concepts (Spanish) to their exact representation in the codebase (English).

**Rule for LLMs and Developers:** ALWAYS use the exact English term defined in the "Code (EN)" column for variables, structs, classes, API routes, and database tables.

## Core E-Invoicing & Legal Entities

| Term (ES / Business)       | Meaning / Context                                                                     | Code (EN)                         |
| :------------------------- | :------------------------------------------------------------------------------------ | :-------------------------------- |
| **Facturador / Inquilino** | The company that owns the workspace and receives or issues invoices.                  | `Tenant` or `Organization`        |
| **Contacto / Tercero**     | Any external legal entity or person trading with the Tenant. UI label: Contacto.      | `Party`                           |
| **Proveedor**              | A Party that issues invoices to the Tenant (Accounts Payable).                        | `Supplier` (role)                 |
| **Adquirente / Cliente**   | A Party that receives invoices from the Tenant (Accounts Receivable).                 | `Customer` (role)                 |
| **NIT / Cédula**           | The unique tax identification number of a Party.                                      | `TaxID` (avoid using `CompanyID`) |
| **Razón social propia**    | The Tenant's own legal identity used to match inbound invoice receivers. Not a Party. | `LegalEntity`                     |
| **DIAN**                   | The Colombian tax authority.                                                          | `DIAN` (kept as-is)               |

## Documents

| Term (ES / Business)     | Meaning / Context                                                                                                 | Code (EN)                    |
| :----------------------- | :---------------------------------------------------------------------------------------------------------------- | :--------------------------- |
| **Factura Electrónica**  | The electronic invoice document validated by DIAN (legal type, not direction).                                    | `Invoice`                    |
| **Factura Recibida**     | Invoice where the Tenant is the Receiver (Accounts Payable). UI: Facturas recibidas. Current `invoices` module.   | `Invoice` (inbound)          |
| **Factura Emitida**      | Invoice where the Tenant is the Issuer (Accounts Receivable). UI reserved: Facturas emitidas. Not in product yet. | `Invoice` (outbound; future) |
| **Emisor**               | The party that created and sent the invoice. On inbound invoices, maps to a `Supplier`.                           | `Issuer`                     |
| **Receptor**             | The party that receives the invoice. On inbound invoices, maps to the `Tenant`.                                   | `Receiver`                   |
| **CUFE**                 | Unique electronic invoice code (Código Único de Facturación Electrónica).                                         | `CUFE` (kept as-is)          |
| **UBL**                  | Universal Business Language (XML format used by DIAN).                                                            | `UBL` (kept as-is)           |
| **Línea de Factura**     | A single item/row inside an invoice.                                                                              | `InvoiceLine` or `LineItem`  |
| **Impuestos**            | Taxes applied to the invoice or line item.                                                                        | `TaxAmount`, `TaxTotal`      |
| **Fecha de Emisión**     | The date the invoice was issued.                                                                                  | `IssueDate`                  |
| **Fecha de Vencimiento** | The date the invoice is due (`cbc:DueDate`, else `PaymentDueDate`).                                               | `DueDate`                    |
| **Descuento**            | Document-level allowance; Payable = lines + tax − this amount.                                                    | `AllowanceTotal`             |

## Catalog & Inventory

| Term (ES / Business)            | Meaning / Context                                                                                                                                                                                         | Code (EN)                                |
| :------------------------------ | :-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | :--------------------------------------- |
| **Catálogo**                    | The master list of products, services, or assets.                                                                                                                                                         | `Catalog`                                |
| **Producto / Servicio**         | A single entry in the catalog.                                                                                                                                                                            | `Item`                                   |
| **Tipo de Ítem**                | Whether the item is a physical good, service, or asset.                                                                                                                                                   | `Kind` (`Goods`, `Service`, `Asset`)     |
| **Código interno**              | Tenant-canonical item code on the catalog item. Required on manual create and when confirming a provisional; immutable once set. Not a supplier identifier.                                               | `InternalCode` (column on `Item`)        |
| **SKU / Código de proveedor**   | A supplier-scoped code that maps an invoice line onto a catalog item. Stored as an alias, not as `InternalCode`.                                                                                          | `Alias` (`SupplierSKU`), `SellerSKU`     |
| **GTIN**                        | Global Trade Item Number from UBL `StandardItemIdentification` when it is 8, 12, 13, or 14 digits. No check digit. Unscoped catalog alias.                                                                | `GTIN`                                   |
| **Código del adquirente**       | Buyer’s own item code on the invoice line (`BuyersItemIdentification`). Used as a hard lookup against `InternalCode`; not stored as an alias.                                                             | `BuyerCode` (`buyer_code`)               |
| **Cruce**                       | Visible mapping of supplier SKU / GTIN aliases onto a catalog item. Distinct from the tenant `InternalCode`.                                                                                              | `Alias`, crosswalk                       |
| **Emparejamiento / Enlace**     | The act of linking an invoice line item to a catalog item.                                                                                                                                                | `Match`, `Link` (e.g., `MatchMemory`)    |
| **Acuñar / Mint (provisional)** | Auto-create a provisional catalog item from invoice line evidence (supplier party + usable seller SKU), attach `supplier_sku` and optional `gtin` aliases, and link the line. Not a manual master create. | `Mint`, `mintProvisional`, `Provisional` |

## Platform & Integrations

| Term (ES / Business)             | Meaning / Context                                             | Code (EN)                 |
| :------------------------------- | :------------------------------------------------------------ | :------------------------ |
| **Buzón Tributario / Recepción** | The system that receives emails with XML/PDF invoices.        | `Inbox` or `UnifiedInbox` |
| **Conexión**                     | An external integration (e.g., connecting a Gmail account).   | `Connection`              |
| **Permisos / Roles**             | Role-Based Access Control (RBAC) rules for users.             | `Permissions`, `Roles`    |
| **Planes / Suscripciones**       | Feature limits and access levels for a Tenant.                | `Entitlements`            |
| **Credenciales**                 | Encrypted sensitive data (passwords, tokens) for connections. | `Secrets`                 |
