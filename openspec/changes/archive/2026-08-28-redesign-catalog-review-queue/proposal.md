## Why

La cola de revisión actual (`Cola de revisión`) es apenas un esqueleto funcional que requiere que el usuario escriba manualmente el ID (ULID) crudo del ítem para realizar una vinculación manual, lo cual es inoperable en un escenario real. Tampoco expone las sugerencias calculadas por el motor de conciliación ni permite enseñarle al sistema rechazos ("nunca vincular"). Además, la vista maestra de facturas no expone el estado global de vinculación, ocultando a los usuarios qué facturas requieren atención en el catálogo.

## What Changes

- Rediseño de la página de "Cola de revisión" en el módulo de Catálogo (Frontend) para incluir un buscador visual (Combobox) integrado al catálogo.
- Visualización interactiva de las sugerencias pre-calculadas (con sus _scores_) en la revisión de cada línea.
- Nuevas acciones de revisión: "Vincular", "Rechazar (nunca vincular)" y "Crear nuevo ítem provisional" a partir de la evidencia de la línea.
- Reutilización de esta misma experiencia de vinculación línea por línea dentro de la página de "Detalle de factura" (`invoices`), permitiendo resolver el catálogo sin salir de la factura.
- Inclusión de la columna de estado global de vinculación (`linking_status`) en la página maestra de facturas (`invoices`).
- En el backend, las queries y handlers se actualizarán para proveer los nombres reales de los ítems en las sugerencias pre-calculadas en lugar de solo los `item_id`.

## Capabilities

### New Capabilities

### Modified Capabilities

- `catalog`: Flujo de revisión asistida que permite al usuario resolver líneas huérfanas mediante sugerencias, búsqueda visual por nombre/SKU, y acciones de rechazo o creación provisional rápida.
- `invoices`: Se expone el estado consolidado de vinculación de catálogo en la tabla principal y se integra la experiencia de vinculación visual a nivel de línea.

## Impact

- **Frontend (`@apps/pwa`)**:
  - Creación de componente compartido `app-catalog-linker` exportado por el módulo de catálogo (aislamiento e independencia UI).
  - Refactor en `apps/pwa/src/app/catalog/presentation/pages/review` usando el nuevo componente.
  - Refactor en la tarjeta de línea de `apps/pwa/src/app/invoices/presentation/pages/detail` usando el nuevo componente.
  - Actualización de columnas en `apps/pwa/src/app/invoices/presentation/pages/master`.
- **Backend (`apps/backend`)**:
  - La query `ListReviewQueueQuery` del módulo de catálogo poblará los nombres localmente.
  - La query `GetInvoiceByIDQuery` del módulo de facturas utilizará un puerto interno (Anti-Corruption Layer) hacia el módulo de catálogo para resolver los nombres, previniendo el acoplamiento cruzado de bases de datos (reach-through).
