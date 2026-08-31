## Context

Hoy `PwaService` captura `beforeinstallprompt` y `app.component.ts` renderiza un `hlm-card` fijo (`canInstall()` → siempre visible). El banner de update compite en la esquina opuesta. Ver `proposal.md` para motivación.

## Goals / Non-Goals

**Goals:**

- Paquete workspace `@bowerbird/system-notices` con orquestador reutilizable (una notice visible, prioridad, scope).
- Sub-unidades PWA: `core/analytics/`, `core/pwa-install/` — notices concretas implementan `SystemNotice` del paquete.
- Aggregate `InstallEngagement` con VOs, intent methods y domain events; application delgada.
- Notices concretas como **adapters** del port `SystemNotice` — delegan en commands/coordinator, sin reglas de engagement.
- Snackbar desktop + bottom sheet mobile (Chromium); guía iOS en sheet.
- Analytics vía port inyectable (fase 1: adapter consola).

**Non-Goals:**

- Endpoint backend analytics (fase 2).
- Publicar `@bowerbird/system-notices` en npm registry externo (solo workspace interno).
- Acoplar install-promotion a `HlmSidebarService` u otros módulos de presentación.
- Lógica de elegibilidad o cooldowns en layouts o `app.component`.

## Modular boundaries

### Mapa de sub-unidades

```
┌─────────────────────────────────────────────────────────────────┐
│ apps/pwa — composition roots (delgados)                         │
│  app.config.ts     → provideSystemNotices() + providePwaInstall │
│  app.component     → bb-system-notices-host (scope: global)     │
│  tenant-layout     → bb-system-notices-host (scope: tenant)     │
│                    + menú pull (PwaInstallCoordinator)          │
└──────────┬──────────────────────────────┬───────────────────────┘
           │                              │
  ┌────────▼──────────────┐    ┌──────────▼──────────┐
  │ @bowerbird/           │    │ core/pwa-install     │
  │ system-notices        │◀───│ notices + domain     │
  │ (orquestación UI)     │    └──────────┬──────────┘
  └───────────────────────┘               │
                               ┌──────────▼──────────┐
                               │ core/analytics       │
                               │ (solo pwa-install)   │
                               └─────────────────────┘
```

### Criterio de split — `@bowerbird/system-notices`

| Criterio           | ¿Aplica? | Señal                                                                             |
| ------------------ | -------- | --------------------------------------------------------------------------------- |
| Lenguaje distinto  | ✓        | «notice», «scope», «priority» vs «engagement», «install», «cooldown»              |
| Cadencia de cambio | ✓        | Orquestador genérico vs reglas PWA/iOS                                            |
| Escala/SLO         | —        | Mismo runtime browser                                                             |
| Consistencia       | ✓        | Cola en memoria (package) vs localStorage (pwa-install) — transacciones distintas |
| Ownership          | ✓        | Package reutilizable por futuras apps del monorepo                                |
| Dolor observable   | ✓        | Acoplar orquestador a PWA impedía test y reuso                                    |

### Ownership de estado

| Módulo           | Estado autoritativo                       | Persistencia                             |
| ---------------- | ----------------------------------------- | ---------------------------------------- |
| `system-notices` | Cola activa + notice en pantalla (sesión) | Memoria (orchestrator)                   |
| `pwa-install`    | `InstallEngagement` aggregate             | `bb:pwa:*` (localStorage/sessionStorage) |
| `analytics`      | Ninguno                                   | —                                        |

### Matriz de integración

| Origen → Destino                 | Contrato                                         | Sync/Async                        | Fallo                                |
| -------------------------------- | ------------------------------------------------ | --------------------------------- | ------------------------------------ |
| `pwa-install` → `system-notices` | `SystemNotice` (implements)                      | Sync (`canShow`/`show`/`dismiss`) | `show()` contenido; no crash app     |
| `pwa-install` → `analytics`      | `AnalyticsPort.track()`                          | Sync fire-and-forget              | try/catch en adapter; nunca throw    |
| `app.config` → ambos             | `provideSystemNotices()` + `providePwaInstall()` | Bootstrap                         | Fallo en registro = fail fast en dev |
| Layouts → `system-notices`       | Host component (`scope` input)                   | Sync                              | Host delgado; sin lógica de negocio  |
| Layouts → `pwa-install`          | `PwaInstallCoordinator`                          | Sync                              | Solo menú pull; sin storage directo  |

