## Why

Incoming electronic invoices already capture line `item_code` and descriptions, but there is no tenant catalog or supplier master—so products cannot be tracked, matched across invoices, or prepared for future inventory. The commercial catalog boundary must be established now (analytics-first) without baking warehouse/manufacturing into invoicing.

## What Changes

- Introduce a **`parties`** bounded context: thin counterparties (tax id / NIT, name, supplier/customer roles), bootstrapped from invoice issuers.
- Introduce a **`catalog`** bounded context: thin **Item** master (kinds: goods, service, asset, unknown), multi-identifier aliases, link status, and match memory for user corrections.
- Wire **invoice persistence** to resolve/create parties and resolve/link (or suggest) catalog items per line using a fixed resolution policy.
- Expose APIs (and minimal PWA surfaces) to list parties/items, review unmatched/suggested lines, confirm or correct matches, and have the system remember those decisions.
- Keep invoice financial truth on invoice tables; catalog/parties only own identity and resolution.

## Non-goals / out of scope

- Inventory quantities, warehouses, goods receipts, or silent stock seeding from invoices.
- Manufacturing, BOM, shipping/dispatch, purchasing modules.
- Soft-match **auto-link** (suggestions only in this change).
- Renaming entitlements SaaS “product” → “module” (follow-up vocabulary cleanup).
- Full party CRM (addresses, contacts, credit) beyond identity needed for alias scope.
- ML training pipelines; learning is explicit match-memory + aliases.

## Capabilities

### New Capabilities

- `parties`: Tenant counterparties (suppliers/customers) identified primarily by tax id; bootstrap and link from invoice issuers.
- `catalog`: Tenant items (goods/service/asset/unknown), aliases, provisional mint on hard identity, resolution pipeline, match memory, and user-driven link/correct flows.
- `invoices`: Invoice issuer and line linking seams to parties and catalog (optional FKs, link status, no inventory side effects).

### Modified Capabilities

- (none — no existing `openspec/specs/` capabilities yet)

## Impact

- **Backend**: new `internal/parties` and `internal/catalog` features (hexagonal layout); tenant migrations; contracts/events or application ports called from `internal/invoices` after extract/persist; JSON:API endpoints under tenant API.
- **PWA**: thin catalog/parties list + match-review UX on invoice detail (or dedicated queue); store orchestration in `application/*store.ts`.
- **Data**: new tenant tables for parties, items, aliases, match memory, line/party link fields; existing `invoice_headers` / `invoice_lines` denormalized fields retained for document fidelity.
- **Future**: inventory/manufacturing will reference `item_id` only; stock movements remain a separate explicit event later.
