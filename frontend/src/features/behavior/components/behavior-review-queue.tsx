/**
 * BehaviorReviewQueue — admin view of pending behavior notes.
 *
 * Uses the shared DataTable component with approve/reject actions per row.
 */

"use client";

import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
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
import { Textarea } from "@/components/ui/textarea";
import { Loader2, AlertTriangle } from "lucide-react";

import { useBehaviorPendingQueue, useReviewBehaviorNote } from "../hooks/use-behavior";
import type { PendingNoteItem } from "@/lib/api/behavior";

// ─── Student Cell with badges ─────────────────────────────────────────────

function StudentCell({ note }: { note: PendingNoteItem }) {
    return (
        <div className="flex items-center gap-2">
            <span className="font-medium">{note.student_full_name}</span>
            <Badge variant="outline" className="text-[10px]">
                {note.class_name}
            </Badge>
            <Badge className="text-[10px]">{note.category_name}</Badge>
            {note.is_urgent && (
                <Badge variant="destructive" className="gap-1 text-[10px]">
                    <AlertTriangle className="h-3 w-3" />
                    Urgent
                </Badge>
            )}
        </div>
    );
}

// ─── Actions cell ─────────────────────────────────────────────────────────

function ActionsCell({ note }: { note: PendingNoteItem }) {
    const reviewNote = useReviewBehaviorNote();

    const [rejectDialogOpen, setRejectDialogOpen] = useState(false);
    const [adminNote, setAdminNote] = useState("");

    const handleApprove = () => {
        reviewNote.mutate({ noteId: note.id, payload: { decision: "APPROVED" } });
    };

    const handleReject = () => {
        reviewNote.mutate({
            noteId: note.id,
            payload: { decision: "REJECTED", admin_note: adminNote || undefined },
        });
        setRejectDialogOpen(false);
        setAdminNote("");
    };

    return (
        <div className="flex items-center gap-2">
            <Button size="sm" onClick={handleApprove} disabled={reviewNote.isPending}>
                {reviewNote.isPending ? <Loader2 className="mr-1 h-3 w-3 animate-spin" /> : null}
                Approve
            </Button>
            <Button
                size="sm"
                variant="destructive"
                onClick={() => setRejectDialogOpen(true)}
                disabled={reviewNote.isPending}
            >
                Reject
            </Button>

            <AlertDialog open={rejectDialogOpen} onOpenChange={setRejectDialogOpen}>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>Reject behavior note?</AlertDialogTitle>
                        <AlertDialogDescription>
                            This will discard the teacher&apos;s input. You can provide a note
                            explaining why.
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <Textarea
                        placeholder="Optional reason for rejection..."
                        value={adminNote}
                        onChange={(e) => setAdminNote(e.target.value)}
                    />
                    <AlertDialogFooter>
                        <AlertDialogCancel>Cancel</AlertDialogCancel>
                        <AlertDialogAction onClick={handleReject} className="bg-destructive">
                            Reject
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </div>
    );
}

// ─── Description Cell ─────────────────────────────────────────────────────

function DescriptionCell({ note }: { note: PendingNoteItem }) {
    return (
        <div className="space-y-1">
            <p className="text-muted-foreground line-clamp-2 text-xs">{note.description}</p>
            <p className="text-muted-foreground text-[10px]">
                By {note.authored_by_name} &middot; {note.date}
            </p>
        </div>
    );
}

// ─── Columns ──────────────────────────────────────────────────────────────

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
        width: "200px",
        align: "right",
        cell: (row) => <ActionsCell note={row} />,
    },
];

// ─── Component ────────────────────────────────────────────────────────────

export function BehaviorReviewQueue() {
    const { data, isError } = useBehaviorPendingQueue();

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
                queryKey={["behavior", "queue"]}
                queryFn={() => Promise.resolve({ items: notes, total: notes.length })}
                columns={columns}
                getRowId={(row) => row.id}
                emptyState="No behavior notes waiting for review."
                noResultsState="No notes match your search."
                pageSize={50}
                height={500}
            />
        </div>
    );
}
