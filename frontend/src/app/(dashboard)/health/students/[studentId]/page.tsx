/**
 * Student health detail page.
 *
 * Shows composite view: health profile + recent medical incidents.
 */

import { getVerifiedRole } from "@/lib/auth-server";

export default async function StudentHealthPage({
    params,
}: {
    params: Promise<{ studentId: string }>;
}) {
    const role = await getVerifiedRole();
    const { studentId } = await params;

    if (!role) {
        return (
            <article>
                <p>Unable to verify your session. Please log in again.</p>
            </article>
        );
    }

    const allowedRoles = ["NURSE", "SCHOOL_ADMIN", "SYSTEM_ADMIN", "TEACHER"];
    if (!allowedRoles.includes(role)) {
        return (
            <article>
                <p>You do not have access to this page.</p>
            </article>
        );
    }

    const { StudentHealthView } = await import("@/features/health/components/student-health-view");

    return (
        <div className="space-y-6">
            <h1 className="text-2xl font-bold">Student Health</h1>
            <p className="text-muted-foreground">Student ID: {studentId}</p>
            <StudentHealthView studentId={studentId} />
        </div>
    );
}
