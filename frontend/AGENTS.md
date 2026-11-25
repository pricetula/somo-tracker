# Frontend Agent Rules

This file takes precedence over the root `AGENTS.md` for anything under `frontend/`. The root file still governs git conventions (conventional commits, pre-push compliance, no secrets in git).

Rules marked **MUST / NEVER** are hard rules. Everything else is a strong default: deviate only with a stated reason.

## Stack

- Next.js 16+ (App Router), React 19, TypeScript (strict), Tailwind CSS, shadcn/ui (Base UI variant).
- TanStack Query for all client-side server state.
- react-hook-form + zod for forms. lucide-react for icons.
- BFF pattern: route handlers under `src/app/api/` proxy to the Go backend. The Go backend sets session cookies on responses forwarded through Next.js. Auth guarding is already implemented; do not modify or re-implement it.
- Package manager: pnpm. This is a monorepo, and all commands run from `frontend/`.

## Commands

```
cd frontend
pnpm dev
pnpm build
pnpm typecheck
pnpm lint
pnpm test                 # all tests
pnpm test <path>          # single file
```

**Definition of done:** `pnpm typecheck`, `pnpm lint`, and `pnpm test` all pass before a task is declared complete.

## Directory Map

```
frontend/
├── AGENTS.md
├── components.json
└── src/
    ├── app/                          # ROUTING ONLY (see "App Directory Rules")
    │   ├── layout.tsx
    │   ├── error.tsx
    │   ├── not-found.tsx
    │   ├── (public)/                 # login, onboarding, etc.
    │   │   └── <route>/page.tsx
    │   ├── (app)/                    # authenticated shell
    │   │   ├── layout.tsx            # renders {children} and {overlay}
    │   │   ├── @overlay/             # single slot for all dialogs and side panels
    │   │   │   ├── default.tsx       # return null
    │   │   │   ├── error.tsx
    │   │   │   ├── [...catchAll]/page.tsx   # return null (closes overlay on soft nav)
    │   │   │   └── (.)<module>/
    │   │   │       ├── add/page.tsx
    │   │   │       └── [id]/page.tsx
    │   │   └── <module>/
    │   │       ├── page.tsx          # list
    │   │       ├── loading.tsx
    │   │       ├── error.tsx
    │   │       ├── add/page.tsx      # full page (hard refresh)
    │   │       └── [id]/page.tsx     # full page (hard refresh)
    │   └── api/
    │       └── <module>/route.ts     # thin BFF proxy handlers
    │
    ├── features/
    │   └── <module>/
    │       ├── api/
    │       │   ├── <module>.api.ts   # plain async fetchers, no React
    │       │   └── <module>.keys.ts  # query key factory
    │       ├── hooks/
    │       │   ├── use-<thing>.ts    # useQuery wrappers
    │       │   └── use-<action>.ts   # useMutation wrappers (own invalidation)
    │       ├── components/
    │       │   ├── <module>-add-view.tsx
    │       │   ├── <module>-detail-view.tsx
    │       │   └── ...               # one component per file
    │       ├── schemas/
    │       │   └── <thing>.schema.ts # zod; infer types with z.infer
    │       ├── types/
    │       │   └── index.ts
    │       └── index.ts              # the module's public surface
    │
    ├── components/
    │   ├── ui/                       # shadcn CLI only
    │   ├── overlay/                  # route-dialog, route-sheet, route-overlay-context
    │   ├── layout/                   # app shell, sidebar, page-shell, page-header
    │   └── providers/                # QueryClientProvider, theme, toaster
    ├── hooks/                        # generic hooks (use-debounce, use-media-query)
    ├── lib/
    │   ├── api-client.ts             # shared fetch wrapper (errors, JSON, credentials)
    │   ├── query-client.ts           # QueryClient factory and defaults
    │   ├── utils.ts                  # cn()
    │   └── server/                   # server-only: BFF proxy helper, cookie forwarding
    ├── types/                        # truly global types (ApiError, Paginated<T>)
    └── config/                       # nav items, constants
```

### Naming

- Directories and files are kebab-case. The directory is `features/` (plural).
- Role suffixes: `.api.ts`, `.keys.ts`, `.schema.ts`. Hooks are `use-*.ts`.
- Tests are colocated as `*.test.ts(x)` next to the file under test.

