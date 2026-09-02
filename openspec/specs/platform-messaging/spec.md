# platform-messaging Specification

## Purpose

Define la semántica vendor-agnostic de integration events, background jobs (task queue) y scheduler, más las reglas de uso que deben cumplir productores y consumidores en cualquier deployment profile.

## Requirements

### Requirement: Distinción integration event vs background job

El sistema SHALL tratar un integration event como un hecho de dominio ya ocurrido (pasado) y un background job como una unidad de trabajo pendiente con semántica de worker (reintentos/backoff). Los contratos de mensaje SHALL etiquetarse de forma inequívoca como event o job.

#### Scenario: Publicación de hecho entre módulos

- **WHEN** un módulo completa una acción de negocio que puede interesar a otros bounded contexts
- **THEN** el sistema publica un integration event (nombre en pasado, p.ej. `InboxMessageReceived`) y el productor no conoce la lista de consumidores

#### Scenario: Trabajo pesado diferido

- **WHEN** un módulo necesita ejecutar trabajo largo o reintentable fuera del request HTTP
- **THEN** el sistema encola un background job tipado (p.ej. `InvoiceExtractionRequested`) en la task queue

### Requirement: Handlers de integration events no ejecutan trabajo pesado

Un handler de integration event SHALL limitarse a validación, checks de feature y orquestación ligera. Si hay trabajo de sincronización, extracción u otra operación larga, el handler MUST encolar un background job y no ejecutar esa operación inline.

#### Scenario: ConnectionAdded encola sync

- **WHEN** se entrega el integration event `ConnectionAdded` a inbox
- **THEN** el handler encola un job de sincronización de cuenta y no ejecuta el sync completo dentro del handler del evento

#### Scenario: InboxMessageReceived encola extracción

- **WHEN** se entrega `InboxMessageReceived` y el mensaje califica como candidato de factura
- **THEN** el handler encola `InvoiceExtractionRequested` (o equivalente tipado) y no extrae documentos inline

### Requirement: Contratos estandarizados con CloudEvents 1.0

Los integration events publicados entre módulos MUST empaquetarse utilizando el estándar **CloudEvents 1.0 (JSON Format)**. El envelope MUST incluir los atributos requeridos por la especificación (`id`, `source`, `specversion`, `type`, `time`) y el payload de negocio MUST inyectarse en el atributo `data`.

Además, la metadata operativa (`tenant_slug`, `correlation_id`) MUST persistirse como _Extension Attributes_ de CloudEvents. En cualquier deployment profile, esta estructura CloudEvents MUST persistirse vía transactional outbox; el contrato lógico MUST permanecer estable.

#### Scenario: Consumidor estándar de nube

- **WHEN** un integration event es entregado a un sistema externo o consumer
- **THEN** el consumidor puede parsear el mensaje usando SDKs estándar de CloudEvents sin conocer la estructura interna del payload `data`, pudiendo enrutar mediante el atributo `type` o los extension attributes.

### Requirement: Transactional outbox en todos los deployment profiles

`EventBus.Publish` MUST escribir en `outbox_events` dentro de la misma transacción de base de datos que el write de negocio asociado, en deployment profile `aws` y `onprem`. MUST NOT publicar directamente a EventBridge, RabbitMQ ni otro broker desde application code.

#### Scenario: Commit atómico mensaje + evento

- **WHEN** se persiste un nuevo mensaje de correo y se publica `InboxMessageReceived`
- **THEN** la fila de negocio y la fila outbox quedan committed juntas o ninguna queda visible

### Requirement: Transactional outbox para background jobs

`TaskQueue.Enqueue` MUST escribir en `outbox_jobs` cuando el use case dispone de unit of work, en ambos deployment profiles. MUST NOT encolar directamente en SQS ni RabbitMQ desde application code.

#### Scenario: Job encolado vía outbox

- **WHEN** un handler de integration event encola `InvoiceExtractionRequested`
- **THEN** el job queda como fila pending en `outbox_jobs` hasta que el relay lo publique al broker

### Requirement: Separación de abstracciones events vs jobs

El sistema MUST mantener tablas/handlers distintos para integration events (`outbox_events`) y background jobs (`outbox_jobs`).

#### Scenario: Fallo de job no bloquea events

- **WHEN** un job falla y aplica reintento en outbox_jobs
- **THEN** el relay puede seguir procesando outbox_events con política de reintento independiente

### Requirement: Task queue multi-tenant

