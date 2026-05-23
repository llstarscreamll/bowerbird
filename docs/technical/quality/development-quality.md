# Calidad de desarrollo

## AI Skills (OpenCode)

Para mantener calidad y consistencia, se recomienda instalar skills de OpenCode para frontend, performance, arquitectura y seguridad.

- Skill base recomendada: `GoogleChrome/modern-web-guidance`.
- Instalar skills solo desde fuentes oficiales o depuradas.
- Fuente curada recomendada: `https://github.com/tech-leads-club/agent-skills`.

Instalacion desatendida:

```bash
npm i -g @tech-leads-club/agent-skills
npx skills add GoogleChrome/modern-web-guidance
agent-skills install -s \
  accessibility \
  aws-advisor \
  best-practices \
  chrome-devtools \
  coding-guidelines \
  core-web-vitals \
  docs-writer \
  domain-analysis \
  frontend-blueprint \
  mermaid-studio \
  modular-decomposition \
  modular-design-principles \
  perf-astro \
  perf-lighthouse \
  perf-web-optimization \
  playwright-skill \
  security-best-practices \
  sentry \
  seo \
  solo-founder-gtm \
  tactical-ddd \
  technical-design-doc-creator \
  tlc-spec-driven
```

## Formateo de código

- Prettier configurado en raíz (`.prettierrc.json`)
- Convenciones principales: `singleQuote`, `trailingComma: all`, `printWidth: 120`
- Alcance principal: TypeScript y Markdown

Comandos:

- `pnpm run format`
- `pnpm run format:check`

## Git Hooks (Husky + lint-staged)

Para garantizar que ningún código sin formatear o con errores de análisis estático llegue al repositorio, utilizamos **Husky** y **lint-staged**.

Esta es la configuración de nivel industrial más reputada para monorepos:

- **Husky** intercepta el evento `pre-commit`.
- **lint-staged** ejecuta los formateadores (`prettier`, `gofmt`) _únicamente_ sobre los archivos modificados que están en _staging_. Esto hace que el commit sea instantáneo.
- Finalmente, se ejecuta `pnpm run lint` (orquestado por Turbo) para validar TypeScript y Go vet en todo el proyecto antes de permitir el commit. Gracias al caché de Turbo, esto toma milisegundos si no hay errores nuevos.

**Archivos de configuración:**

- `.husky/pre-commit` (hook principal)
- `.lintstagedrc.json` (reglas por extensión)
