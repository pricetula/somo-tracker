import ParentsPage from "../page";

/**
 * Children slot for `/parents/add`.
 *
 * Renders the exact same listing page as `/parents` while the `@modal` slot
 * overlays the BulkInviteForm dialog on top.
 */
export default function ParentsAddPage() {
    return <ParentsPage />;
}
