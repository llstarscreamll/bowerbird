# On-prem fleet deployment (Pulumi)

AWS SaaS (`apps/deploy/aws/`) and on-prem client VMs (`apps/deploy/onprem/`) are
parallel release tracks. They do not share Pulumi state. `pnpm run
deploy` runs both; use `deploy:aws` or `deploy:onprem` to run one.

Dual-runtime adapters: [Runtime profiles](../architecture/runtime-profiles.md).
VM service list: [On-prem stack](../architecture/onprem-runtime.md).

## What Pulumi does

1. Build app and Caddy images on the operator machine, tagged
   `bowerbird-onprem-app:$ONPREM_RELEASE` and
   `bowerbird-onprem-caddy:$ONPREM_RELEASE`.
2. For **each** host in the inventory, in parallel:
   - Copy `docker-compose.yml` and `Caddyfile` over SSH.
   - `docker save | docker load` those two images.
   - Run migrations, then `docker compose up -d --no-build`.

Postgres, RabbitMQ, and MinIO stay as upstream images on the VM.
Pulumi does **not** overwrite `apps/deploy/onprem/.env` on the host
(per-client secrets).

Removing a host from the inventory is a no-op on that VM
(`retainOnDelete`).

## Inventory

Copy `apps/deploy/onprem/hosts.example.json` to `apps/deploy/onprem/hosts.json`
(gitignored). Override the path with `ONPREM_HOSTS_FILE`.

```json
[
  {
    "id": "acme",
    "address": "203.0.113.10",
    "user": "bowerbird",
    "port": 22,
    "remoteDir": "/opt/bowerbird"
  }
]
```

`id` must match `^[a-z0-9][a-z0-9-]*$`. Defaults: `user=bowerbird`,
`port=22`, `remoteDir=/opt/bowerbird`.

An empty or missing inventory makes `pnpm run deploy:onprem` skip
(exit 0) so AWS-only applies still work.

## Operator environment (repo-root `.env`)

| Variable              | Required when inventory is non-empty | Purpose                                                  |
| --------------------- | ------------------------------------ | -------------------------------------------------------- |
| `ONPREM_RELEASE`      | Yes                                  | Image tag (git sha or version)                           |
| `ONPREM_SSH_KEY_PATH` | Yes                                  | SSH private key for every host                           |
| `ONPREM_HOSTS_FILE`   | No                                   | Inventory path (default `apps/deploy/onprem/hosts.json`) |

## Bootstrap (once per VM)

Do this before the first fleet apply:

1. Install Docker Engine and Compose v2. Open SSH for the deploy key.
2. Create `remoteDir/apps/deploy/onprem/.env` from
   `apps/deploy/onprem/.env.example` with **that client's** secrets.
3. Optional: run Compose once by hand to pull Postgres/RabbitMQ/MinIO.

The fleet script fails if `.env` is missing on the host.

## Release

```bash
export ONPREM_RELEASE="$(git rev-parse --short HEAD)"
export ONPREM_SSH_KEY_PATH="$HOME/.ssh/bowerbird-onprem"
pnpm run deploy:onprem
```

Or set those in the repo-root `.env` and run `pnpm run deploy` to ship
AWS and the fleet together.

First Pulumi stack: created automatically as `fleet` in project
`bowerbird-onprem` (`cd apps/deploy/onprem`).

## Failure behavior

Independent host resources apply in parallel. If one SSH/load/up fails,
`pulumi up` exits non-zero. Hosts that already completed keep the new
images. Re-run the same `ONPREM_RELEASE` to retry the failed hosts.

## Commands

```bash
pnpm --filter @bowerbird/onprem lint
pnpm --filter @bowerbird/onprem test
pnpm --filter @bowerbird/onprem synth
pnpm run deploy:onprem
```
