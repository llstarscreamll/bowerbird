## 1. Documentation

- [x] 1.1 Update `docs/domain/GLOSSARY.md` to map `Party` to "Contacto / Tercero" and remove references to "Contraparte".

## 2. PWA Refactoring (Parties UI)

- [x] 2.1 Edit `apps/pwa/src/app/core/presentation/layouts/tenant-layout/tenant-layout.component.ts` to rename "Contrapartes" to "Contactos" in the side navigation menu and tooltips.
- [x] 2.2 Edit `apps/pwa/src/app/parties/presentation/pages/master/master.page.ts` to update the main view title and primary action buttons (e.g. "Nueva Contraparte" -> "Nuevo Contacto").
- [x] 2.3 Edit `apps/pwa/src/app/parties/application/parties.store.ts` to update all ToastService success/error messages from "Contraparte(s)" to "Contacto(s)".

## 3. PWA Refactoring (Invoices UI)

- [x] 3.1 Edit `apps/pwa/src/app/invoices/presentation/pages/detail/detail.page.ts` to remove visual references to "Contraparte" and replace them with "Contacto".

## 4. Verification

- [x] 4.1 Run `pnpm run lint` and `pnpm --filter @bowerbird/pwa build` to ensure no syntax errors were introduced by the string replacements.
