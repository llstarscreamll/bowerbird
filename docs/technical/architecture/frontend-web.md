# Arquitectura frontend (Angular + PWA)

## Stack

- Angular 21 (Zoneless con `provideZonelessChangeDetection`)
- Signals para gestión de estado local
- NgRx SignalStore (`@ngrx/signals`) para estado global estructurado
- TailwindCSS
- PWA con Service Worker de Angular
- Vitest para Unit Testing (ultra rápido con `@angular/build:unit-test`)

## Capacidades tecnicas implementadas

- SPA con rutas standalone.
- Vista principal orquestada por el `SystemStore` (NgRx SignalStore) consumiendo `/api/health`.
- Proxy local y DNS emulado (`app.bowerbird.dev` a `api.bowerbird.dev`).
- Instalacion PWA (manifest + iconos + prompt de instalacion).
- Actualizacion de versiones con `SwUpdate` y accion de refresh.
- Setup Moderno Zoneless (sin `zone.js` y usando la API de Vitest predeterminada de Angular 21).

## Gestion de Estado (Convencion del Proyecto)

El proyecto asume una gestión de estado orientada a Signals y **prescinde totalmente del NgRx clásico (Redux/RxJS)**.

### Reglas de uso de estado:

1. **Estado local (Componentes):** Usa `signal()` o `computed()` directamente en la clase del componente.
2. **Estado compartido simple:** Usa un servicio (`Injectable`) estándar que exponga Signals.
3. **Estado global/estructurado (Múltiples entidades o asincronismo pesado):** Usa `@ngrx/signals` (`SignalStore`).
   - Se provee de `rxMethod` para enlazar observables limpios (como peticiones HTTP) al estado de los signals sin perder reactividad.
   - Todo SignalStore global debe ubicarse en `apps/pwa/src/app/core/store/`.

## Configuracion clave

- Angular app config: `apps/pwa/src/app/app.config.ts`
- Configuraciones de Build/Test: `apps/pwa/angular.json`
- Config cache SW: `apps/pwa/ngsw-config.json`
- Manifest: `apps/pwa/public/manifest.webmanifest`
- Estado Principal de ejemplo: `apps/pwa/src/app/core/store/system.store.ts`

## Verificacion local de PWA

```bash
pnpm --filter @bowerbird/pwa build
pnpm --filter @bowerbird/pwa preview:pwa
```

Luego abrir `http://localhost:4300` y validar en DevTools > Application.
