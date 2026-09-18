# Design: catalog-master-header-actions

## 1. Problem Statement

El header del maestro de catálogo (`MasterPage`) pone cuatro acciones heterogéneas en un `flex-wrap` junto al título. Con sidebar + `max-w-5xl` los labels en español se apilan; un quinto botón no cabe. `Resolver duplicados` ocupa chrome permanente incluso con 0 clusters.

## 2. Technical Design (Evolutionary Modular Architecture)

Plataforma: modular monolith existente (`internal/<bc>/`, PWA Angular por feature). P11–P19 ceden ante AGENTS.md. **No módulo nuevo, no backend, no OHS, no eventos.**

### Scope (Phase 0)

Cambio de presentación en el BC `catalog`. Los aggregates, HTTP y stores no cambian. Solo reorganiza chrome del maestro.

### Boundaries (Phase 1–2)

| Contexto   | Tipo | Este change                              |
| ---------- | ---- | ---------------------------------------- |
| `catalog`  | Core | Reorganizar acciones del maestro (PWA)   |
| `invoices` | Core | Fuera. Merge bar y clusters siguen igual |

```
PWA MasterPage
  ├─ CatalogStore          (items, duplicateClusters)  — sin cambio de contrato
  └─ CatalogImportStore    (dialog, activeImport)      — sin cambio de contrato
           │
           └── HTTP catalog (ya existente)
```

Estado: `catalog` sigue siendo único writer de `catalog_items` / `catalog_imports`. La UI no inventa un agregado “Toolbar”.

Tactical intensity: **mínima**. No hay invariante de dominio; es layout. No extraer componente compartido de page-header (YAGNI: solo este maestro).

### Ubiquitous language (UI)

- **Nuevo** — create manual (`creation_source=manual`). Única primaria del header.
- **Cargar** — familia de import CSV (`CatalogImport`). Dialog existente + historial `/catalog/imports`.
- **Duplicados pendientes** — excepción de identidad (`duplicate-clusters`). Banner, no acción de listado.

### Layout (Phase 3 — presentation only)

`apps/atta/web/src/app/catalog/presentation/pages/master/master.page.ts`. Feature convention: store orquesta, página delgada.

```
┌─────────────────────────────────────────────────────────┐
│ Catálogo                         [Cargar ▾]  [Nuevo]    │
│ Ítems vinculados…                                       │
│                                                         │
│ ⚠ N grupos duplicados                         [Revisar] │  ← solo si n > 0
│ ℹ Carga masiva {estado}                       [Ver]     │  ← sin cambio
│                                                         │
│ [buscar…] [Buscar]                                      │
│ [tabla…]                                                │
└─────────────────────────────────────────────────────────┘
```

**Header**

- Título + subtítulo a la izquierda.
- Derecha, `flex` **sin wrap**: split `Cargar` + `Nuevo` (`hlmBtn` default).
- Quitar los `<a>` permanentes `Resolver duplicados` y `Cargas masivas`.

**Split Cargar** (compose, no Helm nuevo)

- No instalar `button-group`. Inbox ya usa `HlmDropdownMenuImports`.
- Botón outline `Cargar` → `imports.openDialog()` (mismo dialog CSV).
- Botón outline `size="icon"` con chevron, `aria-label` “Más acciones de carga”, `[hlmDropdownMenuTrigger]`.
- Ítem de menú: enlace `Cargas masivas` → `routerLink="imports"`.
- Plantilla CSV se queda **dentro** del dialog (no duplicar en el menú).

**Banner duplicados**

- Mismo patrón que el alert de import activo: `<a [routerLink]="['duplicates']">` + `hlmAlert`.
- Visible **solo** si `store.duplicateClusters().length > 0`.
- Título: “Duplicados pendientes”. Descripción: count + “grupos que parecen el mismo producto”.
- Orden bajo el header: duplicados → import activo → error de store.

**Fuera**

- Barra sticky de fusión (2–5 seleccionados): intacta.
- Páginas `/catalog/imports`, `/catalog/duplicates`, `/catalog/new`: intactas.
- Facturas / Contactos: fuera de este change.

### Communication / resilience (Phase 4–5)

Nada. Sin puertos, ACL, outbox ni retries nuevos. `loadDuplicateClusters()` / `loadImports()` en `ngOnInit` se mantienen.

### Stack (Phase 6)

Angular standalone + Spartan Helm (`hlmBtn`, `hlmAlert`, `hlm-dropdown-menu`, `ng-icon`). Sin HTML de arquitectura de plataforma: el artefacto de este schema es `design.md`.

### BDD (código, no Markdown)

E2E Playwright browser (no hay specs de catálogo en `tests/browser/` hoy). Sembrar ítems con `platformApi` (fixture ya extendida).

### Non-goals

Overflow genérico “Más”, tabs Ítems/Cargas/Duplicados, toolbar en la fila de search (opción 3), icon-only, extraer header compartido, backend, Helm `button-group`.
