# Authentication & Multi-Tenancy Tasks

## Setup & Database

- [ ] 1. **Control Plane Migrations:** Create SQL migrations for `users`, `user_identities`, `tenants`, and `tenant_memberships` (with `deleted_at`, `role`, and `created_at` fields).
- [ ] 2. **Tenant DB Migrations:** Create SQL migrations for tenant-specific `users` profile data.
- [ ] 3. **Go Domain Models:** Define `User`, `UserIdentity`, `Tenant`, `TenantMembership` entities in Go backend.

## Backend Authentication Service

- [ ] 4. **JWT & Refresh Token Logic:** Implement token generation (short-lived access token, long-lived refresh token).
- [ ] 5. **Auth Repository:** Implement PostgreSQL repository for identity management and account linking logic.
- [ ] 6. **Local Auth (Email/Pass):** Implement login/register endpoints for local envs (password hashing, returning tokens in body + HttpOnly cookie).
- [ ] 7. **OAuth2 Integration:** Implement Google and Microsoft OAuth2 callbacks (linking identities or creating them).
- [ ] 8. **Refresh Endpoint:** Implement endpoint to validate HttpOnly cookie and issue new access token.
- [ ] 9. **Auth Middleware:** Implement middleware to validate the JWT in the `Authorization` header.

## Backend Tenant Management (Lobby)

- [ ] 10. **Create Tenant Flow:** Endpoint to create a tenant. Must populate `tenants`, `tenant_memberships` (as `OWNER`), and insert the user's profile in the new Tenant's DB.
- [ ] 11. **List User Tenants:** Endpoint to list all tenants the authenticated user belongs to.
- [ ] 12. **Leave Tenant / Delete Account Logic:** Implement endpoints/services for obfuscation and soft-deletes (`deleted_at`).

## Frontend Integration (Angular)

- [ ] 13. **Auth Service & SignalStore:** Implement `AuthService` handling login, refresh, storing Access Token in memory (`SignalStore`).
- [ ] 14. **Login UI:** Create login page with OAuth buttons and local dev login form.
- [ ] 15. **Lobby UI:** Create lobby page to list available tenants and provide a "Create Tenant" form.
- [ ] 16. **HTTP Interceptor:** Create an Angular interceptor to attach the Bearer token to API calls and handle 401s by calling the refresh endpoint automatically.
