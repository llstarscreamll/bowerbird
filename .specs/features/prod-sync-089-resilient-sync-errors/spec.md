# Especificacion - UX Resiliente en Fallos de Sincronizacion

## Estado

- Estado: Aprobada para diseno e implementacion
- Fecha: 2026-05-27
- Epic: PROD-SYNC-089
- Idioma: ES

## Objetivo

Estandarizar y enriquecer el manejo de errores de sincronizacion de bandejas externas (Gmail, Outlook/Hotmail, Yahoo) para que la UI muestre estados accionables, sin cargas infinitas ni mensajes genericos, bajo contrato JSON:API consistente y seguro.

## Contexto de negocio

- El fallo silencioso de sincronizacion incrementa churn y tickets de soporte.
- El sistema actual responde de forma inconsistente ante revocacion OAuth y rate limiting.
- Producto requiere un catalogo reusable de estados de error para UX modular.

## Requisitos funcionales

### RF-ERR (Contrato de error y UX accionable)

- [PROD-SYNC-089-RF-001] Todo error 4xx/5xx de endpoints de sincronizacion debe responder en formato JSON:API `errors[]`.
- [PROD-SYNC-089-RF-002] Cada error de sincronizacion debe incluir `links.about` con URL de ayuda autodescriptiva por tipo de error.
- [PROD-SYNC-089-RF-003] `detail` debe incluir proveedor y cuenta afectada (ej: cuenta Outlook `personal@outlook.com` requiere atencion).
- [PROD-SYNC-089-RF-004] Si el fallo requiere reconexion OAuth, el error debe incluir `meta.requires_reauth=true` y `meta.provider=<PROVIDER>` para habilitar CTA inmediato en UI.
- [PROD-SYNC-089-RF-005] Si el fallo es transitorio por bloqueo/rate limit, el error debe incluir `meta.retry_after_seconds` para countdown y bloqueo temporal de reintento.
- [PROD-SYNC-089-RF-006] Ningun fallo de sync debe dejar estados indeterminados (spinner infinito o pantalla en blanco).

### RF-SEC-BE (Seguridad en backend/ingesta)

- [PROD-SYNC-089-RF-007] Access/Refresh tokens deben persistirse cifrados en reposo (AES-GCM-256 o equivalente) en capa de infraestructura.
- [PROD-SYNC-089-RF-008] El MIME/EML crudo puede almacenarse en S3 para auditoria, pero los datos operativos para UI deben persistirse saneados.
- [PROD-SYNC-089-RF-009] Campos estructurales (from/to/date) deben validarse contra formatos RFC; contenido invalido/malicioso se rechaza o limpia.
- [PROD-SYNC-089-RF-010] Debe limitarse tamano de contenido (texto/HTML) previo a parseo para mitigar agotamiento de memoria por payloads extremos.
- [PROD-SYNC-089-RF-011] El worker de sincronizacion debe fallar controladamente ante payloads maliciosos/pesados (ej: zip bomb textual), registrar error, ack/nack segun politica, y continuar con procesamiento normal de otros mensajes.

### RF-SEC-FE (Seguridad de visualizacion en Angular)

- [PROD-SYNC-089-RF-012] El HTML enriquecido de correo debe sanitizarse estrictamente antes de renderizarse (sin scripts/event handlers/iframes no permitidos).
- [PROD-SYNC-089-RF-013] Imagenes externas deben bloquearse por defecto con mecanismo opt-in explicito del usuario.
- [PROD-SYNC-089-RF-014] Enlaces en contenido de correo deben forzar `target="_blank"` y `rel="noopener noreferrer"`.
- [PROD-SYNC-089-RF-015] El render final de correo HTML debe aislarse visualmente con `iframe sandbox` sin permisos de ejecucion de script.

### RF-PRIV (Privacidad y opacidad de errores)

- [PROD-SYNC-089-RF-016] `detail` y `meta` no deben exponer secretos (tokens, passwords, firmas criptograficas).
- [PROD-SYNC-089-RF-017] En `meta._debug` (dev/local), aplicar masking de PII/secrets antes de serializar respuesta.

## Requisitos no funcionales

- [PROD-SYNC-089-RNF-001] Diseno modular por bounded contexts (`connections`, `inbox`, `platform`, `pwa feature stores/components`).
- [PROD-SYNC-089-RNF-002] Errores de negocio expresados con codigos semanticos estables (no acoplados al proveedor).
- [PROD-SYNC-089-RNF-003] Extensibilidad para nuevos providers (ej: iCloud) sin cambios en componentes core de alerta UI.
- [PROD-SYNC-089-RNF-004] Telemetria trazable por tenant, provider, account_id, correlation_id.

## Restriccion de arquitectura confirmada

- [PROD-SYNC-089-ARC-001] No se implementaran DLQ independientes por tenant por costo operativo.
- [PROD-SYNC-089-ARC-002] La resiliencia se implementa con aislamiento logico en ejecucion del worker: limites de recursos por mensaje, timeout/cancelacion por contexto, manejo de panics, y descarte controlado del mensaje problematico para evitar degradacion global.

## Casos de uso

- [PROD-SYNC-089-UC-001] Token revocado en Outlook -> API responde JSON:API con `requires_reauth=true` -> UI muestra alerta con CTA de reconexion.
- [PROD-SYNC-089-UC-002] Rate limit de Yahoo/Microsoft -> API responde `retry_after_seconds=120` -> UI inicia countdown y bloquea reintento.
- [PROD-SYNC-089-UC-003] Mensaje con payload HTML malicioso -> backend limpia/limita, frontend sanitiza y renderiza aislado sin ejecucion de script.
- [PROD-SYNC-089-UC-004] Payload de correo excesivo (zip bomb textual) -> worker falla controladamente ese mensaje y sigue procesando la cola normal.

## Criterios de aceptacion

1. Ningun endpoint de sync retorna errores fuera de JSON:API.
2. En Stage/UAT no se reproducen estados de carga infinita ante fallos de proveedores.
3. Flujos de reautenticacion y countdown de rate limit funcionan con datos del contrato `meta`.
4. Pruebas de seguridad verifican sanitizacion, bloqueo de imagenes externas y enlace seguro.
5. Pruebas de resiliencia demuestran que un mensaje problematico no bloquea el consumo normal de la cola.

## Fuera de alcance

- Implementar DLQ dedicada por tenant.
- Rediseno completo de observabilidad fuera de campos minimos requeridos para trazabilidad de sync.
