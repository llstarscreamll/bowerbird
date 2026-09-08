# Deploy

AWS SaaS and on-prem client VMs are **parallel** release tracks. A
version can go to Lambda (one AWS stack) and, independently, to a pool
of client IPs. Turbo runs both `deploy` tasks at the same time.

| Track             | Path      | Tooling          | Package             | Root command             |
| ----------------- | --------- | ---------------- | ------------------- | ------------------------ |
| AWS Lambda (SaaS) | `aws/`    | Pulumi           | `@bowerbird/infra`  | `pnpm run deploy:aws`    |
| On-prem fleet     | `onprem/` | Pulumi + Compose | `@bowerbird/onprem` | `pnpm run deploy:onprem` |

`pnpm run deploy` runs **both** tracks. If
`apps/deploy/onprem/hosts.json` is missing or empty, the on-prem track
skips.

Backend and PWA live in `apps/backend` and `apps/pwa`.

## AWS

Pulumi stack: Lambda, API Gateway, CloudFront, Neon, Cloudflare DNS, SSM.

```bash
pnpm run deploy:aws
```

First-time stack: `cd apps/deploy/aws && pulumi stack init "$ENV"`.

Details: [AWS deploy](../../docs/technical/deployment/aws.md).

## On-prem

Compose is the payload on each VM. Pulumi fans that payload out to the
host inventory (SSH, load images, `docker compose up`). Hosts apply in
parallel. Removing a host from the inventory does **not** tear down that
VM.

```bash
cp apps/deploy/onprem/hosts.example.json apps/deploy/onprem/hosts.json
# set ONPREM_RELEASE, ONPREM_SSH_KEY_PATH in the repo-root .env
pnpm run deploy:onprem
```

Each VM needs Docker and `apps/deploy/onprem/.env` **before** the first
fleet apply. Bootstrap one machine with Compose; after that, fleet
releases ship new image tags.

Details: [On-prem fleet](../../docs/technical/deployment/onprem.md) and
[On-prem stack](../../docs/technical/architecture/onprem-runtime.md).
