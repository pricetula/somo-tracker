/**
 * ClassTable — Virtualized table for listing classrooms.
 *
 * Mirrors the pattern from student-table / member-table.
 * Columns: Name, Stream, Grade, Status, Actions.
 */

"use client";

import * as React from "react";
import Link from "next/link";
import { useReactTable, getCoreRowModel, flexRender, type ColumnDef } from "@tanstack/react-table";
import { useVirtualizer } from "@tanstack/react-virtual";

import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { MoreHorizontal, Search } from "lucide-react";
import { Input } from "@/components/ui/input";
import type { ClassItem } from "@/features/classes/types";

// ─── Columns ───────────────────────────────────────────────────────────────

const columns: ColumnDef<ClassItem>[] = [
    {
        accessorKey: "name",
        header: "Class Name",
        cell: ({ row }) => (
            <Link
                href={`/classes/${row.original.id}?tab=timetable`}
                className="text-sm font-medium transition-colors hover:text-teal-600 hover:underline"
            >
                {row.original.name}
            </Link>
        ),
    },
    {
        accessorKey: "stream",
        header: "Stream",
        cell: ({ row }) => (
            <span className="text-muted-foreground text-sm">{row.original.stream || "—"}</span>
        ),
    },
    {
        accessorKey: "is_active",
        header: "Status",
        cell: ({ row }) => (
            <Badge
                variant={row.original.is_active ? "default" : "secondary"}
                className="text-[11px] capitalize"
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
                        <Link href={`/classes/${row.original.id}?tab=timetable`}>View Details</Link>
                    </DropdownMenuItem>
                    <DropdownMenuItem>Manage Students</DropdownMenuItem>
                    <DropdownMenuItem className="text-destructive">Deactivate</DropdownMenuItem>
                </DropdownMenuContent>
            </DropdownMenu>
        ),
    },
];

// ─── Props ─────────────────────────────────────────────────────────────────

interface ClassTableProps {
    classes: ClassItem[];
    search: string;
    onSearchChange: (value: string) => void;
    isLoading: boolean;
    /** Renders the filter button — injected from parent so the filter stays
     *  connected to the page-level state. */
    filterSlot?: React.ReactNode;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function ClassTable({
    classes,
    search,
    onSearchChange,
    isLoading,
    filterSlot,
}: ClassTableProps) {
    // eslint-disable-next-line react-hooks/incompatible-library
    const table = useReactTable({
        data: classes,
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
            {/* ── Controls Row ─────────────────────────────────────────── */}
            <div className="flex items-center gap-3 px-6 py-3">
                <div className="relative max-w-sm flex-1">
                    <Search className="text-muted-foreground pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2" />
                    <Input
                        placeholder="Search classes..."
                        value={search}
                        onChange={(e) => onSearchChange(e.target.value)}
                        className="bg-muted/50 h-9 border-none pl-8 text-sm"
                    />
                </div>
                {filterSlot}
            </div>

            {/* ── Table Canvas ──────────────────────────────────────────── */}
            <div
                ref={parentRef}
                className="flex-1 overflow-auto px-6"
                style={{ contain: "strict" }}
            >
                <div className="min-w-[500px]">
                    {/* Sticky Header */}
                    <div className="bg-background/95 sticky top-0 z-10 backdrop-blur-sm">
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
                        {isLoading && rows.length === 0
                            ? Array.from({ length: skeletonRows }).map((_, i) => (
                                  <div
                                      key={`skeleton-${i}`}
                                      className="border-border/40 flex h-12 items-center border-b px-3"
                                  >
                                      <Skeleton className="mr-3 h-4 w-40" />
                                      <Skeleton className="mr-3 h-4 w-20" />
                                      <Skeleton className="h-4 w-16" />
                                  </div>
                              ))
                            : rows.length === 0
                              ? null
                              : virtualizer.getVirtualItems().map((virtualRow) => {
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
                                })}
                    </div>
                </div>
            </div>

            {/* ── Footer counter ────────────────────────────────────────── */}
            <div className="border-border/40 flex items-center justify-between border-t px-6 py-2">
                <p className="text-muted-foreground text-xs">
                    {classes.length} class{classes.length !== 1 ? "es" : ""}
                </p>
            </div>
        </div>
    );
}
