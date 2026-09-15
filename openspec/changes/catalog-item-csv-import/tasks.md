## 1. Dominio

- [x] 1.1 Añadir `CreationSourceImport` y `NewImportedItem` con tests
- [x] 1.2 Modelar `ImportActor`, `CatalogImport` (Start/RecordChunk/Complete/Fail/Cancel) y `ImportRowError` con tests

## 2. Persistencia

- [x] 2.1 Migración tenant `000018` (CHECK import, `catalog_imports`, `catalog_import_errors`, índices)
- [x] 2.2 Repo: list items con cursor, get-by-internal-codes, batch insert/update, CRUD import + batch errores + purge

## 3. Platform files

- [x] 3.1 `FileStore.OpenFile` (Range + SizeBytes) en interfaz, S3 y fakes de tests

## 4. Application / jobs

- [x] 4.1 Contratos `CatalogImportRequested` y `CatalogImportPurgeRequested`
- [x] 4.2 Commands queue/process chunk/cancel/purge (+ platform purge)
- [x] 4.3 `RegisterJobs` / `RegisterSchedules`; wire HTTP host, messaging, AWS schedule

## 5. HTTP

- [x] 5.1 Template, POST 202, GET list/detail, POST cancel, GET errors paginado
- [x] 5.2 Paginación `GET /catalog/items`

## 6. PWA

- [x] 6.1 Rutas `catalog/imports` y `catalog/imports/:importId`
- [x] 6.2 Historial, detalle (monitor, cancel, errores paginados), dialog CSV, master paginado + linker `page[size]`

## 7. Verificación

- [x] 7.1 Tests backend del change + `pnpm --filter @bowerbird/backend test`
- [x] 7.2 E2E HTTP de imports y list paginado
