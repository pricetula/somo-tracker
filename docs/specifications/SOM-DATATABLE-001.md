---
id: SOM-DATATABLE-001
title: Virtualized Infinite Data Table Core
status: approved
author: agent
date: 2026-11-13
updated: 2026-11-13
fr_ids: [FE-007, FE-008, FE-009, FE-010, FE-011, FE-015, FE-016, FE-017, FE-018, FE-019, FE-020, FE-021, FE-023, FE-025, FE-026, FE-027]
nf_ids: []
api_contract: no
migrations: no
generated_files: []
---

## 1. Summary

A shared, virtualized infinite data table at `src/components/shared/data-table/`. It combines TanStack Table v9 (manual sorting/filtering/pagination), TanStack Virtual (row-only virtualization over an infinite client cache), and TanStack Query v5 (`useInfiniteQuery` with cursor pagination).

This is the shared `<DataTable>` component mandated by FE-019. All list screens (students, schools, academics, fees, health, behavior, timetable, attendance, enrollments) use it; the only exemption is `RollCallList`.

It is a **shared composite** under `src/components/shared/` and therefore **never fetches data** (FE-021). Data fetching is owned by feature modules via `useInfiniteQuery` + `createListQueryFn`, and data is passed into `<DataTable>` as flattened props.

## 2. Core Architecture

### Virtualization Over Infinite Cache

TanStack Virtual requires a `totalCount` to size the scrollbar. Keyset pagination has no DB total count. The solution:

- All loaded pages are flattened into one array: `pages.flatMap(p => p.items)`
- `useVirtualizer({ count: flatArray.length, ... })` — `totalCount` is rows loaded **so far**, not DB total
- Scroll near bottom → `fetchNextPage` → append new page → virtualizer `totalCount` grows
- User can scroll back up through previously loaded pages (still in React Query cache)

```
[Page 1] [Page 2] [Page 3] → flattened → virtualizer count = 150
                          ↓ scroll near bottom
                          fetchNextPage → [Page 4] appended → count = 200
```

### Cursor Contract

```
GET /api/v1/{resource}?limit=50&sort=name_asc&cursor=&filters=<base64url>
```

| Param | Source | Notes |
|-------|--------|-------|
| `limit` | UI state | Default 50, max 100 |
| `cursor` | `nextCursor` from previous page | Opaque string; null/empty on page 1 |
| `sort` | UI state | Whitelisted name (e.g. `name_asc`) |
| `filters` | Feature state | Base64url-encoded JSON (see §4) |

Cursor payload contains: sort keys, last row values, UUID tie-breaker, and a context hash (`sha256(sortName + "|" + filtersEncoded)`). Context mismatch → backend returns `INVALID_CURSOR` (400) → frontend resets cursor to page 1.

### Encoded Filter Param

Filters are serialized as JSON, Base64url-encoded, sent as a single `filters` query param:

```
filters=eyJmdWxsX25hbWUiOnsib3AiOiJjb250YWlucyIsInYiOiJqb25uIn19
```

Wire format (snake_case field names):
```json
{
  "full_name": {"op": "contains", "v": "john"},
  "gender": {"op": "eq", "v": "male"},
  "date_of_birth": {"op": "between", "v": ["2000-01-01", "2000-12-31"]},
  "country_id": {"op": "is_null", "v": null}
}
```

- Changing filters resets cursor to page 1 (new `filters` value changes cursor context hash)
- Filters are **not** URL-readable (base64 is opaque) — acceptable per product decision
- Round-trip: `parse(serialize(x)) === x` is trivially testable

### Shareability

Shared links strip `cursor` and keep `filters` → recipient gets page 1 with the same filters, scoped to their own tenant (tenant is session-based, never in URL).

---

## 3. Requirements (FR/NFR IDs with rationale)

