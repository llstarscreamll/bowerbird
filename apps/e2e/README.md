# E2E con Playwright

Este paquete contiene la suite E2E orientada a user journeys reales.

## Objetivos

- Asegurar comportamiento del sistema end-to-end.
- Documentar flujos reales con `test.step(...)`.
- Emular uso real del usuario final.
- Mantener una suite escalable con patrones conocidos.

## Estructura

- `tests/auth/*`: escenarios de negocio legibles para Dev, QA, PM y UX.
- `tests/pages/*`: Page Objects para aislar locators y acciones de UI.
- `tests/fixtures/*`: fixtures compartidas para preparar contexto de pruebas.
- `tests/support/*`: clientes API y factories para datos de prueba.

## Journey inicial

- `auth/email-password.spec.ts`
  - Creacion de cuenta local (endpoint local `register-local`).
  - Inicio de sesion por UI con email y password.
  - Caso negativo de credenciales invalidas.

Nota: hoy la PWA no tiene formulario de registro local en UI. Por eso la creacion de cuenta se ejecuta por API para habilitar el journey completo de autenticacion local.

## Ejecucion

Prerequisitos:

- Stack local arriba (`pnpm run dev` en otra terminal, desde la raiz).
- Navegador de Playwright instalado: `pnpm run test:e2e:install`.
- El backend debe correr en modo `local` o `development` para permitir `register-local`.

Pasos recomendados desde la raiz del monorepo:

```bash
mise install
pnpm install
pnpm run test:e2e:install
pnpm run test:e2e
```

Desde raiz del monorepo:

```bash
pnpm run test:e2e
pnpm run test:e2e:headed
pnpm run test:e2e:ui
```

Desde el paquete `apps/e2e`:

```bash
pnpm run test:e2e
pnpm run test:e2e:headed
pnpm run test:e2e:ui
```

Ver tests sin ejecutarlos:

```bash
pnpm --filter @bowerbird/e2e run test:e2e --list
```

Abrir reporte HTML despues de una ejecucion:

```bash
pnpm --filter @bowerbird/e2e exec playwright show-report
```

## Troubleshooting rapido

- Si aparece `Executable doesn't exist`, ejecuta `pnpm run test:e2e:install`.
- Si falla con `status=502` en `register-local`, valida que `pnpm run dev` este levantado y que `https://api.bowerbird.dev/api/health` responda OK.
- Si quieres apuntar a otro entorno, define `E2E_BASE_URL` y `E2E_API_BASE_URL`.

## Variables de entorno

- `E2E_BASE_URL` (default: `https://app.bowerbird.dev`)
- `E2E_API_BASE_URL` (default: derivado a `https://api.bowerbird.dev`)
- `E2E_RUN_ID` (opcional, para agrupar datos de prueba)
