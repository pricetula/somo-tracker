import { redirect } from "next/navigation";
import { getVerifiedRole } from "@/lib/auth-server";

export default async function Home() {
    const role = await getVerifiedRole();

    if (!role) {
        redirect("/logout");
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
        case "PARENT": {
            const { ParentDashboardPage } =
                await import("@/features/dashboard/components/parent-dashboard");
            return <ParentDashboardPage />;
        }
        default:
            return (
                <article>
                    <p>Unknown role. Please contact support.</p>
                </article>
            );
    }
}
