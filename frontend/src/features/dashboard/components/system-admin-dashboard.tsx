"use client";

import React from "react";
import { AcademicYearHandler } from "@/features/academicyears/components/academic-year-handler";
import { UserCount } from "./user-count";

export function SystemAdminDashboardPage() {
    return (
        <article>
            <header className="flex flex-col justify-between gap-14 border-b pb-8 sm:flex-row">
                <AcademicYearHandler />
                <UserCount />
            </header>
            {/* The selected academic year id and term id can be used elsewhere in the dashboard */}
        </article>
    );
}
