# Spartan UI — PWA conventions

[Spartan UI](https://spartan.ng/) (brain + helm), Slate base, indigo primary.

## Layout

- Helm under `apps/pwa/src/app/shared/ui/` (`@spartan-ng/cli`)
- Import `@spartan-ng/helm/<component>`
- Lucide icons in `shared/icons/app-icons.ts`, provided in `app.config.ts`

Installed helm: `alert`, `alert-dialog`, `avatar`, `badge`, `button`, `card`, `checkbox`, `dialog`, `dropdown-menu`, `empty`, `field`, `input`, `item`, `label`, `progress`, `scroll-area`, `separator`, `sheet`, `sidebar`, `skeleton`, `sonner`, `spinner`, `table`, `tooltip`, `utils`.

## Shared presentation

| Component                    | Path                                                   | Notes                                               |
| ---------------------------- | ------------------------------------------------------ | --------------------------------------------------- |
| `app-file-upload`            | `core/presentation/components/file-upload/`            | Types from barrel; optional `[validateFile]`        |
| `app-connection-status-chip` | `core/presentation/components/connection-status-chip/` | Status helpers in `core/domain` + presentation maps |

`ThemeService` watches `document.documentElement` `dark` for embedded views (email iframe).

## Theming

CSS variables in `src/styles.css`. Dark mode via `classList.toggle('dark', …)` (tenant layout).

## Feedback

| Use                 | Component                                   |
| ------------------- | ------------------------------------------- |
| Global toast        | `ToastService` → Sonner / `<hlm-toaster />` |
| Inline errors       | `<hlm-alert variant="destructive">`         |
| Modal               | `<hlm-dialog>`                              |
| Destructive confirm | `<hlm-alert-dialog>`                        |

## Add a component

From `apps/pwa`:

```bash
npx ng g @spartan-ng/cli:ui <name> --defaults --directory=src/app/shared/ui
```

## File upload

No Spartan dropzone — use `app-file-upload` (card + item + progress + scroll-area + empty + alert). Optional `[maxFileSizeBytes]`, `[maxFiles]`, clear/retry outputs.

## Icons

```html
<ng-icon name="lucideInbox" />
```

Register new Lucide icons in `app-icons.ts`.

## Prettier

Tailwind functions: `hlm`, `cva`, `classes` (root `.prettierrc.json`).