### Dependency direction

- `app/` may import from `features/`, `components/`, `hooks/`, `lib/`, `types/`, `config/`.
- `features/` may import from `components/`, `hooks/`, `lib/`, `types/`, `config/`. NEVER from `app/`.
- A feature MUST NOT import another feature's internals. Cross-feature use goes through the other feature's `index.ts` only.
- `components/`, `hooks/`, and `lib/` NEVER import from `features/`.
- Files in `lib/server/` start with `import "server-only"`.

### Module hygiene

- Export from `index.ts` only what other code needs. Everything else is private.
- Do not create empty folders. Add `schemas/` or `types/` only when the module needs them.
- Promote code to `components/` or `hooks/` when a second feature needs it, not before.
- When a module's `components/` exceeds about 10 files, group by sub-domain (`components/invoice-table/`), not by kind.

## App Directory Rules (routing only)

- `src/app/` MAY contain only: `page.tsx`, `layout.tsx`, `loading.tsx`, `error.tsx`, `not-found.tsx`, `default.tsx`, `route.ts`.
- NEVER put components, hooks, `_components` folders, providers, or utilities in `src/app/`.
- Page files contain no logic, only composition. Aim for under 15 lines.
- Server components by default. `"use client"` goes in feature components, as low in the tree as possible, never in page files.
- Next 16: `params` and `searchParams` are Promises. Always `await` them.
- Every route segment has its own `error.tsx`, and major routes also get `loading.tsx`.
- Route handlers in `app/api/` stay thin: validate, forward through the `lib/server` proxy helper, return. No business logic.
- Enforce this with a CI check that fails if `src/app/` contains a file outside the allowlist.

## Intercepted Routes (add dialogs and detail panels)

Pattern: `/<module>/add` opens a dialog, `/<module>/[id]` opens a right-side panel. On a hard refresh or direct visit, both render as full pages.

**Core idea:** the full-page route and the intercepted overlay route render the same `*-view` component. Only the wrapper differs.

### Required route files per resource

| Purpose            | Path                                           |
| ------------------ | ---------------------------------------------- |
| Add, full page     | `app/(app)/<module>/add/page.tsx`              |
| Add, dialog        | `app/(app)/@overlay/(.)<module>/add/page.tsx`  |
| Detail, full page  | `app/(app)/<module>/[id]/page.tsx`             |
| Detail, side panel | `app/(app)/@overlay/(.)<module>/[id]/page.tsx` |

Plus two views in `features/<module>/components/`: `<module>-add-view.tsx` and `<module>-detail-view.tsx`.

### Rules

- ONE `@overlay` slot serves all modules. Do not create `@modal` or `@panel` slots.
- The `@overlay` slot MUST keep `default.tsx` (returns `null`), `[...catchAll]/page.tsx` (returns `null`, so the overlay closes when navigating elsewhere), and `error.tsx`.
- The path under `@overlay` mirrors the real URL with `(.)` on the first segment. Slots and route groups are not URL segments.
- Use `RouteDialog` for create/edit forms and `RouteSheet` for detail views. NEVER build ad-hoc `Dialog` or `Sheet` usage in route files.
- Views MUST finish through `useRouteExit(fallbackHref)`. It closes the overlay if inside one, otherwise navigates to `fallbackHref`. NEVER call `router.back()` or `router.push()` directly inside a view.
- Views know nothing about whether they render in a dialog, a sheet, or a page.
- Navigate to overlay routes with `<Link>`. A reload or a new-tab open shows the full page, which is intended.

### Examples

```tsx
// app/(app)/@overlay/(.)invoices/add/page.tsx
export default function Page() {
  return (
    <RouteDialog title="Add invoice">
      <InvoiceAddView />
    </RouteDialog>
  );
}

// app/(app)/invoices/add/page.tsx
export default function Page() {
  return (
    <PageShell title="Add invoice">
      <InvoiceAddView />
    </PageShell>
  );
}

// app/(app)/@overlay/(.)invoices/[id]/page.tsx
export default async function Page({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  return (
    <RouteSheet title="Invoice">
      <InvoiceDetailView id={id} />
    </RouteSheet>
  );
}
```

