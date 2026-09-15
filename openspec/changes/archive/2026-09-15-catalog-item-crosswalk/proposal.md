## Why

Enseñar el cruce línea→ítem es opaco (el linker siempre graba `remember`+`lock` sin mostrar qué regla se crea) y no se puede auditar ni corregir: el detalle del ítem oculta aliases, `item_code` aplasta GTIN y SKU del emisor, y una línea locked no se reabre. El maestro importable por `internal_code` no pegará facturas hasta que el cruce sea un objeto de negocio visible.

## What Changes

- Superficie de **códigos de proveedor / GTIN** en el ítem (listar, añadir, quitar). Conflicto 409 si el tuple ya apunta a otro ítem (merge no se ejecuta aquí; el cliente puede navegar al ítem dueño).
- **BREAKING** — Dejar de colapsar identificadores UBL en `item_code`. Parser y persistencia de línea guardan por separado `buyer_code` (`BuyersItemIdentification`), `seller_sku` (`SellersItemIdentification`) y `gtin` (solo si `StandardItemIdentification` pasa el clasificador GTIN).
- Resolución dura por **acuerdo**: hits de código interno (buyer), GTIN global y `supplier_sku` por party. Si coinciden → auto-link `hard`; si discrepan → `suggested` con conflicto, sin mint. Mint provisional **solo** con `party` + `seller_sku` usable; al mint o al recordar, adjuntar todos los aliases válidos de la línea al mismo ítem.
- Linker/cola/detalle de factura: mostrar las tres evidencias; checkboxes **Recordar** y **Bloquear esta línea** (default on); preview de reglas; **desbloquear** para corregir.
- Alias con `source` (`invoice` | `manual`) y schemes `supplier_sku` | `gtin`. `InternalCode` sigue siendo atributo del ítem; `BuyersItemIdentification` es lookup, no alias.
- Códigos de baja entropía (`seller_sku` demasiado genérico) no se auto-aprenden ni se usan para mint; el usuario puede crear el alias a mano.
- **BREAKING** — Contratos JSON:API de línea de factura, payload de decisión, detalle de ítem (aliases), y evidencias de match memory (clavean `seller_sku`, no el código colapsado).
- Glosario: GTIN y código del adquirente.

## Non-goals / out of scope

- Merge / fusión de ítems (otro change). Este change **no** reasigna aliases entre ítems ni apaga provisionals. El 409 expone el `item_id` dueño.
- Vigencia temporal (`valid_from` / `valid_to`) de aliases.
- Import CSV de cruce (NIT + SKU + `internal_code`).
- `AdditionalItemIdentification`, check digit GS1 obligatorio, UoM/empaque en el cruce.
- Inventario, stock, BOM/kits.
- Reescritura masiva de líneas históricas locked al editar un alias (solo líneas no locked se re-resuelven si hay job de reproceso existente; no se añade backfill de links).
- FK viva o join a tablas de `parties` desde catalog (el alias guarda `party_id`; la PWA resuelve el nombre).
- Shared Kernel / paquete compartido de GTIN; catalog importando dominio invoices (o al revés) para clasificar identificadores.
- Catalog escribiendo `invoice_lines` (merge futuro usará puerto invoices).
- Eventos de outbox para linking; reestructura flat-by-aggregate de los módulos.

## Capabilities

### New Capabilities

- _(ninguna)_

### Modified Capabilities

- `catalog`: gestión de aliases, resolución multi-id, mint, `RememberDecision`, detalle con cruce.
- `invoices`: evidencia UBL estructurada en la línea, UI de enseñanza explícita, unlock, decisiones que adjuntan todos los aliases válidos.

## Impact

- **Backend catalog** (`apps/backend/internal/catalog`): dominio Alias/GTIN, `RememberDecision` (1 TX aliases+memory), `ResolveInvoiceLine` por acuerdo, OHS `api.InvoiceSupport` por capacidades, HTTP aliases, writer único de tablas catalog.
- **Backend invoices** (`apps/backend/internal/invoices`): parser ACL UBL → `LineIdentifiers` (GTIN clasificado **aquí**, sin paquete compartido), persistencia de línea **BREAKING**, `CatalogACL` traduce OHS, `ApplyLineDecision` (unlock; remember → un call OHS), writer único de `invoice_lines`.
- **Migraciones tenant**: columnas de línea + backfill clasificando `item_code` existente; drop de `item_code` tras backfill; `source` en aliases.
- **PWA** (`apps/pwa/src/app/catalog`, `invoices`): detalle de ítem con tabla de cruce; linker con evidencias + remember/lock/unlock; master/review/detalle de factura sin `item_code` único.
- **E2E** (`apps/e2e/tests/http`): contratos catalog aliases e invoice lines.
- **Docs**: `docs/domain/GLOSSARY.md`.
- **No infra AWS nueva.**
