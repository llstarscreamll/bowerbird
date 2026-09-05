# Backend architecture

## Stack

- Go `net/http` (no web framework)
- PostgreSQL via `pgx` and raw SQL
- Dual runtime: **onprem** (RabbitMQ + MinIO + worker processes) or **aws** (EventBridge + SQS + Lambdas + S3)

See [Runtime profiles](./runtime-profiles.md) for the full deployment comparison.

## Layout

`internal/` splits platform utilities from business modules.

### `internal/platform/`

Cross-cutting adapters only (no business logic):

| Area           | Packages                                                             |
| -------------- | -------------------------------------------------------------------- |
| Config         | `config` — env + optional SSM merge                                  |
| Database       | `database` — pools, transactions                                     |
| Messaging      | `messaging`, `events/adapters`, `jobs/adapters` — broker abstraction |
| Outbox         | `outbox`, `outbox/relay` — transactional publish                     |
| Object storage | `storage/s3` — S3/MinIO `FileStore`                                  |
| Scheduler      | `scheduler` — periodic outbox job ticks (onprem worker)              |

Broker and storage implementations are selected by `DEPLOYMENT_TARGET` at composition root (`internal/platform/wire.go`).

### Feature modules (bounded contexts)

Every bounded context lives under `internal/<bc>/`:

```text
internal/<bc>/
  wire.go         # only Go facade other packages import
  api/            # Open Host Service (interfaces + DTOs)
  domain/         # rich model; no infra
  application/    # commands, queries, ports, OHS impl
  adapters/       # HTTP, events, jobs, repos, providers
  contracts/      # job/event JSON payloads only
```

| Layer          | Role                                                        |
| -------------- | ----------------------------------------------------------- |
| `wire.go`      | Composition root. Host imports this package only.           |
| `api/`         | Published language for other BCs. No `application` imports. |
| `contracts/`   | Message payloads on the wire. Not OHS interfaces.           |
| `adapters/`    | Drivers. Other BCs must not import this layer.              |
| `application/` | Use cases. Other BCs must not import this layer.            |
| `domain/`      | Pure model. Other BCs must not import this layer.           |

Import rules:

- Host (`cmd/*`, `internal/platform/messaging`) imports the module
  root only (`NewApplication`, `NewHTTPHandler`, `RegisterEvents`,
  `RegisterJobs`, OHS constructors).
- Another BC's ACL imports `{bc}/api` (and
  `internal/contracts/events` for platform events).
- Never import another BC's `application/`, `adapters/`, or
  `domain/`.

`wire.go` constructors (add only what the module needs):

| Function         | When                                                                         |
| ---------------- | ---------------------------------------------------------------------------- |
| `NewApplication` | Always                                                                       |
| `NewHTTPHandler` | HTTP surface                                                                 |
| `RegisterEvents` | Integration event subscribers                                                |
| `RegisterJobs`   | Job handlers                                                                 |
| OHS constructors | Cross-BC sync calls (`NewInvoiceSupport`, `NewInternalService`, and similar) |

Do not add empty `RegisterEvents` / `RegisterJobs` on modules that
have no consumers.

Dependency direction inside a module: `adapters → application →
domain`. `api/` is implemented by `application/` and consumed by
other modules' adapters.

Wire features in `cmd/api/main.go`, `cmd/worker/main.go`, and Lambda
entrypoints — not inside domain or application.

## Entrypoints

| Binary                    | Role                                      |
| ------------------------- | ----------------------------------------- |
| `cmd/api`                 | HTTP API                                  |
| `cmd/worker relay`        | Drains outbox → broker                    |
| `cmd/worker events`       | Integration event handlers                |
| `cmd/worker jobs`         | Background job handlers                   |
| `cmd/worker scheduler`    | Periodic job enqueue (e.g. inbox sweeper) |
| `cmd/lambda/outbox-relay` | AWS relay (scheduled)                     |
| `cmd/lambda/eventbridge`  | AWS events consumer                       |
| `cmd/lambda/sqs`          | AWS jobs consumer                         |
| `cmd/migrate`             | DB migrations CLI                         |

Local dev runs API + workers via Air; see [Getting started](../getting-started.md).

## Config and secrets

| Profile                          | Secrets source                                                                                       |
| -------------------------------- | ---------------------------------------------------------------------------------------------------- |
| `onprem` (local + client deploy) | Plain `.env` (`DATABASE_URL`, `RABBITMQ_URL`, `MINIO_ENDPOINT_URL`, encryption keys, API keys)       |
| `aws`                            | SSM SecureString JSON at `SSM_PARAMETER_NAME` — see [SSM secrets JSON](../deployment/ssm-secrets.md) |

## Local development runtime

- **Messaging:** RabbitMQ + transactional outbox + relay + worker consumers. See [Outbox relay](./outbox-relay.md).
- **Object storage:** MinIO (S3-compatible). See [Object storage](./object-storage.md) and [MinIO](../tooling/minio.md).
- **No AWS emulation** for SQS/EventBridge/SSM in daily dev — set `DEPLOYMENT_TARGET=onprem`.

## Related

- [Events vs jobs](./events-vs-jobs.md)
- [Messaging troubleshooting](./messaging-troubleshooting.md)
- [Database migrations](./database-migrations.md)
