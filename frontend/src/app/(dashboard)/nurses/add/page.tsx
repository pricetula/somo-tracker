import NursesPage from "../page";

/**
 * Children slot for `/nurses/add`.
 *
 * Renders the exact same listing page as `/nurses` while the `@modal` slot
 * overlays the BulkInviteForm dialog on top.
 */
export default function NursesAddPage() {
    return <NursesPage />;
}
