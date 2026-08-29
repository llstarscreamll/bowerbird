## Context

Ver `proposal.md` (Why). Estado técnico relevante:

- `catalog` posee `RememberDecision` + `ListReviewQueue` y `CatalogRepository` implementa `InvoiceLineLinkRepository` con SQL sobre `invoice_lines` / `invoice_headers`.
- `invoices` ya llama `CatalogLineResolver` / `CatalogService` vía ACL en ingest y lecturas; la PWA ya usa `app-catalog-linker` en detalle de factura y en `/catalog/review`.
- Columnas de link viven en schema de facturas (`000013_invoice_catalog_links`); match memory / aliases / items en `000012_catalog`.
- Hoy `DecideManualLink` y constantes de `link_status`/`link_method` de **línea** viven en `catalog/domain` aunque describen estado de factura; `InvoiceLineRecord` es anémico (solo persistencia).

Enfoque acordado: **B** — cola + decisions + persistencia de link en `invoices`; motor de resolución, soft match, match memory y mint en `catalog` vía puertos ACL.

**Deployment:** módulos co-deployados en el mismo proceso y DB tenant (monolito modular). Contratos ACL son in-process; no se asume deploy independiente de catalog/invoices en este change.

## Goals / Non-Goals

**Goals:**

- Eliminar reach-through SQL de `catalog` → tablas `invoice_*`.
- Dueño único de link state: repositorio + dominio `invoices`.
- HTTP y navegación de cola/decisions bajo `invoices`.
- Modelo táctico: VO `LineLink` + métodos de dominio en línea/header; command delgado.
- Partir `RememberDecision` en orquestación (invoices) + efectos de identidad (catalog).
- Migrar PWA en el mismo change (sin periodo largo de dual-write de rutas).

**Non-Goals:**

- BC `linking` separado; mover `ResolveInvoiceLine` / `match_memories` a invoices; rediseño de matching; paginación avanzada.
- Domain Events / outbox (eventual consistency cross-aggregate queda para el futuro).
- Package compartido `internal/linking` ni shared kernel de linking.

## Decisions

### 1. Dominio táctico en invoices (link state)

- **Enfoque**: Introducir VO `LineLink` con `ApplyManualLink` / `RejectLink` (guard de lock) y `RecalculateLinkingStatus` para el header.
- Constantes de estado/método de **línea** viven en `invoices/domain`. Catalog conserva solo outcomes de resolución automática (`LineResolutionResult` para ingest).
- **Alternativa rechazada**: Command que escribe `link_status`/`link_method` como strings sueltos vía repo — perpetúa anemia.
- **Alternativa rechazada**: Package compartido de linking — shared kernel basura.
- **Alternativa rechazada**: `DecideManualLink` + mapper paralelo que bypasea el guard de lock — API dual.

### 2. Frontera ACL catalog — puerto cohesivo + DTOs de contrato

- **Enfoque**: Un puerto en `invoices/application/ports`, p. ej. `CatalogMatchingPort`, con DTOs propios (sin importar `catalog/domain` en application invoices):
  - `ValidateItemExists(ctx, itemID) error`
  - `MintProvisionalFromEvidence(ctx, partyID, itemCode, description) (itemID, error)`
  - `EnsureSupplierAlias(ctx, partyID, itemCode, itemID) error`
  - `RecordMatchMemory(ctx, evidence, action link|never_match, itemID?) error`
- Implementación en `invoices/adapters/linking` delega a commands/queries de `catalog.Application` y mapea DTOs.
- `ResolveInvoiceLine` y `CatalogLineResolver` existentes no cambian de forma.
- **Alternativa rechazada**: Cuatro puertos sueltos sin contrato unificado — superficie difusa.
- **Alternativa rechazada**: Importar tipos de `catalog/domain` en application invoices — leaky boundary.

### 3. Orquestación `ApplyLineDecision` (application delgada)

```
ApplyLineDecision (invoices application)
  1. Load line + header context (repo invoices)
  2. Domain: según action → ApplyManualLink / RejectLink / (create_provisional vía ACL mint primero)
  3. ACL catalog: validate | mint | ensure alias | record memory (si remember)
  4. Persist LineLink + RecalculateLinkingStatus (repo invoices)
```

- El command **no** contiene reglas de status/method; delega al dominio invoices.
- Catalog **no** conoce `InvoiceLineRecord` ni columnas de link.

### 4. Persistencia: link repo solo en invoices

- Mover/reimplementar en postgres `invoices`: `SaveLineLink`, `ListReviewLines`, `GetLineForDecision`, `UpdateHeaderLinkingStatus`.
- `CatalogRepository` deja de implementar `InvoiceLineLinkRepository`; el puerto desaparece de catalog.
- Sin migración SQL nueva. FK `invoice_lines.item_id → catalog_items` se mantiene: integridad referencial compartida del monolito; **escritura** de `item_id` solo vía invoices tras ACL que garantiza ítem existente.

