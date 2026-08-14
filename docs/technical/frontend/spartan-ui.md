# Spartan UI — PWA conventions

The Angular PWA uses [Spartan UI](https://spartan.ng/) (brain + helm) with a Slate base theme and indigo primary accent.

## Layout

- Helm components live under `apps/pwa/src/app/shared/ui/` (generated via `@spartan-ng/cli`).
- Import from `@spartan-ng/helm/<component>` (path alias configured in `tsconfig.json`).
- Shared Lucide icons are registered in `src/app/shared/icons/app-icons.ts` and provided in `app.config.ts`.

### Installed helm components

Only add primitives you actually use. Currently installed:

`alert`, `alert-dialog`, `avatar`, `badge`, `button`, `card`, `checkbox`, `dialog`, `dropdown-menu`, `empty`, `field`, `input`, `item`, `label`, `progress`, `scroll-area`, `separator`, `sheet`, `sidebar`, `skeleton`, `sonner`, `spinner`, `table`, `tooltip`, `utils`

Removed unused packages: `select`, `tabs`, `switch`, `typography`.

## Shared presentation components

Cross-feature UI that does not belong in a single feature module:

| Component                    | Path                                                   | Notes                                                                                      |
| ---------------------------- | ------------------------------------------------------ | ------------------------------------------------------------------------------------------ |
| `app-file-upload`            | `core/presentation/components/file-upload/`            | Import types from the barrel `index.ts`; keep `[validateFile]` validation in the component |
| `app-connection-status-chip` | `core/presentation/components/connection-status-chip/` | Labels/variants from `core/presentation/connection-status/`                                |

`ThemeService` (`core/services/theme.service.ts`) observes the `dark` class on `document.documentElement` for embedded views (e.g. email iframe).

Shared domain for cross-feature status: `ConnectionStatus` and `resolveConnectionStatus()` live in `core/domain/connection-status.model.ts`. Presentation mappings (badge labels, icons) live in `core/presentation/connection-status/`.

## Theming

- CSS variables in `src/styles.css` (`--primary`, `--sidebar-*`, etc.).
- Dark mode: `document.documentElement.classList.toggle('dark', …)` (see `TenantLayoutComponent`).

## Feedback

| Use case                    | Component                                                                   |
| --------------------------- | --------------------------------------------------------------------------- |
| Global toast (5xx, success) | `ToastService` (`showSuccess`, `showError`, …) → Sonner + `<hlm-toaster />` |
| Inline form/page errors     | `<hlm-alert variant="destructive">`                                         |
| Modal                       | `<hlm-dialog>` + `*brnDialogContent`                                        |
| Destructive confirm         | `<hlm-alert-dialog>`                                                        |

## Adding components

From `apps/pwa`:

```bash
npx ng g @spartan-ng/cli:ui <name> --defaults --directory=src/app/shared/ui
```

Ensure `components.json` exists at the app root.

## File upload

Spartan UI has no native dropzone or file-upload primitive. Use `app-file-upload` in `core/presentation/components/file-upload/`, composed from:

- `hlm-card` — dropzone
- `hlm-item` + `hlm-badge` — queue rows
- `hlm-progress` — upload progress
- `hlm-scroll-area` + `ng-scrollbar` — scrollable queue
- `hlm-empty` — no files selected
- `hlm-alert` — rejected file types (via optional `[validateFile]` input)

Optional inputs for standard multi-file behavior:

- `[maxFileSizeBytes]` — per-file size limit (default 1 GB)
- `[maxFiles]` — max items in queue
- `(clearAllRequested)` / `(retryRequested)` — bulk clear and retry failed uploads

Add new icons to `app-icons.ts` when the upload UI needs them.

## Icons

Use Lucide names registered in `app-icons.ts`:

```html
<ng-icon name="lucideInbox" />
```

When adding new icons, import from `@ng-icons/lucide` in `app-icons.ts` and add to `provideIcons`.

## Prettier

Tailwind function names: `hlm`, `cva`, `classes` (see root `.prettierrc.json`).
