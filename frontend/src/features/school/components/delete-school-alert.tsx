"use client";

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
import { useDeleteSchool } from "../hooks/use-schools";
import { type SchoolWithMemberCount } from "../types";

export function DeleteSchoolAlert({
    school,
    open,
    onOpenChange,
}: {
    school: SchoolWithMemberCount;
    open: boolean;
    onOpenChange: (open: boolean) => void;
}) {
    const deleteMutation = useDeleteSchool();

    return (
        <AlertDialog open={open} onOpenChange={onOpenChange}>
            <AlertDialogContent>
                <AlertDialogHeader>
                    <AlertDialogTitle>Delete School</AlertDialogTitle>
                    <AlertDialogDescription>
                        Are you sure you want to delete <strong>{school.name}</strong>? This cannot
                        be undone. All data associated with this school will be permanently removed.
                    </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                    <AlertDialogCancel>Cancel</AlertDialogCancel>
                    <AlertDialogAction
                        onClick={() => deleteMutation.mutate(school.id)}
                        disabled={deleteMutation.isPending}
                        className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                    >
                        {deleteMutation.isPending ? "Deleting…" : "Delete"}
                    </AlertDialogAction>
                </AlertDialogFooter>
            </AlertDialogContent>
        </AlertDialog>
    );
}
