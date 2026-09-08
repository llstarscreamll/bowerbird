# AWS Lambda deployment (Pulumi)

Dual-runtime overview: [Runtime profiles](../architecture/runtime-profiles.md).

This stack deploys the **aws/lambda** target. Postgres runs on **Neon**, not
Amazon RDS. DNS is in **Cloudflare**. Application secrets live in **AWS
Secrets Manager** under a customer-managed KMS key.

## Architecture

| Concern            | Service                                                                                                 |
| ------------------ | ------------------------------------------------------------------------------------------------------- |
| PWA                | Private S3 + CloudFront (OAC, TLS 1.2+, WAF)                                                            |
| HTTP API           | API Gateway HTTP API + Go Lambda (`provided.al2023`, arm64)                                             |
| Jobs               | SQS + Lambda, with a 14-day DLQ                                                                         |
| Integration events | EventBridge custom bus (`source` prefix `bowerbird.`) + Lambda                                          |
| Outbox relay       | EventBridge Scheduler `rate(1 minute)` → relay Lambda                                                   |
| Platform schedules | EventBridge Scheduler → scheduler Lambda (`outbox-sweeper`, optional `inbox-sync-all`)                  |
| Object storage     | Private S3 bucket (KMS), browser CORS for the app origin                                                |
| Postgres           | Neon project in `aws-us-east-1` (pooled URL for Lambdas, direct URL for migrations / `CREATE DATABASE`) |
| Secrets            | Secrets Manager JSON, CMK, rotation-ready                                                               |
| DNS                | Cloudflare DNS-only (grey cloud) CNAMEs to CloudFront and API Gateway                                   |
| Observability      | CloudWatch logs (30/90 day retention), X-Ray, Lambda/SQS alarms, optional SNS email                     |

Lambdas are **not** in a VPC. Neon is reached over TLS on the public pooled
endpoint (PgBouncer). That avoids NAT Gateways and still keeps RDS out of the
design.

Outbox flow: API → Neon outbox → `outbox-relay` Lambda → EventBridge / SQS →
consumer Lambdas.

EventBridge Scheduler's minimum rate is one minute, so AWS relay ticks at
`1 minute` instead of the local 30s loop.

## Well-Architected mapping

| Pillar                 | How this stack applies it                                                                                                                                 |
| ---------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Operational excellence | Pulumi TypeScript, tagged resources, CloudWatch alarms, X-Ray                                                                                             |
| Security               | CMK, Secrets Manager (no secrets in Lambda env), IAM per function, S3 Block Public Access, WAF managed rules, TLS 1.2+, Cloudflare DNS validation for ACM |
| Reliability            | Multi-AZ CloudFront/API Gateway/Lambda, SQS DLQ, Lambda DLQ, Neon HA + PITR history retention                                                             |
| Performance            | Lambda arm64, Neon pooler for bursty connections, CloudFront cache split (hashed vs entry)                                                                |
| Cost                   | No NAT/RDS/RDS Proxy, Lambda + Neon scale-to-zero on non-prod (`suspendTimeoutSeconds`)                                                                   |
| Sustainability         | Graviton Lambdas, serverless data plane                                                                                                                   |

Account-level GuardDuty and CloudTrail stay outside this stack. Enable them on
the AWS account.

## Domains

Set these in the repo-root `.env`. Cloudflare must already host `ROOT_DOMAIN`.

| Variable          | Example         | DNS record                                                                   |
| ----------------- | --------------- | ---------------------------------------------------------------------------- |
| `ROOT_DOMAIN`     | `money-path.co` | Apex CNAME (Cloudflare flattening) → CloudFront                              |
| `APP_SUBDOMAIN`   | `app`           | `app.` → CloudFront                                                          |
| `API_SUBDOMAIN`   | `api`           | `api.` → API Gateway custom domain                                           |
| `MEDIA_SUBDOMAIN` | `media`         | Covered by the ACM certificate; browser uploads use S3 presign + bucket CORS |

Records are **DNS-only** (`proxied: false`) so CloudFront and ACM see the
hostname. Do not orange-cloud these names.

## Secrets

See [AWS secrets](./ssm-secrets.md). Pulumi writes the JSON blob, including
Neon `database_url` (pooler) and `database_direct_url` (direct). Lambdas
receive only `SECRET_ARN`.

Generated once and stored in Pulumi state + Secrets Manager:

- JWT access/refresh secrets
- Inbox and tenant encryption keys
- Messaging attestation secret

Pass `GEMINI_API_KEY` (required) and optional OAuth client IDs/secrets through
`.env` at deploy time. They are copied into Secrets Manager, not Lambda
environment variables.

## Neon

Pulumi creates one Neon project per `ENV`:

- Region: `NEON_REGION_ID` (default `aws-us-east-1`)
- Database / role: `bowerbird`
- Default branch named after `ENV`
- Pooled connection → Lambda `database_url`
- Direct connection → migrate Lambda and tenant `CREATE DATABASE`

Set `NEON_API_KEY` (and optional `NEON_ORG_ID`) in `.env`. The provider reads
the key from the Pulumi Neon provider config.

After the first `pulumi up`, run control-plane migrations:

```bash
pnpm --filter @bowerbird/infra migrate
```

That invokes the migrate Lambda, which uses the **direct** Neon URL.

## Deploy

1. Install the Pulumi CLI (`mise install` includes it) and log in
   (`pulumi login`).
2. Copy `.env.example` → `.env` and fill AWS, Cloudflare, Neon, and Gemini
   values.
3. Create or select the stack named after `ENV`:

   ```bash
   cd packages/infra
   pulumi stack init "$ENV"   # first time only
   pulumi stack select "$ENV"
   ```

4. Build and deploy from the repo root:

   ```bash
   pnpm run build
   pnpm run deploy
   ```

   Root `pnpm run deploy` builds all packages (including PWA assets and Go
   Lambda zips) then runs `pulumi up --yes` for `@bowerbird/infra`.

5. Run `pnpm --filter @bowerbird/infra migrate`.
6. Confirm ACM DNS records in Cloudflare and `pulumi stack output`.

## Constraints

- `AWS_REGION` must be `us-east-1` (CloudFront ACM + CloudFront WAF).
- `ENV`, `AWS_ACCOUNT_ID`, `ROOT_DOMAIN`, `CLOUDFLARE_API_TOKEN`,
  `NEON_API_KEY`, and `GEMINI_API_KEY` are required.
- Web assets come from `apps/pwa/dist/pwa/browser` (build the PWA first).
- S3 web deploy does not prune hashed bundles, so old clients can still load
  previous chunks.
- Cloudflare API token needs Zone Read + DNS Edit on `ROOT_DOMAIN`.

## CloudFront / cache

- SPA fallback: `403/404` → `/index.html`
- Hashed assets: long-lived immutable cache
- Entry points (`index.html`, `ngsw*`, manifest): short / must-revalidate
- Invalidate entry points on deploy
