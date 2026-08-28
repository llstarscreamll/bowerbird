## Why

El módulo de catálogo solo permite listar ítems; la creación ocurre de forma lateral (mint provisional / `create_provisional` desde líneas de factura). Los usuarios necesitan un maestro de ítems: crear, ver detalle y editar sin pasar por la cola de revisión.

## What Changes

- API de escritura de ítems de catálogo: crear (cliente provee ULID) y actualizar campos editables.
- SKU interno canónico expuesto como atributo del **recurso ítem** (no como gestión de aliases): obligatorio al crear manualmente y al confirmar un provisional; **inmutable** una vez fijado.
- Persistencia interna del SKU vía alias `internal_sku` existente (detalle de diseño); el contrato HTTP/PWA no expone el aggregate Alias.
- Ítems creados por el usuario nacen en estado `confirmed`.
- PWA: páginas de detalle, creación y edición; formulario compartido create/edit **dentro** del módulo catalog; master con CTA y navegación al detalle.
- Generación descentralizada de IDs en PWA (ULID), alineada al patrón de facturas / validación ULID en backend.
- Commands de maestro desacoplados de linking/review (solo puertos Item + Alias); escritura item+SKU atómica.
- Modelo táctico rico: `Item` como Aggregate Root con métodos de intención (`Rename`, `ChangeKind`, `Confirm`, `AssignInternalSKU`); VOs `InternalSKU` / `ItemKind`; application como coordinador delgado (sin mutar campos del AR). Sin Domain Events (no hay infra aún).
- **Eliminar `stockable`**: quitar del dominio, persistencia, HTTP y PWA. Editar la migración tenant existente de catálogo (`000012_catalog`) — no crear migración nueva (módulo en desarrollo activo). Stockabilidad se modelará cuando exista el módulo de inventario.

## Non-goals / out of scope

- UI o API de gestión de aliases de proveedor (`supplier_sku`) en create / edit / detail.
- Reubicar la cola de revisión o el linking hacia el módulo de facturas (exploración futura).
- Borrar, fusionar (merge) o descontinuar ítems.
- Movimientos de inventario / stock / flag de stockabilidad (se diseñará en el módulo de inventario).
- Cambiar el flujo de mint provisional desde ingestión de facturas (sigue sin exigir SKU interno hasta confirmación manual).
- Refactor completo del repositorio multi-puerto de catalog; solo no empeorar el acoplamiento en los nuevos use cases.
- Ampliar el ACL invoices→catalog más allá de lo existente (p. ej. no filtrar SKU a invoices en este change).
- Crear migraciones tenant nuevas solo para dropear `stockable` (se edita `000012_*` in-place).

## Capabilities

### New Capabilities

- _(ninguna)_

### Modified Capabilities

- `catalog`: CRUD manual de ítems (create/update/detail), reglas de SKU interno obligatorio e inmutable, superficie PWA de maestro, y retiro de `stockable` del modelo de ítem.

## Impact

- **Backend** (`apps/backend/internal/catalog`): enriquecer `domain.Item` (factories + métodos de intención + VOs; **sin** `Stockable`); commands delgados `CreateItem` / `UpdateItem`; HTTP `POST` / `PATCH` sobre `/api/v1/catalog/items`; read model `ItemView` con `internal_sku`; puerto Alias con batch; validación ULID, unicidad de SKU, TX create/set-SKU; limpiar repo/HTTP de `stockable`.
- **Migraciones**: editar `apps/backend/migrations/tenant/000012_catalog.up.sql` (y `.down.sql` si aplica) para no crear la columna `stockable` — sin archivo de migración nuevo.
- **PWA** (`apps/pwa/src/app/catalog`): rutas `new` / `:itemId` / `:itemId/edit`; store + HTTP service; formulario compartido sin `stockable`; util ULID técnica reutilizable.
- **Specs**: delta sobre `openspec/specs/catalog`.
