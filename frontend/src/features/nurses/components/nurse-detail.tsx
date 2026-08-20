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
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
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
            router.push("/nurses");
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
        return (
            <Alert variant="destructive">
                <AlertDescription>{getErrorMessage(error)}</AlertDescription>
            </Alert>
        );
    }

    // ── Not found state ───────────────────────────────────────────────────
    if (!nurse) {
        return <p className="text-muted-foreground py-4">Nurse not found.</p>;
    }

    return (
        <div className="space-y-6 py-2">
            {/* Status badge */}
            <div className="flex items-center justify-between">
                <h2 className="text-lg font-semibold">Profile</h2>
                <Badge
                    variant="secondary"
                    className={
                        nurse.is_active
                            ? "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400"
                            : "bg-muted text-muted-foreground"
                    }
                >
                    {nurse.is_active ? "Active" : "Inactive"}
                </Badge>
            </div>

            {/* Read-only email */}
            <div className="space-y-1.5">
                <Label>Email</Label>
                <p className="text-muted-foreground text-sm">{nurse.email}</p>
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

                    {/* Error from mutation */}
                    {updateMutation.error && (
                        <p className="text-destructive text-sm">
                            {getErrorMessage(updateMutation.error)}
                        </p>
                    )}

                    {/* Success feedback */}
                    {updateMutation.isSuccess && (
                        <p className="text-sm text-emerald-600">Nurse updated successfully.</p>
                    )}

                    {/* Save button */}
                    <Button type="submit" disabled={updateMutation.isPending} className="w-full">
                        {updateMutation.isPending ? (
                            <>
                                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                                Saving…
                            </>
                        ) : (
                            "Save Changes"
                        )}
                    </Button>
                </form>
            </Form>

            {/* Delete button */}
            <AlertDialog>
                <AlertDialogTrigger asChild>
                    <Button variant="outline" size="sm" className="text-destructive w-full">
                        <Trash2 className="mr-1.5 size-3.5" />
                        Delete Nurse
                    </Button>
                </AlertDialogTrigger>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>Delete Nurse</AlertDialogTitle>
                        <AlertDialogDescription>
                            Are you sure you want to delete &ldquo;{nurse.full_name}&rdquo;? This
                            action cannot be undone.
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
        </div>
    );
}
