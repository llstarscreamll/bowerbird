## Why

El módulo de Contactos (`parties`) solo permite listar; la spec y la UI prometen CRUD pero faltan create/update en API y toda la PWA salvo la tabla. Sin alta y edición manual, el tenant no puede registrar clientes/proveedores antes de recibir facturas.

## What Changes

- `POST /api/v1/parties` — alta manual (status `confirmed`, NIT obligatorio e inmutable).
- PWA: rutas `parties/new`, `parties/:id`, `parties/:id/edit`; formulario con checkboxes de roles (mínimo uno); master con botón "Nuevo Contacto" y filas navegables.
- Store/HTTP: `get`, `create`, `patch` en frontend.
- Dominio rico (VOs, intent methods) dentro del bounded context `parties`; HTTP/PWA delgados; integración con `invoices` sin cambios de contrato (`IssuerPartyResolver`).

## Non-goals

- Eliminar contactos (`DELETE`).
- Editar NIT post-creación.
- Búsqueda/filtros en UI (API ya los soporta; fase posterior).
- Confirmación explícita de contactos `provisional` (bootstrap por factura sigue igual).

## Capabilities

### New Capabilities

_(ninguna — se extiende la capability existente)_

### Modified Capabilities

- `parties`: create HTTP, reglas de status manual vs bootstrap, UI CRUD, inmutabilidad de NIT.

## Impact

| Área        | Módulos                                                                     |
| ----------- | --------------------------------------------------------------------------- |
| Backend     | `internal/parties/` (sin tocar `invoices`/`catalog` salvo wiring existente) |
| Integración | `IssuerPartyResolver` sin cambio; create manual solo vía HTTP               |
| PWA         | `apps/pwa/src/app/parties/`, `app.routes.ts`                                |
| Spec        | `openspec/specs/parties/spec.md` (delta)                                    |

Sin migraciones, eventos ni cambios de infra.
