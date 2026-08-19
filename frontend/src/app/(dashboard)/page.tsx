import { redirect } from "next/navigation";
import { getVerifiedRole } from "@/lib/auth-server";

export default async function Home() {
    const role = await getVerifiedRole();

    if (!role) {
        redirect("/logout");
    }

    switch (role) {
        case "SCHOOL_ADMIN": {
            const { SystemAdminDashboardPage } = await import("@/features/dashboard");
            return <SystemAdminDashboardPage />;
        }
        default:
            return (
                <article>
                    <p>Unknown role. Please contact support.</p>
                </article>
            );
    }
}
