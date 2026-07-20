/**
 * FinanceDetail — displays and edits a finance staff member's profile.
 *
 * Rendered both on the full page /finance/[id] and inside a
 * modal sheet when client-navigated from the finance listing.
 *
 * All forms are validated before submission.
 */

"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
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
import { useFinanceDetail, useUpdateFinance, useDeleteFinanceStaff } from "../hooks/use-finance";

interface FinanceDetailProps {
    id: string;
}

export function FinanceDetail({ id }: FinanceDetailProps) {
    const router = useRouter();
    const { data: member, isLoading, isError, error } = useFinanceDetail(id);
    const updateMutation = useUpdateFinance();
    const deleteMutation = useDeleteFinanceStaff();

    const [fullName, setFullName] = useState("");
    const [nameError, setNameError] = useState("");
    const [hasInitialized, setHasInitialized] = useState(false);

    // Initialise form fields when data loads.
    if (member && !hasInitialized) {
        setFullName(member.full_name ?? "");
        setHasInitialized(true);
    }

    function validate(): boolean {
        if (!fullName.trim()) {
            setNameError("Full name is required");
            return false;
        }
        setNameError("");
        return true;
    }

    const handleDelete = async () => {
        try {
            await deleteMutation.mutateAsync(id);
            router.push("/finance");
        } catch {
            // Error handled by the hook
        }
    };

    function handleSave() {
        if (!validate()) return;

        updateMutation.mutate({
            userId: id,
            payload: { full_name: fullName.trim() },
        });
    }

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
    if (!member) {
        return <p className="text-muted-foreground py-4">Finance staff not found.</p>;
    }

    return (
        <div className="space-y-6 py-2">
            {/* Status badge */}
            <div className="flex items-center justify-between">
                <h2 className="text-lg font-semibold">Profile</h2>
                <Badge
                    variant="secondary"
                    className={
                        member.is_active
                            ? "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400"
                            : "bg-muted text-muted-foreground"
                    }
                >
                    {member.is_active ? "Active" : "Inactive"}
                </Badge>
            </div>

            {/* Read-only email */}
            <div className="space-y-1.5">
                <Label>Email</Label>
                <p className="text-muted-foreground text-sm">{member.email}</p>
            </div>

            {/* Editable full name */}
            <div className="space-y-1.5">
                <Label htmlFor="full-name">Full Name</Label>
                <Input
                    id="full-name"
                    value={fullName}
                    onChange={(e) => {
                        setFullName(e.target.value);
                        if (nameError) setNameError("");
                    }}
                    placeholder="Full name"
                    aria-invalid={!!nameError}
                />
                {nameError && <p className="text-destructive text-sm">{nameError}</p>}
            </div>

            {/* Error from mutation */}
            {updateMutation.error && (
                <p className="text-destructive text-sm">{getErrorMessage(updateMutation.error)}</p>
            )}

            {/* Success feedback */}
            {updateMutation.isSuccess && (
                <p className="text-sm text-emerald-600">Finance staff updated successfully.</p>
            )}

            {/* Save button */}
            <Button onClick={handleSave} disabled={updateMutation.isPending} className="w-full">
                {updateMutation.isPending ? "Saving…" : "Save Changes"}
            </Button>

            {/* Delete button */}
            <AlertDialog>
                <AlertDialogTrigger asChild>
                    <Button variant="outline" size="sm" className="text-destructive w-full">
                        <Trash2 className="mr-1.5 size-3.5" />
                        Delete Finance Staff
                    </Button>
                </AlertDialogTrigger>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>Delete Finance Staff</AlertDialogTitle>
                        <AlertDialogDescription>
                            Are you sure you want to delete &ldquo;{member.full_name}&rdquo;? This
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
