"use client";

import { AcademicYearHandler } from "@/features/academicyears/components/academic-year-handler";

export function SystemAdminDashboardPage() {
    return (
        <article>
            <header className="border-b pb-8">
                <AcademicYearHandler />
            </header>
        </article>
    );
}
