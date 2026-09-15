# Inbox and invoicing

Two bounded contexts: sync mail (`inbox`), extract Colombian **received**
e-invoices (`invoices`, Accounts Payable). Issued invoices (AR) are out of
scope. Domain terms **DIAN** and **CUFE** stay as-is. UI label: **Facturas
recibidas**.

## Product flow

### Inbox

- Connect mail accounts (Gmail / Microsoft); track connection status.
- Incremental sync from the last cursor. Gmail uses History after the first
  successful sync and stores the list page token so a rate-limited or
  time-boxed backfill resumes instead of restarting.
- Sync fetches message metadata first. Full bodies and attachments download
  only for e-invoice candidates (subject keywords such as "factura", or
  XML/PDF/ZIP attachments) or when a user opens the message.
- Gmail API calls are paced per mailbox against the 6,000 units/minute
  per-user quota. Rate-limit responses retry with exponential backoff.
  Each `InboxSyncAccount` job stops under the Lambda cap (15 minutes; the
  command yields after 12 minutes or when the context deadline is close)
  and checkpoints the list page token. The 5-minute poll continues the
  remainder.

### Invoicing

- Candidate filter (subject keywords like “factura”, XML/PDF attachments).
- Classify / unzip DIAN ZIPs; group XML+PDF by normalized filenames.
- Extract:
  - **XML (primary):** DIAN UBL 2.1 parser.
  - **PDF (fallback):** LLM extractor (Gemini) with a strict JSON schema.
- Deduplicate by source message and by **CUFE**.
- Persist only when a tenant `LegalEntity` exists and the invoice receiver TaxID
  matches it (digits only). Mail always syncs; extraction waits for that identity.

## Technical flow

1. Inbox persists metadata for every message. Full body + attachments (and
   `InboxMessageReceived`) only for e-invoice candidates or when the user
   opens the message.
2. Invoices subscriber may enqueue `InvoiceExtractionRequested` (SQS job) when
   a `LegalEntity` exists.
3. Job classifies documents, extracts, dedups, persists header/lines if the
   receiver TaxID matches the tenant `LegalEntity`.
4. Creating or changing a `LegalEntity` TaxID publishes `LegalEntityRegistered`.
   Invoices then pages inbox extraction candidates (`InvoiceInboxBackfillRequested`)
   so mail that arrived before the NIT is registered can still be extracted.

### `inbox` (`internal/inbox`)

Connected accounts, messages, folders, provider clients, encrypted OAuth
credentials, sync commands/jobs.

### `legalentities` (`internal/legalentities`)

Tenant `LegalEntity` (own NIT/cédula). HTTP CRUD. OHS `ReceiverDirectory`.
Publishes `LegalEntityRegistered` on create and TaxID change.

### `invoices` (`internal/invoices`)

Document classification, XML/LLM extractors, dedup repositories,
invoice persistence.

Cross-context sync calls go through `{bc}/api` (catalog invoice
support, parties issuer lookup, secrets document passwords, legalentities
receiver directory, inbox extraction candidates). Async coupling stays on
events and jobs. See [Backend architecture](./backend-api.md) and [Events vs
jobs](./events-vs-jobs.md).
