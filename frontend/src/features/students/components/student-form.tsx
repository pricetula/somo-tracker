/**
 * Student Form — create one or more students (batch).
 *
 * Supports adding multiple student entries that are all submitted
 * together via the batch POST /api/v1/students endpoint.
 *
 * Fields per entry: Full Name (required), Gender, DOB, UPI, KNEC#.
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
import { DatePicker } from "@/components/ui/date-picker";
import { Loader2, Plus, X } from "lucide-react";

import { useCreateStudents, useUpdateStudent } from "../hooks/use-student-detail";
import { getErrorMessage } from "@/lib/errors";
import type { StudentDetail, CreateStudentPayload } from "../types";

// ─── Props ─────────────────────────────────────────────────────────────────

interface StudentFormProps {
    mode: "create" | "edit";
    initialData?: StudentDetail;
    onSuccess?: (id: string) => void;
}

interface StudentEntry {
    key: string;
    fullName: string;
    gender: string;
    dateOfBirth: string;
    upiNumber: string;
    knecNumber: string;
}

let entryIdCounter = 0;
function freshEntry(): StudentEntry {
    entryIdCounter += 1;
    return {
        key: `entry-${entryIdCounter}`,
        fullName: "",
        gender: "",
        dateOfBirth: "",
        upiNumber: "",
        knecNumber: "",
    };
}

// ─── Component ─────────────────────────────────────────────────────────────

export function StudentForm({ mode, initialData, onSuccess }: StudentFormProps) {
    const router = useRouter();
    const createStudents = useCreateStudents();
    const updateStudent = useUpdateStudent();

    const [entries, setEntries] = React.useState<StudentEntry[]>([freshEntry()]);
    const [error, setError] = React.useState<string | null>(null);
    const [fieldErrors, setFieldErrors] = React.useState<Record<string, string[]>>({});

    const isSubmitting = createStudents.isPending || updateStudent.isPending;

    const updateEntry = (key: string, patch: Partial<StudentEntry>) => {
        setEntries((prev) => prev.map((e) => (e.key === key ? { ...e, ...patch } : e)));
    };

    const addEntry = () => {
        setEntries((prev) => [...prev, freshEntry()]);
    };

    const removeEntry = (key: string) => {
        setEntries((prev) => prev.filter((e) => e.key !== key));
    };

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setError(null);
        setFieldErrors({});

        if (mode === "edit" && initialData) {
            // Single-student update — unchanged path
            const entry = entries[0];
            try {
                await updateStudent.mutateAsync({
                    id: initialData.id,
                    data: {
                        full_name: entry.fullName.trim(),
                        gender: entry.gender || undefined,
                        date_of_birth: entry.dateOfBirth || null,
                        upi_number: entry.upiNumber || null,
                        knec_assessment_number: entry.knecNumber || null,
                    },
                });
                if (onSuccess) {
                    onSuccess(initialData.id);
                } else {
                    router.push(`/students/${initialData.id}`);
                }
            } catch (err) {
                setError(getErrorMessage(err));
                if (err && typeof err === "object" && "errors" in err) {
                    setFieldErrors((err as { errors?: Record<string, string[]> }).errors ?? {});
                }
            }
            return;
        }

        // ── Batch create ──────────────────────────────────────────────
        // Validate all entries before submitting
        const validEntries: { entry: StudentEntry; idx: number }[] = [];
        const localErrors: Record<string, string[]> = {};

        for (let i = 0; i < entries.length; i++) {
            const e = entries[i];
            if (!e.fullName.trim()) {
                localErrors[`students.${i}.full_name`] = ["Full name is required"];
            } else {
                validEntries.push({ entry: e, idx: i });
            }
        }

        if (validEntries.length === 0) {
            setError("At least one student with a full name is required");
            setFieldErrors(localErrors);
            return;
        }

        const payload: CreateStudentPayload[] = validEntries.map(({ entry }) => ({
            full_name: entry.fullName.trim(),
            gender: entry.gender || undefined,
            date_of_birth: entry.dateOfBirth || null,
            upi_number: entry.upiNumber || null,
            knec_assessment_number: entry.knecNumber || null,
        }));

        try {
            const result = await createStudents.mutateAsync({ students: payload });

            if (result.ids.length === 1 && onSuccess) {
                onSuccess(result.ids[0]);
            } else {
                router.push("/students");
            }
        } catch (err) {
            setError(getErrorMessage(err));
            if (err && typeof err === "object" && "errors" in err) {
                setFieldErrors((err as { errors?: Record<string, string[]> }).errors ?? {});
            }
        }
    };

    // ── Render ──────────────────────────────────────────────────────────
    return (
        <form onSubmit={handleSubmit} className="space-y-6">
            {error && (
                <div className="text-destructive bg-destructive/10 rounded-md px-3 py-2 text-sm">
                    {error}
                </div>
            )}

            {entries.map((entry, index) => (
                <StudentEntryRow
                    key={entry.key}
                    entry={entry}
                    index={index}
                    isSubmitting={isSubmitting}
                    fieldErrors={fieldErrors}
                    canRemove={entries.length > 1}
                    onUpdate={(patch) => updateEntry(entry.key, patch)}
                    onRemove={() => removeEntry(entry.key)}
                />
            ))}

            {mode === "create" && (
                <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={addEntry}
                    disabled={isSubmitting}
                >
                    <Plus className="mr-1.5 size-4" />
                    Add another student
                </Button>
            )}

            {/* Actions */}
            <div className="flex items-center gap-3 pt-2">
                <Button type="submit" disabled={isSubmitting}>
                    {isSubmitting ? (
                        <>
                            <Loader2 className="mr-1.5 size-4 animate-spin" />
                            {mode === "create" ? "Creating…" : "Saving…"}
                        </>
                    ) : mode === "create" ? (
                        `Create ${entries.length > 1 ? `${entries.length} Students` : "Student"}`
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

// ─── Student Entry Row (sub-component) ────────────────────────────────────

interface StudentEntryRowProps {
    entry: StudentEntry;
    index: number;
    isSubmitting: boolean;
    fieldErrors: Record<string, string[]>;
    canRemove: boolean;
    onUpdate: (patch: Partial<StudentEntry>) => void;
    onRemove: () => void;
}

function StudentEntryRow({
    entry,
    index,
    isSubmitting,
    fieldErrors,
    canRemove,
    onUpdate,
    onRemove,
}: StudentEntryRowProps) {
    return (
        <div className="bg-muted/30 space-y-4 rounded-md p-4">
            {/* Header row with entry number and remove button */}
            <div className="flex items-center justify-between gap-2">
                <p className="text-muted-foreground text-sm font-medium">Student {index + 1}</p>
                {canRemove && (
                    <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        className="size-6"
                        onClick={onRemove}
                        disabled={isSubmitting}
                        aria-label={`Remove student ${index + 1}`}
                    >
                        <X className="size-4" />
                    </Button>
                )}
            </div>

            {/* Full Name */}
            <div className="space-y-1.5">
                <Label htmlFor={`full_name_${entry.key}`}>
                    Full Name <span className="text-destructive">*</span>
                </Label>
                <Input
                    id={`full_name_${entry.key}`}
                    value={entry.fullName}
                    onChange={(e) => onUpdate({ fullName: e.target.value })}
                    placeholder="e.g. John Kiprop"
                    disabled={isSubmitting}
                />
                {fieldErrors[`students.${index}.full_name`] && (
                    <p className="text-destructive text-xs">
                        {fieldErrors[`students.${index}.full_name`][0]}
                    </p>
                )}
            </div>

            {/* Gender + DOB row */}
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <div className="space-y-1.5">
                    <Label htmlFor={`gender_${entry.key}`}>Gender</Label>
                    <Select
                        value={entry.gender}
                        onValueChange={(v) => onUpdate({ gender: v })}
                        disabled={isSubmitting}
                    >
                        <SelectTrigger id={`gender_${entry.key}`}>
                            <SelectValue placeholder="Select gender" />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem value="M">Male</SelectItem>
                            <SelectItem value="F">Female</SelectItem>
                        </SelectContent>
                    </Select>
                </div>
                <div className="space-y-1.5">
                    <Label htmlFor={`dob_${entry.key}`}>Date of Birth</Label>
                    <DatePicker
                        id={`dob_${entry.key}`}
                        value={entry.dateOfBirth}
                        onChange={(v) => onUpdate({ dateOfBirth: v })}
                        placeholder="Select date"
                        disabled={isSubmitting}
                    />
                </div>
            </div>

            {/* UPI + KNEC row */}
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <div className="space-y-1.5">
                    <Label htmlFor={`upi_${entry.key}`}>UPI Number</Label>
                    <Input
                        id={`upi_${entry.key}`}
                        value={entry.upiNumber}
                        onChange={(e) => onUpdate({ upiNumber: e.target.value })}
                        placeholder="e.g. UP123456789"
                        disabled={isSubmitting}
                    />
                </div>
                <div className="space-y-1.5">
                    <Label htmlFor={`knec_${entry.key}`}>KNEC Assessment Number</Label>
                    <Input
                        id={`knec_${entry.key}`}
                        value={entry.knecNumber}
                        onChange={(e) => onUpdate({ knecNumber: e.target.value })}
                        placeholder="e.g. KNEC123456"
                        disabled={isSubmitting}
                    />
                </div>
            </div>
        </div>
    );
}
