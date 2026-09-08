# Database migrations

Uses `golang-migrate` with two trees under `apps/backend/migrations/`:

| Folder          | Applies to            |
| --------------- | --------------------- |
| `controlplane/` | Shared catalog DB     |
| `tenant/`       | Every organization DB |

CLI: `apps/backend/cmd/onprem/migrate/main.go`.

```bash
pnpm run migrate:controlplane   # control plane only
pnpm run migrate:tenants        # all active tenants
pnpm run migrate:all            # control plane, then tenants
```

`migrate:tenants` loads `db_name` from `tenants WHERE status = 'active'` and migrates each DB with `migrations/tenant`.

Onboarding also runs tenant migrations immediately after `CREATE DATABASE`
(see [Onboarding](./onboarding-flow.md)). On AWS, `CREATE DATABASE` and
golang-migrate use Neon's **direct** connection URL
(`database_direct_url`). Lambdas use the pooled URL for ordinary queries.

## AWS (Neon)

`pnpm run deploy` invokes the migrate Lambda when that function's
package changes (it ships `migrations/controlplane` and
`migrations/tenant`). The invoke runs **before** Pulumi updates the
other Lambdas and web objects, and uses Neon's **direct** URL
(`database_direct_url`).

If the invoke fails, Pulumi stops the apply so application code is not
released against an unmigrated schema. Re-run out of band with:

```bash
pnpm --filter @bowerbird/infra migrate
```

## Local full reset

```bash
docker compose down -v
pnpm run infra:up
pnpm run migrate:all
```

Wipes Postgres, MinIO, RabbitMQ, and Caddy volumes. Re-seed with `pnpm run seed` if needed.

Postgres-only:

```bash
docker compose down
docker volume rm bowerbird_postgres_data
pnpm run infra:up
pnpm run migrate:all
```
