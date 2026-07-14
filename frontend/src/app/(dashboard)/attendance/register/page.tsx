/**
 * Attendance register page — drill-down from admin dashboard.
 * Shows the full teacher-style roster for a specific slot/date.
 */

import { Suspense } from "react";
import { getVerifiedRole } from "@/lib/auth-server";

export default async function AttendanceRegisterPage() {
    const role = await getVerifiedRole();

    if (!role) return null;

    const { AttendanceRegisterContainer } =
        await import("@/features/attendance/components/attendance-register-container");

    return (
        <Suspense
            fallback={<div className="text-muted-foreground py-8 text-center">Loading...</div>}
        >
            <AttendanceRegisterContainer role={role} />
        </Suspense>
    );
}
