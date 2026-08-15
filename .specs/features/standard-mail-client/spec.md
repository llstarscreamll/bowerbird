# Standard mail client specification

## Problem statement

Bowerbird's inbox is an invoice-ingestion channel: forward Gmail sync, flat message lists, no real mail use. The product must become a standard mail client (folders, read/unread, threads, compose/reply, archive) without breaking existing DIAN extraction.

## Goals

- [ ] User navigates Inbox, Sent, Drafts, Archive, Trash, and Starred across connected accounts.
- [ ] User marks read/unread, stars, archives, or deletes a message; the change syncs to the provider.
- [ ] User composes, replies, and sends mail from Bowerbird.
- [ ] Sync is incremental (History/delta), async, and does not republish invoicing events on re-sync.
- [ ] Gmail and Microsoft Graph sync the same mail model.

## Out of scope

| Feature                                      | Reason                                               |
| -------------------------------------------- | ---------------------------------------------------- |
| Yahoo / generic IMAP / iCloud                | No OAuth connections or adapter; P3                  |
| Filters, vacation responder, aliases, snooze | Not client MVP                                       |
| Calendar, contacts, PGP                      | Separate product                                     |
| Gmail `users.watch` / Graph subscriptions    | Push is P3; History + jobs cover P1                  |
| Per-tenant DLQ                               | PROD-SYNC-089 decision                               |
| Replace invoice pipeline                     | Keep `InboxMessageReceived`; emit only on new insert |

---

## User stories

### P1: Mail model (folders, flags, recipients) ⭐ MVP

**User story**: As a user, I want mail organized by folder and state (read, starred) so I can work the mailbox like Gmail/Outlook.

**Why P1**: Without a local folders/flags model, other actions have nowhere to persist.

**Acceptance criteria**:

1. WHEN sync downloads a message THEN the system SHALL persist folder (`inbox|sent|drafts|trash|spam|archive`), `is_read`, `is_starred`, `is_draft`, To/Cc/Bcc, and `provider_thread_id`.
2. WHEN Gmail includes label `UNREAD` THEN the system SHALL store `is_read=false`.
3. WHEN the user lists messages with `folder=inbox` THEN the system SHALL return only that folder's messages, paginated.
4. WHEN a connection is `private` THEN the system SHALL hide its messages from non-owners.

**Independent test**: Sync a Gmail message with `INBOX`+`UNREAD` → row with `folder=inbox` and `is_read=false`; `GET /messages?folder=inbox` lists it.

---

### P1: Bidirectional actions ⭐ MVP

**User story**: As a user, I want to mark read, star, archive, and trash so the change also exists in Gmail/Outlook.

**Why P1**: A mail client that does not write back to the provider is a viewer.

**Acceptance criteria**:

1. WHEN the user marks read/unread THEN the system SHALL update local flags and call `ModifyMessage` on the provider.
2. WHEN the user stars or unstars THEN the system SHALL sync `STARRED` (Gmail) or `flag.flagStatus` (Graph).
3. WHEN the user archives THEN the system SHALL remove INBOX (Gmail) or move to Archive (Graph) and set `folder=archive`.
4. WHEN the user trashes THEN the system SHALL trash on the provider and set `folder=trash`.
5. WHEN the provider rejects the action (revoked token) THEN the system SHALL return a JSON:API sync/reauth error and not leave local state half-applied if the provider failed.

**Independent test**: Archive command with fake provider records `removeLabelIds=[INBOX]` and message is `folder=archive`.

---

### P1: Compose and send ⭐ MVP

**User story**: As a user, I want to compose a new message or reply and send from Bowerbird.

**Why P1**: Without send it is not a mail client.

**Acceptance criteria**:

1. WHEN the user sends a message with To and subject THEN the system SHALL call provider `SendMessage` with send scopes.
2. WHEN send succeeds THEN the system SHALL persist a copy in `folder=sent` (or wait for the next SENT sync).
3. WHEN To is empty THEN the system SHALL reject with `ERR_VALIDATION`.
4. WHEN the user replies THEN the system SHALL include `In-Reply-To` / `thread_id` from the source message.

**Independent test**: `POST /messages` with valid To triggers `SendMessage` on the fake provider.

---

### P1: Client OAuth scopes ⭐ MVP

**User story**: As a user, I want to authorize read, modify, and send in one connection so I do not reconnect to use the client.

**Why P1**: `gmail.readonly` on the HTTP client blocks modify/send even when connections already request `gmail.modify`.

**Acceptance criteria**:

1. WHEN Google connections OAuth starts THEN the system SHALL request `email`, `gmail.modify`, and `gmail.send`.
2. WHEN the Gmail adapter builds the HTTP client THEN it SHALL use those same scopes (not `gmail.readonly`).
3. WHEN Microsoft connections OAuth starts THEN the system SHALL request `User.Read`, `Mail.ReadWrite`, `Mail.Send`, `offline_access`.
4. WHEN `GrantedScopes` are persisted THEN they SHALL reflect the scopes actually requested.

