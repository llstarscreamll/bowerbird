## Why

Los tenants necesitan cargar su catálogo maestro (cientos de miles a millones de ítems) desde un archivo, no ítem a ítem. Sin import asíncrono, paginación del listado y trazabilidad del proceso, el maestro deja de ser usable.

## What Changes

- Importación CSV UTF-8 de ítems (`internal_code`, `name`, `kind`) por upload presignado + job con cursor (chunks ~5k filas).
- Upsert por `internal_code`: crear confirmado con `creation_source=import`; si existe, actualizar `name`/`kind` (y confirmar si era provisional). No mutar `internal_code` ni `creation_source`.
- Aggregate `CatalogImport` con snapshots denormalizados de quien solicitó y quien canceló (sin FK a identity).
- Historial y detalle en PWA con monitor de progreso, cancelación cooperativa, y errores de fila persistidos y paginados.
- Listado de catálogo con cursor (`page[size]`, `page[after]`) para soportar volúmenes grandes.
- Purga diaria a medianoche America/Bogota de procesos de importación con más de un año (no borra ítems).

## Non-goals / out of scope

- Excel/xlsx, ZIP, múltiples archivos por import.
- Export CSV de errores.
- Aliases `supplier_sku` / NIT de proveedor en el archivo.
- Merge de provisionals por nombre.
- Dry-run, export del catálogo, borrar ítems importados.
- Entitlement nuevo (sigue gated por invoicing).
- Inventario / stockable.
- FK o lookup en vivo a identity.

## Capabilities

### New Capabilities

- _(ninguna)_

### Modified Capabilities

- `catalog`: import CSV asíncrono, historial/detalle/cancel/errores, paginación del maestro, `creation_source=import`, purga programada de procesos.

## Impact

- **Backend** (`apps/backend/internal/catalog`, `platform/storage`, `platform/http/host`, `platform/messaging`): dominio, migración tenant `000018`, jobs, HTTP, FileStore streaming.
- **PWA** (`apps/pwa/src/app/catalog`): páginas `imports` / `imports/:id`, dialog de carga, master paginado, linker con `page[size]`.
- **Infra AWS** (`apps/deploy/aws/src/stack.ts`): schedule `catalog-import-purge`.
- **E2E** (`apps/e2e/tests/http/catalog`): contratos de import y list paginado.
- **Specs**: delta sobre `openspec/specs/catalog`.
