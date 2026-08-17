"use client";

import React from "react";
import { AcademicYearHandler } from "@/features/academicyears/components/academic-year-handler";
import {
    ClassAttendanceBreakdownChart,
    LearningAreaAbsenteeismChart,
    SchoolAttendanceKPIs,
} from "@/features/attendance";
import { TeacherComplianceChart } from "@/features/teacher-delivery";
import { UserCount } from "./user-count";

export function SystemAdminDashboardPage() {
    return (
        <article className="space-y-10">
            <header className="flex flex-col items-end justify-between sm:flex-row">
                <AcademicYearHandler />
                <UserCount />
            </header>
            <SchoolAttendanceKPIs />
            <ClassAttendanceBreakdownChart />
            <LearningAreaAbsenteeismChart />
            <TeacherComplianceChart />
            {/* The selected academic year id and term id can be used elsewhere in the dashboard */}
        </article>
    );
}
