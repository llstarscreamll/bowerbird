# Arquitectura backend (API + Lambdas)

## Stack

- Go `net/http` (sin frameworks)
- PostgreSQL con `pgx` y queries raw
- Lambdas para API HTTP, SQS y EventBridge

## Capacidades técnicas implementadas

- Endpoint de healthcheck (`/health` y `/api/health`) con verificación real de base de datos.
- Middlewares HTTP de seguridad y CORS para el servidor local.
- Handler Lambda HTTP para API Gateway.
- Handler Lambda para eventos SQS.
- Handler Lambda para eventos EventBridge.
- Ejecucion local con LocalStack para S3, SQS, EventBridge y SSM.
- Event loop local (poller SQS/EventBridge) que reutiliza los mismos handlers de Lambda para evitar dobles implementaciónes.

## Estructura y entrypoints

- Servidor local: `apps/api/cmd/api/main.go`
- Lambda HTTP: `apps/api/cmd/lambda/http/main.go`
- Lambda SQS: `apps/api/cmd/lambda/sqs/main.go`
- Lambda EventBridge: `apps/api/cmd/lambda/eventbridge/main.go`
- Configuración: `apps/api/internal/config`
- DB pool: `apps/api/internal/db`
- Repositorios: `apps/api/internal/repository`
- Handlers: `apps/api/internal/handlers`
- AWS config local/prod: `apps/api/internal/awsconfig`
- Pollers locales de eventos: `apps/api/internal/events/poller.go`

## Carga de Secretos y Configuración

Tanto el servidor local (`cmd/api`) como las Lambdas cargan sus credenciales y variables en el proceso de inicialización (boot) consumiendo el servicio de **SSM Parameter Store** (`SecureString`).

1. La app lee del entorno el nombre del parámetro (ej. `SSM_PARAMETER_NAME=/bowerbird/local/secrets`).
2. Consulta SSM para obtener el JSON con todas las claves (DB, API Keys, etc).
3. Deserializa el JSON hacia el struct de la app `Config`.

En desarrollo local, LocalStack inicializa un parámetro simulado en SSM gracias al script `apps/api/scripts/init-localstack.sh`. En producción, CDK otorga permisos de lectura `ssm:GetParameter` a las Lambdas.

## Live reload en desarrollo

- Config: `apps/api/.air.toml`
- Comando: `pnpm --filter @bowerbird/api dev`
- Instalacion de Air:

```bash
go install github.com/air-verse/air@latest
```

Si `air` no se encuentra, agrega `$(go env GOPATH)/bin` al `PATH`.

## Emulacion local de AWS con simplicidad

- Servicios emulados en LocalStack: S3, SQS, EventBridge, SSM.
- No se hace deploy de Lambdas a LocalStack para el loop diario de desarrollo.
- En su lugar, `cmd/api/main.go` activa un event loop local que consume colas SQS de LocalStack y ejecuta los mismos handlers usados por las Lambdas (`HandleSQSEvent` y `HandleEventBridgeEvent`).
- Resultado: alta fidelidad de infraestructura AWS con ciclo de desarrollo rápido (`air`) y baja complejidad operativa.
