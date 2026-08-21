"use client";

import { Trash2 } from "lucide-react";
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
import { useDeleteStream } from "@/features/streams";
import { type Stream } from "@/features/streams";

export function DeleteStreamAlert({ stream }: { stream: Stream }) {
    const deleteStream = useDeleteStream();

    return (
        <AlertDialog>
            <AlertDialogTrigger>
                <Button
                    size="icon"
                    variant="ghost"
                    className="text-destructive hover:text-destructive h-8 w-8"
                >
                    <Trash2 className="h-4 w-4" />
                    <span className="sr-only">Delete {stream.name}</span>
                </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
                <AlertDialogHeader>
                    <AlertDialogTitle>Delete Stream</AlertDialogTitle>
                    <AlertDialogDescription>
                        Are you sure you want to delete the stream <strong>{stream.name}</strong>?
                        This action cannot be undone.
                    </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                    <AlertDialogCancel disabled={deleteStream.isPending}>Cancel</AlertDialogCancel>
                    <AlertDialogAction
                        onClick={() => deleteStream.mutate(stream.id)}
                        disabled={deleteStream.isPending}
                        className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                    >
                        {deleteStream.isPending ? "Deleting…" : "Delete"}
                    </AlertDialogAction>
                </AlertDialogFooter>
            </AlertDialogContent>
        </AlertDialog>
    );
}
