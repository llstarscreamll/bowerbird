## 1. Inventario y baseline

- [x] 1.1 Inventariar símbolos AWS-leak con `rg` en `apps/backend`; guardar lista en PR/notas
- [x] 1.2 Baseline `pnpm --filter @bowerbird/backend test` y `lint` en HEAD limpio; exit 0
- [x] 1.3 Documentar en design D5/D6/D7: pipeline homogéneo relay→broker→consumers; RabbitMQ onprem; EventBridge+SQS aws

## 2. Ports vendor-agnostic (platform) — rename big-bang

- [x] 2.1 Renombrar `jobs.Queue` → `jobs.TaskQueue`; verificar `go build ./...`
- [x] 2.2 Renombrar `jobs.SQSProcessor` → `jobs.JobHandler`; verificar tests `platform/jobs`
- [x] 2.3 Mover SQS adapter a `jobs/adapters/sqs/` para **consumers aws** y relay broker aws (no application enqueue)
- [x] 2.4 Renombrar `EventBridgeSubscriber` → `IntegrationEventHandler`; verificar tests `platform/events`
- [x] 2.5 `OutboxEventPublisher` en application; eliminar publish directo; verificar application no importa brokers
- [x] 2.6 Actualizar `platform.Dependencies`; verificar `wire.go`
- [x] 2.7 Port stub `Scheduler` en `platform/scheduler/`

## 3. Actualizar features al nuevo port surface

- [x] 3.1–3.3 Actualizar invoices, inbox, connections; tests verdes
- [x] 3.4 Actualizar `cmd/api`, `cmd/worker` (subcomandos relay|events|jobs), `cmd/lambda/outbox-relay`, mantener `cmd/lambda/eventbridge` y `cmd/lambda/sqs`; `go build` cada cmd
- [x] 3.5 Grep: publish al broker solo en `platform/outbox/relay/broker/*`

## 4. Regla de uso: event handler → job

- [x] 4.1–4.2 ConnectionAdded → outbox_jobs
- [x] 4.3 docs `events-vs-jobs.md`

## 5. Contratos CloudEvents-aligned

- [x] 5.1 Refactorizar la serialización de integration events en el Relay para cumplir estrictamente con CloudEvents 1.0 JSON (mapeando payload a `data`, e inyectando `tenant_slug` y `correlation_id` como extension attributes).
- [x] 5.2 Revisar contracts de background jobs para asegurar propagación consistente de headers (no aplican para CloudEvents estricto por su naturaleza de comando).
- [x] 5.3 Validar parseo con SDK/estructuras compatibles con CloudEvents en los handlers (events-consumer / events-processor).

## 6. Config y composition root multi-profile

- [x] 6.1 `DeploymentTarget` (`aws`|`onprem`); default local = `onprem`; tests
- [x] 6.2 Secretos `.env` onprem incl. `RABBITMQ_URL`; sin SSM messaging diario
- [x] 6.3 SSM path aws; smoke
- [x] 6.4 `platform.NewModule` factory por profile; compila ambas ramas
- [x] 6.5 Local dev: `.env` plano; ajustar `infra:up` si deja de depender de LocalStack SSM para messaging

## 7. Outbox store + relay + broker adapters

- [x] 7.1 Migraciones `outbox_events`, `outbox_jobs`; `migrate:all`
- [x] 7.2 `platform/outbox/store` custom (D14); tests rollback, claim concurrente, reintento
- [x] 7.3 `OutboxEventPublisher` + `OutboxTaskQueue`; sin publish directo broker
- [x] 7.4 `BrokerTransport` interface + `broker/aws` (EventBridge + SQS); tests mocks SDK
- [x] 7.5 `broker/rabbitmq`: topology (`bowerbird.events` topic, `bowerbird.jobs` direct, `bowerbird.dlx`), publish confirm, headers tenant; tests con RabbitMQ testcontainer o compose
- [x] 7.6 Relay core fair multi-tenant (aws); test 2 tenants
- [x] 7.7 `SyncAccountCommand` UoW atómico; test
- [x] 7.8 MinIO on-prem; smoke presign
- [x] 7.9 Implementar lógica `max_attempts` y estado `failed` (Poison Pill handling) para filas del outbox en el Store y Relay

