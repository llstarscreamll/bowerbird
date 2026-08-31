## Why

El change `pwa-install-ux` introdujo `core/analytics/` con `AnalyticsPort`, domain events de engagement traducidos a funnel (`pwa_install_*`) y una fase 2 backend que nunca se priorizó. Las métricas de instalación PWA no justifican infraestructura propia: el único caso de uso real era comportamiento de cliente sin estado durable, y el resto de métricas de producto por tenant se resuelven mejor con queries al backend o con un vendor futuro (PostHog, Sentry). Mantener el código añade complejidad sin valor operativo.

## What Changes

- Eliminar el módulo `core/analytics/` (`AnalyticsPort`, `ConsoleAnalyticsAdapter`, `provideAnalytics()`).
- Eliminar `EngagementEventHandler`, `pwa-analytics.events.ts` y todas las llamadas `track()` en notices, coordinator y commands.
- Simplificar `InstallEngagement`: las mutaciones (`recordSessionVisit`, `declineAutoPrompt`) devuelven solo el aggregate actualizado, sin `events[]` ni domain events de analytics.
- Eliminar `domain/events/install-engagement.events.ts` y el wiring `dispatch(events)` en commands.
- Quitar `provideAnalytics()` de `app.config.ts`.
- Actualizar spec `pwa-install`: remover requirements de funnel analytics, analytics no bloqueante y domain events → analytics.

**Sin cambio de comportamiento UX**: promoción, engagement gate, cooldowns, notices y pull model permanecen iguales.

## Non-goals / out of scope

- Integrar PostHog, Amplitude, Sentry u otro vendor de analytics/observabilidad.
- Endpoint backend de analytics (fase 2 descartada definitivamente).
- Métricas operativas por tenant (facturas, proveedores, inbox) — fuera de alcance; ver exploración previa sobre tenant DB queries.
- Cambios en `@bowerbird/system-notices` ni en la UX de instalación PWA.

## Capabilities

### New Capabilities

_(ninguna)_

### Modified Capabilities

- `pwa-install`: eliminar requirements de analytics/funnel y domain events orientados a tracking; el aggregate conserva solo lógica de engagement.

## Impact

| Área  | Módulos                                                                                                            |
| ----- | ------------------------------------------------------------------------------------------------------------------ |
| PWA   | `core/analytics/` — **eliminar**                                                                                   |
| PWA   | `core/pwa-install/domain/install-engagement.aggregate.ts` — simplificar mutaciones                                 |
| PWA   | `core/pwa-install/domain/events/` — **eliminar**                                                                   |
| PWA   | `core/pwa-install/application/engagement-event.handler.ts` — **eliminar**                                          |
| PWA   | `core/pwa-install/application/pwa-analytics.events.ts` — **eliminar**                                              |
| PWA   | `core/pwa-install/application/record-session-visit.command.ts`, `decline-auto-prompt.command.ts` — quitar dispatch |
| PWA   | `core/pwa-install/application/notices/*.ts`, `pwa-install.coordinator.ts` — quitar `track()`                       |
| PWA   | `app.config.ts` — quitar `provideAnalytics()`                                                                      |
| Specs | `openspec/specs/pwa-install/spec.md` — delta vía archive de este change                                            |
