# AGENTS

## Repo essentials

- Use Mise for tool installation and version management (`.mise.toml`). Run `mise install` after clone.
- Node.js runtime is pinned to `24.x` (`package.json` engines + `.nvmrc`).
- Go runtime is pinned to `1.25.x` (`.mise.toml` + `apps/api/go.mod`).
- Package manager is `pnpm` (`packageManager: pnpm@10.16.1`). Prefer `pnpm` commands; do not switch to npm.
- Monorepo layout is fixed by `pnpm-workspace.yaml`: `apps/*` and `packages/*`.
- Task orchestration is through Turborepo (`turbo.json`). Root scripts are the source of truth.
- CodeGraph is configured for codebase structural exploration. Run `codegraph init -i` if index is missing or out of date.

## High-value commands

- Install deps: `pnpm install`
- Full dev stack: `pnpm run dev` (starts Docker infra, API dev server, Angular dev server)
- Infra only: `pnpm run infra:up` / `pnpm run infra:down`
- Format: `pnpm run format` (Prettier on TS/Markdown, etc.)
- Format check: `pnpm run format:check`
- Full checks: `pnpm run lint && pnpm run test && pnpm run build`
- Deploy (root): `pnpm run deploy` (runs build first, then `@bowerbird/infra` deploy)

## Targeted package commands

- API only: `pnpm --filter @bowerbird/api dev|lint|test|build`
- Web only: `pnpm --filter @bowerbird/web dev|lint|test|build`
- Infra only: `pnpm --filter @bowerbird/infra lint|build|deploy|synth`
- PWA local verification: `pnpm --filter @bowerbird/web build && pnpm --filter @bowerbird/web preview:pwa` (serves on `http://localhost:4300`)

## Runtime + setup gotchas

- API environment separation: `.env` handles process/infra flags, `secrets.json` handles AWS SSM secrets (DB URLs, Queues, etc.). See `.env.example` and `secrets.example.json`.
- API dev uses Air live reload via `apps/api/.air.toml`; Air must be installed on host: `go install github.com/air-verse/air@latest`.
- If `air` is not found, add `$(go env GOPATH)/bin` to shell `PATH`.
- Local dependencies required by default flow: Postgres (`5432`), Redis (`6379`), LocalStack (`4566`) from `docker-compose.yml`.
- Local AWS resources (S3/SQS/EventBridge/SSM) are auto-created by `apps/api/scripts/init-localstack.sh` reading from `secrets.json`.
- The project uses `husky` and `lint-staged` for pre-commit hooks to format and lint code dynamically.

## Package boundaries and entrypoints

- `apps/api`
  - Local HTTP server entrypoint: `cmd/api/main.go` (net/http + pgx pool)
  - Lambda entrypoints: `cmd/lambda/http/main.go`, `cmd/lambda/sqs/main.go`, `cmd/lambda/eventbridge/main.go`
  - Internal layers: `internal/config`, `internal/db`, `internal/repository`, `internal/handlers`
- `apps/web`
  - Angular standalone app; service worker enabled in production (`apps/web/angular.json`, `apps/web/src/app/app.config.ts`)
  - Dev server proxies `/api/*` to `http://localhost:8080` via `apps/web/proxy.conf.json`
- `packages/infra`
  - CDK app entrypoint: `packages/infra/bin/infra.ts`
  - Main stack: `packages/infra/lib/bowerbird-stack.ts`

## Deploy-specific constraints (important)

- `packages/infra/.env` is required (`AWS_ACCOUNT_ID`, domain vars, etc.; see `.env.example`).
- CDK app hard-requires `AWS_REGION=us-east-1` (validated at startup).
- Infra stack packages frontend from local build output: `apps/web/dist/web/browser`; run web/root build before deploy.
- CloudFront/S3 deploy strategy intentionally keeps old hashed assets (`prune: false`) and invalidates only entrypoints. Do not change this casually; it prevents old `index.html` from breaking on missing bundles.
