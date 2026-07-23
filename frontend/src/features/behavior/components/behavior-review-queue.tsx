/**
 * BehaviorReviewQueue — admin view of pending behavior notes.
 *
 * Shows cards with approve/reject actions. Urgent items visually distinguished.
 * Approve is a single click; reject requires AlertDialog confirmation.
 */

"use client";

import { useState } from "react";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
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

export function BehaviorReviewQueue() {
    const { data, isLoading, isError } = useBehaviorPendingQueue();
    const reviewNote = useReviewBehaviorNote();

    const [rejectDialog, setRejectDialog] = useState<{
        open: boolean;
        noteId: string;
        adminNote: string;
    }>({ open: false, noteId: "", adminNote: "" });

    const handleApprove = (noteId: string) => {
        reviewNote.mutate({ noteId, payload: { decision: "APPROVED" } });
    };

    const handleReject = () => {
        reviewNote.mutate({
            noteId: rejectDialog.noteId,
            payload: { decision: "REJECTED", admin_note: rejectDialog.adminNote || undefined },
        });
        setRejectDialog({ open: false, noteId: "", adminNote: "" });
    };

    if (isLoading) {
        return (
            <div className="space-y-4">
                {Array.from({ length: 4 }).map((_, i) => (
                    <Skeleton key={i} className="h-32 w-full rounded-lg" />
                ))}
            </div>
        );
    }

    if (isError) {
        return (
            <div className="border-destructive/50 text-destructive rounded-md border p-4">
                Failed to load behavior notes.
            </div>
        );
    }

    const notes = data?.notes ?? [];

    if (notes.length === 0) {
        return (
            <div className="text-muted-foreground flex items-center justify-center py-16">
                <p>No behavior notes waiting for review.</p>
            </div>
        );
    }

    return (
        <div className="space-y-4">
            <h2 className="text-xl font-semibold">Behavior Review Queue ({notes.length})</h2>

            {notes.map((note) => (
                <div
                    key={note.id}
                    className={`rounded-lg border p-4 ${note.is_urgent ? "border-l-4 border-l-red-500" : ""}`}
                >
                    <div className="flex items-start justify-between">
                        <div className="space-y-2">
                            <div className="flex items-center gap-2">
                                <span className="font-semibold">{note.student_full_name}</span>
                                <Badge variant="outline">{note.class_name}</Badge>
                                <Badge>{note.category_name}</Badge>
                                {note.is_urgent && (
                                    <Badge variant="destructive" className="gap-1">
                                        <AlertTriangle className="h-3 w-3" />
                                        Urgent
                                    </Badge>
                                )}
                            </div>
                            <p className="text-muted-foreground">{note.description}</p>
                            <p className="text-muted-foreground text-xs">
                                By {note.authored_by_name} &middot; {note.date}
                            </p>
                        </div>
                        <div className="flex items-center gap-2">
                            <Button
                                size="sm"
                                onClick={() => handleApprove(note.id)}
                                disabled={reviewNote.isPending}
                            >
                                {reviewNote.isPending ? (
                                    <Loader2 className="mr-1 h-3 w-3 animate-spin" />
                                ) : null}
                                Approve
                            </Button>
                            <Button
                                size="sm"
                                variant="destructive"
                                onClick={() =>
                                    setRejectDialog({ open: true, noteId: note.id, adminNote: "" })
                                }
                                disabled={reviewNote.isPending}
                            >
                                Reject
                            </Button>
                        </div>
                    </div>
                </div>
            ))}

            {/* Rejection confirmation dialog */}
            <AlertDialog
                open={rejectDialog.open}
                onOpenChange={(open) => setRejectDialog((prev) => ({ ...prev, open }))}
            >
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
                        value={rejectDialog.adminNote}
                        onChange={(e) =>
                            setRejectDialog((prev) => ({ ...prev, adminNote: e.target.value }))
                        }
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
