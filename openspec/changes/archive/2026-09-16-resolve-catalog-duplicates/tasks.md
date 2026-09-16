# Tasks: resolve-catalog-duplicates

## 1. Code Specifications (BDD as Code)

- [x] 1.1 HTTP e2e `apps/e2e/tests/http/catalog/search-duplicate-clusters.spec.ts`
- [x] 1.2 HTTP e2e `apps/e2e/tests/http/catalog/create-item-merge.spec.ts` (unión de aliases, 410 merged, validación, 409 InternalCode, auth)
- [x] 1.3 HTTP e2e `apps/e2e/tests/http/catalog/create-not-duplicates.spec.ts`

## 2. Backend catalog

- [x] 2.1 Migración tenant `000021`: `status=merged`, `merged_into_id`, `catalog_item_not_duplicates`
- [x] 2.2 Dominio: `StatusMerged`, `MergeInto`, `NotDuplicatePair`; glosario Merge/survivor
- [x] 2.3 Repo TX `MergeItems` (aliases/memories; no AddItemAlias); not-duplicates; listado excluye merged
- [x] 2.4 `MergeItemsCommand` + tests (InternalCode, unión/colisión alias, idempotente)
- [x] 2.5 Query clusters (descripción, SKU cruzado, hard_conflict via port) + `MarkNotDuplicates`

## 3. Backend invoices OHS

- [x] 3.1 `invoices/api.ItemLinkSupport`: RelinkItems, HardConflictPairs, CountLinks
- [x] 3.2 Wire: `catalog.BindItemLinks` en HTTP host y messaging

## 4. HTTP catalog

- [x] 4.1 GET clusters, POST merges, POST not-duplicates; GET merged → 410 `merged_into_id`; `ERR_GONE`; rutas estáticas antes de `{id}`
- [x] 4.2 Tests HTTP handler merge/clusters/410

## 5. PWA

- [x] 5.1 HTTP/store + rutas `catalog/duplicates` y `catalog/merge`
- [x] 5.2 Estudio de fusión (cruce, survivor, impacto)
- [x] 5.3 Master multi-select + barra Fusionar + badge cola
- [x] 5.4 Cola de clusters; CTA 409 en detalle “Fusionar con este ítem”

## 6. Verification

- [x] 6.1 `pnpm --filter @bowerbird/backend test` — paquetes catalog/invoices/HTTP merge OK. Fallos restantes: smoke `tenant_acme` ausente en Postgres fresco (preexistente).
- [ ] 6.2 `pnpm run test:e2e:http` — specs listas; API local 502 (backend no levantado). Requiere migrate:all + API.
