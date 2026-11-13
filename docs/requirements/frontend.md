# Frontend requirements (FE-001…FE-027, FE-AUTH-01…FE-AUTH-03)

Source: `frontend/AGENTS.md`. One line per rule; full rule text lives in the source file.

## Architecture

| ID | Summary |
|----|---------|
| FE-001 | Outside a feature import only from `features/<name>/index.ts`; explicit named exports, no `export *`; features never import other features |
| FE-002 | `routes/` export TanStack Router `Route` with `loader`/`component`; pages compose features via `index.ts`, no business logic or data access |
| FE-003 | No back buttons or "Back to" links; navigation belongs to app shell |
| FE-004 | `entry.server.tsx`/`routes/$server.ts` replace Next.js middleware: session check only, never sole authorization |
| FE-005 | No `setState` in effect to mirror props; compute during render or in handlers |
| FE-006 | One component per `.tsx` file, named after the file; hooks/helpers in `.ts` |

## Data layer

| ID | Summary |
|----|---------|
| FE-007 | Native `fetch` via `src/lib/api/client.ts`; base `/api/v1`; never hardcode `/backend`; no axios; never send school ID / `X-Active-School`; no `[school]` segments |
| FE-008 | API types + Zod schemas generated from `docs/api/openapi.yaml` into `src/features/auth/api/generated/types.ts`; generated types only; snake_case wire types; map snake→camel at API-function layer |
| FE-009 / FE-019 | `useInfiniteQuery` + `getNextPageParam` reading `nextCursor`; limit default 50 max 100; cursor, sort whitelist, `q`/filters; no page numbers/total counts/OFFSET |
| FE-010 | `ApiError` from status + body `{code,message,errors?,request_id}`; global `MutationCache.onError` sonner toast; code→message map; `errors[].field` dotted path to `setError`; 404 identical for missing/cross-tenant; 204 not parsed; 401 triggers login |
| FE-011 | Active school is backend state; `useActiveSchool()`; query keys prefixed with `schoolId` for cache isolation only; cache cleared on change |
| FE-012 | Sensitive lists `staleTime: 0`, `gcTime: 0`, never persisted; no PII/secrets in logs/URLs/query strings/analytics |

## Layout and routing

| ID | Summary |
|----|---------|
| FE-013 | Shell shadcn `sidebar-07`; header only `SidebarTrigger` + `Separator` + dynamic `Breadcrumb`; `<PageHeader>` for title/actions |
| FE-014 | Add/edit/view via intercepting + parallel routes in `Dialog`/`Sheet`; `@modal/(.)add`, `(.)[id]`, `(.)[id]/edit`; full-page route renders same component; `@modal` default returns `null` |
| FE-028 | Create routes use `xxx/add` via intercepting + parallel routes; `Dialog`/`Sheet` on client navigation, full-page route on hard load; `@modal` default returns `null` |
| FE-029 | View routes use `xxx/id`; `Sheet` side panel on client redirect; full-page route renders same page on hard load |
| FE-030 | Update resource per-section on the detail page; no dedicated edit routes |

## UI

| ID | Summary |
|----|---------|
| FE-015 | No native controls; shadcn counterparts; dates via `Popover`+`Calendar` |
| FE-016 | Semantic Tailwind tokens only; status colors from `globals.css` via one style map; never color alone |
| FE-017 | No wrappers without layout/semantic purpose; fragments |
| FE-018 | Base UI: `render={<Component/>}`; never `asChild` |
| FE-019 | Tables use shared `<DataTable>` (react-table + shadcn Table), server-side pagination/sort/filter; exemption `RollCallList` |
| FE-020 | Forms: `react-hook-form` + `zodResolver` + shadcn `Form` parts; no `useState` bindings |
| FE-021 | Never hand-edit `src/components/ui/`; `pnpm dlx shadcn@latest add`; shared composites in `src/components/shared/`, never fetch |
| FE-022 | Charts: shadcn Chart + Recharts via `React.lazy`/`Suspense`; no raw SVG/canvas, Chart.js, D3 |
| FE-031 | Create forms use `react-hook-form` + `zodResolver` + shadcn `Form` parts; uniform styling across resources |
| FE-032 | Form buttons show loading/error states via shadcn components; uniform across all forms |
| FE-033 | Listing tables use shared `<DataTable>` with TanStack Infinite Query + `@tanstack/react-virtual` (full spec pending) |
| FE-034 | Delete confirmation via `Dialog`/`AlertDialog` with accept/cancel; cannot dismiss by clicking overlay |

## Domain and quality

| ID | Summary |
|----|---------|
| FE-023 | `date-fns` + `@date-fns/tz` Africa/Nairobi; date-only `yyyy-MM-dd` strings |
| FE-024 | Performance levels from generated enum + label map; CBC terms exactly (learning area, strand, sub-strand); money minor units via `formatKes()` |
| FE-025 | Mobile-first; touch targets ≥44px; no new heavy deps on attendance/dashboard routes |
| FE-026 | Reference FR/NFR IDs in PRs/commits/tests; Given/When/Then AC; unstated behavior = Open Question |
| FE-027 | Every role-gated screen has a test per role; every modal route has soft-nav and hard-load tests |

## Auth guard and server-only imports

| ID | Summary |
|----|---------|
| FE-AUTH-01 | `beforeLoad` redirect via `redirect()` from `@tanstack/react-router`, never native `Response` |
| FE-AUTH-02 | Client code never imports `@tanstack/react-start/server` or `*.server.*`; use middleware or `createServerOnlyFn()` |
| FE-AUTH-03 | `QueryClientProvider` wraps `<Outlet />` in `routes/__root.tsx` |

## Naming and TypeScript conventions

| ID | Summary |
|----|---------|
| — | kebab-case files/folders; PascalCase components matching file; `use-<name>.ts` hooks exporting `use<Name>`; verb-first API functions |
| — | Named exports only; defaults only for framework entry points |
| — | Query keys `[schoolId,'<feature>','<scope>',params]` built only in feature `query-keys.ts` |
| — | Strict TS; no `any`; `type` aliases / `interface` only when extending; `undefined` absent, `null` only where API sends it |
| — | Icons: `lucide-react` only; no new dependencies without approval |

## Snake_case vs camelCase boundary

Generated wire types snake_case; internal code camelCase; map at `src/features/<name>/api/` adapter layer; never hand-edit generated types; never invent manual snake_case in components.

## API contract changes (pending)

| Endpoint | Method | Body | Replaces |
|----------|--------|------|----------|
| `DELETE /api/v1/{resource}` | `DELETE` | `{ "ids": ["uuid1", "uuid2", ...] }` | `DELETE /api/v1/{resource}/{id}` |

Backend owns `docs/api/openapi.yaml`. After any contract change: backend regenerates `backend/docs/swagger.*` via `make generate-swag`, frontend regenerates types via `pnpm generate:api`, then verify no drift with `make generate-check`.
