## Context

Hoy el catálogo expone solo `GET /items`, `GET /items/{id}`, review-queue y decisiones sobre líneas. `ItemRepository.CreateItem` / `UpdateItem` existen pero no hay commands HTTP de maestro. El SKU interno ya está modelado como alias `internal_sku` (único por tenant, sin party). La PWA tiene master + review; sin create/detail/edit. IDs de escritura en el producto son cliente→servidor (ULID), p. ej. cola de extracción de facturas.

El módulo `catalog` hoy concentra dos sub-capacidades: **maestro de ítems** e **linking/review**. Este change refuerza el maestro **sin** ensanchar el acoplamiento hacia linking.

Además, `catalog_items.stockable` existe en schema/dominio/HTTP pero **no se usa** en ninguna capacidad actual; es diseño prematuro de inventario.

Ver `proposal.md` para motivación y non-goals.

## Goals / Non-Goals

**Goals:**

- Commands + HTTP create/update de ítems con ULID cliente.
- Exponer `internal_sku` canónico en lecturas (list/get) como atributo del **contrato de ítem**, sin UI/API de aliases de proveedor.
- PWA: rutas create / detail / edit; form compartido **dentro** de `catalog/presentation`; util ULID técnica compartida.
- Reglas de dominio: create → `confirmed` + SKU; confirm provisional exige SKU; SKU inmutable tras fijarse.
- Escritura item+SKU **atómica**; commands de maestro sin depender de puertos de linking/review.
- Retirar `stockable` del modelo (dominio, repo, HTTP, PWA) editando la migración `000012_catalog` existente.

**Non-Goals:**

- Endpoints o UI CRUD de `supplier_sku`.
- Mover review-queue / linking a `invoices`.
- Delete/merge de ítems.
- Modelar stockabilidad / inventario (futuro módulo de inventario).
- Crear migraciones tenant **nuevas** para este cleanup (solo editar `000012_*` in-place; entorno en desarrollo activo).
- Refactor completo del `CatalogRepository` monolítico; solo no empeorar el acoplamiento en los nuevos use cases.

## Decisions

### 1. Contrato público = ítem; persistencia SKU = alias (ACL interna)

- **Enfoque**: La API JSON:API expone `attributes.internal_sku` en el resource `catalog_items`. Por dentro, create/update escriben/leen via `AliasRepository` (`scheme=internal_sku`, `party_id` NULL).
- **Rationale**: Consumidores dependen del **código canónico del ítem**, no del aggregate Alias.
- **Alternativa rechazada**: Columna `catalog_items.internal_sku` — duplica identidad.
- **Alternativa rechazada**: Exponer `relationships.aliases` en create/edit/detail.

### 2. Read model `ItemView` (no contaminar `domain.Item`)

- Queries HTTP devuelven `ItemView { Item, InternalSKU *string }` vía batch lookup. `domain.Item` **no** gana campo `InternalSKU`.
- Invoices ACL (`CatalogService.GetItemNames`) **no** cambia.

### 3. Contrato HTTP

| Método  | Ruta                              | Notas                                                                                                         |
| ------- | --------------------------------- | ------------------------------------------------------------------------------------------------------------- |
| `POST`  | `/api/v1/catalog/items`           | JSON:API; `data.id` ULID obligatorio; attrs: `name`, `kind`, `internal_sku` (req); status forzado `confirmed` |
| `PATCH` | `/api/v1/catalog/items/{id}`      | attrs parciales: `name`, `kind`, `status`; `internal_sku` solo si aún no existe                               |
| `GET`   | `/api/v1/catalog/items` / `/{id}` | Incluir `internal_sku` (null/omitido si ausente); **sin** `stockable`                                         |

- Alias id del `internal_sku` en create: ULID generado en **backend**. Item id sí es del cliente.
- **Idempotencia POST**: mismo `data.id` ya existente → **409 Conflict**.
- Errores: validation (SKU faltante, ULID inválido, cambio de SKU); conflict (SKU/id duplicado).

### 4. Retirar `stockable` (YAGNI hasta inventario)

