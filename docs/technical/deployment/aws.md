# Despliegue AWS (CDK)

## Stack desplegado

- Frontend Angular en S3 privado + CloudFront.
- Backend HTTP en API Gateway + Lambda Go.
- Lambdas para SQS y EventBridge.
- Route53 para dominios de app y API.

## Dominios

- App: `${APP_SUBDOMAIN}.${ROOT_DOMAIN}` (por defecto `app.money-path.co`)
- API: `${API_SUBDOMAIN}.${ROOT_DOMAIN}` (por defecto `api.money-path.co`)

## Flujo de deploy

Desde la raíz:

```bash
pnpm run build
cd packages/infra
pnpm exec cdk bootstrap aws://$AWS_ACCOUNT_ID/$AWS_REGION
pnpm exec cdk deploy --all --require-approval never
```

## Restricciones importantes

- `AWS_REGION` debe ser `us-east-1` (validado en `packages/infra/bin/infra.ts`).
- `packages/infra/.env` es obligatorio.
- El stack espera build web en `apps/pwa/dist/web/browser`.

## CloudFront + estrategia de cache SPA/PWA

- Fallback SPA: `403/404 -> /index.html`.
- Routing de `https://<app-domain>/api/*` hacia `https://<api-domain>`.
- Assets hash (`*.js`, `*.css`, etc.): `Cache-Control: public, max-age=31536000, immutable`.
- Entry points (`index.html`, `ngsw.json`, `ngsw-worker.js`, `safety-worker.js`, `manifest.webmanifest`):
  `Cache-Control: public, max-age=0, must-revalidate, s-maxage=300`.
- `prune: false` para mantener bundles antiguos y evitar que `index.html` viejos rompan por assets inexistentes.
- Invalidacion selectiva solo de entry points en cada deploy.
