# Multi-tenancy

Database-per-tenant plus a shared control plane.

## Control plane

Shared Postgres DB: tenant catalog (`slug` → `db_name`), platform metadata, global operators.

## Data plane

One Postgres database per organization. All operational business tables live here.

## Resolution

### Frontend

- Routes under `/:tenantId/...` inside `TenantLayoutComponent`.
- `tenant.interceptor.ts` reads the first path segment (skipping global routes like `/login`) and sets `X-Tenant-ID`.

### Backend

1. `tenant.Middleware` reads `X-Tenant-ID` into `context.Context`.
2. Repositories ask the tenant `Registry` for a `pgxpool`.
3. Registry caches one pool per tenant database (`db_name`). On miss, it resolves `db_name` from the control plane and opens a pool.

Pools are lazy (`MinConns=0`, `MaxConns=4`, idle 30s). The outbox relay lists every active tenant on each tick and `GetPool`s them, so a reserved `MinConns` per tenant would be held by API + relay + consumers + scheduler at once and exhaust Postgres (`53300 too many clients`) as soon as e2e (or any flow) creates many tenants.

### Shared AWS resources

SQS / EventBridge / S3 are shared with logical isolation. Tenant job
and event handlers restore `TenantID` from message attributes into
context so DB routing matches HTTP. Platform jobs (scheduler ticks)
have no tenant on the message; the handler lists active tenants and
sets tenant context per iteration.

CORS allows app origins for the configured root domain.

## Language

- Product / UI: **Organization**
- Code / headers: **Tenant**
