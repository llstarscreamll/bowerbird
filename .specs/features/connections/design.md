# Design: Connections Domain & Inbox Refactoring

## Architecture

This feature introduces a new bounded context `connections` and refactors `inbox` to depend on it via an anti-corruption layer / port.

### 1. `connections` Bounded Context

**Domain:** `apps/backend/internal/connections`

**Aggregates:**

- `Connection`
  - `ID`: ULID
  - `TenantID`: ULID (Implicit via tenant middleware context, but enforced)
  - `OwnerUserID`: ULID (User who initiated the connection)
  - `Provider`: String (`gmail`, `microsoft`)
  - `ProviderAccountEmail`: String (e.g., `user@gmail.com`)
  - `EncryptedCredentials`: Byte Array
  - `Status`: String (`active`, `requires_reconnect`, `paused`)
  - `GrantedScopes`: String Array (e.g., `["https://www.googleapis.com/auth/gmail.modify"]`)
  - `SharingPolicy`: String (`private`, `tenant_all`)
  - `CreatedAt`, `UpdatedAt`: Time

**Internal API (Port exposed to other domains):**

```go
type InternalService interface {
    GetActiveConnections(ctx context.Context) ([]ConnectionInfo, error)
    DecryptCredentials(ctx context.Context, connectionID string) ([]byte, error)
    MarkRequiresReconnect(ctx context.Context, connectionID string, reason string) error
    GetSharingPolicy(ctx context.Context, connectionID string) (SharingPolicy, error)
}
```

**Events:**

- `ConnectionEstablished` (Emitted when a user successfully connects an account).

### 2. `inbox` Bounded Context Refactoring

**Domain Changes:**

- Remove `ConnectedAccount` from `inbox/domain/models.go`.
- Introduce `InboxSyncCursor`:
  - `ConnectionID`: ULID
  - `LastSyncedAt`: Time
  - `LastError`: String
  - `Status`: String (`syncing`, `idle`, `error`)
- Update `MailProviderClient` to include:
  - `CreateLabel(ctx context.Context, userID, labelName string) (string, error)`
  - `AddLabelToMessage(ctx context.Context, userID, messageID, labelID string) error`

**Use Cases:**

- `SyncAccountsUseCase` will be updated to query `connections.InternalService` to get a list of active connections and their credentials.
- It will then iterate over them, fetching the `InboxSyncCursor` for each, and syncing messages.
- If authentication fails, it calls `connections.InternalService.MarkRequiresReconnect`.

### 3. Database Schema

**Migration:**
We will rename the existing `connected_accounts` table to `connections` and add the new columns (`owner_user_id`, `sharing_policy`, `granted_scopes`). We will also create a new `inbox_sync_cursors` table for the inbox domain to track sync state.

1. Rename `connected_accounts` -> `connections`.
2. Add columns:
   - `owner_user_id CHAR(26)` (Nullable initially, to populate via a script if needed, or set to a default admin).
   - `sharing_policy VARCHAR(50) DEFAULT 'tenant_all'`.
   - `granted_scopes JSONB DEFAULT '[]'::jsonb`.
3. Create table `inbox_sync_cursors`:
   - `connection_id CHAR(26) PRIMARY KEY REFERENCES connections(id) ON DELETE CASCADE`.
   - `last_synced_at TIMESTAMP WITH TIME ZONE`.
   - `last_error TEXT`.
   - `status VARCHAR(30) DEFAULT 'idle'`.

_Data Migration Strategy:_ We will populate `inbox_sync_cursors` using the existing `last_synced_at` and `last_error` from the old `connected_accounts` table. Then we can drop those columns from `connections` in a follow-up step or the same transaction.

### 4. Cross-Domain Access Control

In the `inbox` HTTP Handlers (e.g., `ListMessages`):

- Before returning messages, the handler (or usecase) will fetch the sharing policies of the connections.
- If `SharingPolicy == "private"` and `CurrentUser.ID != Connection.OwnerUserID`, the messages associated with that connection will be filtered out of the response.
