import { AdminsInviteDialog } from "@/features/members";

/**
 * Intercepted `@modal` slot for `/admins/add`.
 *
 * Matches only during client-side navigation (the DataTable "Add" link), so
 * the BulkInviteForm renders in a dialog. On a hard navigation/refresh this
 * intercept is skipped and `@modal/default.tsx` renders, leaving
 * `/admins/add` to render the full-page form.
 */
export default function AdminsInviteModal() {
    return <AdminsInviteDialog />;
}
