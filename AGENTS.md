# Somotracker — Root Agent Contract

## Monorepo Structure

```
├── .github/      # CI/CD workflows
├── backend/      # Go (Fiber) REST API
├── docs/         # Markdown templates and feature specs
├── frontend/     # Next.js (App Router) — multi-tenant educational dashboard
└── public/       # SvelteKit + shadcn-svelte — marketing website
```

**Strictly isolate all changes to their respective top-level directory.** Read that directory's `AGENTS.md` for rules before making any changes.
