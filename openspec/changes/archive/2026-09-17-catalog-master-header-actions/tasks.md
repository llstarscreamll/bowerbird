# Tasks: catalog-master-header-actions

## 1. Code Specifications (BDD as Code)

- [x] 1.1 Playwright browser `apps/atta/e2e/tests/browser/catalog/master-header.spec.ts`: maestro sin clusters — heading Catálogo; header con `Nuevo` y `Cargar`; sin link/botón `Resolver duplicados`; sin link `Cargas masivas` en el header; `Cargar` abre el dialog; chevron “Más acciones de carga” muestra `Cargas masivas` y navega a `/catalog/imports`
- [x] 1.2 Mismo spec: sembrar dos ítems con el mismo nombre vía `platformApi`; el maestro muestra alert `Duplicados pendientes`; el enlace lleva a `/catalog/duplicates`; el header sigue sin `Resolver duplicados`

## 2. Implementation

- [x] 2.1 Header de `master.page.ts`: quitar wrap de cuatro botones; izquierda título; derecha `flex` nowrap `[Cargar][▾] [Nuevo]`; `Cargar` → `imports.openDialog()`
- [x] 2.2 Chevron `size="icon"` + `HlmDropdownMenuImports`; menú con `Cargas masivas` → `routerLink="imports"`; no duplicar plantilla CSV
- [x] 2.3 Alert `Duplicados pendientes` (mismo patrón que import activo) solo si `duplicateClusters().length > 0`; `routerLink` a `duplicates`; orden: duplicados → import activo → error

## 3. Verification

- [x] 3.1 `mise run test:full`
