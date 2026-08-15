# Product entitlements

## Problem Statement

Bowerbird necesita prender y apagar productos y funciones por tenant, administrado por operadores de plataforma (por encima de toda organización). El primer corte: producto Correo, y la función Enviar, sin cortar la captura de facturas.

## Goals

- [ ] Un operador habilita o deshabilita el producto Correo para un tenant, con vigencia opcional (prueba).
- [ ] Con Correo on, el operador puede apagar Enviar; archivar/etiquetar/leído siguen.
- [ ] Con Correo off, Facturas sigue sincronizando correo entrante.
- [ ] JWT no lleva roles ni features; operador y accesos se resuelven en runtime.

## Out of Scope

| Feature                    | Reason                                       |
| -------------------------- | -------------------------------------------- |
| Toggle de Facturas en UI   | Catálogo sí; UI en corte posterior           |
| Billing / Stripe / seats   | Entitlements son el modelo; cobro es otro BC |
| UI para nominar operadores | `PLATFORM_OPERATOR_EMAILS` basta             |
| Feature flags de deploy    | Otro problema                                |

## Catalog

| Product     | Feature                        | Required | UI                                                |
| ----------- | ------------------------------ | -------- | ------------------------------------------------- |
| `invoicing` | `invoicing.workspace`          | yes      | Facturas                                          |
| `invoicing` | `invoicing.capture_from_email` | yes      | (ingesta; no es UI de Mails)                      |
| `mail`      | `mail.inbox`                   | yes      | Correo: leer el buzón                             |
| `mail`      | `mail.organize`                | yes      | Archivar, etiquetar, leído, destacar              |
| `mail`      | `mail.send`                    | no       | Enviar: redactar/responder en nombre de la cuenta |

Pack por defecto (todos los tenants): invoicing.\* + mail.inbox + mail.organize. Sin mail.send.

## User Stories

### P1: Operador administra accesos ⭐ MVP

**User Story**: Como operador, quiero prender/apagar Correo y Enviar por organización, y dar una prueba con fecha de fin.

**Acceptance Criteria**:

1. WHEN el operador apaga Correo THEN la API de cliente de correo SHALL devolver `ERR_FORBIDDEN` y la nav Mails SHALL ocultarse.
2. WHEN Correo está apagado AND Facturas está activa THEN sync, conexiones e `InboxMessageReceived` SHALL seguir.
3. WHEN Correo está on AND Enviar off THEN `POST /inbox/messages` SHALL ser 403 AND modify/archive SHALL funcionar.
4. WHEN un acceso tiene `ends_at` en el pasado THEN SHALL evaluarse como apagado.
5. WHEN un usuario sin `platform_role=operator` llama `/api/v1/platform/*` THEN SHALL recibir 403 aunque el JWT sea válido.
6. JWT SHALL NOT incluir `platform_role` ni feature keys.

**Independent Test**: Evaluate de trial expirado no incluye `mail.inbox`; ingest `invoicing.capture_from_email` sigue true.
