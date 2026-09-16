# Design: party-profile-enrichment

## 1. Problem Statement

`Party` solo persiste NIT y nombre. El XML UBL del emisor ya trae scheme, tipo de contribuyente, responsabilidades fiscales, correo, teléfono y dirección, y el usuario no puede añadir canales extra con fuente distinta a la factura.

## 2. Technical Design (Evolutionary Modular Architecture)

**Contextos.** Supporting `parties` posee el agregado y las tablas. `invoices` es customer: extrae UBL y llama OHS; no escribe `parties`. Sin Shared Kernel con `legalentities` (copiar códigos DIAN `31`/`13` en `parties/domain`).

**Agregado `Party`.** Identidad en raíz: `tax_id` (único, inmutable), `name`, `roles`, `status`, `creation_source` (nacimiento, inmutable). Fill-if-empty: `scheme_id`, `taxpayer_kind` (`1` jurídica / `2` natural). Unión: `tax_level_codes`. Hijos (entidades de canal, no agregados): `PartyEmail`, `PartyPhone`, `PartyAddress` con `source` `invoice`|`manual`. Invariante: unicidad por valor normalizado dentro del party, independiente de `source`. Métodos: `AddEmail` / `AddPhone` / `AddAddress` (no-op si duplicado), `Remove*`, `FillScheme` / `FillTaxpayerKind`, `UnionTaxLevelCodes`. Sync de factura nunca borra ni cambia `source`.

**OHS.** `IssuerPartyLookup.ResolveIssuer(ctx, IssuerProfile) (partyID, error)`. ACL en `invoices/adapters/linking` mapea `invoices.domain.Party` → perfil. `create_invoice.applyLinking` pasa el emisor en memoria; `invoice_headers` sigue denormalizando solo nombre+NIT. Fallo de resolve no pierde la factura.

**Extracción.** `mapParty` copia `AdditionalAccountID`, `Contact.ElectronicMail`/`Telephone`, `PhysicalLocation` y `RegistrationAddress` (dedupe por fingerprint). Sin `CustomText`. Kind de email `tax_mailbox` si el valor parece buzón DTE; si no `general`. Kind de dirección `physical`|`registration`.

**Persistencia (tenant `000022`).** Columnas en `parties`. Tablas hijas `party_emails`, `party_phones`, `party_addresses` (unique por party+normalized/fingerprint, CHECK source/kind). Repo: Create inserta hijos; Update identidad; Insert* `ON CONFLICT DO NOTHING`; Delete* por id; GetByID/GetByTaxID hidratan colecciones; List no.

**HTTP.** GET embebe colecciones. POST `/parties/{id}/emails|phones|addresses` (`source=manual`). DELETE por id de canal. Create acepta `scheme_id` y colecciones opcionales. PATCH identidad; `scheme_id`/`taxpayer_kind` solo si null.

**PWA.** Detalle: listas con badge de fuente. Alta/edición: tipo de documento + add/remove canales.

**Fuera de alcance.** Receptor→cliente, CustomText, backfill XML, CRM.