```tsx
// components/overlay/route-overlay-context.tsx
"use client";
// Overlay: close() animates out, then router.back() runs from onOpenChangeComplete.
// useRouteExit(fallback): overlay ? overlay.close() : router.push(fallback)
```

`RouteDialog` and `RouteSheet` use `onOpenChangeComplete` so the exit animation finishes before history pops. If the generated shadcn `dialog.tsx` or `sheet.tsx` does not pass this prop through, do NOT edit them. Re-add via the shadcn CLI or move the handling elsewhere.

### Testing

Each resource has one Playwright test: click through from the list and expect the overlay, then reload and expect the full page. This behavior regresses silently.

## Data Fetching (TanStack Query)

- Components NEVER call `fetch` or `api/` functions directly. They go through hooks in `hooks/`.
- Fetchers in `api/` are plain async functions using `lib/api-client.ts`, typed from `types/`, and contain no React.
- Each module has one key factory in `api/<module>.keys.ts` (e.g. `invoiceKeys.list(filters)`, `invoiceKeys.detail(id)`). NEVER write inline key arrays.
- Mutation hooks own their invalidation via the key factory.
- API mutations pass IDs in the request body, not only in URL params.
- The `QueryClient` comes from `lib/query-client.ts`. Do not construct ad-hoc clients.
- Prefetch on the server only when it removes a visible loading state, and hydrate via `HydrationBoundary`.

## Components

- One component per file. Flat layout tree, no div soup.
- Do not wrap elements or shadcn primitives in a `div` unless it introduces an active Grid/Flex container or a concrete layout padding step. Prefer React Fragments for sibling returns.
- Use shadcn compound components (`DialogHeader`, `DialogContent`, `FormField`, etc.) instead of custom wrapper nodes. Names are PascalCase concatenated, not dot notation.
- Spacing comes from `space-y-*` and `gap-*`. `p-*` is allowed on the page container only.

```tsx
// Bad
<div><div><Label>Name</Label></div><div><Input /></div></div>

// Good
<div className="space-y-2">
  <Label>Name</Label>
  <Input />
</div>
```

## shadcn/ui

- Primitives live in `src/components/ui/` and are added ONLY via `pnpm dlx shadcn@latest add <component>`.
- NEVER hand-edit files in `src/components/ui/`. Fix bugs, type errors, and warnings by re-adding or upgrading via the CLI. If a component mismatches its underlying library (e.g. `react-day-picker`), update the library or re-add the component.
- This project uses the **Base UI** variant. Use `render={<Component />}` for element composition. **NEVER use `asChild`.** Lint blocks it.

### shadcn-only components

Every UI element is a shadcn component, for consistency. If the primitive you need is not installed, add it via the CLI. NEVER hand-roll a substitute, and NEVER reach for another UI library.

| Need                    | Use                                                 | NEVER use                                             |
| ----------------------- | --------------------------------------------------- | ----------------------------------------------------- |
| Choosing from options   | `Combobox`                                          | native `<select>`, shadcn `Select`, custom dropdowns  |
| Picking a date          | `Calendar` inside a `Popover` (date picker pattern) | `<input type="date">`, text inputs for dates          |
| Single-line text        | `Input`                                             | native `<input>`                                      |
| Multi-line text         | `Textarea`                                          | native `<textarea>`                                   |
| Actions                 | `Button`                                            | native `<button>`, styled `<a>` or `<div>` as buttons |
| Checkbox, radio, switch | `Checkbox`, `RadioGroup`, `Switch`                  | native inputs                                         |
| Labels and field wiring | `Label`, `Form`/`Field` primitives                  | bare `<label>`                                        |

- Single-select and multi-select both use `Combobox`. Wire it to react-hook-form through a `FormField`/`Controller`.
- Date ranges use `Calendar` in range mode. Dates are stored as ISO strings or `Date` objects in the form and validated in the zod schema.
- Buttons that navigate use `<Button render={<Link href="..." />}>`, never `asChild`.
- Wrap a repeated combination (e.g. a combobox bound to a form field and an API-backed option list) in a reusable component in `components/` once a second feature needs it. Do not rebuild it per feature.

