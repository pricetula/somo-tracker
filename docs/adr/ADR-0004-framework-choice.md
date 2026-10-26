---
id: ADR-0004
title: Frontend Framework — Adopt TanStack Start
status: accepted
depends_on: []
touches: [frontend]
---

# ADR-0004 Frontend Framework — Adopt TanStack Start

## Context

The frontend was previously built on Next.js 14 (App Router) + shadcn/ui. This ADR records the decision to migrate to TanStack Start (TanStack Router + Vite) to achieve tighter TanStack ecosystem integration, faster cold starts, and a more conventional file-based routing model that aligns with the existing `@tanstack/react-query`, `@tanstack/react-table`, and `@tanstack/react-virtual` dependencies already in use.

The key drivers are:

- Unified data-fetching layer under TanStack Query
- File-based routing with a dedicated router (no App Router nested layouts)
- Vite-powered dev server (instant HMR, lighter than Next.js)
- Elimination of Next.js-specific concerns (middleware.ts, next.config.js, `next/dynamic` quirks, `next lint`/`next build` conventions)

## Options Considered

| Option                          | Description                                                                                                                                       | Rationale                                                                                                                                         |
| ------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| **1. Stay with Next.js**        | Rejected. Would not leverage existing TanStack Query/Table/Virtual investments.                                                                   | Next.js is stable but lacks first-class TanStack Router integration.                                                                             |
| **2. Switch to TanStack Start** | Adopted. Migrate the entire frontend to TanStack Start (TanStack Router + Vite + minimal shadcn/ui setup). Rewrite FE rules to TanStack Start conventions. | Best TanStack-native DX; Vite ergonomics; future-proof as TanStack investments grow. Requires re-architecting routing, middleware, and dev setup. |
| **3. Hybrid approach**          | Rejected. Maintaining two routing systems increases complexity without benefit.                                                                   | Incremental path adds migration risk without upside.                                                                                             |

## Decision

**Adopt TanStack Start** as the frontend framework. **Status: accepted** (human review complete).

**Rationale**: The project already depends heavily on TanStack packages (`@tanstack/react-query`, `@tanstack/react-table`, `@tanstack/react-virtual`). TanStack Start provides first-class integration with these, a file-based routing model that reduces "routing confusion" (App Router vs pages router), and Vite's fast dev loop. The migration cost is justified by long-term DX and alignment with the data-fetching layer.

## Consequences

| Area                     | Impact                                                                                                                                                                                                                                                    |
| ------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Routing**              | File-based `routes/` replaces `app/` routing. Intercepting routes `(.)[id]` / `@modal/(.)add` become `<Outlet />` + route params; parallel routes become route tree siblings.                                                                             |
| **Middleware / Session** | `middleware.ts` replaced by root `entry.server.ts` or `routes/$server.ts`; session check stays but via TanStack Router's `useLoaderData()` / `useNavigation()`.                                                                                           |
| **Dev Server**           | `next dev` → `vite dev` / `pnpm dev` (Vite). Build: `next build` → `vite build` / `pnpm build`. Start: `next start` → `vite preview` / `pnpm start`.                                                                                                       |
| **Config**               | `next.config.js` / `next-env.d.ts` removed. Tailwind remains (copy `tailwind.config.js`, `globals.css`). Vite config at `vite.config.ts`. Vite env uses `.env.*` with `VITE_` prefix.                                                                    |
| **shadcn/ui**            | `pnpm dlx shadcn@latest add` still works; components install into `src/components/ui/` as usual. No `asChild` rule (FE-018) stays; `render={<Component/>}` pattern used.                                                                                  |
| **FE Rules Rewrite**     | All FE-xxx rules must be reviewed/adapted to TanStack Start conventions: routing (FE-013, FE-014), data fetching (FE-007, FE-008, FE-009), forms (FE-020), UI (FE-015, FE-016, FE-018), dates (FE-023), charts (FE-022).                                            |
| **Build/ CI**            | `Makefile` targets: `build-frontend` changed from `next build` to `vite build` (`cd frontend && pnpm build`). `lint` changed from `next lint` to ESLint (`cd frontend && pnpm lint`). `typecheck` stays `tsc --noEmit`.                                 |
| **Deployment**           | Existing Dockerfile references `next start` — updated to `pnpm start` (Vite preview). CI pipelines referencing Next.js artifacts updated.                                                                                                                |
| **Risk**                 | TanStack Start is at v0.x / early adoption breaking changes possible. Existing tests (Jest + JSDOM for Next.js) may need adapter changes. Learning curve for TanStack Router patterns.                                                                    |

## Status

**accepted** — Implementation may proceed.

---

_This ADR depends on [FEATURE-xxx] (to be filed when the migration spec is written) and touches the entire `frontend/` layer. All FE-xxx rule updates will follow in a separate spec._