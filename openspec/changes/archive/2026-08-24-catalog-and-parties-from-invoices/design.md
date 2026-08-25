## Context

See proposal.md for motivation. Today `internal/invoices` stores denormalized issuer fields and line `item_code` / `description` only (`invoice_headers`, `invoice_lines`). EFI choreography already separates bounded contexts via ports/events. Entitlements uses the word “product” for SaaS packs—commercial catalog MUST use `catalog` + `Item` naming to avoid collision (entitlements rename is out of scope).

## Goals / Non-Goals

**Goals:**

- Hexagonal `internal/parties` and `internal/catalog` modules with tenant migrations.
- Deterministic resolution policy (lock → memory → hard → soft-suggest; provisional mint on party+code miss).
- Invoice seam: optional FKs + link metadata; financial rows stay source of truth.
- Minimal JSON:API + PWA for list/review/correct/remember.

**Non-Goals:**

- Inventory, BOM, shipping, soft auto-link, entitlements vocabulary rename.
- Shared DB tables owned by invoices for item master data.

## Decisions

### D1 — Separate `parties` and `catalog` contexts

- **Choice:** Two modules; catalog aliases reference `party_id` for supplier SKU scope.
- **Why:** Trading-partner identity will outlive catalog (AP/AR, POs). Embedding suppliers inside catalog couples the wrong axes.
- **Rejected:** Parties as a subpackage of catalog; parties only as strings on invoices forever.

### D2 — Sync application ports from invoices (v1), events optional later

- **Choice:** After successful invoice insert, invoicing calls parties + catalog application commands/ports in-process (same worker). Publish domain events later if other consumers appear.
- **Why:** One consumer today; simpler transactional/retry story. Matches “prefer existing patterns” over premature choreography.
- **Rejected:** Mandatory EventBridge event before any linking (extra infra for one subscriber).

### D3 — Resolution trust order (locked)

```
user-locked link → match memory → hard alias (party + code) → soft suggestions only
provisional mint only when party + non-empty code and no memory/alias hit
```

- Soft matchers implement a `Matcher` port; v1 may ship exact normalized-description equality as suggest-only.
- **Rejected:** Soft auto-link above a score threshold (poisons catalog before inventory).

### D4 — Provisional items on hard identity only

- **Choice:** Auto-create `Item(status=provisional, kind=unknown)` + `supplier_sku` alias when party+code miss.
- **Why:** Analytics can group by item_id immediately for coded lines; description-only stays unmatched to limit noise (services/PDF).
- **Rejected:** Mint on every line; observe-only with no mint (weaker analytics UX).

### D5 — Match memory as first-class table, not ML

- Evidence key: `party_id` (nullable), `item_code` (nullable), `description_fingerprint` (nullable), `evidence_kind`.
- Decision: `item_id`, `action` (`link` | `never_match`), `locked` preference for future lines optional.
- User writes alias + memory on “remember”.
- **Rejected:** Black-box model in v1.

### D6 — Schema sketch (tenant DB)

- `parties`: id, tax_id (unique nullable carefully—empty tax ids not in unique index), name, roles flags/array, status, timestamps, raw_data optional.
- `catalog_items`: id, name, kind, status (`provisional`|`confirmed`), stockable intent flag default false/unset, timestamps.
- `catalog_item_aliases`: item_id, scheme, party_id null, value, unique (scheme, party_id, value).
- `catalog_match_memories`: evidence columns + decision columns, unique evidence fingerprint.
- `invoice_headers.issuer_party_id` nullable FK.
- `invoice_lines`: item_id nullable, link_status, link_method, link_locked bool, suggestions jsonb optional.

### D7 — Failure isolation

- Prefer: party resolve + catalog resolve in same DB transaction as invoice insert when all repos share tenant DB.
- If that becomes too large: commit invoice first, then resolve with idempotent “link invoice” command and `linking_status` on header for retry. Specs allow either if invoice is never lost.

### D8 — PWA surface (thin)

- Invoice detail: show party chip, per-line item link/status, actions confirm/correct/remember.
- Simple `/catalog` and `/parties` list pages under tenant routes.
- Stores in `application/*store.ts`; Spartan/Helm patterns as elsewhere.

### D9 — Vocabulary

- Code: `parties`, `catalog`, `Item`, `Party`. UI ES: “productos/servicios/activos” for items; avoid calling SaaS entitlements “product” in new copy where easy.
- **Rejected:** Module name `products` next to entitlements.

## Risks / Trade-offs

- **[Risk] Provisional mint floods catalog with junk SKUs** → Mitigation: provisional filter in UI; merge/confirm later; no mint without code.
- **[Risk] Same physical good, different supplier codes** → Mitigation: manual link + memory; merge items deferred (open question).
- **[Risk] Description fingerprint collisions** → Mitigation: description memory only when user explicitly remembers; not used for auto-mint.
- **[Risk] Transaction size / lock time on large invoices** → Mitigation: D7 fallback linking_status + retry job.
- **[Trade-off] In-process ports vs events** → Faster v1; revisit when inventory consumes `InvoiceLinked` / `GoodsReceived`.

## Migration Plan

1. Tenant migrations for parties, catalog tables, invoice FK/columns (nullable, backward compatible).
2. Deploy backend with resolution on new ingests only (no mandatory backfill).
3. Optional follow-up task: backfill parties/items for existing invoices via batch command.
4. Rollback: nullable columns unused if feature flagged off; tables can remain empty.

## Open Questions

- Item merge UX when two provisionals are the same real-world product (defer; not required for v1 acceptance).
- Whether `stockable` defaults per kind (`goods` → unset vs false) until inventory ships—recommend leave unset/false and ignore for v1 analytics.
