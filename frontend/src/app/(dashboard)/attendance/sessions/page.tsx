/**
 * Attendance Sessions page — manage lesson execution sessions.
 *
 * TEACHER / SCHOOL_ADMIN: mark lessons as submitted/skipped, view session history.
 */

import { getVerifiedRole } from "@/lib/auth-server";

export default async function AttendanceSessionsPage() {
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

    const { SessionList } = await import("@/features/attendance/components/session-list");

    return (
        <div className="space-y-6">
            <h1 className="text-2xl font-bold">Attendance Sessions</h1>
            <p className="text-muted-foreground">
                Mark lesson sessions as submitted or skipped, and view session history.
            </p>
            <SessionList />
        </div>
    );
}
