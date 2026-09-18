# Tasks: party-profile-enrichment

## 1. Code Specifications (BDD as Code)

- [x] 1.1 HTTP e2e: create con `scheme_id`; POST/DELETE emails, phones, addresses; GET embebe `source`

## 2. Domain and persistence

- [x] 2.1 Value objects y métodos de agregado (`scheme_id`, `taxpayer_kind`, `tax_level_codes`, canales) con tests
- [x] 2.2 Migración tenant `000022` (columnas + tablas hijas + unique/CHECK)
- [x] 2.3 Repositorio postgres: hidratar colecciones, Insert ON CONFLICT DO NOTHING, Delete

## 3. Invoice seam

- [x] 3.1 `mapParty` UBL (Contact, direcciones, AdditionalAccountID) + test fixture iShop
- [x] 3.2 OHS `ResolveIssuer(IssuerProfile)`; `applyLinking` pasa emisor completo; actualizar stubs

## 4. Application and HTTP

- [x] 4.1 `ResolveOrCreateFromIssuer`: fill-if-empty + unión; tests
- [x] 4.2 Commands add/remove canal; create/update scheme fill-if-empty
- [x] 4.3 JSON:API GET embebe; POST/DELETE canales

## 5. PWA

- [x] 5.1 Modelo, HTTP, store, detalle (badges de fuente), form tipo de documento y canales

## 6. Verification

- [x] 6.1 `pnpm --filter @atta/backend test` (e2e HTTP parties no corre: API local 502)
