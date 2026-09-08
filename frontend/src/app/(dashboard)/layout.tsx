import { AppLayout } from "@/components/layout/layout";
import { DashboardAuthLayout } from "@/features/auth/components/dashboard-auth-layout";

/**
 * Dashboard layout — wraps all authenticated pages.
 * Add sidebar, header, and navigation chrome here.
 *
 * The `modal` slot is the @modal parallel route that intercepts
 */
export default function DashboardLayout({
    children,
    modal,
}: {
    children: React.ReactNode;
    modal: React.ReactNode;
}) {
    return (
        <DashboardAuthLayout>
            <AppLayout>
                {children}
                {modal}
            </AppLayout>
        </DashboardAuthLayout>
    );
}
