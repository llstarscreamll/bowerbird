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
3. Registry caches pools (`sync.Map`); on miss, resolves `db_name` from the control plane and opens a pool.

### Shared AWS resources

SQS / EventBridge / S3 are shared with logical isolation. Async handlers restore `TenantID` from message attributes into context so DB routing matches HTTP.

CORS allows app origins for the configured root domain.

## Language

- Product / UI: **Organization**
- Code / headers: **Tenant**
