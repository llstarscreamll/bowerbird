# Inbox and invoicing

Two bounded contexts: sync mail (`inbox`), extract Colombian e-invoices (`invoices`). Domain terms **DIAN** and **CUFE** stay as-is.

## Product flow

### Inbox

- Connect mail accounts (Gmail / Microsoft); track connection status.
- Incremental sync from the last cursor.
- Download attachments to S3 for downstream processing.

### Invoicing

- Candidate filter (subject keywords like “factura”, XML/PDF attachments).
- Classify / unzip DIAN ZIPs; group XML+PDF by normalized filenames.
- Extract:
  - **XML (primary):** DIAN UBL 2.1 parser.
  - **PDF (fallback):** LLM extractor (Gemini) with a strict JSON schema.
- Deduplicate by source message and by **CUFE**.

## Technical flow

1. Inbox persists message + attachments → publishes `InboxMessageReceived` (EventBridge).
2. Invoices subscriber may enqueue `InvoiceExtractionRequested` (SQS job).
3. Job classifies documents, extracts, dedups, persists header/lines.

### `inbox` (`internal/inbox`)

Connected accounts, messages, folders, provider clients, encrypted OAuth credentials, sync commands/jobs.

### `invoices` (`internal/invoices`)

Document classification, XML/LLM extractors, dedup repositories, invoice persistence. Keep coupling via events/jobs, not direct feature imports.

See also: [Events vs jobs](./events-vs-jobs.md).
