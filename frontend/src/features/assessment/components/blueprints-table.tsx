/**
 * Blueprints Table — displays assessment blueprints.
 *
 * Shows title, type, grade level, year, term, indicators count, created date.
 * Supports filtering by grade_level, type, and academic_year.
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
import { Badge } from "@/components/ui/badge";
import { MoreHorizontal, Plus } from "lucide-react";

import type { AssessmentBlueprint } from "../types";

// ─── Helpers ───────────────────────────────────────────────────────────────

function formatDate(iso: string): string {
    return new Date(iso).toLocaleDateString("en-US", {
        month: "short",
        day: "numeric",
        year: "numeric",
    });
}

function typeLabel(type: string): string {
    return type.replace(/_/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());
}

const TYPE_COLORS: Record<string, string> = {
    Formative_Classroom: "bg-sky-100 text-sky-700 dark:bg-sky-900/30 dark:text-sky-400",
    KNEC_Written_Assessment:
        "bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400",
    KNEC_SBA_Project: "bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400",
    National_KPSEA: "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400",
    National_KJSEA: "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400",
    National_KSSEA: "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400",
};

// ─── Columns ───────────────────────────────────────────────────────────────

function createColumns(onDelete: (id: string) => void): ColumnDef<AssessmentBlueprint>[] {
    return [
        {
            accessorKey: "title",
            header: "Title",
            cell: ({ row }) => (
                <Link
                    href={`/assessment/blueprints/${row.original.id}`}
                    className="text-sm font-medium text-sky-600 transition-colors hover:text-sky-700"
                >
                    {row.original.title}
                </Link>
            ),
        },
        {
            accessorKey: "type",
            header: "Type",
            cell: ({ row }) => (
                <Badge
                    variant="secondary"
                    className={TYPE_COLORS[row.original.type] ?? "bg-muted text-muted-foreground"}
                >
                    {typeLabel(row.original.type)}
                </Badge>
            ),
        },
        {
            accessorKey: "grade_level",
            header: "Grade",
            cell: ({ row }) => (
                <span className="text-muted-foreground text-sm">{row.original.grade_level}</span>
            ),
        },
        {
            accessorKey: "academic_year",
            header: "Year",
            cell: ({ row }) => (
                <span className="text-muted-foreground text-sm">{row.original.academic_year}</span>
            ),
        },
        {
            accessorKey: "term",
            header: "Term",
            cell: ({ row }) => (
                <span className="text-muted-foreground text-sm">Term {row.original.term}</span>
            ),
        },
        {
            accessorKey: "created_at",
            header: "Created",
            cell: ({ row }) => (
                <span className="text-muted-foreground text-sm">
                    {formatDate(row.original.created_at)}
                </span>
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
                            <Link href={`/assessment/blueprints/${row.original.id}`}>
                                View Details
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
            ),
        },
    ];
}

// ─── Props ─────────────────────────────────────────────────────────────────

interface BlueprintsTableProps {
    blueprints: AssessmentBlueprint[];
    total: number;
    isLoading: boolean;
    onDelete: (id: string) => void;
    onCreateClick: () => void;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function BlueprintsTable({
    blueprints,
    total,
    isLoading,
    onDelete,
    onCreateClick,
}: BlueprintsTableProps) {
    const columns = React.useMemo(() => createColumns(onDelete), [onDelete]);

    // eslint-disable-next-line react-hooks/incompatible-library
    const table = useReactTable({
        data: blueprints,
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
                                    <Skeleton className="mr-3 h-4 w-32" />
                                    <Skeleton className="mr-3 h-4 w-20" />
                                    <Skeleton className="mr-3 h-4 w-10" />
                                    <Skeleton className="mr-3 h-4 w-12" />
                                    <Skeleton className="mr-3 h-4 w-14" />
                                    <Skeleton className="mr-3 h-4 w-24" />
                                </div>
                            ))
                        ) : rows.length === 0 ? (
                            <div className="flex items-center justify-center py-16">
                                <div className="text-center">
                                    <p className="text-muted-foreground text-sm font-medium">
                                        No assessment blueprints yet
                                    </p>
                                    <p className="text-muted-foreground mt-1 text-xs">
                                        Create your first blueprint to get started.
                                    </p>
                                    <Button
                                        variant="outline"
                                        size="sm"
                                        className="mt-4"
                                        onClick={onCreateClick}
                                    >
                                        <Plus className="mr-1.5 size-3.5" />
                                        New Blueprint
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
                    {total} blueprint{total !== 1 ? "s" : ""}
                </p>
            </div>
        </div>
    );
}
