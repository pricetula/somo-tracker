import { getVerifiedRole } from "@/lib/auth-server";

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
        case "SYSTEM_ADMIN": {
            const { SystemAdminDashboardPage } =
                await import("@/features/dashboard/components/system-admin-dashboard");
            return <SystemAdminDashboardPage />;
        }
        case "SCHOOL_ADMIN": {
            const { SchoolAdminDashboardPage } =
                await import("@/features/dashboard/components/school-admin-dashboard");
            return <SchoolAdminDashboardPage />;
        }
        case "TEACHER": {
            const { TeacherDashboardPage } =
                await import("@/features/dashboard/components/teacher-dashboard");
            return <TeacherDashboardPage />;
        }
        case "NURSE": {
            const { NurseDashboardPage } =
                await import("@/features/dashboard/components/nurse-dashboard");
            return <NurseDashboardPage />;
        }
        case "FINANCE": {
            const { FinanceDashboardPage } =
                await import("@/features/dashboard/components/finance-dashboard");
            return <FinanceDashboardPage />;
        }
        default:
            return (
                <article>
                    <p>Unknown role. Please contact support.</p>
                </article>
            );
    }
}
