# Bowerbird Monorepo

Monorepo para una arquitectura SPA (Angular) + API/Lambda (Golang) + IaC (AWS CDK TypeScript), orquestado con Turborepo.

## Resumen de funcionalidades

- Backend API en Go (`net/http`, `pgx`, Lambda HTTP/SQS/EventBridge): [docs/technical/architecture/backend-api.md](./docs/technical/architecture/backend-api.md)
- Frontend Angular standalone + PWA (service worker, install prompt, update flow): [docs/technical/architecture/frontend-web.md](./docs/technical/architecture/frontend-web.md)
- Despliegue AWS con CDK (S3 + CloudFront + API Gateway + Route53): [docs/technical/deployment/aws.md](./docs/technical/deployment/aws.md)
- Calidad de desarrollo (AI Skills, Prettier): [docs/technical/quality/development-quality.md](./docs/technical/quality/development-quality.md)

## Documentación completa

- Índice de documentación: [docs/README.md](./docs/README.md)
- Setup y entorno local: [docs/technical/getting-started.md](./docs/technical/getting-started.md)
- Convenciones de documentación de negocio: [docs/product/README.md](./docs/product/README.md)
- Tooling de análisis de código (CodeGraph): [docs/technical/tooling/codegraph.md](./docs/technical/tooling/codegraph.md)

## Inicio rápido

```bash
mise install
pnpm install
pnpm run dev
```

Comandos principales:

- `pnpm run build`
- `pnpm run test`
- `pnpm run lint`
- `pnpm run format`
- `pnpm run format:check`
- `pnpm run deploy`
