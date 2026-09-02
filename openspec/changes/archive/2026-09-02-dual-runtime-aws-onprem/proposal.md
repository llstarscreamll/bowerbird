## Why

Un segundo cliente exige desplegar Bowerbird en VM Docker sin AWS. El backend publica directo a EventBridge/SQS sin outbox. Se necesita transactional outbox homogéneo y **el mismo pipeline en todos los deployment profiles**: outbox → relay → broker → consumers → handlers.

## What Changes

- **BREAKING:** rename vendor-agnostic; eliminar publish directo al broker desde application.
- **Homogeneidad pipeline:** relay solo publica; handlers solo en consumers (no in-process).
- **On-prem + local dev:** **RabbitMQ** (topic events + direct jobs) + `outbox-relay`, `events-consumer`, `jobs-consumer`; Compose + `.env`; MinIO.
- **AWS:** Lambda outbox-relay → EventBridge + SQS; mantener Lambdas consumidoras; fix `source`.
- Desarrollo local = profile `onprem` con RabbitMQ (sin LocalStack messaging).

## Non-goals / out of scope

- Handlers in-process en relay (ningún profile).
- NATS/Kafka v1.
- GCP Pub/Sub adapter v1 (interfaz preparada).
- LocalStack messaging en dev diario.
- Secrets remotos / control-plane SaaS HTTP.

## Capabilities

### New Capabilities

- `platform-messaging`: outbox, relay, BrokerTransport (RabbitMQ + aws), consumers, scheduler.
- `onprem-runtime`: Compose, RabbitMQ, MinIO, relay + consumers, `.env`.

### Modified Capabilities

- _(ninguno)_

## Impact

- **Backend:** `platform/outbox`, `broker/rabbitmq`, `broker/aws`; cmd worker subcommands; Lambda relay.
- **Infra AWS:** CDK outbox-relay; mantener EventBridge/SQS/Lambdas.
- **On-prem / local:** RabbitMQ en Compose; tres procesos consumer/relay además de api.
- **Docs:** outbox-relay.md con pipeline homogéneo y topología RabbitMQ.
