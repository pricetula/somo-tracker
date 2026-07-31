"use client";

import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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
import { useUpdateSchool } from "../hooks/use-schools";
import { getErrorMessage } from "@/lib/errors";
import { type SchoolWithMemberCount } from "../types";
import * as React from "react";

export function EditSchoolDialog({
    school,
    open,
    onOpenChange,
}: {
    school: SchoolWithMemberCount;
    open: boolean;
    onOpenChange: (open: boolean) => void;
}) {
    const [name, setName] = React.useState(school.name);
    const updateMutation = useUpdateSchool();

    // Reset name when dialog opens — this is an initialization effect that syncs
    // external state (school data) into local state. The alternative would be
    // key-based remounting, which would reset scroll and focus state.
    React.useEffect(() => {
        // eslint-disable-next-line react-hooks/set-state-in-effect
        if (open) setName(school.name);
    }, [open, school.name]);

    async function handleSave() {
        if (!name.trim()) return;
        updateMutation.mutate(
            { id: school.id, payload: { name: name.trim() } },
            {
                onSuccess: () => onOpenChange(false),
            }
        );
    }

    return (
        <AlertDialog open={open} onOpenChange={onOpenChange}>
            <AlertDialogContent>
                <AlertDialogHeader>
                    <AlertDialogTitle>Edit School</AlertDialogTitle>
                    <AlertDialogDescription>
                        Update the name for &ldquo;{school.name}&rdquo;
                    </AlertDialogDescription>
                </AlertDialogHeader>
                <div className="space-y-2 py-2">
                    <Label htmlFor="edit-school-name">School Name</Label>
                    <Input
                        id="edit-school-name"
                        value={name}
                        onChange={(e) => setName(e.target.value)}
                        autoFocus
                    />
                    {updateMutation.error && (
                        <p className="text-destructive">{getErrorMessage(updateMutation.error)}</p>
                    )}
                </div>
                <AlertDialogFooter>
                    <AlertDialogCancel>Cancel</AlertDialogCancel>
                    <AlertDialogAction
                        onClick={handleSave}
                        disabled={!name.trim() || updateMutation.isPending}
                    >
                        {updateMutation.isPending ? "Saving…" : "Save"}
                    </AlertDialogAction>
                </AlertDialogFooter>
            </AlertDialogContent>
        </AlertDialog>
    );
}
