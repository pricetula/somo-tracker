import TeachersPage from "../page";

/**
 * Children slot for `/teachers/bulk-invite`.
 *
 * Renders the exact same listing page as `/teachers` while the `@modal` slot
 * overlays the BulkInviteForm dialog on top.
 */
export default function TeachersBulkInvitePage() {
    return <TeachersPage />;
}
