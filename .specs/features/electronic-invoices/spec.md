# Spec — electronic invoice management

## Status

| Field      | Value                                  |
| ---------- | -------------------------------------- |
| Status     | Approved for design and implementation |
| Date       | 2026-05-25                             |
| Feature ID | EFI                                    |
| Language   | EN                                     |

## Goal

Build modular, deterministic electronic invoice management for multi-account tenants: pull-based email ingest, hybrid XML/LLM extraction, deduplication by message and CUFE, and structured persistence for web query and metrics.

## Closed decisions

1. Ingest strategy: `pull`, multi-provider, multiple accounts per tenant.
2. Initial LLM provider: Google Gemini via direct API.
3. LLM extensibility: adapter architecture (OpenAI, Anthropic, and others).
4. LLM credentials: from SSM.
5. Mail account credentials: encrypted in the tenant Postgres database.
6. Frontend in scope: account connection, unified inbox, sync/session status.
7. Cross-cutting routing: event choreography (no central orchestrator yet).

## Functional requirements

### RF-INGEST (mail ingest and monitoring)

- [EFI-RF-001] Register multiple mail accounts per tenant.
- [EFI-RF-002] Allow multiple accounts of the same provider per tenant.
- [EFI-RF-003] Pull-sync every N minutes per active account.
- [EFI-RF-004] Capture and store mail metadata in the tenant database.
- [EFI-RF-005] Download all mail attachments.
- [EFI-RF-006] Upload attachments to S3 and store references in the database.
- [EFI-RF-007] Publish `InboxMessageReceived` for each synced message.

### RF-PIPELINE (decompress and classify)

- [EFI-RF-008] Detect file type (ZIP / XML / PDF / other).
- [EFI-RF-009] Decompress ZIP archives and process contents.
- [EFI-RF-010] Group files that belong to the same commercial document (XML + PDF pair when present).
- [EFI-RF-011] Mark unclassifiable documents without blocking the queue.

### RF-EXTRACT (hybrid engine)

- [EFI-RF-012] Prefer the native UBL 2.1 DIAN XML parser.
- [EFI-RF-013] XML parser extracts at least issuer, receiver, CUFE/UUID, totals, taxes (`TaxTotal`), payment codes, and lines.
- [EFI-RF-014] Without valid XML, use Gemini on PDF with strict structured output (JSON Schema).
- [EFI-RF-015] XML and LLM normalize to the same internal invoice model.
- [EFI-RF-016] Persist unmapped data in `raw_data` JSONB so nothing is lost.

### RF-VALIDATE (business rules and deduplication)

- [EFI-RF-017] Prevent duplicate processing of an already-synced mail message.
- [EFI-RF-018] If a message was already processed for invoicing, skip it without changing files or financial tables.
- [EFI-RF-019] Validate CUFE uniqueness before persisting an invoice.
- [EFI-RF-020] If CUFE already exists, log and skip with no side effects.

### RF-STORAGE (private multi-tenant S3)

- [EFI-RF-021] Shared bucket uses standardized prefixes by tenant and module.
- [EFI-RF-022] Objects stay private with no public access.
- [EFI-RF-023] File access uses presigned URLs from an authenticated, tenant-authorized backend.
- [EFI-RF-024] Idempotency strategy avoids duplicate physical uploads.

### RF-PERSIST (data model)

- [EFI-RF-025] Persist mail, pipeline, and extraction data in tenant tables.
- [EFI-RF-026] Every external entity includes a `raw_data` JSONB column.
- [EFI-RF-027] Invoice header stores CUFE, totals, taxes, support refs, and extraction status.
- [EFI-RF-028] Invoice detail lines link to the header.

### RF-FRONTEND (auth and unified inbox)

- [EFI-RF-029] UI flow to connect accounts (OAuth2 per provider).
- [EFI-RF-030] Unified view of synced mail from all connected tenant accounts.
- [EFI-RF-031] Show per-account status (active, token error, reconnect required, paused).
- [EFI-RF-032] Responsive unified-inbox UX.

## Non-functional requirements

- [EFI-RNF-001] Clean, modular architecture by bounded context (`inbox`, `invoicing`).
- [EFI-RNF-002] Inter-module communication via events (choreography); minimize coupling.
- [EFI-RNF-003] Retries with backoff for external APIs (mail providers, Gemini, S3 when applicable).
- [EFI-RNF-004] Idempotent, deterministic processing by business keys (`provider_message_id`, CUFE, attachment hash).
- [EFI-RNF-005] Structured logging traceable by tenant, account, message, and document.
- [EFI-RNF-006] Observability metrics for sync, classification, extraction, errors, and deduplication.
- [EFI-RNF-007] Secrets via SSM; encrypt external account tokens at rest in the tenant database.
- [EFI-RNF-008] Horizontal worker scaling without breaking per-message / per-document atomicity.

## Use cases (high level)

- [EFI-UC-001] User connects two Gmail accounts and one Outlook account in a tenant; the system syncs all without mixing tenant data.
- [EFI-UC-002] Mail with ZIP containing XML + PDF: decompress, prefer XML, persist invoice and lines.
- [EFI-UC-003] Mail with PDF only: use Gemini, normalize, and persist.
- [EFI-UC-004] Repeated mail: detect duplicate and skip.
- [EFI-UC-005] Invoice with existing CUFE: log deduplication and skip financial writes.
- [EFI-UC-006] Account token expires: UI shows reconnect-required status.

## Acceptance criteria

1. Every RF and RNF traces to implementation tasks.
2. Full flow mail → event → pipeline → extraction → persistence works locally with RabbitMQ + MinIO.
3. Unit tests cover the DIAN XML parser and LLM normalizer.
4. At least one integration test covers the idempotent CUFE deduplication path.
5. UI connects accounts, shows statuses, and lists mail in a responsive unified view.

## Out of scope (this delivery)

- Native push / webhook integrations from mail providers.
- Other document types (non-invoice expenses, legal notices, purchases) beyond event-ready architecture.
- Advanced ML rules for semantic mail classification.
