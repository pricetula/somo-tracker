import type { ReactNode } from "react";

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
}
