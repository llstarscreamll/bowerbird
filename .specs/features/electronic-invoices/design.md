# Design — electronic invoice management (EFI)

## 1. Design scope

Covers:

- Multi-provider, multi-account pull ingest per tenant
- Attachment pipeline (decompress, classify, group documents)
- Hybrid extraction: DIAN XML first, Gemini PDF fallback
- Persistence and idempotency
- Connection UI and unified inbox

## 2. Bounded contexts

### 2.1 `inbox` (mail integration)

**Owns**

- Connected accounts per tenant/provider
- OAuth2 and token refresh
- Periodic pull of messages for active accounts
- Attachment download and S3 storage
- Mail domain event publication

**Does not**

- Apply invoicing rules
- Parse DIAN XML or run invoice LLM extraction

### 2.2 `invoicing` (invoice finance)

**Owns**

- Subscribe to mail events and filter candidates
- Run the document pipeline
- Parse UBL 2.1 DIAN XML
- Run Gemini PDF fallback
- Normalize, deduplicate, and persist invoice + lines

**Does not**

- Manage mail OAuth
- Sync inboxes directly

## 3. Architecture decisions

1. **Event choreography**: `inbox` publishes `InboxMessageReceived`; `invoicing` consumes independently.
2. **No central orchestrator (for now)**: avoids coupling and premature optimization.
3. **Adapter/Strategy for LLM**: stable `InvoiceLLMExtractor` interface; initial implementation is Gemini.
4. **Multi-level idempotency**: message (`provider_message_id`), file (hash), invoice (CUFE).
5. **Shared S3 with prefix segmentation**: logical isolation by tenant / module / stage.

## 4. End-to-end flow

1. `inbox` worker loads active accounts for a tenant.
2. Per account: incremental pull of new / unprocessed mail.
3. Persist raw mail metadata (`raw_data`) and upload attachments to S3.
4. Publish `InboxMessageReceived` with S3 refs.
5. `invoicing` consumer receives the event and decides if it is an invoice candidate.
6. If yes: decompress, classify, and group documents.
7. Valid XML → parse UBL 2.1 (priority 1).
8. No XML → invoke Gemini on PDF (priority 2).
9. Normalize to the internal model; validate CUFE and idempotency.
10. Persist header, lines, and `raw_data`; emit logs and metrics.

## 5. Application components

### 5.1 `apps/backend/internal/inbox`

| Path                                                  | Role                                   |
| ----------------------------------------------------- | -------------------------------------- |
| `application/sync_accounts_usecase.go`                | Sync cycle per account                 |
| `domain/connected_account.go`                         | Connected account entity and status    |
| `domain/email_message.go`                             | Synced message entity                  |
| `domain/attachment.go`                                | Attachment entity (hash, type, S3 key) |
| `domain/events.go`                                    | `InboxMessageReceived` event           |
| `infra/provider/gmail_client.go`, `outlook_client.go` | Provider adapters (iterative)          |
| `infra/repository/postgres/*.go`                      | Tenant repos for accounts/messages     |

### 5.2 `apps/backend/internal/invoicing`

| Path                                         | Role                                         |
| -------------------------------------------- | -------------------------------------------- |
| `application/process_inbox_event_usecase.go` | Event entry point                            |
| `application/classify_documents_usecase.go`  | Document classification / grouping           |
| `application/extract_invoice_usecase.go`     | Choose XML or LLM fallback                   |
| `domain/invoice.go`, `invoice_line.go`       | Invoice aggregate                            |
| `domain/extractors.go`                       | `InvoiceXMLExtractor`, `InvoiceLLMExtractor` |
| `infra/xml/dian_ubl21_parser.go`             | Native UBL 2.1 parser                        |
| `infra/llm/gemini_extractor.go`              | Gemini implementation                        |
| `infra/storage/s3_reader.go`                 | Read documents from S3                       |
| `infra/repository/postgres/*.go`             | Persist invoice and lines with `raw_data`    |

### 5.3 `apps/pwa`

