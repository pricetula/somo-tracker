/**
 * Student attendance history page — admin view.
 * Raw period-by-period data for manual interpretation.
 */

import { getVerifiedRole } from "@/lib/auth-server";

export default async function StudentAttendanceHistoryPage() {
    const role = await getVerifiedRole();
    if (!role) return null;

    const { StudentHistoryContainer } =
        await import("@/features/attendance/components/student-history-container");
    return <StudentHistoryContainer />;
}