## 8. Consumers y runners

- [x] 8.1 `cmd/worker relay`: `relay.RunLoop` + broker rabbitmq (onprem/local); `go build`
- [x] 8.2 `cmd/worker events`: AMQP consumer → `IntegrationEventHandler`; `go build`
- [x] 8.3 `cmd/worker jobs`: AMQP consumer → `JobHandler`; DLQ/nack policy; `go build`
- [x] 8.4 `cmd/lambda/outbox-relay`: `RunOnce` + broker aws; `go build`
- [x] 8.5 `WireMessagingHandlers` compartido: consumers onprem + Lambdas aws
- [x] 8.6 Eliminar pollers embebidos API y `ENABLE_LOCAL_EVENT_LOOP`
- [x] 8.7 Turbo dev: `dev:relay`, `dev:events-consumer`, `dev:jobs-consumer` junto a api; outbox fluye end-to-end
- [x] 8.8 RabbitMQ en `docker-compose.yml` raíz; retirar `sqs,events` de LocalStack `SERVICES`; healthcheck AMQP
- [x] 8.9 Wrapper/Loop de conexión AMQP con _backoff exponencial_ para auto-reconexión ante fallos (Resiliencia)

## 9. Stack Compose on-prem

- [x] 9.1 `deploy/onprem/docker-compose.yml`: `caddy`, `api`, `outbox-relay`, `events-consumer`, `jobs-consumer`, `rabbitmq`, `postgres`, `minio`; `docker compose config`
- [x] 9.2 `.env.example`: `RABBITMQ_URL`, credenciales, `DEPLOYMENT_TARGET=onprem`
- [x] 9.3–9.5 Caddyfile, migrate scripts, healthchecks; `compose up` healthy

## 10. Infra AWS

- [x] 10.1 CDK Scheduler → Lambda `outbox-relay`; synth
- [x] 10.2 Mantener EventBridge rule + `events-processor`; fix `source`
- [x] 10.3 Mantener SQS + `sqs-processor` + DLQ
- [x] 10.4 Doc evolución Lambda relay → Fargate
- [x] 10.5 Smoke aws: outbox → relay → EB/SQS → Lambdas → processed

## 11. Scheduler (mínimo)

- [x] 11.1–11.2 Stub/port; ticks → outbox_jobs
- [x] 11.3 Tarea `OutboxSweeper`: barrido de registros `processed` y `failed` antiguos en `outbox_events` y `outbox_jobs` (Escalabilidad D17)

## 12. Docs y DX

- [x] 12.1 Crear `docs/technical/architecture/outbox-relay.md`: (a) outbox — tablas, tx atómica, qué NO hace application; (b) **relay** — claim `SKIP LOCKED`, publica al broker, marca processed (no ejecuta handlers); (c) **consumers** — ejecutan handlers; (d) pipeline homogéneo; (e) RabbitMQ topology onprem vs EventBridge/SQS aws; diagrama; enlaces en índice docs
- [x] 12.2 `onprem-runtime.md`: Compose, RabbitMQ, relay+consumers; local dev = onprem
- [x] 12.3 `events-vs-jobs.md` + enlace outbox-relay.md
- [x] 12.4 `getting-started.md`: RabbitMQ en dev, subcomandos worker, sin LocalStack messaging
- [x] 12.5 Actualizar documentación de topología, dead-letters y troubleshooting (incluyendo trazabilidad vía `correlation_id`)

## 13. Verificación integral

- [x] 13.1 lint + test verdes
- [x] 13.2 Smoke local/onprem: outbox → relay → RabbitMQ → consumers → processed
- [x] 13.3 Smoke aws: outbox → relay → EB/SQS → Lambdas
- [x] 13.4 ConnectionAdded encola job; test
- [x] 13.5 Grep application layer sin publish directo broker
- [x] 13.6 Test de tolerancia a fallos: Caída de RabbitMQ y verificación de reconexión de runners.
- [x] 13.7 Verificación de Poison Pill: Comprobar transición a estado `failed` en DB tras N fallos de relay hacia broker.
