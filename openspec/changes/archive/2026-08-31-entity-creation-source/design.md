## Context

`parties` y `catalog_items` tienen dos rutas de creación (PWA manual vs pipeline de facturas) pero no persisten el canal de primer nacimiento. Ver `proposal.md`.

## Goals / Non-Goals

**Goals:**

- Campo `creation_source` inmutable (`manual` | `invoice`) en ambas tablas.
- Propagación desde factories/commands de cada entry point.
- Exposición en API + filtro list + UI PWA.

**Non-Goals:**

- `source_invoice_header_id`, enum `import`, migraciones nuevas, backfill, compatibilidad con datos existentes.

## Decisions

### 1. Columna en migraciones existentes (no nueva migración)

Añadir en las definiciones `CREATE TABLE` originales:

```sql
-- 000011_parties.up.sql
creation_source VARCHAR(32) NOT NULL DEFAULT 'manual',
CONSTRAINT parties_creation_source_check CHECK (creation_source IN ('manual', 'invoice'))

-- 000012_catalog.up.sql
creation_source VARCHAR(32) NOT NULL DEFAULT 'manual',
CONSTRAINT catalog_items_creation_source_check CHECK (creation_source IN ('manual', 'invoice'))
```

Índices opcionales `ix_parties_creation_source` / `ix_catalog_items_creation_source` si el filtro list lo justifica (bajo volumen inicial; añadir si queries lo requieren).

**Alternativa rechazada:** migración `000014_*` — el usuario prefiere editar las existentes al no preocuparse por retrocompatibilidad.

### 2. Enum mínimo: `manual` | `invoice`

| Entry point                                                   | Entidad | Valor     |
| ------------------------------------------------------------- | ------- | --------- |
| `CreatePartyCommand` / PWA                                    | Party   | `manual`  |
| `NewConfirmedParty`                                           | Party   | `manual`  |
| `ResolveOrCreateFromIssuerCommand` / `NewProvisionalSupplier` | Party   | `invoice` |
| `CreateItemCommand` / `NewManualItem`                         | Item    | `manual`  |
| `mintProvisional` / `NewProvisionalItem`                      | Item    | `invoice` |
| `ApplyLineDecision` mint path                                 | Item    | `invoice` |

**Alternativa rechazada:** inferir por `status` o aliases — se pierde tras confirmación de ítems.

**Alternativa rechazada:** `import` reservado en CHECK — se añadirá cuando exista el change de CSV.

### 3. Inmutabilidad en dominio

- `Party` y `Item` structs ganan campo `CreationSource string`.
- Constantes: `CreationSourceManual`, `CreationSourceInvoice`.
- Factories reciben/setean el valor; métodos de update (`Rename`, `Confirm`, `UpdateProfile`, etc.) no lo tocan.
- Repository `UPDATE` no incluye `creation_source` en `SET`.

### 4. API JSON:API

Atributo `creation_source` en resources `parties` y `catalog_items` (list + get).

Query param en list: `?creation_source=manual|invoice` (rechazar valores desconocidos con validación o ignorar silenciosamente — preferir validación 400 siguiendo patrón de filtros existentes).

Clients no pueden enviar `creation_source` en create/update body; el servidor lo deriva del entry point.

### 5. PWA

- Models: `creation_source: string` en `party.model.ts`, `catalog.model.ts`.
- Helper de label: `manual` → "Manual", `invoice` → "Desde factura".
- Master: columna adicional; detail: campo en atributos.
- Sin filtro UI en v1 (API lista soporta; filtro UI es non-goal salvo que tasks lo incluyan mínimamente — spec no exige filtro UI, solo display).

### 6. Integración invoices

Sin cambio de ports cross-module. `PartyResolverAdapter` y `CatalogMatchingAdapter` delegan a commands internos que ya setean `invoice`. No propagar `creation_source` por strings en ACL.

```
PWA POST ──▶ CreatePartyCommand ──▶ manual
Invoice ──▶ ResolveOrCreateFromIssuer ──▶ invoice
PWA POST ──▶ CreateItemCommand ──▶ manual
Invoice ──▶ ResolveInvoiceLineCommand.mintProvisional ──▶ invoice
```

## Risks / Trade-offs

| Riesgo                                                        | Mitigación                                           |
| ------------------------------------------------------------- | ---------------------------------------------------- |
| Editar migraciones existentes rompe tenants con DB ya migrada | Aceptado explícitamente; solo entornos fresh/local   |
| Futuro CSV requiere nuevo valor enum                          | Nueva migración + change dedicado cuando llegue      |
| Analítica sin backfill en datos viejos                        | Aceptado; solo registros creados post-implementación |

## Migration Plan

1. Editar `000011_parties.up.sql` y `000012_catalog.up.sql`.
2. Recrear schema tenant local (`migrate:all` en entorno dev).
3. Desplegar backend + PWA en lockstep (API devuelve campo nuevo).

Sin rollback script; revert = revertir commit y recrear DB.
