# Turno Monorepo

Monorepo para una arquitectura SPA (Angular) + API/Lambda (Golang) + IaC (AWS CDK TypeScript), orquestado con Turborepo.

## Stack

- `apps/api`: API HTTP con `net/http`, queries raw a PostgreSQL con `pgx`, y Lambdas para API Gateway, SQS y EventBridge.
- `apps/web`: Angular standalone + signals + TailwindCSS + PWA (manifest + service worker).
- `packages/infra`: AWS CDK con deployment de frontend en S3 + CloudFront y backend en API Gateway + Lambda.

## Requisitos

- Node.js 20+
- pnpm 10+
- Go 1.23+
- Docker + Docker Compose
- AWS CLI autenticado (para deploy)

## Variables de entorno

### API local

1. Copiar `apps/api/.env.example` a `apps/api/.env`.
2. Exportar variables antes de correr la API:

```bash
export $(grep -v '^#' apps/api/.env | xargs)
```

### Infraestructura CDK

1. Copiar `packages/infra/.env.example` a `packages/infra/.env`.
2. Usar el dominio comprado en la cuenta (por defecto `money-path.co`).

Variables clave:

- `ROOT_DOMAIN=money-path.co`
- `APP_SUBDOMAIN=app` (queda `app.money-path.co`)
- `API_SUBDOMAIN=api` (queda `api.money-path.co`)
- `AWS_REGION=us-east-1` (requerido por el certificado de CloudFront)

## Desarrollo local

Levanta infraestructura local (Postgres, Redis, MinIO) y apps de desarrollo:

```bash
pnpm run dev
```

Esto ejecuta:

- `docker compose up -d`
- `go run ./cmd/api` (API local en `http://localhost:8080`)
- `ng serve` (web en `http://localhost:4200`)

La web usa `proxy.conf.json` para redirigir `/api/*` al backend local.

## Scripts utiles

- `pnpm run infra:up`: inicia Postgres, Redis y MinIO.
- `pnpm run infra:down`: detiene infraestructura local.
- `pnpm run build`: build de todo el monorepo con Turbo.
- `pnpm run test`: tests de Go, Angular e Infra.
- `pnpm run lint`: checks estaticos.
- `pnpm run deploy`: despliegue de CDK.

## PWA (Angular)

El frontend incluye setup de PWA siguiendo convenciones recomendadas de Angular:

- `@angular/service-worker` habilitado solo en produccion.
- `ngsw-config.json` con cache de app-shell, assets y estrategia `freshness` para `/api/**`.
- `manifest.webmanifest` con iconos `192`, `512` y `maskable`.
- Captura de `beforeinstallprompt` para mostrar boton de instalacion.
- Deteccion de nuevas versiones con `SwUpdate` y accion de refresh.

Para probar localmente el comportamiento PWA (service worker + offline):

```bash
pnpm --filter @turno/web build
pnpm --filter @turno/web preview:pwa
```

Abre `http://localhost:4300` y valida en DevTools > Application (Manifest / Service Workers).

La orquestacion del monorepo se centraliza en `pnpm run` + Turborepo para evitar redundancias.

## Deploy AWS (CDK)

Desde la raiz:

```bash
pnpm run build
cd packages/infra
pnpm exec cdk bootstrap aws://$AWS_ACCOUNT_ID/$AWS_REGION
pnpm exec cdk deploy --all --require-approval never
```

### Topologia desplegada

- S3 privado para assets Angular.
- CloudFront con:
  - cabeceras de seguridad,
  - fallback SPA (`403/404 -> /index.html`),
  - routing de `https://<app-domain>/api/*` hacia `https://<api-domain>`.
- Estrategia de cache para frontend en deploy:
  - assets compilados con hash (`*.js`, `*.css`, etc.) con `Cache-Control: public, max-age=31536000, immutable`.
  - `prune: false` para no borrar versiones antiguas y evitar fallos cuando un `index.html` viejo referencia bundles previos.
  - entrypoints (`index.html`, `ngsw.json`, `ngsw-worker.js`, `safety-worker.js`, `manifest.webmanifest`) con `Cache-Control: public, max-age=0, must-revalidate, s-maxage=300`.
  - invalidacion selectiva en CloudFront solo para entrypoints en cada deploy.
- API Gateway HTTP API con Lambda Go.
- Dominio custom para API Gateway (`api.<root-domain>`).
- Route53 A records para app y api.
- Lambda adicional para SQS + regla de EventBridge.

## Arquitectura del backend Go

Estructura simple e idiomatica:

- `cmd/api`: servidor HTTP local con middlewares de seguridad y CORS.
- `cmd/lambda/http`: handler para API Gateway.
- `cmd/lambda/sqs`: handler para eventos SQS.
- `cmd/lambda/eventbridge`: handler para eventos EventBridge.
- `internal/db`: conexion `pgxpool`.
- `internal/repository`: acceso a datos con queries raw.
- `internal/handlers`: handlers HTTP y de eventos.
