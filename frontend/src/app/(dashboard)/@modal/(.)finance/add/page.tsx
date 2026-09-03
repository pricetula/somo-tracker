import { FinanceInviteDialog } from "@/features/members";

/**
 * Intercepted `@modal` slot for `/finance/add`.
 *
 * Matches only during client-side navigation (the DataTable "Add" link), so
 * the BulkInviteForm renders in a dialog. On a hard navigation/refresh this
 * intercept is skipped and `@modal/default.tsx` renders, leaving
 * `/finance/add` to render the full-page form.
 */
export default function FinanceInviteModal() {
    return <FinanceInviteDialog />;
}
