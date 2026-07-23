/**
 * Admins layout — keeps the layout thin.
 *
 * Modal intercepts for admins routes (import, invitations, detail) live in the
 * parent dashboard's @modal parallel slot — not in this layout.
 */

export default function AdminsLayout({ children }: { children: React.ReactNode }) {
    return <>{children}</>;
}
