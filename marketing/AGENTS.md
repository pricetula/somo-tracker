# Marketing Site — Agent Rules

Status: skeleton / draft
Scope: Astro static site (latest), not part of `docker-compose.yml`.
Layer contract: root AGENTS.md covers the document-led workflow, commit policy, and monorepo-wide rules. Library versions: see `marketing/package.json`.

## Application boundaries

This file contains marketing-specific rules. Repository-wide rules (commit policy, document-led workflow, cross-application coordination, security principles) live in the root `AGENTS.md` and apply here too. Application-specific rules do not override repository-wide requirements.

## Build / quality

- Build: `pnpm build` (Astro static output to `dist/`)
- Dev: `pnpm dev`
- Preview: `pnpm preview`
- Lint: `pnpm lint`
- Format: `pnpm format`
- Typecheck: `pnpm typecheck` (`astro check`)
- Lint/format/typecheck are included in `make lint` and `make compliance`.
- No `node_modules`, `.astro-cache`, `.cache`, or `dist` in git.

## Architecture

- `src/pages/` only; no custom server.
- `astro.config.mjs` outputs static.
- Not exposed under `/api`; no backend endpoints.
- No PII or secrets in site source or analytics.

## UI components

- **MK-001** One component per file, named after the file. No second component inside it, even a private one; extract it into its own file. The frontend enforces the same rule (FE-006 in `frontend/AGENTS.md`).

## Compliance references

Where they apply to public landing pages, the following frontend rules are referenced from `frontend/AGENTS.md` (they are frontend-owned; do not duplicate them here): FE-007, FE-011, FE-015, FE-016, FE-024.

## Document-led workflow

- All features require a `status: approved` specification before development. Do not implement draft or in-progress specs.
- Root AGENTS.md covers the document-led workflow in full; this file adds nothing beyond the reference above.

## Definition of done

`pnpm lint`, `pnpm typecheck` and `pnpm build` pass (covered by `make lint` / `make compliance` at repo root).
