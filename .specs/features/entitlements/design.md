# Entitlements design

## Bounded contexts

- `identity`: `users.platform_role`; lookup at runtime. JWT is identity only.
- `organization`: tenant lifecycle; on create applies the default pack via a port.
- `entitlements`: in-code catalog + `tenant_entitlements`. Inbox/connections call `Checker`, not the repo.
- `internal/platform` stays infrastructure. HTTP prefix `/api/v1/platform` is the operator facade.

## Evaluation

A grant is effective if `status IN (active, trial)` and `now ∈ [starts_at, ends_at)` (`ends_at` null = open-ended).

## Enforcement

| Surface                        | Feature                                        |
| ------------------------------ | ---------------------------------------------- |
| Inbox list/get/modify/download | `mail.inbox`                                   |
| Inbox send                     | `mail.send`                                    |
| Sync + connections             | `mail.inbox` OR `invoicing.capture_from_email` |

OAuth: without `mail.send`, do not request `gmail.send` / `Mail.Send`.
