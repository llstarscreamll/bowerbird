# Development quality

## Agent skills

Prefer curated skills (frontend, performance, architecture, security) from trusted sources such as [tech-leads-club/agent-skills](https://github.com/tech-leads-club/agent-skills). Base skill often used: `GoogleChrome/modern-web-guidance`.

### Installing Skills

To configure your agent environment, you must install the following core skills:

**TLC Agent Skills (Tech Leads Club):**

```bash
# Documentation writing and editing
opencode skill add https://github.com/tech-leads-club/agent-skills/tree/main/docs-writer
# Coding guidelines to prevent common LLM mistakes
opencode skill add https://github.com/tech-leads-club/agent-skills/tree/main/coding-guidelines
# Tactical DDD detection and refactoring
opencode skill add https://github.com/tech-leads-club/agent-skills/tree/main/tactical-ddd
# Modular architecture and boundary design
opencode skill add https://github.com/tech-leads-club/agent-skills/tree/main/modular-design-principles
# Feature planning and implementation with EARS notation
opencode skill add https://github.com/tech-leads-club/agent-skills/tree/main/tlc-spec-driven
# Cloud advisory for architecture and security
opencode skill add https://github.com/tech-leads-club/agent-skills/tree/main/aws-advisor
```

**OpenSpec:**

```bash
npm install -g @openspec/cli
openspec install --git
```

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
