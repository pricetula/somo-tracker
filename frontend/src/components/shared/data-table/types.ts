import type { LucideIcon } from "lucide-react";
import type { ReactNode } from "react";

// ─── Normalized API result ───────────────────────────────────────────────

/**
 * The common shape your swagger-generated `*ListResult` types are
 * normalized into. Most of your generated list results already look like
 * this (e.g. `ClassListResult`). If a particular resource's generated type
 * uses different field names (e.g. `results` instead of `items`, or
 * `count` instead of `total`), don't change this type — instead pass a
 * `normalize` function to <DataTable /> for that one resource. That keeps
 * the table itself decoupled from any single generated type.
 */
export interface NormalizedListResult<TItem> {
    items: TItem[];
    /** Total number of rows across all pages, if the API returns one. */
    total?: number;
    /** The page number this result represents (1-indexed). */
    page?: number;
    /** Page size used for this result. */
    limit?: number;
    /** Explicit "is there another page" flag, if the API returns one. */
    hasMore?: boolean;
}

// ─── API function signatures ─────────────────────────────────────────────

/**
 * Matches the signature of every generated `list*` function, e.g.:
 *   listClasses(params: { academic_year_id: string; page?: number; limit?: number }): Promise<ClassListResult>
 *
 * TParams is whatever resource-specific filters that function needs
 * (minus page/limit, which the table controls).
 */
export type ListApiFn<TResult, TParams extends object> = (
    params: TParams & { page?: number; limit?: number }
) => Promise<TResult>;

/**
 * Signature for a delete function that the DataTable calls.
 * The function receives the row id and returns a promise.
 */
export type DeleteApiFn<TParams extends object = object> = (
    id: string | number,
    params?: TParams
) => Promise<void>;

// ─── Column definition ───────────────────────────────────────────────────

export interface DataTableColumn<TItem> {
    /** Unique column id, also used as the React key. */
    id: string;
    header: ReactNode;
    /** Render the cell for a given row. */
    cell: (row: TItem, index: number) => ReactNode;
    /**
     * Column width. Any valid CSS grid track value: "120px", "1fr", "minmax(120px,1fr)".
     * Defaults to "1fr".
     */
    width?: string;
    align?: "left" | "center" | "right";
    /** Additional class name for the cell. */
    className?: string;
}

// ─── Filter types ────────────────────────────────────────────────────────

export interface FilterItem {
    id: string;
    label: string;
    icon?: LucideIcon;
    value: string;
    /** If present, this item acts as a label-only grouping node (not selectable). */
    submenu?: FilterItem[];
}

export interface FilterGroup {
    id: string;
    label: string;
    icon?: LucideIcon;
    type: "single" | "multi";
    items: FilterItem[];
}

// ─── DataTable props ─────────────────────────────────────────────────────

export interface DataTableProps<TItem, TParams extends object, TResult> {
    /** Base query key array for the list query, e.g. ["classes"]. */
    queryKey: readonly unknown[];
    /** Any generated `list*` function, e.g. `listClasses`. */
    queryFn: ListApiFn<TResult, TParams>;
    /** Resource-specific filters (excluding page/limit/search/user-filters). */
    params: TParams;
    columns: DataTableColumn<TItem>[];
    /** Extracts a stable id from a row — used by selection and optimistic delete. */
    getRowId: (row: TItem, index: number) => string | number;
    /** Only needed if the generated result's shape isn't {items, total, page, limit}. */
    normalize?: (result: TResult) => NormalizedListResult<TItem>;

    // ─── Search ──────────────────────────────────────────────────────
    isSearchable?: boolean;
    searchPlaceholder?: string;

    // ─── Filter ──────────────────────────────────────────────────────
    filterGroups?: FilterGroup[];

    // ─── Checkbox / Selection ────────────────────────────────────────
    isCheckable?: boolean;

    // ─── Delete ──────────────────────────────────────────────────────
    /** Mutation key for the delete mutation, tracked independently from the list query. */
    deleteMutationKey?: readonly unknown[];
    /** Delete function. Receives the row id. */
    deleteFn?: DeleteApiFn<TParams>;
    /** Extra params forwarded to deleteFn (e.g. school_id for nested resources). */
    deleteParams?: TParams;

    // ─── Add ─────────────────────────────────────────────────────────
    /** If provided, renders an add button as a Link to this href. */
    addHref?: string;

    // ─── Sizing ──────────────────────────────────────────────────────
    /** Estimated row height in px, used by the virtualizer and skeleton rows. Defaults to 44. */
    rowHeight?: number;
    /** Height of the scrollable viewport in px. Defaults to 600. */
    height?: number;
    /** Rows fetched per page. Defaults to 50. */
    pageSize?: number;

    // ─── States ──────────────────────────────────────────────────────
    /** Shown when there's no data at all (no search/filters active). */
    emptyState?: ReactNode;
    /** Shown when search/filters narrowed results to zero. */
    noResultsState?: ReactNode;

    className?: string;
}
