# Frontend architecture

## Stack

- Angular 21, zoneless (`provideZonelessChangeDetection`)
- Signals + NgRx SignalStore (`@ngrx/signals`) — not classic NgRx/Redux
- Tailwind 4 + Spartan UI
- Angular service worker (PWA)
- Vitest via `@angular/build:unit-test`

## State

1. Local: `signal()` / `computed()` in the component.
2. Shared simple: injectable service exposing signals.
3. Global / multi-entity async: SignalStore + `rxMethod` for HTTP.

Keep feature orchestration in `*/application/*store.ts`; keep presentation thin.

## Shared UI

- Components: `apps/pwa/src/app/core/presentation/components/`
- Layouts: `apps/pwa/src/app/core/presentation/layouts/`
- Helm primitives: `apps/pwa/src/app/shared/ui/` — see [Spartan UI](../frontend/spartan-ui.md)
- Tokens: semantic classes from `styles.css` (`bg-background`, `text-muted-foreground`), not hardcoded palettes

### Feedback

| Kind                            | When                             |
| ------------------------------- | -------------------------------- |
| Toast (`ToastService` / Sonner) | Global 5xx, network, success     |
| `<hlm-alert>`                   | Contextual 4xx / form validation |

`error.interceptor.ts` handles JSON:API errors and auto-toasts 5xx/network.

### Tenant shell

`TenantLayoutComponent` wraps `/:tenantId/*` with a collapsible sidebar. `tenant.interceptor.ts` sets `X-Tenant-ID` from the path.

### Patterns

- **Master-detail:** full-height split pane (unified inbox).
- **Dialogs:** Spartan `<hlm-dialog>` / `<hlm-alert-dialog>` (prefer these over bespoke modals).

## Key paths

- Config: `app.config.ts`, `angular.json`, `ngsw-config.json`, `public/manifest.webmanifest`

## Verify PWA build

```bash
pnpm --filter @bowerbird/pwa build
pnpm --filter @bowerbird/pwa preview:pwa
```

Open `http://localhost:4300` → DevTools → Application.
