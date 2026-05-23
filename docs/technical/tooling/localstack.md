# Tooling: LocalStack

## Propósito

Emular servicios AWS críticos del backend en local sin montar una infraestructura paralela compleja.

Servicios emulados actualmente:

- S3
- SQS
- EventBridge
- SSM Parameter Store

## Como se usa en este repo

- LocalStack vive en `docker-compose.yml` y expone `http://localhost:4566`.
- La inicialización de recursos se hace automáticamente al arrancar con:
  `apps/api/scripts/init-localstack.sh`.
- El backend local usa `AWS_ENDPOINT_URL` para apuntar al endpoint local.

## Recursos inicializados automáticamente

- Cola SQS principal: `bowerbird-local-sqs`
- Cola SQS para eventos de EventBridge: `bowerbird-local-eventbridge`
- Event bus: `bowerbird-local-bus`
- Rule EventBridge: `bowerbird-local-rule` (source `bowerbird.app` -> cola `bowerbird-local-eventbridge`)
- Bucket S3: `bowerbird-local-bucket`
- Parámetro SSM: `/bowerbird/local/sample`

## Principio de simplicidad aplicado

- Se emula infraestructura AWS con LocalStack.
- Se mantiene ejecución del código Go en proceso local con `air`.
- Los handlers de Lambda se reutilizan en desarrollo mediante pollers locales para evitar dos implementaciones distintas.
