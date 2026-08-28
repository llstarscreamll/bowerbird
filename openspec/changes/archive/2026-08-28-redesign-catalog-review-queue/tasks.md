## 1. Backend: Módulo Catalog (Ownership y Búsqueda)

- [x] 1.1 Actualizar DTO `ports.ReviewLine` en el módulo `catalog` para incluir el nombre del ítem en sus sugerencias. Para esto, se debe crear un DTO `EnrichedSuggestion` en la capa de aplicación/puertos y NO modificar el Value Object `domain.Suggestion` del núcleo del dominio.
- [x] 1.2 Modificar `CatalogRepository.ListReviewLines` (`catalog_repository.go`) para hacer el JOIN con `catalog_items` y popular los nombres (es seguro porque son tablas del mismo contexto).
- [x] 1.3 Actualizar el endpoint `/api/v1/catalog/review-queue` (`catalog/adapters/http/v1/controller.go`) para emitir los nuevos DTOs enriquecidos.
- [x] 1.4 Crear el endpoint de búsqueda `GET /api/v1/catalog/items?search={query}` y su respectiva query de Application (si no existe o es deficiente).

## 2. Backend: Módulo Invoices (ACL y Enriquecimiento)

- [x] 2.1 Definir el puerto `ports.CatalogService` en `invoices/application/ports` con el método `GetItemNames(ctx, ids []string) (map[string]string, error)` (Anti-Corruption Layer).
- [x] 2.2 Implementar este puerto en la capa de adaptadores (ej. en el archivo `wire.go` de invoices, inyectando un adaptador que llama a la API del módulo catalog en memoria).
- [x] 2.3 Modificar `GetInvoiceByIDQuery.Execute` (`invoices/application/queries/get_invoice_by_id.go`) para extraer los IDs de sugerencias de las líneas, llamar a `CatalogService.GetItemNames` (batch), y enriquecer las `Suggestions` en memoria sin hacer SQL reach-through a tablas del catálogo.

## 3. Frontend: Componente Composible de Catálogo

- [x] 3.1 Crear el componente `app-catalog-linker` dentro de `apps/pwa/src/app/catalog/presentation/components/`.
- [x] 3.2 Implementar en `app-catalog-linker` el autocompletado (Spartan UI `hlm-command`) consumiendo el endpoint de búsqueda con debounce (via `CatalogStore` o servicio dedicado).
- [x] 3.3 Conectar los botones y acciones internas del linker (vincular, rechazar, nuevo provisional) al `CatalogStore`, emitiendo un Output `(resolved)` cuando termine.

## 4. Frontend: Integración en Contextos Anfitriones

- [x] 4.1 Refactorizar `review.page.ts` (Cola de Revisión en `catalog`) para usar `app-catalog-linker` en cada línea pendiente, eliminando el input manual.
- [x] 4.2 Refactorizar `detail.page.ts` (Detalle en `invoices`) para delegar la sección de catálogo de las líneas no vinculadas a `app-catalog-linker` (Composability).
- [x] 4.3 Actualizar `master.page.ts` (Listado en `invoices`) para mostrar una nueva columna/badge con el estado global `linking_status` de cada factura.
