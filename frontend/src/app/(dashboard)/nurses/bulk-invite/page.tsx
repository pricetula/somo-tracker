import NursesPage from "../page";

/**
 * Children slot for `/nurses/bulk-invite`.
 *
 * Renders the exact same listing page as `/nurses` while the `@modal` slot
 * overlays the BulkInviteForm dialog on top.
 */
export default function NursesBulkInvitePage() {
    return <NursesPage />;
}
