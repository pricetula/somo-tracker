/**
 * Intercepted route — Attendance register rendered as a sliding side sheet.
 *
 * When a user clicks the pencil icon in the admin attendance dashboard,
 * this sheet slides out from the right keeping the dashboard table visible.
 * On hard refresh the full page at /attendance/register takes over.
 */

import { Suspense } from "react";
import { getVerifiedRole } from "@/lib/auth-server";
import { AttendanceRegisterSheet } from "./attendance-register-sheet";

export default async function InterceptedAttendanceRegister() {
    const role = await getVerifiedRole();
    if (!role) return null;

    return (
        <Suspense
            fallback={
                <div className="text-muted-foreground flex items-center justify-center py-12 text-sm">
                    Loading register...
                </div>
            }
        >
            <AttendanceRegisterSheet role={role} />
        </Suspense>
    );
}
