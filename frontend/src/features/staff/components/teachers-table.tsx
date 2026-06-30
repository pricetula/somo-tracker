/**
 * Teachers Listing Table — displays teacher users with educator-specific fields.
 *
 * Columns: Full Name, Email, TSC Number, KNEC Panel Assessor ID,
 *          Core Assignment Role, Account Status (toggle).
 * Uses TanStack Table + TanStack Virtual for performance.
 */

"use client";

import * as React from "react";
import { useReactTable, getCoreRowModel, flexRender, type ColumnDef } from "@tanstack/react-table";
import { useVirtualizer } from "@tanstack/react-virtual";

import { Switch } from "@/components/ui/switch";
import { Skeleton } from "@/components/ui/skeleton";

import type { TeacherMember } from "@/lib/api/teachers";
import { useToggleTeacherActive } from "../hooks/use-teachers";

// ─── Teacher Role Labels ──────────────────────────────────────────────────

const TEACHER_ROLE_LABELS: Record<string, string> = {
    PRIMARY_CLASS_TEACHER: "Primary Class Teacher",
    SUBJECT_TEACHER: "Subject Teacher",
    SUBSTITUTE_TEACHER: "Substitute Teacher",
};

function formatTeacherRole(role: string | null): string {
    if (!role) return "—";
    return TEACHER_ROLE_LABELS[role] ?? role;
}

// ─── Columns ───────────────────────────────────────────────────────────────

function createColumns(): ColumnDef<TeacherMember>[] {
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
            accessorKey: "tsc_number",
            header: "TSC Number",
            cell: ({ row }) => (
                <span className="text-muted-foreground font-mono text-sm">
                    {row.original.tsc_number ?? "—"}
                </span>
            ),
        },
        {
            accessorKey: "knec_panel_assessor_id",
            header: "KNEC Panel Assessor ID",
            cell: ({ row }) => (
                <span className="text-muted-foreground font-mono text-sm">
                    {row.original.knec_panel_assessor_id ?? "—"}
                </span>
            ),
        },
        {
            accessorKey: "teacher_role",
            header: "Core Assignment Role",
            cell: ({ row }) => (
                <span className="text-muted-foreground text-sm">
                    {formatTeacherRole(row.original.teacher_role)}
                </span>
            ),
        },
        {
            id: "is_active",
            header: "Account Status",
            cell: ({ row }) => <StatusToggleCell teacher={row.original} />,
        },
    ];
}

// ─── Status Toggle Cell ────────────────────────────────────────────────────

function StatusToggleCell({ teacher }: { teacher: TeacherMember }) {
    const toggleMutation = useToggleTeacherActive();

    const handleToggle = (checked: boolean) => {
        toggleMutation.mutate({ userId: teacher.id, isActive: checked });
    };

    return (
        <div className="flex items-center gap-2">
            <Switch
                checked={teacher.is_active}
                onCheckedChange={handleToggle}
                disabled={toggleMutation.isPending}
                aria-label={teacher.is_active ? "Deactivate teacher" : "Activate teacher"}
            />
            <span
                className={
                    teacher.is_active
                        ? "text-xs font-medium text-emerald-600"
                        : "text-muted-foreground text-xs"
                }
            >
                {teacher.is_active ? "Active" : "Inactive"}
            </span>
        </div>
    );
}

// ─── Props ─────────────────────────────────────────────────────────────────

interface TeachersTableProps {
    teachers: TeacherMember[];
    total: number;
    isLoading: boolean;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function TeachersTable({ teachers, total, isLoading }: TeachersTableProps) {
    const columns = React.useMemo(() => createColumns(), []);

    // eslint-disable-next-line react-hooks/incompatible-library
    const table = useReactTable({
        data: teachers,
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
                                    <Skeleton className="mr-3 h-4 w-16 flex-1" />
                                    <Skeleton className="mr-3 h-4 w-16 flex-1" />
                                    <Skeleton className="mr-3 h-4 w-24 flex-1" />
                                    <Skeleton className="mr-3 h-6 w-16 flex-1" />
                                </div>
                            ))
                        ) : rows.length === 0 ? (
                            <div className="flex items-center justify-center py-16">
                                <p className="text-muted-foreground text-sm font-medium">
                                    No teachers yet
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
                <p className="text-muted-foreground text-xs">
                    {total} teacher{total !== 1 ? "s" : ""}
                </p>
            </div>
        </div>
    );
}
