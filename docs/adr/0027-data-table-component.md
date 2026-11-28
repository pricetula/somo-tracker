# ADR-0027: Reusable Server-Driven DataTable

**Status:** Accepted  
**Date:** 2026-11-26  
**Scope:** Frontend shared infrastructure and backend API contract

## 1. Context

We need a reusable, generic data table component for the application, capable of handling large datasets across different resources such as students, teachers, classes, fees, payments and other entities.

The implementation must avoid duplicating fetching, pagination, selection, sorting, filtering, virtualization and deletion logic across resource-specific pages.

The component must integrate:

- Next.js
- TanStack Query
- TanStack Table
- TanStack Virtual
- shadcn/ui
- TypeScript

The backend is independently owned and can be designed to support the frontend's requirements.

The objective is to build a shared component that owns common table infrastructure while allowing individual resource pages to define their columns, filters and resource-specific actions.

## 2. Decision

We will implement a generic, server-driven DataTable component.

The shared component will own the common data lifecycle and table interactions. Resource-specific configuration will supply data definitions and business-specific operations.

### 2.1 Technology responsibilities

| Technology       | Responsibility                                                       |
| ---------------- | -------------------------------------------------------------------- |
| TanStack Query   | Data fetching, caching, infinite queries and mutations               |
| TanStack Table   | Column definitions, sorting, filtering state and row selection       |
| TanStack Virtual | Efficient rendering of large loaded datasets                         |
| shadcn/ui        | Table controls, dialogs, dropdowns, toolbars and other UI primitives |
| Next.js          | Application routing and client/server component boundaries           |

The DataTable itself will be a client-side component.

### 2.2 Pagination

**Decision: Infinite scrolling with cursor-based pagination.**

The component will use TanStack Query's `useInfiniteQuery`.

Requirements:

- Fetch the initial page when the table mounts.
- Automatically fetch subsequent pages as the user approaches the bottom.
- Use an opaque backend cursor.
- Allow configurable page size, defaulting to 50.
- Prevent duplicate requests while a page is loading.
- Maintain previously loaded records.
- Display appropriate loading and error states.
- Avoid fetching beyond the end of the dataset.

The virtualizer will control rendering, not backend pagination.

The backend must provide deterministic ordering and stable cursor semantics.

### 2.3 Loading states

**Decision: Skeletons render inside DataTableBody, headers always visible.**

Loading states use shadcn `<Skeleton>` primitives rendered directly inside `DataTableBody`:

**Initial load** (`isInitialLoading`):
- `DataTableToolbar` and `DataTableHeader` are always visible.
- Only the row section renders skeletons: 5 rows matching the column grid,
  cell borders, and sticky column structure (selection + actions columns).
- Previously loaded rows (none on first load) are not replaced — there are none.
- No centered "Loading records…" text; skeletons provide the visual feedback.

**Load more** (`isLoadingMore` while data exists):
- Previously loaded data rows remain visible in `DataTableBody`.
- 5 skeleton rows stack **below** the existing data rows.
- The "Loading more…" text appears in the load-more bar beneath the skeletons.
- When the page arrives, skeletons unmount and `DataTableBody` renders all rows
  (old + new).

Skeleton rows mirror `DataTableBody` exactly:
- Same `gridTemplateColumns` from `columnWidths`.
- Same cell borders (`border-r border-border`) and row borders (`border-b border-border`).
- Same padding (`px-4 py-3`).
- Same sticky selection column (with `<Skeleton>` instead of `<Checkbox>`).
- Same sticky actions column (with `<Skeleton>` instead of dropdown menu).

The standalone `DataTableLoading` and `DataTableSkeleton` components have been removed.
All loading UI lives inside `DataTableBody` via `renderRowsLoading()`.

### 2.3 Row virtualization

Use TanStack Virtual to render only the rows needed for the current viewport and its overscan region.

Virtualization must not interfere with:

- Row selection.
- Column rendering.
- Row actions.
- Loading additional pages.
- Previously loaded records.

Selection must never depend on which rows are currently mounted in the DOM.

### 2.4 Server-side searching, filtering and sorting

All data operations will be server-driven.

The frontend will send search, filter and sorting parameters to the backend. The backend will execute these operations against the complete authorized dataset.

