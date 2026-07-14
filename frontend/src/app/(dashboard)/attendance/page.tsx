/**
 * Attendance page — role-agnostic route.
 *
 * TEACHER: shows their current/next period roster
 * SCHOOL_ADMIN: shows school-wide completion dashboard
 * PARENT: shows linked child attendance summary
 */

import { getVerifiedRole } from "@/lib/auth-server";

export default async function AttendancePage() {
    const role = await getVerifiedRole();

    if (!role) {
        return (
            <article>
                <p>Unable to verify your session. Please log in again.</p>
            </article>
        );
    }

    switch (role) {
        case "TEACHER": {
            const { TeacherAttendanceView } =
                await import("@/features/attendance/components/teacher-attendance-view");
            return <TeacherAttendanceView />;
        }
        case "SCHOOL_ADMIN":
        case "SYSTEM_ADMIN": {
            const { AdminAttendanceDashboard } =
                await import("@/features/attendance/components/admin-attendance-dashboard");
            return <AdminAttendanceDashboard />;
        }
        case "PARENT": {
            const { ParentAttendanceView } =
                await import("@/features/attendance/components/parent-attendance-view");
            return <ParentAttendanceView />;
        }
        default:
            return (
                <article>
                    <p>You do not have access to this page.</p>
                </article>
            );
    }
}
