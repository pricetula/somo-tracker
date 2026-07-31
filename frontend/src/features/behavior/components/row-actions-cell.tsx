"use client";

import { RowActions } from "@/components/shared/data-table/row-actions";
import { type RowAction } from "@/components/shared/data-table/row-actions";
import { Trash2 } from "lucide-react";
import { useDeleteBehaviorNote } from "../hooks/use-behavior";
import { type PendingNoteItem } from "@/lib/api/behavior";

export function RowActionsCell({ note }: { note: PendingNoteItem }) {
    const deleteMutation = useDeleteBehaviorNote();

    const rowActions: RowAction[] = [
        {
            label: "Delete",
            icon: Trash2,
            destructive: true,
            onClick: () => deleteMutation.mutate(note.id),
        },
    ];

    return (
        <RowActions
            rowId={note.id}
            label={`behavior note for ${note.student_full_name}`}
            actions={rowActions}
        />
    );
}
