/**
 * AddClassForm — Creates a new class.
 *
 * Uses reusable comboboxes from their respective feature modules:
 *  - GradeLevelCombobox from grade-level feature
 *  - StreamCombobox from streams feature
 *  - AcademicYearCombobox from academic-terms feature
 *
 * The academic_term_id is resolved server-side from the current active term.
 */

"use client";

import { useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Loader2 } from "lucide-react";

import { createClass, type Class } from "@/lib/api/classes";
import { getErrorMessage, isApiError } from "@/lib/errors";
import { GradeLevelCombobox } from "@/features/grade-level";
import { StreamCombobox } from "@/features/streams";
import { AcademicYearCombobox } from "@/features/academic-terms";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription } from "@/components/ui/alert";

// ─── Props ─────────────────────────────────────────────────────────────────

interface AddClassFormProps {
    /** Called when the class is successfully created. */
    onSuccess?: (cls: Class) => void;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function AddClassForm({ onSuccess }: AddClassFormProps) {
    const router = useRouter();
    const queryClient = useQueryClient();

    // ── Form state ────────────────────────────────────────────────────
    const [gradeLevel, setGradeLevel] = useState("");
    const [streamId, setStreamId] = useState("");
    const [academicYearId, setAcademicYearId] = useState("");
    const [error, setError] = useState<string | null>(null);
    const [fieldErrors, setFieldErrors] = useState<Record<string, string[]>>({});

    // ── Mutation ──────────────────────────────────────────────────────
    const createMutation = useMutation({
        mutationFn: () => {
            if (!gradeLevel) throw new Error("Grade level is required.");
            if (!streamId) throw new Error("Stream is required.");
            if (!academicYearId) throw new Error("Academic year is required.");

            return createClass({
                grade_level: gradeLevel,
                stream_id: streamId,
                academic_year_id: academicYearId,
                academic_term_id: "", // resolved server-side to current active term
                student_ids: [],
            });
        },
        onSuccess: (cls) => {
            queryClient.invalidateQueries({ queryKey: ["classes"] });
            onSuccess?.(cls);
            router.back();
        },
        onError: (err) => {
            if (isApiError(err) && err.status === 400 && err.errors) {
                setFieldErrors(err.errors);
                setError(null);
            } else {
                setFieldErrors({});
                setError(getErrorMessage(err));
            }
        },
    });

    // ── Handlers ──────────────────────────────────────────────────────
    const handleSubmit = useCallback(
        (e: React.FormEvent) => {
            e.preventDefault();
            setError(null);
            setFieldErrors({});
            createMutation.mutate();
        },
        [createMutation]
    );

    return (
        <form onSubmit={handleSubmit} className="space-y-4">
            {/* General error */}
            {error && (
                <Alert variant="destructive">
                    <AlertDescription>{error}</AlertDescription>
                </Alert>
            )}

            {/* Grade Level */}
            <div className="space-y-1.5">
                <label className="text-sm font-medium">Grade Level</label>
                <GradeLevelCombobox
                    value={gradeLevel}
                    onChange={(v) => {
                        setGradeLevel(v);
                        setFieldErrors({});
                    }}
                />
                {fieldErrors.grade_level && (
                    <p className="text-destructive text-xs">{fieldErrors.grade_level[0]}</p>
                )}
            </div>

            {/* Stream */}
            <div className="space-y-1.5">
                <label className="text-sm font-medium">Stream</label>
                <StreamCombobox
                    value={streamId}
                    onChange={(v) => {
                        setStreamId(v);
                        setFieldErrors({});
                    }}
                    onCreateItem={(search) => {
                        router.push(`/streams/add?value=${encodeURIComponent(search)}`);
                    }}
                />
                {fieldErrors.stream_id && (
                    <p className="text-destructive text-xs">{fieldErrors.stream_id[0]}</p>
                )}
            </div>

            {/* Academic Year */}
            <div className="space-y-1.5">
                <label className="text-sm font-medium">Academic Year</label>
                <AcademicYearCombobox
                    value={academicYearId}
                    onChange={(v) => {
                        setAcademicYearId(v);
                        setFieldErrors({});
                    }}
                />
                {fieldErrors.academic_year_id && (
                    <p className="text-destructive text-xs">{fieldErrors.academic_year_id[0]}</p>
                )}
            </div>

            {/* Submit */}
            <div className="flex items-center justify-end gap-2 pt-2">
                <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => router.back()}
                    disabled={createMutation.isPending}
                >
                    Cancel
                </Button>
                <Button type="submit" size="sm" disabled={createMutation.isPending}>
                    {createMutation.isPending ? (
                        <>
                            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                            Creating...
                        </>
                    ) : (
                        "Create Class"
                    )}
                </Button>
            </div>
        </form>
    );
}
