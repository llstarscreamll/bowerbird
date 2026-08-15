# Backend architecture

## Stack

- Go `net/http` (no web framework)
- PostgreSQL via `pgx` and raw SQL
- Lambdas for HTTP, SQS, and EventBridge

## Layout

`internal/` splits platform utilities from business modules.

### `internal/platform/`

Cross-cutting adapters only (no business logic): AWS SDK, config/SSM, DB pools, EventBridge events, SQS jobs.

### Feature modules (bounded contexts)

Follow `internal/invoices`-style layout:

| Layer          | Role                                      |
| -------------- | ----------------------------------------- |
| `domain/`      | Pure model and outbound interfaces        |
| `application/` | Use cases (`commands`, `queries`) + ports |
| `contracts/`   | Cross-boundary DTOs (jobs/events)         |
| `adapters/`    | HTTP, jobs/events, repos, providers       |
| `wire.go`      | Composition root for the feature          |

Dependency direction: `adapters → application → domain`. Features must not import other features’ adapters; couple via IDs or events.

Wire features in `cmd/api/main.go` (and Lambda entrypoints), not inside domain/application.

## Config and secrets

At boot, read `SSM_PARAMETER_NAME`, fetch the SecureString JSON, map into `Config`. LocalStack seeds SSM from `secrets.json`. Production Lambdas get `ssm:GetParameter`.

## Local AWS loop

Emulate S3, SQS, EventBridge, SSM in LocalStack. Do not deploy Lambdas locally. With `ENABLE_LOCAL_EVENT_LOOP`, `cmd/api` runs separate EventBridge and SQS pollers that reuse the same handlers as production Lambdas.