**Independent test**: Connections OAuth config includes `gmail.send`; Gmail oauth client test does not use `gmail.readonly`.

---

### P1: Reliable sync (History, jobs, idempotency) ⭐ MVP

**User story**: As an operator, I want sync that does not block HTTP, drop messages, or reprocess invoices.

**Why P1**: The `after:unix` cursor and publish-on-every-upsert break a real client and the DIAN pipeline.

**Acceptance criteria**:

1. WHEN a Gmail cursor has `history_id` THEN the system SHALL use the History API instead of `after:unix`.
2. WHEN History returns 404 (expired id) THEN the system SHALL fall back to incremental list and store a new history id.
3. WHEN `UpsertInboxMessage` does not insert (already existed) THEN the system SHALL NOT publish `InboxMessageReceived`.
4. WHEN the user triggers `POST /inbox/sync` THEN the system SHALL enqueue one SQS job per account and respond 202 without inline sync.
5. WHEN the job runs THEN it SHALL update `inbox_sync_cursors.status`.

**Independent test**: Second sync of the same provider id does not increase published events.

---

### P1: Real attachments and usable list ⭐ MVP

**User story**: As a user, I want real attachment names, downloads, and a paginated inbox.

**Why P1**: Current UI shows placeholders and loads all messages in memory.

**Acceptance criteria**:

1. WHEN message detail has attachments THEN the system SHALL return real id, filename, mime, and size.
2. WHEN the user downloads an attachment THEN the system SHALL serve the authenticated S3 object.
3. WHEN the list requests `limit`/`offset` THEN the system SHALL paginate in SQL, not the client.
4. WHEN search `q` is present THEN the system SHALL filter server-side by subject/sender/snippet.

**Independent test**: `GET /messages?limit=1` with two rows returns one item and `total=2`.

---

### P2: Threads in UI

**User story**: As a user, I want a conversation grouped by `provider_thread_id`.

**Why P2**: The model already stores thread id; grouping improves UX but does not block send/archive.

**Acceptance criteria**:

1. WHEN several messages share `thread_id` THEN the list MAY group them showing the latest and a count.
2. WHEN the user opens a thread THEN the system SHALL list thread messages ordered by date.

---

### P2: Microsoft Graph (connection + adapter)

**User story**: As an Outlook user, I want to connect Microsoft and use the same client.

**Why P2**: Model and ports are provider-agnostic; Graph is the second adapter.

**Acceptance criteria**:

1. WHEN the user requests `GET /connections/microsoft` THEN they receive an Azure AD `auth_url`.
2. WHEN the callback saves the connection THEN `provider=microsoft` and the factory builds a Graph client.
3. WHEN sync runs for Microsoft THEN list/get/download/modify/send work against Graph.

---

### P3: Push (Gmail watch / Graph subscriptions)

Out of MVP. History + jobs cover acceptable latency.

---

## Edge cases

- WHEN a message is in TRASH and INBOX THEN folder SHALL be `trash`.
- WHEN History omits a message THEN the `after:` fallback MUST NOT duplicate rows (`ON CONFLICT account_id, provider_message_id`).
- WHEN SendMessage fails AFTER local persist THEN SHALL return error and not mark sent.
- WHEN mail HTML is shown THEN SHALL follow PROD-SYNC-089 sanitization rules.
- WHEN `sharing_policy=private` THEN list/get/download/modify/send SHALL apply the same owner filter.

---

## Requirement traceability

| Requirement ID | Story                            | Phase   | Status       |
| -------------- | -------------------------------- | ------- | ------------ |
| MAIL-01        | P1: Folders/flags model          | Execute | Implementing |
| MAIL-02        | P1: Bidirectional actions        | Execute | Implementing |
| MAIL-03        | P1: Compose and send             | Execute | Implementing |
| MAIL-04        | P1: OAuth scopes                 | Execute | Implementing |
| MAIL-05        | P1: History + jobs + idempotency | Execute | Implementing |
| MAIL-06        | P1: Attachments and pagination   | Execute | Implementing |
| MAIL-07        | P2: Threads UI                   | Execute | Implementing |
| MAIL-08        | P2: Microsoft Graph              | Execute | Implementing |

**ID format:** `MAIL-NN`

**Coverage:** 8 total

---

## Success criteria

- [ ] A Gmail user can read, star, archive, delete, and send mail from the PWA.
- [ ] Re-sync does not fire a second `InboxMessageReceived` for the same provider message id.
- [ ] `POST /inbox/sync` returns 202 and work runs on SQS.
- [ ] Gmail scopes include modify+send; Microsoft connections exist with Mail.ReadWrite+Mail.Send.
