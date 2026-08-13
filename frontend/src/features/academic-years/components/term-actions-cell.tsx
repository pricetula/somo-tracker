"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
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
    AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { activateTerm, deleteTerm } from "@/lib/api/academic-terms";
import { type AcademicTerm } from "@/lib/api/academic-terms";
import { getErrorMessage } from "@/lib/errors";

export function TermActionsCell({
    term,
    onEdit,
}: {
    term: AcademicTerm;
    onEdit: (term: AcademicTerm) => void;
}) {
    const queryClient = useQueryClient();

    const activateMutation = useMutation({
        mutationFn: () => activateTerm(term.id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["academic-terms"] });
            queryClient.invalidateQueries({ queryKey: ["academic-years"] });
            toast.success(`"${term.name}" activated as the current term.`);
        },
        onError: (err) => toast.error(getErrorMessage(err)),
    });

    const deleteMutation = useMutation({
        mutationFn: () => deleteTerm(term.id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["academic-terms"] });
            queryClient.invalidateQueries({ queryKey: ["academic-years"] });
            toast.success(`"${term.name}" deleted.`);
        },
        onError: (err) => toast.error(getErrorMessage(err)),
    });

    return (
        <div className="flex items-center justify-end gap-2">
            {!term.is_current && (
                <Button
                    variant="outline"
                    size="sm"
                    onClick={() => activateMutation.mutate()}
                    disabled={activateMutation.isPending}
                >
                    Activate
                </Button>
            )}
            <Button variant="outline" size="sm" onClick={() => onEdit(term)}>
                Edit
            </Button>
            <AlertDialog>
                <AlertDialogTrigger asChild>
                    <Button variant="outline" size="sm" className="text-destructive">
                        Delete
                    </Button>
                </AlertDialogTrigger>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>Delete Term</AlertDialogTitle>
                        <AlertDialogDescription>
                            Are you sure you want to delete &ldquo;{term.name}&rdquo;? This action
                            cannot be undone.
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel>Cancel</AlertDialogCancel>
                        <AlertDialogAction
                            onClick={() => deleteMutation.mutate()}
                            disabled={deleteMutation.isPending}
                        >
                            Delete
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </div>
    );
}
