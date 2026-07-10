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
│       ├── hooks/               # Data fetching and local state
│       ├── services/           # API clients, server actions, SDK wrappers
│       ├── types/               # TypeScript interfaces for this feature
│       └── index.ts            # Public API — the only import entry point
│
├── components/                 # Global, generic UI only (e.g. shadcn primitives)
└── lib/                        # Global utilities (e.g. tailwind-merge, auth config)
```

- Each feature is self-contained: logic, UI, and state all live within its folder.
- External code imports a feature only through its `index.ts` — never internal paths.
- Features must not import from each other. Shared logic belongs in `lib/`.
- Route handlers live in `app/api/…/route.ts` — never in `features/`.
- Page files (`page.tsx`) render a single feature container. No logic in page files.
- Do not define multiple components in one `.tsx` file — one component per file.
- Avoid div bloat — no wrapper `<div>` that serves no layout/semantic purpose.

---

## 2. Package Manager

Use **pnpm** exclusively — including for one-off/codemod commands (`pnpm dlx`) and
npm-scripts (`pnpm run <script>`). Never invoke `npm` or `npx` directly.

- `pnpm install` / `pnpm install --frozen-lockfile` (CI/Docker)
- `pnpm add <pkg>` / `pnpm add -D <pkg>` / `pnpm remove <pkg>`
- `pnpm dlx <tool>` for one-off commands (never global installs)
- Use `--ignore-scripts` in CI/Docker unless a postinstall script is explicitly required.

---

## 3. React State-in-Effect Policy

1. Never call `setState` inside `useEffect`. Derive values with `useMemo` or compute
   inline during render.
2. Use event handlers for reactive updates (e.g. auto-filling end time when start time
   changes) — not effects.
3. Prefer `useMemo` over `useEffect + setState` for any value computable from existing
   state or props.

Run `pnpm lint` before pushing — the `react-hooks/set-state-in-effect` rule enforces this.

---

## 4. Documentation & Tooltip Synchronization

All contextual inline UI help must derive from `content/docs/*.mdx` frontmatter via
`<FeatureHelp slug="filename" anchorId="heading-anchor" />`.

- Never hardcode descriptive text inside UI markup or labels.
- Every doc file must declare a `tooltipSummary` string under 160 characters — plain
  text, no Markdown.
- Before completing any task touching routing, settings UI, or backend flag
  configuration, run `pnpm run audit:docs` and fix misalignments before pushing.

---

## 5. Routing — `proxy.ts`

As of Next.js 16, `middleware.ts` is deprecated in favor of `proxy.ts`. For anything not
covered below, check the current [Next.js proxy.js docs](https://nextjs.org/docs/app/api-reference/file-conventions/proxy) rather than relying on memory — behavior here is
still being finalized upstream.

- **Location:** Since this project uses `frontend/src/app/`, proxy must live at
  `frontend/src/proxy.ts` (adjacent to `app/`), not `frontend/proxy.ts`.
- **Export:** **Default export only** (Next.js 16.2 loads via `middlewareModule.default`;
  named exports cause `adapterFn is not a function`). Named export `proxy` is not
  supported in this version. Do NOT use `middleware()`.
- **Signature:** `export default function proxy(request: NextRequest, event?: NextFetchEvent)`
- **Runtime:** Node.js only — the `runtime` config option is not available in proxy
  files and will throw if set.
- **Matcher is required.** Without it, proxy runs on every request including static
  files. Use negative matches to exclude assets:

```ts
export const config = {
    matcher: ["/((?!api|_next/static|_next/image|favicon.ico|sitemap.xml|robots.txt|.*\\.png$).*)"],
};
```

- **State:** Proxy runs separately from render code — no shared modules/globals. Pass
  data via headers, cookies, rewrites, or redirects only.
- **Server Functions are not separate routes** in the execution chain — a matcher that
  excludes a path also skips Server Function calls on that path. Verify auth inside
  each Server Function; don't rely on proxy alone.
- Do not recreate `middleware.ts`.

**Migrating an existing `middleware.ts`:** run the official codemod —
`pnpm dlx @next/codemod@canary middleware-to-proxy .` — it renames the file and export
automatically.

---

## 6. Listing

Use TanStack virtualized lists for any list where the query may return large result
sets.

---

## 7. Visual Guidance — reduce borders and cards

- Avoid excessive borders unless necessary or requested. Separate sections with
  margin/padding (`space-y-*`, `gap-*`, `p-*`) instead.
- Avoid excessive card/`shadow` styling.
- Build tables flat against the background — no cell borders or row outlines. Use
  clean vertical alignment instead.
- Avoid excessive `<Separator />` / `<hr />` dividers; prefer spatial grouping unless
  a divider is explicitly needed.

---

## 8. Shadcn UI Components — never modify, never add by hand

Files under `src/components/ui/` are auto-generated shadcn primitives.

- The only way to add or change one is `pnpm dlx shadcn@latest add <component>`.
- Bugs, type errors, or Tailwind warnings in these files are fixed by re-adding or
  upgrading via the shadcn CLI — never by hand.
- If a shadcn component has a type mismatch with its underlying library (e.g.
  `react-day-picker`), update the library or re-add the component.

---

## 9. Error Handling

### `ApiError` (`src/lib/api/client.ts`)

- Defined only here. Properties: `status: number`, `code: string`, `message: string`,
  `errors?: Record<string, string[]>`.
- Every non-2xx response throws `ApiError`; unparseable bodies throw with fallback
  message "Unexpected error".
- On any 401, the client forces a redirect to `/logout` unless
  `skipGlobal401Handler: true` is set.
- Backend contract reference: `internal/middleware/errors.go`.

### `getErrorMessage` (`src/lib/errors.ts`)

- `getErrorMessage(err: unknown): string` handles `ApiError`, `Error`, string, and
  unknown throws. Never throws, never returns `undefined`.
- All catch blocks use `getErrorMessage(err)` — `(err as Error).message` is forbidden.

### React Query

- `useQuery`: every call site handles `isError` — rendering nothing on error is
  forbidden; at minimum render an `<Alert>`.
- `useMutation`: every data-modifying mutation (create/update/delete/import/upload)
  includes an `onError` that at minimum calls `toast.error(getErrorMessage(err))`.

### Async handlers and hooks

- Every async function not called through React Query needs a `try/catch`.
- Forbidden: `void someAsyncFn()`, `fetch().then(r => r.json())` with no `.catch()`,
  empty catch blocks.
- Background/polling async: retry up to a defined max, then surface a non-intrusive
  status indicator — never silently drop the error.

### Error boundaries

- Every major route/feature is wrapped in `src/components/error-boundary.tsx`.
- `ApiError` (operational) → show `error.message` gracefully. Other errors
  (programming) → report to error tracker, show generic "Something went wrong".
- `src/app/error.tsx` follows the same distinction and reports to the error tracker.

### Form validation (400 responses)

- A mutation receiving `ApiError` with `status === 400` and an `errors` map drives
  field-level errors via `form.setError` — not a generic toast. Canonical pattern:
  `src/features/auth/components/register-form.tsx`.

### Web Workers (`src/workers/`)

- Every worker has `self.onerror` posting a structured error back to the main thread.
- The main thread handler shows a visible error state to the user.

---

## 10. Simplicity & References

- Don't overcomplicate. For standard patterns (virtualized list, form, table), start
  from the library's canonical docs example and adapt — don't reverse-engineer or add
  speculative layers.
- When stuck, fetch the current version's official docs/API reference before
  debugging assumptions about library behavior.

---
