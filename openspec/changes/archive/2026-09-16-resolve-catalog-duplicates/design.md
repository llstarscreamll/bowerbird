# Design: resolve-catalog-duplicates

## 1. Problem Statement

Dos `Item` del tenant representan el mismo producto (mint por proveedor, `hard_conflict` GTIN vs SKU, 409 de cruce). El operador no puede colapsarlos a un registro conservado: el cruce (`supplier_sku`/`gtin`) ya existe y el 409 solo apunta al dueño.

## 2. Technical Design (Evolutionary Modular Architecture)

Plataforma: modular monolith Go hexagonal (`internal/<bc>/`, facade `wire.go` + `api/`). P11–P19 (flat-by-aggregate) ceden ante AGENTS.md. No módulo nuevo.

### Boundaries

| Contexto   | Tipo       | Este change                                                                |
| ---------- | ---------- | -------------------------------------------------------------------------- |
| `catalog`  | Core       | Merge, not-duplicate, clusters de identidad, `status=merged`               |
| `invoices` | Core       | Re-apuntar `invoice_lines.item_id`, pares `hard_conflict`, conteo de links |
| `parties`  | Supporting | `party_id` opaco en aliases (sin JOIN)                                     |

```
  PWA ──HTTP──▶ catalog (merge / clusters / not-duplicates)
                    │
                    │ OHS sync (Customer/Supplier)
                    ▼
              invoices/api.ItemLinkSupport
                    │
                    └── invoice_lines (único writer: invoices)
```

Propiedad de tablas (P8): catalog escribe `catalog_items`, `catalog_item_aliases`, `catalog_match_memories`, `catalog_item_not_duplicates`. invoices escribe `invoice_lines`. Sin FKs nuevas cross-BC. `item_id` en líneas sigue CHAR(26).

OHS síncrono (no outbox): el operador necesita cruce + líneas alineados en la misma request. Eventual consistency dejaría `hard_conflict` huérfanos. Relink es idempotente.

Circular wire: host crea `catalogApp` → `invoicesApp(catalog OHS)` → `catalog.BindItemLinks(invoices OHS)`.

### Aggregates (catalog)

- `Item` (AR): `MergeInto(survivorID)` → `status=merged`, `merged_into_id`. No revertir confirmed→provisional. No merge de un merged (seguir cadena = 400).
- `Alias`, `MatchMemory`: ARs distintos (unicidad inter-ítem). Unión en **domain service** `MergeItems` + **una TX** de aplicación catalog (no `AddItemAlias`).
- `NotDuplicatePair`: VO `(least_id, greatest_id)`.

Survivorship de atributos en el command (defaults del plan); `creation_source` no se toca.

### invoices OHS

`invoices/api.ItemLinkSupport`:

- `RelinkItems(fromIDs, toID)` — `item_id` → conservado; conserva `link_locked`; limpia/reescribe suggestions que citaban fromIDs.
- `HardConflictPairs()` — pares de item ids distintos en `suggestions` con reason `hard_conflict`, ítems no merged.
- `CountLinks(ids)` — conteo de líneas por `item_id` (cola UI).

Catalog ports: `InvoiceLineRelinker` / `HardConflictReader` / `LinkCounter` (un port `ItemLinkSupport` en `catalog/application/ports` implementado por adapter wrapping invoices/api).

### HTTP (catalog)

```
GET  /api/v1/catalog/items/duplicate-clusters
POST /api/v1/catalog/items/merges
POST /api/v1/catalog/items/not-duplicates
```

Registrar **antes** de `GET .../items/{id}`.

- Merge body JSON:API `catalog_item_merges`: `survivor_id`, `source_ids[]` (1–4 además del survivor; total 2–5), `attributes` opcionales `{name, kind, internal_code}`. Aliases no en body.
- 200 + detalle del conservado (aliases unidos).
- GET ítem `merged`: 410 + `meta.merged_into_id`. Listado excluye `merged` por defecto.
- not-duplicates: 204. 2+ ids → todas las parejas canónicas.

### Clusters (on-read)

Grafo no dirigido, excluye merged y pares not-duplicate:

1. Misma descripción normalizada (repo catalog).
2. `HardConflictPairs` (invoices OHS).
3. Mismo `supplier_sku.value`, distinto `party_id`, distinto `item_id`.

Razón por componente: `normalized_description` | `hard_conflict` | `cross_party_sku`.

### Merge TX

1. Catalog TX: atributos del conservado; `MergeInto` sources; re-apuntar aliases (colisión tuple → DELETE source alias); re-apuntar memories (`link` gana a `never_match` en misma evidence key).
2. `RelinkItems`. Si falla: 5xx; merge catalog ya committed; retry del POST es idempotente si sources ya `merged_into` survivor.

### PWA

Rutas `catalog/duplicates`, `catalog/merge` antes de `:itemId`. Estudio columnas=ítems; cruce unión read-only. CTA 409 en detalle. Master checkboxes 2–5 + barra Fusionar. Cola de clusters.

### Non-goals (diseño)

Auto-merge, unmerge, fuzzy, eventos outbox, CSV de cruce, cambiar POST/DELETE aliases, catalog escribiendo `invoice_lines`, reorg flat-by-aggregate.
