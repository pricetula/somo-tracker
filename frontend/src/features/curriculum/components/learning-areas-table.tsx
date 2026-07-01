/**
 * Learning Areas Table — lists learning areas for the current school.
 *
 * Columns: Code, Name, Education Level, strand count.
 * Click a row to navigate to the tree view.
 */

"use client";

import * as React from "react";
import { useReactTable, getCoreRowModel, flexRender, type ColumnDef } from "@tanstack/react-table";
import { useVirtualizer } from "@tanstack/react-virtual";
import { useRouter } from "next/navigation";

import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Plus } from "lucide-react";

import type { LearningArea } from "@/lib/api/curriculum";

// ─── Education Level Labels ───────────────────────────────────────────────

const EDUCATION_LEVEL_LABELS: Record<string, string> = {
    Early_Years: "Early Years",
    Upper_Primary: "Upper Primary",
    Junior_Secondary: "Junior Secondary",
    Senior_School: "Senior School",
};

function formatEducationLevel(level: string): string {
    return EDUCATION_LEVEL_LABELS[level] ?? level;
}

// ─── Columns ───────────────────────────────────────────────────────────────

function createColumns(): ColumnDef<LearningArea>[] {
    return [
        {
            accessorKey: "code",
            header: "Code",
            cell: ({ row }) => (
                <span className="font-mono text-sm font-medium">{row.original.code}</span>
            ),
        },
        {
            accessorKey: "name",
            header: "Name",
            cell: ({ row }) => <span className="text-sm">{row.original.name}</span>,
        },
        {
            accessorKey: "education_level",
            header: "Education Level",
            cell: ({ row }) => (
                <span className="text-muted-foreground text-sm">
                    {formatEducationLevel(row.original.education_level)}
                </span>
            ),
        },
    ];
}

// ─── Props ─────────────────────────────────────────────────────────────────

interface LearningAreasTableProps {
    learningAreas: LearningArea[];
    total: number;
    isLoading: boolean;
    onCreateClick: () => void;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function LearningAreasTable({
    learningAreas,
    total,
    isLoading,
    onCreateClick,
}: LearningAreasTableProps) {
    const router = useRouter();
    const columns = React.useMemo(() => createColumns(), []);

    // eslint-disable-next-line react-hooks/incompatible-library
    const table = useReactTable({
        data: learningAreas,
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
                <div className="min-w-150">
                    {/* Sticky Header */}
                    <div className="bg-background/95 sticky top-0 z-10 backdrop-blur-sm">
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
                                    <Skeleton className="mr-3 h-4 w-16 flex-1" />
                                    <Skeleton className="mr-3 h-4 w-24 flex-1" />
                                    <Skeleton className="mr-3 h-4 w-20 flex-1" />
                                </div>
                            ))
                        ) : rows.length === 0 ? (
                            <div className="flex flex-col items-center gap-2 py-16">
                                <p className="text-muted-foreground text-sm font-medium">
                                    No learning areas yet
                                </p>
                                <p className="text-muted-foreground text-xs">
                                    Add a learning area to get started.
                                </p>
                                <Button
                                    variant="outline"
                                    size="sm"
                                    className="mt-2"
                                    onClick={onCreateClick}
                                >
                                    <Plus className="mr-1.5 size-3.5" />
                                    Add Learning Area
                                </Button>
                            </div>
                        ) : (
                            virtualizer.getVirtualItems().map((virtualRow) => {
                                const row = rows[virtualRow.index];
                                return (
                                    <div
                                        key={virtualRow.key}
                                        className="group border-border/40 hover:bg-muted/30 absolute right-0 left-0 flex cursor-pointer items-center border-b transition-colors"
                                        style={{
                                            position: "absolute",
                                            top: 0,
                                            left: 0,
                                            width: "100%",
                                            height: `${virtualRow.size}px`,
                                            transform: `translateY(${virtualRow.start}px)`,
                                        }}
                                        onClick={() =>
                                            router.push(`/curriculum/${row.original.id}`)
                                        }
                                        onKeyDown={(e) => {
                                            if (e.key === "Enter" || e.key === " ") {
                                                e.preventDefault();
                                                router.push(`/curriculum/${row.original.id}`);
                                            }
                                        }}
                                        role="button"
                                        tabIndex={0}
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
                    {total} learning area{total !== 1 ? "s" : ""}
                </p>
            </div>
        </div>
    );
}