**Composition root único:** `app.config.ts` es el único lugar que registra notices (`multi: true` en token `SYSTEM_NOTICE`). `providePwaInstall()` contribuye notices concretas; **no** instancia el orchestrator ni importa su interior.

| Unidad           | Ubicación                        | Responsabilidad                      | Superficie pública                                                                                  |
| ---------------- | -------------------------------- | ------------------------------------ | --------------------------------------------------------------------------------------------------- |
| `system-notices` | `packages/system-notices/`       | Cola, prioridad, scope, host Angular | `SystemNotice`, `SystemNoticesOrchestrator`, `SystemNoticesHostComponent`, `provideSystemNotices()` |
| `analytics`      | `apps/pwa/.../core/analytics/`   | Eventos de producto                  | `AnalyticsPort.track()`                                                                             |
| `pwa-install`    | `apps/pwa/.../core/pwa-install/` | Runtime, aggregate, notices PWA      | `PwaInstallCoordinator`, commands, `providePwaInstall()`, notice classes                            |

**Regla de dependencia:**

```
@bowerbird/system-notices  →  (sin dep de pwa-install ni Bowerbird domain)
core/pwa-install           →  @bowerbird/system-notices, core/analytics
apps/pwa layouts           →  @bowerbird/system-notices (host), pwa-install (coordinator)
```

`@bowerbird/system-notices` MUST NOT import código de `apps/pwa` ni conocer PWA install, engagement ni analytics.

### Principios aplicados

| Principio                   | Aplicación                                                                                                                                         |
| --------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Boundaries**              | Cada sub-unidad exporta vía `index.ts`; internals no re-exportados                                                                                 |
| **State isolation**         | `system-notices`: cola en memoria (sesión). `pwa-install`: `bb:pwa:*` vía repository. `analytics`: stateless. Sin storage compartido entre módulos |
| **Composability**           | Package sin conocer consumers; notices registradas en composition root vía token `SYSTEM_NOTICE`                                                   |
| **Deployment independence** | `@bowerbird/system-notices` build/lint/test en CI aislado (`pnpm --filter`)                                                                        |
| **Explicit communication**  | Notices implementan `SystemNotice`; analytics vía `AnalyticsPort`; runtime vía `PwaRuntimePort`                                                    |
| **Replaceability**          | `AnalyticsPort`, `EngagementStoragePort`, `PwaRuntimePort`, `ViewportPort` — adapters intercambiables en tests                                     |
| **Independence**            | Aggregate + VOs testables sin Angular ni storage                                                                                                   |
| **Observability**           | Todos los eventos con prefijo `pwa_` / `system_notice_`; `track()` nunca lanza                                                                     |
| **Fail independence**       | Fallo en analytics o en `show()` de una notice no bloquea la app ni otras notices                                                                  |

### Anti-patrones prohibidos

- `tenant-layout` importando `localStorage`, `beforeinstallprompt` o claves `bb:pwa:*`.
- `PwaInstallNotice` leyendo `HlmSidebarService` — usar `ViewportPort` propio (mismo breakpoint, sin acoplar al sidebar).
- Mega-servicio `PwaService` con UI, engagement, analytics y SW — dividir responsabilidades.
- Orchestrator con lógica de negocio de install — solo prioridad y ciclo de vida de notices.
- `PwaEngagementService` con `if (visits.length >= 2)` o `prefs.dismissCount++` — lógica MUST vivir en el aggregate.
- Setters públicos o DTOs mutables (`InstallPrefs { dismissCount }` con asignación directa).
- `pwa-install` importando orchestrator internals (solo `SystemNotice` port + token de registro).
- `providePwaInstall()` instanciando `SystemNoticesOrchestrator` — wiring solo en `provideSystemNotices()`.
- `core/analytics/` como grab-bag cross-app — solo `pwa-install` lo consume (fase 1); extraer a package si ≥2 consumers.