| FR/NFR ID | Rationale |
|-----------|-----------|
| FE-007 | Native `fetch` via `src/lib/api/client.ts`; base `/api/v1`; never send school ID header; no `[school]` segments |
| FE-008 | Generated OpenAPI types; snake_case wire → camelCase at feature API-function layer |
| FE-009 / FE-019 | `useInfiniteQuery` + `getNextPageParam` reading `nextCursor`; limit default 50 max 100; cursor, sort whitelist, `q`/filters; no page numbers/total counts/OFFSET; query key includes sort/filters; `maxPages` set |
| FE-010 | `ApiError` from `{code,message,errors?,request_id}`; global toast; 404 identical missing/cross-tenant; 204 not parsed; 401 triggers login |
| FE-011 | Active school from `useActiveSchool()`; query keys prefixed with `schoolId` for cache isolation; cache cleared on change |
| FE-015 | No native controls; shadcn `Table`, `Checkbox`, `DropdownMenu`, `AlertDialog` counterparts |
| FE-016 | Semantic Tailwind tokens only; status colors from one style map; never color alone |
| FE-017 | No wrappers without layout/semantic purpose; fragments |
| FE-018 | Base UI `render={<Component/>}`; never `asChild` |
| FE-019 | All tables use shared `<DataTable>` (`@tanstack/react-table` + shadcn `Table`), server-side pagination/sort/filter; exemption `RollCallList` |
| FE-020 | `react-hook-form` + `zodResolver` + shadcn `Form` parts (for inline filter/edit UI) |
| FE-021 | Shared composites never fetch; live in `src/components/shared/` |
| FE-023 | `date-fns` + `@date-fns/tz` Africa/Nairobi; date-only `yyyy-MM-dd` strings |
| FE-025 | Mobile-first; touch targets ≥ 44px |
| FE-026 | Reference FR/NFR IDs; Given/When/Then AC |
| FE-027 | Role-gated screens have per-role tests; modal routes have soft-nav + hard-load tests |

## 4. Domain Model Affected

No database changes. Frontend-only shared component.

### UI & Visual Design

The data table follows a **clean, minimal, modern aesthetic** with uniform rows and pleasing UX. Styling decisions:

| Decision | Specification |
|----------|--------------|
| **Styling tokens** | Semantic Tailwind only (`bg-background`, `text-foreground`, `bg-muted`, `text-muted-foreground`, `border-border`, `text-sm`) per FE-016. No hex colors, no custom CSS. |
| **Row height** | Uniform via `rowHeight` prop (default 48px). Every row identical height; no variably-sized rows. |
| **Borders** | Subtle horizontal dividers only (`border-b border-border`). No vertical borders, no excess cell padding. |
| **Background** | `bg-background` table surface; `bg-muted/50` hovered row state; no zebra striping. |
| **Typography** | `text-sm` for cell content; `text-xs text-muted-foreground` uppercase letter-spaced headers. |
| **Padding** | Uniform `px-4 py-3` on all cells; no content-based padding variance. |
| **Interactive states** | Hover: `bg-muted/50` on rows. Focus: ring-1 ring-ring for checkboxes/links per shadcn. Active selection: `bg-accent/50` with `text-accent-foreground` text. |
| **Empty state** | Centered "No results" message; never blank page (FE-038). |
| **Loading state** | Skeleton rows matching uniform row height; no layout shift (FE-039). |
| **Links** | `text-primary` underline-on-hover via shadcn-styled `<Link>`; never blue by default. |
| **Scrollbar** | Thin, minimal, `bg-muted` thumb on `bg-background` track; hidden on mobile. |
| **Touch targets** | ≥ 44px for checkboxes and action buttons (FE-025). |
| **Cards/boxes** | No card wrappers around the table; sections separated by spacing, not borders (FE-016, FE-017). |

### Files (`src/components/shared/data-table/`)

| File | Type | Responsibility |
|------|------|----------------|
| `data-table.tsx` | Component | `<DataTable>` — TanStack Table v9 + `useVirtualizer` encapsulated inside; ≤150 DOM nodes; row height via `rowHeight` prop (default 48px) |
| `create-list-query-fn.ts` | Helper | `createListQueryFn<TData>(endpoint)` — serializes params into GET; `AbortSignal` passthrough; snake→camel mapping; returns `DataTablePage<TData>` |
| `column-factories.ts` | Helper | `textColumn`, `numberColumn`, `dateColumn`, `badgeColumn`, `selectColumn`, `linkColumn` |
| `selection-store.ts` | Store | `Set<string>` — toggle, selectAllLoaded, selectRange, prune, derived state |
| `data-table-bulk-bar.tsx` | Component | Selected count + `BulkAction` dropdown + `AlertDialog` confirmation |
| `router-adapter.ts` | Helper | `getSearchParams`, `setSearchParams`, `serialize`/`parse` for pagination params (`limit`, `cursor`, `sort`, `filters`) |
| `data-table.test.tsx` | Test | Vitest + RTL tests |

### `DataTablePage<TData>`

```ts
interface DataTablePage<TData> {
  items: TData[];
  nextCursor: string | null;
}
```

### `DataTableProps<TData>`

```ts
interface DataTableProps<TData> {
  columns: ColumnDef<TData>[];
  data: TData[];              // flattened pages
  nextCursor: string | null;
  isLoading: boolean;
  isFetchingNextPage: boolean;
  fetchNextPage: () => void;
  onSortChange: (sort: string) => void;
  onFilterChange: (filters: Record<string, FilterWire>) => void;
  rowHeight?: number;
}
```

