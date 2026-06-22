# Somotracker Frontend — Agent Architecture Contract

Next.js (App Router), TypeScript, Feature-Module architecture.

---

## 1. Core Architecture

`src/app/` is routing only — no custom components or business logic live there.

```
src/
├── app/                        # Next.js routing layer — keep lean
│   ├── layout.tsx
│   ├── page.tsx                # Imports and renders feature containers
│   └── dashboard/
│       └── page.tsx            # Imports <DashboardContainer />
│
├── features/                   # Feature-Module layer — core business logic
│   └── analytics/              # Example feature
│       ├── components/         # Presentational UI (feature-scoped)
│       ├── hooks/              # Data fetching and local state
│       ├── services/           # API clients, server actions, SDK wrappers
│       ├── types/              # TypeScript interfaces for this feature
│       └── index.ts            # Public API — the only import entry point
│
├── components/                 # Global, generic UI only (e.g. shadcn primitives)
└── lib/                        # Global utilities (e.g. tailwind-merge, auth config)
```

- Each feature is **self-contained**: logic, UI, and state all live within its folder.
- External code imports a feature **only** through its `index.ts` — never from internal paths.
- Features must not import from each other. Shared logic belongs in `lib/`.

---

## 2. Package Manager

Use **pnpm** exclusively — never `npm` or `yarn`.

- `pnpm install` — local dev
- `pnpm install --frozen-lockfile` — CI/Docker
- `pnpm add <pkg>` / `pnpm add -D <pkg>` / `pnpm remove <pkg>`
- `pnpm exec` / `pnpm dlx` for one-off commands (never global installs)
- Use `--ignore-scripts` in CI/Docker unless a postinstall script is explicitly required.

---

## 3. React State-in-Effect Policy

`setState` inside `useEffect` causes cascading renders and potential infinite loops.

1. **Never call `setState` inside `useEffect`.** Derive values with `useMemo` or compute inline during render.
2. **Use event handlers for reactive updates** (e.g. auto-filling end time when start time changes) — not effects.
3. **Prefer `useMemo` over `useEffect + setState`** for any value computable from existing state or props.

Run `pnpm lint` before pushing. The `react-hooks/set-state-in-effect` ESLint rule enforces this.

---

## 4. Documentation & Tooltip Synchronization

All contextual inline UI help must derive from `content/docs/*.mdx` frontmatter via `<FeatureHelp slug="filename" anchorId="heading-anchor" />`.

- Never hardcode descriptive text inside UI markup or labels.
- Every doc file must declare a `tooltipSummary` string under 160 characters — plain text, no Markdown.

Before completing any task touching routing, settings UI, or backend flag configuration, run:

```bash
npm run audit:docs
```

Fix misalignments before pushing.

---

## 5. Routing Conventions

- `middleware.ts` → renamed to `proxy.ts`; export is `proxy()` not `middleware()`. Do not recreate `middleware.ts`.
- Route handlers live in `app/api/…/route.ts` — never in `features/`.
- Page files (`page.tsx`) render a single feature container. No logic in page files.

**Changelog:**
| Date | Change |
|------|--------|
| 2026-06-12 | `middleware.ts` renamed to `proxy.ts`; `middleware()` export renamed to `proxy()`. |

---

## 6. Listing

For listing prefer to use tanstack virtualized lists since the query might have large amounts of data

---

## 7. Visual Guidance reducing border lines and cards

- **Excessive use of Bborders are discouraged:** Avoid using alot of borders unless necessary or prompted to add them. Separate sections cleanly using margins and padding (`space-y-*`, `gap-*`, `p-*`).
- Avoid excessive use of card component or elements `shadow` styling.
- Build tables flat against the background container without encapsulating cell borders or surrounding row outlines. Use clean vertical alignment instead.
- Avoid excessive use of horizontal `<Separator />` lines or explicit `<hr />` dividers. Maintain layout groupings purely through spatial rules unless when necessary or prompted to add.

**_ IMPORTANT _**

- **Do not define multiple components in a .tsx file**: Every .tsx file should contain only one react component definition
- **Avoid Div bloat**: do not make useless divs check

---

## 8. Error Handling

### ApiError class (`src/lib/api/client.ts`)

- `ApiError` is defined **only** in `src/lib/api/client.ts`.
- Properties: `status: number`, `code: string`, `message: string`, `errors?: Record<string, string[]>`.
- Every non-2xx response throws `ApiError`. If the body is unparseable, throws with fallback message "Unexpected error".
- **Global 401 eviction:** On any 401, the client forces a redirect to `/logout` (unless `skipGlobal401Handler: true` is set).
- Backend contract reference: `internal/middleware/errors.go`.

### getErrorMessage utility (`src/lib/errors.ts`)

- `getErrorMessage(err: unknown): string` — handles `ApiError`, `Error`, string, and unknown throws.
- Never throws. Never returns `undefined`.
- **All catch blocks must use `getErrorMessage(err)`** — `(err as Error).message` is forbidden.

### React Query rules

- **`useQuery`:** Every call site must handle the `isError` state. Rendering `null` or nothing when `isError` is true is forbidden. At minimum render an `<Alert>` component.
- **`useMutation`:** Every data-modifying mutation must include an `onError` callback. An omitted `onError` on create/update/delete/import/upload is forbidden. The callback must at minimum call `toast.error(getErrorMessage(err))`.

### Async handlers and hooks

- Every async function not called through React Query must have a `try/catch`.
- Forbidden: `void someAsyncFn()`, `fetch().then(r => r.json())` with no `.catch()`, empty catch blocks.
- Background/polling async: retry up to defined max, then surface non-intrusive status indicator. Never silently drop the error.

### Error boundaries

- **Every major route or feature** must be wrapped in a React `ErrorBoundary` (`src/components/error-boundary.tsx`).
- Distinguish:
    - `ApiError` (operational): show `error.message` gracefully.
    - Other errors (programming): report to error tracker, show generic "Something went wrong".
- The global `src/app/error.tsx` follows the same distinction and reports to the error tracker.

### Form validation errors (400 responses)

- When a mutation receives an `ApiError` with `status === 400` and an `errors` map, it must drive field-level errors using `form.setError` — not a generic toast.
- See `src/features/auth/components/register-form.tsx` for the canonical implementation pattern.

### Web Worker errors (`src/workers/`)

- Every worker must have `self.onerror` that posts a structured error message back to the main thread.
- The main thread handler must display a visible error state to the user.

### Forbidden patterns

- `(err as Error).message` — use `getErrorMessage(err)` instead.
- Empty `catch (e) {}` blocks.
- `void someAsyncFn()` (fire-and-forget).
- `fetch(...).then(r => r.json())` with no `.catch()`.
- Omitted `onError` on data-modifying mutations.
- `isError` state ignored in `useQuery`.
- `ApiError` defined anywhere other than `src/lib/api/client.ts`.