### 5. Consistencia mint + link — secuencia, sin TX cross-aggregate

- **Política fija (opción A):** una transacción = un aggregate.
  1. ACL catalog: mint / memory / alias (TX propia de catalog).
  2. Dominio invoices: aplicar `LineLink`.
  3. Repo invoices: persistir línea + header (TX propia de invoices).
- Si falla paso 3 tras mint exitoso: ítem provisional puede quedar sin link; línea sin cambio; error al cliente; recuperable desde cola/UI.
- **Alternativa rechazada**: TX compartida catalog+invoices — borra frontera de ownership y viola one-aggregate-per-transaction.
- **Alternativa rechazada**: sagas/outbox en este change.

### 6. Contratos HTTP (**BREAKING**)

| Retirar                                         | Añadir                                                                 |
| ----------------------------------------------- | ---------------------------------------------------------------------- |
| `GET /api/v1/catalog/review-queue`              | `GET /api/v1/invoicing/review-queue`                                   |
| `POST /api/v1/catalog/lines/{lineId}/decisions` | `POST /api/v1/invoicing/invoices/{invoiceId}/lines/{lineId}/decisions` |

- Payload funcional sin cambio (`action`, `item_id`, `remember`, `lock`); `data.type` = `invoice_line_decisions`.
- Cola: recursos `invoice_lines` con `relationships.invoice` (type `invoices`).
- Prefijo HTTP alineado al módulo existente (`/api/v1/invoicing/...`).
- Rutas catalog retiradas en el mismo deploy (PWA única consumidora).

### 7. PWA

| Antes                              | Después                           |
| ---------------------------------- | --------------------------------- |
| `/catalog/review`                  | `/invoices/review`                |
| CTA en maestro catálogo            | CTA / entrada en maestro facturas |
| `catalog.store` review + decisions | `invoices` store/HTTP             |

- **Mover** `app-catalog-linker` a `invoices/presentation` (o `shared` si se prefiere widget cross-feature); **no** dejarlo en catalog como dueño de decisions de factura.
- Búsqueda de ítems: sigue `GET /api/v1/catalog/items`. Decisions: HTTP de invoices.

### 8. Wire / composition

- `invoices.NewApplication` recibe `CatalogMatchingPort` (+ resolvers existentes).
- Adapters en `invoices/adapters/linking` implementan puertos contra `catalog.Application`.
- Catalog HTTP deja de registrar review-queue/decisions.

### 9. Observabilidad

- Logs estructurados atribuibles por módulo:
  - `invoices.apply_line_decision` — `line_id`, `invoice_header_id`, `action`, outcome.
  - `catalog.mint_provisional`, `catalog.record_match_memory` — `item_id` / `evidence_key`, `action`.
- Errores ACL catalog propagados con código de dominio/platform sin mutar link state de la línea.

### 10. Semántica de fallo en boundary

| Caso                        | Comportamiento                                                |
| --------------------------- | ------------------------------------------------------------- |
| Ítem no existe (link)       | Validation/404; **sin** escribir link                         |
| ACL catalog falla tras mint | Error al cliente; línea inalterada; provisional puede existir |
| Línea locked                | Conflict; sin override                                        |
| Línea no encontrada         | Not found                                                     |

## Risks / Trade-offs

| Riesgo                                                | Mitigación                                                                    |
| ----------------------------------------------------- | ----------------------------------------------------------------------------- |
| Ítem provisional sin link si falla persist invoices   | Aceptado; cola/UI re-vincula; log `catalog.mint_provisional` + error en apply |
| Duplicar reglas de link manual                        | Una sola API de VO: `ApplyManualLink` / `RejectLink` + tests                  |
| Regresión PWA                                         | Mismo PR: rutas + API + mover linker                                          |
| Solape con `catalog-manual-item-crud`                 | No tocar maestro; rebase consciente de wire catalog                           |
| Ingest sigue usando `LineResolutionResult` en catalog | Mantener; solo manual link state migra a invoices domain                      |

## Migration Plan

1. Dominio invoices (`LineLink`, mover `DecideManualLink`) + tests.
2. Puertos ACL + adapters + repo link en invoices.
3. `ApplyLineDecision` + HTTP invoices; PWA migrada.
4. Retirar SQL reach-through, HTTP linking y `DecideManualLink` de catalog.
5. Archivar change → sync specs principales.

Rollback: revertir deploy conjunto backend+PWA (breaking acoplado).

## Open Questions

Ninguna bloqueante. Resource `type` JSON:API exacto se fija en implementación siguiendo patrón invoices existente.
