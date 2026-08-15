# Spec — authentication and multi-tenancy

## Core requirements

### Authentication methods

- Google OAuth2 (all environments)
- Microsoft OAuth2 (all environments)
- Email/password (local only, for E2E); no email verification for now

### Account linking

Transparent merge: password signup then Google login with the same email shares one identity.

### Session management

- Short-lived access JWT in JSON body
- Long-lived refresh token in `HttpOnly`, `Secure`, `SameSite` cookie
- Frontend keeps access token in memory (`SignalStore`) to reduce XSS risk

### Multi-tenant lobby and routing

After auth, users land in a Welcome Lobby to pick or create a tenant. Single-tenant auto-route is optional UX; lobby remains required for tenant creation.

### Data segregation

**Control plane (identity)**

| Table                | Key fields                                                                  |
| -------------------- | --------------------------------------------------------------------------- |
| `users`              | `id`, `email` (unique), timestamps — no names/avatars                       |
| `user_identities`    | `id`, `user_id`, `provider`, `provider_id`, `created_at`                    |
| `tenants`            | `id`, `name`, timestamps, `deleted_at` (soft delete)                        |
| `tenant_memberships` | `user_id`, `tenant_id`, `role` (`OWNER` / `ADMIN` / `MEMBER`), `created_at` |

**Tenant DB (profile and domain)**

- First name, last name, avatar live here (not editable via control plane).
- Granular in-tenant roles come later; `tenant_memberships.role` covers initial access.

### Data privacy (right to be forgotten)

Never hard-delete; obfuscate and soft-delete.

- **Leave tenant:** remove `tenant_memberships`; mark inactive in tenant DB.
- **Full account deletion:** obfuscate control-plane `email` / `user_identities` and tenant PII. If the user is the sole `OWNER`, soft-delete that tenant (`deleted_at`) so all access ends.

## Use cases

- [AUTH-001] First Google login → identity in control plane → Lobby.
- [AUTH-002] Create tenant → control plane tenant + `OWNER` membership → tenant DB profile → dashboard.
- [AUTH-003] Login with no tenant → Lobby.
- [AUTH-004] Login with tenant(s) → select from Lobby.
- [AUTH-005] Revoke membership → access gone; inactive in tenant DB.
- [AUTH-006] Delete account → PII obfuscated; sole-owner tenants soft-deleted.
