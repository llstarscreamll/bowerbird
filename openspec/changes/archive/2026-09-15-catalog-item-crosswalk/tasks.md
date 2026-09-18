## 1. Dominio catalog

- [x] 1.1 VO `GTIN` y `SellerSKU.Usable()` **en catalog** (misma regla publicada: dígitos 8/12/13/14; denylist); `Alias.Source`; factories; tests — sin importar invoices
- [x] 1.2 `HardHits.Agree`, `CanMintProvisional` por seller usable, `LinkedByHardConflict` (suggestions `hard_conflict`); tests de acuerdo / conflicto / mint gate

## 2. Dominio invoices + parser

- [x] 2.1 VO `LineIdentifiers` (parse GTIN **en invoices**, sin paquete compartido); `LineLink.Unlock`; `LineForDecision`; tests
- [x] 2.2 Parser DIAN: `BuyersItemIdentification`, clasificador GTIN, fallback Standard→seller; tests (GTIN, no-GTIN igual a seller, fallback)

## 3. Persistencia

- [x] 3.1 Migración tenant `000019_catalog_item_crosswalk` (`source` en aliases; columnas de línea; backfill; drop `item_code`; down)
- [x] 3.2 `AliasRepository`: list by item, delete, create con source, conflicto con owner id; `CreateItemWithAliases`; `ItemRepository` lookup `internal_code` exacto para buyer hit
- [x] 3.3 Repos invoices: leer/escribir `buyer_code`/`seller_sku`/`gtin`; quitar `item_code` de scans/inserts

## 4. Application catalog + OHS

- [x] 4.1 Commands `AddItemAlias` / `RemoveItemAlias`; query `GetItem` incluye aliases; 409 `WithMeta("item_id")`; tests
- [x] 4.2 Command `RememberDecision`: 1 TX aliases (`source=invoice`) + memory; 409 owner id; mint `CreateItemWithAliases`
- [x] 4.3 `ResolveInvoiceLine` multi-hit + tests (buyer, gtin, seller, conflicto, no-mint unusable, mint+gtin)
- [x] 4.4 OHS `InvoiceSupport` BREAKING (`ResolveLine`/`RememberDecision`/`Mint…` con BuyerCode/SellerSKU/GTIN); `CatalogACL` único traductor; application invoices **no** importa `catalog/api`

## 5. Application + HTTP invoices

- [x] 5.1 `ApplyLineDecision`: `unlock`; remember → `RememberDecision` (un call); catalog antes de persistir el link; tests
- [x] 5.2 JSON:API detalle/review/decisions: tres identificadores; action `unlock`; sin `item_code`

## 6. HTTP catalog

- [x] 6.1 `GET item` con `attributes.aliases`; `POST/DELETE .../aliases`; tests de handler (201, 204, 409+meta)

## 7. PWA

- [x] 7.1 Modelos/HTTP/store catalog: aliases en detalle; add/remove; resolver nombre de party en cliente
- [x] 7.2 Detalle ítem: tabla de cruce + alta + borrar; conflicto muestra ítem dueño
- [x] 7.3 Modelos factura/review sin `item_code`; mostrar buyer/seller/gtin
- [x] 7.4 Linker: evidencias, preview, remember/lock default on, unlock, `remember: false` no enseña

## 8. Docs y e2e

- [x] 8.1 Glosario: GTIN, código del adquirente (`buyer_code`), cruce vs `InternalCode`
- [x] 8.2 E2E HTTP: aliases CRUD, línea con tres ids, unlock, decisión remember adjunta aliases
- [x] 8.3 `pnpm --filter @atta/backend test` y `pnpm --filter @atta/web test`
