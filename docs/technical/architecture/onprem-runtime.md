# On-prem stack

Single-VM Docker Compose deployment (`deploy/onprem/`).

## Services

| Service           | Role                                         |
| ----------------- | -------------------------------------------- |
| `api`             | HTTP API                                     |
| `outbox-relay`    | Drains outbox → RabbitMQ                     |
| `events-consumer` | Integration event handlers                   |
| `jobs-consumer`   | Background job handlers                      |
| `scheduler`       | Enqueues periodic outbox jobs (e.g. sweeper) |
| `rabbitmq`        | Message broker                               |
| `postgres`        | Control-plane + tenant DB (seed separately)  |
| `minio`           | S3-compatible object storage                 |
| `caddy`           | HTTPS reverse proxy                          |

## Quick start

```bash
cd deploy/onprem
cp .env.example .env
docker compose config
docker compose up -d
```

Run tenant migrations and seed via documented ops scripts after first boot.

Local development uses the same **onprem** profile via root `docker-compose.yml` (Postgres, RabbitMQ, MinIO, Caddy) and `pnpm run dev` worker processes.

See [Runtime profiles](./runtime-profiles.md), [Outbox relay](./outbox-relay.md), [Object storage](./object-storage.md), and [Events vs jobs](./events-vs-jobs.md).
