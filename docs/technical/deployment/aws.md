# AWS deployment (CDK)

Dual-runtime overview: [Runtime profiles](../architecture/runtime-profiles.md).

## Stack

- PWA: private S3 + CloudFront
- HTTP API: API Gateway + Go Lambda
- Workers: SQS and EventBridge Lambdas + **outbox-relay** (EventBridge schedule, 30s)
- DNS: Route53 for app and API hosts

Outbox flow: API → RDS outbox → `outbox-relay` Lambda → EventBridge / SQS → consumer Lambdas.

### Future: relay on Fargate

Replace the scheduled `outbox-relay` Lambda with a long-running Fargate task executing `worker relay` for lower latency and simpler batching. EventBridge/SQS consumers and broker contracts stay unchanged.

## Domains

- App: `${APP_SUBDOMAIN}.${ROOT_DOMAIN}` (default `app.money-path.co`)
- API: `${API_SUBDOMAIN}.${ROOT_DOMAIN}` (default `api.money-path.co`)

## Deploy

```bash
pnpm run build
cd packages/infra
pnpm exec cdk bootstrap aws://$AWS_ACCOUNT_ID/$AWS_REGION
pnpm exec cdk deploy --all --require-approval never
```

Or from root: `pnpm run deploy`.

## Constraints

- `AWS_REGION` must be `us-east-1`.
- `packages/infra/.env` required (`ENV`, `AWS_ACCOUNT_ID`, …).
- Web assets from `apps/pwa/dist/pwa/browser` (build PWA first).
- Application secrets: SSM SecureString JSON — see [SSM secrets JSON](../deployment/ssm-secrets.md).

## CloudFront / cache

- SPA fallback: `403/404` → `/index.html`
- `/api/*` on app domain routes to API origin
- Hashed assets: long-lived immutable cache
- Entry points (`index.html`, `ngsw*`, manifest): short / must-revalidate
- S3 deploy `prune: false` so old bundles stay available
- Invalidate entry points only on deploy
