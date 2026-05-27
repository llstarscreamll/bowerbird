# Specification: Connections (Integrations) Domain

## Goal

Extract the concept of "accounts" (currently `ConnectedAccount` inside `inbox`) into an independent bounded context called `connections` (or `integrations`). This new domain will manage user-configured third-party integrations (like Gmail or Microsoft), their authentication lifecycle (OAuth), the granted scopes, and sharing policies within a tenant.

## Requirements

1. **New Connections Bounded Context:**
   - Manage the lifecycle of a connection (create, authorize, pause, reconnect, revoke).
   - Store OAuth credentials securely (encrypted).
   - Track `GrantedScopes` to support future capabilities (e.g., read email, create labels, send email, read calendar).
   - Support a `SharingPolicy`: The owner of the connection can decide if the connection is `private` (only visible to them) or `tenant_all` (visible to all tenant members).

2. **Decouple Inbox:**
   - `inbox` must no longer manage OAuth tokens or the concept of an integrated account.
   - `inbox` will use a port (e.g., `ConnectionProvider`) to request active connections and their credentials from the `connections` domain in order to perform sync.
   - `inbox` will maintain an `InboxSyncCursor` to track the state of synchronization for a given connection, entirely separated from the connection's identity.

3. **New Provider Capabilities (Inbox):**
   - The mail provider client must support `CreateLabel(ctx, userID, labelName)` and `AddLabelToMessage(ctx, userID, messageID, labelID)`.
   - The initial scope requested when establishing a connection must be broad enough to support this (e.g., `https://www.googleapis.com/auth/gmail.modify` for Google).

4. **Data Privacy (Tenant-level):**
   - When fetching emails via the API, the system must enforce the `SharingPolicy` of the underlying connection. If a connection is `private`, only the `OwnerUserID` can read its synced emails.
