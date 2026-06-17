import { AppLayout } from "@/components/layout/layout";

/**
 * Dashboard layout — wraps all authenticated pages.
 * Add sidebar, header, and navigation chrome here.
 *
 * The `modal` slot is the @modal parallel route that intercepts
 * /calendar/new and /classes/generate to render them as dialogs.
 */
export default function DashboardLayout({
    children,
    modal,
}: {
    children: React.ReactNode;
    modal: React.ReactNode;
}) {
    return (
        <AppLayout>
            {children}
            {modal}
        </AppLayout>
    );
}
