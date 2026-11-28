# ADR-0027: Reusable Server-Driven DataTable

## Metadata

| Field | Value |
|-------|-------|
| id | ADR-0027 |
| title | Reusable Server-Driven DataTable |
| status | accepted |
| date | 2026-11-26 |
| author | frontend, backend |
| deciders | frontend, backend |
| relates_to | |

## Status

accepted

## Context

Somotracker needs to display large datasets across many resources — students, teachers, classes, fees, payments, and other entities. Without a shared table abstraction, each resource page duplicates fetching, pagination, selection, sorting, filtering, virtualization, and deletion logic.

The duplication increases maintenance burden, creates inconsistent table behavior across the application, and slows down development of new resource pages. We need a generic, reusable data table component that owns common table infrastructure while allowing individual resource pages to define their columns, filters, and resource-specific actions.

The component must integrate with Next.js, TanStack Query, TanStack Table, TanStack Virtual, shadcn/ui, and TypeScript. The backend is independently owned and can be designed to support the frontend's requirements.

## Decision

We will implement a generic, server-driven DataTable component.

The shared component will own the common data lifecycle and table interactions. Resource-specific configuration will supply data definitions and business-specific operations. The DataTable itself will be a client-side component.

### Technology responsibilities

| Technology | Responsibility |
|------------|----------------|
| TanStack Query | Data fetching, caching, infinite queries and mutations |
| TanStack Table | Column definitions, sorting, filtering state and row selection |
| TanStack Virtual | Efficient rendering of large loaded datasets |
| shadcn/ui | Table controls, dialogs, dropdowns, toolbars and other UI primitives |
| Next.js | Application routing and client/server component boundaries |

### Pagination

**Decision: Infinite scrolling with cursor-based pagination.**

The component uses TanStack Query's `useInfiniteQuery`.

Requirements:

- Fetch the initial page when the table mounts.
- Automatically fetch subsequent pages as the user approaches the bottom.
- Use an opaque backend cursor.
- Allow configurable page size, defaulting to 50.
- Prevent duplicate requests while a page is loading.
- Maintain previously loaded records.
- Display appropriate loading and error states.
- Avoid fetching beyond the end of the dataset.

The virtualizer controls rendering, not backend pagination.

The backend must provide deterministic ordering and stable cursor semantics.

### Row virtualization

Use TanStack Virtual to render only the rows needed for the current viewport and its overscan region.

Virtualization must not interfere with:

- Row selection.
- Column rendering.
- Row actions.
- Loading additional pages.
- Previously loaded records.

Selection must never depend on which rows are currently mounted in the DOM.

### Server-side searching, filtering and sorting

All data operations are server-driven.

The frontend sends search, filter and sorting parameters to the backend. The backend executes these operations against the complete authorized dataset.

Supported in the initial version:

- Search with debounce.
- Resource-defined basic filters.
- Single-column sorting.
- Ascending and descending sort directions.

The shared component does not implement a generic advanced filter builder.

Each resource defines its own supported filter controls and corresponding parameters.

Sorting and filtering must be validated against backend allowlists.

Changing search or filters resets pagination and clears selection.

Changing sorting resets pagination but preserves selected IDs because sorting does not change the underlying filter scope.

### URL synchronization

Search, filters and sorting synchronize with URL query parameters.

Example:

`/students?search=ashley&status=active&sort=created_at&order=desc`

Requirements:

- Refreshing preserves table state.
- Browser back and forward navigation restore previous table state.
- Search changes are debounced.
- Transient search updates use history replacement to avoid excessive history entries.
- Committed filter and sorting changes have sensible browser history behavior.
- URL state is the source of truth for shareable table configuration.
- Cursor state is not persisted in the URL.

### Row selection

Selection uses stable record IDs through TanStack Table's `getRowId`.

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

### Floating bulk action sheet

The DataTable provides a floating action sheet that appears whenever one or more records are selected.

Requirements:

- Fixed positioning near the bottom center of the viewport.
- Responsive layout.
- Rounded design with subtle shadow.
- Displays the selected record count.
- Provides a clear-selection action.
- Supports built-in common actions.
- Supports resource-specific custom actions.
- Disappears when selection is cleared.

Resource-specific actions are supplied through a render prop.

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

The DataTable provides the container and selection context. Resource pages provide their action implementations.

### Bulk deletion

The shared component owns the deletion lifecycle.

Responsibilities:

- Display deletion confirmation.
- Invoke the resource's deletion callback.
- Manage mutation loading state.
- Handle success and error notifications.
- Clear successfully deleted IDs.
- Preserve IDs associated with failed deletions.
- Refresh the relevant query after successful deletion.

Optimistic deletion is not implemented initially.

The resource page supplies the actual deletion operation.

Partial success is supported.

Example:

If 10 records are selected and 8 are deleted successfully, the 8 successful IDs are removed from selection while the remaining 2 stay selected.

### Total record count

Exact total counts are optional.

The backend should provide:

- `next_cursor`
- `has_more`

The frontend does not require an exact total to operate.

