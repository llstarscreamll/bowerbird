# Getting started

## Requirements

- mise
- Docker
- Caddy (via Docker Compose)
- AWS CLI (deploy only)

Pinned in `.mise.toml`: Node `24`, Go `1.25`, pnpm `11.5`, Air `latest`.

## Setup

`mise` must already be on your `PATH`. Then run the local setup script (idempotent):

```bash
pnpm run setup:local
```

If `pnpm` is not available yet, invoke the script directly:

```bash
./scripts/setup-local.sh
```

`setup:local` / `scripts/setup-local.sh`:

1. Installs the mise toolchain from `.mise.toml` (`node`, `go`, `pnpm`, `air`).
2. Runs `pnpm install` for the workspace.
3. Installs agent skills and MCP CLIs documented in
   [Development quality](./quality/development-quality.md).
4. Verifies project MCP registration files (`.cursor/mcp.json`, `opencode.json`).

After setup, copy env/secrets as described below, then start infra/dev.

Optional: `mise x -- pnpm run dev` to use the repo toolchain without changing
globals.

## Environment

### API (local — onprem profile)

1. Copy `apps/backend/.env.example` → `apps/backend/.env`.
2. Set `DEPLOYMENT_TARGET=onprem` and `RABBITMQ_URL=amqp://bowerbird:bowerbird@localhost:5672/`.
3. Provide secrets in `.env` (`GEMINI_API_KEY`, `INBOX_CREDENTIALS_ENCRYPTION_KEY`, `TENANT_SECRETS_ENCRYPTION_KEY`, `DATABASE_URL`, `S3_BUCKET_NAME`).

| Source | Use for                                                                     |
| ------ | --------------------------------------------------------------------------- |
| `.env` | All config for local/onprem: DB, RabbitMQ, MinIO, encryption keys, API keys |

Typical local `.env` values:

- `DEPLOYMENT_TARGET=onprem`
- `RABBITMQ_URL=amqp://bowerbird:bowerbird@localhost:5672/`
- `MINIO_ENDPOINT_URL=http://localhost:9000`
- `AWS_ACCESS_KEY_ID=bowerbird` / `AWS_SECRET_ACCESS_KEY=bowerbirdsecret`
- `S3_BUCKET_NAME=bowerbird-local-bucket`
- `S3_PRESIGN_ENDPOINT_URL=https://media.bowerbird.dev`
- `AWS_REQUEST_CHECKSUM_CALCULATION=when_required` / `AWS_RESPONSE_CHECKSUM_VALIDATION=when_required` (MinIO SDK compatibility)

For raw Go commands: `export $(grep -v '^#' apps/backend/.env | xargs)`.

### CDK

1. Copy `packages/infra/.env.example` → `packages/infra/.env`.
2. Set `AWS_ACCOUNT_ID`, `AWS_REGION=us-east-1`, `ROOT_DOMAIN`, `APP_SUBDOMAIN`, `API_SUBDOMAIN`.

## Local DNS and HTTPS

Add to `/etc/hosts`:

```text
127.0.0.1   api.bowerbird.dev
127.0.0.1   app.bowerbird.dev
127.0.0.1   media.bowerbird.dev
```

Caddy (Compose) uses `network_mode: host` and proxies:

- `app.bowerbird.dev` → Angular `:4200`
- `api.bowerbird.dev` → Go API `:8080`
- `media.bowerbird.dev` → MinIO `:9000`

Host networking is required on Linux so Caddy can reach those host ports. A bridged `host.docker.internal` hop is dropped by UFW/nftables and the browser shows **502**.

### Trust the local CA

**macOS:**

```bash
docker cp bowerbird-caddy:/data/caddy/pki/authorities/local/root.crt ./bowerbird-local-ca.crt
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain ./bowerbird-local-ca.crt
```

**Arch Linux:**

```bash
docker cp bowerbird-caddy:/data/caddy/pki/authorities/local/root.crt ./bowerbird-local-ca.crt
sudo trust anchor --store ./bowerbird-local-ca.crt
```

Equivalent with the CA bundle files:

```bash
sudo cp ./bowerbird-local-ca.crt /etc/ca-certificates/trust-source/anchors/
sudo update-ca-trust
```

**Chromium / Chrome on Arch:** also import into the NSS DB (Chromium does not always use the system store):

```bash
sudo pacman -S --needed nss
mkdir -p ~/.pki/nssdb
certutil -d sql:$HOME/.pki/nssdb -N --empty-password || true
certutil -d sql:$HOME/.pki/nssdb -D -n "Caddy Local Authority - ECC Root" || true
certutil -d sql:$HOME/.pki/nssdb -A -n "Caddy Local Authority - ECC Root" -t "C,," -i ./bowerbird-local-ca.crt
```

Restart the browser after importing.

**Fedora:**

```bash
docker cp bowerbird-caddy:/data/caddy/pki/authorities/local/root.crt ./bowerbird-local-ca.crt
sudo cp ./bowerbird-local-ca.crt /etc/pki/ca-trust/source/anchors/
sudo update-ca-trust
```

**Chrome Flatpak on Fedora:** also import into the Flatpak NSS DB:

```bash
sudo dnf install -y nss-tools
docker cp bowerbird-caddy:/data/caddy/pki/authorities/local/root.crt /tmp/caddy-root.crt
mkdir -p ~/.var/app/com.google.Chrome/.pki/nssdb
certutil -d sql:$HOME/.var/app/com.google.Chrome/.pki/nssdb -N --empty-password || true
certutil -d sql:$HOME/.var/app/com.google.Chrome/.pki/nssdb -D -n "Caddy Local Authority - ECC Root" || true
certutil -d sql:$HOME/.var/app/com.google.Chrome/.pki/nssdb -A -n "Caddy Local Authority - ECC Root" -t "C,," -i /tmp/caddy-root.crt
flatpak kill com.google.Chrome && flatpak run com.google.Chrome
```

If volumes were wiped, Caddy may regenerate the CA — re-export and re-trust. Firefox: import the cert under Settings → Certificates → Authorities.

## Dev

```bash
pnpm run dev
```

Starts Postgres, RabbitMQ, MinIO, Caddy, Go API (Air), PWA, and backend workers (`relay`, `events-consumer`, `jobs-consumer`, `scheduler`). All backend processes use **Air** hot reload. Prefer `*.bowerbird.dev` hosts (cookies/routing).

Manual workers (if needed):

```bash
pnpm --filter @bowerbird/backend dev:relay
pnpm --filter @bowerbird/backend dev:events-consumer
pnpm --filter @bowerbird/backend dev:jobs-consumer
pnpm --filter @bowerbird/backend dev:scheduler
```

See [Runtime profiles](./architecture/runtime-profiles.md) and [Outbox relay](./architecture/outbox-relay.md).

- App: `https://app.bowerbird.dev`
- Tenant example: `https://app.bowerbird.dev/acme/dashboard`
- API: `https://api.bowerbird.dev`
- Media: `https://media.bowerbird.dev/bowerbird-local-bucket/<key>`

`infra:up` / `dev` wait on healthchecks (Postgres, RabbitMQ, MinIO, Caddy 80/443) and bootstrap the MinIO bucket. Orphan containers (e.g. old LocalStack) are removed automatically.

## Commands

`setup:local` · `infra:up` · `infra:down` · `build` · `test` · `lint` · `format` ·
`format:check` · `deploy`

Also: [Development quality](./quality/development-quality.md) ·
[CodeGraph](./tooling/codegraph.md) · [MinIO](./tooling/minio.md)