Supported in the initial version:

- Search with debounce.
- Resource-defined basic filters.
- Single-column sorting.
- Ascending and descending sort directions.

The shared component will not implement a generic advanced filter builder.

Each resource will define its own supported filter controls and corresponding parameters.

Sorting and filtering must be validated against backend allowlists.

Changing search or filters resets pagination and clears selection.

Changing sorting resets pagination but preserves selected IDs because sorting does not change the underlying filter scope.

### 2.5 URL synchronization

Search, filters and sorting will synchronize with URL query parameters.

Example:

`/students?search=ashley&status=active&sort=created_at&order=desc`

Requirements:

- Refreshing preserves table state.
- Browser back and forward navigation restore previous table state.
- Search changes are debounced.
- Transient search updates use history replacement to avoid excessive history entries.
- Committed filter and sorting changes should have sensible browser history behavior.
- URL state is the source of truth for shareable table configuration.
- Cursor state is not persisted in the URL.

### 2.6 Row selection

Selection will use stable record IDs through TanStack Table's `getRowId`.

The initial implementation supports selecting all currently loaded records.

Selection behavior:

1. Users can select individual rows.
2. The header checkbox selects all currently loaded records, including records outside the virtualized viewport.
3. The header checkbox supports checked, unchecked and indeterminate states.
4. Selection persists as additional pages are loaded.
5. Previously selected IDs remain selected when scrolling.
6. Newly fetched records are not automatically selected.
7. Selecting all again includes all records loaded at that point.
8. Changing search or filters clears selection.
9. Changing sorting preserves selection.
10. Selection is cleared after successful completion of applicable bulk actions.

Selecting all matching records across the entire backend dataset is explicitly outside the initial scope.

### 2.7 Floating bulk action sheet

The DataTable will provide a floating action sheet that appears whenever one or more records are selected.

Requirements:

- Fixed positioning near the bottom center of the viewport.
- Responsive layout.
- Rounded design with subtle shadow.
- Displays the selected record count.
- Provides a clear-selection action.
- Supports built-in common actions.
- Supports resource-specific custom actions.
- Disappears when selection is cleared.

Resource-specific actions will be supplied through a render prop.

Conceptual API:

```tsx
renderBulkActions={(context) => (
  <StudentBulkActions
    selectedIds={context.selectedIds}
    selectedCount={context.selectedCount}
    clearSelection={context.clearSelection}
    refresh={context.refresh}
  />
)}
```

The DataTable will provide the container and selection context. Resource pages will provide their action implementations.

### 2.8 Bulk deletion

The shared component will own the deletion lifecycle.

Responsibilities:

- Display deletion confirmation.
- Invoke the resource's deletion callback.
- Manage mutation loading state.
- Handle success and error notifications.
- Clear successfully deleted IDs.
- Preserve IDs associated with failed deletions.
- Refresh the relevant query after successful deletion.

Optimistic deletion will not be implemented initially.

The resource page supplies the actual deletion operation.

Partial success must be supported.

Example:

If 10 records are selected and 8 are deleted successfully, the 8 successful IDs are removed from selection while the remaining 2 stay selected.

### 2.9 Total record count

Exact total counts are optional.

The backend should provide:

- `next_cursor`
- `has_more`

The frontend must not require an exact total to operate.

Avoid expensive count queries solely to satisfy table UI requirements.

### 2.10 Configurable resource filters

The DataTable will support declarative, resource-specific filter configurations.

The shared component owns filter rendering, filter state, URL synchronization and request parameter serialization. Individual resource pages define the available filters and their options.

The initial filter configuration will support:

- Single-select dropdowns.
- Multi-select dropdowns.
- Searchable select options.
- Checkbox-based option selection.
- Grouped options.
- Nested dropdown submenus.

Filter controls will be composable and configurable through typed props.

Example:

```tsx
interface DataTableFilterOption {
  label: string;
  value: string;
  disabled?: boolean;
}

interface DataTableFilter {
  id: string;
  label: string;
  type: "select" | "multi-select";
  options: DataTableFilterOption[];
  searchable?: boolean;
  placeholder?: string;
  group?: string;
  defaultValue?: string | string[];
}
```

Resource configuration example:

