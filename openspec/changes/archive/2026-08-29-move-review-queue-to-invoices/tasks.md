## 1. Dominio invoices — LineLink y reglas de decisión

- [x] 1.1 Introducir VO `LineLink` y métodos de dominio en invoices (`ApplyManualLink`, `RejectLink`, `RecalculateLinkingStatus`); verificar tests unitarios de dominio Given/When/Then
- [x] 1.2 Mover `DecideManualLink` y constantes de `link_status`/`link_method` de línea desde `catalog/domain` a `invoices/domain`; actualizar imports; verificar tests migrados pasan en invoices y catalog sin referencias rotas
- [x] 1.3 Eliminar de `catalog/domain` lo movido (sin package compartido); verificar `catalog` tests de resolve/auto-link siguen verdes

## 2. Catalog ACL — identidad sin tablas invoice

- [x] 2.1 Exponer en `catalog` application commands para mint provisional, ensure supplier alias y record match memory sin `InvoiceLineLinkRepository`; verificar tests unitarios de commands
- [x] 2.2 Definir `CatalogMatchingPort` + DTOs de contrato en `invoices/application/ports` (sin import `catalog/domain` en application); verificar compile
- [x] 2.3 Implementar adapter en `invoices/adapters/linking` contra `catalog.Application`; verificar test stub/integration mínimo
- [x] 2.4 Inyectar puerto en `invoices.NewApplication` / composition root; verificar wire API arranca

## 3. Backend invoices — persistencia y use cases

- [x] 3.1 Implementar en repo postgres invoices: `SaveLineLink`, `ListReviewLines`, `GetLineForDecision`, `SyncHeaderLinkingStatus`; verificar tests repo/SQL equivalentes a los actuales de catalog
- [x] 3.2 Implementar `ListReviewQueueQuery` (enriquece nombres vía `CatalogService`); verificar test líneas unmatched/suggested → DTO con nombres
- [x] 3.3 Implementar `ApplyLineDecisionCommand` delgado (dominio + ACL secuencia mint/memory → persist, sin TX cross-aggregate); verificar tests link+remember, never_match, create_provisional, fallo ACL sin mutar línea
- [x] 3.4 Añadir logs `invoices.apply_line_decision` y propagación de errores según design §10; verificar en tests o assert de campos de log si el proyecto lo permite
- [x] 3.5 Registrar `GET /api/v1/invoicing/review-queue` y `POST /api/v1/invoicing/invoices/{invoiceId}/lines/{lineId}/decisions`; verificar router tests HTTP

## 4. Backend catalog — retirar ownership de cola/decisions

- [x] 4.1 Eliminar de `CatalogRepository` métodos SQL sobre `invoice_*` y puerto `InvoiceLineLinkRepository`; verificar `catalog` compila
- [x] 4.2 Retirar `RememberDecision` / `LinkInvoiceLine` / `ListReviewQueue` del HTTP y wire catalog; verificar router test rutas ausentes/404
- [x] 4.3 Limpiar commands/queries catalog de use cases solo-HTTP de linking; conservar `ResolveInvoiceLine`; verificar tests resolve existentes

## 5. PWA

- [x] 5.1 Mover `app-catalog-linker` a `invoices/presentation` (o `shared`); actualizar imports en detail/review; verificar compile
- [x] 5.2 Añadir HTTP + store review/decisions bajo `invoices` (`/api/v1/invoicing/review-queue`, `/api/v1/invoicing/invoices/{invoiceId}/lines/.../decisions`); verificar service/store
- [x] 5.3 Crear página `/invoices/review` + CTA en maestro facturas; verificar `app.routes.ts` y navegación
- [x] 5.4 Eliminar `/catalog/review`, CTA y métodos review/decisions de catalog store/HTTP; verificar sin referencias a rutas catalog de cola/decisions

## 6. Verificación cruzada

- [x] 6.1 `pnpm --filter @bowerbird/backend test` (catalog+invoices) en verde
- [x] 6.2 `pnpm --filter @bowerbird/pwa lint` (y tests unitarios PWA si aplican) en verde
- [x] 6.3 Smoke: cola en Facturas; decision desde cola y detalle actualiza `linking_status`; fallo validate item no muta línea; Catálogo sin cola
