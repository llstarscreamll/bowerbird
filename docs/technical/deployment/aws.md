# AWS Lambda deployment (Pulumi)

Dual-runtime overview: [Runtime profiles](../architecture/runtime-profiles.md).

Pulumi program: `apps/deploy/aws/` (`@bowerbird/infra`). On-prem fleet is a
**separate** Pulumi project (`apps/deploy/onprem/`) — see
[On-prem fleet](./onprem.md) and [Deploy](../../../apps/deploy/README.md).

This stack deploys the **aws/lambda** target. Postgres runs on **Neon**, not
Amazon RDS. DNS is in **Cloudflare**. Application secrets live in **SSM
Parameter Store** (`SecureString`) under a customer-managed KMS key.

## Architecture

| Concern            | Service                                                                                                            |
| ------------------ | ------------------------------------------------------------------------------------------------------------------ |
| PWA                | Private S3 + CloudFront (OAC, TLS 1.2+, WAF)                                                                       |
| HTTP API           | CloudFront `/api*` → API Gateway DomainName (`api.*`, default endpoint off) + Go Lambda (`provided.al2023`, arm64) |
| Jobs               | SQS + Lambda, with a 14-day DLQ                                                                                    |
| Integration events | EventBridge custom bus (`source` prefix `bowerbird.`) + Lambda                                                     |
| Outbox relay       | EventBridge Scheduler `rate(1 minute)` → relay Lambda                                                              |
| Platform schedules | EventBridge Scheduler → scheduler Lambda (`outbox-sweeper`, `catalog-import-purge`, optional `inbox-sync-all`)     |
| Object storage     | Private S3 bucket (KMS), browser CORS for the app origin; presigns use the S3 REST endpoint                        |
| Postgres           | Neon project in `aws-us-east-1` (pooled URL for Lambdas, direct URL for migrations / `CREATE DATABASE`)            |
| Secrets            | SSM Parameter Store `SecureString` JSON, CMK                                                                       |
| DNS                | Cloudflare DNS-only CNAMEs: apex/`app.` → CloudFront; `api.` → API Gateway origin                                  |
| Observability      | CloudWatch logs (30/90 day retention), X-Ray, Lambda/SQS alarms, optional SNS email                                |

Lambdas are **not** in a VPC. Neon is reached over TLS on the public pooled
endpoint (PgBouncer). That avoids NAT Gateways and still keeps RDS out of the
design.

Outbox flow: API → Neon outbox → `outbox-relay` Lambda → EventBridge / SQS →
consumer Lambdas.

EventBridge Scheduler's minimum rate is one minute, so AWS relay ticks at
`1 minute` instead of the local 30s loop.

## Well-Architected mapping

| Pillar                 | How this stack applies it                                                                                                                                                                                                               |
| ---------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Operational excellence | Pulumi TypeScript, tagged resources, CloudWatch alarms, X-Ray                                                                                                                                                                           |
| Security               | CMK (rotation, CloudFront OAC on the web bucket), SSM SecureString, IAM per function, S3 Block Public Access, CloudFront + API Gateway WAF (IP reputation, Common, Known Bad Inputs, SQLi), TLS 1.2+, Cloudflare DNS validation for ACM |
| Reliability            | Multi-AZ CloudFront/API Gateway/Lambda, SQS DLQ, Lambda DLQ, EventBridge Scheduler DLQ, Neon HA + 7-day PITR on prod                                                                                                                    |
| Performance            | Lambda arm64, Neon pooler for bursty connections, CloudFront cache split (hashed vs entry)                                                                                                                                              |
| Cost                   | No NAT/RDS/RDS Proxy, Lambda + Neon scale-to-zero on non-prod (`suspendTimeoutSeconds`)                                                                                                                                                 |
| Sustainability         | Graviton Lambdas, serverless data plane                                                                                                                                                                                                 |

Account-level GuardDuty and CloudTrail stay outside this stack. Enable them on
the AWS account.

## Domains

Set these in the repo-root `.env`. Cloudflare must already host `ROOT_DOMAIN`.

| Variable               | Example         | DNS record                                                                                          |
| ---------------------- | --------------- | --------------------------------------------------------------------------------------------------- |
| `ROOT_DOMAIN`          | `money-path.co` | Apex CNAME (Cloudflare flattening) → CloudFront                                                     |
| `APP_SUBDOMAIN`        | `app`           | `app.` → CloudFront (PWA at `/`, API at `/api`)                                                     |
| `MEDIA_SUBDOMAIN`      | `media`         | ACM SAN only (not a CloudFront alias or DNS record). Browser uploads use S3 presign + bucket CORS   |
| `API_ORIGIN_SUBDOMAIN` | `api`           | Origin-only grey-cloud CNAME → API Gateway. Not a CloudFront alias. Browser traffic stays on `app.` |

Records are **DNS-only** (`proxied: false`) so CloudFront and ACM see the
hostname. Do not orange-cloud these names.

## Secrets

See [AWS secrets](./ssm-secrets.md). Pulumi writes the JSON blob, including
Neon `database_url` (pooler) and `database_direct_url` (direct). Lambdas
receive only `SSM_PARAMETER_NAME`. At cold start, `config.Load()` decrypts
the parameter and `platform.NewModule()` wires the infrastructure layer.

Generated once and stored in Pulumi state + Parameter Store:

- JWT access/refresh secrets
- Inbox and tenant encryption keys
- Messaging attestation secret

Pass `GEMINI_API_KEY` (required) and optional OAuth client IDs/secrets
through `.env` at deploy time. They are copied into the SecureString
parameter, not Lambda environment variables.

## Neon

Pulumi creates one Neon project per `ENV`:

