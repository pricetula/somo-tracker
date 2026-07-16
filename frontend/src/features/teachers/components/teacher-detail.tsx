/**
 * TeacherDetail — displays and edits a teacher's profile.
 *
 * Rendered both on the full page /teachers/[id] and inside a
 * modal sheet when client-navigated from the teachers listing.
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
import { useTeacherDetail, useUpdateTeacher } from "../hooks/use-teachers";

interface TeacherDetailProps {
    id: string;
}

export function TeacherDetail({ id }: TeacherDetailProps) {
    const { data: teacher, isLoading, isError, error } = useTeacherDetail(id);
    const updateMutation = useUpdateTeacher();

    const [fullName, setFullName] = useState("");
    const [tscNumber, setTscNumber] = useState("");
    const [knecAssessor, setKnecAssessor] = useState("");
    const [nameError, setNameError] = useState("");
    const [hasInitialized, setHasInitialized] = useState(false);

    // Initialise form fields when teacher data loads.
    if (teacher && !hasInitialized) {
        setFullName(teacher.full_name ?? "");
        setTscNumber(teacher.tsc_number ?? "");
        setKnecAssessor(teacher.knec_panel_assessor_id ?? "");
        setHasInitialized(true);
    }

    function validate(): boolean {
        let valid = true;
        if (!fullName.trim()) {
            setNameError("Full name is required");
            valid = false;
        } else {
            setNameError("");
        }
        return valid;
    }

    function handleSave() {
        if (!validate()) return;

        updateMutation.mutate({
            userId: id,
            payload: {
                full_name: fullName.trim() || undefined,
                tsc_number: tscNumber.trim() || null,
                knec_panel_assessor_id: knecAssessor.trim() || null,
            },
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
                <Skeleton className="h-5 w-32" />
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
    if (!teacher) {
        return <p className="text-muted-foreground py-4">Teacher not found.</p>;
    }

    return (
        <div className="space-y-6 py-2">
            {/* Status badge */}
            <div className="flex items-center justify-between">
                <h2 className="text-lg font-semibold">Profile</h2>
                <Badge
                    variant="secondary"
                    className={
                        teacher.is_active
                            ? "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400"
                            : "bg-muted text-muted-foreground"
                    }
                >
                    {teacher.is_active ? "Active" : "Inactive"}
                </Badge>
            </div>

            {/* Read-only email */}
            <div className="space-y-1.5">
                <Label>Email</Label>
                <p className="text-muted-foreground text-sm">{teacher.email}</p>
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

            {/* Editable TSC number */}
            <div className="space-y-1.5">
                <Label htmlFor="tsc-number">TSC Number</Label>
                <Input
                    id="tsc-number"
                    value={tscNumber}
                    onChange={(e) => setTscNumber(e.target.value)}
                    placeholder="e.g. TSC123456"
                />
            </div>

            {/* Editable KNEC Panel Assessor ID */}
            <div className="space-y-1.5">
                <Label htmlFor="knec-assessor">KNEC Panel Assessor ID</Label>
                <Input
                    id="knec-assessor"
                    value={knecAssessor}
                    onChange={(e) => setKnecAssessor(e.target.value)}
                    placeholder="e.g. KNEC-12345"
                />
            </div>

            {/* Error from mutation */}
            {updateMutation.error && (
                <p className="text-destructive text-sm">{getErrorMessage(updateMutation.error)}</p>
            )}

            {/* Success feedback */}
            {updateMutation.isSuccess && (
                <p className="text-sm text-emerald-600">Teacher updated successfully.</p>
            )}

            {/* Save button */}
            <Button onClick={handleSave} disabled={updateMutation.isPending} className="w-full">
                {updateMutation.isPending ? "Saving…" : "Save Changes"}
            </Button>
        </div>
    );
}
