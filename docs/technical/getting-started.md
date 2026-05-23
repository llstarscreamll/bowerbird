# Getting started técnico y entorno local

## Requisitos

- Mise
- Air
- Docker + Docker Compose
- AWS CLI autenticado (solo para deploy)

Versiones definidas por el repo en `.mise.toml`:

- Node.js `24`
- Go `1.25`
- pnpm `10.16.1`

## Setup inicial

```bash
mise install
pnpm install
go install github.com/air-verse/air@latest
```

Si quieres ejecutar comandos con el toolchain del repo sin tocar lo global:

```bash
mise x -- pnpm run dev
```

## Variables de entorno

### API local

1. Copiar `apps/api/.env.example` a `apps/api/.env`.
2. Copiar `apps/api/secrets.example.json` a `apps/api/secrets.json`.
3. Revisar y ajustar los valores locales.

**¿Cuándo usar `.env` y cuándo `secrets.json`?**

La regla general del proyecto es que el **`.env` controla el contenedor del proceso** y el **`secrets.json` controla el negocio y la infraestructura sensible**.

Usa **`.env`** únicamente para variables de infraestructura de ejecución que cambian cómo arranca la app:

- Puerto del servidor local (`PORT`).
- Credenciales AWS de IAM para el SDK (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`).
- Región de AWS (`AWS_REGION`).
- URL de emulación para el SDK (`AWS_ENDPOINT_URL`).
- El path en SSM donde buscar los secretos (`SSM_PARAMETER_NAME`).
- Flags de ejecución de desarrollo (`ENABLE_LOCAL_EVENT_LOOP`).

Usa **`secrets.json`** para cualquier clave secreta, cadena de conexión o recurso creado _dentro_ de AWS que sea sensible:

- URL de la Base de datos (`database_url`).
- Nombres de buckets (`s3_bucket_name`).
- URLs de colas de SQS y EventBridge (`sqs_queue_url`, `eventbridge_queue_url`).
- API Keys de proveedores terceros (`third_party_api_key`).
- Tokens o Salts de encriptación.

**Nota:** La API carga su configuración principal leyendo el `SSM_PARAMETER_NAME` definido en el `.env`. En tu entorno local, LocalStack lee automáticamente tu archivo `secrets.json` en disco y lo inyecta como un `SecureString` dentro de SSM para emular el flujo productivo.

4. Exportar variables si vas a ejecutar comandos directos de Go:

```bash
export $(grep -v '^#' apps/api/.env | xargs)
```

Variables clave en `.env` para emulación AWS local:

- `AWS_ENDPOINT_URL=http://localhost:4566`
- `ENABLE_LOCAL_EVENT_LOOP=true`
- `SSM_PARAMETER_NAME=/bowerbird/local/secrets`

**Nota:** La API carga su configuración principal (base de datos, colas) desde el parámetro de SSM en el boot. LocalStack lee automáticamente tu `secrets.json` y lo inyecta como un SecureString en SSM al inicializarse.

### Infraestructura CDK

1. Copiar `packages/infra/.env.example` a `packages/infra/.env`.
2. Variables clave:

- `AWS_ACCOUNT_ID`
- `AWS_REGION=us-east-1`
- `ROOT_DOMAIN=money-path.co`
- `APP_SUBDOMAIN=app`
- `API_SUBDOMAIN=api`

## Desarrollo local

Ejecuta todo el stack de desarrollo:

```bash
pnpm run dev
```

Esto levanta:

- Postgres, Redis y LocalStack (S3/SQS/EventBridge/SSM) en Docker
- API Go con live reload (`air -c .air.toml`) en `http://localhost:8080`
- Angular dev server en `http://localhost:4200`

La web usa `apps/web/proxy.conf.json` para enrutar `/api/*` hacia `http://localhost:8080`.

LocalStack inicializa automáticamente recursos con `apps/api/scripts/init-localstack.sh` al arrancar Docker.

## Comandos principales

- `pnpm run infra:up`
- `pnpm run infra:down`
- `pnpm run build`
- `pnpm run test`
- `pnpm run lint`
- `pnpm run format`
- `pnpm run format:check`
- `pnpm run deploy`

## Tooling adicional

Para exploración estructural del código y análisis de impacto, revisar CodeGraph:

- [Tooling: CodeGraph](./tooling/codegraph.md)
- [Tooling: LocalStack](./tooling/localstack.md)
