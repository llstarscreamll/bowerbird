# Eventos vs Jobs (Arquitectura Backend)

Este documento define, de forma operativa, la diferencia entre **eventos** y **jobs** en Bowerbird, y deja reglas claras para decidir cuándo usar cada uno.

## TL;DR

- Usa **eventos (EventBridge)** para comunicar hechos de dominio entre contextos desacoplados.
- Usa **jobs (SQS)** para ejecutar trabajo asíncrono procesable por workers.
- Si un flujo necesita ambas cosas: primero publica evento, luego un consumidor decide si encola jobs.

## Definiciones

### Evento

Un evento representa un hecho que ya ocurrió en el negocio.

- Ejemplo: `InboxMessageReceived`.
- Se publica en EventBridge (`internal/platform/events`).
- Puede tener múltiples suscriptores (fan-out por reglas del bus).
- El productor no conoce quién lo consumirá.

### Job

Un job representa trabajo pendiente por ejecutar.

- Ejemplo: `InvoiceExtractionRequested`.
- Se encola en SQS (`internal/platform/jobs`).
- Lo procesa un handler de jobs (`HandleSQSEvent`).
- Tiene semántica de cola: retries, backoff y control de procesamiento.

## Setup actual del proyecto

### Capa de eventos (`apps/backend/internal/platform/events`)

- `eventbridge_publisher.go`: publica eventos de negocio a EventBridge.
- `handler.go`: enruta eventos EventBridge por `DetailType()` a `EventBridgeSubscriber`.
- `poller.go`: poller exclusivo de EventBridge queue (en local loop).

### Capa de jobs (`apps/backend/internal/platform/jobs`)

- `sqs_queue.go`: encola jobs en SQS (`JobType`, `TenantID`, `Payload`).
- `handler.go`: enruta mensajes SQS por `JobType()` a `SQSProcessor`.
- `poller.go`: poller exclusivo de SQS jobs (en local loop).

### Wiring en runtime

En `apps/backend/cmd/api/main.go` el wiring es explícito y separado:

1. Se crea `eventHandler` con suscriptores EventBridge.
2. Se crea `jobHandler` con procesadores de jobs SQS.
3. Se inician dos pollers distintos:
   - `platformJobs.NewPoller(...)` para jobs.
   - `events.NewPoller(...)` para eventos.

## Cuándo usar cada uno

Usa esta regla de decisión:

1. Si comunicas un hecho de dominio para que otros contextos reaccionen sin acoplamiento directo, usa **Evento**.
2. Si necesitas ejecutar una tarea concreta en background con semántica de cola, usa **Job**.
3. Si una reacción a evento requiere trabajo pesado o reintentos, convierte esa reacción en **Job**.

## Matriz de decisión

| Pregunta                                                             | Evento (EventBridge) | Job (SQS)                        |
| -------------------------------------------------------------------- | -------------------- | -------------------------------- |
| ¿Describe algo que ya pasó?                                          | Si                   | No                               |
| ¿Representa trabajo pendiente?                                       | No                   | Si                               |
| ¿Puede haber varios consumidores independientes?                     | Si (fan-out)         | Normalmente no (consumo de cola) |
| ¿Necesita semántica de worker/reintento de tarea?                    | No (por defecto)     | Si                               |
| ¿El productor debe permanecer totalmente desacoplado del consumidor? | Si                   | Puede conocer el tipo de job     |

## Anti-patrones a evitar

- Publicar en EventBridge un mensaje que en realidad es un comando de ejecución inmediata.
- Meter orquestación de jobs en la capa `platform/events`.
- Mezclar handlers/pollers de eventos y jobs en la misma abstracción.
- Usar jobs para propagar hechos de negocio entre bounded contexts cuando lo correcto es evento.

## Ejemplo de flujo recomendado

1. `inbox` detecta un nuevo mensaje y publica `InboxMessageReceived` (evento).
2. `invoices` consume ese evento y evalúa si corresponde procesamiento.
3. Si corresponde trabajo costoso, encola `InvoiceExtractionRequested` (job).
4. Un processor de jobs ejecuta la extracción y persistencia.

## Regla de diseño para nuevas features

Antes de implementar, clasifica explícitamente cada mensaje como `event` o `job` en la PR.

- Si es `event`, implementa en `internal/platform/events`.
- Si es `job`, implementa en `internal/platform/jobs`.
- Si hay ambos, mantenlos como dos contratos distintos, aunque compartan datos similares.
