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
import { DataTableSkeleton } from "./data-table-skeleton";
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
  onRowClick,
}: DataTableProps<T>) {
  const {
    query,
    allRows,
    totalRowCount,
    isInitialLoading,
    isLoadingMore,
    error,
    hasMorePages,
    loadMore,
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
    actionsColumnWidth,
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
    renderRowActions,
    onRowClick,
  });

  const sentinelRef = useRef<HTMLDivElement | null>(null);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [deleteIds, setDeleteIds] = useState<string[]>([]);
  const deleteDialogOpenRef = useRef(deleteDialogOpen);
  const toast = useToastManager();

  // Kept for backwards-compat with any external scroll observers; the primary
  // pagination trigger is now the explicit "Load more" button.
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
          loadMore();
        }
      },
      { threshold: 0.5 },
    );
    observer.observe(el);
    return () => observer.disconnect();
  }, [hasMorePages, isLoadingMore, query.isFetching, loadMore]);

  const handleToggleSelectAll = useCallback(
    (checked: boolean) => {
      toggleSelectAll(checked);
    },
    [toggleSelectAll],
  );

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
    <div className="relative flex flex-col h-full border border-background rounded-md bg-background">
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
        className="flex-1 overflow-auto min-h-0 border border-border rounded-b-md"
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
          actionsColumnWidth={actionsColumnWidth}
          renderRowActions={!!renderRowActions}
        />
        {isLoadingMore ? (
          <DataTableSkeleton
            columnCount={
              columns.length +
              (enableRowSelection ? 1 : 0) +
              (renderRowActions ? 1 : 0)
            }
            enableRowSelection={enableRowSelection}
            renderRowActions={!!renderRowActions}
          />
        ) : (
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
            actionsColumnWidth={actionsColumnWidth}
            refresh={refresh}
            onRowClick={onRowClick}
          />
        )}
        {hasMorePages && (
          <div className="py-4 text-center border-t border-background/50">
            {isLoadingMore ? (
              <div className="inline-flex items-center gap-2 text-muted-foreground animate-pulse">
                Loading more…
              </div>
            ) : (
              <button
                onClick={loadMore}
                disabled={isLoadingMore}
                className="inline-flex items-center gap-2 border border-input rounded-md bg-background px-4 py-2 font-medium hover:bg-accent transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Load more
              </button>
            )}
          </div>
        )}
      </div>

      {enableRowSelection && someRowsSelected && (
        <DataTableSelectionSheet
          open={true}
          onDelete={handleDelete}
          onClearSelection={() => toggleSelectAll(false)}
          onClose={() => toggleSelectAll(false)}
        />
      )}

      <DataTableDeleteDialog
        open={deleteDialogOpen}
        onOpenChange={setDeleteDialogOpen}
        ids={deleteIds}
        onConfirm={handleConfirmDelete}
        deleting={isDeleting}
      />
    </div>
  );
}
