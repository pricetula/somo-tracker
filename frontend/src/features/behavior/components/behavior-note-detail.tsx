/**
 * BehaviorNoteDetail — full detail view of a single behavior note with edit and review actions.
 */

"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { format } from "date-fns";
import { Pencil, Trash2 } from "lucide-react";

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
import {
    getBehaviorNote,
    updateBehaviorNote,
    deleteBehaviorNote,
    type BehaviorNote,
} from "@/lib/api/behavior";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
    AlertDialogTrigger,
} from "@/components/ui/alert-dialog";

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
                        <p className="text-destructive">{getErrorMessage(updateMutation.error)}</p>
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
    const router = useRouter();
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
    const [deleteOpen, setDeleteOpen] = useState(false);

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
                    <p className="text-muted-foreground">
                        Student ID: {note.student_id.slice(0, 8)}… &mdash; Category:{" "}
                        {note.category_id.slice(0, 8)}… &mdash; Date: {note.date}
                    </p>
                </div>
                <div className="flex items-center gap-2">
                    {note.status === "PENDING_REVIEW" && (
                        <Button variant="outline" size="sm" onClick={() => setEditOpen(true)}>
                            <Pencil className="mr-1.5 size-3.5" />
                            Edit
                        </Button>
                    )}
                    <AlertDialog open={deleteOpen} onOpenChange={setDeleteOpen}>
                        <AlertDialogTrigger asChild>
                            <Button variant="outline" size="sm" className="text-destructive">
                                <Trash2 className="size-3.5" />
                            </Button>
                        </AlertDialogTrigger>
                        <AlertDialogContent>
                            <AlertDialogHeader>
                                <AlertDialogTitle>Delete Behavior Note</AlertDialogTitle>
                                <AlertDialogDescription>
                                    Are you sure you want to delete this behavior note? This action
                                    cannot be undone.
                                </AlertDialogDescription>
                            </AlertDialogHeader>
                            <AlertDialogFooter>
                                <AlertDialogCancel>Cancel</AlertDialogCancel>
                                <AlertDialogAction
                                    variant="destructive"
                                    onClick={async () => {
                                        try {
                                            await deleteBehaviorNote(noteId);
                                            toast.success("Behavior note deleted");
                                            router.push("/behavior");
                                        } catch {
                                            // handled by catch below
                                        }
                                    }}
                                >
                                    Delete
                                </AlertDialogAction>
                            </AlertDialogFooter>
                        </AlertDialogContent>
                    </AlertDialog>
                </div>
            </div>

            {/* ── Description ──────────────────────────────────────────── */}
            <div className="space-y-2">
                <h2 className="text-foreground font-medium">Description</h2>
                <p className="text-muted-foreground bg-muted/30 rounded-md px-4 py-3">
                    {note.description}
                </p>
            </div>

            {/* ── Details ────────────────────────────────────────────────── */}
            <div className="grid grid-cols-2 gap-4">
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
                        {note.created_at
                            ? format(new Date(note.created_at), "MMM d, yyyy, h:mm a")
                            : "—"}
                    </p>
                </div>
                {note.reviewed_by_id && (
                    <div>
                        <span className="text-muted-foreground">Reviewed By</span>
                        <p className="font-medium">
                            {note.reviewed_by_id.slice(0, 8)}…
                            {note.reviewed_at &&
                                ` at ${format(new Date(note.reviewed_at), "MMM d, yyyy, h:mm a")}`}
                        </p>
                    </div>
                )}
            </div>

            <EditNoteDialog note={note} open={editOpen} onOpenChange={setEditOpen} />
        </div>
    );
}
