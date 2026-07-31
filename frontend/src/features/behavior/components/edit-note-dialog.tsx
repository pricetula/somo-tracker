"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogFooter,
    DialogClose,
} from "@/components/ui/dialog";
import { getErrorMessage } from "@/lib/errors";
import { updateBehaviorNote, type BehaviorNote } from "@/lib/api/behavior";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

export function EditNoteDialog({
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