```tsx
const filters: DataTableFilter[] = [
  {
    id: "status",
    label: "Status",
    type: "multi-select",
    options: [
      { label: "Active", value: "active" },
      { label: "Inactive", value: "inactive" },
      { label: "Graduated", value: "graduated" },
    ],
  },
];
```

The component must support nested filter menus where appropriate, without coupling the filter implementation to any specific resource.

Filter state will be synchronized with URL query parameters and serialized into the backend request.

Changing filters resets cursor pagination and clears row selection.

The backend remains responsible for validating filter fields and values.

The initial implementation will not include an advanced filter builder, arbitrary filter expressions, or user-defined filter logic.

Date filters and other specialized filter types may be introduced through future extensions.

### 2.11 Configurable row-level actions

The DataTable will support configurable actions for individual records.

Resource pages may supply a `renderRowActions` render prop.

Example:

```tsx
interface DataTableRowActionContext<T> {
  row: T;
  rowId: string;
  refresh: () => void;
}

interface DataTableProps<T> {
  renderRowActions?: (context: DataTableRowActionContext<T>) => React.ReactNode;
}
```

Requirements:

- Row actions are optional.
- Actions are rendered independently for each row.
- Resource pages control the available actions.
- Actions may include navigation, editing, status changes and deletion.
- Resource-specific permission checks remain outside the shared component.
- Actions must work with virtualized rows.
- Row actions must not accidentally trigger row selection.
- The component must support a consistent shadcn dropdown-menu presentation.

The shared component provides the rendering location and row context, not resource-specific business logic.

## 3. Backend API contract

### 3.1 List endpoint

Example:

`GET /api/v1/students`

Supported parameters:

| Parameter        | Description                         |
| ---------------- | ----------------------------------- |
| `limit`          | Requested page size                 |
| `cursor`         | Opaque pagination cursor            |
| `search`         | Search query                        |
| `sort`           | Whitelisted sortable field          |
| `order`          | `asc` or `desc`                     |
| Resource filters | Resource-specific filter parameters |

Example:

```http
GET /api/v1/students?limit=50&sort=created_at&order=desc&status=active
```

### 3.2 Standard response

```json
{
  "data": [
    {
      "id": "019...",
      "first_name": "Ashley",
      "last_name": "Example",
      "status": "active"
    }
  ],
  "meta": {
    "next_cursor": "opaque-cursor",
    "has_more": true
  }
}
```

Requirements:

- `data` contains the records for the current batch.
- `next_cursor` is opaque to the frontend.
- `has_more` determines whether another request is necessary.
- `next_cursor` should be null when there are no more records.
- Exact totals are not required.

### 3.3 Cursor stability

The backend must implement deterministic ordering.

For example, sorting by `created_at` should use a unique ID as a secondary tie-breaker.

Conceptually:

```sql
ORDER BY created_at DESC, id DESC
```

Cursor pagination must account for both values.

The cursor must not be treated as a client-controlled database query.

### 3.4 Bulk deletion endpoint

Recommended convention:

`POST /api/v1/students/bulk-delete`

Request:

```json
{
  "ids": ["019...", "019...", "019..."]
}
```

Response:

```json
{
  "deleted_ids": ["019...", "019..."],
  "failed": [
    {
      "id": "019...",
      "code": "DEPENDENCY_EXISTS",
      "message": "Student has an active dependency."
    }
  ]
}
```

The backend must independently validate:

- Organization access.
- School access.
- User permissions.
- Record existence.
- Business deletion constraints.

Selected IDs supplied by the frontend must never be considered authorization evidence.

## 4. Component API

The initial public API should remain generic and resource-independent.

Proposed TypeScript contract:

```tsx
interface DataTablePage<T> {
  data: T[];
  nextCursor: string | null;
  hasMore: boolean;
}

interface DataTableBulkActionContext {
  selectedIds: string[];
  selectedCount: number;
  clearSelection: () => void;
  refresh: () => void;
}

interface DataTableProps<T> {
  queryKey: QueryKey;

  columns: ColumnDef<T>[];

  fetchPage: (params: {
    cursor: string | null;
    limit: number;
    search?: string;
    sorting: SortingState;
    filters: Record<string, unknown>;
  }) => Promise<DataTablePage<T>>;

  getRowId: (row: T) => string;

  pageSize?: number;

  renderBulkActions?: (context: DataTableBulkActionContext) => React.ReactNode;

  onDelete?: (ids: string[]) => Promise<BulkDeleteResult>;

  enableRowSelection?: boolean;

  enableVirtualization?: boolean;
}
```

