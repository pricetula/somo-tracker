"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useToastManager } from "@/components/ui/toast";

import { useDataTable } from "../hooks/use-data-table";
import { DataTableToolbar } from "./data-table-toolbar";
import { DataTableHeader } from "./data-table-header";
import { DataTableBody } from "./data-table-body";
import { DataTableLoading } from "./data-table-loading";
import { DataTableEmpty } from "./data-table-empty";
import { DataTableError } from "./data-table-error";
import { DataTableSelectionSheet } from "./data-table-selection-sheet";
import { DataTableDeleteDialog } from "./data-table-delete-dialog";
import type { DataTableProps } from "../types";
import type { RowData } from "@tanstack/react-table";

export function DataTable<T extends RowData>({
  queryKey,
  columns,
  fetchPage,
  getRowId,
  pageSize,
  initialFilters,
  initialSorting,
  filters: filterConfig = [],
  renderBulkActions,
  renderRowActions,
  onDelete,
  enableRowSelection = false,
  enableVirtualization = true,
}: DataTableProps<T>) {
  const {
    query,
    allRows,
    totalRowCount,
    isInitialLoading,
    isLoadingMore,
    error,
    hasMorePages,
    fetchNextPage,
    search,
    setSearch,
    sorting,
    setSorting,
    filters,
    setFilters,
    selectedIds,
    selectedCount,
    someRowsSelected,
    allRowsSelected,
    toggleRowSelection,
    toggleSelectAll,
    virtualizer,
    virtualRows,
    scrollRef,
    deleteMutation,
    isDeleting,
    refresh,
    allColumnWidths,
    selectionColumnWidth,
  } = useDataTable<T>({
    queryKey,
    columns,
    fetchPage,
    getRowId,
    pageSize,
    initialFilters,
    initialSorting,
    enableRowSelection,
    enableVirtualization,
    onDelete,
  });

  const sentinelRef = useRef<HTMLDivElement | null>(null);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [deleteIds, setDeleteIds] = useState<string[]>([]);
  const deleteDialogOpenRef = useRef(deleteDialogOpen);
  const toast = useToastManager();

  // Infinite scroll: fetch next page when the sentinel reaches the viewport.
  useEffect(() => {
    const el = sentinelRef.current;
    if (!el) return undefined;
    const observer = new IntersectionObserver(
      (entries) => {
        if (
          entries[0].isIntersecting &&
          hasMorePages &&
          !isLoadingMore &&
          !query.isFetching
        ) {
          fetchNextPage();
        }
      },
      { threshold: 0.5 },
    );
    observer.observe(el);
    return () => observer.disconnect();
  }, [hasMorePages, isLoadingMore, query.isFetching, fetchNextPage]);

  const handleToggleSelectAll = useCallback((checked: boolean) => {
    toggleSelectAll(checked);
  }, [toggleSelectAll]);

  const handleDelete = useCallback(() => {
    setDeleteIds(Array.from(selectedIds));
    setDeleteDialogOpen(true);
  }, [selectedIds]);

  const handleConfirmDelete = useCallback(
    (ids: string[]) => {
      if (ids.length === 0 || !onDelete) return;
      deleteMutation.mutate(ids);
    },
    [onDelete, deleteMutation],
  );

  // Deletion outcome toasts.
  useEffect(() => {
    if (!deleteMutation.isSuccess) return;
    const { deletedIds, failed } = deleteMutation.data ?? {
      deletedIds: [],
      failed: [],
    };
    if (deletedIds.length) {
      toast.add({
        title: "Deleted",
        description: `${deletedIds.length} record(s) deleted.`,
        type: "success",
      });
    }
    if (failed.length) {
      toast.add({
        title: "Some deletions failed",
        description: `${failed.length} record(s) could not be deleted and remain selected.`,
        type: "error",
      });
    }
  }, [deleteMutation, toast]);

  useEffect(() => {
    if (deleteMutation.isError) {
      toast.add({
        title: "Deletion failed",
        description: "Failed to delete records.",
        type: "error",
      });
    }
  }, [deleteMutation.isError, toast]);

  // Close the confirmation dialog once a deletion completes.
  useEffect(() => {
    if (deleteMutation.isSuccess && deleteDialogOpenRef.current) {
      deleteDialogOpenRef.current = false;
      setDeleteDialogOpen(false);
      setDeleteIds([]);
    }
    if (!deleteMutation.isSuccess && !deleteMutation.isPending) {
      deleteDialogOpenRef.current = deleteDialogOpen;
    }
  }, [deleteMutation.isSuccess, deleteMutation.isPending, deleteDialogOpen]);

  if (isInitialLoading) {
    return <DataTableLoading />;
  }

  if (error) {
    return <DataTableError error={error} retry={query.refetch} />;
  }

  if (allRows.length === 0) {
    return <DataTableEmpty />;
  }

  return (
    <div className="flex flex-col h-full border border-background rounded-md bg-background">
      <DataTableToolbar
        search={search}
        setSearch={setSearch}
        filters={filterConfig}
        filterValues={filters}
        onFilterChange={setFilters}
        isLoading={isLoadingMore}
      />
      <div
        ref={(el) => {
          scrollRef.current = el;
        }}
        className="flex-1 overflow-auto min-h-0"
        style={{ maxHeight: "calc(100dvh - 16rem)" }}
      >
        <DataTableHeader
          columns={columns as any}
          sorting={sorting}
          setSorting={setSorting}
          enableRowSelection={enableRowSelection}
          someRowsSelected={someRowsSelected}
          allRowsSelected={allRowsSelected}
          onToggleSelectAll={handleToggleSelectAll}
          columnWidths={allColumnWidths}
          selectionColumnWidth={selectionColumnWidth}
        />
        <DataTableBody
          columns={columns as any}
          virtualRows={virtualRows}
          allRows={allRows}
          getRowId={getRowId}
          enableRowSelection={enableRowSelection}
          virtualized={enableVirtualization}
          selectedIds={selectedIds}
          onToggleRowSelection={toggleRowSelection}
          renderRowActions={renderRowActions}
          columnWidths={allColumnWidths}
          selectionColumnWidth={selectionColumnWidth}
          refresh={refresh}
        />
        {hasMorePages && (
          <div
            ref={sentinelRef}
            className="py-4 text-xs text-muted-foreground text-center"
          >
            {isLoadingMore ? "Loading more…" : "Scroll to load more"}
          </div>
        )}
      </div>

      {enableRowSelection && someRowsSelected && (
        <DataTableSelectionSheet
          open={true}
          selectedCount={selectedCount}
          onClearSelection={() => toggleSelectAll(false)}
          onClose={() => {}}
        >
          {onDelete && (
            <button
              onClick={handleDelete}
              className="inline-flex items-center gap-2 border border-transparent rounded-md bg-destructive/10 text-destructive hover:bg-destructive/20 text-xs font-medium px-3 py-1.5"
            >
              Delete selected
            </button>
          )}
          {renderBulkActions && (
            renderBulkActions({
              selectedIds: Array.from(selectedIds),
              selectedCount,
              clearSelection: () => toggleSelectAll(false),
              refresh,
            })
          )}
        </DataTableSelectionSheet>
      )}

      <DataTableDeleteDialog
        open={deleteDialogOpen}
        onOpenChange={setDeleteDialogOpen}
        selectedCount={deleteIds.length}
        ids={deleteIds}
        onConfirm={handleConfirmDelete}
        deleting={isDeleting}
      />
    </div>
  );
}
