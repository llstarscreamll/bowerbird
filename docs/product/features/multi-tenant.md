# Multi-organization (multi-tenant)

One platform serves many customer organizations with full data isolation.

## Value

- Physical DB isolation so financial data never mixes across customers.
- Path-based org context in the app (for example `/acme/...`).
- Shared collaboration inside one organization on a single source of truth.

## Behavior

### Access

- Org context comes from the first URL segment (`/:tenantId`), not a global “pick company” menu before login.
- The PWA injects `X-Tenant-ID` on API calls from that segment.

### Data isolation

- Each organization has its own PostgreSQL database.
- Backups, restores, and retention can run per organization.

### Async work

- Background jobs carry `TenantID` and resolve the same DB routing as HTTP requests.

## Naming

- Product / UI: **Organization**
- Code / infra: **Tenant**
- Do not use Account or Workspace for the customer entity.
