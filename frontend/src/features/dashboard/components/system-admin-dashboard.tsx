"use client";

import React from "react";
import { AcademicYearHandler } from "@/features/academicyears/components/academic-year-handler";
import {
    AttendanceSummary,
    AttendanceCalendar,
    LowestAttendanceStudents,
} from "@/features/attendance";
import { UserCount } from "./user-count";

export function SystemAdminDashboardPage() {
    return (
        <article className="space-y-16">
            <header className="flex flex-col gap-8 border-b border-dashed pb-8 sm:flex-row sm:items-end sm:justify-between">
                <AcademicYearHandler />
                <UserCount />
            </header>
            <section className="grid-cols grid gap-4 sm:gap-4 lg:grid-cols-3 lg:gap-8">
                <AttendanceSummary />
                <AttendanceCalendar />
                <LowestAttendanceStudents />
            </section>
            {/* The selected academic year id and term id can be used elsewhere in the dashboard */}
        </article>
    );
}
