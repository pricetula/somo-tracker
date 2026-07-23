/**
 * Term report detail — role-agnostic route.
 *
 * SCHOOL_ADMIN: management view (generate, publish)
 * PARENT: read-only compiled report
 */

import { getVerifiedRole } from "@/lib/auth-server";

export default async function TermReportPage() {
    const role = await getVerifiedRole();

    if (!role) {
        return (
            <article>
                <p>Unable to verify your session. Please log in again.</p>
            </article>
        );
    }

    switch (role) {
        case "SCHOOL_ADMIN":
        case "SYSTEM_ADMIN": {
            return (
                <div className="space-y-6">
                    <h1 className="text-2xl font-bold">Term Reports</h1>
                    <p className="text-muted-foreground">
                        Select a student to generate or manage reports.
                    </p>
                </div>
            );
        }
        case "PARENT": {
            return (
                <div className="space-y-6">
                    <h1 className="text-2xl font-bold">Term Report</h1>
                    <p className="text-muted-foreground">Loading report...</p>
                </div>
            );
        }
        default:
            return (
                <article>
                    <p>You do not have access to this page.</p>
                </article>
            );
    }
}
