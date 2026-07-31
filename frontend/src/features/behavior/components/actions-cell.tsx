"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
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
import { Loader2 } from "lucide-react";
import { useReviewBehaviorNote } from "../hooks/use-behavior";
import { type PendingNoteItem } from "@/lib/api/behavior";

export function ActionsCell({ note }: { note: PendingNoteItem }) {
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
