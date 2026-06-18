"use client";

import { MyAttendanceSection } from "@/features/cbc/components/dashboard/my-attendance";

/**
 * Teacher Dashboard.
 *
 * This is the landing page for users with the TEACHER role.
 * It surfaces:
 *  - "My Attendance" (today's schedule / quick attendance marking)
 *  - Upcoming tasks / recent activity (placeholder for future work)
 */

// TODO: extract teacherId from session context / auth hook
const DEMO_TEACHER_ID = "current-teacher-id";

export function TeacherDashboardPage() {
    return (
        <div className="flex flex-1 flex-col gap-6 p-6">
            {/* ── Welcome ─────────────────────────────────────────────── */}
            <div>
                <h1 className="text-2xl font-semibold tracking-tight">Teacher dashboard</h1>
                <p className="text-muted-foreground mt-1 text-sm">
                    Manage your classes, attendance, and assessments.
                </p>
            </div>

            {/* ── My Attendance ───────────────────────────────────────── */}
            <MyAttendanceSection teacherId={DEMO_TEACHER_ID} />
        </div>
    );
}
