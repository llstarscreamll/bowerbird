## Context

`pwa-install` ya está implementado con `core/analytics/`, `EngagementEventHandler` y domain events que solo alimentan `track()`. Ver `proposal.md` — Why.

El aggregate `InstallEngagement` devuelve `{ engagement, events[] }` aunque la lógica de negocio (`wasEligible`, `VisitHistory`, `AutoPromptPrefs`) no depende de esos eventos.

## Goals / Non-Goals

**Goals:**

- Eliminar toda infraestructura de analytics propia del módulo PWA install.
- Simplificar el aggregate y commands sin cambiar UX observable.
- Dejar `pwa-install` enfocado en promoción + engagement, sin acoplamiento a tracking.

**Non-Goals:**

- Integrar PostHog, Sentry u otro vendor ahora.
- Tocar `@bowerbird/system-notices`, presenters ni reglas de engagement.
- Backend de analytics ni métricas operativas por tenant.

## Decisions

### 1. Eliminar `core/analytics/` completo

**Decisión**: Borrar el módulo entero (`AnalyticsPort`, `ConsoleAnalyticsAdapter`, `provideAnalytics()`).

**Alternativa rechazada**: Mantener el port vacío como stub para PostHog futuro — añade API sin consumer y presiona a diseñar abstracción antes de elegir vendor.

### 2. Eliminar `EngagementEventHandler` y `pwa-analytics.events.ts`

**Decisión**: Quitar el handler y todas las llamadas `track()` en notices, coordinator y commands.

**Alternativa rechazada**: Dejar `track()` como no-op — código muerto que confunde sobre qué se mide.

### 3. Simplificar mutaciones del aggregate

**Decisión**: `recordSessionVisit()` y `declineAutoPrompt()` devuelven `InstallEngagement` directamente (o un tipo mínimo sin `events`).

```typescript
// Antes
recordSessionVisit(now): InstallEngagementMutation  // { engagement, events[] }

// Después
recordSessionVisit(now): InstallEngagement
```

La lógica interna (`wasEligible`, umbral 2ª visita) se mantiene; solo se elimina la colección de eventos.

**Alternativa rechazada**: Conservar domain events para uso futuro — YAGNI; PostHog no los necesita.

### 4. Commands sin dispatch post-mutación

**Decisión**: `RecordSessionVisitCommand` y `DeclineAutoPromptCommand` hacen load → intent method → save → return. Sin `EngagementEventHandler.dispatch()`.

### 5. Instrumentación futura (PostHog)

**Decisión**: Cuando llegue el momento, `posthog.capture()` en 3–5 puntos puntuales (prompt shown, install accepted/dismissed, `appinstalled`) sin reintroducir capa intermedia hasta que el vendor sea incómodo de testear.

**Nota**: Sentry cubre errores/performance, no funnels de producto — son complementarios, no sustitutos.

## Archivos afectados

| Acción      | Ruta                                                                                                                |
| ----------- | ------------------------------------------------------------------------------------------------------------------- |
| Eliminar    | `apps/pwa/src/app/core/analytics/`                                                                                  |
| Eliminar    | `core/pwa-install/application/engagement-event.handler.ts`                                                          |
| Eliminar    | `core/pwa-install/application/pwa-analytics.events.ts`                                                              |
| Eliminar    | `core/pwa-install/domain/events/install-engagement.events.ts`                                                       |
| Simplificar | `install-engagement.aggregate.ts`                                                                                   |
| Simplificar | `record-session-visit.command.ts`, `decline-auto-prompt.command.ts`                                                 |
| Simplificar | `pwa-install-chromium.notice.ts`, `pwa-ios-install.notice.ts`, `pwa-update.notice.ts`, `pwa-install.coordinator.ts` |
| Simplificar | `app.config.ts` — quitar `provideAnalytics()`                                                                       |

## Grafo de dependencias (después)

```
core/pwa-install  →  @bowerbird/system-notices
                   (sin core/analytics)
```

## Risks / Trade-offs

| Riesgo                                        | Mitigación                                                   |
| --------------------------------------------- | ------------------------------------------------------------ |
| Pérdida de visibilidad del funnel PWA         | Aceptado; se recuperará con PostHog cuando haya tráfico real |
| Tests que assertean `track()` o domain events | Actualizar/eliminar en el mismo change                       |
| Reintroducir analytics ad hoc en notices      | Documentar en non-goals; vendor directo cuando aplique       |

## Migration Plan

1. Simplificar aggregate y commands (sin events).
2. Quitar `track()` de notices y coordinator.
3. Eliminar handler, analytics module y `provideAnalytics()`.
4. Ajustar tests del módulo `pwa-install`.
5. `pnpm --filter @bowerbird/pwa lint && test && build`.

Rollback: revert del commit; no hay migración de datos ni API pública afectada.

## Open Questions

_(ninguna — alcance acotado a eliminación de código muerto/innecesario)_
