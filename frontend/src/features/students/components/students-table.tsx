/**
 * Students Listing Table — the most data-heavy screen in the app.
 *
 * Columns: Full Name, Admission Number, UPI Number, KNEC Assessment Number,
 *          Class, Gender, Enrollment Status, Actions (delete).
 * Search targets: Full Name, Admission Number, UPI Number, KNEC Assessment Number.
 * Filters split into:
 *   - Curriculum Group: Education Levels or Grade Levels (multi-select)
 *   - Lifecycle Group: Enrollment Status (Active, Suspended, Transferred)
 * Bulk Import opens the intercepted parallel route modal.
 *
 * Uses TanStack Table + TanStack Virtual for performance.
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
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { FilterDropdown } from "@/components/shared/data-table/filter-dropdown";
import type { FilterGroup } from "@/components/shared/data-table/types";
import { MoreHorizontal, Search, Upload } from "lucide-react";

import type { Student } from "@/lib/api/students";
import { useDeleteStudent } from "../hooks/use-students";

// ─── Filter Groups ────────────────────────────────────────────────────────
// Matches the multi-select architecture from the Curriculum page.

const CURRICULUM_FILTER_GROUPS: FilterGroup[] = [
    {
        id: "curriculum_filters",
        label: "Curriculum Filters",
        items: [
            {
                id: "education_level",
                label: "Education Level",
                type: "sub_menu_multi",
                submenu: [
                    { id: "early_years", label: "Early Years", value: "Early_Years" },
                    { id: "upper_primary", label: "Upper Primary", value: "Upper_Primary" },
                    {
                        id: "junior_secondary",
                        label: "Junior Secondary",
                        value: "Junior_Secondary",
                    },
                    { id: "senior_school", label: "Senior School", value: "Senior_School" },
                ],
            },
            {
                id: "grade_level",
                label: "Grade",
                type: "sub_menu_multi",
                submenu: [
                    { id: "pp1", label: "PP1", value: "PP1" },
                    { id: "pp2", label: "PP2", value: "PP2" },
                    { id: "g1", label: "Grade 1", value: "G1" },
                    { id: "g2", label: "Grade 2", value: "G2" },
                    { id: "g3", label: "Grade 3", value: "G3" },
                    { id: "g4", label: "Grade 4", value: "G4" },
                    { id: "g5", label: "Grade 5", value: "G5" },
                    { id: "g6", label: "Grade 6", value: "G6" },
                    { id: "g7", label: "Grade 7", value: "G7" },
                    { id: "g8", label: "Grade 8", value: "G8" },
                    { id: "g9", label: "Grade 9", value: "G9" },
                    { id: "g10", label: "Grade 10", value: "G10" },
                    { id: "g11", label: "Grade 11", value: "G11" },
                    { id: "g12", label: "Grade 12", value: "G12" },
                ],
            },
        ],
    },
    {
        id: "lifecycle_group",
        label: "Lifecycle Group",
        items: [
            {
                id: "enrollment_status",
                label: "Enrollment Status",
                type: "sub_menu_single",
                submenu: [
                    { id: "active_status", label: "Active", value: "ACTIVE" },
                    { id: "suspended_status", label: "Suspended", value: "SUSPENDED" },
                    { id: "transferred_status", label: "Transferred", value: "TRANSFERRED" },
                ],
            },
        ],
    },
];

// ─── Columns ───────────────────────────────────────────────────────────────

function createColumns(onDeleteRequest: (student: Student) => void): ColumnDef<Student>[] {
    return [
        {
            accessorKey: "full_name",
            header: "Full Name",
            cell: ({ row }) => (
                <span className="text-sm font-medium">{row.original.full_name || "—"}</span>
            ),
        },
        {
            accessorKey: "admission_number",
            header: "Admission No.",
            cell: ({ row }) => (
                <span className="text-muted-foreground font-mono text-sm">
                    {row.original.admission_number ?? "—"}
                </span>
            ),
        },
        {
            accessorKey: "upi_number",
            header: "UPI Number",
            cell: ({ row }) => (
                <span className="text-muted-foreground font-mono text-sm">
                    {row.original.upi_number ?? "—"}
                </span>
            ),
        },
        {
            accessorKey: "knec_assessment_number",
            header: "KNEC No.",
            cell: ({ row }) => (
                <span className="text-muted-foreground font-mono text-sm">
                    {row.original.knec_assessment_number ?? "—"}
                </span>
            ),
        },
        {
            accessorKey: "class_name",
            header: "Class",
            cell: ({ row }) => (
                <span className="text-muted-foreground text-sm">
                    {row.original.class_name ?? "—"}
                </span>
            ),
        },
        {
            accessorKey: "gender",
            header: "Gender",
            cell: ({ row }) => (
                <span className="text-muted-foreground text-sm">
                    {row.original.gender === "M"
                        ? "Male"
                        : row.original.gender === "F"
                          ? "Female"
                          : "—"}
                </span>
            ),
        },
        {
            id: "enrollment_status",
            header: "Status",
            cell: ({ row }) => (
                // Enrollment status is not piped through the list query currently;
                // this shows active/inactive based on the student record.
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
                            <Link href={`/students/${row.original.id}`}>View Details</Link>
                        </DropdownMenuItem>
                        <DropdownMenuItem
                            className="text-destructive focus:text-destructive"
                            onClick={() => onDeleteRequest(row.original)}
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

interface StudentsTableProps {
    students: Student[];
    total: number;
    isLoading: boolean;
    search: string;
    onSearchChange: (value: string) => void;
    /** Active filter values keyed by FilterItem id. */
    activeFilters: Record<string, string | string[]>;
    onToggleButton: (itemId: string, itemValue: string) => void;
    onSelectSingle: (itemId: string, subValue: string) => void;
    onToggleMulti: (itemId: string, subValue: string) => void;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function StudentsTable({
    students,
    total,
    isLoading,
    search,
    onSearchChange,
    activeFilters,
    onToggleButton,
    onSelectSingle,
    onToggleMulti,
}: StudentsTableProps) {
    const deleteMutation = useDeleteStudent();
    const [studentToDelete, setStudentToDelete] = React.useState<Student | null>(null);

    const columns = React.useMemo(() => createColumns(setStudentToDelete), []);

    // eslint-disable-next-line react-hooks/incompatible-library
    const table = useReactTable({
        data: students,
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

    const handleDeleteConfirm = () => {
        if (!studentToDelete) return;
        deleteMutation.mutate(studentToDelete.id, {
            onSettled: () => setStudentToDelete(null),
        });
    };

    return (
        <div className="flex flex-1 flex-col">
            {/* Search bar */}
            <div className="mb-3 flex items-center gap-3">
                <div className="relative max-w-xs flex-1">
                    <Search className="text-muted-foreground absolute top-2.5 left-2.5 size-4" />
                    <Input
                        placeholder="Search by name, admission no., UPI, or KNEC no…"
                        value={search}
                        onChange={(e) => onSearchChange(e.target.value)}
                        className="pl-8"
                    />
                </div>
                <FilterDropdown
                    groups={CURRICULUM_FILTER_GROUPS}
                    activeFilters={activeFilters}
                    onToggleButton={onToggleButton}
                    onSelectSingle={onSelectSingle}
                    onToggleMulti={onToggleMulti}
                />
                <Button variant="outline" size="sm" asChild>
                    <Link href="/students/import">
                        <Upload className="mr-1.5 size-3.5" />
                        Bulk Import
                    </Link>
                </Button>
            </div>

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
                                    <Skeleton className="mr-3 h-4 w-16 flex-1" />
                                    <Skeleton className="mr-3 h-4 w-16 flex-1" />
                                    <Skeleton className="mr-3 h-4 w-16 flex-1" />
                                    <Skeleton className="mr-3 h-4 w-16 flex-1" />
                                    <Skeleton className="mr-3 h-4 w-12 flex-1" />
                                    <Skeleton className="mr-3 h-6 w-16 flex-1" />
                                    <Skeleton className="mr-3 h-4 w-4 flex-1" />
                                </div>
                            ))
                        ) : rows.length === 0 ? (
                            <div className="flex items-center justify-center py-16">
                                <div className="text-center">
                                    <p className="text-muted-foreground text-sm font-medium">
                                        {search
                                            ? "No students match your search"
                                            : "No students yet"}
                                    </p>
                                    <p className="text-muted-foreground mt-1 text-xs">
                                        {search
                                            ? "Try a different search term."
                                            : "Import students to build your roster."}
                                    </p>
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
                    {total} student{total !== 1 ? "s" : ""}
                </p>
            </div>

            {/* Delete confirmation dialog */}
            <AlertDialog
                open={!!studentToDelete}
                onOpenChange={(open) => {
                    if (!open) setStudentToDelete(null);
                }}
            >
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>Delete student</AlertDialogTitle>
                        <AlertDialogDescription>
                            Are you sure you want to permanently delete{" "}
                            <strong>{studentToDelete?.full_name || studentToDelete?.id}</strong>?
                            This action cannot be undone and will remove all enrollment records.
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel disabled={deleteMutation.isPending}>
                            Cancel
                        </AlertDialogCancel>
                        <AlertDialogAction
                            onClick={handleDeleteConfirm}
                            disabled={deleteMutation.isPending}
                        >
                            {deleteMutation.isPending ? "Deleting..." : "Delete"}
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </div>
    );
}
