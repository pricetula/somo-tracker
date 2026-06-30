/**
 * Nurses Listing Table — displays nurse users.
 *
 * Columns: Full Name, Email, Account Status (toggle).
 * Uses TanStack Table + TanStack Virtual for performance.
 */

"use client";

import * as React from "react";
import { useReactTable, getCoreRowModel, flexRender, type ColumnDef } from "@tanstack/react-table";
import { useVirtualizer } from "@tanstack/react-virtual";

import { Skeleton } from "@/components/ui/skeleton";

import { StatusToggleCell } from "./status-toggle-cell";
import type { Member } from "@/lib/api/nurses";
import { useToggleNurseActive } from "../hooks/use-nurses";

// ─── Columns ───────────────────────────────────────────────────────────────

function createColumns(
    toggleMutation: ReturnType<typeof useToggleNurseActive>
): ColumnDef<Member>[] {
    return [
        {
            accessorKey: "full_name",
            header: "Full Name",
            cell: ({ row }) => (
                <span className="text-sm font-medium">{row.original.full_name || "—"}</span>
            ),
        },
        {
            accessorKey: "email",
            header: "Email",
            cell: ({ row }) => (
                <span className="text-muted-foreground text-sm">{row.original.email}</span>
            ),
        },
        {
            id: "is_active",
            header: "Account Status",
            cell: ({ row }) => (
                <StatusToggleCell
                    member={row.original}
                    onToggle={(userId, isActive) => toggleMutation.mutate({ userId, isActive })}
                    isPending={toggleMutation.isPending}
                    label={{
                        activate: "Activate nurse",
                        deactivate: "Deactivate nurse",
                    }}
                />
            ),
        },
    ];
}

// ─── Props ─────────────────────────────────────────────────────────────────

interface NursesTableProps {
    nurses: Member[];
    total: number;
    isLoading: boolean;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function NursesTable({ nurses, total, isLoading }: NursesTableProps) {
    const toggleMutation = useToggleNurseActive();
    const columns = React.useMemo(() => createColumns(toggleMutation), [toggleMutation]);

    // eslint-disable-next-line react-hooks/incompatible-library
    const table = useReactTable({
        data: nurses,
        columns,
        getCoreRowModel: getCoreRowModel(),
    });

    const parentRef = React.useRef<HTMLDivElement>(null);
    const rows = table.getRowModel().rows;

    const virtualizer = useVirtualizer({
        count: rows.length,
        getScrollElement: () => parentRef.current,
        estimateSize: () => 48,
        overscan: 10,
    });

    const skeletonRows = 8;

    return (
        <div className="flex flex-1 flex-col">
            {/* Table Canvas */}
            <div
                ref={parentRef}
                className="flex-1 overflow-auto"
                style={{
                    contain: "layout paint",
                    minHeight: rows.length === 0 ? "200px" : undefined,
                }}
            >
                <div className="min-w-175">
                    {/* Sticky Header */}
                    <div className="bg-background/95 sticky top-0 z-10 rounded-lg backdrop-blur-sm">
                        {table.getHeaderGroups().map((headerGroup) => (
                            <div key={headerGroup.id} className="border-border/40 flex border-b">
                                {headerGroup.headers.map((header) => (
                                    <div
                                        key={header.id}
                                        className="text-muted-foreground flex h-10 items-center px-3 text-xs font-medium tracking-wider uppercase"
                                        style={{ flex: 1 }}
                                    >
                                        {flexRender(
                                            header.column.columnDef.header,
                                            header.getContext()
                                        )}
                                    </div>
                                ))}
                            </div>
                        ))}
                    </div>

                    {/* Virtualized Body */}
                    <div
                        style={{
                            height: `${virtualizer.getTotalSize()}px`,
                            position: "relative",
                        }}
                    >
                        {isLoading && rows.length === 0 ? (
                            Array.from({ length: skeletonRows }).map((_, i) => (
                                <div
                                    key={`skeleton-${i}`}
                                    className="border-border/40 flex h-12 items-center border-b px-3"
                                >
                                    <Skeleton className="mr-3 h-4 w-20 flex-1" />
                                    <Skeleton className="mr-3 h-4 w-20 flex-1" />
                                    <Skeleton className="mr-3 h-6 w-16 flex-1" />
                                </div>
                            ))
                        ) : rows.length === 0 ? (
                            <div className="flex items-center justify-center py-16">
                                <p className="text-muted-foreground text-sm font-medium">
                                    No nurses yet
                                </p>
                            </div>
                        ) : (
                            virtualizer.getVirtualItems().map((virtualRow) => {
                                const row = rows[virtualRow.index];
                                return (
                                    <div
                                        key={virtualRow.key}
                                        className="group border-border/40 hover:bg-muted/30 absolute right-0 left-0 flex items-center border-b transition-colors"
                                        style={{
                                            position: "absolute",
                                            top: 0,
                                            left: 0,
                                            width: "100%",
                                            height: `${virtualRow.size}px`,
                                            transform: `translateY(${virtualRow.start}px)`,
                                        }}
                                    >
                                        {row.getVisibleCells().map((cell) => (
                                            <div
                                                key={cell.id}
                                                className="flex items-center truncate px-3 text-sm"
                                                style={{ flex: 1 }}
                                            >
                                                {flexRender(
                                                    cell.column.columnDef.cell,
                                                    cell.getContext()
                                                )}
                                            </div>
                                        ))}
                                    </div>
                                );
                            })
                        )}
                    </div>
                </div>
            </div>

            {/* Footer counter */}
            <div className="border-border/40 flex items-center justify-between border-t px-3 py-2">
                <p className="text-muted-foreground text-xs">
                    {total} nurse{total !== 1 ? "s" : ""}
                </p>
            </div>
        </div>
    );
}
