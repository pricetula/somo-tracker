"use client";

import { useMe } from "@/hooks/use-auth";
import { MyAttendanceSection } from "@/features/cbc/components/dashboard/my-attendance";

/**
 * Teacher Dashboard.
 *
 * This is the landing page for users with the TEACHER role.
 * It surfaces:
 *  - "My Attendance" (today's schedule / quick attendance marking)
 *  - Upcoming tasks / recent activity (placeholder for future work)
 */

export function TeacherDashboardPage() {
    const { data: me } = useMe();
    const teacherId = me?.user_id ?? "";

    return (
        <div className="flex flex-1 flex-col gap-6 p-6">
            {/* ── Welcome ─────────────────────────────────────────────── */}
            <div>
                <h1 className="text-2xl font-semibold tracking-tight">
                    Welcome back{me?.first_name ? `, ${me.first_name}` : ""}
                </h1>
                <p className="text-muted-foreground mt-1 text-sm">
                    Manage your classes, attendance, and assessments.
                </p>
            </div>

            {/* ── My Attendance ───────────────────────────────────────── */}
            <MyAttendanceSection teacherId={teacherId} />
        </div>
    );
}
