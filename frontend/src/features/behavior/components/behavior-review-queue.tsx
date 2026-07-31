"use client";

import { DataTable } from "@/components/shared/data-table";
import { type DataTableColumn } from "@/components/shared/data-table/types";
import { useBehaviorPendingQueue, useDeleteBehaviorNote } from "../hooks/use-behavior";
import { type PendingNoteItem } from "@/lib/api/behavior";

const columns: DataTableColumn<PendingNoteItem>[] = [
    {
        id: "student",
        header: "Student",
        cell: (row) => <StudentCell note={row} />,
    },
    {
        id: "description",
        header: "Description",
        width: "minmax(200px, 1fr)",
        cell: (row) => <DescriptionCell note={row} />,
    },
    {
        id: "actions",
        header: "",
        width: "240px",
        align: "right",
        cell: (row) => (
            <div className="flex items-center justify-end gap-1">
                <ActionsCell note={row} />
                <RowActionsCell note={row} />
            </div>
        ),
    },
];

import { StudentCell } from "./student-cell";
import { ActionsCell } from "./actions-cell";
import { DescriptionCell } from "./description-cell";
import { RowActionsCell } from "./row-actions-cell";

export function BehaviorReviewQueue() {
    const { data, isError } = useBehaviorPendingQueue();
    const deleteMutation = useDeleteBehaviorNote();

    if (isError) {
        return (
            <div className="border-destructive/50 text-destructive rounded-md border p-4">
                Failed to load behavior notes.
            </div>
        );
    }

    const notes = data?.items ?? [];

    return (
        <div className="space-y-4">
            <DataTable
                isCheckable
                queryKey={["behavior", "queue"]}
                queryFn={() => Promise.resolve({ items: notes, total: notes.length })}
                columns={columns}
                getRowId={(row) => row.id}
                deleteFn={(id) => deleteMutation.mutateAsync(String(id))}
                emptyState="No behavior notes waiting for review."
                noResultsState="No notes match your search."
                pageSize={50}
                height={500}
            />
        </div>
    );
}
