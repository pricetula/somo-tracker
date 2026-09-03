/**
 * NurseDetail — displays and edits a nurse's profile.
 *
 * Rendered both on the full page /nurses/[id] and inside a
 * modal sheet when client-navigated from the nurses listing.
 *
 * All forms are validated before submission.
 */

"use client";

import React from "react";
import { Loader2, Trash2 } from "lucide-react";
import { z } from "zod";
import { toast } from "sonner";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import {
    Form,
    FormControl,
    FormField,
    FormItem,
    FormLabel,
    FormMessage,
} from "@/components/ui/form";
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
import { getErrorMessage } from "@/lib/errors";
import { useNurseDetail, useUpdateNurse, useDeleteNurse } from "../hooks/use-nurses";

interface NurseDetailProps {
    id: string;
}

const nurseDetailSchema = z.object({
    fullName: z.string().trim().min(2, "Full name required with a minimum of 2 characters"),
});

type NurseDetailSchema = z.infer<typeof nurseDetailSchema>;

export function NurseDetail({ id }: NurseDetailProps) {
    const router = useRouter();
    const { data: nurse, isLoading, isError, error } = useNurseDetail(id);
    const updateMutation = useUpdateNurse();
    const deleteMutation = useDeleteNurse();

    const form = useForm<NurseDetailSchema>({
        resolver: zodResolver(nurseDetailSchema),
        defaultValues: {
            fullName: "",
        },
    });

    React.useEffect(() => {
        if (isError) {
            toast.error(getErrorMessage(error));
        }
    }, [isError, error]);

    React.useEffect(() => {
        if (updateMutation.error) {
            toast.error(getErrorMessage(updateMutation.error));
        }
        if (updateMutation.isSuccess) {
            router.back();
            toast.success("Nurse updated successfully.");
        }
    }, [updateMutation, router]);

    const onSubmit = React.useCallback(
        (values: NurseDetailSchema) => {
            updateMutation.mutate({
                userId: id,
                payload: { full_name: values.fullName.trim() },
            });
        },
        [updateMutation, id]
    );

    React.useEffect(() => {
        if (nurse?.full_name) {
            form.setValue("fullName", nurse.full_name);
        }
    }, [nurse, form]);

    const handleDelete = async () => {
        try {
            await deleteMutation.mutateAsync(id);
            router.back();
        } catch {
            // Error handled by the hook
        }
    };

    // ── Loading state ─────────────────────────────────────────────────────
    if (isLoading) {
        return (
            <div className="space-y-4 py-4">
                <Skeleton className="h-5 w-32" />
                <Skeleton className="h-9 w-full" />
                <Skeleton className="h-5 w-24" />
                <Skeleton className="h-9 w-full" />
            </div>
        );
    }

    // ── Error state ───────────────────────────────────────────────────────
    if (isError) {
        return null;
    }

    // ── Not found state ───────────────────────────────────────────────────
    if (!nurse) {
        return <p className="text-muted-foreground py-4">Nurse not found.</p>;
    }

    return (
        <div className="space-y-6 py-2">
            {/* Read-only email */}
            <div className="space-y-1.5">
                <Label>Email</Label>
                <p className="text-muted-foreground">{nurse.email}</p>
            </div>

            {/* Editable fields form */}
            <Form {...form}>
                <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
                    {/* Editable full name */}
                    <FormField
                        control={form.control}
                        name="fullName"
                        render={({ field }) => (
                            <FormItem>
                                <FormLabel htmlFor="full-name">Full Name</FormLabel>
                                <FormControl>
                                    <Input
                                        id="full-name"
                                        placeholder="Full name"
                                        autoFocus
                                        {...field}
                                    />
                                </FormControl>
                                <FormMessage />
                            </FormItem>
                        )}
                    />

                    <footer className="flex gap-4">
                        {/* Save button */}
                        <Button type="submit" disabled={updateMutation.isPending}>
                            {updateMutation.isPending ? (
                                <>
                                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                                    Saving…
                                </>
                            ) : (
                                "Save Changes"
                            )}
                        </Button>
                        {/* Delete button */}
                        <AlertDialog>
                            <AlertDialogTrigger
                                render={
                                    <Button variant="outline" className="text-destructive">
                                        <Trash2 className="mr-1.5 size-3.5" />
                                        Delete Nurse
                                    </Button>
                                }
                            />
                            <AlertDialogContent>
                                <AlertDialogHeader>
                                    <AlertDialogTitle>Delete Nurse</AlertDialogTitle>
                                    <AlertDialogDescription>
                                        Are you sure you want to delete &ldquo;{nurse.full_name}
                                        &rdquo;? This action cannot be undone.
                                    </AlertDialogDescription>
                                </AlertDialogHeader>
                                <AlertDialogFooter>
                                    <AlertDialogCancel>Cancel</AlertDialogCancel>
                                    <AlertDialogAction
                                        variant="destructive"
                                        onClick={handleDelete}
                                        disabled={deleteMutation.isPending}
                                    >
                                        {deleteMutation.isPending ? "Deleting…" : "Delete"}
                                    </AlertDialogAction>
                                </AlertDialogFooter>
                            </AlertDialogContent>
                        </AlertDialog>
                    </footer>
                </form>
            </Form>
        </div>
    );
}
