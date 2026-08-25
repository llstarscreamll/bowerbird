## 1. Tenant schema

- [x] 1.1 Add tenant migration for `parties` (id, tax_id, name, roles, status, timestamps) with unique index on non-empty `tax_id`
- [x] 1.2 Add tenant migration for `catalog_items`, `catalog_item_aliases`, `catalog_match_memories`
- [x] 1.3 Add nullable `issuer_party_id` on `invoice_headers` and link columns on `invoice_lines` (`item_id`, `link_status`, `link_method`, `link_locked`, `suggestions`)
- [x] 1.4 Verify migrations up/down via `pnpm --filter @bowerbird/backend migrate:all` (or project-equivalent) against local Postgres

## 2. Backend parties module

- [x] 2.1 Scaffold `internal/parties` hexagonal layout (domain, application, adapters, wire) mirroring `internal/invoices`
- [x] 2.2 Implement party domain + Postgres repository (create/get/list by role/search, unique tax id conflicts)
- [x] 2.3 Implement `ResolveOrCreateFromIssuer` command (supplier role, provisional) with unit tests
- [x] 2.4 Expose JSON:API list/get (and optional patch name/roles) under tenant routes; register in API wire
- [x] 2.5 Add parties HTTP handler tests for duplicate tax id and list filter

## 3. Backend catalog module

- [x] 3.1 Scaffold `internal/catalog` hexagonal layout (domain Item/Alias/MatchMemory, application, adapters, wire)
- [x] 3.2 Implement item + alias repositories (unique alias tuple, kinds, provisional/confirmed status)
- [x] 3.3 Implement match-memory repository and “remember decision” command (link / never_match) with tests
- [x] 3.4 Implement resolution command: lock → memory → hard alias → soft suggest → provisional mint on party+code; unit tests for each branch
- [x] 3.5 Add suggest-only soft matcher (normalized description equality) behind Matcher port; assert no auto-link
- [x] 3.6 Expose JSON:API list/get items, review-queue query (unmatched/suggested lines), manual link/correct/remember endpoints
- [x] 3.7 Wire catalog into API composition root; handler tests for remember + locked link

## 4. Invoice seam

- [x] 4.1 Extend invoice domain/persistence records and Postgres repo for party/line link fields (read/write)
- [x] 4.2 After successful invoice insert, call parties resolve then catalog resolve per line (prefer same tenant TX; else `linking_status` + retry path per design D7)
- [x] 4.3 Ensure resolution failure does not lose CUFE-unique invoice; add test for link-after-insert / retry observability
- [x] 4.4 Extend invoice JSON:API responses with `issuer_party_id`, line `item_id`, link status/method/locked/suggestions
- [x] 4.5 Integration-style test: persist sample UBL-derived invoice → party created → provisional item for coded line → empty code unmatched

## 5. PWA

- [x] 5.1 Add tenant routes + thin list pages for parties and catalog items (stores in `application/*store.ts`)
- [x] 5.2 Extend invoice detail to show issuer party and per-line link status / item
- [x] 5.3 Add match actions on line (assign item, remember, lock) and simple review queue page calling catalog APIs
- [x] 5.4 Inline alerts for 4xx on conflict (duplicate alias/tax id); toast only for unexpected 5xx

## 6. Verification

- [x] 6.1 Backend: `pnpm --filter @bowerbird/backend lint && pnpm --filter @bowerbird/backend test` for parties, catalog, invoices packages touched
- [x] 6.2 PWA: `pnpm --filter @bowerbird/pwa lint` (and targeted tests if added)
- [x] 6.3 Manual smoke on local stack: ingest or fixture invoice → party + provisional item → correct match with remember → second invoice auto-links via memory
