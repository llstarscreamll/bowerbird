# Development quality

## Agent skills

Prefer curated skills (frontend, performance, architecture, security) from trusted sources such as [tech-leads-club/agent-skills](https://github.com/tech-leads-club/agent-skills).

### Installing Skills

To configure your agent environment, you must install the following core skills:

**Google Chrome Guidance:**

```bash
# For Opencode:
opencode skill add https://github.com/GoogleChrome/modern-web-guidance

# For Cursor, Roo Code (Cline), or Windsurf, append the rules to your project config:
curl -sL https://raw.githubusercontent.com/GoogleChrome/modern-web-guidance/main/guidance.md >> .cursorrules
# (replace .cursorrules with .clinerules or .windsurfrules as needed)
```

**TLC Agent Skills (Tech Leads Club):**

```bash
# The agent-skills CLI auto-detects and supports Cursor, Roo Code, Windsurf, and Copilot.
npx @tech-leads-club/agent-skills install -s docs-writer coding-guidelines tactical-ddd modular-design-principles tlc-spec-driven aws-advisor
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
