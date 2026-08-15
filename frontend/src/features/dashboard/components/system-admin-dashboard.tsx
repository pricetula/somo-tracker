"use client";

import React from "react";
import { AcademicYearHandler } from "@/features/academicyears/components/academic-year-handler";

export function SystemAdminDashboardPage() {
    const [_academicYearId, setAcademicYearId] = React.useState<string | null>(null);
    const [_academicTermId, setAcademicTermId] = React.useState<string | null>(null);

    return (
        <article>
            <header className="border-b pb-8">
                <AcademicYearHandler
                    onAcademicYearChange={setAcademicYearId}
                    onAcademicTermChange={setAcademicTermId}
                />
            </header>
            {/* The selected academic year id and term id can be used elsewhere in the dashboard */}
        </article>
    );
}
