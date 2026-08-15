# Tasks — connections domain and inbox refactor

## Phase 1: Database migrations

- [ ] 1.1 Create migration to rename `connected_accounts` to `connections`.
- [ ] 1.2 Add `owner_user_id`, `sharing_policy`, and `granted_scopes` to `connections`.
- [ ] 1.3 Create `inbox_sync_cursors` table.
- [ ] 1.4 SQL data migration: populate `inbox_sync_cursors` from existing connection sync fields (`last_synced_at`, `last_error`, `status`).
- [ ] 1.5 Drop `last_synced_at` and `last_error` from `connections`.

## Phase 2: Connections domain foundation

- [ ] 2.1 Create `apps/backend/internal/connections` layout (application, domain, infrastructure, presentation).
- [ ] 2.2 Define `Connection`, `SharingPolicy`, `GrantedScopes` domain models.
- [ ] 2.3 Implement `postgres_repository.go` for connections.
- [ ] 2.4 Move OAuth handler logic (Google login, callback) from `identity`/`inbox` into `connections/presentation/http`.
- [ ] 2.5 Request `https://www.googleapis.com/auth/gmail.modify` in OAuth URL generation.
- [ ] 2.6 Implement `InternalService` for other domains.

## Phase 3: Inbox domain refactoring

- [ ] 3.1 Define `InboxSyncCursor` in `inbox/domain/models.go`.
- [ ] 3.2 Add `CreateLabel` and `AddLabelToMessage` to `inbox/domain/provider_client.go`.
- [ ] 3.3 Implement those methods in the Gmail client (`infrastructure/provider/gmail/client.go`).
- [ ] 3.4 Refactor `SyncAccountsUseCase` to use `connections.InternalService` instead of `ConnectedAccount`.
- [ ] 3.5 Point inbox Postgres repository at `inbox_sync_cursors` instead of `connected_accounts`.
- [ ] 3.6 On sync auth failure, call `InternalService.MarkRequiresReconnect`.

## Phase 4: API and UI integration

- [ ] 4.1 Enforce `SharingPolicy` in inbox HTTP handlers (`ListMessages`, etc.) via `owner_user_id` vs current user.
- [ ] 4.2 Expose connections HTTP endpoints to list connections and update `SharingPolicy`.
- [ ] 4.3 Wire `connections` in `router.go`/`main.go` and inject `InternalService` into `inbox`.
