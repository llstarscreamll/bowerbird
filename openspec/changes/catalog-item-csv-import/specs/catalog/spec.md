## ADDED Requirements

### Requirement: Import catalog items from CSV

Authorized tenant users MUST be able to import catalog items from a single UTF-8 CSV (comma or semicolon delimiter detected from the header) via a presigned file upload followed by an asynchronous job. The file MUST include columns `internal_code` and `name`; `kind` MAY be omitted and MUST default to `unknown`. Kind values MUST accept `goods`, `service`, `asset`, `unknown` and Spanish aliases `bien`, `servicio`, `activo`, `desconocido`. The system MUST NOT require an item id in the file. The system MUST reject Excel/xlsx and MUST reject a second import while one is `queued` or `processing` for the tenant.

The import MUST upsert by `internal_code`: create a confirmed item with `creation_source` `import` when missing; when present, update `name` and `kind` without changing `internal_code` or `creation_source`. A provisional item that already has that `internal_code` MUST become `confirmed`. Duplicate codes inside the file MUST last-write-wins. Invalid rows MUST NOT abort the rest of the file.

The HTTP create of the import MUST copy the acting user as `requested_by` (`user_id`, `email`, `name`) from the session, not from the request body. The client MUST supply the import id as a valid ULID. The response MUST be `202` with the import resource.

#### Scenario: Queue import

- **WHEN** a user uploads a valid CSV key and POSTs `/api/v1/catalog/imports` with a ULID
- **THEN** the import is persisted as `queued` with `requested_by` snapshot and a job is enqueued; the response is 202

#### Scenario: Upsert by internal code

- **WHEN** a row has `internal_code` already present on another item
- **THEN** that item is renamed/reclassified and its `creation_source` remains unchanged

#### Scenario: New imported item

- **WHEN** a row has a new `internal_code`, name, and kind
- **THEN** a confirmed item is created with `creation_source` `import`

#### Scenario: Reject concurrent import

- **WHEN** a tenant already has an import in `queued` or `processing`
- **THEN** a new POST is rejected with a conflict error

#### Scenario: Invalid rows do not stop the batch

- **WHEN** some CSV rows lack `internal_code` or `name`
- **THEN** those rows are recorded as errors and valid rows continue to upsert

### Requirement: Catalog import history and progress

Authorized tenant users MUST be able to list import processes (paginated, newest first) and open one by id. The detail MUST include status, byte progress (`byte_offset` / `file_size_bytes`), row counters (`created`, `updated`, `failed`, `total_rows` when known), `requested_by`, and `cancelled_by` when cancelled. The detail MUST NOT embed the error list. The PWA MUST provide pages for history and detail with a live monitor that polls only while `queued` or `processing`.

#### Scenario: Open import detail

- **WHEN** a user opens an import by id
- **THEN** the system returns status, counters, actors, and timestamps without the error collection

#### Scenario: List import history

- **WHEN** a user lists imports with `page[size]`
- **THEN** the system returns a page of import processes including requester snapshot and `failed_count`

### Requirement: Cancel catalog import

Authorized tenant users MUST be able to cancel an import that is `queued` or `processing`. Cancel MUST copy the acting user as `cancelled_by` from the session. The worker MUST stop after the current chunk and MUST NOT enqueue the next chunk. Items already upserted MUST remain. Cancel of a terminal import MUST be rejected with a conflict error.

#### Scenario: Cancel in-flight import

- **WHEN** a user cancels a `processing` import
- **THEN** the import becomes `cancelled` with `cancelled_by` snapshot and the worker does not process further chunks

#### Scenario: Reject cancel of completed import

- **WHEN** a user cancels an import that is `completed`, `failed`, or `cancelled`
- **THEN** the system rejects the request with a conflict error

### Requirement: Catalog import row errors

The system MUST persist each invalid CSV row in the tenant database with the file row number (1-based, header is row 1), optional column, truncated cell evidence, a machine `code`, and a Spanish `message`. Authorized users MUST retrieve errors paginated (`page[size]`, `page[after]`) with `meta.total` equal to the import `failed_count`. Retrying a chunk MUST NOT duplicate errors for the same import/row/column.

#### Scenario: List row errors

- **WHEN** a user lists errors for an import
- **THEN** each error identifies the file row, the column when applicable, the values read, and a Spanish explanation; `meta.total` matches `failed_count`

#### Scenario: Chunk retry is idempotent for errors

- **WHEN** the same chunk is processed twice
- **THEN** the error table does not gain duplicate rows for the same `import_id`, `file_row`, and column

### Requirement: Paginated catalog item list

Authorized tenant users MUST list catalog items with `page[size]` (default 50, max 100) and cursor `page[after]` ordered by `(name, id)`. Search MUST remain available and MUST apply the same page size cap. The catalog master and invoice catalog linker MUST not load the unbounded tenant catalog.

#### Scenario: First page of items

- **WHEN** a user lists items without a cursor
- **THEN** the system returns at most `page[size]` items and a next cursor when more exist

#### Scenario: Search is bounded

- **WHEN** a user searches catalog items
- **THEN** the response is limited to the requested page size

### Requirement: Daily purge of stale catalog imports

The system MUST run a scheduled platform job every day at midnight America/Bogota that deletes catalog import processes (and their row errors) whose `created_at` is older than one year and whose status is not `queued` or `processing`. The purge MUST NOT delete catalog items.

#### Scenario: Purge year-old imports

- **WHEN** the daily purge runs
- **THEN** import processes older than one year in a terminal status are deleted along with their errors, and catalog items remain

#### Scenario: Skip active imports

- **WHEN** an import is still `queued` or `processing`
- **THEN** the purge does not delete it even if `created_at` is older than one year

## MODIFIED Requirements

### Requirement: Catalog item creation source

The system SHALL persist an immutable `creation_source` on each catalog item with allowed values `manual`, `invoice`, and `import`. The value MUST be set only at creation time and MUST NOT change on update, rename, kind change, confirmation, or upsert from a later import.

#### Scenario: CSV import sets import source

- **WHEN** a catalog item is created from a CSV import
- **THEN** the persisted item has `creation_source` `import`

#### Scenario: Import upsert keeps original source

- **WHEN** an import updates an item that was created from an invoice
- **THEN** `creation_source` remains `invoice`