- **Enfoque**: Eliminar el campo de `domain.Item`, repositorio postgres, serializers HTTP y modelo PWA. Editar **in-place** `apps/backend/migrations/tenant/000012_catalog.up.sql` (y `.down.sql` si define la columna) para que `catalog_items` no incluya `stockable`. No añadir `000014_…`.
- **Rationale**: El flag no participa en ninguna capacidad actual; inventar stockabilidad ahora contamina el maestro. Cuando exista inventario, se modelará con el lenguaje de ese bounded context.
- **Alternativa rechazada**: Dejar columna nullable “dormant” — mantiene deuda de schema sin valor.
- **Alternativa rechazada**: Migración nueva solo para `DROP COLUMN` — innecesaria en desarrollo activo con migraciones aún editables.
- **Ops local**: tras editar `000012_*`, entornos locales deben recrear/resetear el schema tenant (o reaplicar migraciones desde cero); no se asume `migrate up` incremental sobre DBs que ya corrieron la versión con `stockable`.

### 5. Modelo táctico (Aggregate rico) — maestro desacoplado de linking

**Diagnóstico:** `Item` es parcialmente anémico — solo `NewProvisionalItem` + `IsProvisional()`; campos públicos mutables. `Alias` sin factory de internal SKU. Sin Domain Events.

#### 5.1 Consistency boundary

- **Item** = Aggregate Root. SKU interno canónico en su frontera de consistencia (confirm exige SKU; inmutable tras asignarse), persistido como fila `internal_sku` en aliases.
- Una operación de aplicación = una unidad de consistencia del Item (incl. alias internal cuando el AR lo asigna).

#### 5.2 Value Objects

| VO            | Responsabilidad                                 |
| ------------- | ----------------------------------------------- |
| `InternalSKU` | Trim/normalize; no vacío; igualdad por valor    |
| `ItemKind`    | Parse/validate `goods\|service\|asset\|unknown` |

#### 5.3 API de dominio en `Item`

- `NewManualItem(id, name, kind, sku, now) (Item, Alias, error)` — nace `confirmed` con SKU obligatorio; construye alias `internal_sku`.
- `Rename(name)` / `ChangeKind(kind ItemKind)`.
- `Confirm(currentSKU *InternalSKU, newSKU *InternalSKU) (*InternalSKU, error)` — solo desde provisional; SKU resultante obligatorio.
- `AssignInternalSKU(current *InternalSKU, next InternalSKU) error` — inmutable si ya hay current.

**Sin** `SetStockable`. **Prohibido** en commands mutar campos del AR directamente.

#### 5.4 Application (coordinador delgado)

- `CreateItemCommand` / `UpdateItemCommand`: solo `ItemRepository` + `AliasRepository`.
- Update: load Item → load current InternalSKU → `Rename` / `ChangeKind` / `Confirm` / `AssignInternalSKU` → save (+ `CreateAlias` si SKU nuevo) en **una TX**.
- Enrichment: `ListInternalSKUsByItemIDs` → `ItemView`.
- **Domain Events:** no introducir bus en este change.

#### 5.5 Factory de alias

- `NewInternalSKUAlias(id, itemID, sku InternalSKU, now)` — scheme `internal_sku`, sin party.

### 6. PWA

```
/catalog              master (+ CTA Nuevo; filas → detail)
/catalog/new          create
/catalog/:itemId      detail
/catalog/:itemId/edit edit
/catalog/review       sin cambio de intención
```

- Form: `name`, `kind`, `internal_sku` (required/readonly según modo) — **sin** stockable.
- `generateUlid()` en `core`/`shared`.

### 7. Frontera catalog vs invoices

No mover review-queue ni aliases de proveedor. ACL invoices→catalog no se ensancha.

### 8. Observabilidad mínima

- Logs `catalog.create_item` / `catalog.update_item` + `item_id`.

## Risks / Trade-offs

| Riesgo                              | Mitigación                                        |
| ----------------------------------- | ------------------------------------------------- |
| N+1 al listar `internal_sku`        | Batch / LEFT JOIN                                 |
| Typo en SKU inmutable               | Solo crear ítem nuevo; copy UI                    |
| Id duplicado en POST                | 409                                               |
| Item huérfano sin alias             | TX única                                          |
| DB local ya migrada con `stockable` | Reset/recreate tenant schema tras editar `000012` |
| Commands anémicos                   | Métodos de intención + tests de dominio           |
| Stockabilidad vuelve a colarse      | Non-goal explícito; inventario futuro             |

## Migration Plan

1. Editar `000012_catalog.up.sql` / `.down.sql` — quitar `stockable` de `CREATE TABLE catalog_items`.
2. Limpiar domain/repo/HTTP/PWA de referencias a `stockable`.
3. En local: recrear DB tenant o reaplicar migraciones desde cero (no migración incremental nueva).
4. Deploy backend (+ PWA) con POST/PATCH de maestro.

## Open Questions

- Ninguna: etiqueta UI “SKU” = internal SKU canónico.
