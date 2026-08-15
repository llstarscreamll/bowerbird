# Events vs jobs

- **Events (EventBridge):** domain facts between decoupled contexts (`InboxMessageReceived`). Fan-out; producer does not know consumers.
- **Jobs (SQS):** work to execute (`InvoiceExtractionRequested`). Retries, backoff, worker semantics.

If both are needed: publish an event; a consumer may enqueue a job.

## Code map

| Concern                    | Location                   |
| -------------------------- | -------------------------- |
| Publish / subscribe events | `internal/platform/events` |
| Enqueue / process jobs     | `internal/platform/jobs`   |

`cmd/api` wires separate `eventHandler` and `jobHandler`, plus two local pollers when `ENABLE_LOCAL_EVENT_LOOP` is on.

## Decision guide

| Question                        | Event          | Job               |
| ------------------------------- | -------------- | ----------------- |
| Something already happened?     | Yes            | No                |
| Pending work unit?              | No             | Yes               |
| Multiple independent consumers? | Yes (fan-out)  | Usually no        |
| Needs worker/retry semantics?   | Not by default | Yes               |
| Producer fully decoupled?       | Yes            | May know job type |

## Avoid

- Treating commands as EventBridge “events”
- Mixing event and job pollers/handlers in one abstraction
- Using jobs to broadcast domain facts across contexts

## Example

`inbox` → `InboxMessageReceived` → `invoices` evaluates → enqueue `InvoiceExtractionRequested` → processor extracts and persists.

In PRs, label each new message contract as `event` or `job`.
