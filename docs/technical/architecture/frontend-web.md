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

## Componentes y Utilidades Transversales (Shared UI)

Para mantener la consistencia visual y fomentar la reutilización bajo principios de diseño modular, los componentes puramente presentacionales de uso transversal se centralizan.

### Ubicación

- **Componentes Angular:** Ubicados en `apps/pwa/src/app/core/presentation/components/`.
- **Layouts Angular:** Ubicados en `apps/pwa/src/app/core/presentation/layouts/`.
- **Estilos globales (Primitivas):** Definidos vía Tailwind (`@apply`) en `apps/pwa/src/styles.css` (Ej. `.card`, `.btn-primary`, `.input-field`).

### Manejo de Errores y Retroalimentación UI

La arquitectura frontend distingue entre dos patrones visuales para comunicar resultados o errores al usuario, manejados globalmente por el `error.interceptor.ts`.

#### Toast (`ToastService`)

Servicio inyectable para emitir mensajes globales y efímeros (transitorios).

- **Responsabilidad Global:** El `error.interceptor.ts` se encarga de dispararlos automáticamente para errores severos como **5xx (Server Error)** y **0 (Network Error)**.
- **Uso desde Componentes:** Debe utilizarse para notificaciones de **éxito** tras completar un flujo importante (ej. _"Organización creada correctamente"_) o para avisos informativos no bloqueantes.
- **Comportamiento:** Flotan sobre la interfaz y desaparecen solos luego de un par de segundos.

#### Alert (`<app-alert>`)

Componente presentacional (Callout) para mensajes contextuales estáticos e incrustados.

- **Responsabilidad Local:** Se debe renderizar de forma _inline_ dentro del contexto afectado (arriba de un formulario, al tope de una lista vacía, o cerca del botón de acción).
- **Uso desde Componentes:** Diseñado para mostrar errores **4xx (Bad Request / Validation)** o problemas de dominio ("El email ya está registrado"). El componente de vista es responsable de atrapar el error mapeado por el interceptor y asignarlo al componente Alert.
- **Comportamiento:** Permanece en pantalla obligando al usuario a leerlo y, opcionalmente, ofrece un botón para cerrarlo. Soporta contenido HTML interno (ej. listar viñetas de errores de campos de un formulario) a través de proyección (`<ng-content>`).

### Layouts Principales

#### Tenant Layout (`<app-tenant-layout>`)

Actúa como el "shell" (contenedor principal) para todas las rutas protegidas que operan dentro del contexto de una organización.

- **Enrutamiento:** Envuelve las rutas hijas bajo el path `/:tenantId` (ej. `/:tenantId/inbox/unified`), extrayendo dinámicamente este parámetro de la URL.
- **Diseño Visual:** Implementa un diseño SaaS moderno con una barra lateral (Sidebar) colapsable a la izquierda (transición suave de 260px a 80px) y un lienzo principal (Canvas) que ocupa el resto de la pantalla.
- **Selector de Contexto:** Presenta de forma visual el nombre del tenant activo interactuando con la barra lateral, integrándose con el fondo blanco para una estética minimalista.

## Patrones de Interfaz (UI Patterns)

Para mantener la consistencia a nivel de producto, se establecen patrones de estructura visual según el caso de uso:

### Split-Pane (Master-Detail)

Utilizado para vistas de alta densidad de información, implementado en la **Bandeja Unificada (Inbox)**.

- **Propósito:** Evitar saltos de página constantes entre listas y detalles, optimizando el flujo de revisión de información.
- **Implementación:** El componente ocupa absolutamente todo el ancho y alto disponible provisto por el _Tenant Layout_ (`h-full w-full flex bg-white`).
- **Composición:**
  - **Panel Maestro (Izquierda):** Lista escaneable (ej. correos), con un ancho fijo (`w-[380px]`), barra de búsqueda, filtros rápidos y un indicador visual sobre el ítem activo.
  - **Panel de Detalle (Derecha):** Área de lienzo completo (`flex-1`) que muestra un "Empty State" elegante centrado cuando no hay selección, o expone el detalle estructurado de la entidad seleccionada (ej. asunto, remitente, contenido y tarjetas para documentos XML/PDF adjuntos).

### Componentes Destacados

#### Alert (`<app-alert>`)

Componente Callout diseñado para captar la atención del usuario siguiendo convenciones visuales semánticas de color e íconos. (Ver sección de [Manejo de Errores](#manejo-de-errores-y-retroalimentación-ui) para lineamientos de uso versus Toasts).

- **Casos de uso:**
  - **`error`** (rojo): Informar sobre errores en operaciones, fallos al leer información o listar errores devueltos por validaciones del backend tras el envío de formularios.
  - **`warning`** (amarillo): Alertar acerca de restricciones del sistema o acciones riesgosas.
  - **`info`** (azul): Recordar información útil contextual para realizar una acción.
  - **`success`** (verde): Confirmar la finalización exitosa de un proceso.
- **Especificaciones Técnicas:**
  - Utiliza la API moderna de Angular basada en Signals (`input`, `output`, `computed`) en un entorno _zoneless_.
  - Accesibilidad nativa implementada dinámicamente (`role="alert"` o `role="status"` según el nivel de severidad).
  - Uso de `<ng-content>` para permitir la inyección de marcado HTML enriquecido (ej. `<ul>` con múltiples mensajes de validación).

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
