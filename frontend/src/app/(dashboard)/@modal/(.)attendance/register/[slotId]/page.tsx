/**
 * Intercepted route — Attendance register rendered as a sliding side sheet.
 *
 * When a user clicks the pencil icon in the admin dashboard, this sheet slides
 * out keeping the dashboard visible. On hard refresh the full page at
 * /attendance/register/[slotId] takes over.
 *
 * `slotId` comes from the URL path, `date` is an optional query param.
 */

import { getVerifiedRole } from "@/lib/auth-server";
import { AttendanceRegisterSheet } from "../attendance-register-sheet";

interface PageProps {
    params: Promise<{ slotId: string }>;
    searchParams: Promise<{ date?: string }>;
}

export default async function InterceptedAttendanceRegister({ params, searchParams }: PageProps) {
    const role = await getVerifiedRole();
    if (!role) return null;

    const { slotId } = await params;
    const { date } = await searchParams;

    return <AttendanceRegisterSheet role={role} slotId={slotId} date={date} />;
}
