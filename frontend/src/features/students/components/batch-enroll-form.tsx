/**
 * BatchEnrollForm — class selection form for enrolling selected students.
 *
 * Reads selected student IDs from the enrollment store, lets the user pick
 * a class, then submits all enrollments in one batch request. Academic term
 * is resolved server-side.
 */

"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Loader2, Users } from "lucide-react";

import { ClassCombobox } from "@/features/classes";
import { useBatchEnrollStudents } from "../hooks/use-student-detail";
import { useEnrollmentStore } from "../store/enrollment-store";
import { getErrorMessage } from "@/lib/errors";

// ─── Props ─────────────────────────────────────────────────────────────────

interface BatchEnrollFormProps {
    /** Called after successful enrollment. */
    onSuccess?: () => void;
    /** Called when user cancels. */
    onCancel?: () => void;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function BatchEnrollForm({ onSuccess, onCancel }: BatchEnrollFormProps) {
    const router = useRouter();
    const [selectedClassId, setSelectedClassId] = React.useState("");
    const [error, setError] = React.useState<string | null>(null);

    const { selectedStudentIds, clearSelectedStudentIds } = useEnrollmentStore();
    const batchEnroll = useBatchEnrollStudents();

    const studentCount = selectedStudentIds.length;

    const handleEnroll = async () => {
        if (!selectedClassId) {
            setError("Please select a class");
            return;
        }
        if (studentCount === 0) {
            setError("No students selected");
            return;
        }

        setError(null);

        try {
            await batchEnroll.mutateAsync({
                enrollments: selectedStudentIds.map((id) => ({
                    student_id: id,
                    class_id: selectedClassId,
                })),
            });
            clearSelectedStudentIds();
            onSuccess?.();
        } catch (err) {
            setError(getErrorMessage(err));
        }
    };

    const handleCancel = () => {
        clearSelectedStudentIds();
        onCancel?.();
    };

    return (
        <div className="space-y-6">
            {error && (
                <div className="text-destructive bg-destructive/10 rounded-md px-3 py-2 text-sm">
                    {error}
                </div>
            )}

            {/* Student count */}
            <div className="text-muted-foreground flex items-center gap-2 text-sm">
                <Users className="size-4" />
                <span>
                    {studentCount} student{studentCount === 1 ? "" : "s"} selected
                </span>
            </div>

            {/* Class selection */}
            <div className="space-y-1.5">
                <Label>Class to Enroll In</Label>
                <ClassCombobox
                    value={selectedClassId}
                    onChange={(v) => setSelectedClassId(v as string)}
                    placeholder="Select a class"
                    onCreateItem={() => {
                        router.push("/classes/add");
                    }}
                />
            </div>

            {/* Actions */}
            <div className="flex items-center justify-end gap-3 pt-2">
                <Button variant="ghost" onClick={handleCancel} disabled={batchEnroll.isPending}>
                    Cancel
                </Button>
                <Button
                    onClick={handleEnroll}
                    disabled={!selectedClassId || studentCount === 0 || batchEnroll.isPending}
                >
                    {batchEnroll.isPending ? (
                        <>
                            <Loader2 className="mr-1.5 size-4 animate-spin" />
                            Enrolling…
                        </>
                    ) : (
                        `Enroll ${studentCount} Student${studentCount === 1 ? "" : "s"}`
                    )}
                </Button>
            </div>
        </div>
    );
}
