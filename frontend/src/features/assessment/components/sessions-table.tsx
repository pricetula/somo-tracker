/**
 * Sessions Table — displays assessment sessions.
 *
 * Shows blueprint title, class, date administered, assessed by, results count.
 * Actions: Create, Delete, "Score" button → navigate to score page.
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
import { Skeleton } from "@/components/ui/skeleton";
import { MoreHorizontal, Plus, ClipboardCheck } from "lucide-react";

import type { AssessmentSession } from "../types";

// ─── Helpers ───────────────────────────────────────────────────────────────

function formatDate(iso: string): string {
    return new Date(iso).toLocaleDateString("en-US", {
        month: "short",
        day: "numeric",
        year: "numeric",
    });
}

// ─── Columns ───────────────────────────────────────────────────────────────

interface SessionRow extends AssessmentSession {
    blueprint_title?: string;
    assessed_by_name?: string;
    results_count?: number;
    class_name?: string;
}

function createColumns(onDelete: (id: string) => void): ColumnDef<SessionRow>[] {
    return [
        {
            accessorKey: "blueprint_title",
            header: "Blueprint",
            cell: ({ row }) => (
                <span className="text-sm font-medium">{row.original.blueprint_title || "—"}</span>
            ),
        },
        {
            accessorKey: "class_name",
            header: "Class",
            cell: ({ row }) => (
                <span className="text-muted-foreground text-sm">
                    {row.original.class_name || "—"}
                </span>
            ),
        },
        {
            accessorKey: "date_administered",
            header: "Date",
            cell: ({ row }) => (
                <span className="text-muted-foreground text-sm">
                    {row.original.date_administered
                        ? formatDate(row.original.date_administered)
                        : "—"}
                </span>
            ),
        },
        {
            accessorKey: "assessed_by_name",
            header: "Assessed By",
            cell: ({ row }) => (
                <span className="text-muted-foreground text-sm">
                    {row.original.assessed_by_name || "—"}
                </span>
            ),
        },
        {
            accessorKey: "results_count",
            header: "Results",
            cell: ({ row }) => (
                <span className="text-muted-foreground text-sm">
                    {row.original.results_count ?? "—"}
                </span>
            ),
        },
        {
            id: "actions",
            header: "",
            cell: ({ row }) => (
                <div className="flex items-center gap-1">
                    <Button variant="ghost" size="icon-sm" asChild>
                        <Link href={`/assessment/sessions/${row.original.id}/score`}>
                            <ClipboardCheck className="size-4" />
                            <span className="sr-only">Score</span>
                        </Link>
                    </Button>
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
                                <Link href={`/assessment/sessions/${row.original.id}`}>
                                    View Details
                                </Link>
                            </DropdownMenuItem>
                            <DropdownMenuItem asChild>
                                <Link href={`/assessment/sessions/${row.original.id}/score`}>
                                    Score
                                </Link>
                            </DropdownMenuItem>
                            <DropdownMenuItem
                                className="text-destructive focus:text-destructive"
                                onClick={() => onDelete(row.original.id)}
                            >
                                Delete
                            </DropdownMenuItem>
                        </DropdownMenuContent>
                    </DropdownMenu>
                </div>
            ),
        },
    ];
}

// ─── Props ─────────────────────────────────────────────────────────────────

interface SessionsTableProps {
    sessions: SessionRow[];
    total: number;
    isLoading: boolean;
    onDelete: (id: string) => void;
    onCreateClick: () => void;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function SessionsTable({
    sessions,
    total,
    isLoading,
    onDelete,
    onCreateClick,
}: SessionsTableProps) {
    const columns = React.useMemo(() => createColumns(onDelete), [onDelete]);

    // eslint-disable-next-line react-hooks/incompatible-library
    const table = useReactTable({
        data: sessions,
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
            <div
                ref={parentRef}
                className="flex-1 overflow-auto"
                style={{
                    contain: "layout paint",
                    minHeight: rows.length === 0 ? "200px" : undefined,
                }}
            >
                <div className="min-w-[700px]">
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
                                            width: header.id === "actions" ? 80 : "auto",
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
                                    <Skeleton className="mr-3 h-4 w-32" />
                                    <Skeleton className="mr-3 h-4 w-20" />
                                    <Skeleton className="mr-3 h-4 w-24" />
                                    <Skeleton className="mr-3 h-4 w-20" />
                                    <Skeleton className="mr-3 h-4 w-10" />
                                </div>
                            ))
                        ) : rows.length === 0 ? (
                            <div className="flex items-center justify-center py-16">
                                <div className="text-center">
                                    <p className="text-muted-foreground text-sm font-medium">
                                        No assessment sessions yet
                                    </p>
                                    <p className="text-muted-foreground mt-1 text-xs">
                                        Create a session from a blueprint to start scoring.
                                    </p>
                                    <Button
                                        variant="outline"
                                        size="sm"
                                        className="mt-4"
                                        onClick={onCreateClick}
                                    >
                                        <Plus className="mr-1.5 size-3.5" />
                                        New Session
                                    </Button>
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
                                                        cell.column.id === "actions" ? 80 : "auto",
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
                    {total} session{total !== 1 ? "s" : ""}
                </p>
            </div>
        </div>
    );
}