## Layout de módulos

```
packages/system-notices/                    # @bowerbird/system-notices
├── package.json
├── project.json / ng-packagr or tsc config
├── src/
│   ├── lib/
│   │   ├── ports/
│   │   │   └── system-notice.port.ts       # SystemNotice interface + tokens
│   │   ├── system-notices.orchestrator.ts  # framework-agnostic core
│   │   └── angular/
│   │       ├── system-notices-host.component.ts
│   │       └── provide-system-notices.ts
│   └── index.ts                            # public API del package
└── README.md

apps/pwa/src/app/core/analytics/
├── application/ports/analytics.port.ts
├── infrastructure/console-analytics.adapter.ts
└── index.ts

apps/pwa/src/app/core/pwa-install/
├── domain/
│   ├── value-objects/ ...
│   ├── install-engagement.aggregate.ts
│   └── events/ ...
├── application/
│   ├── ports/ ...
│   ├── commands/ ...
│   ├── pwa-install.coordinator.ts
│   ├── engagement-event.handler.ts
│   └── notices/                            # implements SystemNotice from @bowerbird/system-notices
│       ├── pwa-update.notice.ts
│       ├── pwa-install-chromium.notice.ts
│       └── pwa-ios-install.notice.ts
├── infrastructure/ ...
├── presentation/ ...
└── index.ts
```

### Package `@bowerbird/system-notices` — decisiones

| Aspecto               | Decisión                                                                                          |
| --------------------- | ------------------------------------------------------------------------------------------------- |
| **Nombre**            | `@bowerbird/system-notices` (workspace `packages/system-notices/`)                                |
| **Peer deps**         | `@angular/core` (solo entry `angular/`); core orchestrator sin Angular                            |
| **Exports**           | `./` → port + orchestrator; `./angular` → host + `provideSystemNotices`                           |
| **Estado**            | Cola en memoria (sesión) — owned by orchestrator inside package                                   |
| **Extensión**         | Consumers registran via `provideX({ multi: true, useClass: Notice })` sobre token `SYSTEM_NOTICE` |
| **Observabilidad**    | Eventos `system_notice_*` emitidos por package; `pwa_*` por pwa-install                           |
| **Fail independence** | `show()`/`dismiss()` en notice concreta no propagan; orchestrator continúa cola                   |

**Alternativa rechazada:** `core/system-notices/` dentro de `apps/pwa` — no reutilizable por otras apps del monorepo ni testeable aisladamente en CI del package.

**Alternativa rechazada:** publicar en npm externo — over-engineering; workspace package es suficiente.

## Decisions

### 1. Package workspace + sub-unidades PWA en lugar de servicios planos

**Por qué:** `core/services/pwa.service.ts` mezcla runtime, señales UI y SW. El orquestador de notices es cross-cutting y vive en `@bowerbird/system-notices`; PWA install es consumer con notices concretas.

**Alternativa rechazada:** todo en `core/services/` o `core/system-notices/` dentro de `apps/pwa`.

### 2. `SystemNotice` como port composable

```typescript
interface SystemNotice {
  readonly id: string;
  readonly priority: number;
  readonly scope: 'global' | 'tenant';
  canShow(): boolean;
  show(): void;
  dismiss(reason: string): void;
}
```

Registro en `app.config.ts`:

```typescript
// app.config.ts — único composition root
provideSystemNotices(),   // orchestrator + host infra
providePwaInstall(),      // multi-provide SYSTEM_NOTICE: update, install-chromium, install-ios
```

Host importado desde `@bowerbird/system-notices/angular`. Notices concretas viven en `pwa-install/application/notices/` e implementan el port del package.

| Notice                 | Priority | Scope    |
| ---------------------- | -------- | -------- |
| `pwa-update`           | 100      | `global` |
| `pwa-install-chromium` | 50       | `tenant` |
| `pwa-install-ios`      | 50       | `tenant` |

**Alternativa rechazada:** orchestrator hardcodeado con imports de notices — no composable ni testeable.