### Filter Wire Format

```ts
interface FilterWire {
  op: "eq" | "in" | "between" | "contains" | "is_null";
  v: string | string[] | null;
}
```

### Selection Store

```ts
function createSelectionStore<TData>(): {
  toggle: (id: string) => void;
  selectAllLoaded: (ids: string[]) => void;
  selectRange: (fromIndex: number, toIndex: number, loadedIds: string[]) => void;
  selectNone: () => void;
  prune: (loadedIds: string[]) => { pruned: string[]; remaining: Set<string> };
  getState: () => { selected: Set<string>; count: number };
  subscribe: (listener: () => void) => () => void;
}
```

- Stores `Set<string>` of IDs (not full row objects)
- Survives sort changes and virtual unmounting (IDs retained)
- Pruned on background refetch when IDs disappear from backend data

### Bulk Action

```ts
interface BulkAction<TData> {
  id: string;
  label: string;
  variant?: "default" | "destructive";
  confirm: {
    title: string;
    description: (selectedCount: number) => string;
    confirmLabel: string;
  };
  onSelect: (ids: string[]) => void | Promise<void>;
  maxChunkSize?: number;  // default 100; chunks oversized selections
}
```

## 5. Workflow (Given/When/Then acceptance criteria)

### Virtualization

**Given** 10,000 rows loaded across multiple pages
**When** the virtualizer renders the visible window
**Then** rendered DOM nodes ≤ 150 (visible rows + buffer)

**Given** the user scrolls near the bottom
**When** the last visible row approaches the viewport edge
**Then** `fetchNextPage` is called; new page appended to flat array; virtualizer `totalCount` grows

**Given** the user scrolls back up
**Then** previously loaded rows remain in cache; no refetch; virtualizer shows them instantly

### Cursor & Filters

**Given** filters are applied (`gender=male`, `full_name=contains:john`)
**When** the user navigates to page 2 via `nextCursor`
**Then** backend receives `cursor` + `filters`; cursor context hash matches filter set

**Given** the user changes a filter
**When** the new filter state is submitted
**Then** cursor resets to `null`; backend returns page 1 of the new filter set; `nextCursor` recomputed

**Given** a cursor from `gender=male` is used on a request with `gender=female`
**When** backend decodes the cursor
**Then** context hash mismatch → `INVALID_CURSOR` (400); frontend resets cursor to page 1

**Given** `serialize(filters)` then `parse(result)`
**Then** result deeply equals original (`parse(serialize(x)) === x`)

### Selection

**Given** rows on the current page
**When** the user toggles a row checkbox
**Then** the row ID is added to/removed from the `Set<string>`; `count` derives correctly

**Given** selected IDs
**When** the table sorts or virtualizes rows out of the DOM
**Then** selection persists (IDs retained in store, not tied to DOM nodes)

**Given** selected IDs that are no longer in refetched data
**When** a background refetch completes
**Then** missing IDs are pruned; a toast notification is shown; remaining selection preserved

**Given** a focused row and Shift+Space to a target row
**When** `selectRange` is invoked
**Then** all loaded rows between the two indices (inclusive, sort order) are selected

### Bulk Actions

**Given** 120 selected rows and a bulk delete action (`maxChunkSize: 100`)
**When** the user confirms
**Then** the action is transmitted in 2 chunks (100 + 20); each chunk is `{ ids: [...] }`

**Given** a bulk action where the second chunk fails
**When** partial failure occurs
**Then** only the failed chunk's IDs remain selected; successfully deleted IDs removed from store

## 6. Implementation Plan (files to touch, by application)

### Frontend — `src/components/shared/data-table/`

1. **`column-factories.ts`** — typed factories: `textColumn`, `numberColumn`, `dateColumn`, `badgeColumn`, `selectColumn`, `linkColumn` returning `ColumnDef` with `ColumnMeta`
2. **`selection-store.ts`** — `createSelectionStore<TData>()`: `Set<string>`-based store, toggle, selectAllLoaded, selectRange, prune, derived state, subscribe
3. **`create-list-query-fn.ts`** — `createListQueryFn<TData>(endpoint)`: param conversion (limit, cursor, sort, filters→base64url), `AbortSignal` passthrough, snake→camel mapping, returns `DataTablePage<TData>`
4. **`router-adapter.ts`** — `getSearchParams`, `setSearchParams` (push/replace), `serialize`/`parse` for pagination params (`limit`, `cursor`, `sort`, `filters`)
5. **`data-table.tsx`** — `<DataTable>`: TanStack Table v9 (`manualSorting`, `manualFiltering`, `manualPagination`), row virtualization via `useVirtualizer` (≤150 DOM nodes, `rowHeight` prop default 48px), shadcn `Table`, header/row rendering via column factories
6. **`data-table-bulk-bar.tsx`** — `<BulkBar>`: selected count readout, bulk action `DropdownMenu`, `AlertDialog` confirmation, clear selection
7. **`index.ts`** — named exports only
8. **`data-table.test.tsx`** — Vitest + RTL tests per §7

