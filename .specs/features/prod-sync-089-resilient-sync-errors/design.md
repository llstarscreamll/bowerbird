# Diseno Tecnico - PROD-SYNC-089

## 1. Alcance de arquitectura

Esta feature introduce un contrato de error resiliente y seguro de extremo a extremo para sincronizacion de correo:

- Backend Go: clasificacion de fallos de proveedor -> mapeo JSON:API consistente -> meta accionable para UI.
- Frontend Angular: render de alertas contextuales con acciones dinamicas (`reauth`, `retry_after_seconds`).
- Seguridad: sanitizacion de contenido de correo en backend y frontend, PII masking en errores.
- Resiliencia de workers: tolerancia a mensajes problematicos sin DLQ por tenant.

## 2. Principios DDD y modularidad

### Bounded Contexts impactados

1. `platform` (shared kernel tecnico)
   - `apperrors`: taxonomia de errores y payload seguro.
   - `http/api`: serializacion JSON:API y mapeo HTTP.
2. `connections`
   - Traduccion de errores OAuth/provider a codigos semanticos de negocio.
3. `inbox`
   - Pipeline de sync, protecciones de parseo y resiliencia de workers.
4. `apps/pwa` (`core` + feature inbox/connections)
   - Interceptor + store/component para UX accionable.

### Contrato anticorrupcion (ACL)

Los errores crudos de SDK/proveedores (Google, Microsoft, Yahoo) no cruzan al frontend.
Se traducen a un modelo canonical:

- `ERR_SYNC_REAUTH_REQUIRED`
- `ERR_SYNC_RATE_LIMITED`
- `ERR_SYNC_PROVIDER_TEMPORARY`
- `ERR_SYNC_PAYLOAD_REJECTED`
- `ERR_SYNC_INTERNAL`

## 3. Diseno backend

## 3.1 Modelo de error canonical

Definir tipo de error enriquecido (en `platform/apperrors` o paquete dedicado de sync errors):

- Campos base: `code`, `message`, `cause`.
- Campos de contexto seguro: `provider`, `account_email_masked`, `requires_reauth`, `retry_after_seconds`, `help_path`.
- Politica de redaccion: funciones de masking para secrets/PII.

Regla: `message/detail` nunca contiene token o payload crudo de proveedor.

## 3.2 Adaptador JSON:API

Extender `api.MapError`/`api.Wrap` para:

- Construir `errors[0].code/title/detail/status`.
- Emitir `links.about` desde catalogo central de documentacion de errores.
- Emitir `meta` solo con whitelist permitida.
- Mantener `id` de correlacion (`sentry-trace` o equivalente).
- Mantener `meta._debug` solo en local/dev, ya redactado.

## 3.3 Catalogo de errores de sync

Crear tabla/archivo de mapeo deterministico:

- Input: tipo de error proveedor + contexto (HTTP status proveedor, subcode, headers, retry-after).
- Output: codigo canonical + status HTTP + flags de UX (`requires_reauth`, `retry_after_seconds`).

Objetivo: agregar nuevo provider sin cambiar frontend core.

## 3.4 Resiliencia de workers sin DLQ por tenant

Se implementa estrategia de fallo controlado por mensaje:

1. Guardrails de tamano:
   - Max bytes por payload MIME/HTML/texto.
   - Rechazo temprano antes de parseo profundo.
2. Guardrails de tiempo y memoria:
   - `context.WithTimeout` por mensaje.
   - Cancelacion cooperativa en parseadores y llamadas IO.
3. Guardrails de robustez:
   - `recover()` por unidad de trabajo para evitar caida del worker.
   - Clasificacion de error no-retriable para payload malicioso/invalido.
4. Continuidad operativa:
   - Registrar incidente estructurado y marcar mensaje fallido.
   - Continuar inmediatamente con siguiente mensaje de cola.

Nota: no se crean DLQs por tenant; se mantiene topologia de colas actual.

## 3.5 Seguridad de ingesta

- Tokens OAuth cifrados en reposo en infraestructura (`connections` repo layer).
- MIME crudo en S3 (auditoria), datos operativos saneados en DB.
- Validacion RFC para campos email/date.
- Sanitizacion server-side para evitar persistir HTML activo no confiable.

## 4. Diseno frontend Angular

## 4.1 Contrato de error tipado

Agregar/ajustar tipo compartido JSON:API error en `core`:

- `JsonApiError` con `meta` tipada para sync:
  - `requires_reauth?: boolean`
  - `provider?: 'GMAIL' | 'OUTLOOK' | 'YAHOO' | 'HOTMAIL' | string`
  - `retry_after_seconds?: number`
  - `account_email?: string`

## 4.2 Comportamiento UX

- 5xx/network: interceptor sigue enviando toast global.
- 4xx sync business: componente/store muestra alerta contextual persistente.
- Si `requires_reauth`: render CTA y redireccion a login provider.
- Si `retry_after_seconds`: countdown visual + disable de reintento.

## 4.3 Seguridad de render de email

- Sanitizacion estricta con DOMPurify integrada con seguridad Angular.
- Hook para reforzar enlaces `target` + `rel`.
- Bloqueo de imagenes externas por defecto + accion "mostrar imagenes".
- Render aislado via `iframe sandbox` para contenido enriquecido.

## 5. Observabilidad y soporte

Registrar eventos estructurados:

- `sync_error_classified`
- `sync_reauth_required`
- `sync_rate_limited`
- `sync_payload_rejected`

Campos minimos: `tenant_id`, `provider`, `account_id`, `error_code`, `correlation_id`.

## 6. Estrategia de pruebas

### Backend

- Unit: mapper provider error -> canonical error.
- Unit: redaction/masking de `detail` y `meta._debug`.
- Integration: handler retorna JSON:API estricto con `links.about` y `meta` esperada.
- Worker resilience test: mensaje oversized/malicioso falla y el siguiente mensaje se procesa exitosamente.

### Frontend

- Unit: parseo de `meta` y branching (`reauth`, `retry_after_seconds`).
- Unit: countdown deshabilita boton hasta llegar a 0.
- Unit: sanitizacion elimina scripts/handlers.
- Component/e2e: alerta contextual visible, CTA correcta por provider.

## 7. Riesgos y mitigaciones

- Riesgo: sobreexposicion de datos en errores debug.
  - Mitigacion: redaction centralizada y pruebas de snapshot de payload.
- Riesgo: falsos positivos en rechazo de payload por limites.
  - Mitigacion: limites configurables y metricas para tuning.
- Riesgo: divergencia de contrato entre backend y frontend.
  - Mitigacion: contrato tipado compartido + pruebas de contrato.
