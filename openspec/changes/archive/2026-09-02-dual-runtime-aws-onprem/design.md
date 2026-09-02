## Context

Hoy `platform.NewModule` instancia siempre EventBridge, SQS y S3; `config.Load` asume SSM; producción AWS usa Lambdas + EventBridge + SQS; local usa LocalStack + pollers embebidos en `cmd/api`. Interfaces `EventBus` y `Queue` existen pero los tipos filtran nombres AWS. El cliente on-prem es **un solo servidor CentOS + Docker**, full local, secretos en **`.env` planos**.

**Término elegido:** **deployment profile** (`aws` | `onprem`) — denota el mismo código desplegado con distinta infraestructura de ejecución. No usar “ambiente” (confunde con `APP_ENV`) ni “escenario” (demasiado vago).

Ver `proposal.md` y specs `platform-messaging`, `onprem-runtime`.

## Goals / Non-Goals

**Goals:**

- **Homogeneidad de pipeline:** en todos los profiles: `outbox (tx)` → **relay** (solo publica) → **broker** → **consumers** (handlers). Sin handlers in-process en el relay.
- **Application layer homogéneo:** `EventBus.Publish` / `TaskQueue.Enqueue` → `outbox_events` / `outbox_jobs` en misma tx; nunca publish directo al broker desde use cases.
- **Brokers por profile:** aws → EventBridge + SQS; onprem + **local dev** → **RabbitMQ** (events + jobs).
- Un codebase; adapters en `platform/outbox/relay/broker/{aws,rabbitmq}`.
- Compose on-prem: `api`, `outbox-relay`, `events-consumer`, `jobs-consumer`, `rabbitmq`, `postgres`, `minio`, `caddy` + `.env`.
- Rename big-bang vendor-agnostic + regla handler ligero → job.
- AWS v1: Lambda **outbox-relay** (Scheduler) + Lambdas consumidoras existentes.

**Non-Goals (design-level):**

- Handlers in-process en relay (ningún profile).
- NATS, Kafka, Redis Streams como broker v1.
- LocalStack EventBridge/SQS en flujo diario de desarrollo.
- Outbox cross-DB atómico — outbox vive en la DB del write.
- Relay dedicado por tenant.
- GCP adapter en v1 (misma interfaz `BrokerTransport`; deferible).

## Decisions

### D1 — Deployment profile por config

- **Decisión:** `DEPLOYMENT_TARGET=aws|onprem` selecciona broker adapter, secrets, file store, relay runner y consumer packaging.
- **Alternativa rechazada:** detectar por `AWS_ENDPOINT_URL` — frágil.
- **Alternativa rechazada:** forks por cliente.

### D2 — On-prem packaging = Docker Compose (`deploy/onprem/`)

- Imagen Go única; entrypoints: `api`, `outbox-relay`, `events-consumer`, `jobs-consumer`.
- **RabbitMQ** como servicio Compose obligatorio.

### D3 — Secrets: `.env` (onprem) / SSM (aws)

- RabbitMQ user/password en `.env` on-prem (sin vault remoto v1).

### D4 — Object storage: S3 (aws) / MinIO (onprem)

- Mismo adapter SDK S3; endpoint distinto por profile.

### D5 — Transactional outbox para events **y** jobs (ambos profiles)

**Tablas:**

| Tabla           | Contenido          | Tx con                      |
| --------------- | ------------------ | --------------------------- |
| `outbox_events` | Integration events | Writes de negocio en esa DB |
| `outbox_jobs`   | Background jobs    | Writes que disparan el job  |

**Application ports:** siempre insert outbox en tx; **nunca** broker directo.

**Relay (homogéneo — solo publica al broker):**

```
platform/outbox/relay
  ClaimPendingEvents / ClaimPendingJobs   // SKIP LOCKED; fair multi-tenant (aws)
  BrokerTransport.DeliverEvent / DeliverJob
  MarkProcessed / MarkFailed
```

### D6 — Pipeline homogéneo relay → broker → consumers

| Profile       | Relay runner                      | Broker (events)                       | Broker (jobs)                     | Consumers                                   |
| ------------- | --------------------------------- | ------------------------------------- | --------------------------------- | ------------------------------------------- |
| **onprem**    | `outbox-relay` (loop)             | RabbitMQ **topic** `bowerbird.events` | RabbitMQ **direct** + work queues | `events-consumer`, `jobs-consumer`          |
| **aws**       | Lambda `outbox-relay` (Scheduler) | EventBridge                           | SQS                               | Lambdas `events-processor`, `sqs-processor` |
| **local dev** | = onprem                          | RabbitMQ en compose raíz              | idem                              | idem                                        |