- Region: `NEON_REGION_ID` (default `aws-us-east-1`)
- Database / role: `bowerbird`
- Default branch named after `ENV`
- History window: 7 days on prod (`604800`), 6 hours otherwise
- Pooled connection → Lambda `database_url`
- Direct connection → migrate Lambda and tenant `CREATE DATABASE`

Set `NEON_API_KEY` (and optional `NEON_ORG_ID`) in `.env`. The provider reads
the key from the Pulumi Neon provider config.

After the first `pulumi up`, control-plane migrations already ran as
part of that apply (see Deploy). Re-run them out of band with:

```bash
pnpm --filter @bowerbird/infra migrate
```

That invokes the migrate Lambda, which uses the **direct** Neon URL.

## Deploy

Use **`pnpm run deploy:aws`**. Root `pnpm run deploy` runs AWS **and** the
on-prem fleet in parallel.

1. Install the Pulumi CLI (`mise install` includes it) and log in
   (`pulumi login`).
2. Copy `.env.example` → `.env` and fill AWS, Cloudflare, Neon, and Gemini
   values. Do **not** deploy with the local MinIO dummy keys
   (`AWS_ACCESS_KEY_ID=bowerbird`). Pulumi and the AWS SDK read those
   names. Use an IAM role/profile, or a dedicated file:

   ```bash
   ENV_FILE=.env.aws pnpm run deploy:aws
   ```

   Omit `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` in that file so the
   SDK uses the shared credentials file or SSO.

3. Create or select the stack named after `ENV`:

   ```bash
   cd apps/deploy/aws
   pulumi stack init "$ENV"   # first time only
   pulumi stack select "$ENV"
   ```

4. Build and deploy from the repo root:

   ```bash
   pnpm run deploy:aws
   ```

   That runs `pnpm run build` (PWA assets + Go Lambda zips) then
   `pulumi up --yes` for `@bowerbird/infra`. Preview without applying:

   ```bash
   pnpm --filter @bowerbird/infra synth
   ```

   When the migrate Lambda package changes (control-plane or tenant SQL
   lives in that zip), Pulumi updates it, invokes it, and only then
   publishes the other Lambdas and web objects. If the invoke fails, the
   apply stops and application artifacts stay on the previous version.

5. Confirm ACM DNS records in Cloudflare and `pulumi stack output`
   (`webUrl`, `apiUrl`, `ssmParameterName`, `neonProjectId`,
   `jobsQueueUrl`, `migrateFunctionName`).

## Schedules

EventBridge Scheduler (not EventBridge rules). Unix crontab on-prem is
5-field; AWS cron is 6-field with `?`.

| Name                   | Expression          | Target           | When                                       |
| ---------------------- | ------------------- | ---------------- | ------------------------------------------ |
| `outbox-relay`         | `rate(1 minute)`    | relay Lambda     | Always                                     |
| `outbox-sweeper`       | `rate(1 hour)`      | scheduler Lambda | Always                                     |
| `catalog-import-purge` | `cron(0 5 * * ? *)` | scheduler Lambda | Always (05:00 UTC)                         |
| `inbox-sync-all`       | `rate(5 minutes)`   | scheduler Lambda | Google or Microsoft OAuth client id+secret |

## Constraints

- `AWS_REGION` must be `us-east-1` (CloudFront ACM + CloudFront WAF).
- `ENV`, `AWS_ACCOUNT_ID`, `ROOT_DOMAIN`, `CLOUDFLARE_API_TOKEN`,
  `NEON_API_KEY`, and `GEMINI_API_KEY` are required.
- Optional: `APP_SUBDOMAIN` (default `app`), `MEDIA_SUBDOMAIN` (default
  `media`), `API_ORIGIN_SUBDOMAIN` (default `api`), `NEON_ORG_ID`,
  `NEON_REGION_ID` (default `aws-us-east-1`),
  `NEON_PG_VERSION` (default `16`), `ALARM_EMAIL`, `GEMINI_MODEL`,
  `GEMINI_ENDPOINT`, Google/Microsoft OAuth client ids and secrets.
- Web assets come from `apps/pwa/dist/pwa/browser` (the root build
  produces this before Pulumi runs).
- S3 web deploy does not prune hashed bundles, so old clients can still load
  previous chunks.
- Cloudflare API token needs Zone Read + DNS Edit on `ROOT_DOMAIN`.
- OAuth redirect URIs at the identity provider must use the app host
  (`https://app.<ROOT_DOMAIN>/api/v1/auth/.../callback`), not a
  separate `api.` hostname.

## CloudFront / cache

- `/api*` → API Gateway custom domain (`api.<ROOT_DOMAIN>`), cache disabled.
  `execute-api` endpoint is disabled. CloudFront sends `X-Origin-Verify`;
  the regional WAF blocks requests without it. Origin request policy
  `AllViewerExceptHostHeader` so API Gateway sees the origin `Host`.
- Other paths → S3 (PWA). A CloudFront Function rewrites extensionless
  SPA routes to `/index.html`. Do not use distribution-wide 403/404
  custom error pages: they would rewrite API 404s into the SPA shell.
- Hashed assets: `Cache-Control: public, max-age=31536000, immutable`
- Entry points (`index.html`, `ngsw*`, manifest):
  `max-age=0, must-revalidate, s-maxage=300`. There is no CloudFront
  invalidation; a new `index.html` can take up to five minutes to appear
  at the edge.
- WAF: CloudFront (scope `CLOUDFRONT`) and API Gateway stage (scope
  `REGIONAL`). Managed rule groups: Amazon IP Reputation, Common,
  Known Bad Inputs, SQLi.
