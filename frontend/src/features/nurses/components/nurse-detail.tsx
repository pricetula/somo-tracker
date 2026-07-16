/**
 * NurseDetail — displays and edits a nurse's profile.
 *
 * Rendered both on the full page /nurses/[id] and inside a
 * modal sheet when client-navigated from the nurses listing.
 *
 * All forms are validated before submission.
 */

"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { getErrorMessage } from "@/lib/errors";
import { useNurseDetail, useUpdateNurse } from "../hooks/use-nurses";

interface NurseDetailProps {
    id: string;
}

export function NurseDetail({ id }: NurseDetailProps) {
    const { data: nurse, isLoading, isError, error } = useNurseDetail(id);
    const updateMutation = useUpdateNurse();

    const [fullName, setFullName] = useState("");
    const [nameError, setNameError] = useState("");
    const [hasInitialized, setHasInitialized] = useState(false);

    // Initialise form fields when nurse data loads.
    if (nurse && !hasInitialized) {
        setFullName(nurse.full_name ?? "");
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
                <p className="text-sm text-emerald-600">Nurse updated successfully.</p>
            )}

            {/* Save button */}
            <Button onClick={handleSave} disabled={updateMutation.isPending} className="w-full">
                {updateMutation.isPending ? "Saving…" : "Save Changes"}
            </Button>
        </div>
    );
}
