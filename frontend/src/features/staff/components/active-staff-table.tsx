/**
 * Active Staff Table — displays active staff users by role.
 *
 * Uses TanStack Table + TanStack Virtual for performance.
 * Each row shows full_name, full_name, email, created_at
 * with Edit and Deactivate action stubs.
 */

"use client";

import * as React from "react";
import { useReactTable, getCoreRowModel, flexRender, type ColumnDef } from "@tanstack/react-table";
import { useVirtualizer } from "@tanstack/react-virtual";

import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { MoreHorizontal, UserPlus } from "lucide-react";
import Link from "next/link";

import type { Member } from "@/lib/api/members";
// Re-export Member as User for backward compatibility
type User = Member;

// ─── Helpers ───────────────────────────────────────────────────────────────

function formatDate(iso: string): string {
    return new Date(iso).toLocaleDateString("en-US", {
        month: "short",
        day: "numeric",
        year: "numeric",
    });
}

// ─── Columns ───────────────────────────────────────────────────────────────

function createColumns(): ColumnDef<User>[] {
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
            accessorKey: "created_at",
            header: "Joined",
            cell: ({ row }) => (
                <span className="text-muted-foreground text-sm">
                    {formatDate(row.original.created_at)}
                </span>
            ),
        },
        {
            id: "actions",
            header: "",
            cell: () => (
                <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                        <Button
                            variant="ghost"
                            size="icon-sm"
                            className="opacity-0 transition-opacity group-hover:opacity-100"
                        >
                            <MoreHorizontal className="size-4" />
                            <span className="sr-only">Actions</span>
                        </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end" className="w-36">
                        {/* TODO: Implement Edit profile action */}
                        <DropdownMenuItem>Edit</DropdownMenuItem>
                        {/* TODO: Implement Deactivate user action */}
                        <DropdownMenuItem>Deactivate</DropdownMenuItem>
                    </DropdownMenuContent>
                </DropdownMenu>
            ),
        },
    ];
}

// ─── Props ─────────────────────────────────────────────────────────────────

interface ActiveStaffTableProps {
    users: User[];
    total: number;
    roleLabel: string;
    addHref: string;
    isLoading: boolean;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function ActiveStaffTable({
    users,
    total,
    roleLabel,
    addHref,
    isLoading,
}: ActiveStaffTableProps) {
    const columns = React.useMemo(() => createColumns(), []);

    // eslint-disable-next-line react-hooks/incompatible-library
    const table = useReactTable({
        data: users,
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

    const skeletonRows = 10;

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
                <div className="min-w-150">
                    {/* Sticky Header */}
                    <div className="bg-background/95 sticky top-0 z-10 rounded-lg backdrop-blur-sm">
                        {table.getHeaderGroups().map((headerGroup) => (
                            <div key={headerGroup.id} className="border-border/40 flex border-b">
                                {headerGroup.headers.map((header) => (
                                    <div
                                        key={header.id}
                                        className="text-muted-foreground flex h-10 items-center px-3 text-xs font-medium tracking-wider uppercase"
                                        style={{
                                            width:
                                                header.getSize() ||
                                                (header.id === "actions" ? 48 : "auto"),
                                            flex: header.id !== "actions" ? 1 : "0 0 auto",
                                        }}
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
                                    <Skeleton className="mr-3 h-4 w-20" />
                                    <Skeleton className="mr-3 h-4 w-20" />
                                    <Skeleton className="mr-3 h-4 w-36" />
                                    <Skeleton className="mr-3 h-4 w-24" />
                                </div>
                            ))
                        ) : rows.length === 0 ? (
                            <div className="flex items-center justify-center py-16">
                                <div className="text-center">
                                    <p className="text-muted-foreground text-sm font-medium">
                                        No active {roleLabel.toLowerCase()} yet
                                    </p>
                                    <p className="text-muted-foreground mt-1 text-xs">
                                        Invite {roleLabel.toLowerCase()} to get started.
                                    </p>
                                    <div className="mt-4">
                                        <Button variant="secondary" size="sm" asChild>
                                            <Link href={addHref}>
                                                <UserPlus className="mr-1.5 size-3.5" />
                                                Invite {roleLabel}
                                            </Link>
                                        </Button>
                                    </div>
                                </div>
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
                                                className={
                                                    "flex items-center truncate px-3 text-sm" +
                                                    (cell.column.id === "actions"
                                                        ? " justify-end"
                                                        : "")
                                                }
                                                style={{
                                                    width:
                                                        cell.column.getSize() ||
                                                        (cell.column.id === "actions"
                                                            ? 48
                                                            : "auto"),
                                                    flex:
                                                        cell.column.id !== "actions"
                                                            ? 1
                                                            : "0 0 auto",
                                                }}
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
                    {total} active {roleLabel.toLowerCase()}
                    {total !== 1 ? "" : ""}
                </p>
            </div>
        </div>
    );
}
