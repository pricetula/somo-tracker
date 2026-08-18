"use client";

import React from "react";
import { AcademicYearHandler } from "@/features/academicyears/components/academic-year-handler";
import { AttendanceSummary } from "@/features/attendance";
import { UserCount } from "./user-count";

export function SystemAdminDashboardPage() {
    return (
        <article className="space-y-10">
            <header className="flex flex-col items-end justify-between border-b border-dashed pb-8 sm:flex-row">
                <AcademicYearHandler />
                <UserCount />
            </header>
            <section className="grid-cols grid gap-6 sm:grid-cols-2">
                <AttendanceSummary />
            </section>
            {/* The selected academic year id and term id can be used elsewhere in the dashboard */}
        </article>
    );
}
