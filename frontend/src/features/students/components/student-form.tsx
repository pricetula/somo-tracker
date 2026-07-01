/**
 * Student Form — create or edit student demographics.
 *
 * Fields: Full Name (required), Gender, DOB, UPI, KNEC#, Class.
 */

"use client";

import * as React from "react";
import { useRouter } from "next/navigation";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { Loader2 } from "lucide-react";

import { useCreateStudent, useUpdateStudent } from "../hooks/use-student-detail";
import { getErrorMessage } from "@/lib/errors";
import type { StudentDetail } from "../types";

// ─── Props ─────────────────────────────────────────────────────────────────

interface StudentFormProps {
    mode: "create" | "edit";
    initialData?: StudentDetail;
    onSuccess?: (id: string) => void;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function StudentForm({ mode, initialData, onSuccess }: StudentFormProps) {
    const router = useRouter();
    const createStudent = useCreateStudent();
    const updateStudent = useUpdateStudent();

    const [fullName, setFullName] = React.useState(initialData?.full_name ?? "");
    const [gender, setGender] = React.useState(initialData?.gender ?? "");
    const [dateOfBirth, setDateOfBirth] = React.useState(initialData?.date_of_birth ?? "");
    const [upiNumber, setUpiNumber] = React.useState(initialData?.upi_number ?? "");
    const [knecNumber, setKnecNumber] = React.useState(initialData?.knec_assessment_number ?? "");
    const [error, setError] = React.useState<string | null>(null);
    const [fieldErrors, setFieldErrors] = React.useState<Record<string, string[]>>({});

    const isSubmitting = createStudent.isPending || updateStudent.isPending;

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setError(null);
        setFieldErrors({});

        if (!fullName.trim()) {
            setError("Full name is required");
            setFieldErrors({ full_name: ["Full name is required"] });
            return;
        }

        try {
            if (mode === "create") {
                const result = await createStudent.mutateAsync({
                    full_name: fullName.trim(),
                    gender: gender || undefined,
                    date_of_birth: dateOfBirth || null,
                    upi_number: upiNumber || null,
                    knec_assessment_number: knecNumber || null,
                });

                if (onSuccess) {
                    onSuccess(result.id);
                } else {
                    router.push(`/students/${result.id}`);
                }
            } else if (initialData) {
                await updateStudent.mutateAsync({
                    id: initialData.id,
                    data: {
                        full_name: fullName.trim(),
                        gender: gender || undefined,
                        date_of_birth: dateOfBirth || null,
                        upi_number: upiNumber || null,
                        knec_assessment_number: knecNumber || null,
                    },
                });

                if (onSuccess) {
                    onSuccess(initialData.id);
                } else {
                    router.push(`/students/${initialData.id}`);
                }
            }
        } catch (err) {
            setError(getErrorMessage(err));
            // If ApiError with field errors, set them
            if (err && typeof err === "object" && "errors" in err) {
                setFieldErrors((err as { errors?: Record<string, string[]> }).errors ?? {});
            }
        }
    };

    return (
        <form onSubmit={handleSubmit} className="space-y-5">
            {error && (
                <div className="text-destructive bg-destructive/10 rounded-md px-3 py-2 text-sm">
                    {error}
                </div>
            )}

            {/* Full Name */}
            <div className="space-y-1.5">
                <Label htmlFor="full_name">
                    Full Name <span className="text-destructive">*</span>
                </Label>
                <Input
                    id="full_name"
                    value={fullName}
                    onChange={(e) => setFullName(e.target.value)}
                    placeholder="e.g. John Kiprop"
                    disabled={isSubmitting}
                />
                {fieldErrors.full_name && (
                    <p className="text-destructive text-xs">{fieldErrors.full_name[0]}</p>
                )}
            </div>

            {/* Gender */}
            <div className="space-y-1.5">
                <Label htmlFor="gender">Gender</Label>
                <Select value={gender} onValueChange={setGender} disabled={isSubmitting}>
                    <SelectTrigger id="gender">
                        <SelectValue placeholder="Select gender" />
                    </SelectTrigger>
                    <SelectContent>
                        <SelectItem value="M">Male</SelectItem>
                        <SelectItem value="F">Female</SelectItem>
                    </SelectContent>
                </Select>
            </div>

            {/* Date of Birth */}
            <div className="space-y-1.5">
                <Label htmlFor="date_of_birth">Date of Birth</Label>
                <Input
                    id="date_of_birth"
                    type="date"
                    value={dateOfBirth}
                    onChange={(e) => setDateOfBirth(e.target.value)}
                    disabled={isSubmitting}
                />
            </div>

            {/* UPI Number */}
            <div className="space-y-1.5">
                <Label htmlFor="upi_number">UPI Number</Label>
                <Input
                    id="upi_number"
                    value={upiNumber}
                    onChange={(e) => setUpiNumber(e.target.value)}
                    placeholder="e.g. UP123456789"
                    disabled={isSubmitting}
                />
                {fieldErrors.upi_number && (
                    <p className="text-destructive text-xs">{fieldErrors.upi_number[0]}</p>
                )}
            </div>

            {/* KNEC Assessment Number */}
            <div className="space-y-1.5">
                <Label htmlFor="knec_number">KNEC Assessment Number</Label>
                <Input
                    id="knec_number"
                    value={knecNumber}
                    onChange={(e) => setKnecNumber(e.target.value)}
                    placeholder="e.g. KNEC123456"
                    disabled={isSubmitting}
                />
            </div>

            {/* Submit */}
            <div className="flex items-center gap-3 pt-2">
                <Button type="submit" disabled={isSubmitting}>
                    {isSubmitting ? (
                        <>
                            <Loader2 className="mr-1.5 size-4 animate-spin" />
                            {mode === "create" ? "Creating…" : "Saving…"}
                        </>
                    ) : mode === "create" ? (
                        "Create Student"
                    ) : (
                        "Save Changes"
                    )}
                </Button>
                <Button
                    type="button"
                    variant="ghost"
                    onClick={() => router.back()}
                    disabled={isSubmitting}
                >
                    Cancel
                </Button>
            </div>
        </form>
    );
}