- Handlers **mismos**; entrypoints distintos (contenedores Go vs Lambda).
- **Alternativa rechazada:** in-process on-prem — rompe paridad con aws/GCP y concentra blast radius.

### D7 — RabbitMQ on-prem (topología v1)

| Kind   | Exchange           | Tipo     | Routing key                                             | Consumer queue                                       |
| ------ | ------------------ | -------- | ------------------------------------------------------- | ---------------------------------------------------- |
| Events | `bowerbird.events` | `topic`  | `{type}` (CloudEvent type) p.ej. `InboxMessageReceived` | `bowerbird.events.handlers` (binding `#` o por type) |
| Jobs   | `bowerbird.jobs`   | `direct` | `{JobType}` p.ej. `InvoiceExtractionRequested`          | `bowerbird.jobs.work`                                |

- Payload Events: Estándar **CloudEvents 1.0 JSON** (attributes: id, source, specversion, type, time, data, tenant_slug, correlation_id).
- Payload Jobs: JSON envelope interno (ya que son RPC diferido, no eventos de dominio) + headers `tenant_slug`, `message_id`, `correlation_id`.
- **DLQ:** Dead-Letter Exchange (`bowerbird.dlx`) y cola dead-letter (`bowerbird.deadletter`) unificadas obligatorias en on-prem; política de reintento con backoff en código del consumer, finalizando en el DLX tras N intentos fallidos (~SQS `maxReceiveCount` ≈ 5).
- Declaración de topology en bootstrap script o al arrancar relay/consumers (idempotente).
- **Alternativa rechazada:** NATS JetStream — viable pero RabbitMQ más maduro para DLQ/work queues on-prem.

### D8 — AWS: outbox-relay + broker existente

```
API → outbox (RDS) → Lambda outbox-relay
                          ├─ PutEvents → EventBridge → Lambda events-processor
                          └─ SendMessage → SQS → Lambda sqs-processor
```

- Mantener EventBridge rule (fix `source`), SQS + DLQ, Lambdas consumidoras.
- Eliminar publish directo desde API.

### D9 — Multi-tenant relay (aws)

Fair round-robin + caps por tenant antes de publicar al broker. On-prem: un tenant; mismo código relay.

### D10 — Rename big-bang

| Actual                                | Objetivo                         |
| ------------------------------------- | -------------------------------- |
| `EventBridgeSubscriber`               | `IntegrationEventHandler`        |
| `EventBridgePublisher` en application | `OutboxEventPublisher`           |
| `jobs.Queue`                          | `TaskQueue` / `OutboxTaskQueue`  |
| `SQSProcessor`                        | `JobHandler`                     |
| Relay publish aws                     | `broker/aws` (EventBridge + SQS) |
| Relay publish onprem                  | `broker/rabbitmq`                |

### D11 — Regla ConnectionAdded → job

Sin cambio: handler encola vía `outbox_jobs`.

### D12 — Scheduler

Ticks insertan `outbox_jobs` en tx; no integration events ficticios.

### D13 — On-prem single tenant

Control-plane + una tenant DB seedeadas.

### D14 — Outbox store custom + BrokerTransport

- Store: custom `pgx` (~300–500 LOC), interfaz `Store`.
- Relay: custom; **solo** `BrokerTransport` con adapters `aws` y `rabbitmq`.
- Consumers: AMQP long-running (onprem) / Lambda (aws); comparten `WireMessagingHandlers`.

```go
type BrokerTransport interface {
    DeliverEvent(ctx context.Context, row EventRow) error
    DeliverJob(ctx context.Context, row JobRow) error
}
```

- Tests: rollback tx; claim concurrente; relay→mock broker; consumer→handler; fair 2 tenants (aws relay).

### D15 — Desarrollo local = deployment profile `onprem`

- `DEPLOYMENT_TARGET=onprem` en dev; `APP_ENV=local` independiente.
- **RabbitMQ** en `docker-compose.yml` raíz; servicios relay + consumers vía Turbo o compose profile.
- Secretos: `apps/backend/.env`; sin LocalStack messaging/SSM para flujo diario.
- **Overhead:** RabbitMQ ~100–200 MB RAM — aceptable; **menor** que LocalStack events+sqs + pollers API.
- Paridad local ↔ cliente on-prem ↔ patrón aws (todos relay→broker→consumers).

