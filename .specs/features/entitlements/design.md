# Entitlements design

## Bounded contexts

- `identity`: `users.platform_role`; lookup en runtime. JWT solo identidad.
- `organization`: ciclo de vida del tenant; al crear aplica el pack por defecto vía puerto.
- `entitlements`: catálogo en código + `tenant_entitlements`. Inbox/connections llaman `Checker`, no el repo.
- `internal/platform` permanece infraestructura. Prefijo HTTP `/api/v1/platform` es fachada del operador.

## Evaluation

Un grant es efectivo si `status IN (active, trial)` y `now ∈ [starts_at, ends_at)` (`ends_at` null = abierto).

## Enforcement

| Surface                        | Feature                                        |
| ------------------------------ | ---------------------------------------------- |
| Inbox list/get/modify/download | `mail.inbox`                                   |
| Inbox send                     | `mail.send`                                    |
| Sync + connections             | `mail.inbox` OR `invoicing.capture_from_email` |

OAuth: sin `mail.send` no se piden `gmail.send` / `Mail.Send`.
