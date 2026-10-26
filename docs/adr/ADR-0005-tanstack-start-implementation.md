---
id: ADR-0005
title: TanStack Start Implementation Details
status: accepted
depends_on: [ADR-0004]
touches: [frontend]
---

# ADR-0005 TanStack Start Implementation Details

## Context
ADR-0004 adopted TanStack Start. This ADR records the concrete implementation decisions made to align the codebase with current TanStack Start v1.169 documentation.

## Decision
- Use `@tanstack/react-start` plugin `@tanstack/react-start/plugin/vite` with plugin order `tanstackStart()` before `viteReact()`.
- Project type `module`.
- Entry points: `src/entry.client.tsx`, `src/entry.server.tsx` using `createStartHandler`.
- Router created via `src/router.tsx` exporting `getRouter()`.
- Root document rendered via `src/routes/__root.tsx` with `HeadContent` and `Scripts`.
- Build output `.output/server/index.mjs`, start command `node .output/server/index.mjs`.
- Vite server `host:true` for Docker.
- API base path proxied in dev, backend reachable via service DNS `http://backend:8080` for SSR.
- Dockerfile uses multi-stage build with pnpm frozen lockfile, non-root user, copies `.output`.

## Consequences
- Frontend no longer uses `index.html` or manual `RouterProvider`.
- Docker Compose adds healthchecks and `condition: service_healthy`.
- CORS enabled for `http://localhost:3000`.

Status: accepted
