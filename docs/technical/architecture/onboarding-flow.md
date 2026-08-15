# Organization onboarding

Provisioning a new tenant (organization) is more than an insert: create identity, create a database, migrate schema.

## Use case

`apps/backend/internal/organization/application/create.go`:

1. Validate slug (alphanumeric + hyphens); derive `db_name` (for example `tenant_acme_corp`).
2. Ensure slug uniqueness in the control plane.
3. Insert into `tenants` (reserves the slug).
4. `CREATE DATABASE` on the cluster.
5. Run `golang-migrate` with `migrations/tenant` against the new DB.

## Failure modes

`CREATE DATABASE` cannot share a transaction with the control-plane insert.

- Control plane insert OK, DB create fails → orphaned catalog row; HTTP 500.
- DB create OK, migrate fails → empty / partial schema.

**Recovery:** `pnpm run migrate:all` walks active tenants and finishes missing migrations (also used in deploy pipelines).

## API

```http
POST /api/v1/organizations
Content-Type: application/json

{ "name": "Wayne Enterprises", "slug": "wayne" }
```

`201` returns `id`, `name`, `slug`, `status`, `created_at`. The org is then reachable under its tenant path in the app.
