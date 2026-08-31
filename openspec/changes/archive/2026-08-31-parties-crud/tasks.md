## 1. Backend — Value Objects & Aggregate

- [x] 1.1 VOs `TaxID` y `PartyRoles` en `domain/` + tests parse/roles vacíos
- [x] 1.2 Factories `NewProvisionalSupplier`/`NewConfirmedParty` con VOs; eliminar validadores sueltos
- [x] 1.3 `AssignRoles`, `UpdateProfile`; commands sin orquestar `UpdatedAt` manual

## 2. Backend — Application, HTTP & límites

- [x] 2.1 `CreatePartyCommand` vía `ports.PartyRepository` (unicidad NIT); registrar en `wire.go`
- [x] 2.2 `UpdatePartyCommand` → `UpdateProfile`; sin mutar NIT
- [x] 2.3 `POST /api/v1/parties`; handler delgado (map JSON → command, sin reglas); PATCH ignora `tax_id`
- [x] 2.4 Tests HTTP create/update/409/422
- [x] 2.5 Verificar: `invoices`/`catalog` no importan `parties/domain` ni repository (grep o review)

## 3. PWA — Capas aisladas

- [x] 3.1 `PartiesHttpService`: `get`, `create`, `patch` — único punto HTTP
- [x] 3.2 `PartiesStore`: `loadParty`, `createParty`, `updateParty` + toasts; pages sin `HttpClient`

## 4. PWA — UI

- [x] 4.1 `party-form` (nombre, NIT readonly edit, roles ≥1)
- [x] 4.2 Rutas `new`, `detail`, `edit` (patrón `catalog/`)
- [x] 4.3 Master: "Nuevo Contacto", filas clicables

## 5. Verificación

- [x] 5.1 `pnpm --filter @bowerbird/backend test` y `pnpm --filter @bowerbird/pwa lint`
- [x] 5.2 Flujo manual create → list → edit; NIT inmutable
- [x] 5.3 Ingesta factura con NIT existente sigue enlazando party (integración `IssuerPartyResolver` intacta)