| Path                                                            | Role                                    |
| --------------------------------------------------------------- | --------------------------------------- |
| `src/app/features/inbox-connections/*`                          | OAuth connect UI per provider/account   |
| `src/app/features/unified-inbox/*`                              | Responsive multi-provider unified inbox |
| `src/app/core/presentation/components/connection-status-chip/*` | Shared per-account status chip          |
| `src/app/features/invoices/*`                                   | Extracted invoices view (iterative)     |

## 6. Event schema

### `InboxMessageReceived`

Minimum fields:

- `event_id`
- `occurred_at`
- `tenant_id`
- `account_id`
- `provider`
- `provider_message_id`
- `message_internal_id`
- `subject`
- `from`
- `received_at`
- `attachment_refs[]` (S3 key, filename, mime, hash)
- `raw_data_ref`

## 7. Data model (tenant DB)

Proposed tables (initial names):

| Table                | Key columns                                                                                                                                                                                                                             |
| -------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `connected_accounts` | `id`, `tenant_id`, `provider`, `email`, `status`, `encrypted_credentials`, `last_sync_at`, `last_error`, `raw_data`, timestamps                                                                                                         |
| `email_messages`     | `id`, `tenant_id`, `account_id`, `provider_message_id`, `thread_id`, `subject`, `sender`, `received_at`, `processing_status`, `raw_data`, timestamps                                                                                    |
| `email_attachments`  | `id`, `tenant_id`, `message_id`, `filename`, `mime_type`, `size_bytes`, `sha256`, `s3_key`, `raw_data`, timestamps                                                                                                                      |
| `invoice_headers`    | `id`, `tenant_id`, `source_message_id`, `cufe`, `invoice_number`, `issuer_name`, `receiver_name`, `currency`, `subtotal`, `tax_total`, `grand_total`, `document_ref_s3_key`, `extraction_source` (`xml`\|`llm`), `raw_data`, timestamps |
| `invoice_lines`      | `id`, `tenant_id`, `invoice_header_id`, `line_number`, `description`, `quantity`, `unit_price`, `line_tax_total`, `line_total`, `raw_data`, timestamps                                                                                  |

Key indexes / constraints:

- Unique (`tenant_id`, `provider`, `provider_message_id`) on `email_messages`
- Unique (`tenant_id`, `cufe`) on `invoice_headers`
- Optional unique (`tenant_id`, `sha256`, `s3_key_scope`) for physical anti-duplication

## 8. S3 and privacy

Prefix:

`tenant/{tenant_id}/{module}/{stage}/{yyyy}/{mm}/{dd}/{resource_id}/{filename}`

Examples:

- `tenant/t_123/inbox/raw/2026/05/25/msg_abc/factura.zip`
- `tenant/t_123/invoicing/normalized/2026/05/25/inv_999/factura.pdf`

Rules:

- Private bucket with block public access
- No public ACLs
- Downloads only via presigned URL from an authenticated, authorized backend

## 9. Security and secrets

- External account tokens encrypted in tenant DB (`encrypted_credentials`)
- Encryption key from secure environment (KMS / app secret)
- LLM credentials from SSM via backend config
- Auth error handling rotates status; UI surfaces reconnect-required accounts

## 10. Reliability, idempotency, and retries

- Exponential backoff for external APIs and transient failures
- Dead-letter queue after max attempts for unprocessable events
- Idempotent keys:
  - (`tenant_id`, `provider`, `provider_message_id`)
  - (`tenant_id`, `cufe`)
  - (`tenant_id`, file `sha256`)
- DB transactions per atomic invoice unit

## 11. Observability

Structured logs include `tenant_id`, `account_id`, `message_id`, `cufe`, `event_id`, `attempt`.

Minimum metrics:

- `inbox_sync_messages_total`
- `inbox_sync_errors_total`
- `invoicing_documents_classified_total`
- `invoicing_extraction_xml_total`
- `invoicing_extraction_llm_total`
- `invoicing_duplicates_skipped_total`
- `invoicing_processing_latency_ms`

## 12. Future evolution

- Add more `InboxMessageReceived` consumers (expenses, legal, purchases) without changing `inbox`
- Add LLM adapters without changing use cases
- Consider a dedicated routing context only when 3+ consumer domains share complex cross-cutting rules
