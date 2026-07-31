"use client";

import { format } from "date-fns";
import { AlertTriangle } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { DataTable } from "@/components/shared/data-table";
import { type DataTableColumn } from "@/components/shared/data-table/types";
import { RowActions } from "@/components/shared/data-table/row-actions";
import { useTeacherNotes, useDeleteBehaviorNote } from "../hooks/use-behavior";
import { type TeacherNoteItem } from "@/lib/api/behavior";

function formatDate(dateStr: string): string {
    try {
        return format(new Date(dateStr), "MMM d, yyyy");
    } catch {
        return dateStr;
    }
}
function createColumns(
    deleteMutation: ReturnType<typeof useDeleteBehaviorNote>
): DataTableColumn<TeacherNoteItem>[] {
    return [
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
                <span className="text-muted-foreground line-clamp-2 text-xs">
                    {row.description}
                </span>
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
        {
            id: "actions",
            header: "",
            width: "48px",
            align: "right",
            cell: (row) => (
                <RowActions
                    rowId={row.id}
                    label={`note for ${row.student_full_name}`}
                    onDelete={() => deleteMutation.mutate(row.id)}
                    disabled={deleteMutation.isPending}
                />
            ),
        },
    ];
}

import { StatusBadge } from "./status-badge";

export function TeacherBehaviorView() {
    const { data, isError } = useTeacherNotes();
    const deleteMutation = useDeleteBehaviorNote();
    const columns = createColumns(deleteMutation);

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
