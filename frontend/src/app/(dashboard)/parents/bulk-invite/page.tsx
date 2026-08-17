import ParentsPage from "../page";

/**
 * Children slot for `/parents/bulk-invite`.
 *
 * Renders the exact same listing page as `/parents` while the `@modal` slot
 * overlays the BulkInviteForm dialog on top.
 */
export default function ParentsBulkInvitePage() {
    return <ParentsPage />;
}
