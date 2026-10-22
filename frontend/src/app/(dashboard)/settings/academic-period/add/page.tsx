"use client";

import { AcademicPeriodAddForm } from "@/features/academic-period";

export default function AcademicPeriodAddPage() {
    return (
        <div className="mx-auto max-w-2xl space-y-6 p-6">
            <div>
                <h1 className="text-2xl font-semibold">Create Academic Period</h1>
                <p className="text-muted-foreground text-sm">
                    Add a new academic year with its terms.
                </p>
            </div>
            <AcademicPeriodAddForm />
        </div>
    );
}
