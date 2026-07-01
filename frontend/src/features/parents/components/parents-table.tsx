/**
 * Parents Table — displays parent/guardian profiles.
 *
 * Columns: Full Name, Email, Phone, Linked Students count, Active status.
 * Supports searching and navigating to detail.
 */

"use client";

import * as React from "react";
import Link from "next/link";
import { useReactTable, getCoreRowModel, flexRender, type ColumnDef } from "@tanstack/react-table";
import { useVirtualizer } from "@tanstack/react-virtual";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Input } from "@/components/ui/input";
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { MoreHorizontal, Plus, Search } from "lucide-react";

import type { Parent } from "../types";

// ─── Columns ───────────────────────────────────────────────────────────────

function createColumns(onDelete: (id: string) => void): ColumnDef<Parent>[] {
    return [
        {
            accessorKey: "full_name",
            header: "Full Name",
            cell: ({ row }) => (
                <Link
                    href={`/parents/${row.original.id}`}
                    className="text-sm font-medium text-sky-600 transition-colors hover:text-sky-700"
                >
                    {row.original.full_name || "—"}
                </Link>
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
            accessorKey: "phone_number",
            header: "Phone",
            cell: ({ row }) => (
                <span className="text-muted-foreground font-mono text-sm">
                    {row.original.phone_number || "—"}
                </span>
            ),
        },
        {
            id: "linked_count",
            header: "Students",
            cell: () => (
                // Count is not available in the list response; shown in detail
                <span className="text-muted-foreground text-sm">—</span>
            ),
        },
        {
            id: "is_active",
            header: "Status",
            cell: ({ row }) => (
                <Badge
                    variant="secondary"
                    className={
                        row.original.is_active
                            ? "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400"
                            : "bg-muted text-muted-foreground"
                    }
                >
                    {row.original.is_active ? "Active" : "Inactive"}
                </Badge>
            ),
        },
        {
            id: "actions",
            header: "",
            cell: ({ row }) => (
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
                        <DropdownMenuItem asChild>
                            <Link href={`/parents/${row.original.id}`}>View Details</Link>
                        </DropdownMenuItem>
                        <DropdownMenuItem
                            className="text-destructive focus:text-destructive"
                            onClick={() => onDelete(row.original.id)}
                        >
                            Delete
                        </DropdownMenuItem>
                    </DropdownMenuContent>
                </DropdownMenu>
            ),
        },
    ];
}

// ─── Props ─────────────────────────────────────────────────────────────────

interface ParentsTableProps {
    parents: Parent[];
    total: number;
    isLoading: boolean;
    search: string;
    onSearchChange: (value: string) => void;
    onDelete: (id: string) => void;
    onCreateClick: () => void;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function ParentsTable({
    parents,
    total,
    isLoading,
    search,
    onSearchChange,
    onDelete,
    onCreateClick,
}: ParentsTableProps) {
    const columns = React.useMemo(() => createColumns(onDelete), [onDelete]);

    // eslint-disable-next-line react-hooks/incompatible-library
    const table = useReactTable({
        data: parents,
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
            {/* Search bar */}
            <div className="mb-3 flex items-center gap-3">
                <div className="relative max-w-xs flex-1">
                    <Search className="text-muted-foreground absolute top-2.5 left-2.5 size-4" />
                    <Input
                        placeholder="Search by name, email, or phone…"
                        value={search}
                        onChange={(e) => onSearchChange(e.target.value)}
                        className="pl-8"
                    />
                </div>
                <Button variant="outline" size="sm" onClick={onCreateClick}>
                    <Plus className="mr-1.5 size-3.5" />
                    Add Parent
                </Button>
            </div>

            {/* Table */}
            <div
                ref={parentRef}
                className="flex-1 overflow-auto"
                style={{
                    contain: "layout paint",
                    minHeight: rows.length === 0 ? "200px" : undefined,
                }}
            >
                <div className="min-w-[600px]">
                    {/* Sticky Header */}
                    <div className="bg-background/95 sticky top-0 z-10 backdrop-blur-sm">
                        {table.getHeaderGroups().map((headerGroup) => (
                            <div key={headerGroup.id} className="border-border/40 flex border-b">
                                {headerGroup.headers.map((header) => (
                                    <div
                                        key={header.id}
                                        className="text-muted-foreground flex h-10 items-center px-3 text-xs font-medium tracking-wider uppercase"
                                        style={{
                                            flex: header.id !== "actions" ? 1 : "0 0 auto",
                                            width: header.id === "actions" ? 48 : "auto",
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

                    {/* Body */}
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
                                    <Skeleton className="mr-3 h-4 w-24 flex-1" />
                                    <Skeleton className="mr-3 h-4 w-32 flex-1" />
                                    <Skeleton className="mr-3 h-4 w-20 flex-1" />
                                    <Skeleton className="mr-3 h-6 w-16 flex-1" />
                                </div>
                            ))
                        ) : rows.length === 0 ? (
                            <div className="flex items-center justify-center py-16">
                                <div className="text-center">
                                    <p className="text-muted-foreground text-sm font-medium">
                                        {search ? "No parents match your search" : "No parents yet"}
                                    </p>
                                    <p className="text-muted-foreground mt-1 text-xs">
                                        {search
                                            ? "Try a different search term."
                                            : "Add parents to manage guardian communication."}
                                    </p>
                                    {!search && (
                                        <Button
                                            variant="outline"
                                            size="sm"
                                            className="mt-4"
                                            onClick={onCreateClick}
                                        >
                                            <Plus className="mr-1.5 size-3.5" />
                                            Add Parent
                                        </Button>
                                    )}
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
                                                    flex:
                                                        cell.column.id !== "actions"
                                                            ? 1
                                                            : "0 0 auto",
                                                    width:
                                                        cell.column.id === "actions" ? 48 : "auto",
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
                    {total} parent{total !== 1 ? "s" : ""}
                </p>
            </div>
        </div>
    );
}
