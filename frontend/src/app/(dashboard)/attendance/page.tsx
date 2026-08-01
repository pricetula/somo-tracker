/**
 * Attendance page — timeline view of today's schedule for the selected class,
 * plus calendar status overview.
 *
 * TEACHER / SCHOOL_ADMIN: view the day's timeline and mark attendance per slot.
 */

import { AttendanceCalendar, CalendarStatusView } from "@/features/attendance";
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

    const allowedRoles = ["TEACHER", "SCHOOL_ADMIN", "SYSTEM_ADMIN"];
    if (!allowedRoles.includes(role)) {
        return (
            <article>
                <p>You do not have access to this page.</p>
            </article>
        );
    }

    return (
        <div className="space-y-6">
            <AttendanceCalendar />
            <CalendarStatusView />
        </div>
    );
}
