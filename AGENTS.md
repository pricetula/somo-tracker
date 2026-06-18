Somotracker Application
This document establishes the directories available for our application.

📂 1. Monorepo Structural Blueprint
The Somotracker platform is organized as a single, unified monorepo. Agents must strictly isolate code changes to their respective top-level directories:

├── .github/          # CI/CD workflows and automation pipelines
├── ./backend/        # Go (Fiber) REST API — Core business logic & analytics engine
├── ./docs/           # Standardized Markdown templates and system feature specs
├── ./frontend/       # Next.js v16+ (App Router) — Multi-tenant educational dashboard
└── ./public/         # Svelte — High-conversion, lightweight marketing website

📝 2. Changelog & Notable Migrations

| Date | Change |
|------|--------|
| 2026-06-12 | **Middleware → Proxy**: Renamed `frontend/middleware.ts` → `frontend/proxy.ts` and export `middleware()` → `proxy()` per Next.js v16 deprecation. See https://nextjs.org/docs/messages/middleware-to-proxy |
| 2026-06-16 | **Migration squash**: Merged `000003` (`is_final`) and `000004` (`stream`) into `000001_initial_schema.up.sql` as inline column declarations. All schema changes must now go directly into `000001`. See §4. |

# Agent Directive: Documentation & Tooltip Synchronization

You must preserve strict architectural synchronization across this workspace.

### Core Rules:
1. All contextual inline UI assistance MUST derive from `content/docs/*.mdx` frontmatter via `<FeatureHelp slug="filename" anchorId="heading-anchor" />`.
2. NEVER hardcode descriptive explanations inside your UI markup or labels.
3. Every doc file must declare a clean, concise, markdown-free `tooltipSummary` string under 160 characters.

### Verification Cycle:
Prior to completing any task involving routing modifications, settings UI adjustments, or backend flag configuration updates, you MUST successfully run:
```bash
npm run audit:docs
```
If errors occur, fix the misalignments immediately before pushing code patches.

---

🗄️ 4. Database Migration Policy

All database schema changes **must** be made directly to the single migration file:

- `backend/internal/database/migrations/000001_initial_schema.up.sql`

**Do NOT create new migration files.** Instead, modify the initial schema file directly:
- Add new columns inline in `CREATE TABLE IF NOT EXISTS` statements
- Add new tables inline in the same file
- Add new indexes, constraints, or views inline

This policy ensures a single-source-of-truth for the schema. Seed data (`000002_seed.up.sql`) is the only exception — it remains a separate file since it's data population, not schema DDL.

When adding columns, always declare them directly in the `CREATE TABLE` statement rather than using `ALTER TABLE … ADD COLUMN` for tables defined in this file. For tables owned by future extensions, use `ALTER TABLE` with `IF NOT EXISTS` guards.

---

⚡ 6. React State-in-Effect Policy

Calling `setState` synchronously within a `useEffect` body causes cascading renders that hurt performance. Effects are intended to synchronize React with external systems (DOM, subscriptions, platform APIs) — not to derive state from other state.

### Core Rules:
1. **Never call `setState` inside `useEffect`.** Derive state from props/state using `useMemo` or compute inline during render.
2. **Use event handlers for reactive updates.** If a state change must trigger another state update (e.g., auto-suggesting end time when start time changes), do it in the event handler (e.g., `onValueChange`) — not in an effect.
3. **Prefer `useMemo` over `useEffect` + `setState`.** If a value can be computed from existing state/props, use `useMemo` instead of an effect that calls `setState`.

### Verification:
Run `pnpm lint` in `./frontend/` before pushing. The `react-hooks/set-state-in-effect` ESLint rule enforces this policy.

### Rationale:
Synchronous `setState` in `useEffect` forces React to re-render the component immediately after the effect runs, wasting a render cycle and potentially causing infinite loops. See the ESLint rule reference: https://react.dev/learn/you-might-not-need-an-effect

---

📦 5. Package Manager Policy
Both `./frontend/` (Next.js) and `./public/` (Svelte) **must** use **pnpm** as their sole package manager.

- **pnpm must be used exclusively.** Never use `npm install`, `yarn add`, or any other package manager in these directories.
- Lock files: Both projects maintain their own `pnpm-lock.yaml`. These must be committed and kept in sync with `dependencies`/`devDependencies`.
- Install commands:
  - `pnpm install` — install dependencies (local dev)
  - `pnpm install --frozen-lockfile` — CI / Docker (prevents lockfile drift)
  - `pnpm add <pkg>` — add a runtime dependency
  - `pnpm add -D <pkg>` — add a dev dependency
  - `pnpm remove <pkg>` — remove a dependency
- The `--ignore-scripts` flag should be used in CI/Docker builds to prevent postinstall attacks, unless a postinstall script is explicitly required (e.g. `sharp`).
- Global installs: Never install packages globally inside a project directory. Use `pnpm exec` or `pnpm dlx` for one-off commands.
- Workspace: If cross-project sharing is ever needed, configure pnpm workspaces via a root `pnpm-workspace.yaml` rather than duplicating packages.