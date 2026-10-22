import { AppLayout } from "@/components/layout/layout";
import { DashboardAuthLayout } from "@/features/auth/components/dashboard-auth-layout";
import { PeriodErrorProvider } from "@/components/providers/period-error-provider";

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
    modal?: React.ReactNode;
}) {
    return (
        <DashboardAuthLayout>
            <PeriodErrorProvider>
                <AppLayout modal={modal}>{children}</AppLayout>
            </PeriodErrorProvider>
        </DashboardAuthLayout>
    );
}
