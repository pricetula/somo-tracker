/**
 * Attendance register page — drill-down from admin dashboard.
 * `slotId` comes from the URL path, `date` is an optional query param.
 */

import { getVerifiedRole } from "@/lib/auth-server";
import { AttendanceRegisterContainer } from "@/features/attendance/components/attendance-register-container";

interface PageProps {
    params: Promise<{ slotId: string }>;
    searchParams: Promise<{ date?: string }>;
}

export default async function AttendanceRegisterPage({ params, searchParams }: PageProps) {
    const role = await getVerifiedRole();
    if (!role) return null;

    const { slotId } = await params;
    const { date } = await searchParams;

    return <AttendanceRegisterContainer role={role} slotId={slotId} date={date} />;
}
