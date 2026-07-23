/**
 * Create Academic Term page.
 *
 * Accepts an optional academic_year_id query param to pre-select the year.
 * If not provided, the user picks an academic year first.
 */

"use client";

import { use, useState } from "react";
import { useRouter } from "next/navigation";
import { ArrowLeft } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { AcademicYearCombobox } from "@/features/academic-terms";
import { TermForm } from "@/features/academic-years/components/term-form";

// ─── Props ────────────────────────────────────────────────────────────────

interface NewAcademicTermPageProps {
    searchParams?: Promise<{ academic_year_id?: string }>;
}

// ─── Page ─────────────────────────────────────────────────────────────────

export default function NewAcademicTermPage(props: NewAcademicTermPageProps) {
    const router = useRouter();
    const searchParams = props.searchParams ? use(props.searchParams) : {};
    const prefilledYearId = searchParams.academic_year_id ?? "";

    const [academicYearId, setAcademicYearId] = useState(prefilledYearId);

    return (
        <div className="mx-auto max-w-lg p-6">
            <Button variant="ghost" size="sm" onClick={() => router.back()} className="mb-4 -ml-2">
                <ArrowLeft className="mr-2 h-4 w-4" />
                Back
            </Button>

            <h1 className="mb-1 text-lg font-semibold">Create Academic Term</h1>
            <p className="text-muted-foreground mb-6">
                Add a new term (e.g. Term 1, Term 2) to an academic year.
            </p>

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
                    <div className="flex items-center gap-2">
                        <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => setAcademicYearId("")}
                            className="text-muted-foreground h-auto p-0 text-xs"
                        >
                            &larr; Change year
                        </Button>
                    </div>
                    <TermForm
                        academicYearId={academicYearId}
                        onSuccess={() => router.push(`/academic-years/${academicYearId}`)}
                    />
                </div>
            )}
        </div>
    );
}
