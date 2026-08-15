# Design — connections domain and inbox refactor

## Architecture

New `connections` bounded context; `inbox` depends on it via an ACL/port.

### `connections` bounded context

Domain: `apps/backend/internal/connections`

**Aggregate: `Connection`**

| Field                    | Type / notes                             |
| ------------------------ | ---------------------------------------- |
| `ID`                     | ULID                                     |
| `TenantID`               | ULID (tenant middleware; enforced)       |
| `OwnerUserID`            | ULID (user who connected)                |
| `Provider`               | `gmail`, `microsoft`                     |
| `ProviderAccountEmail`   | e.g. `user@gmail.com`                    |
| `EncryptedCredentials`   | byte array                               |
| `Status`                 | `active`, `requires_reconnect`, `paused` |
| `GrantedScopes`          | string array                             |
| `SharingPolicy`          | `private`, `tenant_all`                  |
| `CreatedAt`, `UpdatedAt` | time                                     |

**Internal API (port for other domains):**

```go
type InternalService interface {
    GetActiveConnections(ctx context.Context) ([]ConnectionInfo, error)
    DecryptCredentials(ctx context.Context, connectionID string) ([]byte, error)
    MarkRequiresReconnect(ctx context.Context, connectionID string, reason string) error
    GetSharingPolicy(ctx context.Context, connectionID string) (SharingPolicy, error)
}
```

**Events:** `ConnectionEstablished` when a user successfully connects an account.

### `inbox` refactor

**Domain**

- Remove `ConnectedAccount` from `inbox/domain/models.go`.
- Add `InboxSyncCursor`: `ConnectionID`, `LastSyncedAt`, `LastError`, `Status` (`syncing`, `idle`, `error`).
- Extend `MailProviderClient` with `CreateLabel` and `AddLabelToMessage`.

**Use cases**

- `SyncAccountsUseCase` queries `connections.InternalService` for active connections and credentials, loads each `InboxSyncCursor`, then syncs.
- On auth failure, call `MarkRequiresReconnect`.

### Database schema

Migration plan:

1. Rename `connected_accounts` → `connections`.
2. Add columns:
   - `owner_user_id CHAR(26)` (nullable initially for backfill)
   - `sharing_policy VARCHAR(50) DEFAULT 'tenant_all'`
   - `granted_scopes JSONB DEFAULT '[]'::jsonb`
3. Create `inbox_sync_cursors`:
   - `connection_id CHAR(26) PRIMARY KEY REFERENCES connections(id) ON DELETE CASCADE`
   - `last_synced_at TIMESTAMP WITH TIME ZONE`
   - `last_error TEXT`
   - `status VARCHAR(30) DEFAULT 'idle'`

**Data migration:** populate `inbox_sync_cursors` from existing `last_synced_at` / `last_error` on the old table, then drop those columns from `connections` (same transaction or follow-up).

### Cross-domain access control

In inbox HTTP handlers (e.g. `ListMessages`): load sharing policies; if `private` and current user ≠ `OwnerUserID`, filter out that connection’s messages.
