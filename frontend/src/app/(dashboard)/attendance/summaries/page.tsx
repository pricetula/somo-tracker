/**
 * Attendance Summaries page — view term-level attendance summaries.
 *
 * SCHOOL_ADMIN: view class and student attendance percentages per term
 * TEACHER: view their class summaries
 */

import { getVerifiedRole } from "@/lib/auth-server";

export default async function AttendanceSummariesPage() {
    const role = await getVerifiedRole();

    if (!role) {
        return (
            <article>
                <p>Unable to verify your session. Please log in again.</p>
            </article>
        );
    }

    const allowedRoles = ["TEACHER", "SCHOOL_ADMIN", "SYSTEM_ADMIN"];
    if (!allowedRoles.includes(role)) {
        return (
            <article>
                <p>You do not have access to this page.</p>
            </article>
        );
    }

    const { SummaryTable } = await import("@/features/attendance/components/summary-table");

    return (
        <div className="space-y-6">
            <h1 className="text-2xl font-bold">Attendance Summaries</h1>
            <p className="text-muted-foreground">
                View term-level attendance summaries for students and classes.
            </p>
            {/* TODO: Add class/term picker before rendering the table */}
            <SummaryTable classId="" />
        </div>
    );
}