**Nota DDD:** `@bowerbird/system-notices` es **infraestructura de orquestación UI**, no bounded context con modelo de dominio. `SystemNoticesOrchestrator` = application service genérico (cola/prioridad). Sin aggregate ni domain events en el package.

### 3. Bounded context `pwa-install` — lenguaje ubicuo

| Término                         | Tipo DDD               | Significado                                                 |
| ------------------------------- | ---------------------- | ----------------------------------------------------------- |
| `InstallEngagement`             | Aggregate Root         | Estado de promoción por navegador (visitas + prefs dismiss) |
| `VisitHistory`                  | VO                     | Secuencia inmutable de visitas de sesión                    |
| `AutoPromptPrefs`               | VO                     | Cooldowns y silenciamiento tras dismiss                     |
| `DismissReason`                 | VO                     | `timeout`, `not_now`, `continue_browser`                    |
| `EngagementWindow`              | VO                     | Ventana de 7 días desde primera visita                      |
| `recordSessionVisit`            | Intent method          | Nueva sesión → append visita                                |
| `declineAutoPrompt`             | Intent method          | Dismiss atómico (cooldown + contador)                       |
| `canShowAutoPrompt`             | Query (AR)             | engagement ∧ ¬cooldown ∧ ¬silenced                          |
| `PwaInstallCoordinator`         | Application facade     | Consultas + dispara commands; **sin reglas**                |
| `*Notice` (chromium/ios/update) | Adapter (port)         | `canShow()`/`dismiss()` delegan; **sin reglas**             |
| `PwaRuntimeService`             | Infrastructure adapter | Browser events (`beforeinstallprompt`, SW)                  |

### 4. Modelo táctico DDD — `InstallEngagement` (aggregate root)

**Diagnóstico del diseño anterior:** funciones sueltas `isEligibleForAutoPrompt(visits: Date[], prefs: InstallPrefs)` = **Moderate anemia** (primitivos + lógica fuera del modelo, coordinator-style).

**Modelo corregido:**

```
InstallEngagement (Aggregate Root)
├── VisitHistory          VO — inmutable, recordVisit(now)
├── AutoPromptPrefs       VO — inmutable, decline(reason, now)
└── EngagementWindow      VO — 7 días desde primera visita
```

| Building block      | Tipo | Responsabilidad                                     |
| ------------------- | ---- | --------------------------------------------------- |
| `VisitHistory`      | VO   | Registrar visitas; `hasMetEngagementThreshold(now)` |
| `AutoPromptPrefs`   | VO   | Cooldowns; `isSilenced()` tras 3 dismisses          |
| `DismissReason`     | VO   | `timeout` \| `not_now` \| `continue_browser`        |
| `InstallEngagement` | AR   | Única puerta de mutación del estado de promoción    |

**Intent methods (sin setters públicos):**

```typescript
class InstallEngagement {
  static reconstitute(visits: VisitHistory, prefs: AutoPromptPrefs): InstallEngagement;

  recordSessionVisit(now: Date): { engagement: InstallEngagement; events: DomainEvent[] };
  declineAutoPrompt(reason: DismissReason, now: Date): { engagement: InstallEngagement; events: DomainEvent[] };
  canShowAutoPrompt(now: Date): boolean;
  isEligibleForAutoPrompt(now: Date): boolean;
}
```

**VOs con factories:**

```typescript
DismissReason.fromUserAction('not_now' | 'continue_browser' | 'timeout')
VisitHistory.empty() | VisitHistory.reconstitute(timestamps[])
AutoPromptPrefs.initial() | AutoPromptPrefs.reconstitute(dto)
```

**Guards en intent methods (invariantes):**

- `declineAutoPrompt`: `DismissReason` válido; aplica cooldown + incremento en una sola transacción lógica
- `recordSessionVisit`: no duplica visita en la misma sesión (flag detectado en application antes de invocar)
- `canShowAutoPrompt`: false si `prefs.isSilenced()` o cooldown activo

**Domain events** (publicados por intent methods, manejados en application):

