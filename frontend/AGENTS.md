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
