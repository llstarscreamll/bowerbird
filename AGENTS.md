# AGENTS

## Global Agent Rules & Output Style

- **Extreme Conciseness**: Be direct and extremely brief in all your responses.
- **No Conversational Filler**: Do not use preambles, postambles, or conversational phrases (e.g., "Here is the code," "I have finished," "Let me know").
- **Action-Oriented**: Focus strictly on the requested actions, tool calls, and essential explanations.
- **Output Style**: Zero yapping. Stop talking and just output the requested code, steps, or results.
- **Git Commits**: NEVER commit automatically unless explicitly requested. NEVER include co-authorship, assistance tags, or AI watermarks (e.g., "Assisted by...", "Co-authored-by...") in commit messages.
- **Documentation Style**: When writing or updating documentation, specs, or proposals, do NOT omit important details. Present them comprehensively but clearly, structurally (using lists/tables), and concisely (no fluff). For technical documentation, you must use the `docs-writer` skill.

## Toolchain and workspace

- Run `mise install` first. Versions are pinned: Node `24`, Go `1.25`, pnpm `11.5` (`.mise.toml`, `.nvmrc`, root `package.json`, `apps/backend/go.mod`).
- Use `pnpm` only. Workspace roots are `apps/*` and `packages/*` (`pnpm-workspace.yaml`), orchestrated by Turbo (`turbo.json`).

## Commands that matter

- Root dev flow: `pnpm run dev` (always runs `pnpm run infra:up` first, then `turbo run dev`).
- Root verification flow: `pnpm run lint && pnpm run test && pnpm run build`.
- Root deploy: `pnpm run deploy` (builds first, deploys only `@bowerbird/infra`).
- Backend targeted: `pnpm --filter @bowerbird/backend dev|lint|test|build|migrate:all`.
- Backend tests: always `pnpm --filter @bowerbird/backend test` (full `go test ./...`). Never verify with package-scoped or `-run` filtered `go test`.
- PWA targeted: `pnpm --filter @bowerbird/pwa dev|lint|test|build`.
- E2E targeted: `pnpm --filter @bowerbird/e2e lint|test:e2e|test:e2e:ui`.
- Infra targeted: `pnpm --filter @bowerbird/infra lint|test|build|synth|deploy`.

## Backend (`apps/backend`)

- API entrypoint is `cmd/api/main.go`; local `dev` uses Air (`.air.toml`) and sources `apps/backend/.env` if present.
- Background workers (`dev:relay`, `dev:events-consumer`, `dev:jobs-consumer`, `dev:scheduler`) use Air configs `.air.worker-*.toml` with the same reload behavior.
- Feature architecture: follow the `internal/<feature>` modular layout used in `internal/invoices`:
  - `domain/`: core business model and pure domain logic (no infra/framework concerns).
  - `application/`: use cases (`commands`, `queries`) and `ports` interfaces.
  - `contracts/`: cross-boundary payload contracts (feature jobs/events DTOs).
  - `adapters/`: HTTP, events/jobs handlers, repository and external provider implementations.
  - `wire.go`: feature composition root (instantiate adapters and inject into application commands/queries).
- Dependency rule for backend features: keep dependencies inward (`adapters -> application -> domain`), with contracts shared across boundaries; do not import adapters from `domain` or `application`.
- Error Handling & JSON:API: **Never** use `http.Error()`. Handlers must return `error` and be registered using `api.Wrap(handlerFunc, isDev)`.
- Domain Errors: Wrap or create errors using `appErrors.Wrap(err, appErrors.CodeX, "msg")`. `api.Wrap` automatically converts these to JSON:API payloads and injects `meta._debug` stack traces when `isDev` is true.
- Migrations CLI is `cmd/migrate/main.go`; keep migration sets split between `migrations/controlplane` and `migrations/tenant`.
- Runtime config (`internal/platform/config/config.go`):
  - `onprem` (local + client deploy): plain `.env` — `MINIO_ENDPOINT_URL`, `RABBITMQ_URL`, encryption keys, API keys.
  - `aws`: SSM SecureString at `SSM_PARAMETER_NAME` (shape in [docs/technical/deployment/ssm-secrets.md](../docs/technical/deployment/ssm-secrets.md)).
