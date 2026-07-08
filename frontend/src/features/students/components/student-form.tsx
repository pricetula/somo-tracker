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
import { useId } from "react";
import { useRouter } from "next/navigation";

import { Button } from "@/components/ui/button";
import { Loader2, Plus } from "lucide-react";

import { useCreateStudents, useUpdateStudent } from "../hooks/use-student-detail";
import { getErrorMessage, isApiError } from "@/lib/errors";
import { StudentEntryRow, type StudentEntry } from "./student-entry-row";
import type { StudentDetail, CreateStudentPayload } from "../types";

// ─── Props ─────────────────────────────────────────────────────────────────

interface StudentFormProps {
    mode: "create" | "edit";
    initialData?: StudentDetail;
    onSuccess?: (id: string) => void;
}

function freshEntry(baseId: string): StudentEntry {
    return {
        key: `${baseId}-${crypto.randomUUID()}`,
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

    const baseId = useId();
    const [entries, setEntries] = React.useState<StudentEntry[]>([freshEntry(baseId)]);
    const [error, setError] = React.useState<string | null>(null);
    const [fieldErrors, setFieldErrors] = React.useState<Record<string, string[]>>({});

    const isSubmitting = createStudents.isPending || updateStudent.isPending;

    const updateEntry = (key: string, patch: Partial<StudentEntry>) => {
        setEntries((prev) => prev.map((e) => (e.key === key ? { ...e, ...patch } : e)));
    };

    const addEntry = () => {
        setEntries((prev) => [...prev, freshEntry(baseId)]);
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
                if (isApiError(err) && err.errors) {
                    setFieldErrors(err.errors);
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
            if (isApiError(err) && err.errors) {
                setFieldErrors(err.errors);
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
