# Arquitectura backend (API + Lambdas)

## Stack

- Go `net/http` (sin frameworks)
- PostgreSQL con `pgx` y queries raw
- Lambdas para API HTTP, SQS y EventBridge

## Arquitectura Hexagonal y Domain-Driven Design (DDD)

El backend sigue un estricto modelo de diseño guiado por el dominio y arquitectura limpia para mantener el código testable, mantenible e independiente de la infraestructura y los frameworks.

La carpeta `internal/` está rígidamente separada en dos categorías conceptuales con estructuras asimétricas:

### 1. `internal/platform/` (Cross-Cutting Concerns)

No contiene lógica de negocio. Es una estructura plana de librerías y utilitarios genéricos de bajo nivel que "saben cómo conectarse" a cosas (AWS, Bases de Datos, Entorno), pero no saben "por qué".

- **Por qué no usa Clean Architecture**: Aplicar capas como `domain` o `application` a una conexión de base de datos sería sobreingeniería. Son adaptadores técnicos puros.
- `platform/awsconfig`: Adaptador del AWS SDK.
- `platform/config`: Carga del `.env` y lectura de secretos en SSM.
- `platform/database`: Conexión genérica de `pgxpool`.
- `platform/events`: Publicación/suscripción de eventos de dominio (EventBridge).
- `platform/jobs`: Encolado/procesamiento de trabajo asíncrono (SQS jobs).

### 2. Bounded Contexts (Verticales de Negocio)

Carpetas como `internal/health/` (y futuras como `users/`, `invoices/`, `orders/`) representan módulos de negocio. A estas **sí** se les aplica Clean Architecture estricta en 4 capas:

- `domain/`: Reglas de negocio puras, tipos, constantes e **interfaces** de salida (ej. `Repository`). No importa DB ni HTTP ni AWS.
- `application/`: Casos de uso. Orquestan el flujo invocando las interfaces del dominio.
- `infrastructure/`: Implementaciones de las interfaces de salida (ej. `PostgresRepository`). Conocen la librería `pgx` e implementan SQL.
- `presentation/`: Controladores de entrada (ej. `http/handler.go`). Capturan un request, lo parsean, e invocan al caso de uso.

## Cómo añadir un nuevo dominio de negocio

Para añadir un nuevo módulo (ej. `invoices`), no lo mezcles con código existente. Crea su propia estructura en `internal/invoices/`:

1. Define los tipos de negocio e interfaces en `internal/invoices/domain/invoice.go` y `repository.go`.
2. Escribe la lógica orquestadora en `internal/invoices/application/create_invoice.go`.
3. Implementa el SQL en `internal/invoices/infrastructure/postgres_repository.go`.
4. Crea el endpoint en `internal/invoices/presentation/http/handler.go`.
5. Haz el "wiring" (Inyección de Dependencias) **exclusivamente** en el `main.go`.

### Reglas de Dependencias Strictas

- `domain/` no puede importar absolutamente nada de los demás (excepto la librería standard de Go).
- `application/` solo puede importar de `domain/`.
- `presentation/` y `infrastructure/` importan `domain/` y `application/`.
- **Un dominio NUNCA puede importar otro dominio**. Si `invoices` necesita algo de `users`, se deben comunicar por ID referencial o eventos, no por imports directos.
- `platform/` nunca puede importar a un dominio. Los dominios pueden importar tipos base de `platform` si es estrictamente necesario (ej. `pgxpool.Pool`).

---

## Carga de Secretos y Configuración

Tanto el servidor local (`cmd/api`) como las Lambdas cargan sus credenciales y variables en el proceso de inicialización (boot) consumiendo el servicio de **SSM Parameter Store** (`SecureString`).

1. La app lee del entorno el nombre del parámetro (ej. `SSM_PARAMETER_NAME=/bowerbird/local/secrets`).
2. Consulta SSM para obtener el JSON con todas las claves (DB, API Keys, etc).
3. Deserializa el JSON hacia el struct de la app `Config`.

En desarrollo local, LocalStack inicializa un parámetro simulado en SSM gracias al script `apps/backend/scripts/init-localstack.sh`. En producción, CDK otorga permisos de lectura `ssm:GetParameter` a las Lambdas.

## Emulación local de AWS con simplicidad

- Servicios emulados en LocalStack: S3, SQS, EventBridge, SSM.
- No se hace deploy de Lambdas a LocalStack para el loop diario de desarrollo.
- En su lugar, `cmd/api/main.go` activa un event loop local con dos pollers separados: uno para eventos EventBridge y otro para jobs SQS, reutilizando los mismos handlers usados por las Lambdas.
- Resultado: alta fidelidad de infraestructura AWS con ciclo de desarrollo rápido (`air`) y baja complejidad operativa.
