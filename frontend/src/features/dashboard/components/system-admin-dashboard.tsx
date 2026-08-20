"use client";

import React from "react";
import { AcademicYearHandler } from "@/features/academic-years/components/academic-year-handler";
import {
    AttendanceSummary,
    AttendanceCalendar,
    LowestAttendanceStudents,
    WeekdayAttendanceExceptionsChart,
} from "@/features/attendance";
import { UserCount } from "./user-count";

export function SystemAdminDashboardPage() {
    return (
        <article className="space-y-16">
            <header className="flex flex-col gap-8 border-b border-dashed pb-8 sm:flex-row sm:items-end sm:justify-between">
                <AcademicYearHandler />
                <UserCount />
            </header>
            <section className="grid-cols grid gap-4 sm:gap-8 lg:grid-cols-3">
                <AttendanceSummary />
                <AttendanceCalendar />
                <LowestAttendanceStudents />
                <WeekdayAttendanceExceptionsChart />
            </section>
            <section className="space-y-4"></section>
            {/* The selected academic year id and term id can be used elsewhere in the dashboard */}
        </article>
    );
}
