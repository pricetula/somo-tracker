import { getVerifiedRole } from "@/lib/auth-server";
import {
    SystemAdminDashboardPage,
    SchoolAdminDashboardPage,
    TeacherDashboardPage,
    NurseDashboardPage,
    FinanceDashboardPage,
} from "@/features/dashboard";

export default async function Home() {
    const role = await getVerifiedRole();

    // The proxy ensures only authenticated users with a valid role reach here,
    // but we handle the edge case gracefully.
    if (!role) {
        return (
            <article>
                <p>Unable to verify your session. Please log in again.</p>
            </article>
        );
    }

    switch (role) {
        case "SYSTEM_ADMIN":
            return <SystemAdminDashboardPage />;
        case "SCHOOL_ADMIN":
            return <SchoolAdminDashboardPage />;
        case "TEACHER":
            return <TeacherDashboardPage />;
        case "NURSE":
            return <NurseDashboardPage />;
        case "FINANCE":
            return <FinanceDashboardPage />;
        default:
            return (
                <article>
                    <p>Unknown role. Please contact support.</p>
                </article>
            );
    }
}
