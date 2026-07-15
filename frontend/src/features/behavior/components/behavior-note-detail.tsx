/**
 * BehaviorNoteDetail — full detail view of a single behavior note with edit and review actions.
 */

"use client";

import { useState } from "react";
import { useParams } from "next/navigation";
import { Pencil } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogFooter,
    DialogClose,
} from "@/components/ui/dialog";
import { getErrorMessage } from "@/lib/errors";
import { getBehaviorNote, updateBehaviorNote, type BehaviorNote } from "@/lib/api/behavior";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

// ─── Status badge helper ──────────────────────────────────────────────────

function statusBadge(status: string) {
    const variants: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
        PENDING_REVIEW: "secondary",
        APPROVED: "default",
        REJECTED: "destructive",
        INCLUDED_IN_REPORT: "outline",
    };
    return <Badge variant={variants[status] ?? "outline"}>{status.replace(/_/g, " ")}</Badge>;
}

// ─── Edit Dialog ──────────────────────────────────────────────────────────

function EditNoteDialog({
    note,
    open,
    onOpenChange,
}: {
    note: BehaviorNote;
    open: boolean;
    onOpenChange: (open: boolean) => void;
}) {
    const [description, setDescription] = useState(note.description);
    const queryClient = useQueryClient();

    const updateMutation = useMutation({
        mutationFn: (desc: string) => updateBehaviorNote(note.id, { description: desc }),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["behavior-note", note.id] });
            queryClient.invalidateQueries({ queryKey: ["behavior-notes"] });
            toast.success("Behavior note updated");
            onOpenChange(false);
        },
        onError: (err) => toast.error(getErrorMessage(err)),
    });

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent>
                <DialogHeader>
                    <DialogTitle>Edit Behavior Note</DialogTitle>
                </DialogHeader>
                <div className="space-y-2 py-2">
                    <Label htmlFor="edit-description">Description</Label>
                    <Input
                        id="edit-description"
                        value={description}
                        onChange={(e) => setDescription(e.target.value)}
                        placeholder="Describe the behavior incident"
                    />
                    {updateMutation.error && (
                        <p className="text-destructive text-sm">
                            {getErrorMessage(updateMutation.error)}
                        </p>
                    )}
                </div>
                <DialogFooter>
                    <DialogClose asChild>
                        <Button variant="outline">Cancel</Button>
                    </DialogClose>
                    <Button
                        onClick={() => updateMutation.mutate(description)}
                        disabled={!description.trim() || updateMutation.isPending}
                    >
                        {updateMutation.isPending ? "Saving…" : "Save"}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}

// ─── Main Component ───────────────────────────────────────────────────────

export function BehaviorNoteDetail() {
    const params = useParams();
    const noteId = params?.id as string;

    const {
        data: note,
        isLoading,
        isError,
        error,
    } = useQuery({
        queryKey: ["behavior-note", noteId],
        queryFn: () => getBehaviorNote(noteId),
        enabled: !!noteId,
    });

    const [editOpen, setEditOpen] = useState(false);

    // ── Loading ──────────────────────────────────────────────────────────
    if (isLoading) {
        return (
            <div className="space-y-4">
                <Skeleton className="h-8 w-48" />
                <Skeleton className="h-6 w-full" />
                <Skeleton className="h-20 w-full" />
            </div>
        );
    }

    // ── Error ────────────────────────────────────────────────────────────
    if (isError) {
        return (
            <Alert variant="destructive">
                <AlertDescription>{getErrorMessage(error)}</AlertDescription>
            </Alert>
        );
    }

    if (!note) {
        return (
            <Alert>
                <AlertDescription>Behavior note not found.</AlertDescription>
            </Alert>
        );
    }

    return (
        <div className="space-y-6">
            {/* ── Header ──────────────────────────────────────────────── */}
            <div className="flex items-center justify-between">
                <div className="space-y-1">
                    <div className="flex items-center gap-3">
                        <h1 className="text-foreground text-2xl font-semibold">Behavior Note</h1>
                        {statusBadge(note.status)}
                    </div>
                    <p className="text-muted-foreground text-sm">
                        Student ID: {note.student_id.slice(0, 8)}… &mdash; Category:{" "}
                        {note.category_id.slice(0, 8)}… &mdash; Date: {note.date}
                    </p>
                </div>
                {note.status === "PENDING_REVIEW" && (
                    <Button variant="outline" size="sm" onClick={() => setEditOpen(true)}>
                        <Pencil className="mr-1.5 size-3.5" />
                        Edit Description
                    </Button>
                )}
            </div>

            {/* ── Description ──────────────────────────────────────────── */}
            <div className="space-y-2">
                <h2 className="text-foreground text-sm font-medium">Description</h2>
                <p className="text-muted-foreground bg-muted/30 rounded-md px-4 py-3 text-sm">
                    {note.description}
                </p>
            </div>

            {/* ── Details ────────────────────────────────────────────────── */}
            <div className="grid grid-cols-2 gap-4 text-sm">
                <div>
                    <span className="text-muted-foreground">Status</span>
                    <p className="font-medium">{statusBadge(note.status)}</p>
                </div>
                <div>
                    <span className="text-muted-foreground">Urgent</span>
                    <p className="font-medium">{note.is_urgent ? "Yes" : "No"}</p>
                </div>
                <div>
                    <span className="text-muted-foreground">Created</span>
                    <p className="font-medium">
                        {note.created_at ? new Date(note.created_at).toLocaleString() : "—"}
                    </p>
                </div>
                {note.reviewed_by_id && (
                    <div>
                        <span className="text-muted-foreground">Reviewed By</span>
                        <p className="font-medium">
                            {note.reviewed_by_id.slice(0, 8)}…
                            {note.reviewed_at &&
                                ` at ${new Date(note.reviewed_at).toLocaleString()}`}
                        </p>
                    </div>
                )}
            </div>

            <EditNoteDialog note={note} open={editOpen} onOpenChange={setEditOpen} />
        </div>
    );
}