Al encolar un background job, el sistema MUST asociar el tenant del contexto de ejecución al mensaje de forma que el processor pueda restaurar el tenant antes de ejecutar la lógica de negocio.

#### Scenario: Job procesado con tenant

- **WHEN** un consumer recibe un job desde el broker
- **THEN** el processor ejecuta la lógica con ese tenant disponible en el contexto (header/atributo del mensaje)

### Requirement: Scheduler es concern separado

Las tareas periódicas MUST modelarse como disparos de scheduler que encolan commands/jobs, no como integration events de dominio ni como parte del event bus.

#### Scenario: Tick de sync periódico

- **WHEN** el scheduler dispara un tick configurado para sincronización de inbox
- **THEN** el sistema encola background job(s) de sync y no publica un “evento de dominio” ficticio solo por el tick

### Requirement: Relay solo publica al broker

El relay MUST claim filas outbox y publicarlas al broker del deployment profile. MUST NOT ejecutar handlers de negocio in-process en ningún profile.

#### Scenario: Relay no invoca handlers

- **WHEN** el relay procesa una fila outbox exitosamente
- **THEN** publica al broker y marca processed sin llamar a `IntegrationEventHandler` ni `JobHandler`

### Requirement: Entrega on-prem vía RabbitMQ

En deployment profile `onprem`, el relay MUST publicar integration events al exchange topic `bowerbird.events` y background jobs al exchange direct `bowerbird.jobs` de RabbitMQ. Los handlers MUST ejecutarse en procesos consumer separados (`events-consumer`, `jobs-consumer`), no en el relay.

#### Scenario: Evento publicado a RabbitMQ

- **WHEN** el relay on-prem claim un integration event pending
- **THEN** publica en RabbitMQ con routing key igual al detail type y marca outbox processed solo tras ACK del broker

#### Scenario: Job publicado a RabbitMQ

- **WHEN** el relay on-prem claim un background job pending
- **THEN** publica en la cola/work queue de jobs de RabbitMQ y marca processed solo tras publish confirmado

### Requirement: Entrega aws relay → EventBridge/SQS

En deployment profile `aws`, el relay MUST entregar `outbox_events` a EventBridge y `outbox_jobs` a SQS. Los handlers MUST ejecutarse en Lambdas consumidoras, no in-process en el relay.

#### Scenario: Outbox drena a EventBridge

- **WHEN** el relay aws claim un integration event pending
- **THEN** publica en EventBridge y marca processed solo tras `PutEvents` exitoso

#### Scenario: Outbox drena a SQS

- **WHEN** el relay aws claim un background job pending
- **THEN** envía el mensaje a SQS y marca processed solo tras `SendMessage` exitoso

### Requirement: Relay multi-tenant con fairness (aws)

El relay aws MUST procesar outbox de todos los tenants activos usando round-robin (o equivalente) con límite de filas y/o tiempo por tenant por ciclo, antes de publicar al broker.

#### Scenario: Cap por tenant

- **WHEN** un tenant tiene un backlog grande de outbox y otros tenants también tienen trabajo pending
- **THEN** el relay procesa como máximo el límite configurado de filas por tenant antes de pasar al siguiente tenant en el ciclo

### Requirement: Idempotencia en Handlers

Debido a que el relay y los brokers subyacentes proveen semántica de entrega _at-least-once_, todos los handlers (`IntegrationEventHandler` y `JobHandler`) MUST ser idempotentes. MUST usar el `message_id` para deducplicar operaciones o diseñar la lógica para que las repeticiones sean seguras.

#### Scenario: Reprocesamiento de job

- **WHEN** un job se entrega dos veces por un timeout de red
- **THEN** la segunda ejecución del handler detecta que ya se procesó (o aplica operaciones idempotentes) y finaliza exitosamente sin corromper el estado

### Requirement: Manejo de Poison Pills en Outbox (Retry Limits)

El sistema MUST prevenir el bloqueo del relay (head-of-line blocking). El Outbox Store MUST llevar registro de `attempts` y, tras exceder un umbral máximo de fallos (e.g. broker inaccesible repetidas veces para esa fila), marcar el mensaje como `failed` en lugar de `pending` para dejar de procesarlo.

#### Scenario: Fila bloqueante fallida

- **WHEN** un mensaje outbox no puede publicarse en el broker repetidas veces
- **THEN** alcanza el límite de `attempts`, se marca `failed` y el relay continúa con la siguiente fila pendiente
