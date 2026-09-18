# Database migrations

Uses `golang-migrate` with two trees under `apps/atta/backend/migrations/`:

| Folder          | Applies to            |
| --------------- | --------------------- |
| `controlplane/` | Shared catalog DB     |
| `tenant/`       | Every organization DB |

CLI: `apps/atta/backend/cmd/onprem/migrate/main.go`.

```bash
mise //apps/atta:migrate:controlplane   # control plane only
mise //apps/atta:migrate:tenants        # all active tenants
mise //apps/atta:migrate:all            # control plane, then tenants
```

`migrate:tenants` loads `db_name` from `tenants WHERE status = 'active'` and migrates each DB with `migrations/tenant`.

Onboarding also runs tenant migrations immediately after `CREATE DATABASE`
(see [Onboarding](./onboarding-flow.md)). On AWS, `CREATE DATABASE` and
golang-migrate use Neon's **direct** connection URL
(`database_direct_url`). Lambdas use the pooled URL for ordinary queries.

## AWS (Neon)

`mise //apps/atta:deploy:aws` (or `mise //apps/atta:deploy`) invokes the migrate
Lambda when that function's package changes (it ships
`migrations/controlplane` and `migrations/tenant`). The invoke runs
**before** Pulumi updates the other Lambdas and web objects, and uses
Neon's **direct** URL (`database_direct_url`).

If the invoke fails, Pulumi stops the apply so application code is not
released against an unmigrated schema. Re-run out of band with:

```bash
pnpm --filter @atta/infra migrate
```

## Local full reset

Automated full suite (Postgres reset, migrations, Go + PWA unit tests, Playwright
e2e via `mise //apps/atta:test:full`):

```bash
mise //apps/atta:test:full
```

That wipes the **Postgres volume only**, then migrates and seeds. It does
not remove MinIO or Caddy volumes.

Nuclear reset (Postgres, MinIO, RabbitMQ, and Caddy volumes):

```bash
docker compose -f apps/atta/docker-compose.yml down -v
mise //apps/atta:infra:up
mise //apps/atta:migrate:all
```

Re-seed with `mise //apps/atta:seed` if needed.

Postgres-only (same data wipe as `test:full`, without running tests):

```bash
docker compose -f apps/atta/docker-compose.yml down
docker volume rm atta_postgres_data
mise //apps/atta:infra:up
mise //apps/atta:migrate:all
```
