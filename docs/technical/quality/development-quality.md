# Development quality

## Local setup script

Prefer the automated installer for a new machine or after pulling toolchain/
agent-tooling changes:

```bash
pnpm run setup:local
```

Equivalent:

```bash
./scripts/setup-local.sh
```

Requires `mise` on `PATH`. The script installs mise tools, workspace
dependencies (`pnpm install`), agent skills, and MCP CLIs below, then checks
project MCP config files. Details: [Getting started](../getting-started.md).

Manual commands in the following sections remain the source of truth when you
need to install or refresh a single skill/MCP.

## Agent skills

Prefer curated skills (frontend, performance, architecture, security) from trusted sources such as [tech-leads-club/agent-skills](https://github.com/tech-leads-club/agent-skills).

### Installing Skills

`pnpm run setup:local` installs these. To configure them manually:

Generated agent skill/command copies under `agent/`, `.agents/`, `.claude/`,
`.cursor/` (except `mcp.json`), `.opencode/`, and `.windsurf/` are gitignored —
refresh via `setup:local`. Track `skills-lock.json` for reproducible `skills`
restores. Track `openspec/` (config, specs, changes) and root `opencode.json`.

**Google Chrome Guidance:**

```bash
npx skills add GoogleChrome/modern-web-guidance --all
```

**Spartan UI (Angular / Helm):**

```bash
npx skills add spartan-ng/spartan --all
```

**TLC Agent Skills (Tech Leads Club):**

```bash
# The agent-skills CLI auto-detects and supports Cursor, Roo Code, Windsurf, and Copilot.
npx @tech-leads-club/agent-skills install -s docs-writer coding-guidelines tactical-ddd modular-design-principles tlc-spec-driven aws-advisor
```

**OpenSpec:**

```bash
npm install -g @fission-ai/openspec@latest
openspec init --tools cursor,opencode,claude,agents --no-animation --force
```

## MCP servers

### Installing MCP servers

`pnpm run setup:local` installs these. To install manually:

**Spartan UI:**

```bash
npm install -g @spartan-ng/mcp
```

Project registration (requires the global binary above):

- Cursor: `.cursor/mcp.json`
- OpenCode: `opencode.json` (`mcp.spartan-ui`)

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
