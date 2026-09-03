import { NursesInviteDialog } from "@/features/members";

/**
 * Intercepted `@modal` slot for `/nurses/add`.
 *
 * Matches only during client-side navigation (the DataTable "Add" link), so
 * the BulkInviteForm renders in a dialog. On a hard navigation/refresh this
 * intercept is skipped and `@modal/default.tsx` renders, leaving
 * `/nurses/add` to render the full-page form.
 */
export default function NursesInviteModal() {
    return <NursesInviteDialog />;
}
