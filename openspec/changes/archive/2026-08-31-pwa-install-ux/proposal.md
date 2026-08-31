## Why

El banner fijo de instalación PWA en `app.component.ts` permanece visible sin opción de cerrar, interrumpe el flujo de trabajo y no sigue las prácticas actuales (engagement gate, dismiss con cooldown, pull model). Los usuarios recurrentes de Bowerbird se benefician de instalar la app, pero la promoción debe ser contextual, respetuosa y medible.

## What Changes

- Extraer orquestador de **system notices** a paquete workspace `@bowerbird/system-notices` (`packages/system-notices/`).
- Sub-unidades en PWA: `core/analytics/` (shared kernel), `core/pwa-install/` (runtime, engagement, notices concretas).
- Orquestador con prioridad, scope (`global` | `tenant`) y una notice visible a la vez — consumido vía dependency del paquete.
- Promoción automática solo dentro del **tenant layout** (`/:tenantId/*`), nunca en lobby/login.
- Gate de engagement: **2ª visita dentro de 7 días** — aggregate `InstallEngagement` (VOs + intent methods + `reconstitute`); notices como adapters sin reglas de dominio.
- **Desktop**: snackbar Sonner (6 s). **Mobile**: bottom sheet (`HlmSheet`). Viewport vía `ViewportPort` propio (no acoplado al sidebar).
- **iOS Safari**: sheet instructivo; sin `beforeinstallprompt`.
- Cooldowns progresivos; pull model en menú usuario (delega a `PwaInstallCoordinator`).
- Registro de notices en **composition root único** (`app.config.ts`) vía token `SYSTEM_NOTICE` (`multi: true`).

- `AnalyticsPort` con adapter consola (fase 1); backend en fase 2.

## Non-goals

- Endpoint backend de analytics (fase 2).
- Publicar `@bowerbird/system-notices` fuera del monorepo (npm registry público).
- Extraer `core/analytics/` a package (solo un consumer en fase 1).
- Acoplar install-promotion a `HlmSidebarService`.
- Lógica de elegibilidad en layouts o `app.component`.
- Cambios a `manifest.webmanifest`.
- Promoción en lobby/login.
- A/B de copy.

## Capabilities

### New Capabilities

- `system-notices`: paquete `@bowerbird/system-notices` — orquestador reutilizable, port `SystemNotice`, host Angular.
- `pwa-install`: promoción PWA, engagement, notices concretas (consumer), iOS guide, funnel analytics (cliente).

### Modified Capabilities

_(ninguna)_

## Impact

| Área    | Módulos                                                                                                     |
| ------- | ----------------------------------------------------------------------------------------------------------- |
| Package | `packages/system-notices/` → `@bowerbird/system-notices` (orchestrator, port, host, `provideSystemNotices`) |
| PWA     | `core/analytics/` (nuevo — `AnalyticsPort` + adapter consola)                                               |
| PWA     | `core/pwa-install/` (nuevo — domain, application, infrastructure, presentation, notices concretas)          |
| PWA     | `package.json` — dependency `@bowerbird/system-notices`                                                     |
| PWA     | `app.config.ts` (registro providers y notices)                                                              |
| PWA     | `app.component.ts` (host `scope="global"`; quitar cards)                                                    |
| PWA     | `tenant-layout/` (host `scope="tenant"` + menú pull vía coordinator)                                        |
| PWA     | `core/services/pwa.service.ts` (deprecar / migrar a `pwa-install`)                                          |
| Turbo   | `turbo.json` — task `build`/`lint`/`test` del nuevo package                                                 |
| Backend | Sin cambios (fase 1)                                                                                        |
