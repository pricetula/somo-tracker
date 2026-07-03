"use client";

import { useVirtualizer } from "@tanstack/react-virtual";
import { useEffect, useRef, useState } from "react";
import { useDebouncedValue } from "./use-debounced-value";
import type { DataTableColumn, ListApiFn, NormalizedListResult } from "./types";
import { useInfiniteListQuery } from "./use-infinite-list-query";

export interface DataTableProps<TItem, TParams extends object, TResult> {
    /** Base query key, e.g. ["classes"]. */
    queryKey: readonly unknown[];
    /** Any generated `list*` function, e.g. `listClasses`. */
    queryFn: ListApiFn<TResult, TParams>;
    /** Resource-specific filters (excluding page/limit/search). */
    params: TParams;
    columns: DataTableColumn<TItem>[];
    getRowId: (row: TItem, index: number) => string | number;
    /** Only needed if the generated result's shape isn't {items, total, page, limit}. */
    normalize?: (result: TResult) => NormalizedListResult<TItem>;

    /**
     * If provided, a search input is shown above the table. The input's
     * value is debounced (350ms) and passed to onSearch — typically you'd
     * fold that back into `params` (e.g. a `search` field) in the parent.
     */
    onSearch?: (term: string) => void;
    searchPlaceholder?: string;

    /** Rows fetched per page. Defaults to 50. */
    pageSize?: number;
    /** Estimated row height in px, used by the virtualizer. Defaults to 44. */
    rowHeight?: number;
    /** Height of the scrollable viewport. Defaults to 600px. */
    height?: number;
    emptyState?: React.ReactNode;
    className?: string;
}

export function DataTable<TItem, TParams extends object, TResult>({
    queryKey,
    queryFn,
    params,
    columns,
    getRowId,
    normalize,
    onSearch,
    searchPlaceholder = "Search...",
    pageSize = 50,
    rowHeight = 44,
    height = 600,
    emptyState,
    className,
}: DataTableProps<TItem, TParams, TResult>) {
    const {
        rows,
        total,
        isLoading,
        isError,
        error,
        isFetchingNextPage,
        hasNextPage,
        fetchNextPage,
    } = useInfiniteListQuery({
        queryKey,
        queryFn,
        params,
        limit: pageSize,
        normalize,
    });

    // --- search -------------------------------------------------------
    const [searchTerm, setSearchTerm] = useState("");
    const debouncedSearch = useDebouncedValue(searchTerm, 350);
    const onSearchRef = useRef(onSearch);
    useEffect(() => {
        onSearchRef.current = onSearch;
    });
    useEffect(() => {
        onSearchRef.current?.(debouncedSearch);
    }, [debouncedSearch]);

    // --- virtualization -------------------------------------------------
    const parentRef = useRef<HTMLDivElement>(null);
    const rowCount = hasNextPage ? rows.length + 1 : rows.length;

    // eslint-disable-next-line react-hooks/incompatible-library
    const virtualizer = useVirtualizer({
        count: rowCount,
        getScrollElement: () => parentRef.current,
        estimateSize: () => rowHeight,
        overscan: 8,
    });

    const virtualItems = virtualizer.getVirtualItems();

    // Fetch the next page once the user scrolls near the last loaded row.
    useEffect(() => {
        const lastItem = virtualItems[virtualItems.length - 1];
        if (!lastItem) return;
        if (lastItem.index >= rows.length - 1 && hasNextPage && !isFetchingNextPage) {
            fetchNextPage();
        }
    }, [virtualItems, rows.length, hasNextPage, isFetchingNextPage, fetchNextPage]);

    const gridTemplateColumns = columns.map((col) => col.width ?? "1fr").join(" ");

    return (
        <div className={className}>
            {onSearch && (
                <div className="mb-3">
                    <input
                        type="text"
                        value={searchTerm}
                        onChange={(e) => setSearchTerm(e.target.value)}
                        placeholder={searchPlaceholder}
                        className="w-full max-w-sm rounded-md border border-gray-300 px-3 py-2 text-sm outline-none focus:border-gray-500"
                    />
                </div>
            )}
            {total}
            <div className="rounded-md border border-gray-200">
                {/* header */}
                <div
                    className="grid border-b border-gray-200 text-xs font-medium tracking-wide text-gray-500 uppercase"
                    style={{ gridTemplateColumns }}
                >
                    {columns.map((col) => (
                        <div
                            key={col.id}
                            className="truncate px-3 py-2"
                            style={{ textAlign: col.align ?? "left" }}
                        >
                            {col.header}
                        </div>
                    ))}
                </div>

                {/* body */}
                {isError ? (
                    <div className="p-6 text-center text-sm text-red-600">
                        Failed to load data{error instanceof Error ? `: ${error.message}` : "."}
                    </div>
                ) : isLoading ? (
                    <div className="p-6 text-center text-sm text-gray-500">Loading…</div>
                ) : rows.length === 0 ? (
                    (emptyState ?? (
                        <div className="p-6 text-center text-sm text-gray-500">
                            No results found.
                        </div>
                    ))
                ) : (
                    <div ref={parentRef} style={{ height, overflow: "auto" }}>
                        <div
                            style={{
                                height: virtualizer.getTotalSize(),
                                position: "relative",
                            }}
                        >
                            {virtualItems.map((virtualRow) => {
                                const isLoaderRow = virtualRow.index >= rows.length;
                                const row = rows[virtualRow.index];

                                return (
                                    <div
                                        key={
                                            isLoaderRow ? "loader" : getRowId(row, virtualRow.index)
                                        }
                                        data-index={virtualRow.index}
                                        ref={virtualizer.measureElement}
                                        className="absolute top-0 left-0 w-full border-b border-gray-100"
                                        style={{ transform: `translateY(${virtualRow.start}px)` }}
                                    >
                                        {isLoaderRow ? (
                                            <div className="px-3 py-2 text-center text-xs text-gray-400">
                                                Loading more…
                                            </div>
                                        ) : (
                                            <div className="grid" style={{ gridTemplateColumns }}>
                                                {columns.map((col) => (
                                                    <div
                                                        key={col.id}
                                                        className="truncate px-3 py-2 text-sm text-gray-800"
                                                        style={{ textAlign: col.align ?? "left" }}
                                                    >
                                                        {col.cell(row, virtualRow.index)}
                                                    </div>
                                                ))}
                                            </div>
                                        )}
                                    </div>
                                );
                            })}
                        </div>
                    </div>
                )}
            </div>

            {typeof total === "number" && (
                <div className="mt-2 text-xs text-gray-400">
                    {rows.length} of {total} loaded
                </div>
            )}
        </div>
    );
}
