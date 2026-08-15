# Spec — connections (integrations) domain

## Goal

Extract “accounts” (today `ConnectedAccount` inside `inbox`) into a `connections` (or `integrations`) bounded context. It owns third-party integrations (e.g. Gmail, Microsoft), OAuth lifecycle, granted scopes, and tenant sharing policy.

## Requirements

### Connections bounded context

- Manage connection lifecycle: create, authorize, pause, reconnect, revoke.
- Store OAuth credentials encrypted.
- Track `GrantedScopes` for future capabilities (read mail, labels, send, calendar).
- Support `SharingPolicy`: `private` (owner only) or `tenant_all` (all tenant members).

### Decouple inbox

- `inbox` no longer owns OAuth tokens or integrated-account identity.
- `inbox` uses a port (e.g. `ConnectionProvider`) for active connections and credentials when syncing.
- `inbox` keeps `InboxSyncCursor` for sync state, separate from connection identity.

### Provider capabilities (inbox)

- Mail provider client supports `CreateLabel(ctx, userID, labelName)` and `AddLabelToMessage(ctx, userID, messageID, labelID)`.
- Initial OAuth scope must cover this (e.g. `https://www.googleapis.com/auth/gmail.modify` for Google).

### Data privacy (tenant)

- When listing emails via API, enforce the connection’s `SharingPolicy`. If `private`, only `OwnerUserID` can read synced mail for that connection.