### Shadcn Dependencies (FE-021: `pnpm dlx shadcn@latest add`, never hand-edit `src/components/ui/`)

- `table` — shadcn Table parts
- `checkbox` — tri-state header + row checkboxes
- `dropdown-menu` — row action menus, bulk action dropdown
- `alert-dialog` — bulk action confirmations

### Backend (referenced, not implemented in this spec)

- One named sqlc query per whitelisted sort, with optional filter params and hardcoded keyset predicate
- Cursor decode + validation (context hash check; `INVALID_CURSOR` → 400)
- `INVALID_CURSOR` error code registered in `apperr`

### Generated Files

None. No OpenAPI change, no swag, no generated types.

## 7. Tests Required

### Vitest Unit Tests (`data-table.test.tsx`)

**Query key determinism:**
- Equivalent states (same sort, filters, limit) produce identical query key arrays
- `cursor` and query function excluded from key equality
- Filter change produces a different key → cursor resets to page 1

**Virtualization bounds (jsdom):**
- Rendered DOM nodes ≤ 150 for large row counts (10k+)
- Container scroll near bottom triggers `fetchNextPage`
- Virtualizer `totalCount` grows as pages append
- Scroll back up shows previously loaded rows without refetch

**Selection store:**
- `toggle` adds/removes ID from `Set<string>`
- `selectAllLoaded` selects all IDs on current page
- `selectRange` selects range between two indices (inclusive)
- `prune` removes IDs not in `loadedIds`; returns `pruned` and `remaining`
- Selection persists across sort changes (IDs retained)
- Derived `count` is correct

**Router adapter:**
- `parse(serialize(x)) === x` for pagination params (limit, cursor, sort, filters)
- `filters` base64url round-trips with special characters (dots, commas, unicode, empty values)
- Malformed `filters` returns safe default, does not throw

### React Testing Library Integration Tests

- Initial mount fires exactly one query after hydration
- `AbortSignal` cancels in-flight `fetch` on rapid unmount
- `nextCursor: null` disables "load more"
- Filter change → exactly one query with cursor reset; filters preserved
- Bulk action chunking: 120 selected → 2 sequential `{ ids: [...] }` calls (100 + 20)
- Bulk action partial failure: chunk 2 fails → only chunk 2's IDs remain selected

### Per-role / modal tests

Covered by domain feature specs (students, schools, etc.) per FE-027 — this spec provides the shared `<DataTable>` only.

## 8. Resolved Decisions

| Question | Decision |
|----------|----------|
| Backend filter param format | Single `filters=<base64url>` param. Backend parser matches `createListQueryFn` serialization exactly (snake_case field names, `op`/`v` structure). |
| `maxPages` default | `maxPages: 10` (caches last 10 pages). |
| Row height source | Set on the component via `rowHeight` prop (default 48px). No CSS variable. |
| `linkColumn` navigation | Always uses TanStack Router `<Link>` (renders `<a>` under the hood). Middle-click, Ctrl/Cmd-click, and screen-reader semantics work natively. No native `<a href>`. |
| Pruning notification | Toast via sonner (already global per FE-010 `MutationCache.onError`). |

## 9. Links

- [FE-019](../requirements/frontend.md) — shared `<DataTable>` requirement
- [FE-009](../requirements/frontend.md) — `useInfiniteQuery` + cursor pagination, query keys include sort/filters
- [FE-011](../requirements/frontend.md) — active school from `useActiveSchool()`, `schoolId` query-key prefix
- [FE-021](../requirements/frontend.md) — shared composites never fetch
- [FE-025](../requirements/frontend.md) — mobile-first, touch targets ≥ 44px
- [FE-026](../requirements/frontend.md) — FR/NFR IDs referenced; Given/When/Then AC
- [FE-027](../requirements/frontend.md) — per-role tests, modal soft-nav + hard-load tests
- [FE-040](../requirements/frontend.md) — search debounce 300ms (handled at feature level)
- [FE-041](../requirements/frontend.md) — active filters as removable chips (handled at feature level)
- [frontend/package.json](../../../frontend/package.json) — `@tanstack/react-table` v9, `@tanstack/react-virtual` v3, `@tanstack/react-query` v5 already installed
- [src/components/shared/](../../../frontend/src/components/shared/) — existing shared composites (app-sidebar, sidebar-provider)
