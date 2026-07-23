/**
 * Class Teachers page — role-agnostic route.
 *
 * SCHOOL_ADMIN: view and manage teacher-to-class assignments.
 */

import { getVerifiedRole } from "@/lib/auth-server";

export default async function ClassTeachersPage() {
    const role = await getVerifiedRole();

    if (!role) {
        return (
            <article>
                <p>Unable to verify your session. Please log in again.</p>
            </article>
        );
    }

    const allowedRoles = ["SCHOOL_ADMIN", "SYSTEM_ADMIN"];
    if (!allowedRoles.includes(role)) {
        return (
            <article>
                <p>You do not have access to this page.</p>
            </article>
        );
    }

    const { ClassTeacherList } =
        await import("@/features/classteachers/components/class-teacher-list");

    return (
        <div className="space-y-6">
            <h1 className="text-2xl font-bold">Teacher Assignments</h1>
            <p className="text-muted-foreground">
                Assign teachers to classes and manage their roles and subjects.
            </p>
            <ClassTeacherList />
        </div>
    );
}
