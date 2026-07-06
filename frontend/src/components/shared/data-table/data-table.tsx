"use client";

import { useCallback, useMemo, useRef, useState } from "react";
import { useVirtualizer } from "@tanstack/react-virtual";
import Link from "next/link";
import { toast } from "sonner";
import { useQueryClient } from "@tanstack/react-query";
import { Search, Plus, Trash2 } from "lucide-react";

import { useDebouncedValue } from "./use-debounced-value";
import { useInfiniteListQuery } from "./use-infinite-list-query";
import { FilterDropdown } from "./filter-dropdown";
import { SkeletonRows } from "./skeleton-rows";
import type { DataTableProps, NormalizedListResult } from "./types";

import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
    AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { getErrorMessage } from "@/lib/errors";
import { cn } from "@/lib/utils";

// ─── Main DataTable ──────────────────────────────────────────────────────

export function DataTable<TItem, TParams extends object, TResult>({
    queryKey,
    queryFn,
    params,
    columns,
    getRowId,
    normalize,
    isSearchable,
    searchPlaceholder = "Search...",
    filterGroups,
    isCheckable,
    deleteFn,
    deleteParams,
    addHref,
    rowHeight = 44,
    height = 600,
    pageSize = 50,
    emptyState,
    noResultsState,
    className,
}: DataTableProps<TItem, TParams, TResult>) {
    const queryClient = useQueryClient();

    // ── Search state ─────────────────────────────────────────────────
    const [searchTerm, setSearchTerm] = useState("");
    const debouncedSearch = useDebouncedValue(searchTerm, 300);

    // ── Filter state ─────────────────────────────────────────────────
    // Keyed by FilterItem id. button → string, sub_menu_single → string, sub_menu_multi → string[].
    const [activeFilters, setActiveFilters] = useState<Record<string, string | string[]>>({});

    // ── Selection state ──────────────────────────────────────────────
    const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());

    // ── Delete confirm open ──────────────────────────────────────────
    const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false);

    // ── Infinite query ───────────────────────────────────────────────
    const listQuery = useInfiniteListQuery({
        queryKey,
        queryFn,
        params,
        search: isSearchable ? debouncedSearch : undefined,
        filters: filterGroups && Object.keys(activeFilters).length > 0 ? activeFilters : undefined,
        limit: pageSize,
        normalize,
    });

    const {
        rows,
        total,
        isPending,
        isError,
        error,
        isFetchingNextPage,
        hasNextPage,
        fetchNextPage,
    } = listQuery;

    // ── Track whether toolbar has ever been enabled ──────────────────
    const hasLoadedOnceRef = useRef(false);
    if (!isPending) {
        hasLoadedOnceRef.current = true;
    }
    const isToolbarDisabled = !hasLoadedOnceRef.current && isPending;

    // ── Toast on pagination errors (page 2+) ─────────────────────────
    const prevErrorRef = useRef<typeof error>(null);
    if (isError && error && error !== prevErrorRef.current && !isPending) {
        prevErrorRef.current = error;
        // Defer toast to avoid setState-in-render issues
        queueMicrotask(() => {
            toast.error(getErrorMessage(error));
        });
    }
    if (!isError) {
        prevErrorRef.current = null;
    }

    // ── Event handlers ───────────────────────────────────────────────

    const handleSearchChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        setSearchTerm(e.target.value);
        setSelectedIds(new Set());
    }, []);

    const handleToggleButton = useCallback((itemId: string, itemValue: string) => {
        setSelectedIds(new Set());
        setActiveFilters((prev) => {
            const current = prev[itemId];
            if (typeof current === "string" && current === itemValue) {
                // Toggle off
                // eslint-disable-next-line @typescript-eslint/no-unused-vars
                const { [itemId]: _, ...rest } = prev;
                return rest;
            }
            // Toggle on
            return { ...prev, [itemId]: itemValue };
        });
    }, []);

    const handleSelectSingle = useCallback((itemId: string, subValue: string) => {
        setSelectedIds(new Set());
        setActiveFilters((prev) => {
            const current = prev[itemId];
            if (typeof current === "string" && current === subValue) {
                // Deselect
                // eslint-disable-next-line @typescript-eslint/no-unused-vars
                const { [itemId]: _, ...rest } = prev;
                return rest;
            }
            return { ...prev, [itemId]: subValue };
        });
    }, []);

    const handleToggleMulti = useCallback((itemId: string, subValue: string) => {
        setSelectedIds(new Set());
        setActiveFilters((prev) => {
            const current = prev[itemId];
            const arr = Array.isArray(current) ? current : [];
            const next = arr.includes(subValue)
                ? arr.filter((v) => v !== subValue)
                : [...arr, subValue];
            if (next.length === 0) {
                // eslint-disable-next-line @typescript-eslint/no-unused-vars
                const { [itemId]: _, ...rest } = prev;
                return rest;
            }
            return { ...prev, [itemId]: next };
        });
    }, []);

    const handleSelectAll = useCallback(
        (checked: boolean) => {
            if (checked) {
                const ids = new Set(rows.map((row, i) => String(getRowId(row, i))));
                setSelectedIds(ids);
            } else {
                setSelectedIds(new Set());
            }
        },
        [rows, getRowId]
    );

    const handleSelectRow = useCallback((id: string) => {
        setSelectedIds((prev) => {
            const next = new Set(prev);
            if (next.has(id)) {
                next.delete(id);
            } else {
                next.add(id);
            }
            return next;
        });
    }, []);

    // ── Delete mutation ──────────────────────────────────────────────
    const [deletingIds, setDeletingIds] = useState<Set<string>>(new Set());

    const handleDeleteConfirm = useCallback(async () => {
        if (!deleteFn) return;

        setDeleteConfirmOpen(false);
        const idsToDelete = [...selectedIds];
        const idSet = new Set(idsToDelete);

        // Mark as deleting
        setDeletingIds((prev) => {
            const next = new Set(prev);
            idsToDelete.forEach((id) => next.add(id));
            return next;
        });

        // Optimistic removal from cache
        await queryClient.cancelQueries({ queryKey });
        const previousData = queryClient.getQueryData(queryKey);

        queryClient.setQueryData(queryKey, (old: unknown) => {
            if (!old) return old;
            const data = old as {
                pages: NormalizedListResult<TItem>[];
                pageParams: unknown[];
            };
            return {
                ...data,
                pages: data.pages.map((page) => ({
                    ...page,
                    items: page.items.filter((row, i) => !idSet.has(String(getRowId(row, i)))),
                })),
            };
        });

        // Clear selection
        setSelectedIds(new Set());

        try {
            // Delete sequentially
            for (const id of idsToDelete) {
                await deleteFn(id, deleteParams);
            }
            // Invalidate to get fresh data
            queryClient.invalidateQueries({ queryKey });
        } catch (err) {
            // Rollback
            if (previousData) {
                queryClient.setQueryData(queryKey, previousData);
            }
            toast.error(getErrorMessage(err));
        } finally {
            setDeletingIds(new Set());
        }
    }, [deleteFn, deleteParams, queryClient, queryKey, selectedIds, getRowId]);

    // ── Determine if "Check all" is checked / indeterminate ──────────
    const selectableRows = useMemo(
        () => rows.filter((row, i) => !deletingIds.has(String(getRowId(row, i)))),
        [rows, deletingIds, getRowId]
    );

    const allSelected =
        selectableRows.length > 0 &&
        selectableRows.every((row, i) => selectedIds.has(String(getRowId(row, i))));
    const someSelected =
        selectableRows.some((row, i) => selectedIds.has(String(getRowId(row, i)))) && !allSelected;

    // ── Grid template ────────────────────────────────────────────────
    const gridTemplateColumns = useMemo(() => {
        const cols: string[] = [];
        if (isCheckable) cols.push("36px");
        columns.forEach((col) => cols.push(col.width ?? "1fr"));
        return cols.join(" ");
    }, [isCheckable, columns]);

    // ── Virtualizer ──────────────────────────────────────────────────
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

    // Fetch next page when scrolling near the last loaded row
    const lastVirtualIndex =
        virtualItems.length > 0 ? virtualItems[virtualItems.length - 1].index : -1;

    if (lastVirtualIndex >= rows.length - 1 && hasNextPage && !isFetchingNextPage) {
        fetchNextPage();
    }

    // ── State checks ─────────────────────────────────────────────────

    const hasData = rows.length > 0;
    const isInitialPending = isPending && !hasData;
    const isInitialError = isError && !hasData;
    const isSearchOrFilterActive =
        (isSearchable && debouncedSearch.length > 0) ||
        Object.values(activeFilters).some((v) => {
            if (Array.isArray(v)) return v.length > 0;
            return v !== "";
        });

    // ── Render ───────────────────────────────────────────────────────

    return (
        <div className={cn("flex flex-col gap-2", className)}>
            {/* ── Toolbar ───────────────────────────────────────── */}
            <div className="flex items-center gap-2">
                {isSearchable && (
                    <div className="relative max-w-60 flex-1">
                        <Search className="text-muted-foreground pointer-events-none absolute top-1/2 left-2 size-3.5 -translate-y-1/2" />
                        <Input
                            type="text"
                            value={searchTerm}
                            onChange={handleSearchChange}
                            placeholder={searchPlaceholder}
                            disabled={isToolbarDisabled}
                            className="pl-7"
                        />
                    </div>
                )}

                {filterGroups && filterGroups.length > 0 && (
                    <FilterDropdown
                        groups={filterGroups}
                        activeFilters={activeFilters}
                        onToggleButton={handleToggleButton}
                        onSelectSingle={handleSelectSingle}
                        onToggleMulti={handleToggleMulti}
                        disabled={isToolbarDisabled}
                    />
                )}

                <div className="ml-auto flex items-center gap-1.5">
                    {selectedIds.size > 0 && deleteFn && (
                        <AlertDialog open={deleteConfirmOpen} onOpenChange={setDeleteConfirmOpen}>
                            <AlertDialogTrigger asChild>
                                <Button
                                    variant="destructive"
                                    size="xs"
                                    disabled={isToolbarDisabled}
                                >
                                    <Trash2 className="size-3" />
                                    Delete {selectedIds.size}
                                </Button>
                            </AlertDialogTrigger>
                            <AlertDialogContent>
                                <AlertDialogHeader>
                                    <AlertDialogTitle>
                                        Delete {selectedIds.size} item
                                        {selectedIds.size !== 1 ? "s" : ""}
                                    </AlertDialogTitle>
                                    <AlertDialogDescription>
                                        This action cannot be undone. The selected items will be
                                        permanently removed.
                                    </AlertDialogDescription>
                                </AlertDialogHeader>
                                <AlertDialogFooter>
                                    <AlertDialogCancel>Cancel</AlertDialogCancel>
                                    <AlertDialogAction
                                        variant="destructive"
                                        onClick={handleDeleteConfirm}
                                    >
                                        Delete
                                    </AlertDialogAction>
                                </AlertDialogFooter>
                            </AlertDialogContent>
                        </AlertDialog>
                    )}

                    {addHref && (
                        <Button variant="outline" size="icon" disabled={isToolbarDisabled} asChild>
                            <Link href={addHref}>
                                <Plus className="size-3.5" />
                            </Link>
                        </Button>
                    )}
                </div>
            </div>

            {/* ── Body ──────────────────────────────────────────── */}
            {isInitialError ? (
                <div
                    className="text-destructive flex items-center justify-center text-xs"
                    style={{ height }}
                >
                    {error instanceof Error ? error.message : "Failed to load data."}
                </div>
            ) : isInitialPending ? (
                <SkeletonRows
                    rowHeight={rowHeight}
                    height={height}
                    columnCount={columns.length}
                    isCheckable={!!isCheckable}
                />
            ) : !hasData ? (
                <div
                    className="text-muted-foreground flex items-center justify-center text-xs"
                    style={{ height }}
                >
                    {isSearchOrFilterActive
                        ? (noResultsState ?? "No results found.")
                        : (emptyState ?? "No data.")}
                </div>
            ) : (
                <>
                    {/* ── Table header ──────────────────────────── */}
                    <div
                        className="text-muted-foreground grid items-center border-b text-[0.625rem] font-medium tracking-wide uppercase"
                        style={{
                            gridTemplateColumns,
                            paddingRight: 8,
                        }}
                    >
                        {isCheckable && (
                            <div className="flex items-center justify-center py-1.5">
                                <Checkbox
                                    checked={
                                        allSelected ? true : someSelected ? "indeterminate" : false
                                    }
                                    onCheckedChange={handleSelectAll}
                                />
                            </div>
                        )}
                        {columns.map((col) => (
                            <div
                                key={col.id}
                                className="truncate px-3 py-1.5"
                                style={{
                                    textAlign: col.align ?? "left",
                                }}
                            >
                                {col.header}
                            </div>
                        ))}
                    </div>

                    {/* ── Virtualized rows ─────────────────────── */}
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
                                const rowId = isLoaderRow
                                    ? null
                                    : String(getRowId(row, virtualRow.index));
                                const isDeleting = !!rowId && deletingIds.has(rowId);

                                return (
                                    <div
                                        key={isLoaderRow ? "loader" : rowId}
                                        data-index={virtualRow.index}
                                        ref={virtualizer.measureElement}
                                        className={cn(
                                            "hover:bg-muted/30 absolute top-0 left-0 grid w-full border-b text-xs/relaxed transition-colors",
                                            isDeleting && "opacity-50"
                                        )}
                                        style={{
                                            height: rowHeight,
                                            transform: `translateY(${virtualRow.start}px)`,
                                            gridTemplateColumns,
                                        }}
                                    >
                                        {isLoaderRow ? (
                                            <div className="text-muted-foreground col-span-full flex items-center justify-center py-2 text-[0.625rem]">
                                                Loading more...
                                            </div>
                                        ) : (
                                            <>
                                                {isCheckable && (
                                                    <div className="flex items-center justify-center">
                                                        <Checkbox
                                                            checked={selectedIds.has(rowId!)}
                                                            onCheckedChange={() =>
                                                                handleSelectRow(rowId!)
                                                            }
                                                            disabled={isDeleting}
                                                        />
                                                    </div>
                                                )}
                                                {columns.map((col) => (
                                                    <div
                                                        key={col.id}
                                                        className={cn(
                                                            "truncate px-3 py-2",
                                                            col.className
                                                        )}
                                                        style={{
                                                            textAlign: col.align ?? "left",
                                                        }}
                                                    >
                                                        {col.cell(row, virtualRow.index)}
                                                    </div>
                                                ))}
                                            </>
                                        )}
                                    </div>
                                );
                            })}
                        </div>
                    </div>

                    {/* ── Footer ───────────────────────────────── */}
                    {typeof total === "number" && (
                        <div className="text-muted-foreground px-1 text-[0.625rem]">
                            {rows.length} of {total} loaded
                        </div>
                    )}
                </>
            )}
        </div>
    );
}