Avoid expensive count queries solely to satisfy table UI requirements.

### Configurable resource filters

The DataTable supports declarative, resource-specific filter configurations.

The shared component owns filter rendering, filter state, URL synchronization and request parameter serialization. Individual resource pages define the available filters and their options.

The initial filter configuration supports:

- Single-select dropdowns.
- Multi-select dropdowns.
- Searchable select options.
- Checkbox-based option selection.
- Grouped options.
- Nested dropdown submenus.

Filter controls are composable and configurable through typed props.

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

The component supports nested filter menus where appropriate, without coupling the filter implementation to any specific resource.

Filter state is synchronized with URL query parameters and serialized into the backend request.

Changing filters resets cursor pagination and clears row selection.

The backend remains responsible for validating filter fields and values.

The initial implementation does not include an advanced filter builder, arbitrary filter expressions, or user-defined filter logic.

Date filters and other specialized filter types may be introduced through future extensions.

### Configurable row-level actions

The DataTable supports configurable actions for individual records.

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
- Actions work with virtualized rows.
- Row actions must not accidentally trigger row selection.
- The component supports a consistent shadcn dropdown-menu presentation.

The shared component provides the rendering location and row context, not resource-specific business logic.

## Alternatives Considered

| Alternative | Pros | Cons | Why rejected |
|-------------|------|------|--------------|
| Per-resource table implementations (duplicate table logic per page) | Simple initially, no shared abstraction needed | High duplication, inconsistent behavior, slow to add new pages | Rejected — does not scale across many resources |
| Client-side pagination (load all records upfront) | No cursor complexity, instant navigation after load | Poor performance on large datasets, excessive memory | Rejected — datasets can be large |
| Exact total count in every response | Simple UI for "X of Y" indicators | Expensive count queries on large tables | Rejected — count is optional per ADR |
| Optimistic deletion | Faster perceived deletion | Complex rollback on failure, partial success handling harder | Rejected — defer to future |
| Multi-column sorting | More flexible sorting | More complex cursor and backend logic | Rejected — single-column sufficient initially |
| Traditional page-number pagination | Familiar UI pattern | Less efficient than cursor pagination for large datasets | Rejected — cursor chosen |
| Generic advanced filter builder | Users can build arbitrary filters | Significant complexity, not needed initially | Rejected — resource-defined filters sufficient |

## Consequences

### Positive

- Consistent table behavior throughout the application.
- Reduced duplication across resource pages.
- Efficient rendering of large datasets.
- Backend remains responsible for data processing and authorization.
- Resource-specific actions remain extensible.
- Common deletion and selection behavior is centralized.
- URL state makes tables shareable and navigable.
- Cursor pagination avoids the need to fetch large datasets upfront.

### Negative

- More coordination required between frontend and backend API contracts.
- Cursor pagination requires careful deterministic ordering.
- Server-driven filtering and sorting introduce network requests.
- Selection state requires explicit management across virtualized and paginated data.
- URL synchronization requires careful handling of browser navigation and query state.

### Neutral / Risks

- Social engineering risk if school admin is compromised (for recovery-related actions) — mitigated by multi-person verification.
- Backend authorization must be enforced independently of frontend state; frontend-supplied IDs are never authorization evidence.

## Implementation Notes

### Backend API contract

#### List endpoint

Example:

`GET /api/v1/students`

Supported parameters:

| Parameter | Description |
|-----------|-------------|
| `limit` | Requested page size |
| `cursor` | Opaque pagination cursor |
| `search` | Search query |
| `sort` | Whitelisted sortable field |
| `order` | `asc` or `desc` |
| Resource filters | Resource-specific filter parameters |

Example:

```http
GET /api/v1/students?limit=50&sort=created_at&order=desc&status=active
```

#### Standard response

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
- `next_cursor` is null when there are no more records.
- Exact totals are not required.

#### Cursor stability

The backend must implement deterministic ordering.

For example, sorting by `created_at` should use a unique ID as a secondary tie-breaker.

Conceptually:

```sql
ORDER BY created_at DESC, id DESC
```

Cursor pagination must account for both values.

The cursor must not be treated as a client-controlled database query.

#### Bulk deletion endpoint

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

### Component API

The initial public API remains generic and resource-independent.

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

### Suggested project structure

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
            ├── data-table-loading.tsx
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
│   ├── filters.tsx
    └── components/
        ├── teachers-table.tsx
        └── teacher-bulk-actions.tsx
```

### Non-goals

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

### Acceptance criteria

The implementation is considered successful when:

1. The same shared component renders different resource types without resource-specific modifications.
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

### Implementation principle

The shared DataTable should own infrastructure, not business logic.

Resource pages define what is displayed and which operations are available. The shared component handles how records are fetched, rendered, selected and acted upon.

The backend remains the source of truth for data, permissions and business constraints.

This ADR establishes the architectural direction for the first implementation.

## Links

- Authentication: `docs/adr/0001-authentication-architecture.md`
- Multi-tenancy: `docs/adr/0002-multi-tenancy-model.md`
