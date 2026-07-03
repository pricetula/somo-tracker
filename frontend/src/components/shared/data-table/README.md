# Generic Data Table (TanStack Query + TanStack Virtual)

A drop-in table component that works with **any** of your generated
`list*` API functions and their matching types — no per-resource table
code needed.

## How it works

- **TanStack Query v5** (`useInfiniteQuery`) fetches pages of rows and
  keeps them cached by `[...queryKey, params, pageSize]`.
- **TanStack Virtual v3** (`useVirtualizer`) renders only the rows in the
  viewport, so tables with thousands of rows stay smooth.
- Scrolling near the bottom of the loaded rows triggers `fetchNextPage()`
  automatically — this is virtualized _infinite scroll_, not classic
  page-number pagination. It's the standard pairing for these two
  libraries and is what makes "large datasets" performant here.
- `onSearch` is a thin, opinionated hook: if you pass it, a debounced
  (350ms) search input appears above the table and calls `onSearch(term)`.
  What you _do_ with that term is up to you — typically you store it in
  state and fold it back into the `params` object you pass to `DataTable`
  (see `example/classes-table.tsx`).

## Install

```bash
npm install @tanstack/react-query@latest @tanstack/react-virtual@latest
```

You'll also need a `QueryClientProvider` mounted once near the root of
your app (e.g. in `app/providers.tsx`) if you don't already have one:

```tsx
"use client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useState } from "react";

export function Providers({ children }: { children: React.ReactNode }) {
    const [client] = useState(() => new QueryClient());
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}
```

## Files

```
components/data-table/
  data-table.tsx             the <DataTable /> component
  use-infinite-list-query.ts TanStack Query wrapper (generic over any list* fn)
  use-debounced-value.ts     debounce helper for the search input
  types.ts                   NormalizedListResult, DataTableColumn, ListApiFn
  index.ts                   barrel export
example/
  classes-table.tsx          usage against listClasses from your snippet
  students-table.tsx         usage against listStudents (class_id/gender filters)
```

Copy the `components/data-table/` folder into your project as-is (e.g.
`src/components/data-table/`). It only imports from `@tanstack/react-query`
and `@tanstack/react-virtual`, so it doesn't care where you put it.

## Usage

```tsx
import { DataTable } from "@/components/data-table";
import { listClasses, type Class } from "@/lib/api/classes";

<DataTable
    queryKey={["classes"]}
    queryFn={listClasses}
    params={{ academic_year_id, academic_term_id }}
    getRowId={(row: Class) => row.id}
    columns={[
        { id: "name", header: "Class", cell: (row: Class) => row.name },
        { id: "grade", header: "Grade", cell: (row: Class) => row.grade_level },
    ]}
    onSearch={(term) => setSearch(term)}
/>;
```

### Response shape

`DataTable` expects every list endpoint to return:

```ts
{ items: TItem[]; total?: number; page?: number; limit?: number; hasMore?: boolean }
```

which matches your standardized Go response shape:

```go
type ListStudentsResponse struct {
    Items []Student `json:"items"`
    Total int       `json:"total"`
    Page  int       `json:"page"`
    Limit int       `json:"limit"`
}
```

As long as every `list*` handler returns `items`/`total`/`page`/`limit`,
you never need to pass a `normalize` prop — `DataTable` reads the result
as-is. This is the recommended default; keep new list endpoints
consistent with it and adding a new resource to the table is genuinely
zero-config (just `queryFn`, `params`, `columns`, `getRowId`).

**Escape hatch:** if you ever integrate a one-off endpoint that can't
follow this convention (a third-party API, a legacy handler you don't
want to touch, etc.), pass `normalize` for just that one call instead of
changing `DataTable`:

```tsx
import { normalizeListResponse } from "@/components/data-table";

// Go handler that still returns `{ students: [...], total, page, limit }`
<DataTable
  queryFn={listStudentsLegacy}
  normalize={normalizeListResponse("students")}
  ...
/>

// Or a completely different shape:
<DataTable
  queryFn={listFromThirdParty}
  normalize={(result: ThirdPartyResult) => ({
    items: result.data.rows,
    total: result.meta.count,
  })}
  ...
/>
```

### Search and other filters

`DataTable` doesn't own filter state beyond the debounced search string —
it just calls `onSearch`. The idiomatic pattern (shown in
`example/classes-table.tsx`) is:

```tsx
const [search, setSearch] = useState("");

<DataTable
  params={{ academic_year_id, academic_term_id, search: search || undefined }}
  onSearch={setSearch}
  ...
/>
```

Changing `params` (including `search`) automatically resets and refetches
the query, since `params` is part of the TanStack Query key. Your Go
`ListFilter` already supports `Search`, `ClassID`, `Gender`, etc. via
`c.Query(...)` — any of those just become fields on `params`, exactly
like `search` above.

### Column definition

```ts
interface DataTableColumn<TItem> {
    id: string;
    header: ReactNode;
    cell: (row: TItem, index: number) => ReactNode;
    width?: string; // any CSS grid track: "120px", "1fr", "minmax(120px,1fr)"
    align?: "left" | "center" | "right";
}
```

Cells can render anything — badges, buttons, formatted dates — since
`cell` just returns `ReactNode`.

### Tuning

| Prop        | Default | Notes                                         |
| ----------- | ------- | --------------------------------------------- |
| `pageSize`  | 50      | Rows fetched per page                         |
| `rowHeight` | 44      | Estimated row height (px) for the virtualizer |
| `height`    | 600     | Height (px) of the scrollable viewport        |

## Styling

The markup uses plain Tailwind utility classes (grid-based rows rather
than a `<table>`, which is required for absolute-positioned
virtualization). Swap the classNames for your design system / shadcn
tokens as needed — the structure won't change.

## Version notes (as of this writing)

- TanStack Query v5 removed `keepPreviousData` as a boolean option in
  favor of `placeholderData: keepPreviousData` (the function) — already
  used internally here.
- TanStack Virtual v3 is the current major version; `useVirtualizer` API
  used here (`getVirtualItems`, `getTotalSize`, `measureElement`) is
  stable for both fixed and dynamic row heights.
- Install with `@latest` rather than pinning versions in this README,
  since both libraries ship frequently — check `npm info @tanstack/react-query version`
  if you want to confirm what you actually installed.
