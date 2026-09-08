# On-prem deployment

Compose payload on each client VM. Fleet apply (pool of IPs) is Pulumi
in this folder — [`../README.md`](../README.md),
[fleet docs](../../../docs/technical/deployment/onprem.md). AWS is a
parallel track in [`../aws/`](../aws/).

## Bootstrap (one VM)

```bash
pnpm --filter @bowerbird/pwa build
cp .env.example .env   # edit secrets before production
docker compose config
docker compose up -d --build
```

Build the PWA before composing Caddy. The Caddy image copies
`apps/pwa/dist/pwa/browser` to `/srv`.

## Fleet release (many VMs)

```bash
cp hosts.example.json hosts.json
# ONPREM_RELEASE + ONPREM_SSH_KEY_PATH in repo-root .env
pnpm run deploy:onprem
```

Does not overwrite remote `.env`.

Services: `caddy`, `api`, `outbox-relay`, `events-consumer`,
`jobs-consumer`, `scheduler`, `postgres`, `rabbitmq`, `minio`.

Health: `docker compose ps` — all services should report healthy after
`migrate` completes.
