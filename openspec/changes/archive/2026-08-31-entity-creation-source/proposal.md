## Why

Contactos (`parties`) e ítems de catálogo pueden nacer por formulario PWA o por el pipeline de facturas. Hoy no hay un campo explícito que registre ese primer nacimiento; `status` no basta (un ítem provisional confirmado pierde la señal). Se necesita trazabilidad y analítica sobre el canal de creación original.

## What Changes

- Columna inmutable `creation_source` en `parties` y `catalog_items` (`manual` | `invoice`), añadida en migraciones existentes (`000011_parties.up.sql`, `000012_catalog.up.sql`).
- Dominio y factories: cada ruta de creación setea el valor correcto y los updates no lo modifican.
- API JSON:API: exponer `creation_source` en list/get; filtro opcional `?creation_source=` en listados.
- PWA: mostrar origen en master y detalle de Contactos y Catálogo.

## Non-goals

- Columna `source_invoice_header_id` ni referencia a factura concreta.
- Valor `import` para CSV masivo (se definirá en un change futuro).
- Migraciones nuevas, backfill de datos existentes ni compatibilidad con tenants ya poblados.
- Cambiar semántica de `status` ni flujos de confirmación de contactos.

## Capabilities

### New Capabilities

_(ninguna)_

### Modified Capabilities

- `parties`: persistir y exponer `creation_source`; manual → `manual`, bootstrap factura → `invoice`; inmutable; filtro en list.
- `catalog`: persistir y exponer `creation_source`; create manual → `manual`, mint provisional → `invoice`; inmutable tras confirmación; filtro en list.

## Impact

| Área        | Módulos                                                                                    |
| ----------- | ------------------------------------------------------------------------------------------ |
| Migraciones | `migrations/tenant/000011_parties.up.sql`, `000012_catalog.up.sql`                         |
| Backend     | `internal/parties/`, `internal/catalog/` (domain, commands, repository, HTTP)              |
| Integración | `invoices/adapters/linking/` sin cambio de contrato; factories internos propagan `invoice` |
| PWA         | `apps/pwa/src/app/parties/`, `apps/pwa/src/app/catalog/`                                   |
| Specs       | deltas `parties`, `catalog`                                                                |
