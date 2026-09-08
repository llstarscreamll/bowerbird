# Bowerbird

Monorepo: Angular PWA + Go API/Lambdas + Pulumi (AWS Lambda + Neon + Cloudflare), orchestrated with Turbo and pnpm.

## Quick start

```bash
mise install
pnpm install
pnpm run dev
```

Open `https://app.bowerbird.dev` (see [Getting started](./docs/technical/getting-started.md) for hosts/Caddy).

## Docs

- [Docs index](./docs/README.md)
- [Getting started](./docs/technical/getting-started.md)
- [Backend](./docs/technical/architecture/backend-api.md) · [Frontend](./docs/technical/architecture/frontend-web.md) · [AWS deploy](./docs/technical/deployment/aws.md) · [On-prem fleet](./docs/technical/deployment/onprem.md)

## Commands

| Command                            | Purpose                                             |
| ---------------------------------- | --------------------------------------------------- |
| `pnpm run build`                   | Build all packages                                  |
| `pnpm run test`                    | Unit/integration tests                              |
| `pnpm run test:e2e`                | Playwright e2e                                      |
| `pnpm run lint`                    | Lint all packages                                   |
| `pnpm run format` / `format:check` | Prettier                                            |
| `pnpm run deploy`                  | Build + Pulumi AWS **and** on-prem fleet (parallel) |
| `pnpm run deploy:aws`              | AWS Lambda track only (`apps/deploy/aws`)           |
| `pnpm run deploy:onprem`           | Client VM fleet only (`apps/deploy/onprem`)         |
