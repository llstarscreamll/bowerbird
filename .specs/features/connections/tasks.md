# Implementation Tasks: Connections Domain & Inbox Refactoring

## Phase 1: Database Migrations

- [ ] 1.1 Create migration to rename `connected_accounts` to `connections`.
- [ ] 1.2 Add `owner_user_id`, `sharing_policy`, and `granted_scopes` columns to `connections`.
- [ ] 1.3 Create `inbox_sync_cursors` table.
- [ ] 1.4 Write data migration script (in SQL) to populate `inbox_sync_cursors` from existing `connections` data (`last_synced_at`, `last_error`, `status`).
- [ ] 1.5 Drop `last_synced_at` and `last_error` from `connections`.

## Phase 2: Connections Domain Foundation

- [ ] 2.1 Create `apps/backend/internal/connections` folder structure (application, domain, infrastructure, presentation).
- [ ] 2.2 Define `Connection`, `SharingPolicy`, `GrantedScopes` domain models.
- [ ] 2.3 Implement `postgres_repository.go` for the `connections` domain.
- [ ] 2.4 Move OAuth handler logic (Google login, callback) from `identity` or `inbox` into `connections/presentation/http`.
- [ ] 2.5 Update OAuth URL generation to request `https://www.googleapis.com/auth/gmail.modify`.
- [ ] 2.6 Implement `InternalService` interface to expose connection data to other domains.

## Phase 3: Inbox Domain Refactoring

- [ ] 3.1 Define `InboxSyncCursor` model in `inbox/domain/models.go`.
- [ ] 3.2 Update `inbox/domain/provider_client.go` to include `CreateLabel` and `AddLabelToMessage`.
- [ ] 3.3 Implement `CreateLabel` and `AddLabelToMessage` in Gmail client (`infrastructure/provider/gmail/client.go`).
- [ ] 3.4 Refactor `SyncAccountsUseCase` to use `connections.InternalService` instead of the old `ConnectedAccount` model.
- [ ] 3.5 Update `inbox` Postgres repository to read/write from `inbox_sync_cursors` instead of `connected_accounts`.
- [ ] 3.6 Ensure auth failures during sync invoke `InternalService.MarkRequiresReconnect`.

## Phase 4: API and UI Integration

- [ ] 4.1 Update the Inbox HTTP Handlers (`ListMessages`, etc.) to enforce `SharingPolicy` by checking `owner_user_id` against the current user's ID.
- [ ] 4.2 Expose HTTP endpoints in `connections` to list connections and update their `SharingPolicy`.
- [ ] 4.3 Ensure `router.go` or `main.go` wires up the new `connections` domain and injects `InternalService` into `inbox`.
