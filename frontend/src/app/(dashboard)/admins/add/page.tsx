import AdminsPage from "../page";

/**
 * Children slot for `/admins/add`.
 *
 * Renders the exact same listing page as `/admins` while the `@modal` slot
 * overlays the BulkInviteForm dialog on top.
 */
export default function AdminsAddPage() {
    return <AdminsPage />;
}
