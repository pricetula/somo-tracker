"use client";

import { Checkbox } from "@/components/ui/checkbox";
import { HugeiconsIcon } from "@hugeicons/react";
import {
  ArrowUp01Icon,
  ArrowDown01Icon,
} from "@hugeicons/core-free-icons";
import type { ColumnDef, SortingState } from "@tanstack/react-table";

interface DataTableHeaderProps<T> {
  columns: ColumnDef<any, any, unknown>[];
  sorting: SortingState;
  setSorting: (next: SortingState) => void;
  enableRowSelection: boolean;
  someRowsSelected: boolean;
  allRowsSelected: boolean;
  onToggleSelectAll: (checked: boolean) => void;
  columnWidths: string[];
}

export function DataTableHeader<T>({
  columns,
  sorting,
  setSorting,
  enableRowSelection,
  someRowsSelected,
  allRowsSelected,
  onToggleSelectAll,
  columnWidths,
}: DataTableHeaderProps<T>) {
  const sortedId = sorting[0]?.id;
  const sortedDesc = sorting[0]?.desc;

  const handleSort = (id: string) => {
    const isSame = sortedId === id;
    const next: SortingState = isSame
      ? !sortedDesc
        ? [{ id, desc: true }]
        : []
      : [{ id, desc: false }];
    setSorting(next);
  };

  return (
    <div
      role="row"
      className="grid border-b border-background bg-muted/50 text-xs text-muted-foreground"
      style={{ gridTemplateColumns: columnWidths.join(" ") }}
    >
      {enableRowSelection && (
        <div
          role="columnheader"
          className="px-4 py-3 flex items-center gap-2"
        >
          <Checkbox
            checked={allRowsSelected}
            onCheckedChange={(checked) => onToggleSelectAll(checked)}
            aria-label={allRowsSelected ? "Select none" : "Select all"}
          />
        </div>
      )}
      {columns.map((column) => {
        const id = column.id as string;
        const isSorted = sortedId === id;
        return (
          <div
            key={id}
            role="columnheader"
            className="px-4 py-3 flex items-center gap-2 cursor-default select-none group"
            onClick={() => handleSort(id)}
            aria-sort={
              isSorted ? (sortedDesc ? "descending" : "ascending") : "none"
            }
          >
            <span className="truncate font-medium">{String(column.header)}</span>
            {isSorted && (
              <HugeiconsIcon
                icon={sortedDesc ? ArrowDown01Icon : ArrowUp01Icon}
                strokeWidth={2}
                className="size-3.5 ml-auto opacity-60 group-data-[sorted=true]:opacity-100"
                data-sorted={isSorted}
              />
            )}
          </div>
        );
      })}
    </div>
  );
}
