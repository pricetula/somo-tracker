/**
 * Health page — role-agnostic route.
 *
 * NURSE / SCHOOL_ADMIN: view and log medical incidents, manage health profiles.
 */

import { getVerifiedRole } from "@/lib/auth-server";

export default async function HealthPage() {
    const role = await getVerifiedRole();

    if (!role) {
        return (
            <article>
                <p>Unable to verify your session. Please log in again.</p>
            </article>
        );
    }

    const allowedRoles = ["NURSE", "SCHOOL_ADMIN", "SYSTEM_ADMIN"];
    if (!allowedRoles.includes(role)) {
        return (
            <article>
                <p>You do not have access to this page.</p>
            </article>
        );
    }

    const { IncidentList } = await import("@/features/health/components/incident-list");

    return (
        <div className="space-y-6">
            <h1 className="text-2xl font-bold">Health</h1>
            <p className="text-muted-foreground">
                Log and manage medical incidents and student health profiles.
            </p>
            <IncidentList />
        </div>
    );
}
