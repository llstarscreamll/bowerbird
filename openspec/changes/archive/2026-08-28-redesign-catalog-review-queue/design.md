## Context

El frontend actualmente requiere escribir el ID crudo de un ítem para vincular una línea (tanto en `catalog/review-queue` como en `invoices/detail`). Además, las queries de backend exponen las sugerencias (`suggestions`) y los IDs vinculados, pero no realizan joins para obtener los nombres reales, delegando al frontend una carga imposible de N+1 peticiones.

## Goals / Non-Goals

**Goals:**

- Implementar búsqueda interactiva de ítems en los componentes de catálogo y facturas (Combobox).
- Exponer los nombres de los ítems en las respuestas del backend (DTOs) para `ListReviewQueueQuery` y `GetInvoiceByIDQuery`.
- Introducir acciones UI para "Vincular", "Rechazar" y "Nuevo" con soporte pleno desde el backend.
- Mostrar el `linking_status` a nivel de cabecera en la tabla maestra de facturas.

**Non-Goals:**

- Cambiar la lógica del motor de coincidencia suave (soft matcher).
- Introducir paginación avanzada en la cola de revisión.
- Violar el aislamiento de las bases de datos (reach-through persistence).

## Decisions

**Decisión 1: Enriquecimiento de Nombres de Ítems sin Reach-Through (Backend)**

- **Enfoque**: Para la cola de revisión (`catalog`), el `CatalogRepository` puede hacer JOIN interno ya que es dueño de las tablas. Para `GetInvoiceByIDQuery` (`invoices`), el repositorio de facturas **no** hará JOIN con las tablas de catálogo. En su lugar, el Application Layer de `invoices` definirá un puerto (ej. `ports.CatalogService`) para inyectar los nombres de los ítems en memoria (ACL) o el DTO de facturas delegará la resolución al cliente.
- **Rationale**: Principio de **State Isolation** y **Well-defined boundaries**. Las consultas directas de un módulo a las tablas de otro (reach-through) crean acoplamiento fuerte y deuda técnica (P0/P1 violation).

**Decisión 2: Componente UI Autónomo (Frontend Composability)**

- **Enfoque**: Crear un componente de presentación inteligente `app-catalog-linker` dentro del módulo `catalog` (exportado en su public API).
- **Rationale**: Principios de **Composability** e **Independence**. El módulo `invoices` consumirá este componente pasándole solo la evidencia (la línea), y el componente internamente interactuará con el `CatalogStore` y los endpoints del catálogo (`GET /items?search=...`, link, reject). `invoices` no necesita conocer las reglas de negocio del catálogo.

**Decisión 3: Búsqueda Visual**

- **Enfoque**: Usar Spartan UI `hlm-command` (o selector con debounce) encapsulado en `app-catalog-linker`.
- **Rationale**: Mitiga la mala UX de IDs manuales con un selector asíncrono y escalable.

## Risks / Trade-offs

- **Risk**: Sobrecarga de comunicación inter-módulos en el backend para enriquecer `GetInvoiceByIDQuery`.
  - **Mitigation**: Dado que las sugerencias precalculadas son pocas, el puerto `CatalogService.GetItemNames(ids)` hará una sola consulta batch (Bulkhead / in-memory join) minimizando la latencia.
- **Risk**: Doble dependencia o estado compartido en el frontend.
  - **Mitigation**: El componente `app-catalog-linker` aislará su estado y peticiones. Empezará y terminará la acción por sí solo, emitiendo un evento genérico `(resolved)` para que el contexto anfitrión (`invoices` o `review-queue`) refresque sus propios datos si lo desea.