The deletion result should provide enough information to distinguish successful and unsuccessful IDs.

```tsx
interface BulkDeleteResult {
  deletedIds: string[];

  failed: {
    id: string;
    code?: string;
    message: string;
  }[];
}
```

This is the initial API contract and may be refined during implementation.

## 5. Suggested project structure

```text
components/
└── shared/
    └── data-table/
        ├── components/
            ├── data-table.tsx
            ├── data-table-header.tsx
            ├── data-table-body.tsx
            ├── data-table-toolbar.tsx
            ├── data-table-selection-sheet.tsx
            ├── data-table-delete-dialog.tsx
            ├── data-table-empty.tsx
            └── data-table-error.tsx
        ├── hooks/
        │   ├── use-data-table.ts
        │   ├── use-data-table-selection.ts
        │   └── use-data-table-url-state.ts
        ├── types.ts
        └── utils.ts

features/
├── students/
│   ├── api.ts
│   ├── columns.tsx
│   ├── filters.tsx
│   └── components/
│       ├── students-table.tsx
│       └── student-bulk-actions.tsx
│
└── teachers/
    ├── api.ts
    ├── columns.tsx
    ├── filters.tsx
    └── components/
        ├── teachers-table.tsx
        └── teacher-bulk-actions.tsx
```

## 6. Non-goals

The following capabilities are explicitly excluded from the initial implementation:

- Selecting all matching backend records.
- Advanced filter builder.
- Multi-column sorting.
- Exact total count requirements.
- Traditional page-number pagination.
- Optimistic deletion.
- Generic export implementation.
- Column resizing.
- Column reordering.
- Column visibility management.
- Resource-specific business logic inside the shared component.

These may be introduced later without compromising the core architecture.

## 7. Consequences

### Positive

- Consistent table behavior throughout the application.
- Reduced duplication across resource pages.
- Efficient rendering of large datasets.
- Backend remains responsible for data processing and authorization.
- Resource-specific actions remain extensible.
- Common deletion and selection behavior is centralized.
- URL state makes tables shareable and navigable.
- Cursor pagination avoids the need to fetch large datasets upfront.

### Trade-offs

- More coordination is required between frontend and backend API contracts.
- Cursor pagination requires careful deterministic ordering.
- Server-driven filtering and sorting introduce network requests.
- Selection state requires explicit management across virtualized and paginated data.
- URL synchronization requires careful handling of browser navigation and query state.

These trade-offs are accepted.

## 8. Acceptance criteria

The implementation will be considered successful when:

1. The same shared component can render different resource types without resource-specific modifications.
2. Initial data loads through TanStack Query.
3. Additional pages load automatically during scrolling.
4. Virtualization renders only the required visible rows.
5. Search, sorting and filters are executed by the backend.
6. Search, sorting and filters synchronize with the URL.
7. Selection works across all currently loaded pages.
8. Selection survives additional page loads.
9. The floating action sheet appears when records are selected.
10. Resource-specific bulk actions can be rendered through the public component API.
11. Bulk deletion supports partial success and failure.
12. Failed deletion IDs remain selected.
13. Changing filters clears selection.
14. Changing sorting preserves selection.
15. The component does not depend on exact total record counts.
16. Backend authorization is enforced independently of frontend state.
17. During initial load, `DataTableToolbar` and `DataTableHeader` remain visible;
    only the row section renders shadcn `<Skeleton>` rows inside `DataTableBody`.
18. During load more, previously loaded data rows remain visible and skeleton rows
    stack below them inside `DataTableBody`.
19. Skeleton rows match the exact grid, borders, padding, and sticky column
    structure of real data rows.

## 9. Implementation principle

The shared DataTable should own infrastructure, not business logic.

Resource pages define what is displayed and which operations are available. The shared component handles how records are fetched, rendered, selected and acted upon.

The backend remains the source of truth for data, permissions and business constraints.

This ADR establishes the architectural direction for the first implementation.
