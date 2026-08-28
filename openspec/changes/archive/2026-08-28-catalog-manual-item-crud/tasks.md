## 1. Backend — quitar `stockable` + schema

- [x] 1.1 Editar `apps/backend/migrations/tenant/000012_catalog.up.sql` (y `.down.sql` si aplica) para eliminar la columna `stockable` de `catalog_items` — **sin** migración nueva; verificar que el SQL ya no menciona `stockable`
- [x] 1.2 Eliminar `Stockable` de `domain.Item`, repositorio postgres (INSERT/UPDATE/SELECT) y serializers HTTP existentes (GET list/get); verificar que el paquete catalog compila y tests existentes pasan (o se actualizan)

## 2. Backend — domain (modelo rico)

- [x] 2.1 Introducir VOs `InternalSKU` e `ItemKind` (parse/normalize/validate); verificar tests unitarios (vacío, kind inválido, igualdad)
- [x] 2.2 Añadir `NewManualItem` (nace `confirmed` + SKU obligatorio, **sin** stockable) y `NewInternalSKUAlias`; verificar tests de factory
- [x] 2.3 Añadir métodos de intención en `Item`: `Rename`, `ChangeKind`, `Confirm`, `AssignInternalSKU` (guards + inmutabilidad de SKU); **prohibido** mutar campos desde application; verificar tests de dominio por escenario
- [x] 2.4 Extender `AliasRepository` con `ListInternalSKUsByItemIDs` (batch); verificar fake en memoria o test de adapter
- [x] 2.5 Asegurar UoW/tx en adapter para persistir Item + alias internal en create y en primera asignación de SKU; verificar que un fallo a mitad no deja huérfanos

## 3. Backend — application (coordinador delgado)

- [x] 3.1 Implementar `CreateItem`: parse VOs → `NewManualItem` → persist tx; solo `ItemRepository` + `AliasRepository`; 409 id/SKU duplicado; verificar tests de command (sin asignar campos del AR)
- [x] 3.2 Implementar `UpdateItem`: load Item + current SKU → `Rename`/`ChangeKind`/`Confirm`/`AssignInternalSKU` → persist; rechazar mutación de SKU y confirm sin SKU; verificar tests de command
- [x] 3.3 Introducir read model `ItemView` y enriquecer list/get vía batch; **no** añadir `InternalSKU` a `domain.Item`; verificar tests de query

## 4. Backend — HTTP

- [x] 4.1 Exponer `POST /api/v1/catalog/items` y `PATCH /api/v1/catalog/items/{id}` (JSON:API, atributo `internal_sku`, sin `stockable`, sin resource de aliases); wire en composition root; verificar tests de handler/router
- [x] 4.2 Serializar `ItemView` en list/get con `internal_sku` y sin `stockable`; verificar respuesta HTTP

## 5. PWA — shared & data layer

- [x] 5.1 Extraer util técnica `generateUlid()` a core/shared (sin tipos de catalog/invoices) y usarla en catálogo; verificar que invoices y catalog compilan
- [x] 5.2 Extender model/HTTP/store de catalog: `internal_sku`, `createItem`, `updateItem`, `getItem`; **quitar** `stockable` del model; verificar tipos y llamadas

## 6. PWA — UI (dentro del módulo catalog)

- [x] 6.1 Crear `app-catalog-item-form` en `catalog/presentation` (name, kind, SKU required/readonly según modo; **sin** stockable); verificar estados create vs edit vs confirm
- [x] 6.2 Añadir páginas `new`, `:itemId` (detail), `:itemId/edit` y rutas; verificar navegación desde master (CTA + click fila)
- [x] 6.3 Actualizar master: mostrar SKU si existe, CTA “Nuevo”, navegación a detail; verificar UI smoke o test de componente

## 7. Verificación

- [x] 7.1 Correr tests backend del paquete catalog y lint PWA del feature; verificar verde
- [x] 7.2 En local, recrear/resetear schema tenant tras editar `000012` y verificar que `catalog_items` no tiene columna `stockable`
