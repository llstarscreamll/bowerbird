## Context

Backend `parties`: `GET`/`PATCH`; create solo vía facturas (`provisional`). PWA read-only. Ver `proposal.md`.

## Goals / Non-Goals

**Goals:** CRUD (sin Delete); create manual `confirmed`; NIT inmutable; roles ≥1; aggregate rico; límites modulares explícitos.

**Non-Goals:** DELETE, editar NIT, filtros UI, confirmación provisionales, domain events (v1), nuevos contratos cross-module.

## Bounded context: `parties`

| Aspecto                  | Decisión                                                                                     |
| ------------------------ | -------------------------------------------------------------------------------------------- |
| **Lenguaje**             | Party, TaxID, supplier/customer roles, Contacto (UI)                                         |
| **Estado propio**        | Tabla `parties` — único writer: módulo `internal/parties`                                    |
| **Referencias externas** | `invoices.issuer_party_id`, `catalog_* .party_id` — solo ID, sin reach-through               |
| **Superficie pública**   | HTTP `/api/v1/parties`; integración sync vía `ResolveOrCreateFromIssuer` (no repo exportado) |

### Mapa de integración

```
┌─────────────┐  IssuerPartyResolver (port)   ┌──────────────┐
│  invoices   │ ───────────────────────────▶ │    parties   │
│             │   PartyResolverAdapter       │  Application │
└─────────────┘   (solo partyID string)      └──────┬───────┘
                                                    │ HTTP JSON:API
┌─────────────┐                                     ▼
│     PWA     │ ◀────────────────────────── parties CRUD
│  parties/   │
└─────────────┘

catalog ──party_id──▶ parties (FK por ID; sin importar domain parties)
```

**Regla:** `invoices`/`catalog` NO importan `parties/domain` ni `parties/adapters/repository`. ACL existente en `invoices/adapters/linking/resolvers.go` se mantiene.

### Capas internas (`internal/parties`)

```
adapters/http ──▶ application/commands|queries ──▶ domain
adapters/repository/postgres ──▶ ports.PartyRepository
wire.go = composition root (único lugar que instancia repo)
```

| Capa        | Responsabilidad                           | Prohibido                  |
| ----------- | ----------------------------------------- | -------------------------- |
| HTTP        | Auth, JSON:API map, delegar command/query | Reglas de negocio, SQL     |
| Application | Orquestación, unicidad NIT vía port       | Tipos de invoices/catalog  |
| Domain      | Invariantes, VOs, intent methods          | HTTP, DB, imports externos |
| Repository  | Persistencia `parties`                    | Lógica de negocio          |

## Diagnóstico DDD (Party)

**Severidad: Moderate**

| Issue                       | Refactor                               |
| --------------------------- | -------------------------------------- |
| `ReplaceRoles` sin guards   | `AssignRoles` con precondición         |
| Primitivos TaxID/roles      | VOs (`ItemKind`/`InternalSKU` pattern) |
| Command orquesta mutaciones | `UpdateProfile` intent method          |
| Unicidad NIT                | Application + `PartyRepository` port   |

## Modelo de dominio

```
Party (Aggregate Root)
├── TaxID      VO — inmutable post-creación
├── PartyRoles VO — ≥1 rol, supplier|customer
└── status     — mutar solo vía factories/métodos
```

| Factory                  | Entry point                 | Status        |
| ------------------------ | --------------------------- | ------------- |
| `NewProvisionalSupplier` | Bootstrap factura (interno) | `provisional` |
| `NewConfirmedParty`      | POST HTTP / PWA             | `confirmed`   |

| Método                                   | Uso               |
| ---------------------------------------- | ----------------- |
| `Rename`, `AssignRoles`, `UpdateProfile` | PATCH manual      |
| `EnsureSupplierRole`                     | Bootstrap factura |

### Application (coordinadores delgados)

```
CreatePartyCommand:  Parse VOs → repo.GetByTaxID → NewConfirmedParty → repo.Create
UpdatePartyCommand:  repo.GetByID → UpdateProfile → repo.Update
ResolveOrCreateFromIssuer: sin cambio de contrato externo
```

## PWA (`apps/pwa/src/app/parties/`)

Espejo de `catalog/`: capas aisladas, store como único orchestrator de estado, pages delgadas.

```
domain/party.model.ts
infrastructure/parties.http.service.ts  ← único acceso HTTP
application/parties.store.ts            ← estado + toasts
presentation/                           ← sin HttpClient directo
```

## Compliance (modular)

| Check                     | Estado esperado post-cambio                                          |
| ------------------------- | -------------------------------------------------------------------- |
| Superficie pública mínima | HTTP + commands existentes; repo no exportado                        |
| State isolation           | Solo `PartyRepository` toca tabla `parties`                          |
| Sin reach-through         | invoices/catalog sin SQL/joins a `parties`                           |
| Reglas en dominio/app     | Handlers/store solo mapean y delegan                                 |
| Referencia por ID         | FKs externos sin acoplar aggregates                                  |
| Fail independence         | Error create/update no afecta ingesta facturas (contextos separados) |

## Risks / Trade-offs

| Riesgo                                | Mitigación                                           |
| ------------------------------------- | ---------------------------------------------------- |
| NIT manual + factura mismo NIT        | `ResolveOrCreateFromIssuer` reutiliza; single writer |
| Adapter importa `parties/application` | Patrón ACL aceptado; no ampliar a domain/repo        |
| PATCH más estricto                    | Guard en dominio; fallo contenido al request         |

## Migration Plan

Deploy backend + PWA juntos. Sin migraciones. Integraciones existentes intactas.
