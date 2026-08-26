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

### API (local)

1. Copy `apps/backend/.env.example` → `apps/backend/.env`.
2. Copy `apps/backend/secrets.example.json` → `apps/backend/secrets.json`.

| Source         | Use for                                                                                                                                                     |
| -------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `.env`         | Process bootstrap: `PORT`, AWS IAM keys, `AWS_REGION`, `AWS_ENDPOINT_URL`, `S3_PRESIGN_ENDPOINT_URL`, `SSM_PARAMETER_NAME`, `ENABLE_LOCAL_EVENT_LOOP`       |
| `secrets.json` | Business secrets and AWS resource names: `database_url`, buckets, queue URLs, API keys, `inbox_credentials_encryption_key`, `tenant_secrets_encryption_key` |

The API loads config from the SSM parameter named in `.env`. LocalStack injects `secrets.json` into SSM on start.

Typical local `.env` values:

- `AWS_ENDPOINT_URL=http://localhost:4566`
- `S3_PRESIGN_ENDPOINT_URL=https://media.bowerbird.dev`
- `ENABLE_LOCAL_EVENT_LOOP=true`
- `SSM_PARAMETER_NAME=/bowerbird/local/secrets`

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
- `media.bowerbird.dev` → LocalStack S3 `:4566`

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

Starts Postgres, Redis, LocalStack, Caddy, Go API (Air), and Angular. Prefer the `*.bowerbird.dev` hosts (cookies/routing).

- App: `https://app.bowerbird.dev`
- Tenant example: `https://app.bowerbird.dev/acme/dashboard`
- API: `https://api.bowerbird.dev`
- Media: `https://media.bowerbird.dev/bowerbird-local-bucket/<key>`

`infra:up` / `dev` wait on healthchecks (Postgres, Redis, SSM `/bowerbird/local/secrets`, Caddy 80/443).

## Commands

`setup:local` · `infra:up` · `infra:down` · `build` · `test` · `lint` · `format` ·
`format:check` · `deploy`

Also: [Development quality](./quality/development-quality.md) ·
[CodeGraph](./tooling/codegraph.md) · [LocalStack](./tooling/localstack.md)
