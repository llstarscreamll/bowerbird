# Messaging troubleshooting

## Correlation ID

Every HTTP request should propagate `correlation_id` into outbox rows and broker messages. Search logs across `api`, `outbox-relay`, `events-consumer`, and `jobs-consumer` using the same ID.

## Dead letters

| Profile | Events/jobs DLQ                                     |
| ------- | --------------------------------------------------- |
| onprem  | RabbitMQ `bowerbird.deadletter` via `bowerbird.dlx` |
| aws     | SQS DLQ (`maxReceiveCount` ≈ 5)                     |

Consumer NACK with requeue on transient errors; poison messages land in DLQ after repeated failures.

## Outbox relay poison pills

Rows exceeding `max_attempts` broker publish failures transition to `failed` in `outbox_events` / `outbox_jobs` so the relay does not block head-of-line.

Inspect: `SELECT id, status, attempts, last_error FROM outbox_events WHERE status = 'failed'`.

## RabbitMQ reconnect

Workers use exponential backoff on connection loss. Expect log lines `consumer disconnected` followed by successful reconnect when the broker returns.

## Common issues

| Symptom                                                     | Check                                                                                   |
| ----------------------------------------------------------- | --------------------------------------------------------------------------------------- |
| Events never handled                                        | `outbox-relay` running? RabbitMQ bindings?                                              |
| Jobs stuck pending                                          | `jobs-consumer` running? `RABBITMQ_URL` matches relay                                   |
| Local dev no messaging                                      | `DEPLOYMENT_TARGET=onprem` + RabbitMQ (not AWS SQS)                                     |
| MinIO `PutObject` `SignatureDoesNotMatch` during inbox sync | Worker env checksum vars; non-ASCII S3 metadata — [Object storage](./object-storage.md) |

See [Outbox relay](./outbox-relay.md) and [Runtime profiles](./runtime-profiles.md).
