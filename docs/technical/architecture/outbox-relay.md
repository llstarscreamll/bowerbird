# Outbox relay pipeline

Homogeneous flow in every deployment profile:

```text
HTTP use case ──► outbox (DB tx) ──► relay ──► broker ──► consumer ──► handler
```

Application code **never** publishes directly to RabbitMQ, EventBridge,
or SQS. The on-prem **scheduler clock** is not application code: named
rules call `BrokerTransport.DeliverJob` (same hop as an EventBridge
rule target). Use cases still go through the outbox.

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

| Profile        | Events                      | Jobs                      |
| -------------- | --------------------------- | ------------------------- |
| onprem / local | `events-consumer` (AMQP)    | `jobs-consumer` (AMQP)    |
| aws            | Lambda `${ENV}-atta-events` | Lambda `${ENV}-atta-jobs` |

Handlers are shared via `internal/platform/messaging.WireMessagingHandlers`.

## Brokers

| Profile | Events                       | Jobs                                           |
| ------- | ---------------------------- | ---------------------------------------------- |
| onprem  | RabbitMQ topic `atta.events` | RabbitMQ direct `atta.jobs` → `atta.jobs.work` |
| aws     | EventBridge                  | SQS                                            |

Job queue bindings are declared at worker boot from registered `JobHandler.JobType()` values (composition root), not hard-coded in platform.

Integration events use **CloudEvents 1.0 JSON** (`data` = business payload; extension attributes `tenant_slug`, `correlation_id`).

Jobs use an internal JSON envelope + headers (`tenant_slug`,
`job_type`, `correlation_id`, `message_id`). Tenant jobs require
`tenant_slug` and HMAC over that slug. Platform jobs (scheduler
ticks) omit `tenant_slug`; HMAC binds the reserved subject
`_platform`. Events stay tenant-scoped.

Dead letters: RabbitMQ `atta.dlx` / `atta.deadletter`; AWS SQS DLQ.

## Runners (local/onprem)

```bash
pnpm --filter @atta/backend dev          # api
pnpm --filter @atta/backend dev:relay
pnpm --filter @atta/backend dev:events-consumer
pnpm --filter @atta/backend dev:jobs-consumer
pnpm --filter @atta/backend dev:scheduler
```

Or `mise //apps/atta:dev` (Turbo runs api + workers + PWA).

## Scheduler (on-prem)

`cmd/onprem/scheduler` is a named-rule clock, not an outbox writer:

```text
rule (name + rate()/crontab) ──► one platform job ──► jobs-consumer
                                      │
                                      └── handler lists tenants / enqueues children
```

The clock has no tenant list. Each rule publishes one job with empty
`tenant_slug`. Handlers that need tenants list them from the control
plane.

| Name                   | Schedule (on-prem) | Job                           | Handler                           |
| ---------------------- | ------------------ | ----------------------------- | --------------------------------- |
| `outbox-sweeper`       | `rate(1 hour)`     | `platform.OutboxSweeper`      | List tenants; purge each DB       |
| `inbox-sync-all`       | `rate(5 minutes)`  | `InboxSyncAllAccounts`        | List tenants; enqueue per account |
| `catalog-import-purge` | `0 5 * * *`        | `CatalogImportPurgeRequested` | List tenants; purge stale imports |

`rate()` matches EventBridge Scheduler. Crontab is Unix 5-field UTC.
AWS uses 6-field EventBridge cron with `?` (`catalog-import-purge` →
`cron(0 5 * * ? *)`). See [AWS deploy](../deployment/aws.md#schedules).

`inbox-sync-all` is omitted unless Google or Microsoft OAuth is set.

Child work (`InboxSyncAccount`) still goes through `TaskQueue` /
outbox and stays tenant-scoped. Entitlement checks run inside the
tenant loop, not on the platform parent.

## Related

- [Events vs jobs](./events-vs-jobs.md)
- [On-prem runtime](./onprem-runtime.md)
- [Runtime profiles](./runtime-profiles.md)
