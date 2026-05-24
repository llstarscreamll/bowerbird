# Authentication & Multi-Tenancy Specification

## Core Requirements

1. **Authentication Methods:**
   - Google OAuth2 (all environments).
   - Microsoft OAuth2 (all environments).
   - Email/Password (local environment only, for E2E testing). No email verification required for now.
2. **Account Linking:**
   - Transparent linking. If a user registers with a password and later logs in with Google using the same email, the accounts are merged under the same identity.
3. **Session Management:**
   - Mixed approach: Short-lived Access Token (JWT) returned in JSON body + Long-lived Refresh Token in an `HttpOnly`, `Secure`, `SameSite` cookie.
   - Frontend stores Access Token in memory (`SignalStore` in Angular) to prevent XSS.
4. **Multi-Tenancy Lobby & Routing:**
   - After authentication, users go to a Welcome Lobby to select a tenant or create a new one.
   - If the user already belongs to a single tenant, they might be routed directly or still go through the lobby (UX to be refined, but lobby is required for tenant creation).
5. **Data Segregation & Synchronization:**
   - **Control Plane (Identity):**
     - `users`: `id`, `email` (unique), `created_at`, `updated_at`. (No personal data like names or avatars).
     - `user_identities`: `id`, `user_id`, `provider`, `provider_id`, `created_at`.
     - `tenants`: `id`, `name`, `created_at`, `updated_at`, `deleted_at` (localized timestamp for soft-deletes).
     - `tenant_memberships`: `user_id`, `tenant_id`, `role` (`OWNER`, `ADMIN`, `MEMBER`), `created_at` (localized timestamp).
   - **Tenant Database (Profile & Domain Data):**
     - User personal data (First Name, Last Name, Avatar) lives here. Users cannot modify this data in the Control Plane.
     - Note: Custom granular roles within the tenant will be built later. The basic `role` in `tenant_memberships` governs initial access.
6. **Data Privacy (Right to be Forgotten):**
   - **Never delete data.** Use obfuscation and logical deletes.
   - **Leave Tenant:** Remove from `tenant_memberships` in Control Plane, and mark as inactive in the Tenant DB.
   - **Full Account Deletion:**
     - Control Plane: Obfuscate `email` and `user_identities`.
     - Tenant DB: Obfuscate personal data.
     - If the user is the _sole_ `OWNER` of a tenant, the entire tenant is soft-deleted (`deleted_at` populated) to revoke access for everyone.

## Use Cases

- [AUTH-001] User logs in for the first time via Google -> Identity created in Control Plane -> Redirect to Lobby.
- [AUTH-002] User creates a Tenant -> Tenant created in Control Plane -> User gets `OWNER` role in `tenant_memberships` -> Profile created in Tenant DB -> Redirect to Tenant Dashboard.
- [AUTH-003] User logs in, has no tenant -> Redirect to Lobby.
- [AUTH-004] User logs in, has tenant(s) -> Selects tenant from Lobby.
- [AUTH-005] User revokes tenant membership -> Access revoked, marked inactive in Tenant DB.
- [AUTH-006] User deletes entire account -> PII obfuscated, sole-owner tenants soft-deleted.
