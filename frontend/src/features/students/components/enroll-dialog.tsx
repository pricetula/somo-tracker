/**
 * Enroll Dialog — modal to enroll a student in a class.
 *
 * Academic term is resolved server-side from the current active term.
 * Features:
 * - Select class
 * - Optional enrollment status (default ACTIVE)
 */

"use client";

import * as React from "react";

import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Loader2 } from "lucide-react";

import { ClassCombobox } from "@/features/classes";
import { useCreateEnrollment } from "../hooks/use-student-detail";
import { getErrorMessage } from "@/lib/errors";

// ─── Props ─────────────────────────────────────────────────────────────────

interface EnrollDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    studentId: string;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function EnrollDialog({ open, onOpenChange, studentId }: EnrollDialogProps) {
    const [selectedClassId, setSelectedClassId] = React.useState("");
    const [error, setError] = React.useState<string | null>(null);

    const createEnrollment = useCreateEnrollment();

    const handleEnroll = async () => {
        if (!selectedClassId) {
            setError("Please select a class");
            return;
        }

        setError(null);

        try {
            await createEnrollment.mutateAsync({
                studentId,
                data: {
                    class_id: selectedClassId,
                },
            });
            onOpenChange(false);
        } catch (err) {
            setError(getErrorMessage(err));
        }
    };

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="max-w-md">
                <DialogHeader>
                    <DialogTitle>Enroll Student</DialogTitle>
                    <DialogDescription>
                        Select a class to enroll this student. The current academic term will be
                        used automatically.
                    </DialogDescription>
                </DialogHeader>

                <div className="space-y-4">
                    {error && (
                        <div className="text-destructive bg-destructive/10 rounded-md px-3 py-2">
                            {error}
                        </div>
                    )}
                </div>

                {/* Class selection */}
                <div className="space-y-1.5">
                    <Label>Class</Label>
                    <ClassCombobox
                        value={selectedClassId}
                        onChange={(v) => setSelectedClassId(v as string)}
                        placeholder="Select a class"
                    />
                </div>

                {/* Actions */}
                <div className="flex items-center justify-end gap-3 pt-2">
                    <Button
                        variant="ghost"
                        onClick={() => onOpenChange(false)}
                        disabled={createEnrollment.isPending}
                    >
                        Cancel
                    </Button>
                    <Button
                        onClick={handleEnroll}
                        disabled={!selectedClassId || createEnrollment.isPending}
                    >
                        {createEnrollment.isPending ? (
                            <>
                                <Loader2 className="mr-1.5 size-4 animate-spin" />
                                Enrolling…
                            </>
                        ) : (
                            "Enroll"
                        )}
                    </Button>
                </div>
            </DialogContent>
        </Dialog>
    );
}
