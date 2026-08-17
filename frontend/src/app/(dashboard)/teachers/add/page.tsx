import TeachersPage from "../page";

/**
 * Children slot for `/teachers/add`.
 *
 * Renders the exact same listing page as `/teachers` while the `@modal` slot
 * overlays the BulkInviteForm dialog on top.
 */
export default function TeachersAddPage() {
    return <TeachersPage />;
}
