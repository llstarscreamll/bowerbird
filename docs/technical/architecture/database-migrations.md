# Database migrations

Uses `golang-migrate` with two trees under `apps/backend/migrations/`:

| Folder          | Applies to            |
| --------------- | --------------------- |
| `controlplane/` | Shared catalog DB     |
| `tenant/`       | Every organization DB |

CLI: `apps/backend/cmd/migrate/main.go`.

```bash
pnpm run migrate:controlplane   # control plane only
pnpm run migrate:tenants        # all active tenants
pnpm run migrate:all            # control plane, then tenants
```

`migrate:tenants` loads `db_name` from `tenants WHERE status = 'active'` and migrates each DB with `migrations/tenant`.

Onboarding also runs tenant migrations immediately after `CREATE DATABASE` (see [Onboarding](./onboarding-flow.md)).

## Local full reset

```bash
docker compose down -v
pnpm run infra:up
pnpm run migrate:all
```

Wipes Postgres, Redis, LocalStack, and Caddy volumes. Re-seed with `pnpm run seed` if needed.

Postgres-only:

```bash
docker compose down
docker volume rm bowerbird_postgres_data
pnpm run infra:up
pnpm run migrate:all
```
