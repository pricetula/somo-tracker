/**
 * Intercepted route for creating a new academic term via the modal slot.
 *
 * Accepts an optional academic_year_id query param to pre-select the year.
 * Renders inside a Dialog when navigating to /academic-terms/new from within
 * the dashboard, preserving the background page.
 */

"use client";

import { use, useState } from "react";
import { useRouter } from "next/navigation";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { AcademicYearCombobox } from "@/features/academic-terms";
import { TermForm } from "@/features/academic-years/components/term-form";

// ─── Props ────────────────────────────────────────────────────────────────

interface NewAcademicTermModalProps {
    searchParams?: Promise<{ academic_year_id?: string }>;
}

// ─── Modal ────────────────────────────────────────────────────────────────

export default function NewAcademicTermModal(props: NewAcademicTermModalProps) {
    const router = useRouter();
    const searchParams = props.searchParams ? use(props.searchParams) : {};
    const prefilledYearId = searchParams.academic_year_id ?? "";

    const [academicYearId, setAcademicYearId] = useState(prefilledYearId);

    return (
        <Dialog
            open
            onOpenChange={(open) => {
                if (!open) router.back();
            }}
        >
            <DialogContent className="sm:max-w-lg">
                <DialogHeader>
                    <DialogTitle>Create Academic Term</DialogTitle>
                    <DialogDescription>
                        Add a new term (e.g. Term 1, Term 2) to an academic year.
                    </DialogDescription>
                </DialogHeader>

                {!academicYearId ? (
                    <div className="space-y-4">
                        <div className="space-y-1.5">
                            <Label>Academic Year</Label>
                            <AcademicYearCombobox
                                value={academicYearId}
                                onChange={setAcademicYearId}
                                placeholder="Select an academic year..."
                            />
                        </div>
                        <p className="text-muted-foreground text-xs">
                            Select an academic year to start creating a term.
                        </p>
                    </div>
                ) : (
                    <div className="space-y-4">
                        <button
                            type="button"
                            onClick={() => setAcademicYearId("")}
                            className="text-muted-foreground hover:text-foreground text-xs underline underline-offset-2 transition-colors"
                        >
                            &larr; Change year
                        </button>
                        <TermForm
                            academicYearId={academicYearId}
                            onSuccess={() => router.push(`/academic-years/${academicYearId}`)}
                        />
                    </div>
                )}
            </DialogContent>
        </Dialog>
    );
}
