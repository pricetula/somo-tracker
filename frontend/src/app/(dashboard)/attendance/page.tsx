/**
 * Attendance page — role-aware route.
 *
 * TEACHER:       shows their current/next period roster
 * SCHOOL_ADMIN:  shows school-wide completion dashboard
 * SYSTEM_ADMIN:  shows school-wide completion dashboard
 * PARENT:        shows linked child attendance summary
 * NURSE/FINANCE: info message (no attendance access)
 */

import { redirect } from "next/navigation";
import { getVerifiedRole } from "@/lib/auth-server";

export default async function AttendancePage() {
    const role = await getVerifiedRole();

    if (!role) {
        redirect("/logout");
    }

    switch (role) {
        case "TEACHER": {
            const { TeacherAttendanceLanding } =
                await import("@/features/attendance/components/teacher-attendance-landing");
            return <TeacherAttendanceLanding />;
        }
        case "SCHOOL_ADMIN":
        case "SYSTEM_ADMIN": {
            const { AdminAttendanceDashboard } =
                await import("@/features/attendance/components/admin-attendance-dashboard");
            return <AdminAttendanceDashboard />;
        }
        case "PARENT": {
            const { ParentAttendanceLanding } =
                await import("@/features/attendance/components/parent-attendance-landing");
            return <ParentAttendanceLanding />;
        }
        case "NURSE":
        case "FINANCE": {
            return (
                <article className="space-y-4">
                    <h1 className="text-foreground text-2xl font-bold">Attendance</h1>
                    <p className="text-muted-foreground">
                        Attendance records are managed by teachers and school administrators. If you
                        need access, please contact your school admin.
                    </p>
                </article>
            );
        }
        default:
            return (
                <article>
                    <p className="text-muted-foreground">Unknown role. Please contact support.</p>
                </article>
            );
    }
}
