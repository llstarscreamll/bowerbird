# Events vs jobs

- **Integration events:** domain facts already occurred (`InboxMessageReceived`). Fan-out; producer does not know consumers.
- **Background jobs:** deferred work units (`InvoiceExtractionRequested`, `InboxSyncAccount`). Retries/backoff; worker semantics.

Rule: event handlers stay light — enqueue a job for heavy work.

See [Outbox relay](./outbox-relay.md) for the full pipeline and [Runtime profiles](./runtime-profiles.md) for broker selection by deployment target.

## Code map

| Concern        | Port / adapter                                        |
| -------------- | ----------------------------------------------------- |
| Publish events | `EventBus` → `OutboxEventPublisher` → `outbox_events` |
| Enqueue jobs   | `TaskQueue` → `OutboxTaskQueue` → `outbox_jobs`       |
| Relay          | `internal/platform/outbox/relay` → broker adapter     |
| Handle events  | `RegisterEvents` on the owning module `wire.go`       |
| Handle jobs    | `RegisterJobs` on the owning module `wire.go`         |

`internal/platform/messaging.WireMessagingHandlers` calls those
registrars. It must not import feature `adapters/`. Split event and
job registration; do not bundle them in one `RegisterMessaging`.

## Decision guide

| Question                        | Event          | Job        |
| ------------------------------- | -------------- | ---------- |
| Something already happened?     | Yes            | No         |
| Pending work unit?              | No             | Yes        |
| Multiple independent consumers? | Yes            | Usually no |
| Needs worker/retry semantics?   | Not by default | Yes        |

## Examples

| Flow                | Pattern                                            |
| ------------------- | -------------------------------------------------- |
| Inbox message saved | Event `InboxMessageReceived`                       |
| Invoice extraction  | Job `InvoiceExtractionRequested`                   |
| Connection added    | Event → handler enqueues sync **job**              |
| Periodic inbox sync | Scheduler tick → **job** (not a fake domain event) |

## Contracts

- Events on the wire: **CloudEvents 1.0 JSON** after relay (`type`, `data`, `tenant_slug`, `correlation_id`).
- Jobs on the wire: internal envelope + `tenant_slug`, `job_type`, `correlation_id`, `message_id`.

Label every new message contract as `event` or `job` in PRs.