## Styling

- NEVER use hardcoded hex/RGB values or arbitrary Tailwind color classes (`bg-[#fff]`, `bg-slate-100`, `text-blue-600`).
- Use semantic tokens only:
  - Backgrounds: `bg-background`, `bg-muted`, `bg-muted/30`
  - Text: `text-foreground`, `text-muted-foreground`
  - Accents: `bg-primary`, `text-primary-foreground`
  - Status: `text-destructive`, `text-emerald-600` (success)
- <!-- TODO: define a `--success` (and `--warning`) token and replace `text-emerald-600` with `text-success`, so the rule has no exceptions. -->
- No custom Tailwind colors in `tailwind` config or CSS beyond the semantic variables.

### Layout style

- No cards or borders to separate page sections: do not use `border`, `bg-card`, or `shadow` on layout you write. (Untouched shadcn primitives such as `Input` and `Select` keep their own internal borders.)
- Separate sections with whitespace (`space-y-6`, `gap-4`) and typographic hierarchy (`font-medium text-muted-foreground`).
- Stack sections vertically, each with an `<h2>` or `<h3>`. Do not use `<Tabs>` to split page content. Sub-navigation belongs in the sidebar or in feature-level route pages.
- Whitespace-only designs lose focus and contrast cues easily: keep visible focus rings, label every form field, and manage focus in dialogs and sheets.
- No "dumb back buttons" (no ArrowLeft icons in pages or features). Navigation goes through the sidebar and breadcrumbs. Cancel and close actions are labelled buttons that call `useRouteExit`.

## Forms

- Follow the onboarding-form pattern: react-hook-form + zod. <!-- TODO: add the path to the reference file, e.g. src/features/onboarding/components/onboarding-form.tsx -->
- Schemas live in `features/<module>/schemas/`. Derive form types with `z.infer`. Do not duplicate them in `types/`.
- Disable submit while the mutation is pending. Surface field errors inline and request errors as a user-facing message.

## Error Handling

- Every async handler has try/catch. Error states render user-facing messages, never raw errors.
- Error boundaries: every route segment has `error.tsx`. A failed overlay must not take down the underlying page.

## Turnstile

- Keep the token in state. The submit button is disabled until a token exists.
- Clear the token on widget expiry or error. Reset the widget and clear the token after every submit attempt.
- Send the token to the backend as `turnstile_token` alongside `email`.
- A 403 or 502 from verification shows a specific message. <!-- TODO: list the exact copy for each. -->

## Required Configuration

Defined in `.env.example`. Only list variables with no default.

| Variable              | Description                   |
| --------------------- | ----------------------------- |
| `API_URL`             | Backend API URL               |
| `APP_URL`             | Public frontend URL           |
| `SESSION_COOKIE_NAME` | Must match backend            |
| `APP_ENV`             | `development` or `production` |

## Enforcement (tooling over prose)

Anything checkable should be a lint or CI rule. Aim for these, and treat a failure as a task failure:

- ESLint `no-restricted-syntax`: `asChild`.
- ESLint/Tailwind rule: arbitrary color values and non-semantic palette classes.
- `no-restricted-imports`: icon libraries other than `lucide-react`.
- ESLint `no-restricted-syntax`: native `<input>`, `<textarea>`, `<select>`, and `<button>` JSX outside `src/components/ui/`, and any `type="date"`.
- `eslint-plugin-boundaries` (or import zones): the dependency direction rules above.
- CI check: `src/app/` contains only allowlisted filenames.
- CI check: `src/components/ui/` matches CLI output (no hand edits).

## Adding a Resource: Checklist

1. `features/<module>/` with `api/`, `hooks/`, `components/`, `schemas/`, `types/`, `index.ts`.
2. BFF handler: `app/api/<module>/route.ts`.
3. Key factory, fetchers, query and mutation hooks.
4. `<module>-add-view.tsx` and `<module>-detail-view.tsx`, using `useRouteExit`.
5. Four route files (add and detail, full-page and `@overlay`) plus list `page.tsx`, `loading.tsx`, `error.tsx`.
6. Playwright test for overlay and hard-refresh behavior.
7. `pnpm typecheck && pnpm lint && pnpm test`.
