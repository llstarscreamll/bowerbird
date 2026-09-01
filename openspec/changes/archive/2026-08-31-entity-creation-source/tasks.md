## 1. Schema

- [x] 1.1 Añadir `creation_source` + CHECK en `migrations/tenant/000011_parties.up.sql` y verificar `migrate:all` recrea `parties` con la columna
- [x] 1.2 Añadir `creation_source` + CHECK en `migrations/tenant/000012_catalog.up.sql` y verificar `migrate:all` recrea `catalog_items` con la columna

## 2. Backend — parties

- [x] 2.1 Añadir constantes `CreationSourceManual`/`CreationSourceInvoice` y campo en `domain.Party`; actualizar `NewProvisionalSupplier` (`invoice`) y `NewConfirmedParty` (`manual`); tests de dominio pasan
- [x] 2.2 Persistir/leer `creation_source` en `party_repository.go`; `UPDATE` no modifica la columna; tests de repo/command pasan
- [x] 2.3 Exponer `creation_source` en HTTP list/get y aceptar filtro `?creation_source=`; `router_test.go` cubre manual vs invoice y filtro
- [x] 2.4 Verificar `ResolveOrCreateFromIssuerCommand` crea parties con `creation_source=invoice` (`resolve_or_create_test.go`)

## 3. Backend — catalog

- [x] 3.1 Añadir constantes y campo en `domain.Item`; actualizar `NewManualItem` (`manual`) y `NewProvisionalItem` (`invoice`); tests de dominio pasan
- [x] 3.2 Persistir/leer `creation_source` en `catalog_repository.go`; writes de update no modifican la columna; tests pasan
- [x] 3.3 Exponer `creation_source` en HTTP list/get y filtro `?creation_source=`; `router_test.go` actualizado
- [x] 3.4 Verificar `mintProvisional` y path de `ApplyLineDecision` mint persisten `creation_source=invoice` (`resolve_line_test.go`, `apply_line_decision_test.go` si aplica)

## 4. PWA

- [x] 4.1 Añadir `creation_source` a models HTTP/store de `parties/` y `catalog/`; mapear labels ES ("Manual", "Desde factura")
- [x] 4.2 Mostrar origen en master y detail de Contactos y Catálogo; build PWA sin errores (`pnpm --filter @bowerbird/pwa build`)

## 5. Verificación

- [x] 5.1 `pnpm --filter @bowerbird/backend test` y `pnpm --filter @bowerbird/pwa lint` pasan
- [x] 5.2 `openspec validate entity-creation-source --strict` pasa
