import { TeachersInviteDialog } from "@/features/members";

/**
 * Intercepted `@modal` slot for `/teachers/add`.
 *
 * Matches only during client-side navigation (the DataTable "Add" link), so
 * the BulkInviteForm renders in a dialog. On a hard navigation/refresh this
 * intercept is skipped and `@modal/default.tsx` renders, leaving
 * `/teachers/add` to render the full-page form.
 */
export default function TeachersInviteModal() {
    return <TeachersInviteDialog />;
}
