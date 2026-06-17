/**
 * Virtual Student Table — TanStack Table + TanStack Virtual.
 *
 * Full-width canvas without card wrappers. Sticky headers, ultra-low
 * contrast horizontal dividers, fixed h-12 rows, hidden row actions
 * triggered via ghost ellipsis dropdown.
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
import { MoreHorizontal, Search, Filter } from "lucide-react";
import { Input } from "@/components/ui/input";
import type { Student } from "@/lib/api/students";

// ─── Columns ───────────────────────────────────────────────────────────────

const columns: ColumnDef<Student>[] = [
    {
        accessorKey: "first_name",
        header: "First Name",
        cell: ({ row }) => <span className="text-sm font-medium">{row.original.first_name}</span>,
    },
    {
        accessorKey: "middle_name",
        header: "Middle Name",
        cell: ({ row }) => (
            <span className="text-muted-foreground text-sm">{row.original.middle_name ?? "—"}</span>
        ),
    },
    {
        accessorKey: "last_name",
        header: "Last Name",
        cell: ({ row }) => <span className="text-sm">{row.original.last_name}</span>,
    },
    {
        accessorKey: "gender",
        header: "Gender",
        cell: ({ row }) => (
            <span className="text-muted-foreground text-sm capitalize">
                {row.original.gender.toLowerCase().replace(/_/g, " ")}
            </span>
        ),
    },
    {
        accessorKey: "date_of_birth",
        header: "Date of Birth",
        cell: ({ row }) => (
            <span className="text-muted-foreground text-sm">{row.original.date_of_birth}</span>
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
                    <DropdownMenuItem>Edit</DropdownMenuItem>
                    <DropdownMenuItem>View Profile</DropdownMenuItem>
                    <DropdownMenuItem className="text-destructive">Deactivate</DropdownMenuItem>
                </DropdownMenuContent>
            </DropdownMenu>
        ),
    },
];

// ─── Props ─────────────────────────────────────────────────────────────────

interface StudentTableProps {
    students: Student[];
    total: number;
    search: string;
    onSearchChange: (value: string) => void;
    onAddStudent: () => void;
    onUploadCSV: () => void;
    isLoading: boolean;
    fetchNextPage?: () => void;
    hasNextPage?: boolean;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function StudentTable({
    students,
    total,
    search,
    onSearchChange,
    onAddStudent,
    onUploadCSV,
    isLoading,
}: StudentTableProps) {
    // ── Table instance ────────────────────────────────────────────────────
    // eslint-disable-next-line react-hooks/incompatible-library
    const table = useReactTable({
        data: students,
        columns,
        getCoreRowModel: getCoreRowModel(),
    });

    // ── Virtualization ────────────────────────────────────────────────────
    const parentRef = React.useRef<HTMLDivElement>(null);
    const rows = table.getRowModel().rows;

    const virtualizer = useVirtualizer({
        count: rows.length,
        getScrollElement: () => parentRef.current,
        estimateSize: () => 48, // h-12 = 3rem = 48px
        overscan: 10,
    });

    // ── Skeleton rows for loading state ────────────────────────────────────
    const skeletonRows = 10;

    return (
        <div className="flex flex-1 flex-col">
            {/* ── Controls Row ─────────────────────────────────────────── */}
            <div className="flex items-center gap-3 px-6 py-3">
                <div className="relative max-w-sm flex-1">
                    <Search className="text-muted-foreground pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2" />
                    <Input
                        placeholder="Search students..."
                        value={search}
                        onChange={(e) => onSearchChange(e.target.value)}
                        className="bg-muted/50 h-9 border-none pl-8 text-sm"
                    />
                </div>
                <Button variant="ghost" size="icon-sm" aria-label="Filter">
                    <Filter className="size-4" />
                </Button>
                <div className="ml-auto flex items-center gap-2">
                    <Button size="sm" onClick={onAddStudent}>
                        Add Student
                    </Button>
                    <Button variant="outline" size="sm" onClick={onUploadCSV}>
                        Upload CSV
                    </Button>
                </div>
            </div>

            {/* ── Table Canvas ──────────────────────────────────────────── */}
            <div
                ref={parentRef}
                className="flex-1 overflow-auto px-6"
                style={{ contain: "strict" }}
            >
                <div className="min-w-[640px]">
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
                            ? // Loading skeleton
                              Array.from({ length: skeletonRows }).map((_, i) => (
                                  <div
                                      key={`skeleton-${i}`}
                                      className="border-border/40 flex h-12 items-center border-b px-3"
                                  >
                                      <Skeleton className="mr-3 h-4 w-24" />
                                      <Skeleton className="mr-3 h-4 w-16" />
                                      <Skeleton className="mr-3 h-4 w-24" />
                                      <Skeleton className="mr-3 h-4 w-16" />
                                      <Skeleton className="h-4 w-24" />
                                  </div>
                              ))
                            : rows.length === 0
                              ? // The empty state is handled by the parent
                                null
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
                    {total} student{total !== 1 ? "s" : ""}
                </p>
            </div>
        </div>
    );
}
