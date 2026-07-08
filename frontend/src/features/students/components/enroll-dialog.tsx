/**
 * Enroll Dialog — modal to enroll a student in a class for a term.
 *
 * Features:
 * - Select academic term
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
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Loader2 } from "lucide-react";

import { useCreateEnrollment } from "../hooks/use-student-detail";
import { listTerms } from "@/lib/api/academic-terms";
import { getErrorMessage } from "@/lib/errors";
import type { AcademicTerm } from "@/lib/api/academic-terms";

// ─── Props ─────────────────────────────────────────────────────────────────

interface EnrollDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    studentId: string;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function EnrollDialog({ open, onOpenChange, studentId }: EnrollDialogProps) {
    const [terms, setTerms] = React.useState<AcademicTerm[]>([]);
    const [termsLoading, setTermsLoading] = React.useState(false);

    const [selectedTermId, setSelectedTermId] = React.useState("");
    const [selectedClassId, setSelectedClassId] = React.useState("");
    // TODO: add classes fetching when the backend endpoint is ready
    const [error, setError] = React.useState<string | null>(null);

    const createEnrollment = useCreateEnrollment();

    // Fetch terms when dialog opens
    React.useEffect(() => {
        if (!open) return;

        const abortController = new AbortController();

        const load = async () => {
            setTermsLoading(true);
            try {
                const termsRes = await listTerms();
                if (abortController.signal.aborted) return;
                setTerms(termsRes.items ?? []);
            } catch {
                if (abortController.signal.aborted) return;
                setTerms([]);
            } finally {
                if (!abortController.signal.aborted) {
                    setTermsLoading(false);
                }
            }
        };
        load();

        return () => abortController.abort();
    }, [open]);

    const handleEnroll = async () => {
        if (!selectedTermId) {
            setError("Please select a term");
            return;
        }
        if (!selectedClassId) {
            setError("Please select a class");
            return;
        }

        setError(null);

        try {
            await createEnrollment.mutateAsync({
                studentId,
                data: {
                    academic_term_id: selectedTermId,
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
                    <DialogTitle>Enroll in New Term</DialogTitle>
                    <DialogDescription>
                        Select a term and class to enroll this student.
                    </DialogDescription>
                </DialogHeader>

                <div className="space-y-4">
                    {error && (
                        <div className="text-destructive bg-destructive/10 rounded-md px-3 py-2 text-sm">
                            {error}
                        </div>
                    )}

                    {/* Term selection */}
                    <div className="space-y-1.5">
                        <Label htmlFor="term">Academic Term</Label>
                        <Select
                            value={selectedTermId}
                            onValueChange={setSelectedTermId}
                            disabled={termsLoading || createEnrollment.isPending}
                        >
                            <SelectTrigger id="term">
                                <SelectValue
                                    placeholder={termsLoading ? "Loading terms…" : "Select a term"}
                                />
                            </SelectTrigger>
                            <SelectContent>
                                {terms.map((t) => (
                                    <SelectItem key={t.id} value={t.id}>
                                        {t.name}
                                    </SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                    </div>

                    {/* Class selection — placeholder until classes endpoint is ready */}
                    <div className="space-y-1.5">
                        <Label htmlFor="class">Class</Label>
                        <Select
                            value={selectedClassId}
                            onValueChange={setSelectedClassId}
                            disabled={createEnrollment.isPending}
                        >
                            <SelectTrigger id="class">
                                <SelectValue placeholder="Select a class" />
                            </SelectTrigger>
                            <SelectContent>
                                <SelectItem value="__placeholder__" disabled>
                                    Classes not yet available
                                </SelectItem>
                            </SelectContent>
                        </Select>
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
                            disabled={
                                !selectedTermId || !selectedClassId || createEnrollment.isPending
                            }
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
                </div>
            </DialogContent>
        </Dialog>
    );
}
