"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  useTable,
  coreColumnsFeature,
  coreRowsFeature,
  coreRowModelsFeature,
  rowSelectionFeature,
  type ColumnDef,
  type RowData,
} from "@tanstack/react-table";
import { useVirtualizer } from "@tanstack/react-virtual";

import { useDataTableUrlState } from "./use-data-table-url-state";
import type {
  BulkDeleteResult,
  DataTablePage,
  DataTableProps,
} from "../types";

export function useDataTable<T extends RowData>(props: DataTableProps<T>) {
  const {
    queryKey,
    columns,
    fetchPage,
    getRowId,
    pageSize = 50,
    initialFilters = {},
    initialSorting = [],
    enableRowSelection = false,
    enableVirtualization = false,
    onDelete,
  } = props;

  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const scrollRef = useRef<HTMLDivElement | null>(null);
  const isMountedRef = useRef(false);

  const {
    search,
    setSearch,
    sorting,
    setSorting,
    filters,
    setFilters,
  } = useDataTableUrlState(initialFilters, initialSorting);

  const query = useInfiniteQuery<DataTablePage<T>, Error>({
    queryKey: [
      ...queryKey,
      "list",
      search,
      JSON.stringify(sorting),
      JSON.stringify(filters),
    ],
    queryFn: ({ pageParam = null }) =>
      fetchPage({
        cursor: pageParam as string | null,
        limit: pageSize,
        search,
        sorting,
        filters,
      }),
    getNextPageParam: (lastPage) =>
      lastPage.hasMore && lastPage.nextCursor ? lastPage.nextCursor : undefined,
    initialPageParam: null,
  });

  const queryClient = useQueryClient();

  const allRows = useMemo(() => {
    const pages = query.data?.pages ?? [];
    return pages.flatMap((page) => page.data);
  }, [query.data?.pages]);

  const pages = query.data?.pages ?? [];
  const isInitialLoading = query.isFetching && pages.length <= 1;
  const isLoadingMore = query.isFetching && pages.length > 1;
  const error = query.error;

  // Clear selection when search or filters change (scope changes), but preserve
  // it when only sorting changes.
  const filtersKey = JSON.stringify(filters);
  useEffect(() => {
    if (!isMountedRef.current) {
      isMountedRef.current = true;
      return;
    }
    setSelectedIds(new Set());
  }, [search, filtersKey]);

  const totalRowCount = allRows.length;
  const selectedCount = selectedIds.size;

  // eslint-disable-next-line react-hooks/incompatible-library
  const virtualizer = useVirtualizer({
    count: totalRowCount,
    getScrollElement: () => scrollRef.current,
    estimateSize: () => (enableVirtualization ? 44 : 0),
    overscan: 10,
  });

  const virtualRows = enableVirtualization
    ? virtualizer.getVirtualItems()
    : [];

  // TanStack Table v9 instance for row selection state. Sorting, filtering and
  // pagination are server-driven and managed externally via the URL state hook.
  const table = useTable(
    {
      features: {
        coreColumnsFeature,
        coreRowsFeature,
        coreRowModelsFeature,
        rowSelectionFeature,
      },
      columns: columns as ColumnDef<any, T, unknown>[],
      data: allRows as T[],
      getRowId,
      enableRowSelection,
    } as unknown as Parameters<typeof useTable>[0],
    (state: any) => ({
      rowSelection: state.rowSelection,
    }),
  ) as any;

  const isRowSelectedViaTable = useCallback(
    (rowId: string) =>
      enableRowSelection ? table.getIsRowSelected(rowId) : false,
    [enableRowSelection, table],
  );

  // Deletion lifecycle owned by the shared component; the resource supplies the
  // actual deletion call. Partial success is supported.
  const deleteMutation = useMutation<
    BulkDeleteResult,
    Error,
    string[],
    { previous: { pages: DataTablePage<T>[] } | undefined }
  >({
    mutationFn: async (ids) => {
      if (!onDelete) {
        throw new Error("No deletion handler configured");
      }
      return onDelete(ids);
    },
    onMutate: async (ids) => {
      await queryClient.cancelQueries({ queryKey });
      const previous = queryClient.getQueryData(queryKey) as
        | { pages: DataTablePage<T>[] }
        | undefined;
      return { previous };
    },
    onSuccess: (result, ids) => {
      // Successfully deleted IDs leave selection; failed IDs stay selected.
      const deleted = new Set(result.deletedIds);
      setSelectedIds((prev) => {
        const next = new Set(prev);
        for (const id of deleted) next.delete(id);
        return next;
      });
      queryClient.invalidateQueries({ queryKey });
    },
    onError: (_error, _ids, context) => {
      queryClient.setQueryData(queryKey, context?.previous);
    },
  });

  const isDeleting = deleteMutation.isPending;

  const toggleRowSelection = useCallback(
    (rowId: string, checked: boolean) => {
      if (enableRowSelection) {
        table.toggleRowSelected(rowId, checked);
      }
      setSelectedIds((prev) => {
        const next = new Set(prev);
        if (checked) next.add(rowId);
        else next.delete(rowId);
        return next;
      });
    },
    [table, enableRowSelection],
  );

  const toggleSelectAll = useCallback(
    (checked: boolean) => {
      if (enableRowSelection) {
        table.toggleSelectAll(checked);
        const selected = table.getSelectedRowModel().map((row: any) => row.id);
        setSelectedIds(new Set(selected));
      } else if (checked) {
        setSelectedIds(new Set(allRows.map(getRowId)));
      } else {
        setSelectedIds(new Set());
      }
    },
    [table, enableRowSelection, allRows, getRowId],
  );

  const isRowSelected = useCallback(
    (rowId: string) => selectedIds.has(rowId) || isRowSelectedViaTable(rowId),
    [selectedIds, isRowSelectedViaTable],
  );

  const someRowsSelected = selectedCount > 0;
  const allRowsSelected =
    totalRowCount > 0 && selectedCount === totalRowCount;

  const refresh = useCallback(() => {
    queryClient.invalidateQueries({ queryKey });
  }, [queryClient, queryKey]);

  const fetchNextPage = useCallback(() => {
    query.fetchNextPage();
  }, [query]);

  const hasMorePages = !!(
    pages.length > 0 &&
    pages[pages.length - 1].hasMore &&
    pages[pages.length - 1].nextCursor
  );

  const columnWidths = useMemo(
    () =>
      columns.map((column) => {
        const meta = (column as { meta?: { width?: string } }).meta;
        return meta?.width ?? "1fr";
      }),
    [columns],
  );

  const selectionColumnWidth = "48px";
  const allColumnWidths = useMemo(
    () =>
      enableRowSelection
        ? [selectionColumnWidth, ...columnWidths]
        : columnWidths,
    [enableRowSelection, columnWidths, selectionColumnWidth],
  );

  return {
    // Query / data
    query,
    pages,
    allRows,
    totalRowCount,
    isInitialLoading,
    isLoadingMore,
    error,
    hasMorePages,
    fetchNextPage,
    // URL state
    search,
    setSearch,
    sorting,
    setSorting,
    filters,
    setFilters,
    pageSize,
    // Selection
    selectedIds,
    selectedCount,
    someRowsSelected,
    allRowsSelected,
    isRowSelected,
    toggleRowSelection,
    toggleSelectAll,
    // Virtualization
    virtualizer,
    virtualRows,
    scrollRef,
    enableVirtualization,
    // Deletion
    deleteMutation,
    isDeleting,
    refresh,
    // Layout
    allColumnWidths,
    selectionColumnWidth,
  };
}