### D16 — Evolución GCP (futuro)

- Nuevo adapter `broker/pubsub` detrás de `BrokerTransport`; relay y consumers Cloud Run; sin cambiar outbox ni handlers.

### D17 — Outbox Sweeper (Escalabilidad y Limpieza)

- **Decisión:** Para evitar el crecimiento ilimitado, se implementará un background job programado (vía Scheduler) que haga sweep/purge de registros en `outbox_events` y `outbox_jobs` marcados como `processed` o `failed` con retención de X días.
- **Alternativa rechazada:** Mantener registros indefinidamente y confiar en particionamiento de BDD (complejo para single-node en on-prem).

### D18 — Observabilidad y Trazabilidad Básica

- Propagar `correlation_id` en todos los mensajes.
- Emitir logs estructurados en el relay detallando `queue_depth` (mensajes pending) y umbrales de fallos de publicación para monitoreo (dead-letter del relay).

### D19 — Estandarización CloudEvents 1.0

- **Decisión:** Todos los integration events publicados por el Relay hacia el broker (RabbitMQ o EventBridge) MUST cumplir estrictamente con la especificación JSON de **CloudEvents 1.0**.
- **Implementación:** El evento de negocio se aloja en el campo `data`. Variables cross-cutting como `tenant_slug` y `correlation_id` se envían como Extension Attributes. Esto facilita el ruteo nativo (por ejemplo, reglas de EventBridge basadas en el `detail-type` mapeado desde el `type` del CloudEvent) y permite consumir los eventos con SDKs agnósticos en el futuro.

## Architecture

```
         ALL PROFILES — APPLICATION (homogéneo)
                              │
                    Use case + UoW → outbox → COMMIT
                              │
                              ▼
                         OUTBOX RELAY
                    (claim, fair, mark processed)
                              │
         ┌────────────────────┴────────────────────┐
         ▼                                          ▼
    ONPREM / LOCAL                              AWS
    RabbitMQ                                    EventBridge + SQS
         │                                          │
    ┌────┴────┐                               ┌────┴────┐
    ▼         ▼                               ▼         ▼
 events-   jobs-                         events-λ   sqs-λ
 consumer  consumer
    └────┬────┘                               └────┬────┘
         └──────── Handlers (mismo código) ────────┘
```

**Homogéneo:** outbox write, relay semantics, handlers, contratos, patrón event→job.  
**Diverge:** broker adapter (RabbitMQ vs EventBridge/SQS) y packaging de consumers.

## Future: optimización relay AWS

Lambda outbox-relay → Fargate relay runner; broker y Lambdas consumidoras intactos.

## Risks / Trade-offs

- **[Risk] Ops RabbitMQ on-prem** → Mitigation: Compose healthcheck; doc backup; DLQ monitoring.
- **[Risk] Más contenedores on-prem** (relay + 2 consumers + rabbitmq) → Mitigation: single VM sizing doc; RAM ~512MB extra.
- **[Risk] Doble hop latencia** → Mitigation: aceptado; aislamiento compensa.
- **[Risk] Topology drift RabbitMQ** → Mitigation: declaración idempotente en código; tests integración compose.
- **[Risk] Outbox sin tx legacy** → Mitigation: refactor inbox sync.
- **[Risk] `.env` on-prem** → Mitigation: threat model documentado.

## Migration Plan

1. Migraciones outbox tables.
2. Outbox publishers + UoW.
3. Store + relay + `broker/rabbitmq` + `broker/aws`.
4. `cmd/outbox-relay`, `cmd/events-consumer`, `cmd/jobs-consumer` (onprem/local); Lambda relay (aws).
5. Compose on-prem + RabbitMQ en compose raíz dev.
6. CDK Scheduler + outbox-relay; mantener EventBridge/SQS Lambdas.
7. Smoke: outbox → relay → RabbitMQ/EventBridge → consumer → processed.

## Open Questions

- Intervalo Lambda relay (10s vs 30s) — env configurable.
- Retención filas outbox `processed` — limpieza deferible.
- ¿Un solo consumer binario con subcomando vs tres binarios? — preferir subcomandos (`worker relay|events|jobs`) para una imagen Docker.
