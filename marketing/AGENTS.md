# Marketing Site — Agent Rules

Status: skeleton / draft
Scope: Astro static site (latest), not on docker-compose.
Layer contract: root AGENTS.md covers commit policy and doc-led workflow.

## Build / Quality

- Build: `pnpm build` (Astro static output to `dist/`)
- Lint / format / typecheck are part of `make lint` / `make compliance`
- No `node_modules` / `.astro-cache` / `.cache` / `dist` in git

## Architecture

- `src/pages/` only; no custom server
- `astro.config.mjs` outputs static
- Not exposed under `/api`; no backend endpoints
- No PII / secrets in site source or analytics

## Compliance refs

- FE-007 / FE-011 / FE-015 / FE-016 / FE-024 where they apply to public landing pages
- All features require `status: approved` spec before development
