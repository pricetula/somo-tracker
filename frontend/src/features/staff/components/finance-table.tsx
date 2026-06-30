/**
 * Finance Staff Listing Table — displays finance staff users.
 *
 * Columns: Full Name, Email, Account Status (toggle).
 * Uses TanStack Table + TanStack Virtual for performance.
 */

"use client";

import * as React from "react";
import { useReactTable, getCoreRowModel, flexRender, type ColumnDef } from "@tanstack/react-table";
import { useVirtualizer } from "@tanstack/react-virtual";

import { Switch } from "@/components/ui/switch";
import { Skeleton } from "@/components/ui/skeleton";

import type { Member } from "@/lib/api/finance";
import { useToggleFinanceActive } from "../hooks/use-finance";

// ─── Columns ───────────────────────────────────────────────────────────────

function createColumns(): ColumnDef<Member>[] {
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
            cell: ({ row }) => <StatusToggleCell member={row.original} />,
        },
    ];
}

// ─── Status Toggle Cell ────────────────────────────────────────────────────

function StatusToggleCell({ member }: { member: Member }) {
    const toggleMutation = useToggleFinanceActive();

    const handleToggle = (checked: boolean) => {
        toggleMutation.mutate({ userId: member.id, isActive: checked });
    };

    return (
        <div className="flex items-center gap-2">
            <Switch
                checked={member.is_active}
                onCheckedChange={handleToggle}
                disabled={toggleMutation.isPending}
                aria-label={
                    member.is_active ? "Deactivate finance staff" : "Activate finance staff"
                }
            />
            <span
                className={
                    member.is_active
                        ? "text-xs font-medium text-emerald-600"
                        : "text-muted-foreground text-xs"
                }
            >
                {member.is_active ? "Active" : "Inactive"}
            </span>
        </div>
    );
}

// ─── Props ─────────────────────────────────────────────────────────────────

interface FinanceTableProps {
    staff: Member[];
    total: number;
    isLoading: boolean;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function FinanceTable({ staff, total, isLoading }: FinanceTableProps) {
    const columns = React.useMemo(() => createColumns(), []);

    // eslint-disable-next-line react-hooks/incompatible-library
    const table = useReactTable({
        data: staff,
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
                style={{ contain: "strict", minHeight: rows.length === 0 ? "200px" : undefined }}
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
                                    No finance staff yet
                                </p>
                            </div>
                        ) : (
                            virtualizer.getVirtualItems().map((virtualRow) => {
                                const row = rows[virtualRow.index];
                                return (
                                    <div
                                        key={row.id}
                                        className="group border-border/40 hover:bg-muted/30 absolute right-0 left-0 flex h-12 items-center border-b transition-colors"
                                        style={{
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
                <p className="text-muted-foreground text-xs">{total} finance staff</p>
            </div>
        </div>
    );
}
