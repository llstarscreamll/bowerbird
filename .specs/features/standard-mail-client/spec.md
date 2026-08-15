# Standard Mail Client Specification

## Problem Statement

El inbox de Bowerbird hoy es un canal de ingesta de facturas: sincroniza Gmail hacia adelante, lista mensajes planos y no permite usar el correo. El producto debe evolucionar a un cliente de correo estándar (carpetas, leído/no leído, hilos, redactar/responder, archivar) sin romper la extracción DIAN existente.

## Goals

- [ ] El usuario navega Inbox, Enviados, Borradores, Archivo, Papelera y Destacados de sus cuentas conectadas.
- [ ] El usuario marca leído/no leído, destaca, archiva o elimina un mensaje y el cambio se refleja en el proveedor.
- [ ] El usuario redacta, responde y envía correo desde Bowerbird.
- [ ] La sincronización es incremental (History/delta), asíncrona y no republica eventos de facturación en re-sync.
- [ ] Gmail y Microsoft Graph sincronizan el mismo modelo de correo.

## Out of Scope

| Feature                                      | Reason                                                          |
| -------------------------------------------- | --------------------------------------------------------------- |
| Yahoo / IMAP genérico / iCloud               | Sin OAuth de connections ni adapter; P3                         |
| Filtros, vacation responder, aliases, snooze | No son MVP de cliente                                           |
| Calendario, contactos, PGP                   | Otro producto                                                   |
| Gmail `users.watch` / Graph subscriptions    | Push queda P3; History + jobs cubren P1                         |
| DLQ por tenant                               | Decisión PROD-SYNC-089                                          |
| Reemplazar el pipeline de facturas           | `InboxMessageReceived` se conserva, solo se emite en alta nueva |

---

## User Stories

### P1: Modelo de correo (folders, flags, destinatarios) ⭐ MVP

**User Story**: Como usuario, quiero ver mis correos organizados por carpeta y estado (leído, destacado) para trabajar el buzón como en Gmail/Outlook.

**Why P1**: Sin modelo local de folders/flags el resto de acciones no tiene dónde persistir.

**Acceptance Criteria**:

1. WHEN el sync descarga un mensaje THEN el sistema SHALL persistir folder (`inbox|sent|drafts|trash|spam|archive`), `is_read`, `is_starred`, `is_draft`, To/Cc/Bcc y `provider_thread_id`.
2. WHEN Gmail incluye label `UNREAD` THEN el sistema SHALL guardar `is_read=false`.
3. WHEN el usuario lista mensajes con `folder=inbox` THEN el sistema SHALL devolver solo mensajes de esa carpeta, paginados.
4. WHEN una connection es `private` THEN el sistema SHALL ocultar sus mensajes a usuarios que no son el owner.

**Independent Test**: Sync de un mensaje Gmail con `INBOX`+`UNREAD` produce una fila con `folder=inbox` e `is_read=false`; `GET /messages?folder=inbox` lo lista.

---

### P1: Acciones bidireccionales ⭐ MVP

**User Story**: Como usuario, quiero marcar leído, destacar, archivar y enviar a papelera para que el cambio exista también en Gmail/Outlook.

**Why P1**: Un cliente de correo que no escribe de vuelta al proveedor es un visor.

**Acceptance Criteria**:

1. WHEN el usuario marca leído/no leído THEN el sistema SHALL actualizar flags locales y llamar `ModifyMessage` en el proveedor.
2. WHEN el usuario destaca o quita estrella THEN el sistema SHALL sincronizar `STARRED` (Gmail) o `flag.flagStatus` (Graph).
3. WHEN el usuario archiva THEN el sistema SHALL quitar INBOX (Gmail) o mover a Archive (Graph) y setear `folder=archive`.
4. WHEN el usuario envía a papelera THEN el sistema SHALL trash en el proveedor y setear `folder=trash`.
5. WHEN el proveedor no acepta la acción (token revocado) THEN el sistema SHALL devolver error JSON:API de sync/reauth y no dejar el estado local a medias si el proveedor falló.

**Independent Test**: Command de archive con fake provider registra `removeLabelIds=[INBOX]` y el mensaje queda `folder=archive`.

---

### P1: Redactar y enviar ⭐ MVP

**User Story**: Como usuario, quiero redactar un correo nuevo o responder uno existente y enviarlo desde Bowerbird.

**Why P1**: Sin envío no es un cliente de correo.

**Acceptance Criteria**:

1. WHEN el usuario envía un mensaje con To y asunto THEN el sistema SHALL llamar `SendMessage` del proveedor con scopes de envío.
2. WHEN el envío tiene éxito THEN el sistema SHALL persistir una copia en `folder=sent` (o esperar el próximo sync de SENT).
3. WHEN To está vacío THEN el sistema SHALL rechazar con `ERR_VALIDATION`.
4. WHEN el usuario responde THEN el sistema SHALL incluir `In-Reply-To` / `thread_id` del mensaje origen.

**Independent Test**: `POST /messages` con To válido dispara `SendMessage` en el fake provider.

---

### P1: OAuth scopes de cliente ⭐ MVP

**User Story**: Como usuario, quiero autorizar lectura, modificación y envío en una sola conexión para no reconectar al usar el cliente.

**Why P1**: `gmail.readonly` en el HTTP client impide modify/send aunque connections ya pida `gmail.modify`.

**Acceptance Criteria**:

