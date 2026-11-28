"use client";

import { Skeleton } from "@/components/ui/skeleton";
import { useCallback, type ReactNode } from "react";
import { Checkbox } from "@/components/ui/checkbox";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { MoreHorizontal } from "lucide-react";
import type { ColumnDef, RowData } from "@tanstack/react-table";

interface DataTableBodyProps<T extends RowData> {
  columns: ColumnDef<any, T, unknown>[];
  virtualRows: { index: number; start: number; size: number }[];
  allRows: T[];
  getRowId: (row: T) => string;
  enableRowSelection: boolean;
  virtualized: boolean;
  selectedIds: Set<string>;
  onToggleRowSelection: (rowId: string, checked: boolean) => void;
  renderRowActions?: (context: {
    row: T;
    rowId: string;
    refresh: () => void;
  }) => React.ReactNode;
  columnWidths: string[];
  selectionColumnWidth: string;
  actionsColumnWidth: string;
  refresh: () => void;
  onRowClick?: (row: T, rowId: string) => void;
  isLoadingMore: boolean;
}

function cellValue(column: any, row: any): ReactNode {
  const c = column as {
    accessorFn?: unknown;
    accessorKey?: unknown;
    id: string;
  };
  if (c.accessorFn) {
    return (c.accessorFn as (row: any) => unknown)(row) as ReactNode;
  }
  if (typeof c.accessorKey === "string") {
    return row[c.accessorKey] ?? "";
  }
  return row[c.id] ?? "";
}

export function DataTableBody<T extends RowData>({
  columns,
  virtualRows,
  allRows,
  getRowId,
  enableRowSelection,
  virtualized,
  selectedIds,
  onToggleRowSelection,
  renderRowActions,
  columnWidths,
  selectionColumnWidth,
  actionsColumnWidth,
  refresh,
  onRowClick,
  isLoadingMore,
}: DataTableBodyProps<T>) {
  const totalWidths = useCallback(() => columnWidths.join(" "), [columnWidths]);

  const renderRow = useCallback(
    (row: T, index: number, style?: React.CSSProperties) => {
      const rowId = getRowId(row);
      const checked = selectedIds.has(rowId);

      const handleRowClick = (e: React.MouseEvent) => {
        if (!onRowClick) return;
        const target = e.target as Element | null;
        // Do not trigger row navigation when interacting with checkboxes,
        // buttons, or the actions dropdown.
        if (target?.closest("button")) return;
        if (target?.closest('[role="checkbox"]')) return;
        onRowClick(row, rowId);
      };

      return (
        <div
          key={rowId}
          role="row"
          className="grid border-b border-border hover:bg-accent data-[selected=true]:bg-accent data-[selected=true]:text-accent-foreground transition-colors"
          style={{
            ...style,
            gridTemplateColumns: totalWidths(),
            width: "100%",
            minWidth: "max-content",
          }}
          data-selected={checked}
          onClick={handleRowClick}
        >
          {enableRowSelection && (
            <div
              role="cell"
              className="px-4 py-3 flex items-center sticky left-0 z-20 bg-background border-r border-border"
            >
              <Checkbox
                checked={checked}
                onClick={(e) => {
                  e.stopPropagation();
                  onToggleRowSelection(rowId, !checked);
                }}
                onCheckedChange={(checked) =>
                  onToggleRowSelection(rowId, checked)
                }
                aria-label={`Select row ${rowId}`}
              />
            </div>
          )}
          {columns.map((column) => {
            const id = column.id as string;
            return (
              <div
                key={id}
                role="cell"
                className="px-4 py-3 flex items-center truncate border-r border-border"
              >
                {typeof column.cell === "function"
                  ? ((column.cell as (ctx: any) => unknown)({
                      row: {
                        id: getRowId(row),
                        index,
                        original: row,
                      },
                    }) as ReactNode)
                  : cellValue(column, row)}
              </div>
            );
          })}
          {renderRowActions && (
            <div
              role="cell"
              className="px-3 py-3 flex items-center justify-end sticky right-0 z-20 bg-background border-l border-border"
            >
              <DropdownMenu>
                <DropdownMenuTrigger>
                  <MoreHorizontal size={16} />
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  {renderRowActions({
                    row,
                    rowId,
                    refresh,
                  })}
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          )}
        </div>
      );
    },
    [
      columns,
      enableRowSelection,
      getRowId,
      onToggleRowSelection,
      renderRowActions,
      refresh,
      selectedIds,
      totalWidths,
      onRowClick,
    ],
  );

  const renderRowsLoading = useCallback(() => {
    return Array.from({ length: 5 }, (_, i) => i + 1).map((index) => (
      <div
        key={index}
        role="row"
        className="grid border-b border-border hover:bg-accent data-[selected=true]:bg-accent data-[selected=true]:text-accent-foreground transition-colors"
        style={{
          gridTemplateColumns: totalWidths(),
          width: "100%",
          minWidth: "max-content",
        }}
      >
        {enableRowSelection && (
          <div
            role="cell"
            className="px-4 py-3 flex items-center sticky left-0 z-20 bg-background border-r border-border"
          >
              <Skeleton className="w-4 h-4" />
          </div>
        )}
        {columns.map((column) => {
          const id = column.id as string;
          return (
            <div
              key={id}
              role="cell"
              className="px-4 py-3 flex items-center truncate border-r border-border"
            >
              <Skeleton className="h-4 w-full" />
            </div>
          );
        })}
        {renderRowActions && (
          <div
            role="cell"
            className="px-3 py-3 flex items-center justify-end sticky right-0 z-20 bg-background border-l border-border"
          >
            <Skeleton className="w-6 h-6 rounded-full" />
          </div>
        )}
      </div>
    ));
  }, [columns, enableRowSelection, renderRowActions, totalWidths]);

  if (virtualized) {
    const lastVirtual = virtualRows[virtualRows.length - 1];
    const totalHeight = lastVirtual ? lastVirtual.start + lastVirtual.size : 0;
    return (
      <div
        style={{
          height: totalHeight,
          position: "relative",
          width: "100%",
          minWidth: "max-content",
        }}
      >
        {virtualRows.map((virtualRow) =>
          renderRow(allRows[virtualRow.index], virtualRow.index, {
            position: "absolute",
            top: virtualRow.start,
            left: 0,
            right: 0,
            height: virtualRow.size,
          }),
        )}
        {isLoadingMore && renderRowsLoading()}
      </div>
    );
  }

  return (
    <div
      style={{
        display: "grid",
        gridTemplateColumns: totalWidths(),
        width: "100%",
        minWidth: "max-content",
      }}
    >
      {allRows.map((row, index) => renderRow(row, index))}
      {isLoadingMore && renderRowsLoading()}
    </div>
  );
}