- Local dev object storage: MinIO in root `docker-compose.yml` (`:9000` API, `:9001` console). Bucket bootstrap: `apps/backend/scripts/init-minio.sh` via `pnpm run infra:up` (`minio-init` service).

## PWA (`apps/pwa`)

- Angular standalone + zoneless app. Wiring is in `src/app/app.config.ts`, routes in `src/app/app.routes.ts`.
- Serve command is fixed to `ng serve --host 0.0.0.0 --port 4200`; `angular.json` only allows host `app.bowerbird.dev`.
- Tenant routing: Tenant pages are children of the `/:tenantId` route and wrapped by `TenantLayoutComponent`.
- Tenant header is derived from the `tenantId` param via `core/interceptors/tenant.interceptor.ts`.
- Error Handling & UI Feedback: `error.interceptor.ts` globally handles JSON:API responses and logs `meta._debug` to the console.
  - **Toast (`ToastService`)**: Sonner-backed; `<hlm-toaster />` in `app.component.ts`. Use for global transient messages (5xx, network, success). The interceptor handles 5xx/network toasts automatically.
  - **Inline alerts**: Use `hlm-alert` within forms or pages for contextual 4xx validation errors.
- Auth refresh behavior is in `core/interceptors/auth.interceptor.ts` (401 -> refresh -> retry).
- Feature convention: keep business/data orchestration in `*/application/*store.ts`; keep `presentation` components thin.
- Shared cross-feature types: `ConnectionStatus` in `src/app/core/domain/connection-status.model.ts`; mail provider helpers in `src/app/inbox/domain/inbox.types.ts`.
- Shared presentation: `app-connection-status-chip` and `app-file-upload` under `src/app/core/presentation/components/`.
- `ThemeService` in `src/app/core/services/theme.service.ts` for dark-mode-aware embedded views.
- **Spartan UI**: Helm components in `src/app/shared/ui/`; icons via Lucide + `src/app/shared/icons/app-icons.ts`. See `docs/technical/frontend/spartan-ui.md`.

## E2E Testing (`apps/e2e`)

- Uses Playwright. Always run `pnpm run test:e2e:install` to ensure the local browser is present before running tests.
- Execution requires the local backend to be running (`pnpm run dev`) with `api.bowerbird.dev` accessible (Caddy routing).
- To test the full auth flow, the backend must be in `local` or `development` mode so the `/api/v1/auth/register-local` endpoint is enabled.
- UI doesn't have a signup form yet, so `test.fixture.ts` relies on the API `registerLocalOrFail` directly for setup.
- Commands from root: `pnpm run test:e2e` (headless), `pnpm run test:e2e:ui` (interactive).

## Local infra and deploy constraints

- `docker-compose.yml` runs Postgres `5432`, RabbitMQ `5672`, MinIO `9000/9001`, Caddy `80/443`.
- `Caddyfile` maps `app.bowerbird.dev -> :4200` and `api.bowerbird.dev -> :8080`; use these domains locally for cookie/routing behavior.
- Infra CDK entrypoint is `packages/infra/bin/index.ts` and requires `packages/infra/.env`:
  - `ENV` and `AWS_ACCOUNT_ID` must be set.
  - `AWS_REGION` must be `us-east-1` (enforced).
- Web deploy consumes `apps/pwa/dist/pwa/browser`; build PWA before infra deploy/synth checks that depend on assets.
- In `bowerbird-stack.ts`, S3 deployments use `prune: false` for assets/entrypoints; do not change casually.

## Hooks, formatting, and docs

- Pre-commit runs `pnpm lint-staged` then `pnpm run lint`, then optional `codegraph sync` (`.husky/pre-commit`).
- `lint-staged` applies Prettier to staged web/docs files and `gofmt -w` to staged Go files.
- Root `pnpm run format` only formats `*.{ts,tsx,md,mdx}`.
- Product/domain specs live under `.specs/features/*` in English.
- **Ubiquitous Language**: ALWAYS refer to `docs/domain/GLOSSARY.md` to map Spanish business terms (e.g., Factura, Adquirente) to their exact English counterparts for code. Keep Colombian e-invoicing acronyms (DIAN, CUFE, UBL) as-is.
