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

📦 3. Package Manager Policy
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