## Context

See proposal.md — Why. Plataforma existente: modular monolith Go hexagonal (`internal/<bc>/`, facade `wire.go` + `api/`). Este change **revisa** el cruce catalog↔invoices; no es un módulo nuevo ni un rediseño a flat-by-aggregate (P11–P19 ceden ante AGENTS.md y P1). Specs: `specs/catalog/spec.md`, `specs/invoices/spec.md`.

Hoy el cruce ya cruza contextos por OHS (`catalog.api.InvoiceSupport`) y ACL (`invoices/adapters/linking.CatalogACL`). El diseño previo filtraba GTIN “reutilizable” entre módulos y dejaba que invoices orquestara Alias + Memory por separado — eso viola P1/P5.

## Goals / Non-Goals

**Goals:**

- Cruce como capacidad de **catalog** (identidad); evidencia UBL como VO de **invoices** (documento).
- Contrato OHS por **capacidades**, no por tablas (`RememberDecision`, no dos writes de alias/memory desde invoices).
- ACL UBL→`LineIdentifiers` solo en invoices; catalog no ve XML ni tipos UBL.
- Estado: cada BC escribe solo sus tablas; referencias cross-context por id.

**Non-Goals (diseño):** Domain Events nuevos para linking (OHS síncrono se mantiene a propósito); Shared Kernel de GTIN; vigencia; CSV de cruce; merge; drop de FKs históricas en este change; reorganizar carpetas a flat-by-aggregate; circuit breakers en llamadas in-process al mismo deploy.

## Boundaries

### Clasificación

| Contexto   | Tipo       | Responsabilidad en este change                          |
| ---------- | ---------- | ------------------------------------------------------- |
| `catalog`  | Core       | Ítem, Alias, MatchMemory, resolución, mint, enseñanza   |
| `invoices` | Core       | Factura, evidencia de línea, `LineLink`, unlock, ingest |
| `parties`  | Supporting | `party_id` opaco para scope de `supplier_sku`           |

### Mapa

```
  DIAN UBL ──ACL parser──▶ invoices (LineIdentifiers + LineLink)
                               │
                               │ Customer / OHS  (sync, DTOs)
                               ▼
                         catalog.api.InvoiceSupport
                               │
                               ├── Item + Alias + Memory  (mismo BC, 1 TX al enseñar)
                               └── party_id  (id only; no JOIN parties)

  PWA ──HTTP──▶ catalog | invoices | parties   (composición en cliente)
```

Invoices **no** importa `catalog/domain` ni `catalog/application`. Solo `catalog/api` vía adapter `CatalogACL`, que traduce `ports.CatalogMatchingPort` (lenguaje invoices) ↔ DTOs OHS.

### Propiedad de tablas (P8)

| Tabla                                                               | Writer   |
| ------------------------------------------------------------------- | -------- |
| `catalog_items`, `catalog_item_aliases`, `catalog_match_memories`   | catalog  |
| `invoice_headers`, `invoice_lines` (ids UBL **y** columnas de link) | invoices |
| `parties`                                                           | parties  |

**Deuda P8 (no tocar aquí):** FKs existentes `invoice_lines.item_id → catalog_items`, `catalog_item_aliases.party_id → parties`, `invoice_headers.issuer_party_id → parties`. Este change **no** añade FKs nuevas, **no** hace JOIN a `parties` desde catalog, **no** escribe tablas ajenas. `item_id` / `party_id` son CHAR(26) validados por OHS/código.

### Por qué no eventos de linking (vs outbox)

OHS síncrono es el patrón Customer/Supplier ya desplegado: el operador necesita el link en la misma request de ingest/decisión. Eventual consistency rompería la cola de revisión. `linking_status` ya aísla fallo de matching del write financiero (P10). Evolución: el OHS puede publicarse luego como `catalog.item.identity-remembered` sin cambiar invoices si el adapter se queda.

## Decisions

### 1. Evidencia UBL vive en invoices; GTIN no es Shared Kernel

- **Enfoque:** VO `LineIdentifiers{BuyerCode, SellerSKU, GTIN}` en dominio invoices. El parser (ACL DIAN) solo traduce XML → VO. Persistencia: tres columnas; drop `item_code`.
- **Published Language (regla, no paquete):** GTIN = dígitos 8/12/13/14 tras strip. Invoices la aplica al construir el VO; catalog la aplica en `NewGTINAlias`. Dos implementaciones, mismos tests de ejemplos. **Prohibido** `internal/shared/gtin` o import invoices→catalog / catalog→invoices para clasificar.
- **Fallback seller:** Sellers vacío y Standard no-GTIN → `SellerSKU = Standard` (traducción ACL, no identidad GS1).
- **Rechazado:** función “reutilizable” compartida (P19 accidental). JSONB. Conservar `item_code`. Tratar todo Standard como GTIN.

### 2. Alias no entra al AR Item (siguen dos aggregates en catalog)

- **Enfoque:** `Alias` entidad propia. Unicidad `(scheme, party, value)` es **entre** ítems → Domain Service + unique index, no frontera de `Item`. `MatchMemory` es otro AR. HTTP add/remove = un Alias por request.
- **Enseñar (mismo BC):** un command de aplicación catalog `RememberDecision` abre **una TX** y escribe Alias(es) + Memory. Invoices no orquesta esos writes (P5).
- **Rechazado:** aliases como hijos de `Item`. JSON:API `included`. Extraer un BC `matching` (misma lengua, mismos ciclos de cambio).

### 3. HTTP catalog de aliases (superficie del BC dueño)

```
GET    /api/v1/catalog/items/{id}                  → attributes.aliases[] (id, scheme, value, party_id, source)
POST   /api/v1/catalog/items/{id}/aliases          → 201; client ULID; attrs: scheme, value, party_id?
DELETE /api/v1/catalog/items/{id}/aliases/{aliasId} → 204
```

