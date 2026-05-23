# Arquitectura frontend (Angular + PWA)

## Stack

- Angular standalone + signals
- TailwindCSS
- PWA con Service Worker de Angular

## Capacidades técnicas implementadas

- SPA con rutas standalone.
- Vista principal con estado de backend consumiendo `/api/health`.
- Proxy local de `/api` a backend Go en dev.
- Instalacion PWA (manifest + iconos + prompt de instalación).
- Actualizacion de versiones con `SwUpdate` y acción de refresh.

## Configuracion clave

- Angular app config: `apps/web/src/app/app.config.ts`
- Service worker build: `apps/web/angular.json`
- Config cache SW: `apps/web/ngsw-config.json`
- Manifest: `apps/web/public/manifest.webmanifest`
- Servicio PWA: `apps/web/src/app/core/services/pwa.service.ts`

## Verificacion local de PWA

```bash
pnpm --filter @bowerbird/web build
pnpm --filter @bowerbird/web preview:pwa
```

Luego abrir `http://localhost:4300` y validar en DevTools > Application.