| Evento                     | Cuándo                                                     |
| -------------------------- | ---------------------------------------------------------- |
| `SessionVisitRecorded`     | Tras `recordSessionVisit`                                  |
| `AutoPromptBecameEligible` | Primera vez que `isEligibleForAutoPrompt` pasa a true      |
| `AutoPromptDeclined`       | Tras `declineAutoPrompt`                                   |
| `PwaInstalled`             | Runtime detecta `appinstalled` (application, no aggregate) |

`EngagementEventHandler` (application) mapea eventos → `AnalyticsPort`. El dominio **no** importa analytics.

**Application delgada:**

```typescript
// record-session-visit.command.ts
const engagement = await repo.load();
const result = engagement.recordSessionVisit(clock.now());
await repo.save(result.engagement);
eventHandler.dispatch(result.events);
```

**Alternativa rechazada:** `PwaEngagementService` con lógica `if (visits.length >= 2)` — coordinator anémico.

### 5. Notices como adapters (sin lógica de dominio)

```typescript
// pwa-install-chromium.notice.ts — adapter del port SystemNotice
class PwaInstallChromiumNotice implements SystemNotice {
  canShow(): boolean {
    return this.runtime.canInstallNative()
      && this.coordinator.canShowAutoPrompt()  // delega al aggregate vía coordinator
  }
  dismiss(reason: string): void {
    void this.declineCommand.execute(DismissReason.fromUserAction(reason))
  }
  show(): void { this.presenter.open({ onInstall: ..., onDismiss: ... }) }
}
```

**Anti-patrón:** `canShow()` con `visits.length >= 2` o lectura directa de `localStorage` en la notice.

### 6. Repository port (no engagement port con primitivos)

```typescript
interface InstallEngagementRepository {
  load(): InstallEngagement;
  save(engagement: InstallEngagement): void;
}
```

`EngagementStorageRepository` serializa el aggregate completo; único writer de `bb:pwa:*`.

**Alternativa rechazada:** `EngagementPort { recordVisit(), getPrefs(), setDismissCount() }` — expone mutación granular.

### 7. Visit tracking y reglas (dentro del aggregate)

Nueva sesión detectada en application (sessionStorage flag) → invoca `recordSessionVisit`.

Elegible = `VisitHistory.hasMetEngagementThreshold()`:

- `visitCount >= 2`
- `(secondVisit - firstVisit) <= EngagementWindow.days` (7)

### 8. Cooldowns (dentro de `AutoPromptPrefs`)

| Reason                         | Cooldown                                                    |
| ------------------------------ | ----------------------------------------------------------- |
| `timeout`                      | 3 días                                                      |
| `not_now` / `continue_browser` | 7 días                                                      |
| 2º dismiss explícito           | 30 días                                                     |
| 3º dismiss explícito           | `permanentlySilenced` — `canShowAutoPrompt()` siempre false |

`declineAutoPrompt(reason)` aplica cooldown + incremento atómicamente (invariante).

### 9. Presentación delgada

Presenters solo reciben copy + callbacks. Lógica de `canShow`, dismiss y analytics vive en notices/coordinator.

| Plataforma       | Presenter                                      |
| ---------------- | ---------------------------------------------- |
| Chromium desktop | `InstallSnackbarPresenter` (Sonner 6s)         |
| Chromium mobile  | `InstallSheetPresenter`                        |
| iOS Safari       | `IosInstallSheetPresenter`                     |
| Pull model       | `tenant-layout` → `coordinator.openFromMenu()` |

Copy (spec): título «Instala Bowerbird», cuerpo «Tu espacio de trabajo, a un toque.»; iOS «Añade Bowerbird a tu inicio».

### 10. iOS detection (infrastructure, no expuesto)

Detección encapsulada en `PwaRuntimeService` o adapter dedicado; consumers usan `canShowIosGuide(): boolean`.

### 11. Analytics como shared kernel

```typescript
interface AnalyticsPort {
  track(event: PwaAnalyticsEvent, properties?: Record<string, unknown>): void;
}
```

Implementación fase 1: `ConsoleAnalyticsAdapter` (`console.debug` en dev, no-op silencioso en prod o debug-only). **Nunca throw.**

Fase 2: `HttpAnalyticsAdapter` sin cambiar consumidores.

