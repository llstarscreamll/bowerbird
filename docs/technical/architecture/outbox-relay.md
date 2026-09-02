# Outbox relay pipeline

Homogeneous flow in every deployment profile:

```text
HTTP use case ──► outbox (DB tx) ──► relay ──► broker ──► consumer ──► handler
```

Application code **never** publishes directly to RabbitMQ, EventBridge, or SQS.

## Outbox tables (tenant DB)

| Table           | Purpose            |
| --------------- | ------------------ |
| `outbox_events` | Integration events |
| `outbox_jobs`   | Background jobs    |

Columns include `status` (`pending` | `processed` | `failed`), `attempts`, `max_attempts`, `correlation_id`.

Use cases write via `EventBus.Publish` / `TaskQueue.Enqueue` (`OutboxEventPublisher`, `OutboxTaskQueue`). When a unit of work is available, inserts share the same PostgreSQL transaction as the business write.

## Relay

Location: `internal/platform/outbox/relay`.

1. `ClaimPendingEvents` / `ClaimPendingJobs` with `FOR UPDATE SKIP LOCKED`
2. `BrokerTransport.DeliverEvent` / `DeliverJob`
3. `MarkProcessed` or `MarkFailed` (poison pill after `max_attempts`)

Relay **does not** execute business handlers.

Relay iterates **all active tenants** from the control-plane on every profile (on-prem included).

## Consumers

| Profile        | Events                    | Jobs                   |
| -------------- | ------------------------- | ---------------------- |
| onprem / local | `worker events` (AMQP)    | `worker jobs` (AMQP)   |
| aws            | Lambda `events-processor` | Lambda `sqs-processor` |

Handlers are shared via `internal/platform/messaging.WireMessagingHandlers`.

## Brokers

| Profile | Events                            | Jobs                                                     |
| ------- | --------------------------------- | -------------------------------------------------------- |
| onprem  | RabbitMQ topic `bowerbird.events` | RabbitMQ direct `bowerbird.jobs` → `bowerbird.jobs.work` |
| aws     | EventBridge                       | SQS                                                      |

Job queue bindings are declared at worker boot from registered `JobHandler.JobType()` values (composition root), not hard-coded in platform.

Integration events use **CloudEvents 1.0 JSON** (`data` = business payload; extension attributes `tenant_slug`, `correlation_id`).

Jobs use an internal JSON envelope + headers (`tenant_slug`, `job_type`, `correlation_id`, `message_id`).

Dead letters: RabbitMQ `bowerbird.dlx` / `bowerbird.deadletter`; AWS SQS DLQ.

## Runners (local/onprem)

```bash
pnpm --filter @bowerbird/backend dev          # api
pnpm --filter @bowerbird/backend dev:relay
pnpm --filter @bowerbird/backend dev:events-consumer
pnpm --filter @bowerbird/backend dev:jobs-consumer
pnpm --filter @bowerbird/backend dev:scheduler
```

Or root `pnpm run dev` (Turbo runs api + workers + PWA).

## Related

- [Events vs jobs](./events-vs-jobs.md)
- [On-prem runtime](./onprem-runtime.md)
- [Runtime profiles](./runtime-profiles.md)
