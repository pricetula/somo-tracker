"use client";

import React from "react";
import { AcademicYearHandler } from "@/features/academicyears/components/academic-year-handler";
import { UserCount } from "./user-count";

export function SystemAdminDashboardPage() {
    const [_academicYearId, setAcademicYearId] = React.useState<string | null>(null);
    const [_academicTermId, setAcademicTermId] = React.useState<string | null>(null);

    return (
        <article>
            <header className="flex gap-14">
                <AcademicYearHandler
                    onAcademicYearChange={setAcademicYearId}
                    onAcademicTermChange={setAcademicTermId}
                />
                <UserCount />
            </header>
            {/* The selected academic year id and term id can be used elsewhere in the dashboard */}
        </article>
    );
}
