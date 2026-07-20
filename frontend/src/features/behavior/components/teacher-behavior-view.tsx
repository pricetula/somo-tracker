/**
 * TeacherBehaviorView — shows a teacher's own submitted behavior notes.
 *
 * Uses the shared DataTable component for listing with review status badges.
 */

"use client";

import { AlertTriangle } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import { useTeacherNotes, useDeleteBehaviorNote } from "../hooks/use-behavior";
import type { TeacherNoteItem } from "@/lib/api/behavior";

// ─── Status Badge ─────────────────────────────────────────────────────────

function StatusBadge({ status }: { status: string }) {
    switch (status) {
        case "PENDING_REVIEW":
            return (
                <Badge variant="outline" className="text-amber-600">
                    Pending Review
                </Badge>
            );
        case "APPROVED":
            return (
                <Badge className="bg-green-100 text-green-700 hover:bg-green-100">Approved</Badge>
            );
        case "REJECTED":
            return <Badge variant="destructive">Rejected</Badge>;
        case "INCLUDED_IN_REPORT":
            return <Badge className="bg-sky-100 text-sky-700 hover:bg-sky-100">In Report</Badge>;
        default:
            return <Badge variant="outline">{status}</Badge>;
    }
}

function formatDate(dateStr: string): string {
    try {
        const date = new Date(dateStr);
        return date.toLocaleDateString(undefined, {
            month: "short",
            day: "numeric",
            year: "numeric",
        });
    } catch {
        return dateStr;
    }
}

// ─── Columns ──────────────────────────────────────────────────────────────

const columns: DataTableColumn<TeacherNoteItem>[] = [
    {
        id: "student",
        header: "Student",
        cell: (row) => (
            <div className="flex items-center gap-2">
                <span className="font-medium">{row.student_full_name}</span>
                <Badge variant="outline" className="text-[10px]">
                    {row.class_name}
                </Badge>
            </div>
        ),
    },
    {
        id: "description",
        header: "Description",
        width: "minmax(200px, 1fr)",
        cell: (row) => (
            <span className="text-muted-foreground line-clamp-2 text-xs">{row.description}</span>
        ),
    },
    {
        id: "category",
        header: "Category",
        width: "140px",
        cell: (row) => (
            <div className="flex flex-wrap items-center gap-1">
                <Badge variant="secondary" className="text-[10px]">
                    {row.category_name}
                </Badge>
                {row.is_urgent && (
                    <Badge variant="destructive" className="gap-1 text-[10px]">
                        <AlertTriangle className="h-3 w-3" />
                        Urgent
                    </Badge>
                )}
            </div>
        ),
    },
    {
        id: "status",
        header: "Status",
        width: "140px",
        cell: (row) => <StatusBadge status={row.status} />,
    },
    {
        id: "date",
        header: "Date",
        width: "120px",
        cell: (row) => (
            <span className="text-muted-foreground text-xs">{formatDate(row.date)}</span>
        ),
    },
];

// ─── Component ────────────────────────────────────────────────────────────

export function TeacherBehaviorView() {
    const { data, isError } = useTeacherNotes();
    const deleteMutation = useDeleteBehaviorNote();

    if (isError) {
        return (
            <div className="border-destructive/50 text-destructive rounded-md border p-4">
                Failed to load your behavior notes. Please try again later.
            </div>
        );
    }

    const notes = data?.items ?? [];

    return (
        <div className="space-y-4">
            <DataTable
                isCheckable
                queryKey={["behavior", "my-notes"]}
                queryFn={() => Promise.resolve({ items: notes, total: notes.length })}
                columns={columns}
                getRowId={(row) => row.id}
                deleteFn={(id) => deleteMutation.mutateAsync(String(id))}
                emptyState={
                    <div className="text-muted-foreground flex flex-col items-center gap-4 py-16">
                        <p className="font-medium">No behavior notes yet</p>
                        <p className="mt-1 max-w-sm text-center">
                            You haven&apos;t submitted any behavior notes yet.
                        </p>
                    </div>
                }
                noResultsState="No notes match your search."
                renderToolBarComponents={() => null}
            />
        </div>
    );
}