1. WHEN se inicia OAuth Google de connections THEN el sistema SHALL pedir `email`, `gmail.modify` y `gmail.send`.
2. WHEN el adapter Gmail construye el HTTP client THEN SHALL usar esos mismos scopes (no `gmail.readonly`).
3. WHEN se inicia OAuth Microsoft de connections THEN el sistema SHALL pedir `User.Read`, `Mail.ReadWrite`, `Mail.Send`, `offline_access`.
4. WHEN `GrantedScopes` se persisten THEN SHALL reflejar los scopes realmente solicitados.

**Independent Test**: Config OAuth de connections incluye `gmail.send`; test del oauth client Gmail no usa `gmail.readonly`.

---

### P1: Sync fiable (History, jobs, idempotencia) ⭐ MVP

**User Story**: Como operador, quiero que el sync no bloquee HTTP, no pierda mensajes y no reprocese facturas.

**Why P1**: El cursor `after:unix` y el publish en cada upsert rompen un cliente real y el pipeline DIAN.

**Acceptance Criteria**:

1. WHEN existe `history_id` en el cursor Gmail THEN el sistema SHALL usar History API en lugar de `after:unix`.
2. WHEN History responde 404 (id expirado) THEN the sistema SHALL hacer fallback a listado incremental y guardar un history id nuevo.
3. WHEN `UpsertInboxMessage` no inserta (ya existía) THEN the sistema SHALL NO publicar `InboxMessageReceived`.
4. WHEN el usuario dispara `POST /inbox/sync` THEN the sistema SHALL encolar un job SQS por cuenta y responder 202 sin ejecutar el sync inline.
5. WHEN el job corre THEN SHALL actualizar `inbox_sync_cursors.status`.

**Independent Test**: Segundo sync del mismo provider id no incrementa eventos publicados.

---

### P1: Adjuntos reales y lista usable ⭐ MVP

**User Story**: Como usuario, quiero ver el nombre real de los adjuntos, descargarlos y paginar la bandeja.

**Why P1**: La UI actual muestra placeholders y carga todos los mensajes en memoria.

**Acceptance Criteria**:

1. WHEN el detalle de un mensaje tiene adjuntos THEN the sistema SHALL devolver id, filename, mime y size reales.
2. WHEN el usuario descarga un adjunto THEN the sistema SHALL servir el objeto de S3 autenticado.
3. WHEN la lista pide `limit`/`offset` THEN the sistema SHALL paginar en SQL, no en el cliente.
4. WHEN hay búsqueda `q` THEN the sistema SHALL filtrar en servidor por subject/sender/snippet.

**Independent Test**: `GET /messages?limit=1` con dos filas devuelve un item y `total=2`.

---

### P2: Hilos en UI

**User Story**: Como usuario, quiero ver una conversación agrupada por `provider_thread_id`.

**Why P2**: El modelo ya guarda thread id; agrupar en lista/detalle mejora UX pero no bloquea enviar/archivar.

**Acceptance Criteria**:

1. WHEN varios mensajes comparten `thread_id` THEN the lista MAY agruparlos mostrando el más reciente y un conteo.
2. WHEN el usuario abre un hilo THEN the sistema SHALL listar los mensajes del thread ordenados por fecha.

---

### P2: Microsoft Graph (conexión + adapter)

**User Story**: Como usuario de Outlook, quiero conectar Microsoft y usar el mismo cliente.

**Why P2**: El modelo y puertos son provider-agnostic; Graph es el segundo adapter.

**Acceptance Criteria**:

1. WHEN el usuario pide `GET /connections/microsoft` THEN recibe `auth_url` de Azure AD.
2. WHEN el callback guarda la connection THEN `provider=microsoft` y el factory construye un cliente Graph.
3. WHEN sync corre para Microsoft THEN lista/get/download/modify/send funcionan contra Graph.

---

### P3: Push (Gmail watch / Graph subscriptions)

Fuera del MVP. History + jobs cubren latencia aceptable.

---

## Edge Cases

- WHEN un mensaje está en TRASH e INBOX a la vez THEN folder SHALL ser `trash`.
- WHEN History omite un mensaje THEN el fallback `after:` no debe duplicar filas (`ON CONFLICT account_id, provider_message_id`).
- WHEN SendMessage falla AFTER persistir local THEN SHALL devolver error y no marcar sent.
- WHEN el HTML del correo se muestra THEN SHALL seguir las reglas de sanitización PROD-SYNC-089.
- WHEN sharing_policy=private THEN list/get/download/modify/send SHALL aplicar el mismo filtro de owner.

---

## Requirement Traceability

| Requirement ID | Story                             | Phase   | Status       |
| -------------- | --------------------------------- | ------- | ------------ |
| MAIL-01        | P1: Modelo folders/flags          | Execute | Implementing |
| MAIL-02        | P1: Acciones bidireccionales      | Execute | Implementing |
| MAIL-03        | P1: Redactar y enviar             | Execute | Implementing |
| MAIL-04        | P1: OAuth scopes                  | Execute | Implementing |
| MAIL-05        | P1: History + jobs + idempotencia | Execute | Implementing |
| MAIL-06        | P1: Adjuntos y paginación         | Execute | Implementing |
| MAIL-07        | P2: Hilos UI                      | Execute | Implementing |
| MAIL-08        | P2: Microsoft Graph               | Execute | Implementing |

**ID format:** `MAIL-NN`

**Coverage:** 8 total

---

## Success Criteria

- [ ] Un usuario con Gmail puede leer, destacar, archivar, borrar y enviar un correo desde la PWA.
- [ ] Re-sync no dispara un segundo `InboxMessageReceived` para el mismo provider message id.
- [ ] `POST /inbox/sync` responde 202 y el trabajo corre en SQS.
- [ ] Scopes Gmail incluyen modify+send; Microsoft connections existe con Mail.ReadWrite+Mail.Send.
