# On-prem deployment

```bash
cp .env.example .env   # edit secrets before production
docker compose config
docker compose up -d
```

Services: `caddy`, `api`, `outbox-relay`, `events-consumer`, `jobs-consumer`, `postgres`, `rabbitmq`, `minio`.

Health: `docker compose ps` — all services should report healthy after `migrate` completes.
