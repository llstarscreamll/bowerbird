# Development quality

## Agent skills

Prefer curated skills (frontend, performance, architecture, security) from trusted sources such as [tech-leads-club/agent-skills](https://github.com/tech-leads-club/agent-skills). Base skill often used: `GoogleChrome/modern-web-guidance`.

## Formatting

- Prettier at repo root (`.prettierrc.json`): `singleQuote`, `trailingComma: all`, `printWidth: 120`
- Scope: TypeScript and Markdown (`pnpm run format` / `format:check`)
- Go: `gofmt` via lint-staged

## Git hooks

Husky `pre-commit`:

1. `lint-staged` (Prettier / gofmt on staged files)
2. `pnpm run lint` (Turbo; cache keeps it fast)
3. Optional `codegraph sync`

Config: `.husky/pre-commit`, `.lintstagedrc.json`
