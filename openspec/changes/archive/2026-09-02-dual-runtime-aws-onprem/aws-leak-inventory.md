# AWS-leak inventory (apps/backend)

Symbols and locations to vendor-neutralize during dual-runtime change.

| Symbol / pattern                              | Files                                                                                                   |
| --------------------------------------------- | ------------------------------------------------------------------------------------------------------- |
| `jobs.Queue`                                  | `platform/jobs/queue.go`, `platform/wire.go`, `inbox/wire.go`, `invoices/wire.go`, application commands |
| `SQSQueue` / direct SQS enqueue               | `platform/jobs/sqs_queue.go`, `platform/wire.go`                                                        |
| `SQSProcessor` / `HandleSQS`                  | `platform/jobs/handler.go`, invoice/inbox job adapters                                                  |
| `EventBridgePublisher` / direct EB publish    | `platform/events/eventbridge_publisher.go`, `platform/wire.go`                                          |
| `EventBridgeSubscriber` / `HandleEventBridge` | `platform/events/handler.go`, invoice/inbox event adapters                                              |
| `EnableLocalEventLoop` + embedded pollers     | `platform/config/config.go`, `cmd/api/main.go`                                                          |
| SQS pollers (local dev)                       | `platform/jobs/poller.go`, `platform/events/poller.go`                                                  |
| Lambda SQS/EventBridge entrypoints            | `cmd/lambda/sqs`, `cmd/lambda/eventbridge`                                                              |
| AWS config in composition root                | `platform/wire.go`, `platform/awsconfig/`                                                               |
| SSM secrets merge (messaging)                 | `platform/config/config.go`, `scripts/init-localstack.sh`                                               |
| `SQSSyncAccountJobDispatcher` naming          | `inbox/application/commands/sqs_sync_dispatcher.go`                                                     |

**Target:** publish/enqueue only via outbox; broker access only in `platform/outbox/relay/broker/*`; consumers in worker/Lambda cmd.
