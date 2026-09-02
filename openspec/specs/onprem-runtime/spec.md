# onprem-runtime Specification

## Purpose

Define el runtime de despliegue on-premise full-local en un solo servidor (Docker Compose): control-plane y tenant locales, MinIO, RabbitMQ, relay/consumers, y configuración por archivos `.env` planos.

## Requirements

### Requirement: Deployment profile onprem

El sistema MUST soportar un deployment profile `onprem` que ejecute API HTTP, outbox-relay, events-consumer, jobs-consumer, RabbitMQ, PostgreSQL (control-plane + tenant), MinIO y reverse proxy en el mismo servidor, sin EventBridge, SQS, SSM ni Lambdas de AWS.

#### Scenario: Arranque Compose health

- **WHEN** un operador ejecuta el stack on-prem documentado en una VM con Docker
- **THEN** los servicios `api`, `outbox-relay`, `events-consumer`, `jobs-consumer`, `rabbitmq`, `postgres`, `minio` y el proxy HTTPS responden healthy según los healthchecks documentados

### Requirement: Full local control-plane y auth

En profile `onprem`, identity, entitlements, memberships y la base control-plane MUST operar contra PostgreSQL local. El sistema MUST NOT requerir llamadas HTTP a un control-plane SaaS remoto para autenticar usuarios ni resolver entitlements.

#### Scenario: Login sin SaaS

- **WHEN** un usuario inicia sesión contra la API on-prem
- **THEN** la autenticación y autorización se resuelven con datos locales (control-plane DB) sin dependencia de servicios HTTP remotos de Bowerbird SaaS

### Requirement: Tenant DB fija en el servidor

En el despliegue on-prem de un cliente, el runtime MUST operar contra una única tenant database local configurada.

#### Scenario: Requests de negocio usan tenant local

- **WHEN** la API procesa una operación de negocio autenticada en on-prem
- **THEN** lee/escribe la tenant DB local configurada

### Requirement: Object storage MinIO obligatorio

En profile `onprem`, el almacenamiento de objetos MUST usar MinIO (API compatible S3). Credenciales y endpoint MUST configurarse vía `.env`.

#### Scenario: Upload/presign contra MinIO

- **WHEN** la aplicación genera una URL de upload o descarga de archivo en on-prem
- **THEN** la URL apunta al endpoint MinIO configurado

### Requirement: Secretos y passwords vía .env planos

Secretos de aplicación, passwords de Postgres, RabbitMQ y MinIO MUST poder suministrarse mediante `.env` planos. MUST NOT exigir vault remoto ni SSM.

#### Scenario: Arranque con .env

- **WHEN** el operador coloca un `.env` con variables documentadas incluyendo `RABBITMQ_URL`
- **THEN** `api`, relay y consumers arrancan sin consultar SSM

### Requirement: RabbitMQ broker on-prem

El profile `onprem` MUST usar RabbitMQ para integration events (topic exchange) y background jobs (direct exchange + work queue). MUST NOT usar handlers in-process en el relay.

#### Scenario: Pipeline equivalente a aws

- **WHEN** un integration event se publica vía outbox en on-prem
- **THEN** el flujo es outbox → outbox-relay → RabbitMQ → events-consumer → handler, análogo a outbox → relay → EventBridge → Lambda en aws

### Requirement: Procesos separados relay y consumers

El profile `onprem` MUST ejecutar outbox-relay, events-consumer y jobs-consumer como procesos/contenedores distintos del HTTP `api`.

#### Scenario: Fallo aislado en jobs-consumer

- **WHEN** el jobs-consumer falla o reinicia
- **THEN** el outbox-relay y events-consumer pueden seguir operando

### Requirement: Compatibilidad del deployment profile aws

El deployment profile `aws` MUST usar el mismo modelo outbox y relay→broker→consumers, con EventBridge/SQS y Lambdas en lugar de RabbitMQ y contenedores Go.

#### Scenario: Mismo outbox write

- **WHEN** un use case publica en aws u onprem
- **THEN** ambos escriben outbox en la misma transacción; solo difiere el broker adapter post-relay

### Requirement: Desarrollo local usa profile onprem con RabbitMQ

El entorno de desarrollo local MUST usar `DEPLOYMENT_TARGET=onprem` con RabbitMQ en Docker, relay y consumers — idéntico al cliente on-prem. MUST NOT usar LocalStack EventBridge/SQS ni pollers embebidos en API.

#### Scenario: pnpm run dev con RabbitMQ

- **WHEN** un desarrollador ejecuta el flujo local documentado
- **THEN** RabbitMQ está disponible, relay publica outbox y consumers procesan mensajes sin LocalStack messaging

#### Scenario: Secretos locales vía .env

- **WHEN** el backend arranca en desarrollo local
- **THEN** carga configuración desde `apps/backend/.env` incluyendo conexión RabbitMQ

### Requirement: Trazabilidad y Correlation IDs

El sistema MUST generar o propagar un `correlation_id` desde el origen (request HTTP) y persistirlo en el outbox payload. Este ID MUST viajar a través del broker y ser impreso en los logs estructurados del consumer (events y jobs) para garantizar observabilidad end-to-end.

#### Scenario: Seguimiento de logs cross-process

- **WHEN** un error ocurre en el `jobs-consumer`
- **THEN** el operador puede buscar el `correlation_id` en los logs del agregador y trazar el flujo hasta la request HTTP original de la `api`

### Requirement: Auto-reconexión y Backoff

Los procesos background (`outbox-relay`, `events-consumer`, `jobs-consumer`) MUST implementar auto-reconexión robusta con backoff exponencial. MUST tolerar cortes de red, reinicios de PostgreSQL o caídas de RabbitMQ sin abortar (crash loop interminable) a menos que ocurra un error fatal de configuración.

#### Scenario: Reinicio del Broker en vivo

- **WHEN** el contenedor RabbitMQ se reinicia mientras los consumers on-prem están activos
- **THEN** los consumers detectan el cierre de la conexión, pausan operaciones y reconectan exitosamente usando backoff hasta que el broker vuelve a estar disponible
