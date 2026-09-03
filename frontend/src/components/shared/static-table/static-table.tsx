"use client";

import { useCallback, useMemo, useRef, useState } from "react";
import type { ReactNode } from "react";

import type { DataTableColumn } from "@/components/shared/data-table/types";
import { Checkbox } from "@/components/ui/checkbox";
import { cn } from "@/lib/utils";

// ─── Props ────────────────────────────────────────────────────────────────

export interface StaticTableProps<TItem> {
    /** Column definitions — uses the same DataTableColumn type as DataTable. */
    columns: DataTableColumn<TItem>[];
    /** Static row data to render. */
    data: TItem[];
    /** Extracts a stable id from a row — used by selection and React keys. */
    getRowId: (row: TItem, index: number) => string | number;

    // ─── Checkbox / Selection ────────────────────────────────────────
    isCheckable?: boolean;
    /**
     * Controlled selected IDs. When provided, `onSelectionChange` is required
     * and selection is managed externally. When omitted, selection is managed
     * internally via `defaultSelectedIds`.
     */
    selectedIds?: Set<string>;
    /** Callback fired when selection changes. Required when `selectedIds` is controlled. */
    onSelectionChange?: (selectedIds: Set<string>) => void;
    /** Initial selected IDs when selection is uncontrolled. */
    defaultSelectedIds?: Set<string>;

    // ─── Sizing ──────────────────────────────────────────────────────
    /** Estimated row height in px. Defaults to 40. */
    rowHeight?: number;
    /** Height of the scrollable viewport in px. Defaults to 600. */
    height?: number;

    // ─── States ──────────────────────────────────────────────────────
    /** Shown when data is empty. */
    emptyState?: ReactNode;

    className?: string;
}

// ─── StaticTable ─────────────────────────────────────────────────────────

export function StaticTable<TItem>({
    columns,
    data,
    getRowId,
    isCheckable,
    selectedIds: controlledSelectedIds,
    onSelectionChange,
    defaultSelectedIds,
    rowHeight = 40,
    height = 600,
    emptyState,
    className,
}: StaticTableProps<TItem>) {
    // ── Selection state (controlled vs uncontrolled) ─────────────────
    const isControlled = controlledSelectedIds !== undefined;
    const [internalSelectedIds, setInternalSelectedIds] = useState<Set<string>>(
        defaultSelectedIds ?? new Set()
    );
    const selectedIds = isControlled ? controlledSelectedIds : internalSelectedIds;

    const setSelectedIds = useCallback(
        (ids: Set<string> | ((prev: Set<string>) => Set<string>)) => {
            if (isControlled) {
                const next = typeof ids === "function" ? ids(controlledSelectedIds!) : ids;
                onSelectionChange?.(next);
            } else {
                setInternalSelectedIds(ids);
            }
        },
        [isControlled, controlledSelectedIds, onSelectionChange]
    );

    // ── Grid template ────────────────────────────────────────────────
    const gridTemplateColumns = useMemo(() => {
        const cols: string[] = [];
        if (isCheckable) cols.push("36px");
        columns.forEach((col) => cols.push(col.width ?? "1fr"));
        return cols.join(" ");
    }, [isCheckable, columns]);

    // ── Event handlers ───────────────────────────────────────────────

    const handleSelectAll = useCallback(
        (checked: boolean) => {
            if (checked) {
                const ids = new Set(data.map((row, i) => String(getRowId(row, i))));
                setSelectedIds(ids);
            } else {
                setSelectedIds(new Set());
            }
        },
        [data, getRowId, setSelectedIds]
    );

    const handleSelectRow = useCallback(
        (id: string) => {
            setSelectedIds((prev) => {
                const next = new Set(prev);
                if (next.has(id)) {
                    next.delete(id);
                } else {
                    next.add(id);
                }
                return next;
            });
        },
        [setSelectedIds]
    );

    // ── Determine if "Select all" is checked / indeterminate ─────────
    const allSelected =
        data.length > 0 && data.every((row, i) => selectedIds.has(String(getRowId(row, i))));
    const someSelected =
        data.some((row, i) => selectedIds.has(String(getRowId(row, i)))) && !allSelected;

    // ── Scroll ref for auto-scrolling ────────────────────────────────
    const scrollRef = useRef<HTMLDivElement>(null);

    // ── Render ───────────────────────────────────────────────────────

    return (
        <div className={cn("flex flex-col", className)}>
            {/* overflow-x-auto enables horizontal scroll on small screens;
                min-w-max prevents the grid from shrinking below its content width */}
            <div className="overflow-x-auto rounded-md border">
                <div className="min-w-max">
                    {/* ── Table header ──────────────────────────── */}
                    <div
                        className="text-muted-foreground bg-accent/50 grid items-center border-b text-[0.625rem] font-medium tracking-wide uppercase"
                        style={{ gridTemplateColumns, height: rowHeight }}
                    >
                        {isCheckable && (
                            <div className="flex items-center justify-center">
                                <Checkbox
                                    checked={allSelected}
                                    indeterminate={someSelected && !allSelected}
                                    onCheckedChange={handleSelectAll}
                                />
                            </div>
                        )}
                        {columns.map((col) => {
                            const borderClass =
                                !isCheckable && col.id === columns[0].id ? "" : "border-l";
                            return (
                                <div
                                    key={col.id}
                                    className={cn(
                                        "flex h-full items-center truncate px-3",
                                        borderClass
                                    )}
                                    style={{ textAlign: col.align ?? "left" }}
                                >
                                    {col.header}
                                </div>
                            );
                        })}
                    </div>

                    {data.length === 0 ? (
                        /* ── Empty state ────────────────────── */
                        <div
                            className="text-muted-foreground flex items-center justify-center text-xs"
                            style={{ height }}
                        >
                            {emptyState ?? "No data."}
                        </div>
                    ) : (
                        /* ── Scrollable rows (vertical) ─────── */
                        <div ref={scrollRef} style={{ height, overflow: "auto" }}>
                            {data.map((row, index) => {
                                const rowId = String(getRowId(row, index));
                                const isChecked = selectedIds.has(rowId);

                                return (
                                    <div
                                        key={rowId}
                                        className={cn(
                                            "hover:bg-muted/30 grid w-full border-b text-xs/relaxed transition-colors",
                                            isChecked && "bg-muted/20"
                                        )}
                                        style={{
                                            height: rowHeight,
                                            gridTemplateColumns,
                                        }}
                                    >
                                        {isCheckable && (
                                            <div className="flex items-center justify-center">
                                                <Checkbox
                                                    checked={isChecked}
                                                    onCheckedChange={() => handleSelectRow(rowId)}
                                                />
                                            </div>
                                        )}
                                        {columns.map((col) => {
                                            const borderClass =
                                                !isCheckable && col.id === columns[0].id
                                                    ? ""
                                                    : "border-l";
                                            return (
                                                <div
                                                    key={col.id}
                                                    className={cn(
                                                        "flex items-center truncate px-3",
                                                        col.className,
                                                        borderClass
                                                    )}
                                                    style={{ textAlign: col.align ?? "left" }}
                                                >
                                                    {col.cell(row, index)}
                                                </div>
                                            );
                                        })}
                                    </div>
                                );
                            })}
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
}
