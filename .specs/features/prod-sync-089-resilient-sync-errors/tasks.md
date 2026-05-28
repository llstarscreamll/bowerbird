# Tasks - PROD-SYNC-089

## Estado general

- Total tareas: 10
- Paralelizables: 3 (`[P]`)
- Gate final: lint + test + build en backend y pwa

## Tareas

### T1 - Definir catalogo canonical de errores de sincronizacion

- Tipo: Backend
- Depends on: none
- What:
  - Definir codigos canonical (`ERR_SYNC_REAUTH_REQUIRED`, `ERR_SYNC_RATE_LIMITED`, etc.).
  - Definir mapeo a status HTTP y `links.about`.
  - Definir schema de `meta` permitida.
- Where:
  - `apps/backend/internal/platform/apperrors/*`
  - `apps/backend/internal/platform/http/api/*`
- Done when:
  - Existe catalogo versionado y usado por mapper central.
  - No hay codigos hardcodeados fuera del catalogo.
- Tests:
  - unit tests de mapeo code -> status/title/help.

### T2 - Extender serializacion JSON:API con meta accionable y masking

- Tipo: Backend
- Depends on: T1
- What:
  - Extender `api.Wrap`/`MapError` para incluir `links.about` y meta de sync.
  - Aplicar whitelist para `meta`.
  - Redactar secretos en `detail` y `meta._debug`.
- Where:
  - `apps/backend/internal/platform/http/api/errors.go`
  - `apps/backend/internal/platform/http/api/errors_test.go`
- Done when:
  - Errores de sync salen en JSON:API valido con campos esperados.
  - No se filtra informacion sensible en respuestas de error.
- Tests:
  - tests unitarios y de integracion de payload JSON:API.

### T3 - Traductor de errores de proveedor -> canonical

- Tipo: Backend
- Depends on: T1
- What:
  - Implementar traductor para OAuth revocado/expirado y rate limiting por provider.
  - Extraer `retry_after_seconds` cuando exista.
- Where:
  - `apps/backend/internal/connections/application/*`
  - `apps/backend/internal/inbox/application/*`
- Done when:
  - Errores externos no salen crudos; siempre pasan por canonical mapper.
- Tests:
  - unit tests por provider para escenarios principales.

### T4 - Guardrails de resiliencia en worker (sin DLQ por tenant)

- Tipo: Backend
- Depends on: T3
- What:
  - Limite de tamano de payload (MIME/HTML/texto).
  - Timeout por mensaje con cancelacion de contexto.
  - `panic recovery` por unidad de trabajo.
  - Clasificar payload malicioso/oversized como no-retriable y continuar cola.
- Where:
  - `apps/backend/internal/inbox/application/*`
  - `apps/backend/internal/inbox/infrastructure/*`
- Done when:
  - Un mensaje problematico falla controladamente y no bloquea procesamiento posterior.
- Tests:
  - integration test de cola con mensaje malo + mensaje valido subsecuente.

### T5 - Validacion estructural y sanitizacion server-side de contenido correo

- Tipo: Backend
- Depends on: T4
- What:
  - Validar `from/to/date` (RFC).
  - Limpiar contenido operativo previo a persistencia.
- Where:
  - `apps/backend/internal/inbox/domain/*`
  - `apps/backend/internal/inbox/application/*`
- Done when:
  - Datos invalidos/maliciosos no llegan a modelo operativo persistido.
- Tests:
  - unit tests de validadores y sanitizadores.

### T6 [P] - Tipado de contrato de error sync en frontend

- Tipo: Frontend
- Depends on: T2
- What:
  - Extender tipos de error JSON:API para `requires_reauth`, `provider`, `retry_after_seconds`.
- Where:
  - `apps/pwa/src/app/core/*`
- Done when:
  - Interceptor/store pueden consumir meta tipada sin casts inseguros.
- Tests:
  - unit tests de parsing/typing.

### T7 [P] - UX accionable: alerta contextual + CTA reauth + countdown

- Tipo: Frontend
- Depends on: T6
- What:
  - Render de alerta para 4xx sync con CTA de reconexion.
  - Countdown para rate limit con boton deshabilitado hasta expiry.
- Where:
  - `apps/pwa/src/app/features/inbox/*`
  - `apps/pwa/src/app/core/interceptors/error.interceptor.ts`
- Done when:
  - Escenarios reauth/rate-limit se muestran correctamente y sin spinner infinito.
- Tests:
  - unit/component tests de estado y countdown.

### T8 [P] - Capa de seguridad visual HTML en frontend

- Tipo: Frontend
- Depends on: none
- What:
  - Integrar DOMPurify.
  - Bloquear imagenes externas por defecto con opt-in.
  - Enlaces seguros (`target`, `rel`).
  - Render aislado con `iframe sandbox`.
- Where:
  - `apps/pwa/src/app/features/inbox/*`
- Done when:
  - HTML enriquecido no ejecuta scripts ni carga imagenes externas sin consentimiento.
- Tests:
  - unit tests de sanitizacion y mutacion de anchors/img.

### T9 - Observabilidad y eventos de soporte

- Tipo: Backend + Frontend
- Depends on: T2, T4, T7
- What:
  - Emitir logs/eventos estructurados de errores de sync y decisiones UX.
- Where:
  - `apps/backend/internal/*`
  - `apps/pwa/src/app/*`
- Done when:
  - Se puede rastrear un error de sync por `correlation_id` extremo a extremo.
- Tests:
  - validacion de campos minimos en logs (si aplica mediante tests de utilidades).

### T10 - Validacion final (gate de feature)

- Tipo: Verificacion
- Depends on: T1..T9
- What:
  - Ejecutar lint/test/build por modulo impactado.
  - Ejecutar smoke manual Stage/UAT segun criterios de aceptacion.
- Commands:
  - `pnpm --filter @bowerbird/backend lint`
  - `pnpm --filter @bowerbird/backend test`
  - `pnpm --filter @bowerbird/backend build`
  - `pnpm --filter @bowerbird/pwa lint`
  - `pnpm --filter @bowerbird/pwa test`
  - `pnpm --filter @bowerbird/pwa build`
- Done when:
  - Todas las pruebas pasan y criterios de aceptacion estan evidenciados.

## Orden de ejecucion recomendado

1. T1 -> T2 -> T3 -> T4 -> T5
2. En paralelo: T6, T7, T8
3. T9
4. T10

## Notas de implementacion

- Mantener compatibilidad con patron actual: `api.Wrap(handler, isDev)` y `apperrors.Wrap(...)`.
- No introducir `http.Error()` en handlers.
- No introducir DLQ por tenant; resiliencia por guardrails en el worker.