- Maestro **sin** aliases embebidos. Search `ILIKE` en `catalog_item_aliases` (tabla propia).
- 409 `WithMeta("item_id", ownerID)`.
- PATCH ítem sin aliases.
- PWA resuelve nombre de party por HTTP de **parties** (id), no catalog.

### 4. OHS catalog: capacidades, no storage

Contrato `catalog.api.InvoiceSupport` (BREAKING). DTOs primitivos (strings/ids), nunca VOs de invoices ni entidades de catalog.

| Capacidad                                                                               | Sustituye                                                |
| --------------------------------------------------------------------------------------- | -------------------------------------------------------- |
| `ResolveLine(LineID, PartyID, BuyerCode, SellerSKU, GTIN, Description, existing link…)` | `ItemCode` único                                         |
| `RememberDecision(PartyID, SellerSKU, GTIN, Description, Action, ItemID)`               | `EnsureSupplierAlias` + `RecordMatchMemory` por separado |
| `MintProvisionalFromEvidence(PartyID, SellerSKU, GTIN, Description)`                    | `ItemCode`; internamente ítem+aliases en 1 TX            |
| `ValidateItemExists` / displays                                                         | igual                                                    |

`RememberDecision`: no-op de alias si seller unusable; omite GTIN vacío; 409 si un alias es de otro ítem (TX abort, invoices no persiste el link). Memory clavea seller usable (o solo description). `source=invoice` en aliases creados por esta vía.

Adapter `CatalogACL`: único lugar que mapea `invoices/application/ports` ↔ `catalog/api`. El command `ApplyLineDecision` habla el puerto invoices (`RememberDecision` / `Mint…`), no el OHS crudo.

- **Rechazado:** `EnsureLineAliases` público (filtra persistencia). Invoices llamando dos commands catalog.

### 5. Pipeline de resolución (catalog, Domain Service)

`HardHits.Agree()` en dominio catalog (cruza Item.internal_code + Alias; no es un tercer BC):

1. Lock → preserve (invoices ya mandó estado existente).
2. Memory por evidencia seller/description.
3. Hits independientes: buyer=`internal_code`; gtin alias; supplier_sku+party.
4. Acuerdo → `hard`. Conflicto → `suggested` `hard_conflict`, sin mint.
5. Cero hits → mint si party + `SellerSKU.Usable()`; adjuntar GTIN si no conflicto.
6. Soft suggest.

`Usable()`: trim; len≥3; denylist `{1,01,001,n/a,na,serv,servicio,item}` case-insensitive.

### 6. LineLink pertenece a invoices

`ApplyLineDecision` `action`: `link` | `never_match` | `create_provisional` | `unlock`.

Orden (P10, igual que hoy): **catalog OHS primero** → si ok, persistir `LineLink`. Unlock no llama catalog.

- `remember=false`: no `RememberDecision`.
- PWA: Recordar/Bloquear default on; preview; Desbloquear.

### 7. Seam merge (otro change; consumidor del invariante)

- Mover aliases B→A; tuple duplicado en A: drop del de B.
- Memory `item_id` → ganador.
- `invoice_lines.item_id` del perdedor → ganador **solo** si `link_locked=false` (invoices escribe sus filas; merge catalog **no** UPDATE `invoice_lines` — invoices expone OHS/`RelinkUnlockedLines(from,to)` o el merge se implementa como orquestación en application host). **Decisión:** merge catalog no será writer de `invoice_lines` (P8). El change de merge llamará un puerto invoices, no SQL cruzado.
- Este change no fusiona. 409 + `meta.item_id` es la pista UX.

### 8. Sin jobs/eventos/infra nueva

Resolución in-process post-insert. Idempotencia: `RememberDecision` y mint son seguros ante retry (unique alias + upsert memory). No backfill de re-link. No schedule AWS.

## Risks / Trade-offs

- **[Risk] Backfill `item_code` mal clasificado** → misma regla Published Language; Standard-no-GTIN → seller. Ruido residual aceptado.
- **[Risk] Memory histórica con código colapsado deja de pegar** → BREAKING aceptado; locked no se toca.
- **[Risk] GTIN-13 vs 14 duplica** → sin pad.
- **[Risk] Buyer code falso positivo** → conflicto si hay otro hit; si es único, auto-link + corrección humana.
- **[Risk] FKs cross-BC existentes** → no ampliar; merge futuro no las usa como licencia para writes ajenos.
- **[Trade-off] OHS sync vs eventos** → UX de ingest; P10 vía `linking_status` en el write financiero.
- **[Trade-off] GTIN duplicado en dos dominios** → 15 líneas ×2 vs Shared Kernel. Gana P1.
- **[Trade-off] PWA N+1 parties** → N aliases bajo; no JOIN en catalog.

## Migration Plan

Tenant `000019_catalog_item_crosswalk` (DDL de **ambas** tablas en un archivo de migración tenant: el runner es platform; cada BC sigue siendo el único writer en runtime):

1. `catalog_item_aliases.source` NOT NULL DEFAULT `invoice` + CHECK.
2. `invoice_lines.buyer_code`, `seller_sku`, `gtin`.
3. Backfill: GTIN si `item_code` pasa la regla; si no, `seller_sku = item_code`.
4. `DROP item_code`.
5. Down: `item_code = COALESCE(NULLIF(seller_sku,''), gtin, buyer_code)`.

API + PWA mismo release. Rollback = migración down + binarios previos.

## Open Questions

Ninguna que altere specs. El puerto invoices para relink en merge se especifica en **ese** change; aquí solo queda el veto P8.
