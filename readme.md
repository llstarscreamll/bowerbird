# Bowerbird

Monorepo: Angular PWA + Go API/Lambdas + Pulumi (AWS Lambda + Neon + Cloudflare), orchestrated with Turbo and pnpm.

## Quick start

```bash
mise install
pnpm install
mise run dev
```

Open `https://app.bowerbird.dev` (see [Getting started](./docs/technical/getting-started.md) for hosts/Caddy).

## Testing

Run the full local suite (Postgres reset, Go + PWA unit tests, e2e):

```bash
mise run test:full
```

Stop `mise run dev` first. Details: [Full test loop](./docs/technical/getting-started.md#full-test-loop).

## Docs

- [Docs index](./docs/README.md)
- [Getting started](./docs/technical/getting-started.md)
- [Backend](./docs/technical/architecture/backend-api.md) · [Frontend](./docs/technical/architecture/frontend-web.md) · [AWS deploy](./docs/technical/deployment/aws.md) · [On-prem fleet](./docs/technical/deployment/onprem.md)

## Commands

| Command                            | Purpose                                                                       |
| ---------------------------------- | ----------------------------------------------------------------------------- |
| `mise run dev`                     | Local stack (infra + API/PWA/workers, `.env`)                                 |
| `mise run test:full`               | Full suite: wipe Postgres, Go + PWA tests, e2e (`.env.test`; chromium + http) |
| `pnpm run build`                   | Build all packages                                                            |
| `pnpm run test`                    | Unit/integration tests (no DB reset, no e2e)                                  |
| `pnpm run test:e2e`                | Playwright e2e (stack must already be running)                                |
| `pnpm run lint`                    | Lint all packages                                                             |
| `pnpm run format` / `format:check` | Prettier                                                                      |
| `pnpm run deploy`                  | Build + Pulumi AWS **and** on-prem fleet (parallel)                           |
| `pnpm run deploy:aws`              | AWS Lambda track only (`apps/deploy/aws`)                                     |
| `pnpm run deploy:onprem`           | Client VM fleet only (`apps/deploy/onprem`)                                   |
