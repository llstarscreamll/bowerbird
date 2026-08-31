## 0. Package `@bowerbird/system-notices`

- [x] 0.1 Scaffold `packages/system-notices/` (`package.json`, build config, `src/index.ts`, `src/angular/`); verificar `pnpm --filter @bowerbird/system-notices build` y dependency `workspace:*` en PWA
- [x] 0.2 Implementar `SystemNotice` port + `SYSTEM_NOTICE` token + `SystemNoticesOrchestrator` con tests aislados; verificar prioridad, una-notice-activa y fail independence en `show()`
- [x] 0.3 Implementar `SystemNoticesHostComponent` + `provideSystemNotices()` en entry `./angular`; exportar solo API pública; verificar filtro por `scope`

## 1. Shared kernel — analytics (PWA)

- [x] 1.1 Crear `core/analytics/` con `AnalyticsPort` y `ConsoleAnalyticsAdapter`; verificar que `track()` nunca lanza

## 2. PWA install — domain (rich model)

- [x] 2.1 Crear VOs (`VisitHistory`, `AutoPromptPrefs`, `DismissReason`, `EngagementWindow`) con factories `empty`/`reconstitute`/`fromUserAction`; tests puros tabla-driven
- [x] 2.2 Crear aggregate `InstallEngagement` con `reconstitute()`, intent methods y guards; tests: 1ª visita, 2ª ≤7d, 2ª >7d, cooldowns, 3er dismiss silenciado, decline atómico
- [x] 2.3 Crear domain events en `domain/events/` retornados por intent methods; verificar separación de `pwa-analytics.events.ts` (application)

## 3. PWA install — application e infrastructure

- [x] 3.1 Crear `InstallEngagementRepository` + `EngagementStorageRepository` con serialize/deserialize vía `reconstitute()`; único writer `bb:pwa:*`
- [x] 3.2 Crear `RecordSessionVisitCommand`, `DeclineAutoPromptCommand` + `EngagementEventHandler` → `AnalyticsPort`
- [x] 3.3 Implementar `PwaRuntimeService`, `PwaInstallCoordinator` (sin reglas de engagement), `ViewportAdapter`
- [x] 3.4 Implementar notices como adapters (`canShow`/`dismiss` delegan); `providePwaInstall()` solo `multi` provide del token; grep sin reglas en `notices/`

## 4. Composición e integración UI

- [x] 4.1 Registrar `provideSystemNotices()` + `providePwaInstall()` en `app.config.ts`; montar `<bb-system-notices-host>` por scope; eliminar cards legacy
- [x] 4.2 Menú pull vía `coordinator.openFromMenu()`; deprecar `core/services/pwa.service.ts`

## 5. Verificación

- [x] 5.1 `pnpm --filter @bowerbird/system-notices lint|test` y `pnpm --filter @bowerbird/pwa lint` pasan
- [x] 5.2 Checklists DDD + modular en `design.md` (aggregate-only invariants, grep notices, `bb:pwa:` aislado)
- [ ] 5.3 Flujos manuales: snackbar, sheet, iOS, pull menú, update > install, cooldowns
