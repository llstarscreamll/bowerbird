# Runtime profiles

`DEPLOYMENT_TARGET` selects infrastructure adapters at boot. Application code (use cases, domain, contracts) stays the same; only platform wiring changes.

| Profile  | When                                    | Messaging         | Object storage        | Secrets               |
| -------- | --------------------------------------- | ----------------- | --------------------- | --------------------- |
| `onprem` | Local dev, client VM (`deploy/onprem/`) | RabbitMQ          | MinIO (S3-compatible) | Plain `.env`          |
| `aws`    | Production SaaS (CDK)                   | EventBridge + SQS | AWS S3                | SSM SecureString JSON |

See [On-prem stack](./onprem-runtime.md), [AWS deploy](../deployment/aws.md), [SSM secrets JSON](../deployment/ssm-secrets.md).

## End-to-end flow (both profiles)

```text
HTTP handler ──► use case ──► outbox (same DB tx) ──► relay ──► broker ──► consumer ──► handler
```

Application code never publishes directly to RabbitMQ, EventBridge, or SQS. See [Outbox relay](./outbox-relay.md) and [Events vs jobs](./events-vs-jobs.md).

## Local infrastructure

Root `docker-compose.yml` (started by `pnpm run infra:up`):

| Service      | Role                                                    |
| ------------ | ------------------------------------------------------- |
| `postgres`   | Control-plane + tenant DB                               |
| `rabbitmq`   | Message broker (`:5672`, management UI `:15672`)        |
| `minio`      | S3-compatible object storage (`:9000`, console `:9001`) |
| `minio-init` | One-shot bucket bootstrap (`infra:up` profile)          |
| `caddy`      | HTTPS reverse proxy (`network_mode: host`)              |

Caddy routes (see root `Caddyfile`):

- `app.bowerbird.dev` → Angular `:4200`
- `api.bowerbird.dev` → Go API `:8080`
- `media.bowerbird.dev` → MinIO `:9000` (presigned uploads/downloads)

There is **no LocalStack** and **no Redis** in the local stack.

## Process model

### On-prem / local dev

| Process         | Entrypoint             | Dev runner                                   |
| --------------- | ---------------------- | -------------------------------------------- |
| HTTP API        | `cmd/api`              | `pnpm --filter @bowerbird/backend dev` (Air) |
| Outbox relay    | `cmd/worker relay`     | `dev:relay`                                  |
| Events consumer | `cmd/worker events`    | `dev:events-consumer`                        |
| Jobs consumer   | `cmd/worker jobs`      | `dev:jobs-consumer`                          |
| Scheduler       | `cmd/worker scheduler` | `dev:scheduler`                              |

Root `pnpm run dev` starts infra, API, all four workers, and the PWA via Turbo.

Workers and the API use **Air** hot reload (`.air.toml`, `.air.worker-*.toml`).

### AWS

| Process         | Entrypoint                                                      |
| --------------- | --------------------------------------------------------------- |
| HTTP API        | Lambda (`cmd/api` build)                                        |
| Outbox relay    | Lambda (`cmd/lambda/outbox-relay`), EventBridge schedule (~30s) |
| Events consumer | Lambda (`cmd/lambda/eventbridge`)                               |
| Jobs consumer   | Lambda (`cmd/lambda/sqs`)                                       |

## Platform adapters (by concern)

| Concern          | Package                                        | `onprem`          | `aws`                                |
| ---------------- | ---------------------------------------------- | ----------------- | ------------------------------------ |
| Config / secrets | `internal/platform/config`                     | `.env`            | SSM JSON merge                       |
| Events publish   | `internal/platform/outbox` + `events/adapters` | RabbitMQ topic    | EventBridge                          |
| Jobs enqueue     | `internal/platform/outbox` + `jobs/adapters`   | RabbitMQ direct   | SQS                                  |
| Broker transport | `internal/platform/messaging`                  | AMQP              | AWS SDK                              |
| Object storage   | `internal/platform/storage/s3`                 | MinIO endpoint    | AWS S3                               |
| Scheduler        | `internal/platform/scheduler`                  | In-process worker | EventBridge rules (where applicable) |

Feature modules wire handlers in `wire.go`; `internal/platform/messaging.WireMessagingHandlers` registers shared consumer routes.

## Broker topology (onprem)

| Channel            | RabbitMQ                                                |
| ------------------ | ------------------------------------------------------- |
| Integration events | Exchange `bowerbird.events` (topic)                     |
| Background jobs    | Exchange `bowerbird.jobs` → queue `bowerbird.jobs.work` |
| Dead letters       | DLX `bowerbird.dlx` → queue `bowerbird.deadletter`      |

Job queue bindings are declared at worker boot from registered `JobHandler.JobType()` values.

## Related

- [Backend architecture](./backend-api.md)
- [Object storage](./object-storage.md)
- [Messaging troubleshooting](./messaging-troubleshooting.md)
- [MinIO](../tooling/minio.md)
