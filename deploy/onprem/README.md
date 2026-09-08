# On-prem deployment

```bash
pnpm --filter @bowerbird/pwa build
cp .env.example .env   # edit secrets before production
docker compose config
docker compose up -d --build
```

Build the PWA before composing Caddy. The Caddy image copies
`apps/pwa/dist/pwa/browser` to `/srv`.

Caddy serves the PWA at `/` and proxies `/api*` to the
API container.

Services: `caddy`, `api`, `outbox-relay`, `events-consumer`,
`jobs-consumer`, `scheduler`, `postgres`, `rabbitmq`, `minio`.

Health: `docker compose ps` — all services should report healthy after
`migrate` completes.
