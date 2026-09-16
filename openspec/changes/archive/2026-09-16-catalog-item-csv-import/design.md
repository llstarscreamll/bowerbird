## Context

El catálogo solo tiene CRUD síncrono y `ListItems` sin límite. `creation_source=import` quedó diferido. Jobs/files ya existen (outbox, presign, cursor fan-out de inbox backfill). El listado unbounded rompe con millones de ítems.

## Goals / Non-Goals

**Goals:**

- CSV UTF-8 → job chunked → upsert por `internal_code`.
- `CatalogImport` + `ImportRowError` + snapshots de actores.
- HTTP + PWA de historial, monitor, cancel, errores paginados.
- Paginación del maestro y linker.
- Purga diaria de procesos > 1 año.

**Non-Goals:** xlsx, aliases de proveedor, merge por nombre, export de errores, borrar ítems.

## Decisions

### 1. CSV only + streaming FileStore

- Presign `module=catalog`; worker `OpenFile` con Range. No `ReadFile` (carga el objeto entero).
- Delimitador `,`/`;` detectado del header. Límite 500 MiB / 5M filas.

### 2. Upsert en lote con validación de dominio

- Lookup `internal_code = ANY(...)`. Nuevos: `NewImportedItem`. Existentes: `Rename` / `ChangeKind` / `Confirm`. Persistencia UNNEST insert/update en el mismo TX que los errores del chunk.

### 3. Un import activo; cancelación cooperativa

- Unique parcial `status IN ('queued','processing')`.
- Worker relee status al entrar y al terminar el chunk; si `cancelled`, ack sin re-encolar. Sin rollback.

### 4. Errores en Postgres, no S3

- `catalog_import_errors`; GET paginado separado. Unique `(import_id, file_row, column)` para retry. `file_row` 1-based (header = 1).

### 5. Actores snapshot

- JWT `user_id`, `email`, `FirstName+LastName`. Sin FK. `cancelled_by` nullable.

### 6. Purga

- Job platform `CatalogImportPurgeRequested`. On-prem crontab `0 5 * * *` UTC; AWS `cron(0 5 * * ? *)`. DELETE CASCADE errores. No toca `catalog_items`.

### 7. List cursor `(name, id)`

- `page[size]` default 50 max 100. Search con el mismo cap.

## Risks / Trade-offs

- Chunk en vuelo (~5k) puede completar tras cancel: aceptado.
- `ILIKE %q%` en 1M filas puede ser lento; v1 sin `pg_trgm`.
- Un job platform recorre todos los tenants a las 05:00 UTC: DELETE por lotes de 1000.

## Migration Plan

Tenant `000018_catalog_item_import`: CHECK `import`; tablas `catalog_imports` y `catalog_import_errors`; índices name, created_at, unique activo.

## Open Questions

Ninguna (decisiones fijadas en el plan de producto).
