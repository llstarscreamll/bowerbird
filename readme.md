# Canopy

Monorepo habitat for commercial systems (Atta first). Angular, Go
API/Lambdas, and Pulumi (AWS Lambda + Neon + Cloudflare), orchestrated
with Turbo and pnpm.

Why these names: [Naming](./docs/product/naming.md).

## Quick start

```bash
mise install
pnpm install
mise //apps/atta:dev
```

Open `https://app.atta.dev` (see [Getting started](./docs/technical/getting-started.md)
for hosts/Caddy). From `apps/atta` you can run `mise :dev`.

## Testing

Run the full local suite (Postgres reset, Go + web unit tests, e2e):

```bash
mise //apps/atta:test:full
```

Stop `mise //apps/atta:dev` first. Details: [Full test loop](./docs/technical/getting-started.md#full-test-loop).

## Docs

- [Docs index](./docs/README.md)
- [Naming](./docs/product/naming.md) · [Monorepo layout](./docs/technical/architecture/monorepo.md)
- [Getting started](./docs/technical/getting-started.md)
- [Backend](./docs/technical/architecture/backend-api.md) · [Frontend](./docs/technical/architecture/frontend-web.md) · [AWS deploy](./docs/technical/deployment/aws.md) · [On-prem fleet](./docs/technical/deployment/onprem.md)

## Commands

Product tasks live in `apps/<product>/mise.toml`. Habitat scripts
(`lint`, `test`, `build`, `format`) stay at the Canopy root.

| Command                            | Purpose                                                                |
| ---------------------------------- | ---------------------------------------------------------------------- |
| `mise //apps/atta:dev`             | Local stack (infra + Atta API/PWA/workers, `apps/atta/.env`)           |
| `mise //apps/atta:test:full`       | Full suite: wipe Postgres, Go + web tests, e2e (`apps/atta/.env.test`) |
| `mise //apps/atta:infra:up`        | Compose only                                                           |
| `mise //apps/atta:deploy`          | Build + Pulumi AWS **and** on-prem fleet                               |
| `mise //apps/atta:deploy:aws`      | AWS Lambda track only                                                  |
| `mise //apps/atta:deploy:onprem`   | Client VM fleet only                                                   |
| `pnpm run build`                   | Build all packages                                                     |
| `pnpm run test`                    | Unit/integration tests (no DB reset, no e2e)                           |
| `pnpm run lint`                    | Lint all packages                                                      |
| `pnpm run format` / `format:check` | Prettier                                                               |
