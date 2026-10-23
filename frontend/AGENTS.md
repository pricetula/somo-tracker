# Somotracker Frontend: Agent Rules

Scope: Next.js (App Router) + shadcn/ui (Base UI). Root AGENTS.md covers commit policy and the document-led workflow. Library versions: see package.json.
Rules are enforced by lint, CI and tests where possible. Run `pnpm lint`, `pnpm typecheck` and `pnpm test` before finishing.

## Architecture

- **FE-001** Outside a feature, import only from `features/<name>/index.ts` (components, hooks, types; explicit named exports, no `export *`). Never import a feature's `api/` or internals. Inside a feature use relative imports and never import the feature's own `index.ts`. Features never import other features.
- **FE-002** `app/` route files only render a page component from `src/components/pages/`. Pages compose features via `index.ts` and hold no business logic or data access. Cross-feature data is composed in the page via props; if that isn't possible, raise an Open Question. `components/ui/` and `components/shared/` never import from `features/` or `components/pages/`.
- **FE-003** No back buttons or "Back to" links. Navigation belongs to the app shell. Closing a modal is not back navigation.
- **FE-004** `src/proxy.ts` replaces middleware: default export, matcher required. Session check only, never the sole authorization check.
- **FE-005** Never call `setState` in an effect to mirror props or state. Compute during render or use event handlers. Subscriptions and `useSyncExternalStore` are fine.
- **FE-006** One component per `.tsx` file, named after the file. No second component inside it, even a private one; extract it. Hooks and helpers go in `.ts` files. Column definitions and style maps are not components.

## Data layer

- **FE-007** All HTTP uses native `fetch`, wrapped once in `src/lib/api/client.ts`. Paths start with `/api/`. Never hardcode `/backend`. No axios or other HTTP libraries. Never send a school identifier; the backend resolves it.
- **FE-008** API types and Zod schemas are generated from `openapi/openapi.json` into `src/lib/api/generated/`. Never hand-write or edit them. Generate types and schemas only, never a client or SDK. Inside a feature, only `types.ts` (type re-exports) and `schemas.ts` (Zod, `.extend()` / `.pick()` only) import from `generated/`; everything else imports from `../types` or `../schemas`.
- **FE-009** Components never call API functions, `useQuery`, `useMutation` or `useQueryClient` directly. Each feature has `api/<fn>.ts` (plain async functions calling `client.ts`) and `hooks/use-<hook-name>.ts` (one hook per file, the only place TanStack Query is used). Query keys come from the key factory. Mutations invalidate affected keys on success.
- **FE-010** Errors use `ApiError` with `status/code/message/errors`. One global `MutationCache.onError` shows a `sonner` toast. Success and warning toasts are explicit. Submit buttons are disabled with a spinner while pending.
- **FE-011** The active school is backend state. URLs never contain a school segment (`/students`, never `/[school]/students`). Read it only via `useActiveSchool()`. Query keys are prefixed with its `schoolId` for cache isolation only, and the cache is cleared when it changes.
- **FE-012** Health, vaccination, behavior and fee queries use `staleTime: 0` and `gcTime: 0` and are never persisted. No PII or secrets in logs, URLs, query strings or analytics.

## Layout and routing

- **FE-013** The app shell is shadcn `sidebar-07`. The header contains only `SidebarTrigger`, a vertical `Separator` and a dynamic `Breadcrumb`. Page title and primary actions go in the shared `<PageHeader>` in the content area.
- **FE-014** Add, edit and view use intercepting + parallel routes rendered in `Dialog`/`Sheet`, never inline on list views: `<resource>/@modal/(.)add`, `(.)[id]`, `(.)[id]/edit`. Each also has a real full-page route rendering the same page component, and every `@modal` slot has `default.tsx` returning `null`.

## UI

- **FE-015** No native controls (`select`, `input`, `textarea`, `button`, `dialog`, `table`, `input[type=date]`); use shadcn counterparts. Dates use `Popover` + `Calendar`.
- **FE-016** Semantic Tailwind tokens only (`bg-background`, `text-foreground`, `bg-muted`, `text-destructive`, `border-border`). No hex or palette colors. Status colors (success, warning, attendance, EE/ME/AE/BE) come from `globals.css` tokens through one style map and never rely on color alone. Separate sections with spacing, not cards or borders.
- **FE-017** No wrapper elements without a layout or semantic purpose; use fragments.
- **FE-018** This project uses Base UI: compose with `render={<Component/>}`. Never use `asChild`; convert pasted snippets that do.
- **FE-019** All tables use the shared `<DataTable>` (`@tanstack/react-table` + shadcn `Table`) with server-side pagination, sorting and filtering. No custom div/ul grids. Only exemption: `RollCallList`.
- **FE-020** Forms use `react-hook-form` + `zodResolver` + shadcn `Form` parts (`FormField`, `FormItem`, `FormLabel`, `FormControl`, `FormMessage`). No `useState` form bindings.
- **FE-021** Never hand-edit `src/components/ui/`; add via `pnpm dlx shadcn@latest add`. Shared composites live in `src/components/shared/` and never fetch data.
- **FE-022** Charts use shadcn `ChartContainer`, `ChartTooltip`, `ChartTooltipContent` and `ChartConfig` over Recharts, loaded via `next/dynamic`. No raw SVG/canvas, Chart.js or D3.

## Domain and quality

- **FE-023** Dates use `date-fns` + `@date-fns/tz` with Africa/Nairobi. Date-only fields are `yyyy-MM-dd` strings, never `Date` objects.
- **FE-024** Performance levels come from the generated enum with one label map. Use CBC terms exactly (learning area, strand, sub-strand). Money is integer minor units, formatted only via `formatKes()`.
- **FE-025** Mobile-first; touch targets at least 44px. No new heavy dependencies on attendance or dashboard routes.
- **FE-026** Reference FR/NFR IDs in PRs, commits and tests. Don't build requirements without Given/When/Then AC, or Phase 2/Later features. Unstated behavior becomes an Open Question, not an assumption.
- **FE-027** Every role-gated screen has a test per role. Every modal route has soft-nav and hard-load tests.

## Conventions

- **Feature layout:** `features/<name>/{index.ts, types.ts, schemas.ts, query-keys.ts, api/, hooks/, components/}`.
- **Naming:** files and folders are kebab-case. Components are PascalCase, matching the file (`student-form.tsx` exports `StudentForm`). Hooks are `use-<name>.ts` exporting `use<Name>`. API functions are verb-first (`get-students.ts`, `create-student.ts`).
- **Exports:** named exports only. Default exports only where Next.js requires them (`page.tsx`, `layout.tsx`, `proxy.ts`).
- **Query keys:** `[schoolId, '<feature>', '<scope>', params]`, built only in the feature's `query-keys.ts`.
- **TypeScript:** strict. No `any`; use `unknown` and narrow. Use `type` for aliases and `interface` only when extending. Use `undefined` for absent values; `null` only where the API sends it.
- **Icons:** `lucide-react` only.
- **Dependencies:** add none without approval. Prefer what is already installed.
- **Scope:** change only what the task requires. Don't refactor unrelated code.

## Definition of done

Typecheck, lint and tests pass. New routes and role-gated screens have the tests required by FE-027. Requirement IDs are referenced per FE-026.
