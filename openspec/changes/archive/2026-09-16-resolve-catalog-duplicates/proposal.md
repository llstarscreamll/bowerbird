## Why

El catálogo ya cruza `supplier_sku` y GTIN, pero dos maestros pueden representar el mismo bien (mint por proveedor, `hard_conflict` GTIN vs SKU, 409 de alias). El operador no puede colapsarlos a un ítem conservado.

## What Changes

- Merge N-vías (2–5) a un ítem conservado: `status=merged`, `merged_into_id`, unión de aliases en la TX (no `AddItemAlias`).
- Cola de clusters on-read: descripción normalizada, SKU cruzado entre parties, pares `hard_conflict`.
- `not-duplicates` persistido (ids canónicos).
- invoices OHS `ItemLinkSupport`: re-apuntar líneas, pares de conflicto, conteo.
- PWA: estudio de fusión, cola `catalog/duplicates`, multi-select en listado, CTA 409 en detalle.
- GET merged → 410 `ERR_GONE` + `meta.merged_into_id`.

## Non-goals / out of scope

- Auto-merge, unmerge, fuzzy/ML, variantes como entidad.
- Merge de parties. CSV de cruce. Precio/stock.
- Cambiar POST/DELETE de aliases.

## Capabilities

### Modified Capabilities

- `catalog`: merge, clusters, not-duplicates, `status=merged`.
- `invoices`: relink de `invoice_lines.item_id` y suggestions.

## Impact

- Backend catalog + invoices OHS + host/messaging bind.
- PWA catálogo (master, detalle, duplicates, merge).
- E2E HTTP clusters/merge/not-duplicates.
- Glosario: Fusionar duplicados / Ítem conservado.
