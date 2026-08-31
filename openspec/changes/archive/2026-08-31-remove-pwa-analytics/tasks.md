## 1. Domain — simplificar aggregate

- [x] 1.1 Quitar `InstallEngagementMutation`, `events[]` y `domain/events/install-engagement.events.ts`; mutaciones devuelven `InstallEngagement` directamente — verificar tests del aggregate pasan (`pnpm --filter @bowerbird/pwa test -- install-engagement`)

## 2. Application — quitar analytics wiring

- [x] 2.1 Simplificar `RecordSessionVisitCommand` y `DeclineAutoPromptCommand` (load → intent → save, sin `dispatch`) — verificar tests de commands pasan
- [x] 2.2 Eliminar `engagement-event.handler.ts` y `pwa-analytics.events.ts`; quitar `track()` de notices (`pwa-install-chromium`, `pwa-ios-install`, `pwa-update`) y `pwa-install.coordinator.ts` — verificar `grep -r EngagementEventHandler apps/pwa/src` sin resultados

## 3. Eliminar módulo analytics

- [x] 3.1 Borrar `core/analytics/` y quitar `provideAnalytics()` de `app.config.ts` — verificar `grep -r core/analytics apps/pwa/src` sin resultados

## 4. Tests y verificación

- [x] 4.1 Actualizar tests que referencien domain events, `track()` o `AnalyticsPort` — verificar suite `pwa-install` verde
- [x] 4.2 Ejecutar `pnpm --filter @bowerbird/pwa lint && pnpm --filter @bowerbird/pwa test && pnpm --filter @bowerbird/pwa build` — verificar exit 0
