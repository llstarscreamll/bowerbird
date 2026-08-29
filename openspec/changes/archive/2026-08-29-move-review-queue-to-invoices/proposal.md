## Why

La cola de revisión y las decisiones de vinculación viven en Catálogo, pero el sujeto del trabajo son **líneas de factura** pendientes de conciliar con ítems. Eso confunde la navegación (hay que salir de Facturas para ver lo pendiente global) y rompe los límites de módulo: `catalog` hace SQL reach-through sobre `invoice_lines` / `invoice_headers` mientras `invoices` ya consume el motor de resolución vía ACL.

## What Changes

- Mover la **cola de revisión** (query + HTTP + UI) de `catalog` a `invoices`.
- Mover las **decisiones manuales** de vinculación (`link` / `never_match` / `create_provisional`) a `invoices` como orquestador que escribe el estado de link en sus tablas.
- Exponer puertos ACL en `catalog` solo para efectos de identidad/matching (validar ítem, mint provisional, ensure alias, upsert match memory); **catalog deja de leer/escribir tablas de facturas**.
- **BREAKING**: retirar `GET /api/v1/catalog/review-queue` y `POST /api/v1/catalog/lines/{lineId}/decisions`.
- Añadir `GET /api/v1/invoicing/review-queue` y `POST /api/v1/invoicing/invoices/{invoiceId}/lines/{lineId}/decisions`.
- PWA: ruta `/invoices/review`, CTA/entrada desde Facturas; quitar cola de Catálogo.
- Actualizar specs `catalog` e `invoices` para reflejar el nuevo dueño de la cola y de las columnas de link.

## Non-goals / out of scope

- Crear un bounded context `linking` separado (enfoque C).
- Mover el motor `ResolveInvoiceLine`, soft matchers o `catalog_match_memories` a `invoices`.
- Rediseñar reglas de matching / scoring.
- CRUD de maestro de ítems (change `catalog-manual-item-crud`).
- Paginación avanzada de la cola, filtros rich, o bulk resolve.
- Inventario / stock.

## Capabilities

### New Capabilities

<!-- ninguna: se reubica comportamiento entre capabilities existentes -->

### Modified Capabilities

- `catalog`: deja de ser dueño de la cola de revisión y de las decisions HTTP sobre líneas; conserva pipeline de resolución, aliases, match memory y mint; ya no persiste estado de link en tablas de facturas.
- `invoices`: dueño de la cola de revisión cross-invoice, de las decisions manuales, y de la persistencia de `link_*` / `linking_status`; sigue resolviendo nombres de ítems vía ACL.

## Impact

- **Backend**: `apps/backend/internal/catalog` (quitar `InvoiceLineLinkRepository` del repo catalog, partir `RememberDecision`, retirar rutas HTTP); `apps/backend/internal/invoices` (nuevos query/command, puertos ACL ampliados, repo de links, HTTP).
- **PWA**: `apps/pwa/src/app/invoices` (página review + store/HTTP); `apps/pwa/src/app/catalog` (quitar review); `app-catalog-linker` permanece como widget UI reutilizable.
- **API**: breaking en rutas `/catalog/...` de cola/decisions; clientes internos (solo PWA hoy) migran en el mismo change.
- **Specs**: deltas en `openspec/specs/catalog` y `openspec/specs/invoices`.
