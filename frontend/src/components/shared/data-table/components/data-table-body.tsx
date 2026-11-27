"use client";

import { useCallback, type ReactNode } from "react";
import { Checkbox } from "@/components/ui/checkbox";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { HugeiconsIcon } from "@hugeicons/react";
import { More01Icon } from "@hugeicons/core-free-icons";
import type { ColumnDef } from "@tanstack/react-table";

interface DataTableBodyProps<T> {
  columns: ColumnDef<any, any, unknown>[];
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
  refresh: () => void;
}

function cellValue(column: any, row: any): ReactNode {
  const c = column as { accessorFn?: unknown; accessorKey?: unknown; id: string };
  if (c.accessorFn) {
    return (c.accessorFn as (row: any) => unknown)(row) as ReactNode;
  }
  if (typeof c.accessorKey === "string") {
    return row[c.accessorKey] as ReactNode;
  }
  return row[c.id] as ReactNode;
}

export function DataTableBody<T>({
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
  refresh,
}: DataTableBodyProps<T>) {
  const totalWidths = useCallback(
    () => columnWidths.join(" "),
    [columnWidths],
  );

  const renderRow = useCallback(
    (row: T, index: number, style?: React.CSSProperties) => {
      const rowId = getRowId(row);
      const checked = selectedIds.has(rowId);

      return (
        <div
          key={rowId}
          role="row"
          className="grid border-b border-background/50 hover:bg-accent data-[selected=true]:bg-accent/50 transition-colors"
          style={{
            ...style,
            gridTemplateColumns: totalWidths(),
            width: "100%",
          }}
          data-selected={checked}
        >
          {enableRowSelection && (
            <div role="cell" className="px-4 py-3 flex items-center">
              <Checkbox
                checked={checked}
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
                className="px-4 py-3 flex items-center text-sm truncate"
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
            <div role="cell" className="px-3 py-3 flex items-center justify-end">
              <DropdownMenu>
                <DropdownMenuTrigger>
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    className="size-6"
                    onClick={(e) => {
                      e.stopPropagation();
                    }}
                  >
                    <HugeiconsIcon icon={More01Icon} strokeWidth={2} />
                  </Button>
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
    ],
  );

  if (virtualized) {
    const lastVirtual = virtualRows[virtualRows.length - 1];
    const totalHeight = lastVirtual ? lastVirtual.start + lastVirtual.size : 0;
    return (
      <div
        style={{
          height: totalHeight,
          position: "relative",
          width: "100%",
        }}
      >
        {virtualRows.map((virtualRow) =>
          renderRow(
            allRows[virtualRow.index],
            virtualRow.index,
            {
              position: "absolute",
              top: virtualRow.start,
              left: 0,
              right: 0,
              height: virtualRow.size,
            },
          ),
        )}
      </div>
    );
  }

  return (
    <div style={{ display: "grid", gridTemplateColumns: totalWidths(), width: "100%" }}>
      {allRows.map((row, index) => renderRow(row, index))}
    </div>
  );
}
