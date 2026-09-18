# On-prem stack

Single-VM Docker Compose payload (`apps/atta/deploy/onprem/`). Releases to a
**pool of client IPs** are a separate Pulumi track; see
[On-prem fleet](../deployment/onprem.md). AWS SaaS is parallel, not a
substitute: [AWS deploy](../deployment/aws.md).

## Services

| Service           | Role                                        |
| ----------------- | ------------------------------------------- |
| `api`             | HTTP API                                    |
| `outbox-relay`    | Drains outbox → RabbitMQ                    |
| `events-consumer` | Integration event handlers                  |
| `jobs-consumer`   | Background job handlers                     |
| `scheduler`       | Named-rule clock → RabbitMQ jobs            |
| `rabbitmq`        | Message broker                              |
| `postgres`        | Control-plane + tenant DB (seed separately) |
| `minio`           | S3-compatible object storage                |
| `caddy`           | Serves the PWA at `/`; proxies `/api*`      |

## Quick start

Build the PWA first so the Caddy image can copy
`apps/atta/pwa/dist/pwa/browser`:

```bash
pnpm --filter @atta/pwa build
cd apps/atta/deploy/onprem
cp .env.example .env
docker compose config
docker compose up -d --build
```

Run tenant migrations and seed via documented ops scripts after first boot.

Local development uses the same **onprem** profile via
`apps/atta/docker-compose.yml` (Postgres, RabbitMQ, MinIO, Caddy) and
`mise //apps/atta:dev` worker processes.

See [Runtime profiles](./runtime-profiles.md), [Outbox relay](./outbox-relay.md), [Object storage](./object-storage.md), and [Events vs jobs](./events-vs-jobs.md).