### 12. Composición en hosts

```typescript
// app.component — import from @bowerbird/system-notices/angular
<bb-system-notices-host scope="global" />

// tenant-layout
<bb-system-notices-host scope="tenant" />
```

### 13. Eliminar UI legacy

Quitar cards fijas de `app.component.ts`. Deprecar `PwaService` monolítico; migrar a `pwa-install/` con re-export temporal si hace falta.

## DDD compliance checklist (pre-implementación)

- [ ] `InstallEngagement` sin setters públicos; solo intent methods + `reconstitute()`
- [ ] VOs inmutables con factories (`empty`, `reconstitute`, `fromUserAction`)
- [ ] Invariantes (cooldown atómico, silenciamiento) solo en aggregate/VOs — no en notices ni coordinator
- [ ] Tests del aggregate + VOs sin TestBed, storage ni Angular (tabla-driven con `now` inyectado)
- [ ] Commands delgados: load → intent method → save → dispatch events
- [ ] Domain events definidos en `domain/events/`; no confundir con `pwa-analytics.events.ts` (application)
- [ ] `EngagementStorageRepository` serializa/deserializa aggregate vía `reconstitute()`, no DTOs mutables
- [ ] Notices (`*Notice`) solo delegan; grep sin `visits.length` ni `bb:pwa:` en `application/notices/`
- [ ] `PwaInstallCoordinator` sin aritmética de engagement — solo llama `engagement.canShowAutoPrompt()`
- [ ] Grep: `bb:pwa:` solo en `engagement-storage.repository.ts`

## Modular compliance checklist (pre-implementación)

- [ ] `@bowerbird/system-notices` no importa `apps/pwa` ni domain Bowerbird
- [ ] `packages/system-notices/src/index.ts` exporta solo port + orchestrator + token; `./angular` exporta host + `provideSystemNotices`
- [ ] Orchestrator internals (cola, sorting) no re-exportados en public API
- [ ] `pnpm --filter @bowerbird/system-notices test` pasa en CI aislado
- [ ] `apps/pwa/package.json` declara `"@bowerbird/system-notices": "workspace:*"`
- [ ] `providePwaInstall()` solo hace `multi` provide de `SYSTEM_NOTICE`; no crea orchestrator
- [ ] `index.ts` de sub-unidades PWA lista solo exports públicos
- [ ] Ningún layout importa `infrastructure/` de `pwa-install` ni internals del package
- [ ] Grep/dependency check: `packages/system-notices` sin imports de `apps/`
- [ ] Grep: `bb:pwa:` solo en `engagement-storage.repository.ts`
- [ ] `AnalyticsPort.track` envuelto en try/catch en adapter
- [ ] Eventos atribuibles: `system_notice_*` (package) vs `pwa_*` (pwa-install)

## Risks / Trade-offs

| Riesgo                                | Mitigación                                                                                                       |
| ------------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| Usuario borra localStorage            | Degradación aceptable; engagement reinicia                                                                       |
| Duplicar breakpoint mobile vs sidebar | Constante compartida `MOBILE_BREAKPOINT` en `core/constants/` o duplicar valor documentado en `viewport.adapter` |
| Overhead de package workspace         | Turbo cache por package; orchestrator testeable sin levantar PWA                                                 |
| `beforeinstallprompt` one-shot        | `PwaRuntimePort`; menú pull vía `openFromMenu()`                                                                 |
| Dev mode sin SW                       | Notices registran `canShow() === false` en `isDevMode()`                                                         |

## Migration Plan

1. Scaffold `packages/system-notices/` + wire en `pnpm-workspace.yaml` / Turbo.
2. Implementar orchestrator + host Angular en el package; tests aislados.
3. Crear sub-unidades PWA (`analytics`, `pwa-install`) + notices que implementan `SystemNotice`.
4. Registrar providers en `app.config.ts`; montar hosts; eliminar cards legacy.
5. Deprecar `core/services/pwa.service.ts`.
6. Verificar checklists + flujos manuales.

Rollback: revertir commit; sin migraciones backend.

## Open Questions

_(ninguna)_
