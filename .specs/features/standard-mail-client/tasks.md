# Standard mail client tasks

**Spec**: `.specs/features/standard-mail-client/spec.md`
**Design**: `.specs/features/standard-mail-client/design.md`

## Execution plan

1. Domain + migration (MAIL-01)
2. Provider port + Gmail write/history (MAIL-02, MAIL-04, MAIL-05)
3. Sync idempotency + SQS (MAIL-05)
4. Queries/HTTP/PWA (MAIL-02, MAIL-03, MAIL-06, MAIL-07)
5. Microsoft Graph + connections OAuth (MAIL-08)

---

### T1: Migration mail model

- **What**: Columns folder/flags/recipients/snippet + cursor history_id
- **Where**: `apps/backend/migrations/tenant/000009_standard_mail_client.{up,down}.sql`
- **Depends on**: —
- **Done when**: Migration applies; down reverses
- **Req**: MAIL-01

### T2: Domain InboxMessage + SyncCursor + mapping

- **What**: Extend aggregates; map Gmail/Graph labels to folder/flags
- **Where**: `inbox/domain/*`
- **Depends on**: T1
- **Done when**: Unit tests for mapping and constructors
- **Req**: MAIL-01

### T3: MailProviderClient write/history/send

- **What**: Extend port; implement Gmail History/Modify/Trash/Send; parse To/Cc
- **Where**: `provider_client.go`, `gmail/client.go`, `gmail/oauth_client.go`
- **Depends on**: T2
- **Done when**: httptest tests for modify/send/history; scopes without readonly
- **Req**: MAIL-02, MAIL-03, MAIL-04, MAIL-05

### T4: Sync History + idempotent events

- **What**: Use HistoryID; publish event only if inserted
- **Where**: `commands/sync_account.go` + tests
- **Depends on**: T3
- **Done when**: Re-sync test does not publish a second event; history path covered
- **Req**: MAIL-05

### T5: SQS sync dispatcher

- **What**: Job `InboxSyncAccount`; processor; wire into API poller
- **Where**: `inbox/contracts/jobs`, `adapters/jobs`, `commands/sqs_sync_dispatcher.go`, `cmd/api/main.go`, `inbox/wire.go`
- **Depends on**: T4
- **Done when**: SyncAll enqueues; dispatcher tests pass
- **Req**: MAIL-05

### T6: Modify + Send commands and HTTP

- **What**: Use cases and routes for actions/compose
- **Where**: `commands/modify_message.go`, `commands/send_message.go`, HTTP controller/router
- **Depends on**: T3
- **Done when**: Command tests pass; e2e sync contract stays 202
- **Req**: MAIL-02, MAIL-03

### T7: List pagination, sharing, attachments

- **What**: folder/q/limit/offset filter; sharing policy; S3 download
- **Where**: postgres repo, list/get queries, HTTP
- **Depends on**: T2
- **Done when**: Paginated query; get includes real attachments
- **Req**: MAIL-06, MAIL-01

### T8: Connections OAuth scopes + Microsoft

- **What**: gmail.send; Microsoft connect/callback; Graph factory
- **Where**: `connections/wire.go`, connections HTTP, `inbox/adapters/provider/microsoft`
- **Depends on**: T3
- **Done when**: Factory builds microsoft; OAuth scopes updated
- **Req**: MAIL-04, MAIL-08

### T9: PWA mail client UI

- **What**: Folders, flags, compose, simple threads, attachments, pagination
- **Where**: `apps/pwa/src/app/inbox/**`
- **Depends on**: T6, T7
- **Done when**: Store calls new endpoints; dead buttons wired
- **Req**: MAIL-02, MAIL-03, MAIL-06, MAIL-07
