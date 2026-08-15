# Product docs

Capture the _what_ and _why_ for business features. Put implementation detail in `docs/technical/`.

## Conventions

- One file per feature under `docs/product/features/`.
- Prefer names like `YYYY-MM-slug.md`.
- Include scope, actors, rules, acceptance criteria, and metrics.
- Link technical docs instead of copying them.

## Layout

| Path          | Role                 |
| ------------- | -------------------- |
| `features.md` | Living catalog       |
| `features/`   | Feature specs        |
| `_templates/` | New-feature template |

## New feature flow

1. Copy `_templates/feature-spec.md`.
2. Agree acceptance criteria and metrics.
3. Link related `docs/technical/` pages.
4. Add a row to `features.md`.
